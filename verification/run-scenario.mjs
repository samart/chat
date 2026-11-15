#!/usr/bin/env node
// Scenario runner: drives `claude` over stream-json like the editor plugin would,
// records the full bidirectional transcript, and answers permission prompts,
// hook callbacks and interrupts according to the scenario.
//
//   Free, deterministic (fake Messages API, see mock-api.mjs):
//     node run-scenario.mjs --target mock --mock-url http://127.0.0.1:8787 --scenario text --out runs/x
//   Paid (real API, cheapest model, hard caps). Refuses without --allow-spend:
//     node run-scenario.mjs --target real --allow-spend --scenario text --out runs/x
//
// Scenarios: text thinking tool_allow tool_deny file_edit plan_mode mcp_tool
//            interrupt resume max_turns error_429 error_529 error_500 error_400
//   mock-only (need scripted model turns):
//            subagent subagent_background subagent_stop subagent_stop_bg subagent_permission
//            interrupt_pending_approval ask_user_question compact queued_message
//            queued_interrupt queued_cancel slash_prompt_command task_output
//   variants: subagent_forward_text subagent_nested subagent_parallel subagent_to_background
//            subagent_interrupt subagent_bg_interrupt subagent_bg_interrupt_affordance
//            slash_prompt_command_replay
//
// Scenario options beyond the basics:
//   init        extra fields merged into the initialize request
//   on(m, api)  called for every message from the CLI before the default handling;
//               return true to suppress the default (e.g. to leave a can_use_tool unanswered).
//               api = { send, ok, userMsg, control, after, st } (st is per-run scratch state)
//   stampUuid   give every host user message a uuid (as a real UI does); needed to see
//               command_lifecycle / user_message_uuid attribution
//   quietEnd    ms. Instead of closing stdin on the last expected result, close it once a
//               result has been seen and the CLI has been silent for this long. Needed when
//               work continues after a result (background tasks, queued messages).

