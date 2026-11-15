import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(process.argv[2] + '/');
const ts = require('typescript');
const sf = ts.createSourceFile('x.ts', fs.readFileSync(process.argv[3], 'utf8'), ts.ScriptTarget.Latest, true);
const res = [];
sf.forEachChild((n) => {
  const exp = n.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword);
  if (!exp) return;
  if (ts.isFunctionDeclaration(n) || ts.isVariableStatement(n) || ts.isClassDeclaration(n)) {
    const jd = n.jsDoc?.at(-1);
    const c = (typeof jd?.comment === 'string' ? jd.comment : (jd?.comment || []).map((x) => x.text || '').join('')) || '';
    const tags = (jd?.tags || []).map((t) => t.tagName.text);
    let sig = n.getText().replace(/\/\*\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ').replace(/^export declare /, '').trim();
    res.push({ kind: ts.SyntaxKind[n.kind], sig: sig.slice(0, 400), doc: c.replace(/\s+/g, ' ').trim(), tags });
  }
});
console.log(JSON.stringify(res, null, 1));
