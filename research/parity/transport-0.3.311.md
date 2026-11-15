# Claude Agent SDK (TypeScript) 0.3.311 vs the 0.3.325 transport reference

Rule-by-rule check of `transport.md` (written for 0.3.325) against `@anthropic-ai/claude-agent-sdk` **0.3.311** (`claudeCodeVersion` 2.1.311).

- Source read: `reference/sdk-pkgs/v0.3.311/package/sdk.mjs` (self-contained, no chunk imports). Read as data; nothing was executed except a local Node `readline` test (§4, E1). The CLI was never run.
- `@N` = **byte offset into the minified 0.3.311 `sdk.mjs`**. `@325:N` = offset quoted from `transport.md`.
- Pretty-printed copy used for reading: `research/parity/sdk-0.3.311.pretty.js` (made with `tools/pretty.cjs` and the local TypeScript in `tools/ts`).
- 0.3.311 names: `ProcessTransport` = `Cv` (@1219503), `Query` = `Ny` (@1236217), option normalisation = `Fv` (@1294160), prompt dispatch = `$v` (@1301358), `query` = `D6e` (@1301917), startup timer = `Iv` (@1284007). Helpers: `We(f,v)` = `` `${f}=${v}` `` (@1216796), `Py` = two-element-unless-leading-dash (@1216831), `s6` = extraArgs push (@1216925), `se` = `JSON.stringify`, `ze` = `JSON.parse`, `Pa` = `crypto.randomUUID`, `Oe` = truthy env parse (@188659).
- `core.mjs` in 0.3.311 carries the same argv builder (compared after identifier normalisation; identical).

## How the comparison was done

1. Both bundles were pretty-printed. The region from the `--flag=value` helper to the end of the file (transport, Query, option normalisation, `query`/`startup`/`prewarm`) was identifier-normalised and diffed: 79 hunks, all read.
2. String literals and numeric constants in that region were compared as multisets. The only differences are the ones listed in this document.
3. Helpers outside that region that affect the wire were compared one by one: truthy env parse, debug-file path, stderr redaction tables, tail truncation, sandbox merge, SDK-MCP settings normaliser, `createSdkMcpServer`, JSON helpers.
4. Every 0.3.311 handler quoted below was read directly, not inferred from the diff.

Not verified: anything on the CLI side (what CLI 2.1.311 accepts or emits). Bun's `readline` was not tested.

## Result

| class | count | checklist items |
|---|---|---|
| SAME | 31 | 1–8, 10–14, 16–19, 21, 22, 24–28, 30–35, 37 |
| DIFFERENT | 4 | 15, 20, 29, 36 |
| ABSENT in 0.3.311 | 2 | 9, 23 |

Timings (2000 / 5000 / 200 / 250 / 600000 ms), the argv order, environment handling, the string-prompt shape, the stdin-close logic, the swallowed frame set, `can_use_tool` mapping, cancel semantics and the elicitation default are all unchanged. Everything that differs traces back to three 0.3.325 features that 0.3.311 does not have: hook `onFailure` / prompt hold, managed settings via initialize, and typed `error_code` on error replies.

---

## (a) Checklist items