import { spawn } from "node:child_process";
import { mkdirSync, writeFileSync, appendFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { randomUUID } from "node:crypto";

const HERE = dirname(fileURLToPath(import.meta.url));
const argv = {};
for (let i = 2; i < process.argv.length; i++) {
  const a = process.argv[i]; if (!a.startsWith("--")) continue;
  const nx = process.argv[i + 1];
  if (nx === undefined || nx.startsWith("--")) argv[a.slice(2)] = true; else { argv[a.slice(2)] = nx; i++; }
}
const TARGET = argv.target || "mock";
const NAME = argv.scenario;
const OUT = resolve(argv.out || `runs/${NAME}`);
const BIN = argv.bin || "claude";
if (TARGET === "real" && !argv["allow-spend"]) { console.error("refusing: --target real costs money; add --allow-spend"); process.exit(2); }
if (TARGET === "mock" && !/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(argv["mock-url"] || "")) { console.error("need --mock-url http://127.0.0.1:PORT"); process.exit(2); }

// prompt: mock prompts carry a SCENARIO marker; real prompts are written to make
// Haiku do the thing in one or two turns.
const S = {
  text:       { turns: [{ mock: "SCENARIO:text hi", real: "Reply with exactly: hello" }] },
  thinking:   { turns: [{ mock: "SCENARIO:thinking", real: "Think briefly, then reply with exactly: done" }], flags: ["--effort", "low"] },
  tool_allow: { turns: [{ mock: "SCENARIO:tool_allow", real: "Run the shell command `touch allowed-marker` with the Bash tool, then stop." }], permission: "allow" },
  tool_deny:  { turns: [{ mock: "SCENARIO:tool_deny", real: "Run the shell command `touch denied-marker` with the Bash tool. If it is denied, say so and stop." }], permission: "deny" },
  file_edit:  { turns: [{ mock: "SCENARIO:file_edit CWD:{cwd}", real: "Create hello.txt containing 'hello' with the Write tool, then change it to 'hello world' with the Edit tool. No other tools." }], permission: "allow" },
  plan_mode:  { turns: [{ mock: "SCENARIO:plan_mode", real: "Plan (do not execute) creating notes.txt. Present a one-line plan with ExitPlanMode immediately." }], flags: ["--permission-mode", "plan"], permission: "allow", afterPlan: "set_permission_mode default" },
  mcp_tool:   { turns: [{ mock: "SCENARIO:mcp_tool", real: "Call the echo tool from the probe_stdio MCP server with text 'via-mcp', then stop." }], permission: "allow", mcp: true },
  interrupt:  { turns: [{ mock: "SCENARIO:interrupt", real: "Count slowly from 1 to 200, one number per line." }], interruptAfterDeltas: 3 },
  resume:     { turns: [{ mock: "SCENARIO:resume first", real: "Remember the word 'tangerine'. Reply: ok" }], resumeTurn: { mock: "second", real: "What word did I ask you to remember? One word." } },
  max_turns:  { turns: [{ mock: "SCENARIO:max_turns", real: "Run `echo 1`, then `echo 2`, then `echo 3` as three separate Bash calls." }], permission: "allow", maxTurns: 2 },
  rate_limit_warning: { turns: [{ mock: "SCENARIO:rate_limit_warning" }] },
  error_429:  { turns: [{ mock: "SCENARIO:error_429" }], env: { CLAUDE_CODE_MAX_RETRIES: "1" } },
  error_529:  { turns: [{ mock: "SCENARIO:error_529" }], env: { CLAUDE_CODE_MAX_RETRIES: "1" } },
  error_500:  { turns: [{ mock: "SCENARIO:error_500" }], env: { CLAUDE_CODE_MAX_RETRIES: "1" } },
  error_400:  { turns: [{ mock: "SCENARIO:error_400" }] },
  // ---- mock-only scenarios below: they need scripted model turns ----
  subagent: { turns: [{ mock: "SCENARIO:subagent" }], permission: "allow", quietEnd: 1500 },
  subagent_background: { turns: [{ mock: "SCENARIO:subagent_background" }], permission: "allow", quietEnd: 5000 },
  // stop_task while the child's second model request is hanging in the mock (15 s delay).
  subagent_stop: { turns: [{ mock: "SCENARIO:subagent_stop" }], permission: "allow", quietEnd: 3000,
    on(m, api) {
      if (m.type === "system" && m.subtype === "task_started" && !api.st.stopped) { // child: 1.2 s first reply, Bash, then hangs
        api.st.stopped = true;
        api.after(2500, () => api.control("stop_1", { subtype: "stop_task", task_id: m.task_id }));
      }
    } },
  subagent_stop_bg: { turns: [{ mock: "SCENARIO:subagent_stop_bg" }], permission: "allow", quietEnd: 4000, init: { perTaskStopAffordance: true },
    on(m, api) {
      if (m.type === "result" && !api.st.stopped) { // main turn is over, the background child is still running
        api.st.stopped = true;
        api.after(1500, () => api.control("stop_1", { subtype: "stop_task", task_id: api.st.taskId }));
        api.after(2200, () => api.control("stop_2", { subtype: "stop_task", task_id: api.st.taskId })); // second stop of the same task
        api.after(2400, () => api.control("stop_3", { subtype: "stop_task", task_id: "no-such-task" }));
      }
    } },
  subagent_permission: { turns: [{ mock: "SCENARIO:subagent_permission" }], permission: "allow", quietEnd: 1500 },
  // Variants. Several reuse another scenario's mock script (the marker names the script).
  subagent_forward_text: { turns: [{ mock: "SCENARIO:subagent" }], permission: "allow", quietEnd: 1500, flags: ["--forward-subagent-text"] },
  subagent_nested: { turns: [{ mock: "SCENARIO:subagent_nested" }], permission: "allow", quietEnd: 1500 },
  subagent_parallel: { turns: [{ mock: "SCENARIO:subagent_parallel" }], permission: "allow", quietEnd: 5000, maxTurns: 8 },
  // Foreground agent, then the host sends background_tasks (the Ctrl+B equivalent).
  subagent_to_background: { turns: [{ mock: "SCENARIO:subagent_to_background" }], permission: "allow", quietEnd: 5000,
    on(m, api) {
      if (m.type === "system" && m.subtype === "task_started" && !api.st.bg) {
        api.st.bg = true;
        api.after(700, () => api.control("bg_1", { subtype: "background_tasks", tool_use_id: m.tool_use_id }));
      }
    } },
  // Foreground agent mid-run, host interrupts the turn (no stop_task).
  subagent_interrupt: { turns: [{ mock: "SCENARIO:subagent_stop" }], permission: "allow", quietEnd: 3000,
    on(m, api) {
      if (m.type === "system" && m.subtype === "task_started" && !api.st.int) {
        api.st.int = true;
        api.after(2500, () => api.control("int_1", { subtype: "interrupt" }));
      }
    } },
  // Background agent running while the main turn streams; host interrupts the turn.
  // Without perTaskStopAffordance the CLI is documented to kill background tasks on interrupt.
  subagent_bg_interrupt: { turns: [{ mock: "SCENARIO:subagent_bg_interrupt" }], permission: "allow", quietEnd: 4000,
    on(m, api) {
      if (m.type === "stream_event" && m.event?.delta?.type === "text_delta" && ++api.st.deltas === 12) api.control("int_1", { subtype: "interrupt" });
    } },
  // Same with the affordance declared: the agent should survive; stop_task cleans it up afterwards.
  subagent_bg_interrupt_affordance: { turns: [{ mock: "SCENARIO:subagent_bg_interrupt" }], permission: "allow", quietEnd: 4000, init: { perTaskStopAffordance: true },
    on(m, api) {
      if (m.type === "stream_event" && m.event?.delta?.type === "text_delta" && ++api.st.deltas === 12) {
        api.control("int_1", { subtype: "interrupt" });
        api.after(2500, () => api.control("stop_1", { subtype: "stop_task", task_id: api.st.taskId }));
      }
    } },
  subagent_failed: { turns: [{ mock: "SCENARIO:subagent_failed" }], permission: "allow", quietEnd: 1500, env: { CLAUDE_CODE_MAX_RETRIES: "1" } },
  // Probe only: initialize.agentProgressSummaries. Not in compare.sh (see scenarios-2.1.311.md).
  subagent_progress_summary: { turns: [{ mock: "SCENARIO:subagent_stop" }], permission: "allow", quietEnd: 1500, init: { agentProgressSummaries: true } },
  // can_use_tool is left unanswered; the host interrupts instead.
  interrupt_pending_approval: { turns: [{ mock: "SCENARIO:interrupt_pending_approval" }], quietEnd: 2000,
    on(m, api) {
      if (m.type === "control_request" && m.request.subtype === "can_use_tool") {
        api.after(400, () => api.control("int_1", { subtype: "interrupt" }));
        return true;
      }
    } },
  ask_user_question: { turns: [{ mock: "SCENARIO:ask_user_question" }], permission: "allow",
    on(m, api) {
      if (m.type === "control_request" && m.request.subtype === "can_use_tool" && m.request.tool_name === "AskUserQuestion") {
        const q = m.request.input.questions[0];
        api.ok(m.request_id, { behavior: "allow", updatedInput: { ...m.request.input, answers: { [q.question]: q.options[1].label } } });
        return true;
      }
    } },
  compact: { turns: [{ mock: "SCENARIO:compact first turn" }, { mock: "/compact" }, { mock: "a question after compaction" }] },
  // A second user message is written while the first turn is still streaming.
  queued_message: { turns: [{ mock: "SCENARIO:queued_message" }], quietEnd: 3000, stampUuid: true,
    on(m, api) {
      if (m.type === "stream_event" && m.event?.type === "content_block_delta" && ++api.st.deltas === 3)
        api.userMsg("second message, sent while the first turn is streaming", { uuid: randomUUID() });
    } },
  // Same, then a plain interrupt: the queued message should survive (still_queued) and run.
  queued_interrupt: { turns: [{ mock: "SCENARIO:queued_interrupt" }], quietEnd: 3000, stampUuid: true,
    on(m, api) {
      if (m.type !== "stream_event" || m.event?.type !== "content_block_delta") return;
      const n = ++api.st.deltas;
      if (n === 3) api.userMsg("second message, sent while the first turn is streaming", { uuid: randomUUID() });
      if (n === 6 && !api.st.int) { api.st.int = true; api.control("int_1", { subtype: "interrupt" }); }
    } },
  // Same, with interrupt { cancel_queued: true }.
  queued_cancel: { turns: [{ mock: "SCENARIO:queued_cancel" }], quietEnd: 3000, stampUuid: true,
    on(m, api) {
      if (m.type !== "stream_event" || m.event?.type !== "content_block_delta") return;
      const n = ++api.st.deltas;
      if (n === 3) api.userMsg("second message, sent while the first turn is streaming", { uuid: randomUUID() });
      if (n === 6 && !api.st.int) { api.st.int = true; api.control("int_1", { subtype: "interrupt", cancel_queued: true }); }
    } },
  // A prompt-type command replaces the user text, so the SCENARIO marker cannot ride in the
  // prompt; the mock falls back to the x-mock-scenario header.
  slash_prompt_command: { turns: [{ mock: "/init" }], env: { ANTHROPIC_CUSTOM_HEADERS: "x-mock-scenario: slash_prompt_command" } },
  slash_prompt_command_replay: { turns: [{ mock: "/init" }], env: { ANTHROPIC_CUSTOM_HEADERS: "x-mock-scenario: slash_prompt_command" }, flags: ["--replay-user-messages"] },
  task_output: { turns: [{ mock: "SCENARIO:task_output" }], permission: "allow", quietEnd: 5000,
    on(m, api) {
      if (m.type === "system" && m.subtype === "task_started") api.after(500, () => api.control("out_1", { subtype: "get_task_output", task_id: m.task_id }));
      if (m.type === "system" && m.subtype === "task_notification") {
        api.after(200, () => api.control("out_2", { subtype: "get_task_output", task_id: m.task_id }));
        api.after(300, () => api.control("out_3", { subtype: "get_task_output", task_id: "no-such-task" }));
      }
    } },
};
const sc = S[NAME];
if (!sc) { console.error(`unknown scenario ${NAME}; one of ${Object.keys(S).join(" ")}`); process.exit(2); }
if (TARGET === "real" && !sc.turns[0].real) { console.error(`${NAME} is mock-only (errors cannot be forced on the real API)`); process.exit(2); }

// Never overwrite a recording: an existing transcript is a golden for that binary.
if (existsSync(join(OUT, "transcript.ndjson")) && !argv.force) { console.error(`refusing: ${OUT} already holds a recording (use a new --out, or --force)`); process.exit(2); }
mkdirSync(OUT, { recursive: true });
const CWD = join(OUT, "cwd"); mkdirSync(CWD, { recursive: true });
if (readdirSync(CWD).length) { console.error(`refusing: ${CWD} not empty`); process.exit(2); }

// Hermetic env: drop every inherited CLAUDE*/ANTHROPIC* var (a parent Claude Code
// session leaks ~16 of them, e.g. CLAUDE_CODE_ENTRYPOINT, CLAUDECODE).
const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !/^(CLAUDE|ANTHROPIC)/.test(k)));
Object.assign(env, { DISABLE_AUTOUPDATER: "1", DISABLE_TELEMETRY: "1", DISABLE_ERROR_REPORTING: "1" }, sc.env ?? {});
if (TARGET === "mock") {
  Object.assign(env, {
    ANTHROPIC_BASE_URL: argv["mock-url"],
    ANTHROPIC_API_KEY: "sk-ant-mock-not-a-real-key", // a leaked request would 401, never bill
    CLAUDE_CONFIG_DIR: join(OUT, "config"),          // no user state, sessions land here
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: "1",
  });
}

