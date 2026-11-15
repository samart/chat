# Official VS Code extension 2.1.311: what it does when it drives the CLI

Checked 2026-11-15 against `anthropic.claude-code` **2.1.311** (darwin-arm64 VSIX), with the 2.1.325 extension (`reference/vsix-2.1.325/extension/`, linux-x64) as the comparison. Nothing from the VSIX was executed; all findings are static reads of the minified bundles. Offsets are character offsets into the 311 files so each excerpt can be re-found with `tools/ctx.py`.

- Host bundle: `reference/vsix-2.1.311/x/extension/extension.js` (3,678,953 bytes)
- Webview bundle: `reference/vsix-2.1.311/x/extension/webview/index.js` (6,027,333 bytes)

Verdicts: **CONFIRMED** = same as the 2.1.325-derived claim; **DIFFERENT** = present but not the same; **ABSENT** = not in 311.

## Provenance

| Check | Result |
| --- | --- |
| Source | VS Code Marketplace, `…/publishers/anthropic/vsextensions/claude-code/2.1.311/vspackage?targetPlatform=darwin-arm64`, HTTP 200, `application/vsix` |
| VSIX file | `reference/vsix-2.1.311/claude-code-2.1.311-darwin-arm64.vsix`, 127,474,048 bytes, SHA-256 `9dab6fe4ad8635c4233356be3344ab4453339f62177d21ec0b914d1ddade1fb9` |
| Unpacked to | `reference/vsix-2.1.311/x/` |
| `extension/package.json` | `name: claude-code`, `version: 2.1.311`, `publisher: Anthropic` |
| `extension.vsixmanifest` | `Version="2.1.311"`, `TargetPlatform="darwin-arm64"` |
| Bundled binary `extension/resources/native-binary/claude` | 276,185,504 bytes, SHA-256 `85d8c08707fcfd1e78ffa0204eefbc6ec9e4fe22f375fb2767ef85d957ed3161` |
| `reference/sdk-pkgs/v0.3.311/package/manifest.json` → `platforms.darwin-arm64` | checksum `85d8c08707fc…3161`, size 276185504: **match** on both |

Open VSX was not needed. The VSIX itself has no independent published checksum to compare against; the binary inside it matches the SDK manifest.

## Summary

| Claim | Verdict for 2.1.311 |
| --- | --- |
| a. Embedded Agent SDK, `spawnClaude` | CONFIRMED (SDK 0.3.311) |
| b. Options passed to the SDK | DIFFERENT (3 differences) |
| c. Child environment | DIFFERENT (1 variable absent) |
| d. Editor MCP tools and lockfile | CONFIRMED |
| e. Permission handling | CONFIRMED |
| f. Slash commands | CONFIRMED, with a larger locally handled set (same in 325) |
| g. Agent map | CONFIRMED (present, same handlers and record) |
| h. Session storage encoding | CONFIRMED |
| i. Control subtypes | CONFIRMED (identical set of outgoing requests) |
| j. Settings and keybindings | CONFIRMED (one description and one command differ) |

---

## a. Embedded Agent SDK and `spawnClaude` — CONFIRMED

- SDK version is **0.3.311** (325 embeds 0.3.325).

  ```
  @2758521  process.env.CLAUDE_AGENT_SDK_VERSION="0.3.311";let{abortController:D=pe(),additionalDirectories:M=[],…
  @2761090  if(!$$.CLAUDE_CODE_ENTRYPOINT)$$.CLAUDE_CODE_ENTRYPOINT="sdk-ts";if(!$$.CLAUDE_AGENT_SDK_VERSION)$$.CLAUDE_AGENT_SDK_VERSION="0.3.311";
  ```

- `spawnClaude` exists and logs the claimed text, with the CLI version hard-coded:

  ```
  @3459427  async spawnClaude($,J,Q,X,Y,W,z,G,K={},V,U,B,H,q,Z,F,D,M){…
  @3461596  let k="2.1.311";if(this.output.info(`Spawning Claude with SDK query function - cwd: ${A.cwd}, permission mode: ${A.permissionMode}, version: ${k}, ${A.pathToClaudeCodeExecutable}, resume: ${J}`)…
  ```

- The call is the SDK's warm-start entry followed by `.query(prompt)`, not a bare `query()`:

  ```
  let L1=(await bl1({options:A}).finally(…)).query(a);return this.trackClaudeProcess(L1),L1
  @2767121  async function bl1($={}){return(await Da0($)).warm}
  ```

- Three callers of `spawnClaude`: the chat channel (@3130005), a config probe (@3163868, `canUseTool` always denies with "Config loading only", env `CLAUDE_CODE_CONFIG_PROBE`), and an auth-only launch (@3194904, denies with "Auth only").

- **Difference from 325**: 325 retries the spawn once when initialize times out and the process has exited (gate `tengu_vscode_start_second_try`, flag `startRetried`). Neither identifier exists in 311; a failed start is simply reported.