| # | item (short) | class | 0.3.311 evidence |
|---|---|---|---|
| 1 | argv prefix | SAME | `let W=["--output-format","stream-json","--verbose","--input-format","stream-json"]` @1222564 |
| 2 | `--flag=value` single element; `--mcp-config`, `--managed-settings` two elements | SAME | `function We(e,t){return`${e}=${t}`}` @1216796; `W.push(...Py("--mcp-config",se({mcpServers:w})))` @1224230; `Py("--managed-settings",…)` @1225570 |
| 3 | flag order follows §1.2 | SAME (see note) | Push order identical for rows 0–35 and 37. Row 36 has only its `--managed-settings <JSON>` branch in 0.3.311 (X1). |
| 4 | `settingSources: []` → `--setting-sources=` | SAME | `if(D!==void 0)W.push(We("--setting-sources",D.join(",")))` @1224290 |
| 5 | `tools: []` → `--tools=`; preset → `--tools=default` | SAME | `if(ce!==void 0)if(Array.isArray(ce))W.push(We("--tools",ce.join(",")));else W.push("--tools=default")` @1224171 |
| 6 | `canUseTool` → `--permission-prompt-tool=stdio`; both set throws | SAME | `if(v){if(b)throw Error("canUseTool callback cannot be used with permissionPromptToolName. Please use one or the other.");W.push("--permission-prompt-tool=stdio")}else if(b)W.push(We("--permission-prompt-tool",b))` @1223671 |
| 7 | SDK MCP servers never in `--mcp-config` | SAME | `if(In.type==="sdk"&&In.instance)Wv.set(Ft,In);else Gv[Ft]=In` @1298335; `if(w&&Object.keys(w).length>0)` |
| 8 | extraArgs last; `null` → `--key`; `settings` key last | SAME | `let wt={...a??{}};if(this.options.settings)wt.settings=this.options.settings;let dt=o6(wt,de);for(let[Se,Pn]of Object.entries(dt))if(Pn===null)W.push(`--${Se}`);else s6(W,Se,Pn)` @1225625 |
| 9 | implicit `--await-initialize` on resume/continue | **ABSENT** | `extraArgs:Et?{...de,workload:Et}:de` @1299141. See D9. |
| 10 | thinking / maxThinkingTokens mapping | SAME | `else if(Hr!==void 0)ru=Hr===0?{type:"disabled"}:{type:"enabled",budgetTokens:Hr}` @1298684; `if(d.budgetTokens===void 0)W.push("--thinking=adaptive")` |
| 11 | `maxBudgetUsd: 0` emitted, `maxTurns: 0` not | SAME | `if(p)W.push(We("--max-turns",p));if(f!==void 0)W.push(We("--max-budget-usd",f))` |
| 12 | skills → `Skill(x)` in allowedTools and `skills` in initialize | SAME | `let Se=U==="all"?["Skill"]:U.map(He=>`Skill(${P9e(He)})`),Pn=new Set(K);K=[...K,...Se.filter(He=>!Pn.has(He))]` @1222427; `skills:Array.isArray(this.initConfig?.skills)?this.initConfig.skills:void 0` |
| 13 | json_schema in argv and initialize | SAME | `let Kv=cf?.type==="json_schema"?cf.schema:void 0` @1296957; `if(S)W.push(We("--json-schema",se(S)))`; `jsonSchema:this.jsonSchema` |
| 14 | native vs JS detection | SAME | `function I9e(e){return![".js",".mjs",".tsx",".ts",".jsx"].some(n=>e.endsWith(n))}` @1232380; `pn=rt?[...i,...W]:[...i,l,...W]` @1225952 |
| 15 | child env additions | **DIFFERENT** (version literal only) | `pt.CLAUDE_AGENT_SDK_VERSION="0.3.311"` @1297100. See D15. |
| 16 | `NODE_OPTIONS` removed; `DEBUG` only as `1` | SAME | `if(delete c.NODE_OPTIONS,Oe(c.DEBUG_CLAUDE_AGENT_SDK))c.DEBUG="1";else delete c.DEBUG` @1225869 |
| 17 | `options.env` replaces inherited env; three keys scrubbed | SAME | `pt=L?{...L}:{...process.env}`; `if(Ft)pt[In]="1";else if(!L){for(let Br of Object.keys(pt))if(Br.toUpperCase()===In)delete pt[Br]}` @1297972 |
| 18 | initialize is the first frame; `request_id` before `type` | SAME | `let o={request_id:r,type:"control_request",request:e}` @1262383; constructor calls `this.initialize({first:!0})` @1238390 |
| 19 | `systemPrompt` default `[""]`; preset omits it | SAME | `if(i===void 0)f=""` @1294346; `else if(i.type==="preset")m=i.append,g=i.excludeDynamicSections,_=i.snapshot`; `systemPrompt:typeof …==="string"?[…]:…` |
| 20 | hook IDs and matcher shape | **DIFFERENT** | `return{matcher:a.matcher,hookCallbackIds:l,timeout:a.timeout}` @1251863. No `onFailure`. See D20. |
| 21 | manifest capture ≤250 ms; `sdkMcpServers` names | SAME | `var C6=250` @1235965; `a=await Uf(Promise.race([s,i]),C6)`; `t=this.sdkMcpServers.size>0?Array.from(this.sdkMcpServers.keys()):void 0` |
| 22 | string prompt frame | SAME | `e.writeAfterInitialize(se({type:"user",session_id:"",message:{role:"user",content:[{type:"text",text:t}]},parent_tool_use_id:null,...r&&{client_composed:!0}})+"\n")` @1301402 |
| 23 | prompt hold until initialize response, 660 s guard | **ABSENT** | No `promptHold`, no guard timer. See D23. |
| 24 | string mode stdin close | SAME | `if(this.resultReceived=!0,this.sessionState===void 0||this.sessionState==="idle"||!this.hasBidirectionalNeeds())this.endRun();else this.armRunEndCeiling()` @1241681; `endRun(){… if(…,this.isSingleUserTurn)…this.transport.endInput()}` @1269834 |
| 25 | streaming mode | SAME | `await Promise.resolve(this.transport.write(se(this.verbatimPrompts?{...n,client_composed:!0}:n)+"\n"))`; `t>0&&this.hasBidirectionalNeeds()` → `await this.waitForRunEnd()`; then `this.transport.endInput()` @1269024 |
| 26 | swallowed frames | SAME | `control_response` / `control_request` / `control_cancel_request` / `keep_alive` / `transcript_mirror` each end in `continue` @1240479–@1241180; `if("sdk_host_only"in e&&e.sdk_host_only===!0)continue` @1242583 |
| 27 | blank and non-JSON lines skipped; no length limit | SAME | `for await(let n of e)if(n.trim()){let r;try{r=ze(n)}catch(o){nn(`Non-JSON stdout: ${n}`);continue}yield r}` @1231346 |
| 28 | request IDs; no timeout | SAME | `let r=Math.random().toString(36).substring(2,15)` @1262548; `request_id:Pa()` for `mcp_message` @1271786 |
| 29 | reply envelopes; unknown subtype | **DIFFERENT** | `throw Error("Unsupported control request subtype: "+e.request.subtype)` @1250254. No `error_code`. See D29. |
| 30 | `can_use_tool` reply | SAME | `if(r===null)return qc;return{...r,toolUseID:e.request.tool_use_id}` @1247013; `throw Error("canUseTool callback is not provided.")` |
| 31 | `mcp_message` replies | SAME | `if(h6(n.message))return{mcp_response:await this.handleMcpControlRequest(…)};else{if(r.onmessage)r.onmessage(n.message);return{mcp_response:{jsonrpc:"2.0",result:{},id:0}}}` @1247211 |
| 32 | elicitation default; unanswered subtypes | SAME | `return{action:"decline"}` @1247948; `var S6=["remote_tool_call","remote_plumbing_call","remote_tools_probe","remote_tools_reannounce"]` @1235747; no `onUserDialog` → `return …,qc` |
| 33 | cancel aborts handler; late result still sent | SAME | `handleControlCancelRequest(e){let t=this.cancelControllers.get(e.request_id);if(t)t.abort(),this.cancelControllers.delete(e.request_id)}` @1244891; the write in `handleControlRequest` is not conditional on the abort |
| 34 | `pending_*` in initialize response | SAME | `if(!s&&(g.pending_permission_requests||g.pending_user_dialog_requests))…else{if(g.pending_permission_requests)this.processPendingPermissionRequests(…);if(g.pending_user_dialog_requests)this.processPendingUserDialogRequests(…)}`; `pending_elicitation_requests` is never referenced |
| 35 | close: EOF → 2000 ms → SIGTERM → 5000 ms → SIGKILL | SAME | `var x9e=2000` @1216970; `a.kill("SIGTERM"),setTimeout(c=>{if(c.exitCode===null)c.kill("SIGKILL")},5000,a).unref()` @1230671 |
| 36 | exit error text and error-result substitution | **DIFFERENT** (minor) | Text identical. The rule for when the stored error text is cleared differs. See D36. |
| 37 | ≤200 ms stderr drain after exit | SAME | `v9e=200`; `p=setTimeout(m,v9e)` @1221518 |

Note on item 3: I counted it SAME because the relative order of every flag 0.3.311 can emit is unchanged. If you read the item literally ("follows table §1.2 exactly"), row 36 makes it DIFFERENT; the detail is in X1.

---

## (b) Detail for DIFFERENT and ABSENT items

### D9. Implicit `--await-initialize` on resume/continue: ABSENT

0.3.325 (`ZD` @325:1407788, @325:1408188):

```js
let CQ = Boolean(resume || continue) && !options.pathToClaudeCodeExecutable && !options.spawnClaudeCodeProcess
      && extraArgs?.["sdk-url"] === void 0 && !Object.keys(extraArgs ?? {}).some(k => k.startsWith("sdk-url="))
      && !executableArgs.some(a => /^--sdk-url(=|$)/.test(a));
extraArgs: { ...userExtraArgs, ...(workload && {workload}), ...(CQ && {"await-initialize": null}) }
```

