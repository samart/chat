// Generates Go wire types from sdk.d.ts (syntactic TS AST walk, no type checker).
// Usage: node gen.mjs <ts-node_modules-dir> <sdk.d.ts> > types_gen.go
import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(process.argv[2] + '/');
const ts = require('typescript');
const file = process.argv[3];
const sf = ts.createSourceFile(file, fs.readFileSync(file, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);

// ---------- index ----------
const decls = new Map();
sf.forEachChild((n) => {
  if ((ts.isTypeAliasDeclaration(n) || ts.isInterfaceDeclaration(n)) && n.name) {
    if (!decls.has(n.name.text)) decls.set(n.name.text, n);
  }
});
const isExported = (n) => !!n.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword);
const refName = (t) => t.typeName.getText(sf).replace(/^coreTypes\./, '');

// ---------- naming ----------
const RENAME = {
  SDKMessage: 'Message',
  SDKSystemMessage: 'SystemInitMessage',
  SDKPartialAssistantMessage: 'PartialAssistantMessage',
  SDKResultMessage: 'ResultMessage',
  SDKControlRequest: 'ControlRequest',
  SDKControlResponse: 'ControlResponse',
  SDKControlRequestInner: 'ControlRequestInner',
  ControlResponse: 'ControlSuccessResponse',
  ControlErrorResponse: 'ControlErrorResponse',
  SDKControlCancelRequest: 'ControlCancelRequest',
  SDKKeepAliveMessage: 'KeepAliveMessage',
  SDKControlPermissionRequest: 'CanUseToolRequest',
  SDKHookCallbackRequest: 'HookCallbackRequest',
  SDKHookCallbackMatcher: 'InitializeHookMatcher',
  SDKControlElicitationRequest: 'ElicitationControlRequest',
  SDKControlRequestUserDialogRequest: 'UserDialogControlRequest',
  SDKControlInitializeRequest: 'InitializeRequest',
  SDKControlInitializeResponse: 'InitializeResponse',
  SDKUserMessage: 'UserMessage',
  SDKUserMessageReplay: 'UserMessageReplay',
  SDKAssistantMessage: 'AssistantMessage',
  SDKSessionInfo: 'SessionInfo',
  SDKMessageOrigin: 'MessageOrigin',
  SDKStatus: 'Status',
  SdkBeta: 'Beta',
  SdkPluginConfig: 'PluginConfig',
  StdoutMessage: 'StdoutMessage',
  McpServerConfigForProcessTransport: 'McpServerConfigForProcessTransport',
};
function goName(n) {
  if (RENAME[n]) return RENAME[n];
  let g = n.replace(/^SDKControl/, '').replace(/^SDK/, '').replace(/^Sdk/, '');
  return g;
}
const pascal = (s) => s.replace(/(^|[_\-\s$.])([a-zA-Z0-9])/g, (_, __, c) => c.toUpperCase()).replace(/[^A-Za-z0-9]/g, '');
const INITIALISMS = { Id: 'ID', Ids: 'IDs', Url: 'URL', Uri: 'URI', Uuid: 'UUID', Uuids: 'UUIDs', Api: 'API', Json: 'JSON', Mcp: 'MCP', Usd: 'USD', Ms: 'Ms', Ttft: 'TTFT', Lsp: 'LSP', Cwd: 'Cwd' };
function fieldGoName(name) {
  let p = pascal(name.replace(/^_/, 'X_'));
  // apply initialisms on word boundaries (Pascal chunks)
  p = p.replace(/[A-Z][a-z0-9]*/g, (w) => (INITIALISMS[w] && w !== 'Ms' ? INITIALISMS[w] : w));
  if (/^[0-9]/.test(p)) p = 'F' + p;
  return p;
}

// ---------- config ----------
const EXCLUDE = new Set(['Options', 'Settings', 'Query', 'SessionStore', 'SpareProcess', 'SpawnedProcess', 'SpawnOptions', 'Transport', 'WarmQuery',
  'ClaimOptions', 'CanUseTool', 'HookCallback', 'HookCallbackMatcher', 'OnElicitation', 'OnUserDialog', 'SdkMcpToolDefinition', 'CreateSdkMcpServerOptions',
  'McpSdkServerConfigWithInstance', 'McpServerConfig', 'AnyZodRawShape', 'InferShape', 'SandboxCredentialsConfig', 'SandboxFilesystemConfig',
  'SandboxIgnoreViolations', 'SandboxNetworkConfig', 'SandboxSettings', 'ElicitationResult', 'HookJSONOutput', 'NonNullableUsage', 'AsyncHookJSONOutput', 'SyncHookJSONOutput']);
