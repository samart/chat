// usage: node show.mjs <transcript> [maxlen] [--all]
import { readFileSync } from "node:fs";
const [f, max = "700", all] = process.argv.slice(2);
const L = readFileSync(f, "utf8").split("\n").filter(Boolean).map((l) => JSON.parse(l));
for (const x of L) {
  const m = x.msg; if (!m) { console.log(x.ms, x.dir, "RAW", x.raw); continue; }
  if (m.type === "stream_event") { if (all) console.log(x.ms, x.dir, "stream_event", m.event.type, m.parent_tool_use_id ?? ""); continue; }
  if (m.type === "system" && m.subtype === "init") { console.log(x.ms, x.dir, "system/init"); continue; }
  if (m.type === "control_response" && m.response?.request_id === "init_1") { console.log(x.ms, x.dir, "control_response/initialize"); continue; }
  if (m.type === "control_request" && m.request?.subtype === "initialize") { console.log(x.ms, x.dir, JSON.stringify(m)); continue; }
  console.log(x.ms, x.dir, JSON.stringify(m).slice(0, Number(max)));
}
