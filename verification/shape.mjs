#!/usr/bin/env node
// Observed-shape inference, sequence signatures and diffing for Claude Code
// stream-json transcripts.
//
//   node shape.mjs infer   <out.json> <transcript.ndjson>...   build shapes per message key
//   node shape.mjs diff    <old.json> <new.json>                classify differences
//   node shape.mjs seq     <transcript.ndjson>                  normalized message-key sequence
//   node shape.mjs catalog <out.json> <results.json>            value catalog from a probe run
//   node shape.mjs catdiff <old.json> <new.json>                diff two catalogs
//
// Transcript lines may be probe records ({dir, msg}) or raw stream-json lines
// (treated as dir "in").

import { readFileSync, writeFileSync } from "node:fs";

// ---------- message keys ----------
// A "message key" is the unit we build a shape for. Control responses are keyed
// by the subtype of the request they answer (correlated through request_id).
export function keyer() {
  const reqSubtype = new Map();
  return function key(dir, m) {
    if (!m || typeof m !== "object") return `${dir}:<non-json>`;
    const t = m.type ?? "<no-type>";
    if (t === "control_request") {
      reqSubtype.set(m.request_id, m.request?.subtype);
      return `${dir}:control_request/${m.request?.subtype}`;
    }
    if (t === "control_response") {
      const r = m.response ?? {};
      return `${dir}:control_response/${reqSubtype.get(r.request_id) ?? "?"}/${r.subtype}`;
    }
    if (t === "stream_event") {
      const e = m.event ?? {};
      const sub = e.delta?.type ?? e.content_block?.type;
      return `${dir}:stream_event/${e.type}${sub ? "/" + sub : ""}`;
    }
    if (t === "assistant" || t === "user") {
      const c = m.message?.content;
      const kinds = Array.isArray(c) ? [...new Set(c.map((b) => b.type))].sort().join("+") : typeof c;
      return `${dir}:${t}/${kinds}`;
    }
    return `${dir}:${t}${m.subtype ? "/" + m.subtype : ""}`;
  };
}

// ---------- value normalization ----------
// Literal values are kept only for discriminator-like keys (they form enums);
// everything else is reduced to a format tag so ids, timestamps, paths and text
// never cause diffs.
const ENUM_KEYS = new Set([
  "type", "subtype", "status", "mode", "permissionMode", "behavior", "stop_reason", "role",
  "kind", "source", "scope", "apiProvider", "output_style", "session_state", "fast_mode_state",
  "fast_mode_disabled_reason", "current_permission_mode", "autocompactSource", "method", "decision",
  "permission_denial_reason", "terminal_reason", "subscription_type", "level",
  "hook_event_name", "tool_name", "destination", "error", "rateLimitType",
]);
// Tool inputs/outputs are per-tool schemas (covered statically by sdk-tools.d.ts);
// record only that they are objects so Bash vs Write inputs do not merge into noise.
// Paths are prefixed with the message key, e.g. "in:control_request/can_use_tool$.request.input".
const OPAQUE_PATHS = [/\.(tool_input|updatedInput|tool_use_result)$/, /\.content\[\]\.(input|content)$/, /can_use_tool\$\.request\.input$/];
// Object paths whose keys are data (names, model ids, codenames) -> shape as a map.
const MAP_PATHS = [
  /\.rate_limits$/, /\.modelUsage$/, /\.model_usage$/, /\.mcpServers$/, /\.enabledPlugins$/,
  /\.server_tool_use$/, /\.hooks$/,
];
const looksLikeDataKeys = (keys) =>
  keys.length > 0 && keys.every((k) => /^(claude-|mcp__|\/|[0-9a-f]{8}-)|@|\[/.test(k));

function fmt(v) {
  if (typeof v !== "string") return null;
  if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(v)) return "uuid";
  if (/^\d{4}-\d\d-\d\dT\d\d:\d\d/.test(v)) return "datetime";
  if (/^[^@\s]+@[^@\s]+\.[a-z]+$/i.test(v)) return "email"; // also redacts PII
  if (/^(toolu|msg|req|srvtoolu)_[A-Za-z0-9]+$/.test(v)) return "anthropic-id";
  if (v.startsWith("/")) return "path";
  return null;
}

