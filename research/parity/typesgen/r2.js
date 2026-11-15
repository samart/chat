qft,join as Zr}from"path";Je();Xs();nn();var u0=1e4;

async function Yr(e,t,n){let r=Math.max(1,Math.floor(n?.maxPages??1e4)),o;for(let s=0;s<r;s++){if(n?.budget!==void 0){if(n.budget.pagesLeft<1)return{status:"capped"};n.budget.pagesLeft--}let i=await e(o);if(!i.ok)return{status:"error",error:i.error};if(await t(i.value.items),n?.until?.())return{status:"done"};if(o=i.value.cursor,!o)return{status:"done"}}return{status:"capped"}}xt();var nke=`

This is how Claude Code surfaces a prompt a plugin submits between `+"turns — it starts this turn in the user's place. Address the message above.";var rke=(e)=>`Prompt from the ${e} plugin`,oke=(e)=>typeof e==="object"&&e!==null&&("text"in e)&&typeof e.text==="string"&&e.text!=="",ske=(e)=>e.length>28&&e.startsWith("The ")&&e.endsWith(` plugin sent a message:
`);
function ike(e,t){let n=e.slice(0,e.indexOf(`
`)+1);return oke(t)&&ske(n)&&e===n+t.text+nke?{label:n.slice(4,-24),text:t.text}:void 0}
function ake(e,t,n){let r=ike(e,t);if(r===void 0)return e;return`${rke(r.label)}${n}${r.text}`}var d0=(e,t)=>ake(e,t,": ");JA();yt();var lke=/^(?:\s*<[a-z][\w-]*[\s>]|\[Request interrupted by user[^\]]*\])/,cke=/<command-name>(.*?)<\/command-name>/;
function os(e,t,n="stored"){if(e.type!=="user")return;if(e.isMeta===!0||e.isCompactSummary===!0)return;let r=e.message;if(!r)return;let o=r.content,s=n==="shown"&&typeof o==="string"?d0(o,e.pluginPrompt):o,i=s!==o,a=[];if(typeof s==="string")a.push(s);else if(Array.isArray(o))for(let l of o){if(!l||typeof l!=="object")continue;if(l.type==="tool_result")return;if(l.type==="text"&&typeof l.text==="string")a.push(l.text)}for(let l of a){let c=ZA(l).replaceAll(`
`," ").trim();if(!c)continue;let d=i?null:cke.exec(c);if(d){if(!t.commandFallback)t.commandFallback=d[1];continue}let p=i?null:/<bash-input>([\s\S]*?)<\/bash-input>/.exec(c);if(p)return`! ${p[1].trim()}`;if(lke.test(c))continue;if(c.length>200)c=ni(c,200).trim()+"…";return c}return}
function V(e){return typeof e==="object"&&e!==null&&!Array.isArray(e)}var p0='"subtype":"voice_call_talk"';
function Vd(e){if(!V(e)||e.type!=="system"||e.subtype!=="voice_call_talk")return e;let{subtype:t,level:n,msgV:r,content:o,blocks:s,fold:i,...a}=e;if(typeof o!=="string"||!V(a.origin)||a.origin.kind!=="voice-call")return e;let l=Array.isArray(s)?s:o;if(i===void 0)return{...a,type:"user",message:{role:"user",content:l}};if(!V(i)||!V(i.command))return e;let{isMeta:c,origin:d,...p}=a,{command:f,...m}=i;return{...m,...p,type:"attachment",attachment:{...f,type:"queued_command",prompt:l,origin:d}}}var Un=65536,Rke=new Set(["sdk-cli","sdk-ts","sdk-py"]);
function T0(e,t){let n=ql(e,"entrypoint")??fR(t,"entrypoint");if(n&&Rke.has(n))return!0;let r=e.split(`
`).find((s)=>s.includes('"parentUuid":'))??e,o=ql(r,"sessionKind");return o==="daemon"||o==="daemon-worker"}var Oke=/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
function Xe(e){if(typeof e!=="string")return null;return Oke.test(e)?e:null}
function v0(e){if(!e.includes("\\"))return e;try{return JSON.parse(`"${e}"`)}catch{return e}}
function ql(e,t){let n=[`"${t}":"`,`"${t}": "`];for(let r of n){let o=e.indexOf(r);if(o<0)continue;let s=o+r.length,i=s;while(i<e.length){if(e[i]==="\\"){i+=2;continue}if(e[i]==='"')return v0(e.slice(s,i));i++}}return}
function fR(e,t){let n=[`"${t}":"`,`"${t}": "`],r,o=-1;for(let s of n){let i=0;while(!0){let a=e.indexOf(s,i);if(a<0)break;let l=a+s.length,c=l;while(c<e.length){if(e[c]==="\\"){c+=2;continue}if(e[c]==='"'){if(a>o)r=v0(e.slice(l,c)),o=a;break}c++}i=c+1}}return r}
function Zg(e,t,n){return mR(e,n,t)}var wke={"custom-title":"customTitle","ai-title":"aiTitle",tag:"tag"};
function Xn(e,t,n="older"){return mR(e,wke[t],t,n)}
function h0(e,t,n){try{let r=JSON.parse(e);if(typeof r==="object"&&r!==null&&(n===void 0||r.type===n)){let o=r[t];if(typeof o==="string")return o}return}catch{return null}}
function mR(e,t,n,r){let o=n===void 0?void 0:r===void 0?[`"type":"${n}"`]:[`"type":"${n}"`,`"type": "${n}"`],s=`"${t}":`,i=e.length;while(i>0){let a=e.lastIndexOf(`
`,i-1),l=e.slice(a+1,i);if(i=a,(l.includes(s)||r==="none")&&(o===void 0||o.some((c)=>l.includes(c)))){let c=h0(l,t,n);if(c===null&&r!==void 0){let d=l.lastIndexOf(`{${o?.[0]}`);if(c=d>0&&!":,[".includes(l.charAt(d-1))?h0(l.slice(d),t,n):null,c===null&&r==="none")return}if(typeof c==="string")return c}if(a<0)break}return}
function gR(e,t){let n=`"${t}":`,r=0;while(r<e.length){let o=e.indexOf(`
`,r),s=o<0?e.slice(r):e.slice(r,o);if(r=o<0?e.length:o+1,s.includes(n))try{let i=JSON.parse(s);if(typeof i==="object"&&i!==null){let a=i[t];if(typeof a==="string")return a}}catch{}}return}
function hR(e,t){return mR(e,t)}
function Jg(e,t){return e?.realWorkspacePath===void 0?void 0:{hoverRestOn:t,realPath:e.realWorkspacePath}}

