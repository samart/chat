#!/usr/bin/env node
// Zero-model-call probe of the Claude Code CLI stream-json control protocol.
//
// Spawns `claude -p --input-format stream-json --output-format stream-json ...`
// in an empty cwd, sends ONLY control_requests (never a user message), answers
// any CLI->host control_requests (mcp_message for an in-host "SDK" MCP server,
// hook_callback, can_use_tool), records every NDJSON line in both directions,
// then closes stdin.
//
// Usage:
//   node probe.mjs --out <dir> --cwd <new-empty-dir> [--bin claude]
//                  [--mode isolated|user-nomcp] [--extra-subtypes a,b,c]
//                  [--no-context-usage] [--hermetic http://127.0.0.1:8787]
//
// --mode isolated (default): --setting-sources "" --strict-mcp-config.
//   No user/project/local settings, hooks or ambient MCP servers -> reproducible.
// --mode user-nomcp: user settings (hooks, plugins, output styles, permissions)
//   load; ambient MCP servers are still skipped (they can open dashboards etc.).
//
// Network: initialize/mcp_status/get_settings make no model calls. Verified in
// debug.log: zero /v1/messages requests. get_context_usage DOES call
// /v1/messages/count_tokens (~36 calls), which is free but rate-limited; use
// --no-context-usage to skip it.
//
// Safety: no {"type":"user"} line can be written (send() throws). The CLI also
// gets --max-turns 1 --max-budget-usd 0.01 as a backstop.

import { spawn } from "node:child_process";
import { mkdirSync, writeFileSync, appendFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { randomUUID } from "node:crypto";

const HERE = dirname(fileURLToPath(import.meta.url));
const argv = {};
for (let i = 2; i < process.argv.length; i++) {
  const a = process.argv[i];
  if (!a.startsWith("--")) continue;
  const next = process.argv[i + 1];
  if (next === undefined || next.startsWith("--")) argv[a.slice(2)] = true;
  else { argv[a.slice(2)] = next; i++; }
}
const BIN = argv.bin || "claude";
const OUT = resolve(argv.out || "./out");
const CWD = resolve(argv.cwd || join(OUT, "cwd"));
const MODE = argv.mode || "isolated";
mkdirSync(OUT, { recursive: true });
mkdirSync(CWD, { recursive: true });
if (readdirSync(CWD).length) { console.error(`refusing: cwd ${CWD} is not empty`); process.exit(2); }

const mcpConfig = {
  mcpServers: {
    probe_sdk: { type: "sdk", name: "probe_sdk" },                         // served by THIS host via mcp_message
    probe_stdio: { type: "stdio", command: process.execPath, args: [join(HERE, "fake-mcp.mjs")] },
    probe_broken: { type: "stdio", command: "/nonexistent/mcp-server-binary", args: [] },
  },
};

const args = [
  "-p",
  "--input-format", "stream-json",
  "--output-format", "stream-json",
  "--verbose",
  "--permission-prompt-tool", "stdio",
  "--include-partial-messages",
  "--model", "haiku",
  "--max-turns", "1",
  "--max-budget-usd", "0.01",
  "--no-session-persistence",
  "--mcp-config", JSON.stringify(mcpConfig),
  "--strict-mcp-config",
  "--debug-file", join(OUT, "debug.log"),
];
if (MODE === "isolated") args.push("--setting-sources=");
else if (MODE === "user-nomcp") args.push("--setting-sources=user");
else { console.error("unknown --mode"); process.exit(2); }

const transcript = join(OUT, "transcript.ndjson");
writeFileSync(transcript, "");
const t0 = Date.now();
const rec = (dir, obj, raw) =>
  appendFileSync(transcript, JSON.stringify({ ms: Date.now() - t0, dir, msg: obj ?? undefined, raw: obj ? undefined : raw }) + "\n");

// --hermetic <mock-url>: scrub inherited CLAUDE*/ANTHROPIC* vars, use a throwaway
// CLAUDE_CONFIG_DIR and a dummy API key pointed at mock-api.mjs. No user state is
// read or written, so it is safe for running an unreleased/new binary side by side.
let env = { ...process.env };
if (argv.hermetic) {
  env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !/^(CLAUDE|ANTHROPIC)/.test(k)));
  Object.assign(env, { CLAUDE_CONFIG_DIR: join(OUT, "config"), ANTHROPIC_API_KEY: "sk-ant-mock-not-a-real-key",
    ANTHROPIC_BASE_URL: argv.hermetic, CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: "1", DISABLE_AUTOUPDATER: "1" });
}
const child = spawn(BIN, args, { cwd: CWD, env, stdio: ["pipe", "pipe", "pipe"] });
writeFileSync(join(OUT, "argv.json"), JSON.stringify({ bin: BIN, args, cwd: CWD, mode: MODE }, null, 2));

function send(obj) {
  if (obj.type === "user") throw new Error("probe must never send user messages");
  rec("out", obj);
  child.stdin.write(JSON.stringify(obj) + "\n");
}
const ok = (request_id, response) => send({ type: "control_response", response: { subtype: "success", request_id, response } });
const err = (request_id, error) => send({ type: "control_response", response: { subtype: "error", request_id, error } });