0.3.311 (`Fv` @1299141):

```js
extraArgs: Et /*workload*/ ? { ...de, workload: Et } : de
```

0.3.311 behaviour: resume and continue add nothing to extraArgs. `--await-initialize` appears in argv only when `pluginDelivery === "initialize"` with at least one plugin (@1224891), or when the caller puts it in `extraArgs` (`prewarm()` adds `"await-claim": null`, not `await-initialize`). It can never be duplicated.

### D15. Version literal in the child env: DIFFERENT (value only)

```js
// 0.3.325
process.env.CLAUDE_AGENT_SDK_VERSION = "0.3.325"; … if (!env.CLAUDE_AGENT_SDK_VERSION) env.CLAUDE_AGENT_SDK_VERSION = "0.3.325";
// 0.3.311  @1294553, @1297100
process.env.CLAUDE_AGENT_SDK_VERSION = "0.3.311"; … if (!pt.CLAUDE_AGENT_SDK_VERSION) pt.CLAUDE_AGENT_SDK_VERSION = "0.3.311";
```

Logic is identical. See E2 for how this rule is misdescribed in the reference.

### D20. Hook matcher shape: DIFFERENT

```js
// 0.3.325 @325:1361824
return { matcher: m.matcher, hookCallbackIds: ids, timeout: m.timeout, onFailure: m.onFailure };
// 0.3.311 @1251863
return { matcher: a.matcher, hookCallbackIds: l, timeout: a.timeout };
```

0.3.311 behaviour: each matcher serialises as `{"matcher":…,"hookCallbackIds":[…],"timeout":…}` with undefined keys dropped. `onFailure` is never sent, even if the caller sets it; `HookCallbackMatcher` in 0.3.311's `sdk.d.ts` has no such field. ID assignment (`hook_0`, `hook_1`, … across events, matchers, hooks) is unchanged.

One guard also differs: 0.3.311 tests `if (i.length > 0)` where 0.3.325 tests `i?.length > 0`. A hooks object with an `undefined` event value throws a `TypeError` in 0.3.311 and is skipped in 0.3.325.

### D23. Prompt hold and 660 s guard: ABSENT

0.3.325 constructor (@325:1338880 and the constructor body):

```js
if (this.hasHookSetToBlock() || this.initConfig?.managedSettings !== void 0) {
  let g = setTimeout(() => this.cleanup(Error('Claude Code did not answer initialize, so a hook set to onFailure: "block" …')), r7 /*660000*/);
  this.promptHold = this.initialization.then(() => (clearTimeout(g), true), e => (clearTimeout(g), this.cleanup(e), false));
  …
}
// request(): held when HELD.has(subtype) /*interrupt, cancel_async_message*/ || managedSettings && !EXEMPT.has(subtype) /*initialize, get_binary_version*/
// JD: e.writePrompt(...)   → waits on promptHold
```

0.3.311 constructor (@1238390) and writers:

```js
this.sdkMessages = this.readSdkMessages(), this.readMessages(),
this.initializeWritten = new Promise(g => { this.markInitializeWritten = () => { this.initializeHasBeenWritten = !0, g() } }),
this.initialization = this.initialize({first: !0}), this.initialization.catch(() => {});

writeAfterInitialize(e) {                       // @1264108
  if (this.initializeHasBeenWritten) return Promise.resolve(this.transport.write(e));
  return this.initializeWritten.then(() => this.transport.write(e));
}
// request(): (this.initializeHasBeenWritten || this.writingFirstInitialize ? Promise.resolve(m()) : this.initializeWritten.then(m)).catch(p)
```

0.3.311 behaviour:

- There is no `promptHold`, `hasHookSetToBlock`, `writePrompt` or `writeBehindPromptHold`.
- The only gate on any write is "the initialize frame has been written". Nothing waits for the initialize **response**.
- `query()` has no timer of any kind on the initialize path.
- The initialize response is checked for one thing only: `plugins_applied !== true` when plugins were sent via initialize, which logs a warning (@1250461). There are no `hook_on_failure_supported`, `hooks_applied` or `managed_settings_applied` checks.

### D29. Unknown control-request subtype: DIFFERENT

```js
// 0.3.325 @325:1359274, @325:1353050
throw new VE("Unsupported control request subtype: " + subtype, "Unsupported control request subtype", "unsupported_request");
… { type:"control_response", response:{ subtype:"error", request_id, error: msg(err), ...(err instanceof VE && { error_code: err.errorCode }) } }

// 0.3.311 @1250254, @1244581
throw Error("Unsupported control request subtype: " + e.request.subtype);
… let o = { type:"control_response", response:{ subtype:"error", request_id: e.request_id, error: st(r) } };
```

0.3.311 behaviour: the reply is

```json
{"type":"control_response","response":{"subtype":"error","request_id":"<id>","error":"Unsupported control request subtype: <subtype>"}}
```

No error reply written by 0.3.311 ever contains `error_code`; the string `unsupported_request` does not occur in the bundle. `st(x)` is `x instanceof Error ? x.message : String(x)`. Parsing of `error_code` on **inbound** error responses is unchanged (`/^[a-z][a-z0-9_]{0,63}$/`, @1272868).

### D36. Error-result substitution: DIFFERENT (minor)

Exit error strings are identical (`Claude Code process exited with code N`, `Claude Code process terminated by signal SIG`, then `. stderr: <tail>`). The difference is which frames clear the remembered error-result text.

```js
// 0.3.325
else if (!(e.type === "system" && (e.subtype === "session_writes_lost" || e.subtype === "plugin_install"))) {
  if (this.lastErrorResultText = void 0, this.lastStartupFailureReason = void 0, e.type === "system" && e.subtype === "init") …
}
// 0.3.311 @1242583 onward
else {
  if (this.lastErrorResultText = void 0, e.type === "system" && e.subtype === "init") this.modelLessTurnEnded = !1;
  else if (e.type !== "system" || e.subtype === "compact_boundary") this.errorTextBeforeModelLessTurns = void 0, this.modelLessTurnEnded = !1;
  …
}
```

0.3.311 behaviour: a `system/session_writes_lost` or `system/plugin_install` frame that arrives after an error `result` clears the stored text, so a following non-zero exit surfaces as the plain exit error. In 0.3.325 those two frames leave it in place. 0.3.311 also has no `startupFailureReason` on the substituted error. The full 0.3.311 algorithm is in checklist item 63 below.

---

## Section rules (§1–§8) beyond the checklist