async function Xd(e,t,n){return vke(e,t,"w",n)}var Tke=262144;

async function vke(e,t,n,r){if(r!==void 0&&r.hoverRestOn&&(n==="w"||t.length>0))return xke(r.source,t,n);let o=Ske(e,{mode:384,flags:n});try{let s="";for(let i of t)if(s+=JSON.stringify(i)+`
`,s.length>=Tke){let a=o.write(s);if(s="",!a)await g0(o,"drain")}if(s!=="")o.write(s);o.end(),await g0(o,"finish")}catch(s){throw o.destroy(),s}}

async function xke(e,t,n){let{backend:r,key:o}=e,s=t.map((a)=>({data:JSON.stringify(a)+`
`}));if(n==="w"){let a=await r.replaceRecords(o,s,{publishDiscipline:"inPlace",mode:384});if(!a.ok)throw Error("transcript stream replace failed",{cause:a.error});return}let i=await r.append(o,s);if(!i.ok)throw Error("transcript stream append failed",{cause:i.error})}
function Xg(e,t="stored"){let n=0,r={commandFallback:""};while(n<e.length){let o=e.indexOf(`
`,n),s=o>=0?e.slice(n,o):e.slice(n);if(n=o>=0?o+1:e.length,!s.includes('"type":"user"')&&!s.includes('"type": "user"'))continue;if(s.includes('"tool_result"'))continue;if(s.includes('"isMeta":true')||s.includes('"isMeta": true'))continue;if(s.includes('"isCompactSummary":true')||s.includes('"isCompactSummary": true'))continue;try{let i=JSON.parse(s),a=os(i,r,t);if(a!==void 0)return a}catch{continue}}return r.commandFallback}
function x0(e){let t=0;while(t<e.length){let n=e.indexOf(`
`,t),r=n>=0?e.slice(t,n):e.slice(t);if(t=n>=0?n+1:e.length,!r.includes('"type":"user"')&&!r.includes('"type": "user"'))continue;if(r.includes('"tool_result"'))continue;if(r.includes('"isMeta":true')||r.includes('"isMeta": true'))continue;if(r.includes('"type":"image"')||r.includes('"type": "image"'))return"Image";if(r.includes('"type":"document"')||r.includes('"type": "document"'))return"Document"}return""}
function k0(e,t="stored"){let n={commandFallback:""};for(let r of e){if(typeof r!=="object"||r===null)continue;let o=os(r,n,t);if(o!==void 0)return o}return n.commandFallback}var _0=process.platform==="win32"?Yd.O_RDONLY:Yd.O_RDONLY|Yd.O_NOFOLLOW|Yd.O_NONBLOCK;