- CLI arguments built by the embedded SDK (@2686484), same shape as 325: `--output-format stream-json --verbose --input-format stream-json`, then `--thinking=…` / `--max-thinking-tokens`, `--model`, `--permission-prompt-tool=stdio` (when `canUseTool` is set), `--resume`, `--mcp-config`, `--setting-sources`, `--permission-mode`, `--allow-dangerously-skip-permissions`, `--include-partial-messages`, `--add-dir`, and each `extraArgs` key as `--<key>`.

## b. Options passed to the SDK — DIFFERENT

Exact 311 options object (@3459427 onward):

```
A={cwd:j,resume:J,canUseTool:Q,onUserDialog:V,supportedDialogKinds:U,
   permissionMode:W??(m4("claudeProcessWrapper")?"default":void 0),
   ...m4("claudeProcessWrapper")?{}:{onProcessSpawned:(m)=>{Z?.(m),this.noteSpawnedProcess(m)}},
   onStartupHooksRunning:D?.onStartupHooksRunning,
   additionalDirectories:O,allowDangerouslySkipPermissions:z,model:X,
   stderr:(m)=>{…this.output.info(`From claude: ${m}`)},
   systemPrompt:{type:"preset",preset:"claude_code",append:t$4,snapshot:!0},
   enableFileCheckpointing:!0,thinking:G,includePartialMessages:!M1.env.remoteName,
   agentProgressSummaries:void 0,promptSuggestions:void 0,
   perTaskStopAffordance:!0,forwardNestedSubagentToolCalls:!0,
   hooks:{PreToolUse:[{matcher:"Edit|Write|MultiEdit",hooks:[(m)=>_.captureBaseline(m)]},
                      {matcher:"Edit|Write|Read",hooks:[(m,a,{signal:e})=>this.saveFileIfNeeded(m,e)]}],
          PostToolUse:[{matcher:"Edit|Write|MultiEdit",hooks:[(m)=>_.findDiagnosticsProblems(m)]}]},
   settingSources:["user","project","local"],
   toolConfig:{askUserQuestion:{previewFormat:"markdown"}},
   extraArgs:{debug:null,"debug-to-stderr":null,"enable-auth-status":null,"no-chrome":null,"replay-user-messages":null},
   mcpServers:K}
```

Set after construction: `pathToClaudeCodeExecutable`, `executableArgs`, `env` (section c), `abortController`.

| Option | 2.1.325 claim | 2.1.311 |
| --- | --- | --- |
| `promptSuggestions` | `true` | **`void 0`** (not requested) |
| `settingSources` | trusted: `user,project,local`; untrusted: `user` | **always `["user","project","local"]`**; no trust branch |
| `workspaceTrust` (attest) | present when trusted | **absent**; `postureFor` occurs 0 times in 311 (5 in 325) |
| `cwd` | trusted directory or resolved cwd | plain `Y \|\| this.cwd` |
| `extraArgs` | the five flags plus `safe-mode` and launch args such as `--ignore-persisted-trust` | **exactly the five flags** |
| `supportedDialogKinds` | 4 kinds | **3 kinds**: `refusal_fallback_prompt`, `fable_overage_consent_prompt`, `auto_mode_server_fallback` (@3094014). 325 adds `send_to_project_remote_control_offer` |
| everything else in the claim list | — | same |

Also present in both versions but missing from the claim list: `onProcessSpawned` (unless a process wrapper is configured), `onStartupHooksRunning`, `agentProgressSummaries: void 0`, `pathToClaudeCodeExecutable`, `executableArgs`, `abortController`.

Details:

- `includePartialMessages` is `!vscode.env.remoteName`: off in remote windows.
- `thinking` is `{type:"disabled"}` when the level is `off`, otherwise `{type:"enabled",budgetTokens:31999,display:"summarized"|undefined}` (@3091147, `$51=31999`).
- `mcpServers` is `{"claude-vscode": <in-process SDK server>, ...getAdditionalMcpServers()}` (@3127015); see section d.
- The model picker does not send `set_model`: `setModel` writes user settings and pushes them (@3226621).

## c. Child environment — DIFFERENT

Base environment (@3499618):

```
function g$($){let J=KO(m4("environmentVariables")),Q={...process.env};if($)Q.PATH=$;
  Q.MCP_CONNECTION_NONBLOCKING="true",Q.CLAUDE_CODE_ENABLE_TASKS="0",Q.CLAUDE_CODE_EMIT_STARTUP_TIMING="1";
  for(let Y of J){if(e71(Y.name))continue;if(Y.name)Q[Y.name]=Y.value||""}
  let X=$91(J,Boolean(m4("claudeProcessWrapper")));if(X!==void 0)Q.CLAUDE_CONFIG_DIR=X;
  return Object.assign(Q,nk()),Q.CLAUDE_CODE_ENTRYPOINT="claude-vscode",
  delete Q.CLAUDECODE,delete Q.CLAUDE_CODE_CHILD_SESSION,delete Q.TRACEPARENT,delete Q.TRACESTATE,Q}
```