### §1 argv

Table §1.2, 0.3.311 status per row: rows 0, 1, 1b, 2–10, 10b, 11–35 and 37 are **SAME** (same condition, same encoding, same position). Row 36 is **DIFFERENT**.

**X1. Row 36, managed settings.**

```js
// 0.3.325
if (this.options.managedSettingsViaInitialize) { if (!pluginsViaInitialize) argv.push("--await-initialize"); argv.push("--managed-settings-in-initialize"); }
else if (this.options.managedSettings) argv.push(...zE("--managed-settings", this.options.managedSettings));
// 0.3.311 @1225528
if (this.options.managedSettings) W.push(...Py("--managed-settings", this.options.managedSettings));
```

0.3.311 has no `managedSettingsDelivery` option and no `--managed-settings-in-initialize` flag. `managedSettings` always goes out as `--managed-settings` `<JSON.stringify(managedSettings)>` (two elements).

§1.3 derivations: skills, thinking, outputFormat, mcpServers, continue, settings, sandbox merge (`o6` @1216024, `r6` @1215715) and workload are SAME. Implicit `--await-initialize` is ABSENT (D9). §1.4: the extraArgs object is the user's object plus `workload`; `settings` is appended in the transport as before. §1.5 number formatting: SAME.

### §2 executable resolution

SAME. `u6` @1234319 has the same candidate list and musl test; `I9e` @1232380 the same extension list; `D9e` @1232463 the same four error texts.

### §3 environment

SAME apart from the version literal (D15). Verified in 0.3.311: `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING`, `CLAUDE_CODE_SDK_READS_SESSION_STATE`, `CLAUDE_CODE_SDK_READS_STARTUP_MARKS`, `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS`, `CLAUDE_CODE_SDK_HAS_OAUTH_REFRESH`, `CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH`, `CLAUDE_CODE_QUESTION_PREVIEW_FORMAT`, the three scrubbed keys, the OpenTelemetry injection, `CLAUDE_CONFIG_DIR` for store-backed resume, then `NODE_OPTIONS` / `DEBUG` in the transport (@1297065–@1298335, @1225832).

### §4 transport

SAME: stdio and `windowsHide`, child tracking and `killAll`, every `write()` branch, `readline` reader, stderr tail (2 × 2048 trigger, 2048 kept, redaction tables identical), 200 ms drain, exit error texts, `close()` timings, 2000 ms wait in `Query.cleanup()` (@1240025).

**X2.** `onProcessSpawned(pid)` takes one argument in 0.3.311; 0.3.325 adds `{spawnSyncMs}`. 0.3.311's transport also has an `onExit(cb)` method with `exitListeners` that `close()` drains (@1231585). `Query` never calls it. No wire effect.

### §5 Query lifecycle

**X3. Initialize payload fields sent by 0.3.311**, in this key order (@1252143):

```
subtype, hooks, sdkMcpServers, sdkMcpServerConfigs, sdkMcpServerManifests, jsonSchema, systemPrompt,
appendSystemPrompt, planModeInstructions, systemPromptSnapshot, appendSubagentSystemPrompt, toolAliases,
codeRunningMcpTools, excludeDynamicSections, agents, title, skills, webSearchIsolationExemptMcpServers,
promptSuggestions, agentProgressSummaries, forwardSubagentText, supportedDialogKinds, perTaskStopAffordance,
rapidFollowupPreempt, forwardNestedSubagentToolCalls, workspaceTrust, plugins
```

That is the 0.3.325 list minus the final `managedSettings`. Undefined values are dropped by `JSON.stringify`.

**X4. `sdkMcpServerConfigs` entries.** 0.3.311's normaliser (`mw` @864535) keeps `timeout` (positive integer), `disableAutoBackground` (only `true`), `tools`, `connectorId` (non-empty string). 0.3.325 also keeps `autoModeVerdictTools` and `dataFlowServer`; 0.3.311 does not know them and drops them.

**X5.** Manifest capture is SAME (250 ms, same ids, protocol `2025-11-25`, same `clientInfo` fields) except `clientInfo.version` is `"0.3.311"`. That value goes to the in-process server only, not to the CLI.

**X6.** Initialize timeout on `query()`: none at all (D23). `startup()` / `prewarm()` timer class is identical after identifier normalisation (60000 ms counted, 600000 ms uncounted allowance while `SessionStart` / `Setup` hooks run, `prewarm` default 60000; @1283906, @1284007).

§5.2 gating: only "after the initialize frame is written" (D23). §5.3, §5.4, §5.5, §5.6: SAME, including `hasBidirectionalNeeds` (@1237506), `endRun`, `armRunEndCeiling` (@1270059), `streamInput`, `interrupt`, `cleanup`.

### §6 control channel

SAME: request ID generation, UUID ids and `type`-first envelope for server-initiated `mcp_message`, no timeout, SDK→CLI cancel only for signalled requests (`askSideQuestion`, `readCloudSessions`), CLI→SDK cancel, success envelope, the four unanswered subtypes, duplicate and late request handling. DIFFERENT: error envelope has no `error_code` (D29).

### §7 handlers

`can_use_tool` (@1245258): field mapping character-for-character the same as §7.1. `hook_callback`, `mcp_message`, `elicitation`, `request_user_dialog`, `remote_control_work_secret`: SAME.

**X7. `oauth_token_refresh`** (@1248503). 0.3.311 calls `getOAuthToken({signal, onDecline, ...(kind !== undefined && {kind})})`. It does not read `token_state` or `refused_call`, so `tokenState` and `refusedCall` are never passed. Reply shape unchanged.

**X8. `host_auth_token_refresh`** (@1248832). 0.3.311 calls `getHostAuthToken({signal})`; `status` is not passed. Reply shape unchanged.

### §8 other

JSON encoding and frame list: SAME. **X9.** Outbound request payloads that differ: `mcp_authenticate` has no `redirectUriOnly`, `claude_authenticate` has no `consoleProfile`, `reload_plugins` has no `refuse_if_busy`. `dropHookMatchers`, `archiveSession`, `unarchiveSession` and `matchDeniedModels` do not exist. **X10.** Error metadata: `awaitControlResponse` errors carry no `errorCode`.

---

## Errors and imprecisions found in `transport.md` (they apply to 0.3.325 itself)

