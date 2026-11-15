#!/usr/bin/env node
// Fake Anthropic Messages API for deterministic, zero-cost CLI scenario runs.
//
//   node mock-api.mjs --port 8787 --log <requests.ndjson>
//   ANTHROPIC_BASE_URL=http://127.0.0.1:8787 ANTHROPIC_API_KEY=sk-ant-mock claude --bare -p ...
//
// Dispatch: the first user message must contain "SCENARIO:<name>". The step is
// the number of assistant turns already in the request (so a tool round-trip
// advances the script). Requests without a marker (side queries such as title
// generation) get a short generic text reply. Every request is logged.
//
// Extensions for the subagent / queue / compact scenarios:
//   - A subagent's own requests are dispatched the same way: its first user
//     message is the Agent tool's `prompt`, so the parent script embeds a
//     second marker there (e.g. SCENARIO:subagent_child).
//   - A turn may carry `delay` (ms before the response starts) and `slow`
//     (ms between text deltas) so the host can act while a request is in flight.
//   - `__AGENT__` as a tool name resolves to whichever of Agent/Task the
//     request's tool list offers.
//   - When no marker is found (a prompt-type slash command replaces the user
//     text), the `x-mock-scenario` request header is used. The runner sets it
//     through ANTHROPIC_CUSTOM_HEADERS only for scenarios that need it.
//   - `--bodies <file>` additionally dumps every request body (debugging).

import { createServer } from "node:http";
import { appendFileSync, writeFileSync } from "node:fs";

