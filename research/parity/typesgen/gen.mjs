// Generates Go wire types from sdk.d.ts (syntactic TS AST walk, no type checker).
// Base version = 0.3.311; additions from the newer d.ts are merged in and marked
// "// since 0.3.3xx (not in 0.3.311)".
// Usage: node gen.mjs <ts-dir> <base sdk.d.ts> <newer sdk.d.ts> <extern.d.ts> <since-log.json> > types_gen.go
import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(process.argv[2] + '/');
const ts = require('typescript');
const [, , , baseFile, newFile, externFile, logFile] = process.argv;
const parse = (f) => ts.createSourceFile(f, fs.readFileSync(f, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
const sfBase = parse(baseFile), sfNew = parse(newFile), sfExt = parse(externFile);
const BASE_V = '0.3.311';

function index(sf) {
  const m = new Map();
  sf.forEachChild((n) => { if ((ts.isTypeAliasDeclaration(n) || ts.isInterfaceDeclaration(n)) && n.name && !m.has(n.name.text)) m.set(n.name.text, n); });
  return m;
}
const declsBase = index(sfBase), declsNew = index(sfNew), declsExt = index(sfExt);
const isExported = (n) => !!n.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword);
const refName = (t) => t.typeName.getText().replace(/^coreTypes\./, '');
const txt = (n) => n.getText().replace(/\/\*\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ').trim();

// Resolve a name in the context of a node's source file. extern overrides everything.
function lookupDecl(n, ctx) {
  if (declsExt.has(n)) return declsExt.get(n);
  const sf = ctx?.getSourceFile?.();
  if (sf === sfNew) return declsNew.get(n) ?? declsBase.get(n);
  return declsBase.get(n) ?? declsNew.get(n);
}

// ---------- "since" bookkeeping ----------
// Exact release taken from the SDK changelog where it names the item; otherwise a range.
const SINCE_EXACT = {
  credential_expires_at: '0.3.318', invalid_config_file: '0.3.325', SDKSessionWritesLostMessage: '0.3.317',
  refuse_if_busy: '0.3.325', refuseIfBusy: '0.3.325', hook_on_failure_supported: '0.3.317', onFailure: '0.3.317', lossy: '0.3.317',
  archived: '0.3.325', DeniedModelsMatch: '0.3.323', machinePolicy: '0.3.323', managed_lock: '0.3.325',
  sandbox_required_unavailable: '0.3.314', initialize_not_first: '0.3.317', config_file_invalid: '0.3.318',
};
const sinceLog = [];
function since(key, kind, owner) {
  const v = SINCE_EXACT[key] || '0.3.312-0.3.325';
  sinceLog.push({ kind, owner, name: key, version: v });
  return `since ${v} (not in ${BASE_V})`;
}

// ---------- naming ----------
const RENAME = {
  SDKMessage: 'Message', SDKSystemMessage: 'SystemInitMessage', SDKPartialAssistantMessage: 'PartialAssistantMessage',
  SDKResultMessage: 'ResultMessage', SDKControlRequest: 'ControlRequest', SDKControlResponse: 'ControlResponse',
  SDKControlRequestInner: 'ControlRequestInner', ControlResponse: 'ControlSuccessResponse', ControlErrorResponse: 'ControlErrorResponse',
  SDKControlCancelRequest: 'ControlCancelRequest', SDKKeepAliveMessage: 'KeepAliveMessage', SDKControlPermissionRequest: 'CanUseToolRequest',
  SDKHookCallbackRequest: 'HookCallbackRequest', SDKHookCallbackMatcher: 'HookCallbackMatcher', SDKControlElicitationRequest: 'ElicitationControlRequest',
  SDKControlRequestUserDialogRequest: 'UserDialogControlRequest', SDKControlInitializeRequest: 'InitializeRequest',
  SDKControlInitializeResponse: 'InitializeResponse', SDKUserMessage: 'UserMessage', SDKUserMessageReplay: 'UserMessageReplay',
  SDKAssistantMessage: 'AssistantMessage', SDKSessionInfo: 'SessionInfo', SDKMessageOrigin: 'MessageOrigin', SDKStatus: 'Status',
  SdkBeta: 'Beta', SdkPluginConfig: 'PluginConfig', BetaMessage: 'APIMessage', BetaRawMessageStreamEvent: 'StreamEvent',
  NonNullableUsage: 'Usage', BetaContentBlock: 'ContentBlock', ElicitResult: 'ElicitationResult',
};
const goName = (n) => RENAME[n] || n.replace(/^SDKControl/, '').replace(/^SDK/, '').replace(/^Sdk/, '');
const pascal = (s) => s.replace(/(^|[_\-\s$.\/])([a-zA-Z0-9])/g, (_, __, c) => c.toUpperCase()).replace(/[^A-Za-z0-9]/g, '');
const INITIALISMS = { Id: 'ID', Ids: 'IDs', Url: 'URL', Uri: 'URI', Uuid: 'UUID', Uuids: 'UUIDs', Api: 'API', Json: 'JSON', Mcp: 'MCP', Usd: 'USD', Ttft: 'TTFT', Lsp: 'LSP' };
function fieldGoName(name) {
  let p = pascal(name.replace(/^_/, 'X_'));
  p = p.replace(/[A-Z][a-z0-9]*/g, (w) => INITIALISMS[w] || w);
  if (/^[0-9]/.test(p)) p = 'F' + p;
  return p;
}

// ---------- config ----------
const EXCLUDE = new Set(['Settings', 'Query', 'SessionStore', 'SpareProcess', 'SpawnedProcess', 'SpawnOptions', 'Transport', 'WarmQuery',
  'ClaimOptions', 'CanUseTool', 'HookCallback', 'HookCallbackMatcher', 'OnElicitation', 'OnUserDialog', 'SdkMcpToolDefinition', 'CreateSdkMcpServerOptions',
  'McpSdkServerConfigWithInstance', 'AnyZodRawShape', 'InferShape', 'SandboxCredentialsConfig', 'SandboxFilesystemConfig',
  'SandboxIgnoreViolations', 'SandboxNetworkConfig', 'ElicitationResult', 'HookJSONOutput', 'InMemorySessionStore']);
// Host-side (non-wire) object types: plain structs, no raw preservation.
const HOST_ONLY = new Set(['Options', 'ElicitationRequest', 'UserDialogRequest', 'ListSessionsOptions', 'GetSessionInfoOptions', 'GetSessionMessagesOptions',
  'GetSubagentMessagesOptions', 'ListSubagentsOptions', 'SessionMutationOptions', 'ForkSessionOptions', 'ForkSessionResult', 'ImportSessionToStoreOptions',
  'DeniedModelsMatch', 'ResolvedSettings', 'ResolveSettingsOptions', 'ProvenanceEntry', 'SessionKey', 'ToolConfig', 'SDKSessionInfo', 'SessionSummaryEntry']);
// References that map straight to a Go type (null = drop the field: not representable as data).
const EXTERNAL = {
  UUID: 'string', JSONRPCMessage: 'json.RawMessage', Settings: 'Settings', HookJSONOutput: 'HookJSONOutput',
  HookCallbackMatcher: 'HookCallbackMatcher', SessionStore: null, AbortSignal: null, AbortController: null, McpServer: null,
};
// Function-valued Options fields: represented as data so the host can forward them over the network.
const CALLBACK_FLAGS = {
  canUseTool: 'True when the host answers can_use_tool control requests (TS: the canUseTool callback). Implies --permission-prompt-tool=stdio.',
  onElicitation: 'True when the host answers elicitation control requests (TS: the onElicitation callback). Otherwise the SDK answers {action:"decline"}.',
  onUserDialog: 'True when the host answers request_user_dialog control requests (TS: the onUserDialog callback). Otherwise the request is left unanswered.',
  stderr: 'True when the host wants the CLI stderr stream (TS: the stderr callback).',
};

// ---------- numbers ----------
const INT_RE = /(^|_)(tokens?|count|turns|attempt|retries|index|iterations|uses|depth|bytes|requests|lines_added|lines_removed|exit_code|error_status|size)$|(Tokens|Count|Turns|Size|Bytes|Requests|Commands|Skills)$|^(attempt|maxTurns|lastModified|createdAt|mtime|fileSize|insertions|deletions|skippedLinks|hookCount|policyHookCount|error_count|api_error_status|index|batchSize|limit|offset|budgetTokens)$/;
const FLOAT_RE = /usd|USD|cost|Cost|pct|percent|percentage|utilization|Utilization|_ms$|Ms$|seconds|resetsAt|ResetsAt|credits|monthly_limit|threshold|Threshold|fullness|Fullness/;
function numType(field) {
  if (!field) return 'float64';
  if (FLOAT_RE.test(field) && !/tokens|Tokens/.test(field)) return 'float64';
  return INT_RE.test(field) ? 'int' : 'float64';
}

// ---------- doc ----------
function docOf(node) {
  const jd = node?.jsDoc?.[node.jsDoc.length - 1];
  if (!jd) return '';
  let c = typeof jd.comment === 'string' ? jd.comment : (jd.comment || []).map((x) => x.text || '').join('');
  c = (c || '').replace(/\s+/g, ' ').trim();
  let first = c.split(/(?<=[.!?])\s(?=[A-Z`'"(])/)[0] || '';
  if (first.length > 220) first = first.slice(0, 217).replace(/\s+\S*$/, '') + '...';
  for (const t of jd.tags || []) {
    const v = typeof t.comment === 'string' ? t.comment : (t.comment || []).map((x) => x.text).join('');
    if (t.tagName.text === 'default') first += ` Default: ${String(v).replace(/\s+/g, ' ').trim()}.`;
    if (t.tagName.text === 'deprecated') first = 'Deprecated: ' + first;
    if (t.tagName.text === 'alpha') first += ' (alpha)';
  }
  return first.replace(/\*\//g, '* /');
}

// ---------- type helpers ----------
const out = [];
const emitted = new Map();
const shapeNames = new Map();
const literalAliases = new Map();
const literalAliasNullable = new Set();
const unionAliases = new Set();
const rawAliases = new Set();

function stripParens(t) { while (t && ts.isParenthesizedTypeNode(t)) t = t.type; return t; }
function isNullish(t) {
  t = stripParens(t);
  return (ts.isLiteralTypeNode(t) && t.literal.kind === ts.SyntaxKind.NullKeyword) || t.kind === ts.SyntaxKind.UndefinedKeyword || t.kind === ts.SyntaxKind.NullKeyword;
}
const strLit = (t) => { t = stripParens(t); return ts.isLiteralTypeNode(t) && ts.isStringLiteral(t.literal) ? t.literal.text : null; };
const boolLit = (t) => { t = stripParens(t); return ts.isLiteralTypeNode(t) && (t.literal.kind === ts.SyntaxKind.TrueKeyword || t.literal.kind === ts.SyntaxKind.FalseKeyword); };
function unionMembers(t) { t = stripParens(t); return ts.isUnionTypeNode(t) ? t.types.flatMap(unionMembers) : [t]; }
function literalValuesOf(t) {
  if (!t) return null;
  const v = unionMembers(t).filter((m) => !isNullish(m)).flatMap((m) => {
    const x = strLit(m); if (x !== null) return [x];
    const mm = stripParens(m);
    if (ts.isTypeReferenceNode(mm)) { const d = lookupDecl(refName(mm), mm); if (d && ts.isTypeAliasDeclaration(d)) { const r = literalValuesOf(d.type); if (r) return r; } }
    return [null];
  });
  return v.length && v.every((x) => x !== null) ? v : null;
}
const propName = (p) => p.name.getText().replace(/^['"]|['"]$/g, '');

function propsOf(t, seen = new Set()) {
  t = stripParens(t);
  if (!t) return null;
  if (ts.isTypeLiteralNode(t)) return { props: t.members.filter((m) => ts.isPropertySignature(m)), index: t.members.some((m) => ts.isIndexSignatureDeclaration(m)) };
  if (ts.isIntersectionTypeNode(t)) {
    const all = { props: [], index: false };
    for (const x of t.types) {
      const p = propsOf(x, seen); if (!p) return null;
      for (const pp of p.props) { const i = all.props.findIndex((q) => propName(q) === propName(pp)); if (i >= 0) all.props[i] = pp; else all.props.push(pp); }
      all.index ||= p.index;
    }
    return all;
  }
  if (ts.isTypeReferenceNode(t)) {
    const n = refName(t); const d = lookupDecl(n, t);
    if (!d || seen.has(n)) return null;
    seen.add(n);
    if (ts.isInterfaceDeclaration(d)) return { props: d.members.filter((m) => ts.isPropertySignature(m)), index: false };
    return propsOf(d.type, seen);
  }
  return null;
}
function flattenObjUnion(ms, seen = new Set()) {
  const res = [];
  for (const m of ms) {
    const mm = stripParens(m);
    if (ts.isTypeReferenceNode(mm)) {
      const n = refName(mm); const d = lookupDecl(n, mm);
      if (d && ts.isTypeAliasDeclaration(d) && ts.isUnionTypeNode(stripParens(d.type)) && !literalValuesOf(d.type) && !seen.has(n)) {
        seen.add(n); res.push(...flattenObjUnion(unionMembers(d.type).filter((x) => !isNullish(x)), seen)); continue;
      }
    }
    res.push(m);
  }
  return res;
}
function discriminate(members) {
  const infos = members.map((m) => ({ node: m, p: propsOf(m) }));
  if (infos.some((i) => !i.p)) return null;
  const keys = [];
  for (const k of ['type', 'subtype', 'kind', 'behavior', 'hook_event_name', 'hookEventName']) {
    const vals = infos.map((i) => { const p = i.p.props.find((x) => propName(x) === k); if (!p) return undefined; const ms = unionMembers(p.type).filter((m) => !isNullish(m)).map(strLit); return ms.every((x) => x !== null) ? ms : null; });
    if (vals.every((v) => v === undefined) || vals.some((v) => v === null)) continue;
    keys.push(k);
  }
  return keys.length ? { keys, infos } : null;
}
const singular = (f) => (f ? f.replace(/ies$/, 'y').replace(/([^s])s$/, '$1') : f);
const nonNullSingle = (t) => { if (!t) return null; const ms = unionMembers(t).filter((m) => !isNullish(m)); return ms.length === 1 ? ms[0] : stripParens(t); };
const arrayElem = (t) => { t = nonNullSingle(t); if (!t) return null; if (ts.isArrayTypeNode(t)) return t.elementType; if (ts.isTypeReferenceNode(t) && ['Array', 'ReadonlyArray'].includes(refName(t))) return t.typeArguments?.[0]; return null; };
const recordVal = (t) => { t = nonNullSingle(t); if (!t || !ts.isTypeReferenceNode(t)) return null; if (refName(t) === 'Record') return t.typeArguments?.[1]; if (refName(t) === 'Partial') return recordVal(t.typeArguments?.[0]); return null; };

// Convert a TS type node to a Go type; nt = the newer-version counterpart (or null).
function goType(t, ctx, field, nt) {
  t = stripParens(t);
  if (ts.isUnionTypeNode(t)) {
    const ms = unionMembers(t);
    const nonNull = ms.filter((m) => !isNullish(m));
    const nullable = nonNull.length !== ms.length;
    if (nonNull.length === 1) { const r = goType(nonNull[0], ctx, field, nonNullSingle(nt)); return { ...r, nullable: nullable || r.nullable }; }
    const lits = nonNull.map(strLit);
    if (lits.every((x) => x !== null)) {
      const newer = (nt && literalValuesOf(nt)) || [];
      const added = newer.filter((v) => !lits.includes(v));
      return { t: 'string', kind: 'string', nullable, enumVals: lits, enumAdded: added };
    }
    if (nonNull.every((m) => boolLit(m) || m.kind === ts.SyntaxKind.BooleanKeyword)) return { t: 'bool', kind: 'bool', nullable };
    const flat = flattenObjUnion(nonNull);
    const d = discriminate(flat);
    if (d) {
      const name = ctx + fieldGoName(field || 'Value');
      emitUnion(name, null, flat, d, `inline union (field ${field} of ${ctx})`, '', []);
      return { t: name, kind: 'union', nullable };
    }
    return { t: 'json.RawMessage', kind: 'raw', note: 'TS: ' + txt(t).slice(0, 150) };
  }
  switch (t.kind) {
    case ts.SyntaxKind.StringKeyword: return { t: 'string', kind: 'string' };
    case ts.SyntaxKind.NumberKeyword: return { t: numType(field), kind: 'num' };
    case ts.SyntaxKind.BooleanKeyword: return { t: 'bool', kind: 'bool' };
    case ts.SyntaxKind.UnknownKeyword: case ts.SyntaxKind.AnyKeyword: return { t: 'json.RawMessage', kind: 'raw' };
  }
  if (ts.isLiteralTypeNode(t)) {
    if (ts.isStringLiteral(t.literal)) return { t: 'string', kind: 'string', literal: t.literal.text };
    if (boolLit(t)) return { t: 'bool', kind: 'bool', trueLit: t.literal.kind === ts.SyntaxKind.TrueKeyword };
    return { t: 'float64', kind: 'num' };
  }
  if (ts.isArrayTypeNode(t)) {
    const e = goType(t.elementType, ctx, singular(field), arrayElem(nt));
    if (e.skip) return e;
    return { t: '[]' + (e.nullable && !['raw', 'slice', 'map'].includes(e.kind) ? '*' : '') + e.t, kind: 'slice', enumVals: e.enumVals, enumAdded: e.enumAdded };
  }
  if (ts.isTypeLiteralNode(t) && t.members.length && t.members.every((m) => ts.isIndexSignatureDeclaration(m))) {
    const vt = stripParens(t.members[0].type);
    if (vt.kind === ts.SyntaxKind.UnknownKeyword) return { t: 'map[string]any', kind: 'map' };
    const vv = goType(vt, ctx, singular(field), null);
    return { t: 'map[string]' + vv.t, kind: 'map' };
  }
  if (ts.isTypeLiteralNode(t) || ts.isIntersectionTypeNode(t)) {
    const p = propsOf(t);
    if (!p) return { t: 'json.RawMessage', kind: 'raw' };
    const g = emitStruct(ctx + fieldGoName(field || 'Value'), null, p, `inline object (field ${field} of ${ctx})`, false, null, nt ? propsOf(nonNullSingle(nt)) : null, false);
    return { t: g, kind: 'struct' };
  }
  if (ts.isIndexedAccessTypeNode(t)) return { t: 'json.RawMessage', kind: 'raw', note: 'TS: ' + txt(t) };
  if (ts.isMappedTypeNode(t)) return { t: 'map[string]any', kind: 'map' };
  if (ts.isFunctionTypeNode(t)) return { skip: true, fn: true };
  if (ts.isTypeReferenceNode(t)) {
    const n = refName(t);
    const args = t.typeArguments || [];
    if (n === 'Array' || n === 'ReadonlyArray') { const e = goType(args[0], ctx, singular(field), arrayElem(nt)); return { t: '[]' + e.t, kind: 'slice' }; }
    if (n === 'Record' || (n === 'Partial' && args[0] && ts.isTypeReferenceNode(stripParens(args[0])) && refName(stripParens(args[0])) === 'Record')) {
      const rec = n === 'Record' ? t : stripParens(args[0]);
      const [k, v] = rec.typeArguments;
      const kt = ts.isTypeReferenceNode(stripParens(k)) && refName(stripParens(k)) === 'HookEvent' ? 'HookEvent' : 'string';
      if (stripParens(v).kind === ts.SyntaxKind.UnknownKeyword) return { t: 'map[string]any', kind: 'map' };
      const vv = goType(v, ctx, singular(field), recordVal(nt));
      if (vv.skip) return vv;
      return { t: `map[${kt}]${vv.nullable && !['raw', 'slice', 'map'].includes(vv.kind) ? '*' : ''}${vv.t}`, kind: 'map' };
    }
    if (['Partial', 'Readonly', 'NonNullable'].includes(n)) return goType(args[0], ctx, field, nt);
    if (n in EXTERNAL && !declsExt.has(n)) { const g = EXTERNAL[n]; if (!g) return { skip: true }; return { t: g, kind: g === 'string' ? 'string' : g.startsWith('json') ? 'raw' : g === 'Settings' ? 'map' : 'struct' }; }
    const d = lookupDecl(n, t);
    if (!d) return { t: 'json.RawMessage', kind: 'raw', note: 'TS: ' + n };
    if (ts.isTypeAliasDeclaration(d) && ts.isFunctionTypeNode(stripParens(d.type))) return { skip: true, fn: true };
    ensureDecl(n);
    if (literalAliases.has(n)) return { t: literalAliases.get(n), kind: 'string', nullable: literalAliasNullable.has(n) };
    if (rawAliases.has(n)) return { t: goName(n), kind: 'raw' };
    if (unionAliases.has(n)) return { t: goName(n), kind: 'union' };
    return { t: goName(n), kind: 'struct' };
  }
  return { t: 'json.RawMessage', kind: 'raw', note: 'TS: ' + txt(t).slice(0, 120) };
}

function ensureDecl(n) {
  if (EXCLUDE.has(n) && !declsExt.has(n)) return;
  const g = goName(n);
  if (emitted.has(g)) return;
  if (declsExt.has(n)) return emitDecl(n, declsExt.get(n), null, false, true);
  if (declsBase.has(n)) return emitDecl(n, declsBase.get(n), declsNew.get(n) || null, false, false);
  if (declsNew.has(n)) return emitDecl(n, declsNew.get(n), null, true, false);
}

function emitDecl(n, d, dNew, isNew, isExt) {
  const g = goName(n);
  if (emitted.has(g)) return;
  const doc = docOf(d);
  const src = isExt ? `hand-modelled stand-in for TS ${n}` : `TS: ${n}${isExported(d) ? '' : ', not exported (wire type)'}`;
  const sinceLine = isNew ? since(n, 'type', '') : '';
  if (ts.isInterfaceDeclaration(d)) {
    const pNew = dNew && ts.isInterfaceDeclaration(dNew) ? { props: dNew.members.filter((m) => ts.isPropertySignature(m)) } : null;
    emitStruct(g, n, { props: d.members.filter((m) => ts.isPropertySignature(m)) }, doc, HOST_ONLY.has(n), src, pNew, sinceLine);
    return;
  }
  const t = stripParens(d.type);
  if (ts.isFunctionTypeNode(t)) return;
  const lits = literalValuesOf(t);
  if (lits) {
    emitted.set(g, true);
    literalAliases.set(n, g);
    if (unionMembers(t).some(isNullish)) literalAliasNullable.add(n);
    const newer = dNew ? literalValuesOf(dNew.type) || [] : [];
    const L = [];
    if (sinceLine) L.push('// ' + sinceLine);
    L.push(`// ${g} is a string enum.${doc ? ' ' + doc : ''} (${src})`, `type ${g} string`, '', 'const (');
    const constName = (v) => `${g}${pascal(v.replace(/^\//, 'slash ').replace(/ /g, '_')) || 'Empty'}`;
    for (const v of lits) L.push(`\t${constName(v)} ${g} = ${JSON.stringify(v)}`);
    for (const v of newer.filter((v) => !lits.includes(v))) L.push(`\t${constName(v)} ${g} = ${JSON.stringify(v)} // ${since(v, 'enum value', g)}`);
    L.push(')', '');
    out.push(L.join('\n'));
    return;
  }
  if (ts.isUnionTypeNode(t)) {
    const ms = flattenObjUnion(unionMembers(t).filter((m) => !isNullish(m)));
    let added = [];
    if (dNew && ts.isUnionTypeNode(stripParens(dNew.type))) {
      const nm = flattenObjUnion(unionMembers(dNew.type).filter((m) => !isNullish(m)));
      const have = new Set(ms.map((m) => (ts.isTypeReferenceNode(stripParens(m)) ? refName(stripParens(m)) : txt(m))));
      added = nm.filter((m) => !have.has(ts.isTypeReferenceNode(stripParens(m)) ? refName(stripParens(m)) : txt(m)));
    }
    const all = [...ms, ...added];
    const dd = discriminate(all);
    if (dd) { unionAliases.add(n); emitUnion(g, n, all, dd, doc, src, added, sinceLine); return; }
    emitted.set(g, true); rawAliases.add(n);
    out.push(`${sinceLine ? '// ' + sinceLine + '\n' : ''}// ${g} is kept as raw JSON.${doc ? ' ' + doc : ''} TS: ${txt(t).slice(0, 160)}. (${src})\ntype ${g} = json.RawMessage\n`);
    return;
  }
  const p = propsOf(t);
  if (p) { emitStruct(g, n, p, doc, HOST_ONLY.has(n), src, dNew ? propsOf(dNew.type) : null, sinceLine); return; }
  if (ts.isTypeReferenceNode(t)) {
    const r = goType(t, g, null, null);
    if (r.skip) return;
    emitted.set(g, true);
    out.push(`// ${g} aliases ${r.t}.${doc ? ' ' + doc : ''} (${src})\ntype ${g} = ${r.t}\n`);
    return;
  }
  emitted.set(g, true); rawAliases.add(n);
  out.push(`// ${g} is kept as raw JSON (TS: ${txt(t).slice(0, 160)}).${doc ? ' ' + doc : ''} (${src})\ntype ${g} = json.RawMessage\n`);
}

function emitStruct(g, tsName, p, doc, hostOnly, src, pNew, sinceLine) {
  if (!tsName) {
    const k = p.props.map(txt).join('|') + '##' + (pNew ? pNew.props.map(txt).join('|') : '');
    if (shapeNames.has(k)) return shapeNames.get(k);
    shapeNames.set(k, g);
  }
  if (emitted.has(g)) return g;
  emitted.set(g, true);
  const fields = [], discs = [];
  const addField = (m, nm, sinceMark) => {
    const name = propName(m);
    if (!m.type) return;
    if (tsName === 'Options' && CALLBACK_FLAGS[name]) {
      fields.push({ go: fieldGoName(name), typ: 'bool', tag: name + ',omitempty', comment: CALLBACK_FLAGS[name] });
      return;
    }
    const r = goType(m.type, g, name, nm?.type || null);
    if (r.skip) return;
    const opt = !!m.questionToken;
    let typ = r.t, tag = name;
    const scalar = ['num', 'bool', 'struct', 'union'].includes(r.kind);
    if (r.kind === 'bool' && r.trueLit && opt) { /* optional `true` literal: bool + omitempty */ }
    else if (opt && (scalar || (r.kind === 'string' && r.nullable))) typ = '*' + typ;
    else if (!opt && r.nullable && (scalar || r.kind === 'string')) typ = '*' + typ;
    if (opt) tag += ',omitempty';
    if (!opt && r.literal !== undefined) discs.push([fieldGoName(name), r.literal]);
    const notes = [];
    if (r.enumVals) notes.push('one of ' + r.enumVals.map((v) => JSON.stringify(v)).join(', ') + (r.enumAdded?.length ? '; ' + r.enumAdded.map((v) => JSON.stringify(v)).join(', ') + ' ' + since(r.enumAdded[0], 'enum value', g + '.' + fieldGoName(name)) : ''));
    if (r.literal !== undefined) notes.push(`always ${JSON.stringify(r.literal)}`);
    if (r.trueLit) notes.push('TS literal `true`');
    if (r.note) notes.push(r.note);
    if (r.nullable) notes.push(opt ? 'optional, nullable' : 'nullable');
    if (nm && txt(nm.type) !== txt(m.type) && !['struct', 'slice', 'map', 'union'].includes(r.kind) && !r.enumAdded?.length) notes.push(`0.3.325 declares: ${txt(nm.type).slice(0, 120)}`);
    const comment = [docOf(m), notes.length ? '(' + notes.join('; ') + ')' : ''].filter(Boolean).join(' ');
    fields.push({ go: fieldGoName(name), typ, tag, comment, since: sinceMark });
  };
  const baseNames = new Set(p.props.map(propName));
  for (const m of p.props) addField(m, pNew?.props.find((x) => propName(x) === propName(m)) || null, '');
  for (const m of pNew?.props || []) if (!baseNames.has(propName(m))) addField(m, null, since(propName(m), 'field', g));
  const seenF = new Set();
  for (const f of fields) { while (seenF.has(f.go)) f.go += 'X'; seenF.add(f.go); }
  const L = [];
  if (sinceLine) L.push('// ' + sinceLine);
  L.push(`// ${g}${doc ? ' - ' + doc : ''}${src ? ' (' + src + ')' : ''}`);
  L.push(`type ${g} struct {`);
  for (const f of fields) {
    if (f.since) L.push(`\t// ${f.since}`);
    if (f.comment) L.push('\t// ' + f.comment);
    L.push(`\t${f.go} ${f.typ} \`json:"${f.tag}"\``);
  }
  if (!hostOnly) L.push('\t// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.', '\tRaw json.RawMessage `json:"-"`');
  L.push('}', '');
  if (!hostOnly) {
    L.push(`func (v *${g}) UnmarshalJSON(b []byte) error { type plain ${g}; return unmarshalKeep(b, (*plain)(v), &v.Raw) }`);
    const fill = discs.map(([f, val]) => `if v.${f} == "" { v.${f} = ${JSON.stringify(val)} }; `).join('');
    L.push(`func (v ${g}) MarshalJSON() ([]byte, error) { type plain ${g}; ${fill}return marshalKeep(plain(v), v.Raw) }`, '');
  }
  out.push(L.join('\n'));
  return g;
}

const unionSpecs = [];
function emitUnion(g, tsName, members, d, doc, src, added, sinceLine) {
  if (emitted.has(g)) return;
  emitted.set(g, true);
  const variants = members.map((m, i) => {
    const mm = stripParens(m);
    let vt;
    if (ts.isTypeReferenceNode(mm)) { const n = refName(mm); ensureDecl(n); vt = goName(n); }
    else {
      const p = propsOf(mm);
      const lit = d.keys.map((k) => { const pp = p.props.find((x) => propName(x) === k); return pp ? literalValuesOf(pp.type)?.[0] : null; }).filter(Boolean);
      vt = emitStruct(g + pascal(lit.join('_') || String(i)), null, p, `variant of ${g}`, false, null, null, '');
    }
    const p = d.infos[i].p;
    const conds = {};
    for (const k of d.keys) { const pp = p.props.find((x) => propName(x) === k); conds[k] = pp ? { vals: literalValuesOf(pp.type), optional: !!pp.questionToken } : null; }
    const isAdded = added.includes(m);
    return { vt, conds, since: isAdded ? since(ts.isTypeReferenceNode(mm) ? refName(mm) : vt, 'union variant', g) : '' };
  });
  unionSpecs.push({ g, keys: d.keys, variants, doc, src, sinceLine });
}

const UNION_COND_OVERRIDE = {
  'Message.UserMessageReplay': 'u.Type == "user" && peekBool(u.Raw, "isReplay")',
  'StdoutMessage.UserMessageReplay': 'u.Type == "user" && peekBool(u.Raw, "isReplay")',
  'Message.UserMessage': 'u.Type == "user"', 'StdoutMessage.UserMessage': 'u.Type == "user"',
};
const UNION_PRIORITY = { 'Message.UserMessageReplay': 1, 'StdoutMessage.UserMessageReplay': 1 };

function renderUnions() {
  for (const u of unionSpecs) {
    const L = [];
    const df = u.keys.map(fieldGoName);
    if (u.sinceLine) L.push('// ' + u.sinceLine);
    L.push(`// ${u.g} is a discriminated union${u.doc ? ': ' + u.doc : ''}${u.src ? ' (' + u.src + ')' : ''}.`);
    L.push(`// Discriminator key(s): ${u.keys.map((k) => '"' + k + '"').join(', ')}. Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.`);
    L.push(`// Read with Decode or the As<Variant> accessors; build one for sending with New${u.g}.`);
    L.push(`type ${u.g} struct {`);
    u.keys.forEach((k, i) => L.push(`\t// ${df[i]} is the "${k}" discriminator ("" when absent).`, `\t${df[i]} string \`json:"-"\``));
    L.push('\t// Raw is the complete JSON of the value.', '\tRaw json.RawMessage `json:"-"`', '}', '');
    L.push(`func (u *${u.g}) UnmarshalJSON(b []byte) error { return unmarshalUnion(b, &u.Raw, []string{${u.keys.map((k) => JSON.stringify(k)).join(', ')}}, ${df.map((f) => '&u.' + f).join(', ')}) }`);
    L.push(`func (u ${u.g}) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }`, '');
    L.push(`// New${u.g} wraps a variant value (or any JSON-marshalable value) as a ${u.g}.`);
    L.push(`func New${u.g}(v any) (${u.g}, error) { var u ${u.g}; err := wrapUnion(v, &u); return u, err }`, '');
    const condOf = (v) => u.keys.map((k, i) => {
      const c = v.conds[k];
      if (!c) return `u.${df[i]} == ""`;
      const ors = c.vals.map((x) => `u.${df[i]} == ${JSON.stringify(x)}`);
      if (c.optional) ors.push(`u.${df[i]} == ""`);
      return ors.length > 1 ? '(' + ors.join(' || ') + ')' : ors[0];
    }).join(' && ');
    const seen = new Set();
    for (const v of u.variants) {
      if (seen.has(v.vt)) continue; seen.add(v.vt);
      v.cond = UNION_COND_OVERRIDE[u.g + '.' + v.vt] || condOf(v);
      if (v.since) L.push(`// ${v.since}`);
      L.push(`// As${v.vt} decodes u as ${v.vt}; it returns an ErrVariantMismatch error when the discriminator does not match.`);
      L.push(`func (u ${u.g}) As${v.vt}() (*${v.vt}, error) { return asVariant[${v.vt}](u.Raw, ${v.cond}, "${u.g}", "${v.vt}") }`, '');
    }
    L.push(`// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).`);
    L.push(`func (u ${u.g}) Decode() (any, error) {`, '\tswitch {');
    const done = new Set();
    for (const v of [...u.variants].sort((a, b) => (UNION_PRIORITY[u.g + '.' + b.vt] || 0) - (UNION_PRIORITY[u.g + '.' + a.vt] || 0))) {
      if (done.has(v.vt)) continue; done.add(v.vt);
      L.push(`\tcase ${v.cond}:`, `\t\treturn u.As${v.vt}()`);
    }
    L.push('\t}', `\treturn nil, unknownVariant("${u.g}", u.Raw)`, '}', '');
    out.push(L.join('\n'));
  }
}

// ---------- drive ----------
const roots = [...new Set([...declsExt.keys(), ...declsBase.keys(), ...declsNew.keys()])].sort();
for (const n of roots) ensureDecl(n);
renderUnions();
fs.writeFileSync(logFile, JSON.stringify(sinceLog, null, 1));
process.stdout.write('// Code generated by gen.mjs from @anthropic-ai/claude-agent-sdk sdk.d.ts (base 0.3.311, merged with 0.3.325). DO NOT EDIT.\n\npackage claude\n\nimport "encoding/json"\n\n' + out.join('\n'));