- **E1 (§4.3, item 27). Line splitting is understated.** Node's `readline` also splits on **U+2028 and U+2029**. Checked with the same `createInterface({input})` call on local Node v24.10.0: `{"a":"x y"}` (raw character) came back as two lines. In the SDK both halves fail `JSON.parse` and are dropped as `Non-JSON stdout`. A Go reader that splits only on `\n` is more tolerant than the SDK here, not equal to it.
- **E2 (§3, item 15). `CLAUDE_AGENT_SDK_VERSION` is not "set if unset".** The SDK first overwrites the parent's `process.env.CLAUDE_AGENT_SDK_VERSION`, then copies. Without `options.env` the child therefore always gets the SDK's own version, whatever was inherited. "Only if unset or empty" holds only when `options.env` is given.
- **E3 (item 17). `CLAUDE_CODE_AGENT_PROGRESS_SUMMARIES` has no enabling option.** Its flag is the literal `false`, so the key is always removed from an inherited env. §3's pseudo-code is right; the checklist wording is not.
- **E4 (§5.3, item 24). A ceiling of 0 disables the ceiling.** `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS=0` passes the `>= 0` test, and `armRunEndCeiling` returns early when `runEndCeilingMs <= 0`. Stdin then stays open until `idle`, with no timeout. The ceiling is also re-armed by every non-idle `session_state_changed` after a result, not only by the result.
- **E5 (§4.5, item 36). The substitution rule is narrower than "the last result was an error".** The stored text is cleared by any later frame that reaches the bookkeeping branch; it survives a later model-less success result (`num_turns === 0`, `result === ""`); and it is not applied when the thrown error is an `AbortError` or has `name === "SSEHttpError"`.
- **E6 (§5.1, item 34).** `pending_*` fields are dispatched for the initialize response whatever its subtype; the code does not test for `success`.
- **E7 (item 28).** IDs are `Math.random().toString(36).substring(2,15)`: 0 to 13 characters in principle (empty when `Math.random()` returns 0), usually 10 to 12. "1–13" is not guaranteed.
- **E8 (§1.3 sandbox). Two merge rules are missing.** After the deep merge, `sandbox.filesystem.disabled === true` inherited from settings is deleted when the option supplies `filesystem` (or a `credentials.files` entry with `mode:"deny"`) without its own `disabled`; and inherited `network.httpProxyPort` / `socksProxyPort` are deleted when the option supplies `allowedDomains`, a non-empty `deniedDomains` or `strictAllowlist:true` without those ports. Settings that look like JSON but fail to parse are silently treated as `{}`.
- **E9 (§7.4).** `remote_control_work_secret` with no answerer throws `refreshWorkSecret callback is not provided.` (an error reply), and a session mismatch throws `remote_control_work_secret names a session this host did not attach.` The reference lists only the success reply.
- **E10 (§5.5).** `active_goal` and `autocompact_state` are top-level `type` values, not `system` subtypes. The table row reads as if they were subtypes.
- **E11 (§3).** `CLAUDE_CONFIG_DIR` is also set for `query()` with a store-backed resume (through `transport.updateEnv` before the deferred spawn), not only for `startup()`.

None of E1–E11 changed between 0.3.311 and 0.3.325.

---

## (c) Go implementation checklist for SDK 0.3.311 / CLI 2.1.311

Standalone. "Truthy" means a JS truthiness test: `0`, `""`, `null` and absent are all skipped. Whether a Go client should present itself as `sdk-ts` / `0.3.311` (items 14, 15) is a product decision; the list states what the TypeScript SDK does.

### Spawn and argv

1. Command: if the CLI path does **not** end in `.js`, `.mjs`, `.ts`, `.tsx` or `.jsx` (so `.cjs` counts as native), run it directly with `args = [...executableArgs, ...argv]`. Otherwise run `<executable>` (default `node`, or `bun` under Bun) with `args = [...executableArgs, path, ...argv]`.
2. Spawn with three pipes (stdin, stdout, stderr), the caller's `cwd`, and a hidden window on Windows.
3. `argv` starts with exactly `--output-format`, `stream-json`, `--verbose`, `--input-format`, `stream-json`.
4. Every valued built-in flag is **one** element `--flag=value`. The only exceptions are `--mcp-config` and `--managed-settings`, which are two elements (flag, JSON).
5. Push order after the prefix. Emit a row only when its condition holds:

   | # | condition | element(s) |
   |---|---|---|
   | 1 | thinking `enabled` without budget, or `adaptive` | `--thinking=adaptive` |
   | 1 | thinking `enabled` with budget N | `--max-thinking-tokens=N` |
   | 1 | thinking `disabled` | `--thinking=disabled` |
   | 1b | thinking not disabled and `display` truthy | `--thinking-display=<display>` |
   | 2 | `effort` truthy | `--effort=<v>` |
   | 3 | `maxTurns` truthy | `--max-turns=<n>` |
   | 4 | `maxBudgetUsd` defined (0 is emitted) | `--max-budget-usd=<n>` |
   | 5 | `taskBudget` truthy | `--task-budget=<taskBudget.total>` |
   | 6 | `model` truthy | `--model=<m>` |
   | 7 | `agent` truthy | `--agent=<a>` |
   | 8 | `betas` non-empty | `--betas=<comma join>` |
   | 9 | output format is `json_schema` with a truthy schema | `--json-schema=<compact JSON>` |
   | 10 | `debugFile` truthy | `--debug-file=<path>` |
   | 10 | else `debug` truthy | `--debug` |
   | 10b | no `debugFile`, default spawner, and `DEBUG_CLAUDE_AGENT_SDK` truthy in the **parent** env | `--debug-file=<configHome>/debug/sdk-<uuid>.txt` (one path per process; also print `SDK debug logs: <path>` to parent stderr) |
   | 11 | `canUseTool` set | `--permission-prompt-tool=stdio` |
   | 11 | else `permissionPromptToolName` | `--permission-prompt-tool=<name>` |
   | 12 | `permissionPrompts` truthy | `--permission-prompts=<v>` |
   | 13 | `continue` truthy | `--continue` |
   | 14 | `resume` truthy | `--resume=<id>` |
   | 16 | allowed tools (with skills, item 9) non-empty | `--allowedTools=<comma join>` |
   | 17 | `disallowedTools` non-empty | `--disallowedTools=<comma join>` |
   | 18 | `tools` is a list, including empty | `--tools=<comma join>` (empty list gives the literal `--tools=`) |
   | 18 | `tools` is the preset object | `--tools=default` |
   | 19 | at least one non-in-process MCP server | `--mcp-config`, `{"mcpServers":{…}}` (two elements, caller's key order) |
   | 20 | `settingSources` defined, including empty | `--setting-sources=<comma join>` (empty gives `--setting-sources=`) |
   | 21 | `strictMcpConfig` | `--strict-mcp-config` |
   | 22 | `permissionMode` truthy | `--permission-mode=<m>` |
   | 23 | `allowDangerouslySkipPermissions` | `--allow-dangerously-skip-permissions` |
   | 24 | `fallbackModel` truthy | `--fallback-model=<m>` |
   | 25 | `includeHookEvents` | `--include-hook-events` |
   | 26 | `includePartialMessages` | `--include-partial-messages` |
   | 28 | `projectConfigRoot` defined | `--project-config-root=<p>` |
   | 29 | each additional directory | `--add-dir=<d>` |
   | 30 | plugins delivered via initialize (≥1 plugin) | `--await-initialize` |
   | 30 | else each plugin (`type` must be `local`) | `--plugin-dir=<path>`, or `--plugin-dir-no-mcp=<path>` when `skipMcpDiscovery` |
   | 31 | `forkSession` | `--fork-session` |
   | 32 | `resumeSessionAt` truthy | `--resume-session-at=<uuid>` |
   | 33 | `resumeDropsTurn` defined | `--resume-drops-turn=<v>` |
   | 34 | `sessionId` truthy | `--session-id=<uuid>` |
   | 35 | `persistSession === false` | `--no-session-persistence` |
   | 36 | `managedSettings` truthy | `--managed-settings`, `<compact JSON>` (two elements) |
   | 37 | extra args (item 7) | see item 7 |

   Rows 15 (`--channels`) and 27 (`--session-mirror`) exist in the SDK but are reachable only through features a Go client does not have (`channels` is not wired from `query()`; `--session-mirror` needs a session store).
6. Errors before spawn: `canUseTool` together with `permissionPromptToolName`; `fallbackModel` equal to `model`; a plugin whose `type` is not `local`.
7. Extra args come after all built-in flags, in this order: the caller's keys (integer-like keys first in ascending order, then insertion order), then `workload` if the workload option is set, then `settings` if the settings option is set (it keeps the caller's position if the caller already had a `settings` key). A `null` value emits `--key`. Any other value is stringified and emits two elements `--key`, `value`, or one element `--key=value` when the value is longer than one character and starts with `-`.
8. `settings`: an object is sent as compact JSON, a string as-is. With a `sandbox` option, the settings value must look like JSON (`{…}` after trimming) or the call fails with `Cannot use both a settings file path and the sandbox option. Include the sandbox configuration in your settings file instead.` The sandbox option is deep-merged into `settings.sandbox`:
   - arrays at `filesystem.denyRead`, `filesystem.denyWrite`, `network.deniedDomains`, `credentials.files`, `credentials.envVars` are concatenated and de-duplicated;
   - `ripgrep` and `network.tlsTerminate` are replaced, not merged;
   - an inherited `filesystem.disabled: true` is removed when the option supplies `filesystem` (or a `credentials.files` entry with `mode:"deny"`) without `disabled`;
   - inherited `network.httpProxyPort` / `socksProxyPort` are removed when the option supplies `allowedDomains`, non-empty `deniedDomains` or `strictAllowlist: true` without them;
   - `enabled: true` without `failIfUnavailable` sets `failIfUnavailable: true`.