const args = Object.fromEntries(process.argv.slice(2).reduce((a, x, i, all) => (x.startsWith("--") && a.push([x.slice(2), all[i + 1]]), a), []));
const PORT = Number(args.port || 8787);
const LOG = args.log || "requests.ndjson";
const BODIES = args.bodies;
writeFileSync(LOG, "");
if (BODIES) writeFileSync(BODIES, "");
let n = 0;
const id = (p) => `${p}_mock${String(++n).padStart(4, "0")}`;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// ---- canned assistant turns ----
const text = (t, opts = {}) => ({ blocks: [{ type: "text", text: t }], stop: "end_turn", ...opts });
const tool = (name, input, lead) => ({ blocks: [...(lead ? [{ type: "text", text: lead }] : []), { type: "tool_use", name, input }], stop: "tool_use" });
const SCRIPTS = {
  text: [text("Hello from the mock API.")],
  thinking: [{ blocks: [{ type: "thinking", thinking: "Mock reasoning.", signature: "mocksig" }, { type: "text", text: "Done thinking." }], stop: "end_turn" }],
  tool_allow: [tool("Bash", { command: "touch allowed-marker", description: "Create a marker file" }, "Running a command."), text("Created the marker file.")],
  tool_deny: [tool("Bash", { command: "touch denied-marker", description: "Create a marker file" }), text("Understood, I will not run it.")],
  file_edit: [
    tool("Write", { file_path: "__CWD__/hello.txt", content: "hello\n" }),
    tool("Edit", { file_path: "__CWD__/hello.txt", old_string: "hello", new_string: "hello world" }),
    text("Created and edited hello.txt."),
  ],
  plan_mode: [
    tool("ExitPlanMode", { plan: "1. Create notes.txt\n2. Done" }),
    text("Plan approved; stopping here."),
  ],
  mcp_tool: [tool("mcp__probe_stdio__echo", { text: "via-mcp" }), text("MCP echo returned.")],
  interrupt: [text(Array.from({ length: 40 }, (_, i) => `chunk${i} `).join(""), { slow: 250 })],
  resume: [text("First turn reply."), text("Second turn reply after resume.")],
  // Subscription-style unified rate-limit headers on a successful reply -> rate_limit_event
  rate_limit_warning: [text("Near the limit.", { headers: {
    "anthropic-ratelimit-unified-status": "allowed_warning",
    "anthropic-ratelimit-unified-representative-claim": "five_hour",
    "anthropic-ratelimit-unified-reset": String(Math.floor(Date.now() / 1000) + 3600),
    "anthropic-ratelimit-unified-5h-utilization": "0.92",
    "anthropic-ratelimit-unified-5h-reset": String(Math.floor(Date.now() / 1000) + 3600),
    "anthropic-ratelimit-unified-5h-surpassed-threshold": "0.9",
  } })],
  max_turns: [tool("Bash", { command: "echo 1", description: "loop" }), tool("Bash", { command: "echo 2", description: "loop" }), tool("Bash", { command: "echo 3", description: "loop" })],
  // ---- subagents (parent script + child script; the child marker rides in the Agent prompt) ----
  subagent: [
    tool("__AGENT__", { description: "Mock child task", prompt: "SCENARIO:subagent_child Run one command and report.", subagent_type: "general-purpose", run_in_background: false }, "Delegating."),
    text("The subagent finished."),
  ],
  subagent_child: [tool("Bash", { command: "echo child-ran", description: "Print a marker" }, "Child working."), text("Child report: child-ran.")],
  subagent_background: [
    tool("__AGENT__", { description: "Mock background task", prompt: "SCENARIO:subagent_child_slow Run one command and report.", subagent_type: "general-purpose", run_in_background: true }, "Delegating in the background."),
    text("Launched a background agent; continuing."),
    text("The background agent has reported back."),
  ],
  subagent_child_slow: [
    { ...tool("Bash", { command: "echo child-ran", description: "Print a marker" }, "Child working."), delay: 1500 },
    { ...text("Child report: child-ran."), delay: 500 },
  ],
  subagent_stop: [
    tool("__AGENT__", { description: "Mock slow task", prompt: "SCENARIO:subagent_child_stuck Run one command and report.", subagent_type: "general-purpose", run_in_background: false }),
    text("The subagent was stopped."),
  ],
  subagent_stop_bg: [
    tool("__AGENT__", { description: "Mock slow background task", prompt: "SCENARIO:subagent_child_stuck Run one command and report.", subagent_type: "general-purpose", run_in_background: true }),
    text("Launched a background agent; continuing."),
    text("Noted: the background agent was stopped."),
  ],
  subagent_child_stuck: [
    { ...tool("Bash", { command: "echo child-ran", description: "Print a marker" }), delay: 1200 },
    { ...text("Child report (should never be seen)."), delay: 15000 },
  ],
  subagent_permission: [
    tool("__AGENT__", { description: "Mock child needing approval", prompt: "SCENARIO:subagent_child_perm Create a marker file.", subagent_type: "general-purpose", run_in_background: false }),
    text("The subagent finished."),
  ],
  subagent_child_perm: [tool("Bash", { command: "touch child-marker", description: "Create a marker file" }), text("Child report: created child-marker.")],
  // nested: main -> outer child -> inner child (spawn_depth 2, parent_task_id)
  subagent_nested: [
    tool("__AGENT__", { description: "Mock outer task", prompt: "SCENARIO:subagent_child_outer Delegate once more.", subagent_type: "general-purpose", run_in_background: false }),
    text("The nested subagents finished."),
  ],
  subagent_child_outer: [
    tool("__AGENT__", { description: "Mock inner task", prompt: "SCENARIO:subagent_child Run one command and report.", subagent_type: "general-purpose", run_in_background: false }),
    text("Outer child report: the inner agent finished."),
  ],
  // foreground agent moved to the background by the host (background_tasks control request)
  subagent_to_background: [
    tool("__AGENT__", { description: "Mock task to background", prompt: "SCENARIO:subagent_child_slow2 Run one command and report.", subagent_type: "general-purpose", run_in_background: false }),
    text("The agent now runs in the background; continuing."),
    text("The backgrounded agent has reported back."),
  ],
  subagent_child_slow2: [
    { ...tool("Bash", { command: "echo child-ran", description: "Print a marker" }), delay: 2000 },
    { ...text("Child report: child-ran."), delay: 1000 },
  ],
  // two background agents launched from one assistant message
  subagent_parallel: [
    { blocks: [
      { type: "tool_use", name: "__AGENT__", input: { description: "Mock task A", prompt: "SCENARIO:subagent_child_slow Run one command and report.", subagent_type: "general-purpose", run_in_background: true } },
      { type: "tool_use", name: "__AGENT__", input: { description: "Mock task B", prompt: "SCENARIO:subagent_child_slower Run one command and report.", subagent_type: "Explore", run_in_background: true } },
    ], stop: "tool_use" },
    text("Launched two background agents."),
    text("A background agent reported back."),
    text("Another background agent reported back."),
  ],
  subagent_child_slower: [
    { ...tool("Bash", { command: "echo child-ran", description: "Print a marker" }), delay: 3000 },
    { ...text("Child report: child-ran."), delay: 500 },
  ],
  // background agent running while the main turn is interrupted
  subagent_bg_interrupt: [
    tool("__AGENT__", { description: "Mock slow background task", prompt: "SCENARIO:subagent_child_stuck Run one command and report.", subagent_type: "general-purpose", run_in_background: true }),
    text(Array.from({ length: 40 }, (_, i) => `chunk${i} `).join(""), { slow: 150 }),
    text("Noted: the background agent ended."),
  ],
  // child whose model request fails (HTTP 400, not retried) -> failed task
  subagent_failed: [
    tool("__AGENT__", { description: "Mock failing task", prompt: "SCENARIO:subagent_child_error Run one command and report.", subagent_type: "general-purpose", run_in_background: false }),
    text("The subagent failed."),
  ],
  subagent_child_error: "400",
  // ---- permission / question ----
  interrupt_pending_approval: [tool("Bash", { command: "touch pending-marker", description: "Create a marker file" }, "About to run a command."), text("(not reached)")],
  ask_user_question: [
    tool("AskUserQuestion", { questions: [{ question: "Which colour should the button be?", header: "Colour", multiSelect: false, options: [
      { label: "Blue", description: "Calm and conventional" }, { label: "Green", description: "Signals success" } ] }] }),
    text("Thanks, noted the answer."),
  ],
  // ---- compaction: step 0 normal reply, step 1 answers the summarisation request. The summary
  // carries a second marker so the post-compact conversation (whose first user message is the
  // summary) stays scripted.
  compact: [text("First reply, before compaction."), text("<summary>\nSCENARIO:compact_after Mock summary of the conversation so far.\n</summary>")],
  compact_after: [text("Reply after compaction.")],
  // ---- queue ----
  queued_message: [text(Array.from({ length: 20 }, (_, i) => `chunk${i} `).join(""), { slow: 150 }), text("Reply to the queued message.")],
  queued_interrupt: [text(Array.from({ length: 40 }, (_, i) => `chunk${i} `).join(""), { slow: 150 }), text("Reply to the surviving queued message.")],
  queued_cancel: [text(Array.from({ length: 40 }, (_, i) => `chunk${i} `).join(""), { slow: 150 }), text("(not reached: queued message was cancelled)")],
  // ---- prompt-type slash command (dispatched through the x-mock-scenario header) ----
  slash_prompt_command: [text("Mock reply to the expanded slash command.")],
  // ---- background shell + get_task_output ----
  task_output: [
    tool("Bash", { command: "echo bg-line-1; sleep 2; echo bg-line-2", description: "Slow background command", run_in_background: true }),
    text("Started the background command."),
    text("The background command finished."),
  ],
  error_429: "429",
  error_529: "529",
  error_500: "500",
  error_400: "400",
};
const ERRORS = {
  429: { type: "rate_limit_error", message: "mock: rate limited" },
  529: { type: "overloaded_error", message: "mock: overloaded" },
  500: { type: "api_error", message: "mock: internal error" },
  400: { type: "invalid_request_error", message: "mock: prompt is too long" },
};

