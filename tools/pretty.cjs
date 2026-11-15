const ts=require(process.argv[4]);const fs=require('fs');
const src=fs.readFileSync(process.argv[2],'utf8');
const sf=ts.createSourceFile('x.js',src,ts.ScriptTarget.Latest,true,ts.ScriptKind.JS);
const K=ts.SyntaxKind;
(function walk(n){ if(n.kind===K.Block||n.kind===K.CaseBlock) n.multiLine=true; ts.forEachChild(n,walk);})(sf);
// mark positions: emit with original start offset comments for statements
const out=ts.createPrinter({removeComments:false}).printFile(sf);
fs.writeFileSync(process.argv[3],out);