9. Skills: `'all'` appends `Skill` to allowed tools; a list appends `Skill(<name>)` per name; entries already in the caller's allowed tools are not added again.
10. `maxThinkingTokens` (used only when `thinking` is not given): `0` means thinking `disabled`; any other value means `enabled` with that budget.
11. Resume and continue add **no** `--await-initialize`. There is no `--managed-settings-in-initialize`.
12. Numbers are formatted as JavaScript does (`0.5`, `1e-7`, `1e+21`), not as Go's `%g`.

### Environment

13. Base env: the caller's env if one is given (it replaces the inherited env entirely), otherwise a copy of the process env.
14. `CLAUDE_CODE_ENTRYPOINT=sdk-ts` if unset or empty.
15. `CLAUDE_AGENT_SDK_VERSION=0.3.311`: always, when the env is inherited; only if unset or empty, when the caller supplied an env.
16. `CLAUDE_CODE_SDK_READS_SESSION_STATE=1` unless a key with that name in any letter case already exists.
17. `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true` when file checkpointing is enabled. `CLAUDE_CODE_SDK_HAS_OAUTH_REFRESH=1` / `CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH=1` when the matching callback exists. `CLAUDE_CODE_QUESTION_PREVIEW_FORMAT=<v>` when configured.
18. `CLAUDE_CODE_QUESTION_EXTENDED` and `CLAUDE_CODE_QUESTION_OPTIONAL_DESCRIPTIONS`: set to `1` when the option is on. When the option is off **and the env is inherited**, remove every case variant of the key. `CLAUDE_CODE_AGENT_PROGRESS_SUMMARIES` is always removed (every case variant) from an inherited env.
19. Remove `NODE_OPTIONS` (exact case).
20. If `DEBUG_CLAUDE_AGENT_SDK` in the child env is truthy (`1`, `true`, `yes`, `on`, case-insensitive, trimmed), set `DEBUG=1`. Otherwise remove `DEBUG`.
21. Run-end ceiling: read `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS` from the child env. Use it if it parses to a finite number ≥ 0, else 600000. A value of 0 means no ceiling. The timer delay is capped at 2147483647 ms.
22. No environment variable overrides the close or kill timings, or adds a control-request timeout.

### stdin, stdout, stderr

23. Every frame written is compact JSON plus `\n`. Keys keep their stated order; absent values are omitted, but `null`, `false`, `0`, `""` and `[]` are kept. Do not escape `<`, `>`, `&`, U+2028, U+2029 or non-ASCII (Go: custom encoder or `SetEscapeHTML(false)` plus handling for U+2028/9).
24. Writes are not flow-controlled. A write after stdin has been ended is dropped silently. A write to an exited process is an error. Stdin pipe errors (EPIPE) are logged, never raised.
25. Stdout is read line by line with no length limit. Lines that are empty after trimming are skipped. A line that is not valid JSON is skipped, never fatal. The SDK's reader also treats a lone `\r`, U+2028 and U+2029 as line ends; splitting on `\n` (with optional preceding `\r`) is the safe superset.
26. Stderr is always drained. Keep a tail: append each chunk, and once the tail exceeds 4096 UTF-16 units, redact secrets and keep the last 2048 units (dropping a leading unpaired low surrogate). Forward chunks to the caller's stderr callback only if one is set or `DEBUG_CLAUDE_AGENT_SDK` is truthy in the child env.
27. After the process exits, wait up to 200 ms for stderr to close before reporting the exit.

### Initialize

