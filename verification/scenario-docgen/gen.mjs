// Builds scenarios-2.1.311.md from template.md + the recorded transcripts, so every
// sequence and JSON example in the document is taken from the recordings, not retyped.
//   node gen.mjs <template.md> <out.md>
import { readFileSync, writeFileSync } from "node:fs";
const R = "/Users/user/Development/ai/chat-interface-specs/verification/runs/v2.1.311";
const [tplFile, outFile] = process.argv.slice(2);
const cache = {};
const load = (s) => (cache[s] ??= readFileSync(`${R}/mock/${s}/transcript.ndjson`, "utf8").split("\n").filter(Boolean).map((l) => JSON.parse(l)));

// ---------- id shortening ----------
const short = (s) => s
  .replaceAll(/\/Users\/[^"\\ ]*?\/runs\/v2\.1\.311\/mock\/[a-z_]+/g, "<run>")
  .replaceAll(/<run>\/config\/projects\/[^"\\ /]+\//g, "<run>/config/projects/<cwd-slug>/")
  .replaceAll(/\/private\/tmp\/claude-\d+\/[^"\\ ]*?\/([0-9a-f]{8})-[0-9a-f-]{27}\/tasks\//g, "<tmp>/$1…/tasks/")
  .replaceAll(/\/private\/tmp\/claude-\d+\/[^"\\ <]*/g, "<tmp>/…")
  .replaceAll(/([0-9a-f]{8})-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/g, "$1…");

// ---------- example formatting ----------
const trunc = (v) => typeof v === "string" && v.length > 230 ? v.slice(0, 190) + `…[+${v.length - 190} chars]`
  : Array.isArray(v) ? v.map(trunc) : v && typeof v === "object" ? Object.fromEntries(Object.entries(v).map(([k, x]) => [k, trunc(x)])) : v;
function fmt(v, ind = "", width = 150) {
  const inline = JSON.stringify(v);
  if (inline.length + ind.length <= width || v === null || typeof v !== "object") return inline;
  const i2 = ind + "  ";
  if (Array.isArray(v)) return "[\n" + v.map((x) => i2 + fmt(x, i2, width)).join(",\n") + "\n" + ind + "]";
  return "{\n" + Object.entries(v).map(([k, x]) => `${i2}${JSON.stringify(k)}: ${fmt(x, i2, width)}`).join(",\n") + "\n" + ind + "}";
}
const OMIT_RESULT = ["usage", "modelUsage", "model_usage_since_last_result"];
function ex(scenario, pred, nth = 0, opt = {}) {
  const hits = load(scenario).filter((x) => x.msg && pred(x.msg, x));
  const x = hits[nth];
  if (!x) throw new Error(`no example: ${scenario} ${pred} #${nth}`);
  let m = structuredClone(x.msg);
  if (m.type === "result") for (const k of OMIT_RESULT) delete m[k];
  for (const path of opt.omit ?? []) { const ks = path.split("."); let o = m; for (const k of ks.slice(0, -1)) o = o?.[k]; if (o) delete o[ks.at(-1)]; }
  m = JSON.parse(short(JSON.stringify(m))); // shorten ids and paths before cutting long strings
  return "```json\n" + fmt(trunc(m)) + "\n```";
}
const sys = (sub) => (m) => m.type === "system" && m.subtype === sub;
const creq = (sub) => (m) => m.type === "control_request" && m.request?.subtype === sub;
const cres = (id) => (m) => m.type === "control_response" && m.response?.request_id === id;
const blocks = (m) => (Array.isArray(m.message?.content) ? m.message.content.map((b) => b.type) : []);
const child = (type, block) => (m) => m.type === type && m.parent_tool_use_id && blocks(m).includes(block);
const main = (type, block) => (m) => m.type === type && m.parent_tool_use_id === null && blocks(m).includes(block);

const EX = {
  // subagent
  task_started_fg: ["subagent", sys("task_started")],
  child_user_prompt: ["subagent", child("user", "text")],
  task_progress: ["subagent", sys("task_progress")],
  hook_child: ["subagent", (m) => creq("hook_callback")(m) && m.request.input.agent_id],
  child_assistant: ["subagent", child("assistant", "tool_use")],
  child_tool_result: ["subagent", child("user", "tool_result")],
  task_updated_completed: ["subagent", sys("task_updated")],
  task_notification_completed: ["subagent", sys("task_notification")],
  agent_tool_result_fg: ["subagent", (m) => main("user", "tool_result")(m) && m.tool_use_result?.agentId],
  result_fg: ["subagent", (m) => m.type === "result"],
  child_assistant_text: ["subagent_forward_text", child("assistant", "text")],
  // background
  bg_changed: ["subagent_background", sys("background_tasks_changed")],
  task_started_bg: ["subagent_background", sys("task_started")],
  agent_tool_result_bg: ["subagent_background", (m) => main("user", "tool_result")(m) && m.tool_use_result?.isAsync],
  bg_changed_empty: ["subagent_background", sys("background_tasks_changed"), 1],
  result_notification: ["subagent_background", (m) => m.type === "result", 1],
  bg_changed_two: ["subagent_parallel", sys("background_tasks_changed"), 1],
  // stop
  stop_req: ["subagent_stop", creq("stop_task")],
  stop_resp: ["subagent_stop", cres("stop_1")],
  task_updated_killed: ["subagent_stop", (m) => sys("task_updated")(m) && m.patch.status === "killed"],
  task_notification_stopped: ["subagent_stop", (m) => sys("task_notification")(m) && m.status === "stopped"],
  stop_tool_result: ["subagent_stop", (m) => main("user", "tool_result")(m)],
  result_stop: ["subagent_stop", (m) => m.type === "result"],
  stop_bg_child_user: ["subagent_stop_bg", child("user", "text")],
  // failed / nested / to background
  task_updated_failed: ["subagent_failed", sys("task_updated")],
  task_notification_failed: ["subagent_failed", sys("task_notification")],
  task_started_nested: ["subagent_nested", sys("task_started"), 1],
  task_progress_agent: ["subagent_nested", sys("task_progress"), 0],
  bg_req: ["subagent_to_background", creq("background_tasks")],
  bg_resp: ["subagent_to_background", cres("bg_1")],
  task_updated_bg: ["subagent_to_background", (m) => sys("task_updated")(m) && m.patch.is_backgrounded],
  // permission
  can_use_tool_child: ["subagent_permission", creq("can_use_tool")],
  child_tool_result_perm: ["subagent_permission", child("user", "tool_result")],
  // pending approval
  pend_can: ["interrupt_pending_approval", creq("can_use_tool")],
  pend_cancel: ["interrupt_pending_approval", (m) => m.type === "control_cancel_request"],
  pend_int_resp: ["interrupt_pending_approval", cres("int_1")],
  pend_tool_result: ["interrupt_pending_approval", main("user", "tool_result")],
  pend_user_text: ["interrupt_pending_approval", main("user", "text")],
  pend_result: ["interrupt_pending_approval", (m) => m.type === "result"],
  // ask
  ask_can: ["ask_user_question", creq("can_use_tool")],
  ask_resp: ["ask_user_question", (m, x) => x.dir === "out" && m.type === "control_response" && m.response.response?.behavior],
  ask_tool_result: ["ask_user_question", main("user", "tool_result")],
  // compact
  compact_status1: ["compact", (m) => sys("status")(m) && m.status === "compacting"],
  compact_status2: ["compact", (m) => sys("status")(m) && "compact_result" in m],
  compact_boundary: ["compact", sys("compact_boundary")],
  compact_user_synth: ["compact", (m) => m.type === "user" && m.isSynthetic],
  compact_user_stdout: ["compact", (m) => m.type === "user" && m.isReplay === true],
  compact_result: ["compact", (m) => m.type === "result" && m.local_command],
  // queue
  q_user2: ["queued_message", (m, x) => x.dir === "out" && m.type === "user", 1],
  q_life_queued: ["queued_message", (m) => m.type === "command_lifecycle" && m.state === "queued", 1],
  q_life_started: ["queued_message", (m) => m.type === "command_lifecycle" && m.state === "started", 1],
  q_life_completed: ["queued_message", (m) => m.type === "command_lifecycle" && m.state === "completed", 1],
  q_status: ["queued_message", (m) => sys("status")(m) && m.user_message_uuids, 1],
  q_result: ["queued_message", (m) => m.type === "result", 1],
  qi_resp: ["queued_interrupt", cres("int_1")],
  qi_life_cancelled: ["queued_interrupt", (m) => m.type === "command_lifecycle" && m.state === "cancelled"],
  qc_req: ["queued_cancel", creq("interrupt")],
  qc_life: ["queued_cancel", (m) => m.type === "command_lifecycle" && m.state === "cancelled"],
  qc_resp: ["queued_cancel", cres("int_1")],
  qc_result: ["queued_cancel", (m) => m.type === "result"],
  // slash
  slash_replay: ["slash_prompt_command_replay", (m) => m.type === "user" && m.isReplay],
  // task output
  bash_bg_changed: ["task_output", sys("background_tasks_changed")],
  bash_task_started: ["task_output", sys("task_started")],
  bash_tool_result: ["task_output", main("user", "tool_result")],
  out_req: ["task_output", creq("get_task_output")],
  out_resp1: ["task_output", cres("out_1")],
  out_resp2: ["task_output", cres("out_2")],
  out_resp3: ["task_output", cres("out_3")],
  bash_task_notification: ["task_output", sys("task_notification")],
};

// ---------- annotated sequence ----------
// Same key rules as shape.mjs keyer(); adds normalized annotations (agent letters, statuses) and
// collapses consecutive stream_event keys and hook_callback round trips.
function seq(scenario) {
  const reqSub = new Map(); const agents = new Map(); const msgs = new Map();
  const A = (id) => { if (!agents.has(id)) agents.set(id, String.fromCharCode(65 + agents.size)); return agents.get(id); };
  const M = (id) => { if (!msgs.has(id)) msgs.set(id, "M" + (msgs.size + 1)); return msgs.get(id); };
  const out = [];
  const push = (s) => { const last = out.at(-1); if (last === s || last === s + "+") out[out.length - 1] = s + "+"; else out.push(s); };
  for (const { dir, msg: m } of load(scenario)) {
    if (!m) continue;
    const t = m.type; let k, note = "";
    if (t === "control_request") {
      reqSub.set(m.request_id, m.request?.subtype); k = `${dir}:control_request/${m.request?.subtype}`;
      const r = m.request;
      if (r.subtype === "can_use_tool") note = `tool_name=${r.tool_name}${r.agent_id ? `, agent_id=${A(r.agent_id)}` : ""}`;
      if (r.subtype === "hook_callback") note = `PreToolUse ${r.input.tool_name}${r.input.agent_id ? `, agent_id=${A(r.input.agent_id)}` : ""}`;
      if (r.subtype === "stop_task") note = `task_id=${agents.has(r.task_id) ? A(r.task_id) : r.task_id}`;
      if (r.subtype === "get_task_output") note = `task_id=${agents.has(r.task_id) ? A(r.task_id) : r.task_id}`;
      if (r.subtype === "interrupt" && r.cancel_queued) note = "cancel_queued=true";
      if (r.subtype === "initialize" && r.perTaskStopAffordance) note = "perTaskStopAffordance=true";
    } else if (t === "control_response") {
      const r = m.response ?? {}; const sub = reqSub.get(r.request_id) ?? "?"; k = `${dir}:control_response/${sub}/${r.subtype}`;
      if (sub === "interrupt") note = Object.entries(r.response ?? {}).map(([a, b]) => `${a}=[${b.map(M).join(",")}]`).join(", ");
      if (sub === "can_use_tool") note = `behavior=${r.response?.behavior}`;
    } else if (t === "control_cancel_request") { k = `${dir}:control_cancel_request`; note = `for the pending ${reqSub.get(m.request_id)}`; }
    else if (t === "stream_event") k = `${dir}:stream_event/*`;
    else if (t === "assistant" || t === "user") {
      const c = m.message?.content;
      k = `${dir}:${t}/${Array.isArray(c) ? [...new Set(c.map((b) => b.type))].sort().join("+") : typeof c}`;
      const bits = [];
      if (dir === "out") { if (typeof c === "string" && c.startsWith("/")) bits.push(JSON.stringify(c)); if (m.uuid) bits.push(`uuid=${M(m.uuid)}`); }
      if (m.agent_id) bits.push(`agent ${A(m.agent_id)}`);
      if (t === "assistant" && Array.isArray(c)) for (const b of c) if (b.type === "tool_use") bits.push(b.name);
      if (m.isReplay !== undefined) bits.push(`isReplay=${m.isReplay}`);
      if (m.isSynthetic) bits.push("isSynthetic=true");
      if (m.aborted) bits.push("aborted=true");
      if (dir === "in" && t === "user" && Array.isArray(c) && c[0]?.type === "text" && c[0].text.startsWith("[Request interrupted")) bits.push(JSON.stringify(c[0].text));
      note = bits.join(", ");
    } else if (t === "system") {
      k = `${dir}:system/${m.subtype}`;
      if (m.subtype === "status") note = `status=${m.status}${"compact_result" in m ? `, compact_result=${m.compact_result}` : ""}`;
      if (m.subtype === "task_started") note = `task ${A(m.task_id)}, ${m.task_type}, is_backgrounded=${m.is_backgrounded}${m.spawn_depth ? `, spawn_depth=${m.spawn_depth}` : ""}${m.parent_task_id ? `, parent_task_id=${A(m.parent_task_id)}` : ""}`;
      if (m.subtype === "task_progress") note = `task ${A(m.task_id)}, last_tool_name=${m.last_tool_name}`;
      if (m.subtype === "task_updated") note = `task ${A(m.task_id)}, patch=${JSON.stringify(Object.fromEntries(Object.entries(m.patch).filter(([q]) => q !== "end_time" && q !== "error").concat("end_time" in m.patch ? [["end_time", "…"]] : [], "error" in m.patch ? [["error", "…"]] : [])))}`;
      if (m.subtype === "task_notification") note = `task ${A(m.task_id)}, status=${m.status}`;
      if (m.subtype === "background_tasks_changed") note = `tasks=[${m.tasks.map((x) => A(x.task_id)).join(",")}]`;
      if (m.subtype === "compact_boundary") note = `trigger=${m.compact_metadata.trigger}`;
    } else if (t === "result") {
      k = `${dir}:result/${m.subtype}`;
      note = [m.terminal_reason && `terminal_reason=${m.terminal_reason}`, m.local_command && `local_command=${m.local_command}`, m.origin && `origin.kind=${m.origin.kind}`, `result_index=${m.result_index}`].filter(Boolean).join(", ");
    } else if (t === "command_lifecycle") { k = `${dir}:command_lifecycle`; note = `${M(m.command_uuid)} ${m.state}`; }
    else k = `${dir}:${t}${m.subtype ? "/" + m.subtype : ""}`;
    if (/keep_alive/.test(k)) continue;
    push(k + (note ? `   (${note})` : ""));
  }
  // collapse hook round trips
  const res = [];
  for (let i = 0; i < out.length; i++) {
    if (out[i].startsWith("in:control_request/hook_callback") && out[i + 1]?.startsWith("out:control_response/hook_callback/success")) { res.push(out[i].replace("in:control_request/hook_callback", "in:control_request/hook_callback ⇄ out:control_response") ); i++; }
    else res.push(out[i]);
  }
  return "```\n" + res.join("\n") + "\n```";
}

// ---------- model-facing text from the mock's body dump ----------
function body(scenario, step, which = -1) {
  const B = ["mock-bodies-scenarios2.ndjson", "mock-bodies-scenarios2b.ndjson"].flatMap((f) => readFileSync(`${R}/${f}`, "utf8").split("\n").filter(Boolean).map((l) => JSON.parse(l)));
  const b = B.find((x) => x.scenario === scenario && x.step === step);
  if (!b) throw new Error(`no body ${scenario} ${step}`);
  const m = b.body.messages.filter((x) => x.role === "user").at(which);
  const c = typeof m.content === "string" ? [m.content] : m.content.map((x) => x.text ?? JSON.stringify(x));
  return "```\n" + short(c.filter((x) => !/^<system-reminder>\nAttribution/.test(x)).join("\n---\n")).trim() + "\n```";
}

let tpl = readFileSync(tplFile, "utf8");
tpl = tpl.replace(/\{\{seq ([a-z_]+)\}\}/g, (_, s) => seq(s));
tpl = tpl.replace(/\{\{ex ([a-z0-9_]+)\}\}/g, (_, id) => { if (!EX[id]) throw new Error("unknown example " + id); return ex(...EX[id]); });
tpl = tpl.replace(/\{\{body ([a-z_]+) (\d+)\}\}/g, (_, s, n) => body(s, Number(n)));
const left = tpl.match(/\{\{[^}]*\}\}/g);
if (left) throw new Error("unresolved placeholders: " + left.join(" "));
writeFileSync(outFile, tpl);
console.log(`wrote ${outFile}: ${tpl.split("\n").length} lines`);
