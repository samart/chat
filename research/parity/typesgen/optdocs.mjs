import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(process.argv[2] + '/');
const ts = require('typescript');
const sf = ts.createSourceFile('x.ts', fs.readFileSync(process.argv[3], 'utf8'), ts.ScriptTarget.Latest, true);
const which = process.argv[4];
let d; sf.forEachChild((n) => { if ((ts.isTypeAliasDeclaration(n) || ts.isInterfaceDeclaration(n)) && n.name.text === which) d = n; });
const members = ts.isInterfaceDeclaration(d) ? d.members : d.type.members;
const res = [];
for (const m of members) {
  if (!m.name) continue;
  const jd = m.jsDoc?.at(-1);
  const c = (typeof jd?.comment === 'string' ? jd.comment : (jd?.comment || []).map((x) => x.text || '').join('')) || '';
  const tags = Object.fromEntries((jd?.tags || []).map((t) => [t.tagName.text, (typeof t.comment === 'string' ? t.comment : (t.comment || []).map((x) => x.text).join('')) || true]));
  const sig = ts.isMethodSignature(m) ? m.getText().replace(/\/\*\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ').trim() : (m.type ? m.type.getText().replace(/\s+/g, ' ') : '');
  res.push({ name: m.name.getText(), optional: !!m.questionToken, type: sig, doc: c.replace(/\s+/g, ' ').trim(), tags });
}
console.log(JSON.stringify(res, null, 1));