| Variable | Claim | 2.1.311 |
| --- | --- | --- |
| `MCP_CONNECTION_NONBLOCKING=true` | set | CONFIRMED |
| `CLAUDE_CODE_ENABLE_TASKS=0` | set | CONFIRMED |
| `CLAUDE_CODE_EMIT_STARTUP_TIMING=1` | set | CONFIRMED |
| `CLAUDE_CODE_STARTUP_FAILURE_RESULTS=1` | set | **ABSENT**. The name appears only in the bundled env-name registry (@475547, @1737912), never assigned. 325 assigns it (@3428175 in the 325 file) |
| `CLAUDE_CODE_ENTRYPOINT="claude-vscode"` | set | CONFIRMED |
| delete `CLAUDECODE`, `CLAUDE_CODE_CHILD_SESSION`, `TRACEPARENT`, `TRACESTATE` | deleted | CONFIRMED |

Also set in 311 (same in 325):

- User `claudeCode.environmentVariables` entries, applied after the three defaults so they can override them; `CLAUDE_CONFIG_DIR` when derived from those settings.
- `DISABLE_TELEMETRY="1"` when VS Code telemetry is off (`nk()` @3432547); otherwise the variable is removed.
- Per spawn (@3461596 region): `CLAUDE_CODE_EMIT_USER_PROMPT_SUBMIT_HOOK_EVENTS="1"` unless gate `tengu_optimized_puzzle` is false; inherited `CLAUDE_CODE_RESUME_*` variables are stripped, then `CLAUDE_CODE_RESUME_INTERRUPTED_TURN="1"` and `CLAUDE_CODE_RESUME_INTERRUPTED_TURN_MAX_AGE_MS="3600000"` are added when an interrupted turn is being re-run (@3329517); caller extras such as `CLAUDE_CODE_CONFIG_PROBE`.
- By the SDK: `CLAUDE_AGENT_SDK_VERSION="0.3.311"`, `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING="true"`, `CLAUDE_CODE_SDK_READS_SESSION_STATE="1"`; it deletes `NODE_OPTIONS` and `DEBUG`.

Not in 311: `CLAUDE_CODE_IGNORE_PERSISTED_TRUST=1` for withheld-trust launches (325 only).

## d. Editor MCP server and terminal attach — CONFIRMED

All twelve tools are registered on one server (`e84`, @3630700 to @3634400). The server name is `Claude Code VSCode MCP` (or `Cursor` / `Windsurf` / `<appName>`), version = extension version.

| Tool | Inputs | Returns |
| --- | --- | --- |
| `openDiff` | `old_file_path`, `new_file_path`, `new_file_contents`, `tab_name` | `FILE_SAVED` + contents, or `DIFF_REJECTED` + tab name |
| `getDiagnostics` | optional `uri` | JSON diagnostics |
| `close_tab` | `tab_name` | `TAB_CLOSED` |
| `closeAllDiffTabs` | — | `CLOSED_<n>_DIFF_TABS` |
| `openFile` | `filePath`, `preview`, `startText`, `endText`, `selectToEndOfLine`, `makeFrontmost` | (read-only hint) |
| `getOpenEditors`, `getWorkspaceFolders`, `getCurrentSelection`, `getLatestSelection` | — | JSON |
| `checkDocumentDirty`, `saveDocument` | `filePath` | JSON |
| `executeCode` | `code` | Jupyter kernel output |

`openDiff` result logic (@3622500): accepted → `FILE_SAVED`; rejected or the diff tab closed → `DIFF_REJECTED`; when `files.autoSave` is `off`, saving the file also resolves `FILE_SAVED`.

**Two servers, not one.** This is the same in 325 but the spec blurs it:

- The 12-tool server above listens on a local **WebSocket** and is for `claude` running in a terminal.
- The embedded chat passes an in-process SDK server named **`claude-vscode`** that registers only `getDiagnostics` (@3093021 `dE({name:"claude-vscode",version:"2.1.311"})`, @3493086 `registerVSCodeMcpServerTools($){Z04($,ok)}`). The chat's tool is therefore `mcp__claude-vscode__getDiagnostics`. A second server, `claude-vscode-extension`, is added to a channel only while a debugger session or Jupyter is active. In the chat, diffs come from `canUseTool` and the `PreToolUse` / `PostToolUse` hooks, not from `openDiff`.

**Terminal attach** (@3628331 to @3636500):

```
function sq6(){return Math.floor(Math.random()*55536)+1e4}
function n84(){let $=Rf.join(Rz(),"ide");return VW.mkdirSync($,{recursive:!0,mode:448}),$}
function A31($,J){let Q=n84(),X=Rf.join(Q,`${$}.lock`)…
  z={pid:process.ppid,workspaceFolders:W,ideName:xf.env.appName,transport:"ws",runningInWindows:process.platform==="win32",authToken:J};
  VW.writeFileSync(X,JSON.stringify(z),{mode:384})…}
async function a84(){for(let $=0;$<50;$++){let J=sq6();if(await tq6(J))return J}throw Error("Failed to find an available port after multiple attempts")}
@1374391  function Rz(){if(process.env.CLAUDE_CONFIG_DIR)return process.env.CLAUDE_CONFIG_DIR;return x4.join(lU.homedir(),".claude")}
```

