import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(process.argv[2] + '/');
const ts = require('typescript');
const file = process.argv[3];
const sf = ts.createSourceFile(file, fs.readFileSync(file,'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
const out=[];
function walk(n, prefix){
  n.forEachChild(c=>{
    let name;
    if (ts.isModuleDeclaration(c)) { name=c.name.getText(sf); out.push([sf.getLineAndCharacterOfPosition(c.getStart()).line+1, 'namespace', prefix+name, c.end-c.pos]); if (c.body) walk(c.body, prefix+name+'.'); return; }
    if (ts.isVariableStatement(c)) name=c.declarationList.declarations[0].name.getText(sf); else name=c.name?.getText?.(sf);
    const exp = !!c.modifiers?.some(m=>m.kind===ts.SyntaxKind.ExportKeyword);
    out.push([sf.getLineAndCharacterOfPosition(c.getStart()).line+1, ts.SyntaxKind[c.kind].replace('Declaration','')+(exp?'*':''), prefix+(name??'?'), c.end-c.pos]);
  });
}
walk(sf,'');
for (const o of out) console.log(o.join('\t'));
