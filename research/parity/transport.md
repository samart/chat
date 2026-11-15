# Claude Agent SDK (TypeScript) 0.3.325: how the SDK drives the `claude` CLI

Authoritative wire-parity reference for a Go re-implementation.

- Source: `sdk-pkgs/v0.3.325/package/sdk.mjs` (package `@anthropic-ai/claude-agent-sdk` 0.3.325, `claudeCodeVersion` 2.1.325). The code was read as data and never executed.
- Offsets are **byte offsets into the original minified `sdk.mjs`** at the start of the cited snippet. Excerpts are de-minified by hand, and minified identifiers are kept in comments where useful.
- `core.mjs` (the `./core` export) contains a byte-for-byte equivalent `ProcessTransport.initialize()`, with renamed identifiers only. Everything in §1–§4 applies to it too.
- Exports: `query` = `nrt` (@1411011), `startup` = `ort`, `prewarm` = `srt` (@1413963).
- Classes: `ProcessTransport` = `LD` (@1323479), `Query` = `YE` (@1343489), startup/initialize timer = `MD` (@1338938). Option normalisation = `ZD` (@1402630). Prompt dispatch = `JD` (@1410461).

Helpers used throughout:

| minified | meaning | offset |
|---|---|---|
| `nt(f,v)` | `` `${f}=${v}` `` — **every valued flag is emitted as one `--flag=value` argv element** | @1320772 |
| `zE(f,v)` | `v=String(v)`; if `v.length>1 && v.startsWith("-")` → `[f+"="+v]`, else `[f, v]` (two argv elements) | @1320807 |
| `V8(argv,k,v)` | `argv.push(...zE("--"+k, v))` (used for extraArgs) | @1320901 |
| `ue` | plain `JSON.stringify(x)` (no replacer, no indent) | |
| `je` | plain `JSON.parse` | |
| `Gs` | `crypto.randomUUID` | |
| `Pe(x)` | truthy env parse: boolean as-is; else `String(x).toLowerCase().trim()` ∈ `{"1","true","yes","on"}` | |

---

## 1. Options → argv

### 1.1 Assembly

```js
// ProcessTransport.initialize()  @1326577
let argv = ["--output-format","stream-json","--verbose","--input-format","stream-json"];
...
// final: @1330150
const isNative = Qtt(pathToClaudeCodeExecutable);      // see §2
const command  = isNative ? pathToClaudeCodeExecutable : executable;
const args     = isNative ? [...executableArgs, ...argv]
                          : [...executableArgs, pathToClaudeCodeExecutable, ...argv];
spawn(command, args, { cwd, stdio:["pipe","pipe","pipe"], signal: forwardedAbort.signal, env, windowsHide:true });
```

### 1.2 Exact push order and encoding

All entries are a **single argv element** unless marked "2 elems". "truthy" means a JS truthy check, so `0`, `""` and `undefined` are skipped.