// Host-side (non-wire) object types: plain structs without raw preservation.
const HOST_ONLY = new Set(['ElicitationRequest', 'UserDialogRequest', 'ListSessionsOptions', 'GetSessionInfoOptions', 'GetSessionMessagesOptions',
  'GetSubagentMessagesOptions', 'ListSubagentsOptions', 'SessionMutationOptions', 'ForkSessionOptions', 'ForkSessionResult', 'ImportSessionToStoreOptions',
  'DeniedModelsMatch', 'ResolvedSettings', 'ResolveSettingsOptions', 'ProvenanceEntry', 'SessionKey', 'ToolConfig', 'SDKSessionInfo', 'SessionSummaryEntry']);
// TS external / special references -> Go type
const EXTERNAL = {
  UUID: 'string', BetaMessage: 'APIMessage', BetaRawMessageStreamEvent: 'StreamEvent', MessageParam: 'MessageParam', BetaUsage: 'Usage',
  NonNullableUsage: 'Usage', JSONRPCMessage: 'json.RawMessage', ElicitResult: 'ElicitationResult', ElicitationResult: 'ElicitationResult',
  CallToolResult: 'CallToolResult', ToolAnnotations: 'ToolAnnotations', Settings: 'Settings', SandboxSettings: 'SandboxSettings',
  HookJSONOutput: 'HookJSONOutput', SessionStore: 'SessionStore', McpServerConfig: 'McpServerConfig', AbortSignal: null, AbortController: null,
};

// ---------- integer heuristic ----------
const INT_RE = /(^|_)(tokens?|count|turns|attempt|retries|index|iterations|uses|depth|bytes|requests|lines_added|lines_removed|exit_code|error_status|status_code|pre_tokens|post_tokens|size)$|(Tokens|Count|Turns|Size|Bytes|Requests|Commands|Skills)$|^(tokens|attempt|maxTurns|lastModified|createdAt|mtime|fileSize|insertions|deletions|skippedLinks|hookCount|policyHookCount|error_count|api_error_status|squareFullness_never)$/;
const FLOAT_OVERRIDE = /usd|USD|cost|Cost|pct|percent|percentage|utilization|Utilization|_ms$|Ms$|seconds|resetsAt|ResetsAt|credits|limit|Limit|threshold|Threshold|fullness/;
function numType(field) {
  if (!field) return 'float64';
  if (FLOAT_OVERRIDE.test(field) && !/tokens|Tokens/.test(field)) return 'float64';
  return INT_RE.test(field) ? 'int' : 'float64';
}

