// usage: node dump.mjs <scenario> [filter-regex-on-key] [--result-full]
import { readFileSync } from "node:fs";
const R = new URL("../runs/v2.1.311/mock", import.meta.url).pathname;
const [name, filt, rf] = process.argv.slice(2);
const L = readFileSync(`${R}/${name}/transcript.ndjson`, "utf8").split("\n").filter(Boolean).map((l) => JSON.parse(l));
export const short = (s) => s
  .replaceAll(/\/Users\/[^"\\ ]*?\/runs\/v2\.1\.311\/mock\/[a-z_]+/g, "<run>")
  .replaceAll(/\/private\/tmp\/claude-\d+\/[^"\\ ]*?\/([0-9a-f]{8})-[0-9a-f-]{27}\/tasks\//g, "<tmp>/$1…/tasks/")
  .replaceAll(/\/private\/tmp\/claude-\d+\/[^"\\ ]*/g, "<tmp>/…")
  .replaceAll(/([0-9a-f]{8})-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/g, "$1…");
const re = filt && filt !== "-" ? new RegExp(filt) : null;
for (const x of L) {
  const m = x.msg; if (!m) continue;
  let k = m.type + (m.subtype ? "/" + m.subtype : "") + (m.request?.subtype ? "/" + m.request.subtype : "") + (m.response?.request_id ? "/" + m.response.request_id : "");
  if (m.type === "stream_event") { if (!re || !re.test("stream_event")) continue; }
  if (re && !re.test(k)) continue;
  if (!re && (k === "system/init" || /initialize|init_1/.test(k) || m.request?.subtype === "hook_callback" )) { console.log(x.ms, x.dir, k); continue; }
  let o = m;
  if (m.type === "result" && !rf) { const { usage, modelUsage, model_usage_since_last_result, ...rest } = m; o = rest; }
  console.log(x.ms, x.dir, short(JSON.stringify(o)));
}