const sessionId = randomUUID();
function cliArgs(extra = [], label = "1") {
  const a = [
    "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
    "--permission-prompt-tool", "stdio", "--include-partial-messages",
    "--model", "haiku", "--max-turns", String(sc.maxTurns ?? 4), "--max-budget-usd", "0.05",
    "--setting-sources=", "--strict-mcp-config",
    "--debug-file", join(OUT, `debug-${label}.log`),
    ...(sc.flags ?? []), ...extra,
  ];
  // Not --bare: it hard-limits tools to Bash/Edit/Read (even with --tools). Isolation comes from
  // CLAUDE_CONFIG_DIR + ANTHROPIC_API_KEY + --setting-sources= instead.
  if (sc.mcp) a.push("--mcp-config", JSON.stringify({ mcpServers: { probe_stdio: { type: "stdio", command: process.execPath, args: [join(HERE, "fake-mcp.mjs")] } } }));
  return a;
}

const transcript = join(OUT, "transcript.ndjson");
writeFileSync(transcript, "");
const t0 = Date.now();

function runOnce(args, turns, label) {
  return new Promise((done) => {
    const child = spawn(BIN, args, { cwd: CWD, env, stdio: ["pipe", "pipe", "pipe"] });
    const rec = (dir, msg, raw) => appendFileSync(transcript, JSON.stringify({ ms: Date.now() - t0, run: label, dir, msg, raw }) + "\n");
    const send = (o) => { rec("out", o); child.stdin.write(JSON.stringify(o) + "\n"); };
    const ok = (request_id, response) => send({ type: "control_response", response: { subtype: "success", request_id, response } });
    let turn = 0, deltas = 0, interrupted = false, stderr = "", ended = false, sawResult = false, quiet = null;
    const userMsg = (text, extra = {}) => send({ type: "user", message: { role: "user", content: text.replace("{cwd}", CWD) }, parent_tool_use_id: null, session_id: "", ...(sc.stampUuid ? { uuid: randomUUID() } : {}), ...extra });
    const end = () => { if (!ended) { ended = true; if (sc.quietEnd) rec("harness", { type: "harness_stdin_end" }); child.stdin.end(); } };
    const timers = [];
    const api = {
      send, ok, userMsg, st: { deltas: 0 },
      control: (request_id, request) => send({ type: "control_request", request_id, request }),
      after: (ms, fn) => timers.push(setTimeout(() => { if (!ended) fn(); }, ms)),
    };

    send({ type: "control_request", request_id: "init_1", request: {
      subtype: "initialize",
      hooks: { PreToolUse: [{ matcher: "*", hookCallbackIds: ["hook_pre"] }] },
      ...(sc.init ?? {}),
    } });

    let buf = "";
    child.stdout.on("data", (d) => {
      buf += d; let i;
      while ((i = buf.indexOf("\n")) >= 0) {
        const line = buf.slice(0, i); buf = buf.slice(i + 1);
        if (!line.trim()) continue;
        let m; try { m = JSON.parse(line); } catch { rec("in", undefined, line); continue; }
        rec("in", m);
        if (sc.quietEnd && sawResult) { clearTimeout(quiet); quiet = setTimeout(end, sc.quietEnd); }
        if (m.type === "system" && m.subtype === "task_started" && !api.st.taskId) api.st.taskId = m.task_id;
        if (sc.on?.(m, api) === true) continue;
        if (m.type === "control_response" && m.response?.request_id === "init_1") userMsg(turns[turn][TARGET]);
        else if (m.type === "control_request") {
          const r = m.request;
          if (r.subtype === "can_use_tool") {
            if (sc.permission === "allow") ok(m.request_id, { behavior: "allow", updatedInput: r.input });
            else ok(m.request_id, { behavior: "deny", message: "Denied by verification harness" });
            if (r.tool_name === "ExitPlanMode" && sc.afterPlan) send({ type: "control_request", request_id: "mode_1", request: { subtype: "set_permission_mode", mode: "default" } });
          } else if (r.subtype === "hook_callback") ok(m.request_id, {});
          else send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: `harness: unsupported ${r.subtype}` } });
        } else if (m.type === "stream_event" && m.event?.type === "content_block_delta" && sc.interruptAfterDeltas && !interrupted && ++deltas >= sc.interruptAfterDeltas) {
          interrupted = true;
          send({ type: "control_request", request_id: "int_1", request: { subtype: "interrupt" } });
        } else if (m.type === "result") {
          turn++;
          if (turn < turns.length) userMsg(turns[turn][TARGET]);
          else if (sc.quietEnd) { sawResult = true; clearTimeout(quiet); quiet = setTimeout(end, sc.quietEnd); }
          else end();
        }
      }
    });
    child.stderr.on("data", (d) => (stderr += d));
    const kill = setTimeout(() => { rec("harness", { type: "harness_timeout" }); child.kill("SIGTERM"); }, Number(argv.timeout || 90000));
    child.on("exit", (code, signal) => { clearTimeout(kill); clearTimeout(quiet); timers.forEach(clearTimeout); rec("harness", { type: "harness_exit", code, signal }); done({ code, signal, stderr }); });
  });
}

const r1 = await runOnce(cliArgs(sc.resumeTurn ? ["--session-id", sessionId] : ["--no-session-persistence"], "1"), sc.turns, "1");
let r2;
if (sc.resumeTurn) r2 = await runOnce(cliArgs(["--resume", sessionId], "2"), [sc.resumeTurn], "2");
writeFileSync(join(OUT, "meta.json"), JSON.stringify({ scenario: NAME, target: TARGET, sessionId, r1, r2 }, null, 2));
console.error(`${NAME}: exit ${r1.code}${r2 ? ` / resume exit ${r2.code}` : ""} -> ${transcript}`);