async function kke(e){let{backend:t,key:n}=e;try{let r=await t.read([{key:n,offset:0,length:Un},{key:n,tail:Un}]);if(!r.ok)return null;let[o,s]=r.value.items;if(!o.found||o.value.length===0)return null;let i=y0(o.value),a=s.found&&s.value.length>0?y0(s.value):i;return{head:i,tail:a,mtimeMs:o.mtimeMs,totalBytes:o.totalBytes}}catch{return null}}
function y0(e){return Buffer.from(e.buffer,e.byteOffset,e.byteLength).toString("utf8")}var rmt=Buffer.from('"type":"user"'),omt=Buffer.from('"type":"assistant"'),smt=Buffer.from(p0);
function Qg(e,t){if(e.kind!=="scope"||e.scope.namespace!=="transcript"||e.scope.projectKey===void 0||e.scope.sessionId!==void 0||!t(e.scope.projectKey))return;return e.scope.projectKey}
function P0(e){return{skipScopeStats:!0,skipKeyStats:!0,...e!==void 0&&{cursor:e}}}
function _R(e){let t=new Map,n=new Proxy(e.backend,{get(r,o,s){let i=Reflect.get(r,o,s);if(o!=="listEntries")return typeof i==="function"?i.bind(r):i;return(l,c)=>{if(!(l.namespace==="transcript"&&!("projectKey"in l)&&Object.keys(c??{}).every((m)=>m==="cursor"||m==="skipScopeStats"||m==="skipKeyStats")))return r.listEntries(l,c);let p=[c?.cursor??"",c?.skipScopeStats,c?.skipKeyStats].map(String).join("\x00"),f=t.get(p);if(f===void 0){f=r.listEntries(l,c),t.set(p,f);let m=()=>{t.delete(p)};f.then((g)=>{if(!g.ok)m()},m)}return f}}});return{...e,backend:n}}

async function xa(e,t){if(t!==void 0&&t.hoverRestOn)return D0(t.source);try{if(_0===Yd.O_RDONLY){if(!(await Eke(e)).isFile())return null}let n=await O0(e,_0);try{let r=await n.stat();if(!r.isFile())return null;let o=Buffer.allocUnsafe(Un),s=await n.read(o,0,Un,0);if(s.bytesRead===0)return null;let i=o.toString("utf8",0,s.bytesRead),a=Math.max(0,r.size-Un),l=i;if(a>0){let c=await n.read(o,0,Un,a);l=o.toString("utf8",0,c.bytesRead)}return{mtime:r.mtime.getTime(),size:r.size,head:i,tail:l}}finally{await n.close()}}catch{return null}}

async function D0(e){let t=await kke(e);if(t===null)return null;return{mtime:Math.trunc(t.mtimeMs),size:t.totalBytes,head:t.head,tail:t.tail}}var is=200;
function Pke(e){return Math.abs(YA(e)).toString(36)}
function Yl(e){return e.replace(/[^a-zA-Z0-9]/g,"-")}
function Qd(e){let t=Yl(e);if(t.length<=is)return t;return`${t.slice(0,is)}-${Pke(e)}`}
function Vg(e,t){let n=e.replaceAll("\\","/");return t?n.toLowerCase():n}
function S0(e,t,n){let r=rt(e),o=rt(t),s=(i)=>n?i.toLowerCase():i;if(s(Yl(r))!==s(Yl(o)))return!1;return Vg(r,n)!==Vg(o,n)}