// CLI -> host control requests
const hostRequests = [];
function onCliRequest(msg) {
  const r = msg.request || {};
  hostRequests.push(r.subtype);
  switch (r.subtype) {
    case "mcp_message": {
      const m = r.message || {};
      if (m.id === undefined) return ok(msg.request_id, { mcp_response: { jsonrpc: "2.0", result: {}, id: 0 } });
      let result;
      if (m.method === "initialize") result = { protocolVersion: m.params?.protocolVersion ?? "2025-06-18", capabilities: { tools: {} }, serverInfo: { name: "probe_sdk", version: "0.0.1" } };
      else if (m.method === "tools/list") result = { tools: [{ name: "host_ping", description: "Host-side tool", inputSchema: { type: "object", properties: {} } }] };
      else return ok(msg.request_id, { mcp_response: { jsonrpc: "2.0", id: m.id, error: { code: -32601, message: "method not found" } } });
      return ok(msg.request_id, { mcp_response: { jsonrpc: "2.0", id: m.id, result } });
    }
    case "hook_callback":
      return ok(msg.request_id, {}); // no-op hook output
    case "can_use_tool":
      return ok(msg.request_id, { behavior: "deny", message: "probe: tools disabled" });
    default:
      return err(msg.request_id, `probe host: unsupported ${r.subtype}`);
  }
}

const pending = new Map();
let buf = "";
child.stdout.on("data", (d) => {
  buf += d;
  let i;
  while ((i = buf.indexOf("\n")) >= 0) {
    const line = buf.slice(0, i); buf = buf.slice(i + 1);
    if (!line.trim()) continue;
    let msg; try { msg = JSON.parse(line); } catch { rec("in", null, line); continue; }
    rec("in", msg);
    if (msg.type === "control_response") {
      const id = msg.response?.request_id; pending.get(id)?.(msg); pending.delete(id);
    } else if (msg.type === "control_request") onCliRequest(msg);
    // everything else (system/*, keep_alive, control_cancel_request, unknown) is only recorded
  }
});
let stderr = ""; child.stderr.on("data", (d) => (stderr += d));

function control(request, timeoutMs = 15000) {
  const request_id = `probe_${randomUUID().slice(0, 8)}`;
  return new Promise((res) => {
    const timer = setTimeout(() => { pending.delete(request_id); res({ timeout: true }); }, timeoutMs);
    pending.set(request_id, (m) => { clearTimeout(timer); res(m); });
    send({ type: "control_request", request_id, request });
  });
}
const results = {};
async function step(name, request, t) {
  const r = await control(request, t);
  results[name] = r.timeout ? { timeout: true, request } : { request, ...r.response };
  console.error(`${name.padEnd(28)} ${r.timeout ? "TIMEOUT" : r.response?.subtype}${r.response?.error ? "  " + r.response.error : ""}`);
}

// Handshake. One SessionStart hook callback to see whether the CLI calls back without a prompt.
await step("initialize", {
  subtype: "initialize",
  hooks: { SessionStart: [{ matcher: "*", hookCallbackIds: ["hook_0"] }] },
  sdkMcpServers: ["probe_sdk"],
}, 30000);
await new Promise((r) => setTimeout(r, 2500)); // let MCP servers connect
await step("mcp_status", { subtype: "mcp_status" });
await step("get_settings", { subtype: "get_settings" });
if (!argv["no-context-usage"]) await step("get_context_usage", { subtype: "get_context_usage" }, 30000);
await step("set_permission_mode_plan", { subtype: "set_permission_mode", mode: "plan" });
await step("set_permission_mode_bogus", { subtype: "set_permission_mode", mode: "__bogus__" });
await step("set_permission_mode_default", { subtype: "set_permission_mode", mode: "default" });
await step("set_model_haiku", { subtype: "set_model", model: "haiku" });
await step("set_max_thinking_tokens", { subtype: "set_max_thinking_tokens", max_thinking_tokens: 0 });
await step("interrupt_idle", { subtype: "interrupt" });
await step("rewind_files_no_msg", { subtype: "rewind_files", user_message_id: "00000000-0000-0000-0000-000000000000", dry_run: true });
await step("unknown_subtype", { subtype: "__probe_unknown_subtype__" });
await step("initialize_again", { subtype: "initialize", hooks: {} });
// Read-only subtypes found by `strings` on the binary; never pass mutating ones here.
for (const s of String(argv["extra-subtypes"] || "").split(",").filter(Boolean)) await step(`extra_${s}`, { subtype: s });

await new Promise((r) => setTimeout(r, 1500)); // catch unsolicited output (keep_alive, system/*)
child.stdin.end(); // clean shutdown: EOF on stdin, no user turn ever sent
const exit = await new Promise((r) => {
  const k = setTimeout(() => { child.kill("SIGTERM"); r({ code: null, signal: "SIGTERM(timeout)" }); }, 15000);
  child.on("exit", (code, signal) => { clearTimeout(k); r({ code, signal, ms: Date.now() - t0 }); });
});
writeFileSync(join(OUT, "results.json"), JSON.stringify({ exit, hostRequests, results }, null, 2));
writeFileSync(join(OUT, "stderr.txt"), stderr);
console.error("exit", exit, "cli->host requests:", hostRequests);
