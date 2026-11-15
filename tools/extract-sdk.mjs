// Static extractor: @anthropic-ai/claude-agent-sdk sdk.d.ts (+ sdk-tools.d.ts) -> protocol surface manifest JSON.
// Usage: node extract-sdk.mjs <package-dir> > manifest.json
// Parses only (no type-check, no execution of package code).
import { createRequire } from 'node:module';
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const require = createRequire(new URL('./ts/', import.meta.url));
const ts = require('typescript');

const pkgDir = process.argv[2];
const pkg = JSON.parse(fs.readFileSync(path.join(pkgDir, 'package.json'), 'utf8'));
const printer = ts.createPrinter({ removeComments: true, newLine: ts.NewLineKind.LineFeed });

function parse(file) {
  const text = fs.readFileSync(file, 'utf8');
  return ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
}
const norm = (node, sf) => printer.printNode(ts.EmitHint.Unspecified, node, sf).replace(/\s+/g, ' ').trim();
const hash = (s) => crypto.createHash('sha256').update(s).digest('hex').slice(0, 12);
const isExported = (n) => !!n.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword);
function jsdocTags(n) {
  const tags = new Set();
  for (const d of ts.getJSDocTags(n)) tags.add(d.tagName.text);
  return [...tags].filter((t) => ['deprecated', 'alpha', 'beta', 'experimental', 'internal', 'hidden'].includes(t));
}

function indexDecls(sf) {
  const decls = new Map();
  sf.forEachChild((n) => {
    if ((ts.isTypeAliasDeclaration(n) || ts.isInterfaceDeclaration(n) || ts.isClassDeclaration(n) ||
         ts.isFunctionDeclaration(n) || ts.isVariableStatement(n) || ts.isEnumDeclaration(n))) {
      const name = ts.isVariableStatement(n) ? n.declarationList.declarations[0].name.getText(sf) : n.name?.getText(sf);
      if (!name) return;
      if (!decls.has(name)) decls.set(name, []);
      decls.get(name).push(n);
    }
  });
  return decls;
}

const sf = parse(path.join(pkgDir, 'sdk.d.ts'));
const decls = indexDecls(sf);