async function I0(e,t,n,r,o){if(!S0(e,t,n))return!1;if(Gr(e)||Gr(t))return!1;if(await r(e)||await r(t))return!1;let s;try{if(o!==void 0&&o.hoverRestOn){let a=await o.realPath(e);if(!a.ok||!a.value.found)return!1;s=rt(a.value.path)}else s=rt(await w0(e))}catch{return!1}let i=await as(t,o);return S0(s,i,n)}
function L0(e,t,n){if(t===void 0)return!1;let r=Vg(rt(e),n);return t.some((o)=>{let s=Vg(rt(o),n);return r===s||r.startsWith(s.endsWith("/")?s:s+"/")})}
function yR(){return process.platform==="win32"||process.platform==="darwin"}
function Pt(){return Zr(Vt(),"projects")}
function yo(e){return yM()??Qd(e)}
function SR(e){let t=Qd(e);return t===yo(e)?void 0:t}
function Zl(e,t=Pt()){return Zr(t,yo(e))}
function ka(e,t){let n=Cke(e);return Ake(e)===Pt()&&t(n)?n:void 0}

async function as(e,t){try{if(t!==void 0&&t.hoverRestOn){let n=await t.realPath(e);return rt(n.ok&&n.value.found?n.value.path:e)}return rt(await w0(e))}catch{return rt(e)}}
function ER(e,t){if(process.platform!=="win32")return[t];let n=/^([A-Za-z]):[\\/]+(.*)$/s.exec(e);if(n===null)return[t];let[,r="",o=""]=n,s=o.replace(/[\\/]+$/,""),i=s===""?[]:s.split(/[\\/]+/),a=t.replace(/[\\/]+$/,"").split(/[\\/]+/),l=a.length-i.length,c=(d,p)=>d!==void 0&&p!==void 0&&d.length===p.length&&d.toLowerCase()===p.toLowerCase();if(l<1||!i.every((d,p)=>c(d,a[l+p]))||l===1&&c(a[0],`${r}:`))return[t];return[t,`${r}:\\${i.join("\\")}`]}

async function Jl(e,t,n,r){let o=r!==void 0&&r.hoverRestOn?r.source:void 0,s=o===void 0?void 0:ka(e,o.isKeySegment);if(o!==void 0&&s!==void 0){let l=await Dke(s,t,n,o);if(l!==void 0)return l}if(Ze())return!1;let i=Yl(t),a;try{a=await Zd(e,{withFileTypes:!0})}catch{return!1}for(let l of a){if(!l.isFile()||!l.name.endsWith(".jsonl"))continue;let c=await xa(Zr(e,l.name));if(c===null)continue;if(M0(c,i,n))return!0}return!1}
function M0(e,t,n){let r=Zg(e.tail,"relocated","relocatedCwd")??gR(e.head,"cwd");if(r===void 0)return!1;let o=Yl(rt(r));return n?o.toLowerCase()===t.toLowerCase():o===t}

async function Dke(e,t,n,r){let{backend:o,transcriptKey:s,isKeySegment:i}=r,a=Yl(t),l=!1,c=new Set;try{let d=await Yr((p)=>o.listEntries({namespace:"transcript",projectKey:e},P0(p)),async(p)=>{for(let f of p){if(f.kind!=="key"||f.key.namespace!=="transcript"||f.key.projectKey!==e||f.key.agentId!==void 0||!i(f.key.sessionId)||c.has(f.key.sessionId))continue;let m=f.key.sessionId,g=await D0({backend:o,key:s(e,m)});if(g===null)continue;if(c.add(m),M0(g,a,n)){l=!0;return}}},{until:()=>l});if(l)return!0;if(d.status!=="done")return}catch{return}return!1}

async function pr(e,t,n=Pt()){if(t!==void 0&&t.hoverRestOn)return Ike(e,t);if(Ze())return[];let r=Zl(e,n),o=[];try{await Zd(r),o.push(r)}catch{}let s=SR(e);if(s!==void 0){let p=Zr(n,s);try{await Zd(p),o.push(p)}catch{}return o}let i=Qd(e);if(i.length<=is)return o;let a=process.platform==="win32",l=(p)=>a?p.toLowerCase():p,c=l(i.slice(0,is)+"-"),d=l(r);try{for(let p of await Zd(n,{withFileTypes:!0})){if(!p.isDirectory()||!l(p.name).startsWith(c))continue;let f=Zr(n,p.name);if(l(f)!==d&&await Jl(f,e,a))o.push(f)}}catch{}return o}

async function Ike(e,t){let n=await Xl(t.source);if(n===null){let r=SR(e);return[Zl(e),...r!==void 0?[Zr(Pt(),r)]:[]]}return bR(e,n,process.platform==="win32",t)}

async function Xl(e){let{backend:t,isKeySegment:n}=e,r=[],o=new Set,s=0;try{if((await Yr((a)=>t.listEntries({namespace:"transcript"},P0(a)),(a)=>{for(let l of a){let c=Qg(l,n);if(c!==void 0&&!o.has(c))o.add(c),r.push(c)}s++})).status==="error")return s===0?null:r}catch{return s===0?null:r}return r}