| # | Condition | argv element(s) | evidence |
|---|---|---|---|
| 0 | always | `--output-format` `stream-json` `--verbose` `--input-format` `stream-json` (5 elems) | @1326577 |
| 1 | thinkingConfig.type=`enabled` with `budgetTokens===undefined` | `--thinking=adaptive` | @1326723 |
| 1 | `enabled` with budgetTokens | `--max-thinking-tokens=<N>` | @1326759 |
| 1 | `disabled` | `--thinking=disabled` | |
| 1 | `adaptive` | `--thinking=adaptive` | |
| 1b | type≠disabled && `display` truthy | `--thinking-display=<display>` | |
| 2 | `effort` truthy | `--effort=<v>` | @1327022 |
| 3 | `maxTurns` truthy | `--max-turns=<n>` | |
| 4 | `maxBudgetUsd !== undefined` (0 IS emitted) | `--max-budget-usd=<n>` | |
| 5 | `taskBudget` truthy | `--task-budget=<taskBudget.total>` | |
| 6 | `model` truthy | `--model=<m>` | @1327195 |
| 7 | `agent` truthy | `--agent=<a>` | |
| 8 | `betas?.length>0` | `--betas=<b1,b2>` (comma join) | |
| 9 | jsonSchema (from `outputFormat.type==="json_schema"`) | `--json-schema=<JSON.stringify(schema)>` | @1327307 |
| 10 | `debugFile` truthy | `--debug-file=<path>` | |
| 10 | else `debug` truthy | `--debug` | |
| 10b | `!debugFile && !spawnClaudeCodeProcess` and the **parent** env `DEBUG_CLAUDE_AGENT_SDK` is truthy | `--debug-file=<configHome>/debug/sdk-<uuid>.txt` (also printed to parent stderr: `SDK debug logs: <path>`) | @1327549, `Dq` @803220 |
| 11 | `canUseTool` set (and `permissionPromptToolName` set → **throw**) | `--permission-prompt-tool=stdio` | @1327701 |
| 11 | else `permissionPromptToolName` | `--permission-prompt-tool=<name>` | @1327753 |
| 12 | `permissionPrompts` truthy | `--permission-prompts=<v>` | |
| 13 | `continue` truthy (and **no** sessionStore) | `--continue` | |
| 14 | `resume` truthy | `--resume=<id>` | @1327870 |
| 15 | `channels?.length>0` (not wired from `query()` options; dead path) | `--channels=<c>` each | |
| 16 | effective allowedTools non-empty | `--allowedTools=<a,b,c>` | @1328033 |
| 17 | `disallowedTools` non-empty | `--disallowedTools=<a,b>` | |
| 18 | `tools` is an array (including `[]`) | `--tools=<a,b>`; for `[]` this is the literal `--tools=` | |
| 18 | `tools` is the preset object | `--tools=default` | @1328209 |
| 19 | non-SDK mcpServers object has ≥1 key | `--mcp-config` `<JSON>` (2 elems, because JSON starts with `{`) | @1328277 |
| 20 | `settingSources !== undefined` | `--setting-sources=<a,b>`; for `[]` this is the literal `--setting-sources=` | @1328341 |
| 21 | `strictMcpConfig` | `--strict-mcp-config` | |
| 22 | `permissionMode` truthy | `--permission-mode=<m>` | |
| 23 | `allowDangerouslySkipPermissions` | `--allow-dangerously-skip-permissions` | |
| 24 | `fallbackModel` (if equal to `model` → **throw**) | `--fallback-model=<m>` | |
| 25 | `includeHookEvents` | `--include-hook-events` | |
| 26 | `includePartialMessages` | `--include-partial-messages` | |
| 27 | `sessionStore` given (`sessionMirror`) | `--session-mirror` | |
| 28 | `projectConfigRoot !== undefined` | `--project-config-root=<p>` | |
| 29 | each `additionalDirectories` entry | `--add-dir=<d>` (repeated) | @1328920 |
| 30 | pluginDelivery=`initialize` with ≥1 plugin | `--await-initialize` (plugins go in initialize instead) | @1328953 |
| 30 | else each plugin; `type!=="local"` → **throw** `Unsupported plugin type: X` | `--plugin-dir=<path>` or, if `skipMcpDiscovery`, `--plugin-dir-no-mcp=<path>` | @1329107 |
| 31 | `forkSession` | `--fork-session` | @1329216 |
| 32 | `resumeSessionAt` truthy | `--resume-session-at=<uuid>` | |
| 33 | `resumeDropsTurn !== undefined` | `--resume-drops-turn=<v>` | |
| 34 | `sessionId` truthy | `--session-id=<uuid>` | @1329469 |
| 35 | `persistSession === false` | `--no-session-persistence` | @1329556 |
| 36 | managedSettingsDelivery=`initialize` with managedSettings | `--await-initialize` (only if #30 did not already push it), then `--managed-settings-in-initialize` | @1329682 |
| 36 | else managedSettings | `--managed-settings` `<JSON>` (2 elems) | @1329766 |
| 37 | extra args, in object insertion order (see 1.4) | `null` → `--<key>`; otherwise `zE("--"+key, value)` → normally 2 elems `--key` `value`, or 1 elem `--key=value` when the value starts with `-` and has length > 1 | @1329951 |

### 1.3 Per-option derivations (in `ZD`, @1402630)

- **skills** (@1326498): `'all'` appends `"Skill"` to allowedTools. A `string[]` appends `` `Skill(${name})` `` per name. Entries already present are not added again. Names are validated (no `(),` or control chars, no `*`, no leading `/`, and so on; see `Xtt`). The skills array is also sent in initialize (§5.1); `'all'` is not.
- **thinking / maxThinkingTokens** (@1407496): `thinking` wins. Otherwise `maxThinkingTokens===0` → `{type:"disabled"}` → `--thinking=disabled`, and any other defined value → `{type:"enabled",budgetTokens:N}` → `--max-thinking-tokens=N`.
- **outputFormat** (@1405753): when `type==="json_schema"`, `schema` becomes jsonSchema. It is sent **both** as `--json-schema=` and as initialize `jsonSchema`.
- **mcpServers** (@1407246): entries with `type==="sdk" && instance` are removed from argv and handled in-process (§7.3). All others are serialized verbatim as `JSON.stringify({mcpServers:{...rest}})`. If nothing remains, `--mcp-config` is omitted. SDK servers never appear in `--mcp-config`.
- **continue**: suppressed when `sessionStore` is set (`continueConversation: sessionStore ? undefined : continue`).
- **settings** (@1408582): an object becomes `JSON.stringify(settings)` and a string is passed as-is (file path or JSON). It is added as extraArgs key `settings`, so it emits `--settings <value>`.
- **sandbox** (`W8` @1319995): when `sandbox` is given, extraArgs.settings must be JSON-looking (`{…}`). Otherwise the SDK throws `Cannot use both a settings file path and the sandbox option…`. The settings are parsed, `settings.sandbox` is deep-merged with the `sandbox` option (`G8`), and the result is re-serialized as `JSON.stringify({...settings, sandbox: merged})`.
  - `G8` concatenates and de-duplicates the arrays at `filesystem.denyRead`, `filesystem.denyWrite`, `network.deniedDomains`, `credentials.files` and `credentials.envVars`. `ripgrep` and `network.tlsTerminate` are replaced, not merged.
  - `sandbox.enabled===true` without `failIfUnavailable` sets `failIfUnavailable:true`.
  - With no settings, the result is `--settings {"sandbox":{…}}`.
- **workload** option: becomes extraArgs `workload`, which emits `--workload <v>`.
- **Implicit `--await-initialize` on resume** (@1407788, `CQ`): if `(resume || continue)` AND `pathToClaudeCodeExecutable` was **not** given (the bundled native binary is used) AND there is no `spawnClaudeCodeProcess` AND extraArgs has no `sdk-url` / `sdk-url=…` key AND no executableArg matches `/^--sdk-url(=|$)/`, then the SDK adds extraArgs `"await-initialize": null`, which emits a bare `--await-initialize` in the extraArgs section. This can **duplicate** a `--await-initialize` already pushed by #30 or #36.

### 1.4 extraArgs object order

```js
// ZD @1408188
extraArgs: { ...userExtraArgs, ...(workload && {workload}), ...(CQ && {"await-initialize": null}) }
// transport @1329864
let Gt = { ...extraArgs }; if (settings) Gt.settings = settings;   // settings is LAST unless the user already had a "settings" key
let tt = W8(Gt, sandbox);                                           // may rewrite tt.settings
for (const [k,v] of Object.entries(tt)) v === null ? argv.push(`--${k}`) : V8(argv, k, v);
```

JS `Object.entries` order puts integer-like keys first in ascending order, then string keys in insertion order. Values are `String(v)`-converted.

### 1.5 Number formatting

Numeric values go through JS template/String conversion. Examples: `--max-budget-usd=0.5`, `1e-7` → `1e-7`, `1e21` → `1e+21`. A Go implementation must mimic JS `Number.prototype.toString`. `strconv.FormatFloat(f,'g',-1,64)` differs; for example Go gives `1e-07`.

---

## 2. Executable resolution

```js
// ZD @1405567
let path = options.pathToClaudeCodeExecutable;
if (!path) {
  const req = createRequire(fileURLToPath(import.meta.url));
  path = X8(p => req.resolve(p));    // @1337991
  if (!path) throw Error(`Native CLI binary for ${process.platform}-${process.arch} not found. Reinstall @anthropic-ai/claude-agent-sdk without --omit=optional, or set options.pathToClaudeCodeExecutable.`);
}
// X8 candidates (first that existsSync wins); suffix ".exe" on win32
// android: ["@anthropic-ai/claude-agent-sdk-linux-${arch}-android"]
// linux:   musl ? ["…-linux-${arch}-musl", "…-linux-${arch}"] : ["…-linux-${arch}", "…-linux-${arch}-musl"]
// other:   ["@anthropic-ai/claude-agent-sdk-${platform}-${arch}"]
// each + "/claude" + ext
// musl detection: linux && process.report.getReport().header.glibcVersionRuntime === undefined
```

- **Native vs JS** (`Qtt` @1336052): `isNative = ![".js",".mjs",".tsx",".ts",".jsx"].some(ext => path.endsWith(ext))`. Note that **`.cjs` counts as native**.
- **Native**: `command = path`, `args = [...executableArgs, ...argv]`. `executableArgs` are **still prepended** even for a native binary.
- **JS**: `command = executable` (default `"bun"` when running under Bun, i.e. `process.versions.bun` is defined; otherwise `"node"`; `"deno"` is allowed), `args = [...executableArgs, path, ...argv]`.
- `spawnClaudeCodeProcess(opts)` replaces the spawn. It receives `{command, args, cwd, env, signal}` and must return `{stdin, stdout, killed, exitCode, signalCode, kill, on, once, off}`. In that case the auto `--debug-file` (#10b) is skipped.
- Spawn failure messages (`ent` @1336135):
  - ENOENT, native: `Claude Code native binary not found at <p>. Please ensure Claude Code is installed via native installer or specify a valid path with options.pathToClaudeCodeExecutable.`
  - ENOENT, JS: `Claude Code executable not found at <p>. Is options.pathToClaudeCodeExecutable set?`
  - The path exists but fails to launch: `… exists but failed to launch …` (the native variant mentions a musl/glibc mismatch).
  - Other errors: `Failed to spawn Claude Code process: <msg>`.

---

## 3. Environment

The base is `options.env` (shallow-copied) if given, **replacing** `process.env` entirely; otherwise `{...process.env}`.

```js
// ZD @1403023 — mutates the PARENT process env before copying it
process.env.CLAUDE_AGENT_SDK_VERSION = "0.3.325";
let env = options.env ? {...options.env} : {...process.env};
if (!env.CLAUDE_CODE_ENTRYPOINT)      env.CLAUDE_CODE_ENTRYPOINT = "sdk-ts";          // @1405861 (falsy, so "" is overwritten)
if (!env.CLAUDE_AGENT_SDK_VERSION)    env.CLAUDE_AGENT_SDK_VERSION = "0.3.325";
if (enableFileCheckpointing)          env.CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING = "true";
if (!Object.keys(env).some(k => k.toUpperCase()==="CLAUDE_CODE_SDK_READS_SESSION_STATE"))
                                      env.CLAUDE_CODE_SDK_READS_SESSION_STATE = "1";   // @1406116
if (readsStartupMarks /* startup()/prewarm() only */ && !anyKeyCI("CLAUDE_CODE_SDK_READS_STARTUP_MARKS"))
                                      env.CLAUDE_CODE_SDK_READS_STARTUP_MARKS = "1";
// run-end ceiling read from the CHILD env: CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS (finite, ≥0) else 600000   @1406301
if (getOAuthToken)                    env.CLAUDE_CODE_SDK_HAS_OAUTH_REFRESH = "1";
if (getHostAuthToken)                 env.CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH = "1";
if (toolConfig?.askUserQuestion?.previewFormat) env.CLAUDE_CODE_QUESTION_PREVIEW_FORMAT = previewFormat;
for ([flag, NAME] of [[aUQ.extendedQuestions,"CLAUDE_CODE_QUESTION_EXTENDED"],
                      [aUQ.optionalDescriptions,"CLAUDE_CODE_QUESTION_OPTIONAL_DESCRIPTIONS"],
                      [false,"CLAUDE_CODE_AGENT_PROGRESS_SUMMARIES"]])
  if (flag) env[NAME] = "1";
  else if (!options.env) delete every key k with k.toUpperCase()===NAME;   // only scrubs INHERITED env
// OpenTelemetry: if an active OTel context injects "traceparent", delete TRACEPARENT/TRACESTATE
// unless the user env has them, then set the UPPERCASED injected keys (unless the user env has them).
// startup() with sessionStore resume: env.CLAUDE_CONFIG_DIR = <tmp resume dir>
// (win32 also sets CLAUDE_SECURESTORAGE_CONFIG_DIR).

// then ProcessTransport.initialize() @1330030
if (!env.CLAUDE_CODE_ENTRYPOINT) env.CLAUDE_CODE_ENTRYPOINT = "sdk-ts";
delete env.NODE_OPTIONS;                                    // exact-case key only
if (Pe(env.DEBUG_CLAUDE_AGENT_SDK)) env.DEBUG = "1"; else delete env.DEBUG;   // DEBUG is STRIPPED by default
```

Parent-process env vars the SDK reads itself:

- `DEBUG_CLAUDE_AGENT_SDK`: SDK debug log file and auto `--debug-file`.
- `CLAUDE_AGENT_SDK_DISABLE_MCP_MANIFESTS`: disables the manifest capture in §5.1.
- `CLAUDE_CONFIG_DIR`: the config home used for the debug path.

There is **no env override** for the close/kill timeouts or for control request timeouts.

---

## 4. Transport (`ProcessTransport` = `LD`)

### 4.1 stdio

- `stdio: ["pipe","pipe","pipe"]` and `windowsHide: true` (@1324653). Spawn receives `signal: forwardedAbort.signal`, which is aborted only during close (§4.5).
- Every child is tracked. On the parent's `process.on("exit")`, all live children get `SIGTERM`; on win32 they get `stdin.end()` instead (`killAll` @1321266).

### 4.2 stdin writes (`write` @1332697)

- If the abort signal has fired, the write throws `AbortError("Operation aborted")`.
- With deferred spawn, writes are buffered.
- If not ready, it throws `ProcessTransport is not ready for writing`.
- If `stdin.writableEnded`, the write is **silently dropped**.
- If the process was killed or `exitCode !== null`, it throws `Cannot write to terminated process`.
- If an exit error exists, it throws `Cannot write to process that exited with error: …`.
- Otherwise it calls `stdin.write(str)`. A `false` return value is only logged (`Write buffer full, data queued` @1333584). **There is no backpressure or drain wait**; Node buffers internally.
- Every frame written by the SDK is `JSON.stringify(obj) + "\n"`.
- `stdin` `error` events (EPIPE) are logged and set `ready=false`. They are never thrown.

### 4.3 stdout reader (`readMessages` @1334881)

- `readline.createInterface({input: stdout})` @1335100 has **no maximum line length**. Node readline splits on `\n`, `\r\n` **and a lone `\r`**.
- A line where `line.trim()===""` is skipped.
- `JSON.parse(line)` (untrimmed). A parse failure is logged as `Non-JSON stdout: …` and **skipped**, never fatal (@1335336).
- When stdout ends, a stored `exitError` is thrown. Otherwise the reader calls `waitForExit()`, which rejects on a non-zero exit code or a signal.
- A child `error` event closes the reader.

### 4.4 stderr (`spawnLocalProcess` @1324653)

- stderr is always consumed through a UTF-8 `StringDecoder`.
- `stderrTail += chunk`. When its length exceeds `2*2048`, it is cut to the last 2048 UTF-16 units of the **secret-redacted** text (`vm` redacts bearer tokens, `sk-ant-`, AWS keys, GitHub tokens, JWTs and so on), and a leading lone low surrogate is dropped (@1325066).
- Chunks are forwarded to `options.stderr(chunk)` (and the SDK debug log) only if `options.stderr` is set or `DEBUG_CLAUDE_AGENT_SDK` is truthy **in the child env** (@1324934).
- Exit is synthesized as the event `sdk-exit-after-stderr-drained`. It fires when both the `exit` event and stderr `close` have happened, or **200 ms** after `exit` if stderr has not closed (`Ztt=200`, @1325535). All `on("exit")` listeners registered through the wrapper actually listen to this event, so exit errors include the full stderr tail.

### 4.5 Exit errors (`getProcessExitError` @1332040)

- `Claude Code process exited with code <n>` + tail
- `Claude Code process terminated by signal <SIG>` + tail
- The tail is `. stderr: <last 2048 chars, redacted, trimmed>`, or empty.
- If the abort signal fired, the error is instead `AbortError("Claude Code process aborted by user")`.
- **Error-result substitution** (@1352155): when the reader throws (for example on a non-zero exit) and the last `result` frame was an error, the iterator error becomes `Claude Code returned an error result: <text>`. Here `<text>` is `result` when subtype is `success`, otherwise `errors.map(trim).filter(Boolean).join("; ")`.

### 4.6 close / abort / kill (`close` @1333731, constants @1320946)

```js
close():
  spawnAbort(...); stdin.end(); remove abort listener; record exit error if already exited
  if (proc alive)
    setTimeout(2000 /*Ytt*/, () => {
      if exited: forwardAbortIfUserAborted(); return
      if win32: setTimeout(5000, () => { if (exitCode===null) kill("SIGKILL"); forwardAbort… }); return
      kill("SIGTERM"); setTimeout(5000, () => { if (exitCode===null) kill("SIGKILL") }); forwardAbortIfUserAborted()
    }).unref()
```

- Sequence: **stdin EOF → 2000 ms grace → SIGTERM → 5000 ms → SIGKILL**. On Windows there is no SIGTERM: SIGKILL comes at 2000+5000 ms. All timers are `unref`'d.
- Aborting `abortController` calls `transport.close()` and nothing else: no immediate kill, and the same grace period applies. Afterwards `forwardedAbort.abort(reason)` fires, the signal passed to `spawn`, so Node also sends its default `SIGTERM`.
- `Query.cleanup()` calls `transport.close()` and then waits at most **2000 ms** for `waitForExit()` (@1348351).

---

## 5. Query lifecycle (`Query` = `YE`)

### 5.1 Construction and initialize

Constructor order:

1. Connect each in-process SDK MCP server (`instance.connect(transport)`).
2. Start the stdout reader.
3. Call `this.initialize({first:true})` **immediately**, before any prompt is written.

If SDK MCP servers exist and `CLAUDE_AGENT_SDK_DISABLE_MCP_MANIFESTS` is not truthy, the SDK first runs an **in-process manifest capture** (`captureSdkMcpServerManifests` @1363652, overall deadline **250 ms**, `g7`). For each server:

- `initialize` with `id:"sdk-manifest-capture:initialize"`, `protocolVersion:"2025-11-25"`, `capabilities:{}`, `clientInfo:{name:"claude-code",title:"Claude Code",description:"Anthropic's agentic coding tool",websiteUrl:"https://claude.com/claude-code",version:"0.3.325"}`
- then the `notifications/initialized` notification
- then, if the server reports `capabilities.tools`, `tools/list` with `id:"sdk-manifest-capture:tools-list"`. `toolsListResult` is kept only when there is no `nextCursor`, and dropped if the server sent `tools/list_changed` during capture.

Initialize payload (`buildInitializeRequest` @1361603). **Key order matters**. Keys whose value is `undefined` are omitted by `JSON.stringify`, but `false`, `null`, `""` and `[]` are kept.

```js
{ subtype:"initialize",
  hooks,                       // {Event: [{matcher, hookCallbackIds:["hook_0",…], timeout, onFailure}]}; events with empty arrays omitted; whole key omitted if no hooks option
  sdkMcpServers,               // [names] of in-process servers, or omitted
  sdkMcpServerConfigs,         // {name: {timeout?, disableAutoBackground?(only true), tools?, connectorId?, autoModeVerdictTools?, dataFlowServer?}} only non-empty
  sdkMcpServerManifests,       // {name: {initializeResult, toolsListResult?}} from capture
  jsonSchema,                  // outputFormat.schema
  systemPrompt,                // see below
  appendSystemPrompt, planModeInstructions, systemPromptSnapshot, appendSubagentSystemPrompt,
  toolAliases, codeRunningMcpTools, excludeDynamicSections, agents, title,
  skills,                      // only when skills is an array
  webSearchIsolationExemptMcpServers, promptSuggestions, agentProgressSummaries,
  forwardSubagentText, supportedDialogKinds, perTaskStopAffordance, rapidFollowupPreempt,
  forwardNestedSubagentToolCalls, workspaceTrust, plugins /*pluginDelivery=initialize*/, managedSettings /*delivery=initialize*/ }
```

- **hookCallbackIds** (@1361824): `` `hook_${n++}` ``, a single counter starting at 0 and running across all events in `Object.entries(hooks)` order, matcher order, then hook order. Each matcher object has the key order `matcher, hookCallbackIds, timeout, onFailure`; undefined keys are dropped.
- **systemPrompt** (`ZD` @1402816, wrapped into an array in build):

  | `options.systemPrompt` | `systemPrompt` | other fields |
  |---|---|---|
  | undefined | **`[""]`** (empty prompt, not the Claude Code preset) | |
  | `"str"` | `["str"]` | |
  | `string[]` | as-is | |
  | `{type:"custom",prompt,snapshot}` | prompt (a string is wrapped in an array) | `systemPromptSnapshot` |
  | `{type:"preset",append,excludeDynamicSections,snapshot}` | **omitted** | `appendSystemPrompt=append`, `excludeDynamicSections`, `systemPromptSnapshot` |

- `workspaceTrust` is dropped, with a warning, if `accepted===true` and `directory` is not absolute.
- Envelope: `{"request_id":"<id>","type":"control_request","request":{…}}`. Note that **`request_id` comes before `type`**.

**Initialize timeout**:

- `query()` has **no initialize timeout**. `request()` has no timer.
- The only timer on the `query()` path is the prompt-hold guard: **660000 ms** (`r7 = 60000 + 600000`, @1338880). It is armed only when a hook matcher has `onFailure` other than `"continue"`/null, or when managedSettings are delivered via initialize. When it expires, the Query is cleaned up with `Claude Code did not answer initialize, so a hook set to onFailure: "block"…` or `… managed settings could not be confirmed…`.
- `startup()`/`prewarm()` use `MD` (@1413101). With no `initializeTimeoutMs` (startup), the budget is 60000 ms counted plus up to 600000 ms not counted while `SessionStart`/`Setup` hooks are reported running (`system/hook_started`, `hook_response`). With an explicit value, only that many ms are counted, with no uncounted allowance. `prewarm` defaults to 60000. Expiry produces the error `Subprocess initialization did not complete within <N>ms. …` with `errorClass:"initialize_timeout"`, plus `startupMark`/`startupMarkAgeMs` taken from the last `keep_alive.startup_mark`.
- There is no env override for any of these.

**Initialize response checks** (@1360818):

- With a blocking hook: `hook_on_failure_supported!==true` → error and cleanup. On first init, `hooks_applied===false` → error and cleanup.
- managedSettings via initialize with `managed_settings_applied!==true` → error and cleanup.
- Plugins via initialize with `plugins_applied!==true` → warning only.

**pending_*_requests** (@1372481):

- In the response handler of a request whose subtype is `initialize`: each `pending_permission_requests[i]` whose `request.subtype==="can_use_tool"` is re-dispatched through `handleControlRequest`, with `server_policy_ask_only` forced to undefined. Each `pending_user_dialog_requests[i]` whose subtype is `request_user_dialog` is re-dispatched as-is.
- `pending_elicitation_requests` is **ignored**.
- On any other subtype the fields are only logged.

### 5.2 Write ordering and gating

- `request()` (@1372975): if the initialize frame has not been written yet, the write waits on `initializeWritten`. The initialize frame itself is written synchronously when it is built, after manifest capture.
- When `promptHold` is active (blocking hook or managedSettings via initialize), the following are held until the initialize **response** succeeds:
  - user prompts
  - `interrupt` and `cancel_async_message`
  - with managedSettings via initialize, every control request except `initialize`/`get_binary_version`

  If initialize fails, the held writes are dropped.
- SDK→CLI MCP messages are written after the initialize frame is written.

### 5.3 String prompt (`JD` @1410461)

```js
writePrompt(JSON.stringify({type:"user", session_id:"", message:{role:"user",content:[{type:"text",text:prompt}]},
                            parent_tool_use_id:null, ...(verbatimPrompts && {client_composed:true})}) + "\n")
```

- There is **no `uuid`**. `session_id` is `""`. `content` is always a one-element text-block array, never a bare string.
- `isSingleUserTurn = true`. Stdin is closed by `endRun()` (@1381464, `Run ended for single-turn query, closing stdin`).
- `endRun()` triggers on `result` if `sessionState===undefined || sessionState==="idle" || !hasBidirectionalNeeds()`. Otherwise it is deferred until `system/session_state_changed{state:"idle"}`, or until the run-end ceiling expires (default 600000 ms, or `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS` from the child env, clamped to 2^31-1).
- So **with hooks, canUseTool, SDK MCP servers, onElicitation, onUserDialog, getOAuthToken or getHostAuthToken, stdin stays open after the `result`** until the CLI reports `idle`. If the CLI never sends `session_state_changed`, stdin closes at the first `result`. The env `CLAUDE_CODE_SDK_READS_SESSION_STATE=1` tells the CLI to send these frames.
- `hasBidirectionalNeeds` (@1344814) is: sdkMcpServers.size>0 ‖ hooks with ≥1 key ‖ canUseTool ‖ onElicitation ‖ onUserDialog ‖ getOAuthToken ‖ getHostAuthToken ‖ workSecretAnswerer.
- A new `assistant`/`stream_event` with `parent_tool_use_id===null`, or a `session_state_changed` that is not idle, re-opens the run (`runEnded=false`) unless this is a single-turn query whose run already ended. `requires_action` clears the ceiling timer.

### 5.4 AsyncIterable prompt (`streamInput` @1380322)

- Waits until initialize is written.
- Each yielded message is written as `JSON.stringify(verbatimPrompts ? {...msg, client_composed:true} : msg) + "\n"`. The message is otherwise **verbatim**: the SDK adds no `uuid`, `session_id`, `parent_tool_use_id` or `priority`. What the caller supplies goes out unchanged and in its own key order.
- Iteration stops early if the query is aborted.
- After the iterable ends: if ≥1 message was sent and `hasBidirectionalNeeds()`, the SDK waits for `endRun` (`result` + `idle`, as above) and then calls `endInput()`. Otherwise it calls `endInput()` (stdin EOF) immediately.
- `isSingleUserTurn` is false, so `endRun` itself never closes stdin in this mode.

### 5.5 stdout routing (`Query.readMessages` @1348736)

| frame | action |
|---|---|
| `control_response` | resolve the pending request by `response.request_id`; if none, store it in an `unmatched` map (cap 1024, FIFO evict). **Swallowed.** |
| `control_request` | `handleControlRequest` (§6). **Swallowed.** |
| `control_cancel_request` | abort that request's AbortController. **Swallowed.** |
| `keep_alive` | if `startup_mark` is a string of ≤64 chars, record it. **Swallowed.** |
| `transcript_mirror` | forwarded to the sessionStore batcher. **Swallowed.** |
| `system/session_state_changed` with `sdk_host_only===true` | state bookkeeping. **Swallowed.** |
| `system` `post_turn_summary`/`task_summary`/`session_metadata`, `active_goal`, `autocompact_state` | yielded directly, skipping bookkeeping and the frame observer |
| `system/commands_changed` | cached for `supportedCommands()`, then yielded |
| `result` | flush the mirror, record error text, maybe `endRun` (§5.3), yield |
| everything else | yielded unchanged |

### 5.6 End of iteration, interrupt, close, return, abort

- When stdout ends cleanly, the inputStream is marked done → the iterator returns `done`. Then `cleanup()`.
- When the reader throws (non-zero exit, abort), the iterator rejects with that error, or with the error-result substitution in §4.5.
- `interrupt({cancelQueued?})` (@1366033) sends control_request `{subtype:"interrupt"[,cancel_queued:true]}`. It returns `undefined` or `{still_queued:[…], cancelled?:[…]}`. It does not close anything.
- `close()` calls `cleanup()`: abort all in-flight handler signals, `transport.close()` (§4.6), reject pending control requests with `Query closed before response received`, close the in-process MCP transports, end the iterator **normally**, and wait ≤2 s for exit.
- `return()` / `for await … break` calls `cleanup()` and then ends the generator. `throw()` behaves the same.
- Abort via `abortController`:
  - The transport's abort listener calls `transport.close()`; the grace and kill sequence from §4.6 follows.
  - `streamInput` stops.
  - Later writes throw `Operation aborted`.
  - The reader eventually throws `AbortError("Claude Code process aborted by user")`, so the iterator rejects with that AbortError.

---

## 6. Control channel

- **request_id (SDK→CLI requests)**: `Math.random().toString(36).substring(2,15)`, a base-36 string of up to 13 chars (@1372975). The exception is **SDK MCP server→CLI messages**, which use `crypto.randomUUID()` (@1383764) and the envelope `{"type":"control_request","request_id":"<uuid>","request":{"subtype":"mcp_message","server_name":…,"message":…}}`. Here **`type` comes first**. Their responses are never awaited; they land in the unmatched map.
- **Timeout**: none. Control requests wait until a response arrives, cleanup runs, or the optional caller signal fires.
- **SDK→CLI cancel** (@1373415): only for requests made with a `signal`, which applies to `askSideQuestion` and `readCloudSessions`. When the signal aborts, the promise rejects. If the frame was already written, the SDK sends `{"type":"control_cancel_request","request_id":"<id>"}`.
- **CLI→SDK cancel** (@1353595): aborts the AbortSignal passed to the handler. **The SDK still writes the handler's eventual response or error** unless the Query was cleaned up.
- **Response envelope** (@1353050):
  - Success: `{"type":"control_response","response":{"subtype":"success","request_id":"<id>","response":<value>}}`. If the handler returned `undefined`, the `response` key is absent.
  - Error: `{"type":"control_response","response":{"subtype":"error","request_id":"<id>","error":"<message>"[,"error_code":"<code>"]}}`. `error` is `err.message` for an Error, otherwise `String(x)`. `error_code` appears only for the internal `VE` error class.
- **Unknown subtype** → error response with `error:"Unsupported control request subtype: <subtype>"` and `error_code:"unsupported_request"` (@1359274). `ui_copy`, `ui_prompt_read`, `ui_prompt_fill`, `ui_prompt_paint`, `ui_prompt_suggest` and `ui_read_selection` are handled only if a `uiHost` was set; otherwise they get the same error.
- **Leave unanswered** (the `Xu` sentinel, the equivalent of "VS"; no frame is written):
  - subtypes `remote_tool_call`, `remote_plumbing_call`, `remote_tools_probe`, `remote_tools_reannounce` (@1342819)
  - `can_use_tool` when the callback returns `null`
  - `elicitation` when `onElicitation` returns `null`
  - `request_user_dialog` when there is no `onUserDialog`, or it returns `null`
- **Duplicate / late requests** (@1352647):
  - The same `request_id` arriving while already in flight is skipped silently.
  - A request arriving after cleanup is skipped.
  - A response computed after cleanup is not written.
- **Inbound error responses** to SDK requests reject with `Error(error)`. `errorCode` is set only if `error_code` matches `/^[a-z][a-z0-9_]{0,63}$/`.

---

## 7. CLI→host handlers (`processControlRequest` @1353737)

### 7.1 `can_use_tool` (@1353962)

```js
if (!canUseTool) throw Error("canUseTool callback is not provided.");
r = await canUseTool(req.tool_name, req.input, {
  signal, suggestions: req.permission_suggestions, blockedPath: req.blocked_path,
  ...(req.mcp_server && {mcpServer:{name:req.mcp_server.name, source:req.mcp_server.source}}),
  decisionReason: req.decision_reason,
  ...(typeof req.decision_reason_type==="string" && {decisionReasonType}),
  ...(typeof req.decision_reason_code==="string" && {decisionReasonCode}),
  ...(typeof req.server_policy_ask_only==="boolean" && {serverPolicyAskOnly}),
  title: req.title, displayName: req.display_name, description: req.description,
  defaultToNo: req.default_to_no, suppressAlwaysAllowRule: req.suppress_always_allow_rule,
  toolUseID: req.tool_use_id, agentID: req.agent_id,
  ...(typeof req.plugin==="string" && {plugin}),
  requestId: <envelope request_id>,
  ...(typeof req.requires_user_interaction==="boolean" && {requiresUserInteraction}),
  ...(typeof req.server_prompt==="string" && {serverPrompt}),
  ...(computer_folder is {path:string, computer_name:string} && {computerFolder:{path, computer_name}}),
  ...(req.matched_ask_rule && {matchedAskRule:{source, toolName: tool_name, ...(rule_content!==undefined && {ruleContent})}})
});
if (r === null) return LEAVE_UNANSWERED;
return { ...r, toolUseID: req.tool_use_id };   // @1355729 — callback keys first, toolUseID last (overrides)
```

- The result is **not otherwise normalized**. For example `updatedInput` is not filled from `input`. If `tool_use_id` is undefined, `toolUseID` is dropped from the JSON.

### 7.2 `hook_callback` (@1355770, @1382546)

- `fn = hookCallbacks.get(callback_id)`. If it is missing, the SDK throws `No hook callback found for ID: <id>`, which goes out as an error response.
- Response = `await fn(req.input, req.tool_use_id, {signal})`, sent verbatim as `response`.
- There is **no SDK-side timeout**. The matcher `timeout` (seconds) goes in initialize and the CLI enforces it. A CLI cancel aborts `signal`.

### 7.3 `mcp_message` (@1356204)

- Server lookup by `server_name`. If it is missing: `SDK MCP server not found: <name>` (error response).
- If `message` has `"method"` and a non-null `"id"` (a request), it is passed to the in-process transport's `onmessage`. The SDK waits for the server to send a message whose `id` equals the request id (correlation key `${server}:${id}`), then answers with `{"mcp_response": <that JSON-RPC message>}`. There is no timeout. If the transport has no `onmessage`: error `No message handler registered`.
- Anything else (a notification, or a response to a server-initiated request) is passed to `onmessage` and answered immediately with `{"mcp_response":{"jsonrpc":"2.0","result":{},"id":0}}`.
- Messages the server initiates go to the CLI as `control_request`/`mcp_message` with UUID ids (§6). Server→CLI notifications sent before initialize is written are delayed until after it.

### 7.4 Others

- **`elicitation`** (@1356258): calls `onElicitation({serverName: mcp_server_name, message, mode, url, elicitationId: elicitation_id, requestedSchema: requested_schema, title, displayName: display_name, description}, {signal, requestId})`. The result is sent verbatim; `null` means no reply. **With no handler the reply is `{"action":"decline"}`.**
- **`request_user_dialog`** (@1356666): calls `onUserDialog({dialogKind: dialog_kind, payload, toolUseID: tool_use_id}, {signal, requestId})`. The result is verbatim; `null` means no reply. **With no handler there is no reply.** `supportedDialogKinds` without `onUserDialog` throws at `query()`.
- **`oauth_token_refresh`** (@1357188): with no callback it throws `getOAuthToken callback is not provided.` Otherwise it calls `getOAuthToken({signal, onDecline, kind?, tokenState?, refusedCall?})`. The reply is `{accessToken: token ?? null}`, or `{accessToken: null, reason}` when the token is null and `onDecline(reason)` was called with one of `signed_out|identity_changed|transient|refresh_failed`.
- **`host_auth_token_refresh`** (@1357814): with no callback it throws `getHostAuthToken callback is not provided.` Otherwise it calls `getHostAuthToken({signal, status})`. A string or null result is wrapped as `{authToken: v}`; an object is sent verbatim.
- **`remote_control_work_secret`**: replies `{work_secret: … || null}` (only used with remote control).

---

## 8. Other parity details

- **JSON encoding**:
  - JS `JSON.stringify` preserves insertion order and omits `undefined` but keeps `null`, `false`, `0`, `""`.
  - It does **not** escape `<`, `>`, `&`, U+2028 or U+2029, or non-ASCII.
  - It escapes `"`, `\\`, `\b \f \n \r \t`, and other C0 controls as lowercase `\u00xx`, plus lone surrogates.
  - Go `encoding/json` escapes `<>&` (unless `SetEscapeHTML(false)`) and **always** escapes U+2028/U+2029, sorts map keys, and `omitempty` drops `false`/`0`/`""`. A byte-exact implementation needs a custom encoder: ordered fields, "undefined" vs present-zero semantics, and no escaping of `<>&`/U+2028/U+2029.
- **User message fields**:
  - String prompts: `session_id:""`, `parent_tool_use_id:null`, no `uuid`, no `priority`, no `timestamp`.
  - Streamed messages: passed through untouched.
- **Frames written by the SDK**:
  1. initialize `control_request`
  2. user message(s)
  3. `control_response` to CLI requests
  4. other `control_request`s (`interrupt`, `set_model`, `set_permission_mode`, `set_max_thinking_tokens` with `{max_thinking_tokens, thinking_display}`, `mcp_status`, `mcp_set_servers`, `rewind_files`, …)
  5. `control_cancel_request`
  6. MCP-originated `control_request`s
- `setMcpServers` sends `{subtype:"mcp_set_servers", servers:{...nonSdk, ...{name:{type:"sdk",name,...settings}}}}`. `createSdkMcpServer` returns `{type:"sdk",name,instance[,timeout]}`.
- `CLAUDE_AGENT_SDK_VERSION` is also written into the **parent** `process.env` (@1403023).
- The SDK debug log (`Rn`) writes to the same file passed as `--debug-file` when `DEBUG_CLAUDE_AGENT_SDK` is set.

---

## Go implementation checklist

1. `argv` (after any executable/executableArgs prefix) begins with exactly `--output-format`, `stream-json`, `--verbose`, `--input-format`, `stream-json`.
2. Every valued flag pushed by the builder is one element `--flag=value`. The only exceptions are `--mcp-config <json>` and `--managed-settings <json>`, which use two elements, or one `--flag=value` element if the value starts with `-` and has length > 1.
3. The flag order follows table §1.2 exactly. For example, with model+resume+allowedTools the order is `--model=…`, then `--resume=…`, then `--allowedTools=…`.
4. `settingSources: []` produces the literal element `--setting-sources=`; an omitted `settingSources` produces no flag.
5. `tools: []` produces `--tools=`; the preset object produces `--tools=default`.
6. `canUseTool` set produces `--permission-prompt-tool=stdio`. Setting both `canUseTool` and `permissionPromptToolName` is an error before spawn.
7. SDK-type MCP servers (with an in-process instance) never appear in `--mcp-config`. When only SDK servers exist, `--mcp-config` is absent. Otherwise the value is `{"mcpServers":{…}}` in the user's key order.
8. `extraArgs` come after every built-in flag. A `null` value produces `--key`. A string value produces the two elements `--key`, `value`, or one `--key=value` element if the value starts with `-` and has length > 1. The `settings` key (with sandbox merged in) comes last unless the user supplied it earlier.
9. With resume or continue, the bundled binary, no custom spawner and no sdk-url, a bare `--await-initialize` is appended in the extraArgs section, even if one was already pushed for plugins or managed settings.
10. `maxThinkingTokens: 0` produces `--thinking=disabled`, and `N` produces `--max-thinking-tokens=N`. `thinking{type:"enabled"}` without a budget produces `--thinking=adaptive`.
11. `maxBudgetUsd: 0` emits `--max-budget-usd=0`, while `maxTurns: 0` emits nothing.
12. `skills: ['x']` adds `Skill(x)` to `--allowedTools` (de-duplicated) and `skills:["x"]` to initialize. `'all'` adds `Skill` to allowedTools and nothing to initialize.
13. `outputFormat` json_schema emits `--json-schema=<compact JSON>` and also sets initialize `jsonSchema`.
14. Native binary detection: a path not ending in `.js/.mjs/.ts/.tsx/.jsx` is spawned directly with `executableArgs` prepended. JS paths are spawned as `<executable> [...executableArgs] <path> [...argv]`.
15. The child env has `CLAUDE_CODE_ENTRYPOINT=sdk-ts` only if it was unset or empty. It has `CLAUDE_AGENT_SDK_VERSION=0.3.325` if unset. It has `CLAUDE_CODE_SDK_READS_SESSION_STATE=1` unless any case variant of that key exists.
16. The child env never contains `NODE_OPTIONS`. It contains `DEBUG` only as `DEBUG=1`, and only when `DEBUG_CLAUDE_AGENT_SDK` is truthy in the child env.
17. When `options.env` is given, it replaces the inherited env entirely; the SDK additions still apply. When it is not given, inherited `CLAUDE_CODE_QUESTION_EXTENDED`, `CLAUDE_CODE_QUESTION_OPTIONAL_DESCRIPTIONS` and `CLAUDE_CODE_AGENT_PROGRESS_SUMMARIES` (any case) are removed unless the corresponding option enables them.
18. The first frame written to stdin is the initialize control_request. Its JSON starts with `{"request_id":"`, followed by `","type":"control_request","request":{"subtype":"initialize"`.
19. With no `systemPrompt` option, initialize carries `"systemPrompt":[""]`. With the preset, `systemPrompt` is absent and `appendSystemPrompt` carries `append`.
20. Hook callback IDs are `hook_0`, `hook_1`, … assigned in event → matcher → hook order. Each matcher serializes as `{"matcher":…,"hookCallbackIds":[…],"timeout":…,"onFailure":…}`, with absent keys omitted.
21. When in-process MCP servers exist, initialize is delayed by at most about 250 ms for the manifest capture, and contains `sdkMcpServers:[names]`.
22. A string prompt is written as `{"type":"user","session_id":"","message":{"role":"user","content":[{"type":"text","text":…}]},"parent_tool_use_id":null}\n`, after the initialize frame and without waiting for its response (unless a hold applies, item 23).
23. If any hook matcher has `onFailure` set to a value other than `"continue"`, or managed settings are delivered via initialize, prompts, interrupt and cancel_async_message wait for a successful initialize response, with a 660 s guard.
24. String-prompt mode closes stdin at the first `result` when there are no callbacks or in-process servers, or when the CLI never sent `session_state_changed`. Otherwise it closes stdin on `session_state_changed{state:"idle"}` after `result`, or after `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS` (default 600000 ms).
25. Streaming mode writes caller messages verbatim (adding only `client_composed:true` when `verbatimPrompts`). It closes stdin when the input ends, or, if it has callbacks and sent ≥1 message, after the run ends.
26. `control_*`, `keep_alive`, `transcript_mirror` and `session_state_changed` with `sdk_host_only:true` are never yielded. Every other parsed stdout object is yielded.
27. Blank stdout lines and non-JSON stdout lines are skipped silently. There is no line length limit.
28. Outbound control request IDs are 1–13 chars of `[0-9a-z]`. MCP-originated request IDs are UUIDv4. Control requests have no timeout.
29. Success replies are `{"type":"control_response","response":{"subtype":"success","request_id":…,"response":…}}`. Error replies are `{"type":"control_response","response":{"subtype":"error","request_id":…,"error":"…"}}`, and an unknown subtype adds `"error_code":"unsupported_request"` with error text `Unsupported control request subtype: <x>`.
30. `can_use_tool` replies are `{...callbackResult,"toolUseID":<tool_use_id>}`. A null callback result gets no reply. A missing callback produces the error `canUseTool callback is not provided.`.
31. `mcp_message` notifications are acknowledged with `{"mcp_response":{"jsonrpc":"2.0","result":{},"id":0}}`. Requests are answered with `{"mcp_response":<server reply with the same id>}`.
32. `elicitation` with no handler is answered `{"action":"decline"}`. `request_user_dialog` with no handler, and `remote_tool_call`/`remote_plumbing_call`/`remote_tools_probe`/`remote_tools_reannounce`, get no reply.
33. A CLI `control_cancel_request` aborts the handler context, but a late handler result is still sent.
34. `pending_permission_requests` / `pending_user_dialog_requests` in the initialize success response are dispatched as if they had arrived as control requests. `pending_elicitation_requests` is ignored.
35. close/abort sends stdin EOF; then, if the child is alive after 2000 ms, SIGTERM; then SIGKILL after another 5000 ms if it is still alive. On Windows, SIGKILL comes at 7000 ms. Abort does not kill immediately.
36. The exit error text is `Claude Code process exited with code N` + `. stderr: <last ≤2048 chars, redacted>`. It is replaced by `Claude Code returned an error result: <text>` when the last `result` frame was an error.
37. The SDK waits ≤200 ms after process exit for stderr to close before it reports the exit.