function scenarioOf(body) {
  const first = body.messages?.find((m) => m.role === "user");
  const s = JSON.stringify(first?.content ?? "");
  return s.match(/SCENARIO:([a-z0-9_]+)/)?.[1];
}

async function sse(res, model, turn, cwd, agentTool) {
  if (turn.delay) { await sleep(turn.delay); if (res.destroyed) return; }
  res.writeHead(200, { "content-type": "text/event-stream", "cache-control": "no-cache", "request-id": id("req"), ...(turn.headers ?? {}) });
  const ev = (type, data) => res.write(`event: ${type}\ndata: ${JSON.stringify({ type, ...data })}\n\n`);
  ev("message_start", { message: { id: id("msg"), type: "message", role: "assistant", model, content: [], stop_reason: null, stop_sequence: null, usage: { input_tokens: 25, output_tokens: 1, cache_creation_input_tokens: 0, cache_read_input_tokens: 0 } } });
  ev("ping", {});
  let out = 1;
  for (const [index, b0] of turn.blocks.entries()) {
    const b = JSON.parse(JSON.stringify(b0).replaceAll("__CWD__", cwd ?? "."));
    if (b.type === "text") {
      ev("content_block_start", { index, content_block: { type: "text", text: "" } });
      const parts = b.text.match(/.{1,12}/gs) ?? [""];
      for (const p of parts) { ev("content_block_delta", { index, delta: { type: "text_delta", text: p } }); out++; if (turn.slow) await sleep(turn.slow); if (res.destroyed) return; }
    } else if (b.type === "thinking") {
      ev("content_block_start", { index, content_block: { type: "thinking", thinking: "", signature: "" } });
      ev("content_block_delta", { index, delta: { type: "thinking_delta", thinking: b.thinking } });
      ev("content_block_delta", { index, delta: { type: "signature_delta", signature: b.signature } });
    } else if (b.type === "tool_use") {
      ev("content_block_start", { index, content_block: { type: "tool_use", id: id("toolu"), name: b.name === "__AGENT__" ? agentTool : b.name, input: {} } });
      const json = JSON.stringify(b.input);
      for (let i = 0; i < json.length; i += 20) ev("content_block_delta", { index, delta: { type: "input_json_delta", partial_json: json.slice(i, i + 20) } });
    }
    ev("content_block_stop", { index });
  }
  ev("message_delta", { delta: { stop_reason: turn.stop, stop_sequence: null }, usage: { output_tokens: out } });
  ev("message_stop", {});
  res.end();
}