// ---------- shape inference ----------
// Shape node: { n, types: {t: count}, props?: {k: node}, items?: node, values?: {lit: count}, map?: node, formats? }
const newNode = () => ({ n: 0, types: {} });
function observe(node, v, path, key) {
  node.n++;
  const t = v === null ? "null" : Array.isArray(v) ? "array" : typeof v === "number" ? (Number.isInteger(v) ? "integer" : "number") : typeof v;
  node.types[t] = (node.types[t] ?? 0) + 1;
  if (t === "string") {
    const f = fmt(v);
    if (f) (node.formats ??= {})[f] = true;
    else if (ENUM_KEYS.has(key) && v.length < 64) (node.values ??= {})[v] = (node.values[v] ?? 0) + 1;
  } else if (t === "boolean" && ENUM_KEYS.has(key)) (node.values ??= {})[String(v)] = 1;
  else if (t === "array") {
    node.items ??= newNode();
    for (const x of v) observe(node.items, x, path + "[]", key);
  } else if (t === "object") {
    if (OPAQUE_PATHS.some((re) => re.test(path))) { node.opaque = true; return; }
    const keys = Object.keys(v);
    if (MAP_PATHS.some((re) => re.test(path)) || looksLikeDataKeys(keys)) {
      node.map ??= newNode();
      for (const k of keys) observe(node.map, v[k], path + ".*", k);
    } else {
      node.props ??= {};
      node.objN = (node.objN ?? 0) + 1;
      for (const k of keys) observe((node.props[k] ??= newNode()), v[k], `${path}.${k}`, k);
    }
  }
}

function readTranscript(file) {
  return readFileSync(file, "utf8").split("\n").filter(Boolean).map((l) => {
    const o = JSON.parse(l);
    return o.dir && ("msg" in o || "raw" in o) ? { dir: o.dir, msg: o.msg } : { dir: "in", msg: o };
  });
}

function infer(files) {
  const shapes = {};
  for (const f of files) {
    const key = keyer();
    for (const { dir, msg } of readTranscript(f)) {
      const k = key(dir, msg);
      observe((shapes[k] ??= newNode()), msg, k + "$", "");
    }
  }
  return shapes;
}

// Render a node as compact JSON-schema-ish (for humans / golden files).
function render(node) {
  const out = {};
  const types = Object.keys(node.types);
  out.type = types.length === 1 ? types[0] : types.sort();
  if (node.opaque) out.opaque = true;
  if (node.values) out.enum = Object.keys(node.values).sort();
  if (node.formats) out.format = Object.keys(node.formats).sort();
  if (node.props) {
    out.properties = {};
    for (const [k, c] of Object.entries(node.props).sort()) {
      out.properties[k] = render(c);
      if (c.n < node.objN) out.properties[k].optional = true;
    }
  }
  if (node.map) out.additionalProperties = render(node.map);
  if (node.items && node.items.n) out.items = render(node.items);
  return out;
}

// ---------- diff ----------
// Severity:
//   BREAK  - something the plugin may rely on disappeared or changed type
//   ADOPT  - new message key / field / enum value: candidate UI feature
//   INFO   - required->optional, optional->required and similar
function diff(a, b, path, out) {
  const ta = Object.keys(a.types ?? {}), tb = Object.keys(b.types ?? {});
  for (const t of tb) if (!ta.includes(t)) out.push(["ADOPT", path, `new type ${t}`]);
  for (const t of ta) if (!tb.includes(t)) out.push(["BREAK", path, `type ${t} no longer seen`]);
  for (const v of b.enum ?? []) if (!(a.enum ?? []).includes(v)) out.push(["ADOPT", path, `new enum value ${JSON.stringify(v)}`]);
  for (const v of a.enum ?? []) if (!(b.enum ?? []).includes(v)) out.push(["INFO", path, `enum value ${JSON.stringify(v)} not seen`]);
  const pa = a.properties ?? {}, pb = b.properties ?? {};
  for (const k of Object.keys(pb)) {
    if (!(k in pa)) out.push(["ADOPT", `${path}.${k}`, `new field (${[].concat(pb[k].type).join("|")})${pb[k].optional ? " optional" : ""}`]);
    else {
      if (!pa[k].optional && pb[k].optional) out.push(["BREAK", `${path}.${k}`, "required -> optional"]);
      if (pa[k].optional && !pb[k].optional) out.push(["INFO", `${path}.${k}`, "optional -> always present"]);
      diff(pa[k], pb[k], `${path}.${k}`, out);
    }
  }
  for (const k of Object.keys(pa)) if (!(k in pb)) out.push(["BREAK", `${path}.${k}`, "field removed"]);
  if (a.items && b.items) diff(a.items, b.items, `${path}[]`, out);
  if (a.additionalProperties && b.additionalProperties) diff(a.additionalProperties, b.additionalProperties, `${path}.*`, out);
  return out;
}