function propsOf(node, seen = new Set()) {
  // returns Map name -> {optional, type, tags}
  const out = new Map();
  const add = (members) => {
    for (const m of members) {
      if ((ts.isPropertySignature(m) || ts.isMethodSignature(m) || ts.isPropertyDeclaration(m) || ts.isMethodDeclaration(m)) && m.name) {
        const name = m.name.getText(sf).replace(/^['"]|['"]$/g, '');
        const type = ts.isMethodSignature(m) || ts.isMethodDeclaration(m)
          ? norm(m, sf).replace(/^\w+\??/, '')
          : (m.type ? norm(m.type, sf) : 'any');
        out.set(name, { optional: !!m.questionToken, type, ...(jsdocTags(m).length ? { tags: jsdocTags(m) } : {}) });
      }
    }
  };
  const visit = (t) => {
    if (!t) return;
    if (ts.isTypeLiteralNode(t)) add(t.members);
    else if (ts.isIntersectionTypeNode(t)) t.types.forEach(visit);
    else if (ts.isParenthesizedTypeNode(t)) visit(t.type);
    else if (ts.isTypeReferenceNode(t)) {
      const n = t.typeName.getText(sf).split('.').pop();
      if (seen.has(n)) return; seen.add(n);
      for (const d of decls.get(n) || []) visitDecl(d);
    }
  };
  const visitDecl = (d) => {
    if (ts.isTypeAliasDeclaration(d)) visit(d.type);
    else if (ts.isInterfaceDeclaration(d) || ts.isClassDeclaration(d)) {
      add(d.members);
      for (const h of d.heritageClauses || []) for (const e of h.types) {
        const n = e.expression.getText(sf);
        if (!seen.has(n)) { seen.add(n); for (const dd of decls.get(n) || []) visitDecl(dd); }
      }
    }
  };
  if (ts.isTypeAliasDeclaration(node) || ts.isInterfaceDeclaration(node) || ts.isClassDeclaration(node)) visitDecl(node); else visit(node);
  return out;
}

function literalValues(t) {
  if (!t) return null;
  if (ts.isLiteralTypeNode(t) && ts.isStringLiteral(t.literal)) return [t.literal.text];
  if (ts.isUnionTypeNode(t)) { const vs = t.types.map(literalValues); return vs.every(Boolean) ? vs.flat() : null; }
  if (ts.isParenthesizedTypeNode(t)) return literalValues(t.type);
  return null;
}

// Flatten a union type (following aliases) into leaf {name, node} members.
function flattenUnion(t, ownerName, out = [], seen = new Set()) {
  if (ts.isUnionTypeNode(t)) { t.types.forEach((x) => flattenUnion(x, ownerName, out, seen)); return out; }
  if (ts.isParenthesizedTypeNode(t)) return flattenUnion(t.type, ownerName, out, seen);
  if (ts.isTypeReferenceNode(t)) {
    const n = t.typeName.getText(sf).split('.').pop();
    const d = (decls.get(n) || [])[0];
    if (d && ts.isTypeAliasDeclaration(d) && ts.isUnionTypeNode(d.type) && !seen.has(n)) {
      seen.add(n);
      // only flatten unions of object types, not literal unions
      if (!literalValues(d.type)) return flattenUnion(d.type, n, out, seen);
    }
    out.push({ name: n, node: t }); return out;
  }
  out.push({ name: ownerName + '<inline>', node: t }); return out;
}

function discriminate(member, keys) {
  const props = propsOf(member.node);
  const vals = keys.map((k) => {
    const p = props.get(k);
    if (!p) return [null];
    const m = p.type.match(/^"([^"]*)"$|^'([^']*)'$/);
    if (m) return [m[1] ?? m[2]];
    const lits = p.type.split('|').map((s) => s.trim().match(/^["']([^"']*)["']$/));
    return lits.every(Boolean) ? lits.map((x) => x[1]) : [`<${p.type}>`];
  });
  // cartesian product
  let combos = [[]];
  for (const v of vals) combos = combos.flatMap((c) => v.map((x) => [...c, x]));
  return combos.map((c) => ({ key: c.filter((x) => x !== null).join(':'), props }));
}

function fieldsObj(props) {
  return Object.fromEntries([...props.entries()].sort(([a], [b]) => a.localeCompare(b)));
}
function shapeEntry(typeName, props) {
  const fields = fieldsObj(props);
  const d = (decls.get(typeName) || [])[0];
  return {
    typeName,
    exported: d ? isExported(d) : undefined,
    ...(d && jsdocTags(d).length ? { tags: jsdocTags(d) } : {}),
    fields,
    shapeHash: hash(JSON.stringify(fields)),
  };
}

function unionSection(rootName, keys) {
  const d = (decls.get(rootName) || [])[0];
  if (!d) return { _missing: true };
  const out = {};
  for (const m of flattenUnion(d.type, rootName)) {
    for (const { key, props } of discriminate(m, keys)) {
      let k = key || `<no-discriminant:${m.name}>`;
      if (out[k]) k += `#${m.name}`;
      out[k] = shapeEntry(m.name, props);
    }
  }
  return Object.fromEntries(Object.entries(out).sort(([a], [b]) => a.localeCompare(b)));
}

// --- sections ---
const manifest = {
  manifestVersion: 1,
  source: { kind: 'npm', package: pkg.name, version: pkg.version, claudeCodeVersion: pkg.claudeCodeVersion },
  stability: 'stable',
  sdkMessages: unionSection('SDKMessage', ['type', 'subtype']),
  // Full CLI->host wire union (stream-json stdout), when the d.ts declares it (newer SDKs)
  stdoutWire: decls.has('StdoutMessage') ? unionSection('StdoutMessage', ['type', 'subtype']) : undefined,
  controlRequests: unionSection('SDKControlRequestInner', ['subtype']),
  controlResponses: {},
  options: {},
  queryMethods: {},
  exports: {},
  stringEnums: {},
  hookEvents: undefined,
  tools: {},
};

// control responses: every SDKControl*Response type, keyed by name
for (const [name, ds] of decls) {
  if (/^SDKControl\w+Response$/.test(name)) manifest.controlResponses[name] = shapeEntry(name, propsOf(ds[0]));
}
manifest.options = fieldsObj(propsOf(decls.get('Options')[0]));
manifest.queryMethods = fieldsObj(propsOf(decls.get('Query')[0]));
// settings.json keys as typed by the SDK (Settings interface); compare with the JSON schema extractor
if (decls.has('Settings')) manifest.settings = fieldsObj(propsOf(decls.get('Settings')[0]));

// all exported symbols: kind + shape hash of normalized declaration text
for (const [name, ds] of [...decls].sort(([a], [b]) => a.localeCompare(b))) {
  const d = ds[0];
  if (!isExported(d)) continue;
  const kind = ts.SyntaxKind[d.kind].replace('Declaration', '').replace('Statement', '');
  const text = ds.map((x) => norm(x, sf)).join('\n');
  manifest.exports[name] = { kind, shapeHash: hash(text), ...(jsdocTags(d).length ? { tags: jsdocTags(d) } : {}) };
  if (ts.isFunctionDeclaration(d)) manifest.exports[name].signature = ds.map((x) => norm(x, sf).replace(/^export declare /, '')).join(' | ');
}
// every string-literal union alias (exported or not): PermissionMode, HookEvent, etc.
for (const [name, ds] of decls) {
  const d = ds[0];
  if (ts.isTypeAliasDeclaration(d)) {
    const v = literalValues(d.type);
    if (v && v.length > 1) manifest.stringEnums[name] = [...v].sort();
  }
  if (ts.isVariableStatement(d)) {
    const decl = d.declarationList.declarations[0];
    if (decl.type && ts.isTypeOperatorNode(decl.type) && ts.isTupleTypeNode(decl.type.type)) {
      const v = decl.type.type.elements.map((e) => (ts.isLiteralTypeNode(e) && ts.isStringLiteral(e.literal)) ? e.literal.text : null);
      if (v.every(Boolean)) manifest.stringEnums[name] = [...v].sort();
    }
  }
}
manifest.hookEvents = manifest.stringEnums.HookEvent;

// sdk-tools.d.ts: tool input/output schemas (generated from JSON schema)
const toolsFile = path.join(pkgDir, 'sdk-tools.d.ts');
if (fs.existsSync(toolsFile)) {
  const tsf = parse(toolsFile);
  tsf.forEachChild((n) => {
    if ((ts.isInterfaceDeclaration(n) || ts.isTypeAliasDeclaration(n)) && isExported(n)) {
      const name = n.name.text;
      const fields = {};
      const members = ts.isInterfaceDeclaration(n) ? n.members : (ts.isTypeLiteralNode(n.type) ? n.type.members : []);
      for (const m of members) if (ts.isPropertySignature(m) && m.name) {
        fields[m.name.getText(tsf).replace(/^['"]|['"]$/g, '')] = { optional: !!m.questionToken, type: m.type ? norm(m.type, tsf) : 'any' };
      }
      manifest.tools[name] = { shapeHash: hash(norm(n, tsf)), fields };
    }
  });
}

// Count blank-line holes inside declarations: markers of @internal members stripped at publish time.
let holes = 0;
for (const m of fs.readFileSync(path.join(pkgDir, 'sdk.d.ts'), 'utf8').matchAll(/;[ \t\r]*\n([ \t\r]*\n){2,}/g)) holes++;
manifest.source.strippedMemberHoles = holes;

process.stdout.write(JSON.stringify(manifest, null, 2) + '\n');
