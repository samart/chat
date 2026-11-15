import { readFileSync } from "node:fs";
const R = "/Users/user/Development/ai/chat-interface-specs/verification/runs/v2.1.311/mock";
for (const s of process.argv.slice(2)) {
  console.log("=====", s);
  for (const x of readFileSync(`${R}/${s}/transcript.ndjson`, "utf8").split("\n").filter(Boolean).map((l) => JSON.parse(l))) {
    const m = x.msg; if (!m || m.type === "stream_event") continue;
    if (m.type === "control_request" && m.request.subtype === "hook_callback") { console.log(x.ms, "hook_callback PreToolUse", m.request.input.tool_name, "agent_id=" + (m.request.input.agent_id ?? "-"), "agent_type=" + (m.request.input.agent_type ?? "-")); continue; }
    if (m.type === "control_response" && (m.response.request_id === "init_1" || x.dir === "out")) continue;
    if (m.type === "control_request" && m.request.subtype === "initialize") continue;
    const c = m.message?.content;
    const kinds = Array.isArray(c) ? c.map((b) => b.type + (b.type === "tool_use" ? `(${b.name} ${b.id})` : b.type === "tool_result" ? `(${b.tool_use_id})` : b.type === "text" ? `("${b.text.slice(0, 28)}")` : "")).join("+") : typeof c === "string" ? `"${c.slice(0, 30)}"` : "";
    const extra = m.type === "system" ? JSON.stringify({ task_id: m.task_id, tool_use_id: m.tool_use_id, parent_task_id: m.parent_task_id, spawn_depth: m.spawn_depth, is_backgrounded: m.is_backgrounded, patch: m.patch, status: m.status, tasks: m.tasks?.map((t) => t.task_id + (t.parent_task_id ? "<" + t.parent_task_id : "")), desc: m.description, last_tool_name: m.last_tool_name, summary: m.summary, usage: m.usage }) : m.type === "result" ? JSON.stringify({ subtype: m.subtype, terminal_reason: m.terminal_reason, is_error: m.is_error, origin: m.origin, result_index: m.result_index, num_turns: m.num_turns, killed: m.subagent_stats?.killed }) : m.type.startsWith("control") ? JSON.stringify(m.request ?? m.response ?? m).slice(0, 200) : "";
    console.log(x.ms, x.dir, m.type + (m.subtype ? "/" + m.subtype : ""), kinds, m.parent_tool_use_id !== undefined ? "parent_tool_use_id=" + m.parent_tool_use_id : "", m.agent_id ? "agent_id=" + m.agent_id : "", extra);
  }
}