// ---------- sequence signature ----------
// Message keys in order, with runs of identical keys collapsed to "key xN" -> "key+".
// Exact N is dropped because token streaming is nondeterministic.
function seq(file) {
  const key = keyer();
  const ks = readTranscript(file).map(({ dir, msg }) => key(dir, msg)).filter((k) => !/keep_alive/.test(k));
  const out = [];
  for (const k of ks) {
    const last = out[out.length - 1];
    if (last && (last === k || last === k + "+")) out[out.length - 1] = k + "+";
    else out.push(k);
  }
  return out;
}

// ---------- catalog (values, not shapes) ----------
function catalog(results) {
  const r = results.results ?? results;
  const i = r.initialize?.response ?? {};
  const names = (xs, f = (x) => x.name) => (xs ?? []).map(f).sort();
  const enumLeak = (e) => (e?.match(/must be one of (.*)$/)?.[1] ?? "").split(/,\s*/).filter(Boolean);
  return {
    initialize_keys: Object.keys(i).sort(),
    builtin_commands: names((i.commands ?? []).filter((c) => c.builtin)),
    other_commands: names((i.commands ?? []).filter((c) => !c.builtin)),
    agents: names(i.agents),
    output_styles: [...(i.available_output_styles ?? [])].sort(),
    models: names(i.models, (m) => `${m.value} -> ${m.resolvedModel}`),
    unavailable_models: names(i.unavailable_models, (m) => `${m.value}: ${m.description}`),
    model_capability_keys: [...new Set((i.models ?? []).flatMap(Object.keys))].sort(),
    permission_modes: enumLeak(r.set_permission_mode_bogus?.error),
    hook_events: names(r.extra_get_hooks_listing?.response?.eventCatalog),
    accepted_subtypes: Object.entries(r).filter(([, v]) => v.subtype === "success").map(([, v]) => v.request?.subtype).filter(Boolean).sort(),
    rejected_subtypes: Object.entries(r).filter(([, v]) => /Unsupported control request subtype/.test(v.error ?? "")).map(([, v]) => v.request?.subtype).sort(),
  };
}
function catdiff(a, b) {
  const out = [];
  for (const k of new Set([...Object.keys(a), ...Object.keys(b)])) {
    const A = new Set(a[k] ?? []), B = new Set(b[k] ?? []);
    for (const x of B) if (!A.has(x)) out.push(`+ ${k}: ${x}`);
    for (const x of A) if (!B.has(x)) out.push(`- ${k}: ${x}`);
  }
  return out;
}

// ---------- CLI ----------
const [cmd, ...rest] = process.argv.slice(2);
const J = (f) => JSON.parse(readFileSync(f, "utf8"));
if (cmd === "infer") {
  const [out, ...files] = rest;
  const shapes = infer(files);
  const rendered = Object.fromEntries(Object.entries(shapes).sort().map(([k, v]) => [k, render(v)]));
  writeFileSync(out, JSON.stringify(rendered, null, 2));
  console.log(`${Object.keys(rendered).length} message keys -> ${out}`);
} else if (cmd === "diff") {
  const [oa, ob] = rest.map(J);
  const out = [];
  for (const k of Object.keys(ob)) if (!(k in oa)) out.push(["ADOPT", k, "new message key"]);
  for (const k of Object.keys(oa)) if (!(k in ob)) out.push(["INFO", k, "message key not observed (scenario coverage or removal)"]);
  for (const k of Object.keys(ob)) if (k in oa) diff(oa[k], ob[k], k, out);
  const order = { BREAK: 0, ADOPT: 1, INFO: 2 };
  out.sort((x, y) => order[x[0]] - order[y[0]] || x[1].localeCompare(y[1]));
  for (const [s, p, d] of out) console.log(`${s.padEnd(5)} ${p}  ${d}`);
  process.exitCode = out.some((x) => x[0] === "BREAK") ? 1 : 0;
} else if (cmd === "seq") {
  console.log(seq(rest[0]).join("\n"));
} else if (cmd === "catalog") {
  writeFileSync(rest[0], JSON.stringify(catalog(J(rest[1])), null, 2));
  console.log(`catalog -> ${rest[0]}`);
} else if (cmd === "catdiff") {
  console.log(catdiff(J(rest[0]), J(rest[1])).join("\n") || "(no catalog changes)");
} else {
  console.error("usage: shape.mjs infer|diff|seq|catalog|catdiff ...");
  process.exitCode = 2;
}