createServer((req, res) => {
  let raw = "";
  req.on("data", (d) => (raw += d));
  req.on("end", async () => {
    let body = {}; try { body = JSON.parse(raw || "{}"); } catch {}
    const url = new URL(req.url, "http://x");
    const scen = scenarioOf(body) ?? (SCRIPTS[req.headers["x-mock-scenario"]] ? req.headers["x-mock-scenario"] : undefined);
    const head = (m) => (typeof m?.content === "string" ? m.content : JSON.stringify(m?.content ?? "")).slice(0, 300);
    const users = (body.messages ?? []).filter((m) => m.role === "user");
    const step = (body.messages ?? []).filter((m) => m.role === "assistant").length;
    appendFileSync(LOG, JSON.stringify({
      t: Date.now(), method: req.method, path: url.pathname, query: url.search, scenario: scen ?? null, step,
      model: body.model, stream: body.stream, n_messages: body.messages?.length, n_tools: body.tools?.length,
      tool_names: body.tools?.map((t) => t.name).slice(0, 80), beta: req.headers["anthropic-beta"], ua: req.headers["user-agent"],
      last_user_content_types: users.at(-1)?.content?.map?.((c) => c.type),
      first_user_head: head(users[0]), last_user_head: users.length > 1 ? head(users.at(-1)) : undefined,
      via_header: !scenarioOf(body) && !!scen,
    }) + "\n");
    if (BODIES) appendFileSync(BODIES, JSON.stringify({ t: Date.now(), path: url.pathname, scenario: scen ?? null, step, body: { ...body, tools: undefined, system: undefined } }) + "\n");

    if (url.pathname.endsWith("/v1/messages/count_tokens")) {
      res.writeHead(200, { "content-type": "application/json" });
      return res.end(JSON.stringify({ input_tokens: 1000 }));
    }
    if (url.pathname.endsWith("/v1/messages") && req.method === "POST") {
      const script = SCRIPTS[scen];
      if (typeof script === "string") {
        const code = Number(script);
        res.writeHead(code, { "content-type": "application/json", "retry-after": "0", "request-id": id("req"), ...(code === 429 ? { "anthropic-ratelimit-requests-remaining": "0" } : {}), ...(code === 529 ? { "x-should-retry": "true" } : {}) });
        return res.end(JSON.stringify({ type: "error", error: ERRORS[code] }));
      }
      const turn = script ? script[Math.min(step, script.length - 1)] : text("ok");
      const cwd = JSON.stringify(body.messages?.find((m) => m.role === "user")?.content ?? "").match(/CWD:([^\s\\"]+)/)?.[1];
      if (!body.stream) { // non-streaming side query
        res.writeHead(200, { "content-type": "application/json" });
        return res.end(JSON.stringify({ id: id("msg"), type: "message", role: "assistant", model: body.model, content: [{ type: "text", text: "ok" }], stop_reason: "end_turn", stop_sequence: null, usage: { input_tokens: 5, output_tokens: 1 } }));
      }
      const agentTool = ["Agent", "Task"].find((n) => body.tools?.some((t) => t.name === n)) ?? "Agent";
      return sse(res, body.model, turn, cwd, agentTool);
    }
    res.writeHead(404, { "content-type": "application/json" });
    res.end(JSON.stringify({ type: "error", error: { type: "not_found_error", message: `mock: no route ${url.pathname}` } }));
  });
}).listen(PORT, "127.0.0.1", () => console.error(`mock-api on http://127.0.0.1:${PORT}, log ${LOG}`));