- **Lockfile path**: `$CLAUDE_CONFIG_DIR/ide/<port>.lock`, default `~/.claude/ide/<port>.lock`. Directory mode `0700`, file mode `0600`.
- **Contents** (one JSON object): `pid` (the extension host's parent pid, `process.ppid`), `workspaceFolders` (array of fs paths), `ideName` (`vscode.env.appName`), `transport: "ws"`, `runningInWindows` (boolean), `authToken` (a `crypto.randomUUID()`).
- **Port**: random in 10000–65535, up to 50 tries, each probed by binding a throwaway HTTP server to `127.0.0.1`. The real server listens on `127.0.0.1` only.
- **Auth**: each WebSocket upgrade must carry header `x-claude-code-ide-authorization` equal to `authToken`, else close code 1008 "Unauthorized". A new connection replaces the previous client.
- **Discovery by terminals**: `CLAUDE_CODE_SSE_PORT=<port>` is put into the integrated terminal environment through `context.environmentVariableCollection.replace(...)`.
- The lockfile is rewritten when workspace folders change and deleted on dispose.
- Server → client notifications: `diagnostics_changed` (`{uris}`), `selection_changed`, `at_mentioned`, `log_event` (`{eventName, eventData}`, only when telemetry is on).

## e. Permission handling — CONFIRMED

Webview (@3430869):

```
accept($={},J=[]){this.resolved.emit({behavior:"allow",updatedInput:$,updatedPermissions:J})}
reject($,J){this.resolved.emit({behavior:"deny",message:$,interrupt:J})}
```

Host (@3146249):

```
let z=await this.sendRequest($,{type:"tool_permission_request",toolName:J,inputs:Q,suggestions:X,...W},Y);…
let G=z.result;if(G.behavior!=="allow"||G.updatedPermissions===void 0)return G;
let K=Array.isArray(G.updatedPermissions)?G.updatedPermissions:[],V=this.settings.getAllowDangerouslySkipPermissions(),U=PQ(X),
    B=K.filter((H)=>DJ6(H,U)&&(V||!(H.type==="setMode"&&H.mode==="bypassPermissions")));
…return{...G,updatedPermissions:B}
```

`DJ6` (@3093753) accepts an update only if it equals one of the request's suggestions; for non-`setMode` updates the `destination` may differ but must be one of `userSettings`, `projectSettings`, `localSettings`, `session`, `cliArg`. A `setMode` to `bypassPermissions` is dropped unless `allowDangerouslySkipPermissions` is on.

The request the host forwards to the webview carries `toolName`, `inputs`, `suggestions`, `defaultToNo`, `suppressAlwaysAllowRule`, `toolUseId`, `agentId`, `plugin`. Tools prefixed `mcp__claude-in-chrome__` are auto-allowed while Chrome is connected and auto-allow is on. User dialogs: any `dialogKind` outside the supported list is answered `{behavior:"cancelled"}`.

## f. Slash commands — CONFIRMED, with more local handling than the claim lists

Menu click on a CLI-advertised command (@5279230):

```
function ET0($,J,Z){if(J==="usage"&&!Ku($,"usage"))return{run:Z.openUsageDialog,opensIdeDialog:!0};
  if(J==="context"&&Z.openContextDialog!==void 0&&!Ku($,"context"))return{run:Z.openContextDialog,opensIdeDialog:!0};
  return()=>Z.submit(`/${J}`)}
```

So `usage` and `context` open dialogs and everything else submits `/${name}`. But `submit` itself first runs the text through a local router (`Bb0`, table `_T5`, @5339200 to @5341700). A match is handled in the webview and nothing is sent to the CLI. A local entry yields when the CLI advertises a user-defined (non-built-in) command of the same name (`Ku`, @5278581); `/remote-control`, `/rc`, `/login` and `/logout` yield to any advertised command of that name (`Fk`). The same test guards the `usage` and `context` dialogs.

| Typed command | Local action |
| --- | --- |
| `/status` | status dialog |
| `/copy [N]` | copy to clipboard |
| `/remote-control`, `/rc` | toggle remote control (when offered) |
| `/skills` | skills dialog |
| `/login`, `/logout` | sign in / sign out |
| `/tasks` (and its aliases) | background tasks view |
| `/design-login` | Claude Design login (when signed in) |
| `/chrome` | Chrome settings (when supported) |
| `/sandbox`, `/sandbox exclude <pattern>` | sandbox dialog or add an excluded command |
| `/btw [question]` | side question (case-insensitive) |
| `/export [args]` | export dialog or direct export |
| `/plan [args]` | plan handling; only when the CLI's command list has arrived |
| `/bug`, `/feedback [report]` | feedback dialog |
| `/mcp` | MCP config dialog |
| `/config`, `/settings` | general config dialog |
| `/hooks` | hooks dialog |
| `/memory` | memory dialog |
| `/permissions`, `/allowed-tools` | permission rules dialog |
| `/plugin`, `/plugins`, `/marketplace` | plugins dialog |
| `/rewind`, `/checkpoint`, `/undo` | rewind dialog |
| `/resume`, `/continue` | resume-conversation picker |
| `/help` | slash-command browser |
| `/bookmarks` | bookmarks dialog |
| `/model` | model picker (not while a permission request is pending) |

The 2.1.325 table is identical entry for entry, so this is a correction to the claim for both versions, not a 311 difference.

## g. Agent map — CONFIRMED (present, same as 325)

Identifier counts match 325 exactly in the webview: `agentMapAgents` 30/30, `Agent map` 3/3, each `handleTask*` 2/2, `wakeToolUseIds` 8/8, `servedModel` 9/9, `processEnded` 7/7. The handler bodies differ from 325 only in minified names.

Dispatch (@3854925): `system/task_started`, `task_progress`, `task_notification`, `task_updated`, `background_tasks_changed`.

**Record** (created by `Hc1`, @3649153), keyed by `taskId`:

`taskId`, `toolUseId`, `parentToolUseId`, `description`, `prompt`, `subagentType`, `isBackgrounded`, `startTime`, `endTime`, `status`, `wakeToolUseIds`, `processEnded`, `textRemoved`; added later: `usage {totalTokens, toolUses, durationMs}`, `summary`, `outputFile`, `error`, `model`, `servedModel` / `servedModelAt`, `result` / `resultAt`.

**Statuses**: stored `working`, `finished`, `failed`, `stopped`; `waiting` is derived for display when a pending permission request has `agentId` equal to the task (`rl`). Labels: "Working", "Waiting for your permission", "Finished", "Failed", "Stopped". Wire mapping (`tl`, @3657403): `completed` → `finished`, `failed` → `failed`, `killed` and `stopped` → `stopped`.

**Message fields read per handler** (@3857317 to @3861900):

| Handler | Fields read |
| --- | --- |
| `handleTaskStarted` | `task_id`, `task_type`, `tool_use_id`, `is_backgrounded`, `skip_transcript`, `description`, `prompt`, `subagent_type`. Only `task_type === "local_agent"` creates an agent record; other types go to the background-task list |
| `handleTaskProgress` | `task_id`, `tool_use_id`, `usage.total_tokens`, `usage.tool_uses`, `usage.duration_ms`, `description`, `summary`, `last_tool_name` (last 3 kept as `recentTools`) |
| `handleTaskUpdated` | `task_id`, `patch.status`, `patch.error`, `patch.end_time`, `patch.is_backgrounded` |
| `handleTaskNotification` | `task_id`, `tool_use_id`, `status`, `summary`, `usage.{total_tokens,tool_uses,duration_ms}`, `output_file` |
| `handleBackgroundTasksChanged` | `tasks[].task_id`, `task_type`, `description` |

Other writers of the map: a provisional record keyed `call:<tool_use_id>` created from an `Agent` / `Task` tool call and re-keyed when the real `task_id` arrives (`cX1`); rebuild from the transcript on resume (`rX1`); process end turns every `working` record into `stopped` with `processEnded: true` (`Dc1`); stopping an agent calls host `stopSubagent` → `query.stopTask` (`stop_task`) and then marks it `killed`.

## h. Session storage — CONFIRMED

Extension / SDK copy (@576166, constant @587734, hash @426865):

```
function ep4($){return Math.abs(vi($)).toString(36)}
function sU($){return $.replace(/[^a-zA-Z0-9]/g,"-")}
function tU($){let J=sU($);if(J.length<=dJ)return J;return`${J.slice(0,dJ)}-${ep4($)}`}
dJ=200
function vi($){let J=0;for(let Q=0;Q<$.length;Q++)J=(J<<5)-J+$.charCodeAt(Q)|0;return J}
```

- Regex: `/[^a-zA-Z0-9]/g` → `-`.
- Max length: **200**. Longer names become the first 200 characters, `-`, then the hash.
- Hash: the 32-bit string hash `h = (h << 5) - h + charCode | 0` over UTF-16 code units of the **original** path, then `Math.abs(h).toString(36)`.
- The 2.1.311 CLI binary contains the same function and the same constant (read as bytes, not run): `var eEe=200;function Pe(e){return Math.abs(Ome(e)).toString(36)}` with `Ome` defined identically. `Bun.hash` exists beside it but is not used for this name.

Subagent transcripts (@2797896, @2799825):

```
let z=W.filePath.replace(/\.jsonl$/,"");return e5.join(z,"subagents")
filePath:e5.join(Q,...q.key.agentRelPath??[],`agent-${q.key.agentId}.jsonl`)
```

Path is `<projects>/<encoded cwd>/<sessionId>/subagents/[<agentRelPath…>/]agent-<agentId>.jsonl`. Note the optional nested `agentRelPath` segments. `.meta.json` is a reserved suffix in both versions; the string `spawnDepth` appears in neither extension bundle, so that field of the spec is not confirmed from extension code.

## i. Control subtypes — CONFIRMED (no difference in outgoing requests)

Host → CLI requests in the embedded SDK (`async name(…){…this.request({subtype:"x"`), 55 subtypes, identical to 325:

| SDK method | subtype |
| --- | --- |
| `interrupt` | `interrupt` |
| `setPermissionMode` | `set_permission_mode` |
| `setModel` | `set_model` |
| `setMaxThinkingTokens` | `set_max_thinking_tokens` |
| `setCwd` | `set_cwd` |
| `applyFlagSettings` | `apply_flag_settings` |
| `getSettings` / `updateSettings` | `get_settings` / `update_settings` |
| `getStatus` | `get_status` |
| `getContextUsage` | `get_context_usage` |
| `usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET` | `get_usage` |
| `getPlan` | `get_plan` |
| `rewindFiles` | `rewind_files` |
| `readFile` | `read_file` |
| `seedReadState` | `seed_read_state` |
| `stopTask` | `stop_task` |
| `backgroundTasks` | `background_tasks` |
| `getTaskOutput` | `get_task_output` |
| `askSideQuestion` | `side_question` |
| `cancelAsyncMessage` | `cancel_async_message` |
| `renameSession` / `generateSessionTitle` | `rename_session` / `generate_session_title` |
| `claimSession` | `claim_session` |
| `exportConversation` | `export_conversation` |
| `mcpServerStatus` | `mcp_status` |
| `setMcpServers` | `mcp_set_servers` |
| `reconnectMcpServer` / `toggleMcpServer` | `mcp_reconnect` / `mcp_toggle` |
| `mcpAuthenticate` / `mcpClearAuth` / `mcpSubmitOAuthCallbackUrl` | `mcp_authenticate` / `mcp_clear_auth` / `mcp_oauth_callback_url` |
| `readMcpResource` | `mcp_read_resource` |
| `setMcpPermissionModeOverride` | `set_mcp_permission_mode_override` |
| `listPermissionRules` | `list_permission_rules` |
| `getHooksListing` | `get_hooks_listing` |
| `getMemoryDialog` / `getSkillsDialog` / `getSandboxDialog` / `getChromeDialog` | `get_memory_dialog` / `get_skills_dialog` / `get_sandbox_dialog` / `get_chrome_dialog` |
| `getChromeBrowsers` / `selectChromeBrowser` / `setChromeBrowserHints` | `get_chrome_browsers` / `select_chrome_browser` / `set_chrome_browser_hints` |
| `reloadSkills` / `reloadPlugins` / `reloadOutputStyles` | `reload_skills` / `reload_plugins` / `reload_output_styles` |
| `setPromptSuggestionsPaused` | `set_prompt_suggestions_paused` |
| `enableRemoteControl` | `remote_control` |
| `enableChannel` | `channel_enable` |
| `launchUltrareview` | `ultrareview_launch` |
| `messageRated` / `submitFeedback` / `submitSurveyFeedback` | `message_rated` / `submit_feedback` / `submit_survey_feedback` |
| `claudeAuthenticate` / `claudeOAuthCallback` / `claudeOAuthWaitForCompletion` | `claude_authenticate` / `claude_oauth_callback` / `claude_oauth_wait_for_completion` |

Sent outside that pattern: `initialize` (@2716083; fields include `hooks`, `sdkMcpServers`, `sdkMcpServerConfigs`, `sdkMcpServerManifests`, `jsonSchema`, `systemPrompt`, `appendSystemPrompt`, `systemPromptSnapshot`, `agents`, `skills`, `promptSuggestions`, `agentProgressSummaries`, `forwardSubagentText`, `supportedDialogKinds`, `perTaskStopAffordance`, `rapidFollowupPreempt`, `forwardNestedSubagentToolCalls`, `workspaceTrust`, `plugins`; the SDK can carry `workspaceTrust` but the 311 extension never sets it), `mcp_message` (host → CLI for SDK MCP servers), and the UI-surface family `ui_attach`, `ui_detach`, `ui_render`, `ui_focus`, `ui_press`, `ui_scroll`, `ui_panes`, `ui_pane_show`, `ui_pane_focus`, `ui_close`. Control responses use `success` / `error`.

Methods the extension host visibly calls as `.query.<method>(`: `applyFlagSettings`, `askSideQuestion`, `backgroundTasks`, `enableRemoteControl`, `exportConversation`, `generateSessionTitle`, `getChromeBrowsers`, `getChromeDialog`, `getContextUsage`, `getHooksListing`, `getMemoryDialog`, `getPlan`, `getSandboxDialog`, `getSettings`, `getSkillsDialog`, `getStatus`, `getTaskOutput`, `initializationResult`, `interrupt`, `listPermissionRules`, `mcpAuthenticate`, `mcpClearAuth`, `mcpServerStatus`, `mcpSubmitOAuthCallbackUrl`, `messageRated`, `reconnectMcpServer`, `reloadOutputStyles`, `reloadPlugins`, `renameSession`, `rewindFiles`, `selectChromeBrowser`, `setMaxThinkingTokens`, `setMcpPermissionModeOverride`, `setMcpServers`, `setPermissionMode`, `setUiHost`, `stopTask`, `submitFeedback`, `submitSurveyFeedback`, `toggleMcpServer`, `usage_…`. The same list in 325. The login flow also calls `claudeAuthenticate`, `claudeOAuthCallback`, `claudeOAuthWaitForCompletion`. This list comes from a literal-pattern search, so a method reached through another variable name could be missing from it. Not called by the host: `setModel` (settings write instead), `setCwd`, `claimSession`, `seedReadState`, `reloadSkills`, `launchUltrareview`, `cancelAsyncMessage`, `readMcpResource`, `enableChannel`, `setChromeBrowserHints`, `setPromptSuggestionsPaused`.

CLI → host subtypes the 311 bundle tests for: `can_use_tool`, `hook_callback`, `mcp_message`, `elicitation`, `request_user_dialog`, `oauth_token_refresh`, `host_auth_token_refresh`, `remote_control_work_secret`, plus system subtypes `init`, `status`, `informational`, `compact_boundary`, `background_tasks_changed`, `commands_changed`, `session_state_changed`, `session_title_changed`, `session_metadata`, `bridge_state`, `hook_started` / `hook_progress` / `hook_response`, `model_refusal_fallback`, `post_turn_summary`, `task_summary`, `slack_tag_*`, `ui_copy`, `ui_prompt_*`, `ui_read_selection`.

`can_use_tool` fields the 311 SDK reads (@2709197): `tool_name`, `input`, `permission_suggestions`, `blocked_path`, `mcp_server`, `decision_reason`, `decision_reason_type`, `decision_reason_code`, `server_policy_ask_only`, `title`, `display_name`, `description`, `default_to_no`, `suppress_always_allow_rule`, `tool_use_id`, `agent_id`, `plugin`, `requires_user_interaction`, `server_prompt`, `computer_folder`, `matched_ask_rule`.

**Only in 325** (absent from the 311 bundle):

- `subtype:"…"` literals: `get_cloud_session`, `get_cloud_session_transcript`, `list_cloud_sessions`, `voice_call_talk`.
- `subtype===` tests: `dev_intent`, `notification`, `plugin_install`, `session_writes_lost`, `voice_call_talk`.
- `reload_plugins` field `refuse_if_busy` (`add_plugins` is in both).

Nothing is only in 311.

## j. Settings and keybindings — CONFIRMED

`contributes.configuration` (23 properties; same keys, types, defaults and enums as 325):

| Setting | Type | Default | Scope / values |
| --- | --- | --- | --- |
| `claudeCode.environmentVariables` | array | `[]` | machine |
| `claudeCode.useTerminal` | boolean | `false` | |
| `claudeCode.allowDangerouslySkipPermissions` | boolean | unset | machine |
| `claudeCode.claudeProcessWrapper` | string | unset | machine |
| `claudeCode.respectGitIgnore` | boolean | `true` | |
| `claudeCode.initialPermissionMode` | string | unset | machine; `default`, `manual`, `acceptEdits`, `plan`, `bypassPermissions` |
| `claudeCode.disableLoginPrompt` | boolean | `false` | |
| `claudeCode.autosave` | boolean | `true` | |
| `claudeCode.focusView` | boolean | `false` | application |
| `claudeCode.useCtrlEnterToSend` | boolean | `false` | |
| `claudeCode.preferredLocation` | string | `panel` | `sidebar`, `panel` |
| `claudeCode.lockEditorGroups` | boolean | `true` | |
| `claudeCode.enableNewConversationShortcut` | boolean | `false` | |
| `claudeCode.enableReopenClosedSessionShortcut` | boolean | `true` | |
| `claudeCode.hideOnboarding` | boolean | `false` | |
| `claudeCode.attachOpenFile` | boolean | `true` | application |
| `claudeCode.attachSelection` | boolean | `true` | application |
| `claudeCode.continueAfterReload` | boolean | `true` | application |
| `claudeCode.scrollToBottomOnSend` | boolean | `true` | application |
| `claudeCode.showMessageTimestamps` | boolean | `true` | application |
| `claudeCode.spinnerVerbs` | object | `{"mode":"append","verbs":[]}` | |
| `claudeCode.archiveInactiveSessions` | number | `14` | application; `0`, `1`, `2`, `7`, `14` |
| `claudeCode.usePythonEnvironment` | boolean | `true` | |

Keybindings (9, byte-identical to 325):

| Key (mac / win, linux) | Command | When |
| --- | --- | --- |
| `alt+k` | `claude-vscode.insertAtMention` | `editorTextFocus` |
| `cmd+escape` / `ctrl+escape` | `claude-vscode.focus` | `!config.claudeCode.useTerminal && editorTextFocus` |
| `cmd+escape` / `ctrl+escape` | `claude-vscode.blur` | `!config.claudeCode.useTerminal && !editorTextFocus` |
| `cmd+shift+escape` / `ctrl+shift+escape` | `claude-vscode.editor.open` | `!config.claudeCode.useTerminal` |
| `cmd+escape` / `ctrl+escape` | `claude-vscode.terminal.open.keyboard` | `config.claudeCode.useTerminal` |
| `cmd+alt+K` / `ctrl+alt+K` | `claude-code.insertAtMentioned` | `editorTextFocus` |
| `cmd+n` / `ctrl+n` | `claude-vscode.newConversation` | `config.claudeCode.enableNewConversationShortcut && (activeWebviewPanelId == 'claudeVSCodePanel' \|\| (claude-vscode.sideBarActive && !editorFocus && !panelFocus))` |
| `ctrl+alt+f` | `claude-vscode.toggleFocusView` | `(activeWebviewPanelId == 'claudeVSCodePanel' \|\| claude-vscode.sideBarActive) && (isMac \|\| (!(listFocus && listSupportsFind) && !inDebugRepl))` |
| `cmd+shift+t` / `ctrl+shift+t` | `claude-vscode.reopenClosedSession` | `config.claudeCode.enableReopenClosedSessionShortcut && claude-vscode.lastClosedWasSession` |

Differences from 325 elsewhere in `package.json`:

- `claudeCode.continueAfterReload` description: 325 appends "A step in plan mode continues only if the session comes back in plan mode."
- Command `claude-vscode.createWorktree` ("Claude Code: Create Worktree") exists in **311 only** (31 commands vs 30).
- `capabilities.untrustedWorkspaces.supported` is `false` in both; only the description text differs.

---

## Changes needed in the spec for 2.1.311

Line numbers refer to `spec/claude-chat-plugin-spec.md`.

1. **Header (line 9, line 34)**: the reference is extension 2.1.311 embedding SDK 0.3.311, darwin-arm64 VSIX; "Observed" means present in 2.1.311.
2. **Environment table (line 100)**: remove `CLAUDE_CODE_STARTUP_FAILURE_RESULTS`. The 311 extension does not set it. Keep `CLAUDE_CODE_EMIT_STARTUP_TIMING=1`. Add `CLAUDE_CODE_ENABLE_TASKS=0` and `TRACESTATE` (deleted) if the table is meant to mirror the reference.
3. **Startup step 2 (around line 108)**: drop "reference: retry the spawn once if initialize times out and the process died". That retry is 325 only; 311 reports the failure.
4. **`--setting-sources` (line 88) and the trust rule (line 535)**: the 311 reference always passes `user,project,local` and has no per-folder trust posture, no `workspaceTrust` attestation, no `--ignore-persisted-trust`, and no `safe-mode` flag. Keep the untrusted-folder rule as the spec's own requirement, not as observed behaviour.
5. **Prompt suggestions (lines 161, 375)**: the 311 reference passes `promptSuggestions: undefined`, so it does not ask for `prompt_suggestion` messages. Mark the chips as a plugin choice that needs `promptSuggestions: true` in `initialize`, not as reference behaviour.
6. **`request_user_dialog` (line 250)**: the 311 reference declares three kinds: `refusal_fallback_prompt`, `fable_overage_consent_prompt`, `auto_mode_server_fallback`. `send_to_project_remote_control_offer` is 325 only.
7. **Ignored Internal subtypes (line 240)**: `voice_call_talk` and the cloud-session requests (`list_cloud_sessions`, `get_cloud_session`, `get_cloud_session_transcript`) do not exist in 311; remove them or label them 325+.
8. **`session_writes_lost` (lines 617, 644, 693, 711)**: not handled by the 311 extension. Keep it only in the 311 → 325 adoption report, not in the baseline message catalogue.
9. **`reload_plugins`**: `refuse_if_busy` is 325 only; `add_plugins` exists in 311.
10. **IDE MCP server (line 313)**: correct for both versions. The reference's in-process chat server is named `claude-vscode` and exposes only `getDiagnostics` (`mcp__claude-vscode__getDiagnostics`). The 12-tool set including `openDiff` is the WebSocket server for terminal `claude`. In the chat, diffs come from `canUseTool` plus the Edit/Write hooks.
11. **Lockfile (line 568)**: state the real contract: `$CLAUDE_CONFIG_DIR/ide/<port>.lock` (default `~/.claude/ide/`), dir `0700`, file `0600`, JSON `{pid, workspaceFolders, ideName, transport:"ws", runningInWindows, authToken}`, port random 10000–65535 on `127.0.0.1`, header `x-claude-code-ide-authorization`, terminal env `CLAUDE_CODE_SSE_PORT`.
12. **Slash commands**: add the locally handled table from section f. "Only `usage` and `context` are local" is wrong for 311 and 325 alike.
13. **Model picker**: the reference changes model by writing user settings and pushing them, not with `set_model`. If the spec says the reference sends `set_model`, soften it to "available in the SDK".
14. **Agent map (lines 449–500)**: no change needed; 311 matches. Two small corrections that apply to both versions: stored statuses are `working / finished / failed / stopped` (`killed` is a wire value mapped to `stopped`, not a stored status), and the transcript path may include nested `agentRelPath` segments before `agent-<id>.jsonl`. `spawnDepth` in `.meta.json` is unconfirmed from extension code.
15. **Session storage**: no change; record the constants (regex `/[^a-zA-Z0-9]/g`, limit 200, 32-bit `(h<<5)-h+c` hash in base 36) if the spec does not already.
16. **Settings (line 528) and keybindings**: no change. Optionally note the 311-only command `claude-vscode.createWorktree`.
17. **Adoption report (lines 698–716)**: unchanged in content, but it now describes what the plugin gains by moving up from its 311 baseline; `api_error_type`, `credential_expires_at`, `invalid_config_file` and `control_closed` must not be assumed present at 311. None of those four strings appears in either extension bundle, so they remain SDK/binary findings.