28. On start: connect in-process MCP servers, start the stdout reader, then send initialize immediately. It is the first frame on stdin.
29. With in-process MCP servers (and `CLAUDE_AGENT_SDK_DISABLE_MCP_MANIFESTS` not truthy in the parent env), first run the manifest capture with a 250 ms overall deadline: `initialize` (id `sdk-manifest-capture:initialize`, protocol `2025-11-25`, `capabilities:{}`, `clientInfo:{name:"claude-code",title:"Claude Code",description:"Anthropic's agentic coding tool",websiteUrl:"https://claude.com/claude-code",version:"0.3.311"}`), `notifications/initialized`, then `tools/list` (id `sdk-manifest-capture:tools-list`) if the server reports a tools capability. Keep `toolsListResult` only when there is no `nextCursor` and the server sent no `tools/list_changed` during capture.
30. Envelope: `{"request_id":"<id>","type":"control_request","request":{…}}` with `request_id` first.
31. Request keys, in order, each omitted when absent: `subtype` (`"initialize"`), `hooks`, `sdkMcpServers`, `sdkMcpServerConfigs`, `sdkMcpServerManifests`, `jsonSchema`, `systemPrompt`, `appendSystemPrompt`, `planModeInstructions`, `systemPromptSnapshot`, `appendSubagentSystemPrompt`, `toolAliases`, `codeRunningMcpTools`, `excludeDynamicSections`, `agents`, `title`, `skills`, `webSearchIsolationExemptMcpServers`, `promptSuggestions`, `agentProgressSummaries`, `forwardSubagentText`, `supportedDialogKinds`, `perTaskStopAffordance`, `rapidFollowupPreempt`, `forwardNestedSubagentToolCalls`, `workspaceTrust`, `plugins`. There is no `managedSettings` key.
32. `hooks`: `{Event:[{"matcher":…,"hookCallbackIds":["hook_0",…],"timeout":…}]}`. IDs come from one counter starting at 0, assigned in event order, then matcher order, then hook order. Events with an empty list are omitted. The key is absent when no hooks option was given. There is no `onFailure`.
33. `sdkMcpServers`: list of in-process server names, absent when there are none. `sdkMcpServerConfigs`: `{name:{timeout?, disableAutoBackground? (only true), tools?, connectorId?}}`, only for servers with at least one of those. `sdkMcpServerManifests`: `{name:{initializeResult, toolsListResult?}}` from item 29.
34. `systemPrompt`: no option → `[""]`; a string → `["<string>"]`; a list → as given; custom → its prompt (a string is wrapped in a list) plus `systemPromptSnapshot`; preset → key absent, with `appendSystemPrompt`, `excludeDynamicSections`, `systemPromptSnapshot` set from the preset.
35. `skills` is sent only when the option is a list (not for `'all'`). `jsonSchema` is the output-format schema. `plugins` is sent only with initialize delivery. `workspaceTrust` is dropped (with a warning) if `accepted` is true and `directory` is not an absolute path.
36. There is no initialize timeout and no check of the initialize response beyond a warning when plugins were sent via initialize and `plugins_applied` is not `true`.
37. If the initialize response carries `pending_permission_requests`, handle each entry whose `request.subtype` is `can_use_tool` as if it had arrived as a control request, with `server_policy_ask_only` cleared. Do the same for `pending_user_dialog_requests` entries whose subtype is `request_user_dialog`. Ignore `pending_elicitation_requests`. On any other response these fields are ignored.

### Prompts and closing stdin

38. A string prompt is written as `{"type":"user","session_id":"","message":{"role":"user","content":[{"type":"text","text":"<prompt>"}]},"parent_tool_use_id":null}` plus `\n`. No `uuid`, `priority` or `timestamp`. With verbatim prompts, append `"client_composed":true` as the last key.
39. Prompts, and every other frame, wait only until the initialize frame has been **written**. Nothing waits for the initialize response. With no in-process MCP servers the prompt follows initialize immediately.
40. "Callbacks present" means any of: at least one in-process MCP server, a hooks object with at least one key, `canUseTool`, `onElicitation`, `onUserDialog`, `getOAuthToken`, `getHostAuthToken`.
41. Track `sessionState` (the `state` of the most recent `system/session_state_changed`, initially unset) and `resultReceived`.
    - On `result`: set `resultReceived`. If `sessionState` is unset or `idle`, or no callbacks are present, end the run. Otherwise arm the ceiling timer (item 21).
    - On `session_state_changed` with `idle`: end the run if `resultReceived`.
    - On `session_state_changed` with another state (unless this is string mode and the run already ended): mark the run as not ended; for `requires_action` clear the ceiling timer; otherwise, if `resultReceived`, re-arm it.
    - On `assistant` or `stream_event` with `parent_tool_use_id === null` (same exception): mark the run as not ended and clear the ceiling timer.
    - When the ceiling timer fires: end the run.
42. String mode: ending the run closes stdin. It happens once; the run is never reopened.
43. Streaming mode: write each caller message verbatim as one JSON line (adding only `client_composed:true` with verbatim prompts), in the caller's key order. Before each write, reset `resultReceived` and the run-ended flag and clear the ceiling. Stop early on abort. When the input ends: if at least one message was consumed and callbacks are present, wait for the run to end (or abort, or close), then close stdin; otherwise close stdin at once.

### Reading stdout

44. Never yielded: `control_response`, `control_request`, `control_cancel_request`, `keep_alive`, `transcript_mirror`, and `system/session_state_changed` with `sdk_host_only === true`.
45. Every other parsed object is yielded unchanged, including `session_state_changed` without `sdk_host_only`.
46. `control_response`: resolve the pending request with that `response.request_id`. If none is pending, store it in a map capped at 1024 entries (oldest evicted).
47. `keep_alive`: a string `startup_mark` of at most 64 characters is recorded (used only by startup timeouts).

### Control channel