async function bR(e,t,n,r){let o=yo(e),s=Zl(e),i=SR(e),a=i===void 0&&o.length>is?o.slice(0,is)+"-":void 0,l=Pt(),c=(_)=>n?_.toLowerCase():_,d=a!==void 0?c(a):void 0,p=c(o),f=!1,m,g=[],S=new Set;for(let _ of t)if(_===o)f=!0;else if(i!==void 0){if(m===void 0&&c(_)===c(i))m=Zr(l,_)}else if(d!==void 0&&c(_).startsWith(d)&&c(_)!==p&&!S.has(c(_)))S.add(c(_)),g.push(_);let y=[];for(let _ of g){let E=Zr(l,_);if(await Jl(E,e,n,r))y.push(E)}return[...f?[s]:[],...m!==void 0?[m]:[],...y]}

async function Lke(e,t,n=Pt()){let r=[];await Promise.resolve().then(() => qd());for(let s of await ss(e)){if(s===e)continue;let i=await pr(s,t,n);if(t!==void 0&&t.hoverRestOn){let a=Zl(s);if(!i.includes(a))i.push(a)}for(let a of i)r.push({worktreePath:s,projectDir:a})}return r}

async function ai(...e){for await(let t of CR(...e))return t;return}async function*CR(e,t,n,r,o=Pt()){let s=`${e}.jsonl`,i=r&&n!==void 0&&n.isKeySegment(e)?_R(n):void 0,a=i===void 0?void 0:{source:i,hoverRestOn:r};

async function l(d,p){let f=Zr(d,s);if(i){let m=ka(d,i.isKeySegment);if(m!==void 0){try{let g=await i.backend.stat(i.transcriptKey(m,e));if(g.ok){let S=g.value.storedBytes??g.value.size;if(S>0)return{filePath:f,projectPath:p,fileSize:S}}}catch{}return}}try{let m=await bke(f);if(m.size>0)return{filePath:f,projectPath:p,fileSize:m.size}}catch{}return}if(t){let d=await as(t,Jg(n,r)),p=new Set;for(let f of ER(t,d))for(let m of await pr(f,a,o)){let g=process.platform==="win32"?m.toLowerCase():m;if(p.has(g))continue;p.add(g);let S=await l(m,f);if(S)yield S}for(let{worktreePath:f,projectDir:m}of await Lke(d,a,o)){let g=await l(m,f);if(g)yield g}return}if(i){let d=new Map;try{if((await Yr((m)=>i.backend.listEntries({namespace:"transcript"},m!==void 0?{cursor:m}:void 0),(m)=>{for(let g of m){let S=Qg(g,i.isKeySegment);if(S===void 0)continue;let y=g.kind==="scope"&&g.mtimeMs!==void 0?g.mtimeMs:Number.NEGATIVE_INFINITY,_=d.get(S);if(_===void 0)d.set(S,{projectKey:S,mtimeMs:y});else if(y>_.mtimeMs)_.mtimeMs=y}})).status==="error")return}catch{return}let p=[...d.values()];p.sort((f,m)=>f.mtimeMs!==m.mtimeMs?m.mtimeMs-f.mtimeMs:f.projectKey<m.projectKey?-1:f.projectKey>m.projectKey?1:0);for(let{projectKey:f}of p){let m=await l(Zr(o,f),void 0);if(m)yield m}return}let c;try{c=await Zd(o)}catch{return}for(let d of c){let p=await l(Zr(o,d),void 0);if(p)yield p}}var N0=1048576,U0=5242880,Mke=Buffer.from('"compact_boundary"');
function F0(e){try{let t=JSON.parse(e);if(t.type!=="system"||t.subtype!=="compact_boundary")return null;return{hasPreservedSegment:Boolean(t.compactMetadata?.preservedSegment||t.compactMetadata?.preservedMessages)}}catch{return null}}
function va(e,t,n,r){let o=r-n;if(o<=0)return;if(e.len+o>e.buf.length){let s=Buffer.allocUnsafe(Math.min(Math.max(e.buf.length*2,e.len+o),e.cap));e.buf.copy(s,0,0,e.len),e.buf=s}t.copy(e.buf,e.len,n,r),e.len+=o}
function qg(e,t,n,r){return r