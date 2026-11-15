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

import { spawn } from "node:child_process";
import { mkdirSync, writeFileSync, appendFileSync, readdirSync } from "node:fs";
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
};
const sc = S[NAME];
if (!sc) { console.error(`unknown scenario ${NAME}; one of ${Object.keys(S).join(" ")}`); process.exit(2); }
if (TARGET === "real" && !sc.turns[0].real) { console.error(`${NAME} is mock-only (errors cannot be forced on the real API)`); process.exit(2); }

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
    let turn = 0, deltas = 0, interrupted = false, stderr = "";
    const userMsg = (text) => send({ type: "user", message: { role: "user", content: text.replace("{cwd}", CWD) }, parent_tool_use_id: null, session_id: "" });

    send({ type: "control_request", request_id: "init_1", request: {
      subtype: "initialize",
      hooks: { PreToolUse: [{ matcher: "*", hookCallbackIds: ["hook_pre"] }] },
    } });

    let buf = "";
    child.stdout.on("data", (d) => {
      buf += d; let i;
      while ((i = buf.indexOf("\n")) >= 0) {
        const line = buf.slice(0, i); buf = buf.slice(i + 1);
        if (!line.trim()) continue;
        let m; try { m = JSON.parse(line); } catch { rec("in", undefined, line); continue; }
        rec("in", m);
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
          if (turn < turns.length) userMsg(turns[turn][TARGET]); else child.stdin.end();
        }
      }
    });
    child.stderr.on("data", (d) => (stderr += d));
    const kill = setTimeout(() => { rec("harness", { type: "harness_timeout" }); child.kill("SIGTERM"); }, Number(argv.timeout || 90000));
    child.on("exit", (code, signal) => { clearTimeout(kill); rec("harness", { type: "harness_exit", code, signal }); done({ code, signal, stderr }); });
  });
}

const r1 = await runOnce(cliArgs(sc.resumeTurn ? ["--session-id", sessionId] : ["--no-session-persistence"], "1"), sc.turns, "1");
let r2;
if (sc.resumeTurn) r2 = await runOnce(cliArgs(["--resume", sessionId], "2"), [sc.resumeTurn], "2");
writeFileSync(join(OUT, "meta.json"), JSON.stringify({ scenario: NAME, target: TARGET, sessionId, r1, r2 }, null, 2));
console.error(`${NAME}: exit ${r1.code}${r2 ? ` / resume exit ${r2.code}` : ""} -> ${transcript}`);