// ---------- doc ----------
function docOf(node) {
  const jd = node.jsDoc?.[node.jsDoc.length - 1];
  if (!jd) return '';
  let c = typeof jd.comment === 'string' ? jd.comment : (jd.comment || []).map((x) => x.text || x.getText?.(sf) || '').join('');
  c = (c || '').replace(/\s+/g, ' ').trim();
  const tags = (jd.tags || []).map((t) => t.tagName.text).filter((t) => ['deprecated', 'alpha', 'default'].includes(t));
  let first = c.split(/(?<=[.!?])\s(?=[A-Z`'"(])/)[0] || '';
  if (first.length > 220) first = first.slice(0, 217).replace(/\s+\S*$/, '') + '...';
  for (const t of jd.tags || []) {
    if (t.tagName.text === 'default') {
      const v = typeof t.comment === 'string' ? t.comment : (t.comment || []).map((x) => x.text).join('');
      first += ` Default: ${String(v).trim()}.`;
    }
  }
  if (tags.includes('deprecated')) first = 'Deprecated: ' + first;
  if (tags.includes('alpha')) first += ' (alpha)';
  return first.replace(/\*\//g, '* /');
}

// ---------- model ----------
const out = [];            // emitted Go chunks
const emitted = new Map(); // goName -> true
const shapeNames = new Map(); // shape hash -> goName (dedupe inline structs)
const literalAliases = new Map(); // ts alias -> goName (string enums)
const unionAliases = new Set();
const stringConsts = [];

function stripParens(t) { while (t && ts.isParenthesizedTypeNode(t)) t = t.type; return t; }
function isNullish(t) {
  t = stripParens(t);
  return (ts.isLiteralTypeNode(t) && t.literal.kind === ts.SyntaxKind.NullKeyword) || t.kind === ts.SyntaxKind.UndefinedKeyword || t.kind === ts.SyntaxKind.NullKeyword;
}
function strLit(t) { t = stripParens(t); return ts.isLiteralTypeNode(t) && ts.isStringLiteral(t.literal) ? t.literal.text : null; }
function boolLit(t) { t = stripParens(t); return ts.isLiteralTypeNode(t) && (t.literal.kind === ts.SyntaxKind.TrueKeyword || t.literal.kind === ts.SyntaxKind.FalseKeyword); }
function unionMembers(t) { t = stripParens(t); if (ts.isUnionTypeNode(t)) return t.types.flatMap(unionMembers); return [t]; }
function literalValuesOf(t) {
  const ms = unionMembers(t).filter((m) => !isNullish(m));
  const v = ms.flatMap((m) => {
    const x = strLit(m); if (x !== null) return [x];
    const mm = stripParens(m);
    if (ts.isTypeReferenceNode(mm)) { const d = decls.get(refName(mm)); if (d && ts.isTypeAliasDeclaration(d)) { const r = literalValuesOf(d.type); if (r) return r; } }
    return [null];
  });
  return v.every((x) => x !== null) ? v : null;
}

// Resolve a type node to a list of property signatures (for object-like types). Returns null if not object-like.
function propsOf(t, seen = new Set()) {
  t = stripParens(t);
  if (ts.isTypeLiteralNode(t)) return { props: t.members.filter((m) => ts.isPropertySignature(m)), index: t.members.some((m) => ts.isIndexSignatureDeclaration(m)) };
  if (ts.isIntersectionTypeNode(t)) {
    const all = { props: [], index: false };
    for (const x of t.types) {
      const p = propsOf(x, seen); if (!p) return null;
      for (const pp of p.props) { const nm = pp.name.getText(sf); const i = all.props.findIndex((q) => q.name.getText(sf) === nm); if (i >= 0) all.props[i] = pp; else all.props.push(pp); }
      all.index ||= p.index;
    }
    return all;
  }
  if (ts.isTypeReferenceNode(t)) {
    const n = refName(t);
    const d = decls.get(n);
    if (!d || seen.has(n)) return null;
    seen.add(n);
    if (ts.isInterfaceDeclaration(d)) return { props: d.members.filter((m) => ts.isPropertySignature(m)), index: d.members.some((m) => ts.isIndexSignatureDeclaration(m)) };
    return propsOf(d.type, seen);
  }
  return null;
}

// Discriminator detection for a union of object-like members.
function discriminate(members) {
  const infos = members.map((m) => ({ node: m, p: propsOf(m) }));
  if (infos.some((i) => !i.p)) return null;
  const keyCand = ['type', 'subtype', 'kind', 'behavior', 'hook_event_name', 'hookEventName'];
  const keys = [];
  for (const k of keyCand) {
    const vals = infos.map((i) => { const p = i.p.props.find((x) => x.name.getText(sf).replace(/['"]/g, '') === k); return p ? literalValuesOf(p.type) : undefined; });
    if (vals.every((v) => v === undefined)) continue;
    if (vals.some((v) => v === null)) continue;
    keys.push(k);
  }
  if (!keys.length) return null;
  return { keys, infos };
}

function flattenObjUnion(ms, seen = new Set()) {
  const outm = [];
  for (const m of ms) {
    const mm = stripParens(m);
    if (ts.isTypeReferenceNode(mm)) {
      const n = refName(mm); const d = decls.get(n);
      if (d && ts.isTypeAliasDeclaration(d) && ts.isUnionTypeNode(stripParens(d.type)) && !literalValuesOf(d.type) && !seen.has(n)) {
        seen.add(n); outm.push(...flattenObjUnion(unionMembers(d.type).filter((x) => !isNullish(x)), seen)); continue;
      }
    }
    outm.push(m);
  }
  return outm;
}
function memberTsName(m, owner, idx) {
  m = stripParens(m);
  if (ts.isTypeReferenceNode(m)) return refName(m);
  return null;
}

// Convert a TS type node to a Go type. ctx = Go owner name, field = TS field name.
function goType(t, ctx, field) {
  t = stripParens(t);
  // union handling
  if (ts.isUnionTypeNode(t)) {
    const ms = unionMembers(t);
    const nonNull = ms.filter((m) => !isNullish(m));
    const nullable = nonNull.length !== ms.length;
    if (nonNull.length === 1) { const r = goType(nonNull[0], ctx, field); return { ...r, nullable: nullable || r.nullable }; }
    const lits = nonNull.map(strLit);
    if (lits.every((x) => x !== null)) return { t: 'string', kind: 'string', nullable, enumVals: lits };
    if (nonNull.every((m) => boolLit(m) || m.kind === ts.SyntaxKind.BooleanKeyword)) return { t: 'bool', kind: 'bool', nullable };
    // object union?
    const d = discriminate(nonNull);
    if (d) {
      const name = ctx + fieldGoName(field || 'Value');
      emitUnion(name, null, nonNull, d, `inline union (field ${field} of ${ctx})`);
      return { t: name, kind: 'union', nullable };
    }
    return { t: 'json.RawMessage', kind: 'raw', nullable: false, note: 'union: ' + printType(t) };
  }
  switch (t.kind) {
    case ts.SyntaxKind.StringKeyword: return { t: 'string', kind: 'string' };
    case ts.SyntaxKind.NumberKeyword: { const n = numType(field); return { t: n, kind: 'num' }; }
    case ts.SyntaxKind.BooleanKeyword: return { t: 'bool', kind: 'bool' };
    case ts.SyntaxKind.UnknownKeyword: case ts.SyntaxKind.AnyKeyword: return { t: 'json.RawMessage', kind: 'raw' };
  }
  if (ts.isLiteralTypeNode(t)) {
    if (ts.isStringLiteral(t.literal)) return { t: 'string', kind: 'string', literal: t.literal.text };
    if (boolLit(t)) return { t: 'bool', kind: 'bool', trueLit: t.literal.kind === ts.SyntaxKind.TrueKeyword };
    if (ts.isNumericLiteral(t.literal)) return { t: 'float64', kind: 'num' };
  }
  if (ts.isArrayTypeNode(t)) {
    const e = goType(t.elementType, ctx, singular(field));
    return { t: '[]' + (e.nullable && e.kind !== 'raw' ? '*' : '') + e.t, kind: 'slice' };
  }
  if (ts.isTypeLiteralNode(t) || ts.isIntersectionTypeNode(t)) {
    const p = propsOf(t);
    if (!p) return { t: 'json.RawMessage', kind: 'raw' };
    const name = ctx + fieldGoName(field || 'Value');
    const g = emitStruct(name, null, p, `inline object (field ${field} of ${ctx})`, false, null);
    return { t: g, kind: 'struct' };
  }
  if (ts.isIndexedAccessTypeNode(t)) return { t: 'json.RawMessage', kind: 'raw', note: printType(t) };
  if (ts.isMappedTypeNode(t)) return { t: 'map[string]any', kind: 'map' };
  if (ts.isTypeReferenceNode(t)) {
    const n = refName(t);
    const args = t.typeArguments || [];
    if (n === 'Array' || n === 'ReadonlyArray') { const e = goType(args[0], ctx, singular(field)); return { t: '[]' + e.t, kind: 'slice' }; }
    if (n === 'Record' || n === 'Partial' && args[0] && ts.isTypeReferenceNode(stripParens(args[0])) && refName(stripParens(args[0])) === 'Record') {
      const rec = n === 'Record' ? t : stripParens(args[0]);
      const [k, v] = rec.typeArguments;
      let kt = 'string';
      if (ts.isTypeReferenceNode(stripParens(k))) { const kn = refName(stripParens(k)); if (kn === 'HookEvent') kt = 'HookEvent'; }
      if (stripParens(v).kind === ts.SyntaxKind.UnknownKeyword) return { t: 'map[string]any', kind: 'map' };
      const vv = goType(v, ctx, singular(field));
      return { t: `map[${kt}]${vv.nullable && vv.kind !== 'raw' && vv.kind !== 'slice' ? '*' : ''}${vv.t}`, kind: 'map' };
    }
    if (n === 'Partial' || n === 'Readonly' || n === 'NonNullable') return goType(args[0], ctx, field);
    if (n in EXTERNAL) { const g = EXTERNAL[n]; if (!g) return { skip: true }; return { t: g, kind: g === 'string' ? 'string' : g.startsWith('json') ? 'raw' : (['Settings'].includes(g) ? 'map' : 'struct') }; }
    const d = decls.get(n);
    if (!d) return { t: 'json.RawMessage', kind: 'raw', note: 'unresolved ' + n };
    ensureDecl(n);
    if (literalAliases.has(n)) return { t: literalAliases.get(n), kind: 'string', nullable: literalAliasNullable.has(n) };
    if (rawAliases.has(n)) return { t: goName(n), kind: 'raw' };
    if (unionAliases.has(n)) return { t: goName(n), kind: 'union' };
    return { t: goName(n), kind: 'struct' };
  }
  if (ts.isFunctionTypeNode(t)) return { skip: true };
  return { t: 'json.RawMessage', kind: 'raw', note: 'unhandled ' + ts.SyntaxKind[t.kind] };
}
const literalAliasNullable = new Set();
const rawAliases = new Set();
function singular(f) { return f ? f.replace(/ies$/, 'y').replace(/([^s])s$/, '$1') : f; }
function printType(t) { return t.getText(sf).replace(/\s+/g, ' '); }

function ensureDecl(n) {
  const g = goName(n);
  if (emitted.has(g) || EXCLUDE.has(n)) return;
  const d = decls.get(n);
  if (!d) return;
  emitDecl(n, d);
}

function emitDecl(n, d) {
  const g = goName(n);
  if (emitted.has(g)) return;
  const doc = docOf(d);
  const src = `TS: ${n}${isExported(d) ? '' : ' (not exported; wire type)'}`;
  if (ts.isInterfaceDeclaration(d)) {
    emitStruct(g, n, { props: d.members.filter((m) => ts.isPropertySignature(m)), index: false }, doc, HOST_ONLY.has(n), src);
    return;
  }
  const t = stripParens(d.type);
  const lits = literalValuesOf(t);
  if (lits && lits.length >= 1) {
    emitted.set(g, true);
    literalAliases.set(n, g);
    if (unionMembers(t).some(isNullish)) literalAliasNullable.add(n);
    const lines = [`// ${g} is a string enum. ${doc ? doc + ' ' : ''}(${src})`, `type ${g} string`, '', 'const ('];
    for (const v of lits) lines.push(`\t${g}${pascal(v.replace(/^\//, 'Slash ').replace(/ /g, '_')) || 'Empty'} ${g} = ${JSON.stringify(v)}`);
    lines.push(')', '');
    out.push(lines.join('\n'));
    return;
  }
  if (ts.isUnionTypeNode(t)) {
    const ms = flattenObjUnion(unionMembers(t).filter((m) => !isNullish(m)));
    const dd = discriminate(ms);
    if (dd) { unionAliases.add(n); emitUnion(g, n, ms, dd, doc, src); return; }
    emitted.set(g, true);
    rawAliases.add(n);
    out.push(`// ${g} is kept as raw JSON: ${doc ? doc + ' ' : ''}TS type is ${printType(t).slice(0, 200)}. (${src})\ntype ${g} = json.RawMessage\n`);
    return;
  }
  const p = propsOf(t);
  if (p) { emitStruct(g, n, p, doc, HOST_ONLY.has(n), src); return; }
  // alias to another named type
  if (ts.isTypeReferenceNode(t)) {
    const r = goType(t, g, null);
    if (r.skip) return;
    emitted.set(g, true);
    out.push(`// ${g} aliases ${r.t}. ${doc} (${src})\ntype ${g} = ${r.t}\n`);
    return;
  }
  emitted.set(g, true);
  rawAliases.add(n);
  out.push(`// ${g} is kept as raw JSON (TS: ${printType(t).slice(0, 160)}). ${doc} (${src})\ntype ${g} = json.RawMessage\n`);
}

function shapeKey(p) { return p.props.map((x) => x.getText(sf).replace(/\/\*\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ')).join('|'); }

function emitStruct(g, tsName, p, doc, hostOnly, src) {
  if (!tsName) {
    const k = shapeKey(p);
    if (shapeNames.has(k)) return shapeNames.get(k);
    shapeNames.set(k, g);
  }
  if (emitted.has(g)) return g;
  emitted.set(g, true);
  const fields = [];
  const discs = [];
  for (const m of p.props) {
    const name = m.name.getText(sf).replace(/^['"]|['"]$/g, '');
    if (!m.type) continue;
    const r = goType(m.type, g, name);
    if (r.skip) continue;
    const opt = !!m.questionToken;
    let typ = r.t, tag = name;
    const ptrable = ['num', 'bool', 'struct'].includes(r.kind) || (r.kind === 'string' && r.nullable) || (r.kind === 'union' && r.nullable && false);
    if (r.kind === 'bool' && r.trueLit && opt) { /* optional `true` literal: plain bool + omitempty */ }
    else if (opt && ptrable) typ = '*' + typ;
    else if (!opt && r.nullable && ['num', 'bool', 'struct', 'string'].includes(r.kind)) typ = '*' + typ;
    if (opt) tag += ',omitempty';
    if (!opt && r.literal !== undefined) discs.push([fieldGoName(name), r.literal]);
    let fd = docOf(m);
    const notes = [];
    if (r.enumVals) notes.push('one of ' + r.enumVals.map((v) => JSON.stringify(v)).join(', '));
    if (r.literal !== undefined) notes.push(`always ${JSON.stringify(r.literal)}`);
    if (r.trueLit) notes.push('TS literal `true`');
    if (r.note) notes.push(r.note.slice(0, 160));
    if (r.nullable && !opt) notes.push('nullable');
    if (r.nullable && opt) notes.push('optional, nullable');
    const comment = [fd, notes.length ? '(' + notes.join('; ') + ')' : ''].filter(Boolean).join(' ');
    fields.push({ go: fieldGoName(name), typ, tag, comment });
  }
  // de-dup go field names
  const seenF = new Map();
  for (const f of fields) { if (seenF.has(f.go)) f.go += 'X'; seenF.set(f.go, 1); }
  const lines = [];
  const head = `// ${g} ${doc ? '- ' + doc : ''}${src ? ' (' + src + ')' : ''}`;
  lines.push(wrapComment(head));
  lines.push(`type ${g} struct {`);
  for (const f of fields) {
    if (f.comment) lines.push(wrapComment('\t// ' + f.comment));
    lines.push(`\t${f.go} ${f.typ} \`json:"${f.tag}"\``);
  }
  if (!hostOnly) {
    lines.push('\t// Raw is the JSON this value was decoded from. Fields not declared above survive a decode/encode round trip.');
    lines.push('\tRaw json.RawMessage `json:"-"`');
  }
  lines.push('}', '');
  if (!hostOnly) {
    lines.push(`func (v *${g}) UnmarshalJSON(b []byte) error { type plain ${g}; return unmarshalKeep(b, (*plain)(v), &v.Raw) }`);
    const fill = discs.map(([f, val]) => `if v.${f} == "" { v.${f} = ${JSON.stringify(val)} }; `).join('');
    lines.push(`func (v ${g}) MarshalJSON() ([]byte, error) { type plain ${g}; ${fill}return marshalKeep(plain(v), v.Raw) }`, '');
  }
  out.push(lines.join('\n'));
  return g;
}

function wrapComment(s) { return s.replace(/\s+$/, ''); }

// Union emission. members: TS nodes. d: discriminator info.
const unionSpecs = [];
function emitUnion(g, tsName, members, d, doc, src) {
  if (emitted.has(g)) return;
  emitted.set(g, true);
  const variants = [];
  members.forEach((m, i) => {
    let vt;
    const mm = stripParens(m);
    if (ts.isTypeReferenceNode(mm)) {
      const n = refName(mm);
      if (EXTERNAL[n]) vt = EXTERNAL[n];
      else { ensureDecl(n); vt = goName(n); }
    } else {
      const p = propsOf(mm);
      const lit = d.keys.map((k) => { const pp = p.props.find((x) => x.name.getText(sf).replace(/['"]/g, '') === k); return pp ? literalValuesOf(pp.type)?.[0] : null; }).filter(Boolean);
      vt = emitStruct(g + pascal(lit.join('_') || String(i)), null, p, `variant of ${g}`, false, null);
    }
    const p = d.infos[i].p;
    const conds = {};
    for (const k of d.keys) {
      const pp = p.props.find((x) => x.name.getText(sf).replace(/['"]/g, '') === k);
      conds[k] = pp ? { vals: literalValuesOf(pp.type), optional: !!pp.questionToken } : null;
    }
    variants.push({ vt, conds });
  });
  unionSpecs.push({ g, tsName, keys: d.keys, variants, doc, src });
}

function renderUnions() {
  for (const u of unionSpecs) {
    const L = [];
    const dfields = u.keys.map((k) => fieldGoName(k));
    L.push(`// ${u.g} is a discriminated union${u.doc ? ': ' + u.doc : ''}${u.src ? ' (' + u.src + ')' : ''}.`);
    L.push(`// Discriminator key(s): ${u.keys.map((k) => '"' + k + '"').join(', ')}. Variants: ${u.variants.map((v) => v.vt).join(', ')}.`);
    L.push(`// Use the As<Variant> accessors or Decode; build one for sending with New${u.g}.`);
    L.push(`type ${u.g} struct {`);
    u.keys.forEach((k, i) => L.push(`\t// ${dfields[i]} is the "${k}" discriminator (empty if absent).\n\t${dfields[i]} string \`json:"-"\``));
    L.push('\t// Raw is the complete JSON of the value.\n\tRaw json.RawMessage `json:"-"`', '}', '');
    L.push(`func (u *${u.g}) UnmarshalJSON(b []byte) error { return unmarshalUnion(b, &u.Raw, []string{${u.keys.map((k) => JSON.stringify(k)).join(', ')}}, ${dfields.map((f) => '&u.' + f).join(', ')}) }`);
    L.push(`func (u ${u.g}) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }`);
    L.push(`// New${u.g} wraps a variant value (or any JSON-marshalable value) as a ${u.g}.`);
    L.push(`func New${u.g}(v any) (${u.g}, error) { var u ${u.g}; err := wrapUnion(v, &u); return u, err }`);
    const seenAcc = new Set();
    const condOf = (v) => {
      const parts = [];
      u.keys.forEach((k, i) => {
        const c = v.conds[k];
        if (!c) { parts.push(`u.${dfields[i]} == ""`); return; }
        const ors = c.vals.map((x) => `u.${dfields[i]} == ${JSON.stringify(x)}`);
        if (c.optional) ors.push(`u.${dfields[i]} == ""`);
        parts.push(ors.length > 1 ? '(' + ors.join(' || ') + ')' : ors[0]);
      });
      return parts.join(' && ');
    };
    // When two variants share discriminators (e.g. user vs user replay), more specific first is handled by overrides.
    for (const v of u.variants) {
      if (seenAcc.has(v.vt)) continue; seenAcc.add(v.vt);
      const cond = UNION_COND_OVERRIDE[u.g + '.' + v.vt] || condOf(v);
      L.push(`// As${v.vt} decodes u as ${v.vt}; it returns ErrVariantMismatch if the discriminator does not match.`);
      L.push(`func (u ${u.g}) As${v.vt}() (*${v.vt}, error) { return asVariant[${v.vt}](u.Raw, ${cond}, "${u.g}", "${v.vt}") }`);
      v.cond = cond;
    }
    L.push(`// Decode returns a pointer to the matching variant struct, or (nil, ErrUnknownVariant) for a discriminator this package does not know (u.Raw still holds it).`);
    L.push(`func (u ${u.g}) Decode() (any, error) {`, '\tswitch {');
    const order = [...u.variants].sort((a, b) => (UNION_PRIORITY[u.g + '.' + b.vt] || 0) - (UNION_PRIORITY[u.g + '.' + a.vt] || 0));
    const done = new Set();
    for (const v of order) { if (done.has(v.vt)) continue; done.add(v.vt); L.push(`\tcase ${v.cond}:\n\t\treturn u.As${v.vt}()`); }
    L.push('\t}', `\treturn nil, unknownVariant("${u.g}", u.Raw)`, '}', '');
    out.push(L.join('\n'));
  }
}
const UNION_COND_OVERRIDE = {
  'Message.UserMessageReplay': 'u.Type == "user" && peekBool(u.Raw, "isReplay")',
  'Message.UserMessage': 'u.Type == "user"',
};
const UNION_PRIORITY = { 'Message.UserMessageReplay': 1 };

// ---------- drive ----------
const roots = [...decls.keys()].filter((n) => !EXCLUDE.has(n)).sort();
for (const n of roots) ensureDecl(n);
renderUnions();
process.stdout.write('// Code generated by gen.mjs from @anthropic-ai/claude-agent-sdk 0.3.325 sdk.d.ts. Hand edits belong in types.go.\n\npackage claude\n\nimport "encoding/json"\n\n' + out.join('\n'));