48. Outbound request IDs: `Math.random().toString(36).substring(2,15)`, i.e. up to 13 characters of `[0-9a-z]`. Any unique string works. Requests have no timeout; they end on a response, on close (`Query closed before response received`), or on the caller's own cancellation.
49. Cancelling an outbound request (only possible for requests issued with a cancellation signal): reject locally and, if the frame was already written, send `{"type":"control_cancel_request","request_id":"<id>"}`.
50. An inbound error response rejects with its `error` text. Its `error_code` is kept only if it matches `^[a-z][a-z0-9_]{0,63}$`.
51. Replies to CLI requests:
    - success: `{"type":"control_response","response":{"subtype":"success","request_id":"<id>","response":<value>}}`, with `response` absent when the handler returned nothing;
    - error: `{"type":"control_response","response":{"subtype":"error","request_id":"<id>","error":"<message>"}}`. **Never** an `error_code`.
    - An unknown subtype gets the error `Unsupported control request subtype: <subtype>`. `ui_copy`, `ui_prompt_read`, `ui_prompt_fill`, `ui_prompt_paint`, `ui_prompt_suggest`, `ui_read_selection` get the same error unless a UI host is attached.
    - No reply at all for `remote_tool_call`, `remote_plumbing_call`, `remote_tools_probe`, `remote_tools_reannounce`.
    - A request whose `request_id` is already being handled is ignored. Requests arriving after close are ignored, and replies computed after close are not written.
    - `control_cancel_request` from the CLI cancels that handler's context. The handler's eventual result or error is **still written**.

### Handlers

52. `can_use_tool`: call the callback with `tool_name`, `input` and these options (source field → option): `permission_suggestions` → `suggestions`, `blocked_path` → `blockedPath`, `mcp_server{name,source}` → `mcpServer` (only if present), `decision_reason` → `decisionReason`, `decision_reason_type` → `decisionReasonType` (string only), `decision_reason_code` → `decisionReasonCode` (string only), `server_policy_ask_only` → `serverPolicyAskOnly` (boolean only), `title`, `display_name` → `displayName`, `description`, `default_to_no` → `defaultToNo`, `suppress_always_allow_rule` → `suppressAlwaysAllowRule`, `tool_use_id` → `toolUseID`, `agent_id` → `agentID`, `plugin` (string only), envelope `request_id` → `requestId`, `requires_user_interaction` → `requiresUserInteraction` (boolean only), `server_prompt` → `serverPrompt` (string only), `computer_folder{path,computer_name}` → `computerFolder` (both strings), `matched_ask_rule{source,tool_name,rule_content?}` → `matchedAskRule{source,toolName,ruleContent?}`. Reply `{...callbackResult,"toolUseID":<tool_use_id>}` with `toolUseID` last. The result is not normalised in any other way. A null result means no reply. No callback: error `canUseTool callback is not provided.`
53. `hook_callback`: look up `callback_id`; call it with `input`, `tool_use_id` and a cancellable context; reply with its result verbatim. Unknown ID: error `No hook callback found for ID: <id>`. The SDK applies no timeout; the CLI enforces the matcher `timeout`.
54. `mcp_message`: unknown `server_name` → error `SDK MCP server not found: <name>`. If `message` has `method` and a non-null `id`, pass it to the server and reply `{"mcp_response":<the server's message with the same id>}`. Anything else is passed to the server and answered at once with `{"mcp_response":{"jsonrpc":"2.0","result":{},"id":0}}`. Messages the server initiates go out as `{"type":"control_request","request_id":"<uuid v4>","request":{"subtype":"mcp_message","server_name":…,"message":…}}` (`type` first), after the initialize frame; their responses are not awaited.
55. `elicitation`: call the handler with `{serverName: mcp_server_name, message, mode, url, elicitationId: elicitation_id, requestedSchema: requested_schema, title, displayName: display_name, description}`; reply with its result verbatim; a null result means no reply. **No handler: reply `{"action":"decline"}`.**
56. `request_user_dialog`: call the handler with `{dialogKind: dialog_kind, payload, toolUseID: tool_use_id}`; reply verbatim; null means no reply. **No handler: no reply.** Declaring `supportedDialogKinds` without a handler is an error at start.
57. `oauth_token_refresh`: no callback → error `getOAuthToken callback is not provided.` Otherwise call it with `kind` (if present) and a decline hook; reply `{"accessToken":<token or null>}`, or `{"accessToken":null,"reason":<r>}` when the token is null and the decline hook was called with `signed_out`, `identity_changed`, `transient` or `refresh_failed`. `token_state` and `refused_call` are not read.
58. `host_auth_token_refresh`: no callback → error `getHostAuthToken callback is not provided.` Otherwise a string or null result is sent as `{"authToken":<v>}`, an object verbatim. `status` is not passed to the callback.

### Shutdown and errors

59. Close or abort: end stdin. If the process is still alive 2000 ms later, send SIGTERM. If it is still alive 5000 ms after that, send SIGKILL. On Windows there is no SIGTERM step: kill at 2000 + 5000 ms. Abort does not kill immediately. `close()` itself returns at once; the cleanup it starts (and the iterator's `return`) waits at most 2000 ms for the process to exit.
60. On parent exit, send SIGTERM to every live child (on Windows, end its stdin instead).
61. Close also cancels every in-flight handler context, rejects pending control requests with `Query closed before response received`, closes in-process MCP transports, and ends iteration normally.
62. Exit errors: `Claude Code process exited with code <n>` or `Claude Code process terminated by signal <SIG>`, followed by `. stderr: <tail>` when the redacted, trimmed tail (last 2048 units) is non-empty. After an abort the error is `Claude Code process aborted by user` instead.
63. Error-result substitution. Keep three values: `lastErrorText`, `textBeforeModelLess`, `modelLessEnded`.
    - On `result`: `lastErrorText` = (if `is_error`: `result` when subtype is `success`, else each of `errors` trimmed, empties dropped, joined with `"; "`), or unset if empty. `modelLessEnded` = subtype `success`, not `is_error`, `num_turns === 0`, `result === ""`. If not `modelLessEnded`, `textBeforeModelLess = lastErrorText`.
    - `session_state_changed`, the swallowed frames, `system` subtypes `post_turn_summary` / `task_summary` / `session_metadata`, and types `active_goal` / `autocompact_state` change nothing.
    - Any other frame: clear `lastErrorText`. If it is `system/init`, clear `modelLessEnded`. Otherwise, if it is not a `system` frame or it is `system/compact_boundary`, clear `textBeforeModelLess` and `modelLessEnded`. (0.3.311 has no exemption for `system/session_writes_lost` or `system/plugin_install`.)
    - When reading fails: let `t = lastErrorText`, or `textBeforeModelLess` if `modelLessEnded`. If `t` is set and the failure is not an abort, the iteration error is `Claude Code returned an error result: <t>`. Otherwise it is the original error.
64. Spawn failures: missing native binary → `Claude Code native binary not found at <p>. Please ensure Claude Code is installed via native installer or specify a valid path with options.pathToClaudeCodeExecutable.`; missing script → `Claude Code executable not found at <p>. Is options.pathToClaudeCodeExecutable set?`; path exists but cannot launch → `… exists but failed to launch …`; anything else → `Failed to spawn Claude Code process: <msg>`.
