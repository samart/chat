# Claude Agent SDK API surface (target 0.3.311)

**Target:** `@anthropic-ai/claude-agent-sdk` **0.3.311** with Claude Code CLI **2.1.311**.

**Sources:**
- `sdk.d.ts` from the published 0.3.311 tarball, cross-checked against 0.3.325.
- `sdk.mjs` 0.3.311 for the wire behaviour of each method and option.
- Captured CLI 2.1.311 control traffic, for undeclared response shapes. Rows that come from the captures say "observed".

The companion Go file is `types.go` (`package claude`). It names each Go type after its TS source and uses the TS doc first sentence as the comment.

**Version markers.** Everything below is the 0.3.311 surface. Anything that 0.3.325 adds is still listed, and marked **since 0.3.3xx (not in 0.3.311)**. "0.3.312-0.3.325" means the changelog does not name the release. Section 9 collects all of them.

Nothing that exists in 0.3.311 was removed or reshaped by 0.3.325: every change between them is additive. The 0.3.311 form is therefore also a valid subset of 0.3.325.

**Hidden fields.** The published d.ts strips `@internal` members. Roughly 78 blank "holes" in the 0.3.325 declarations show where. The wire therefore carries fields this document cannot list, and `types.go` keeps them in `Raw`.

## Contents
1. Query methods
2. Options
3. Messages (SDKMessage / StdoutMessage)
4. Supporting types (content blocks, stream events, results)
5. Control protocol (both directions, permissions)
6. Hooks
7. MCP, elicitation, user messages, agents, initialize pieces
8. Exported functions
9. Not in 0.3.311
10. Go mapping notes

---

## 1. Query methods

`query({prompt, options})` returns a `Query`. A `Query` is an `AsyncGenerator<SDKMessage>` plus the methods below.

**How control requests are sent.** Every control request is written as `{"type":"control_request","request_id":<random base36, 11-13 chars>,"request":{"subtype":…,…}}`. The reply is `{"type":"control_response","response":{"subtype":"success","request_id":…,"response":{…}}}` or `{…"subtype":"error","error":"…","error_code"?:…}`.

- An error reply rejects the promise.
- A request whose AbortSignal fires is followed by `{"type":"control_cancel_request","request_id":…}`.
- Requests wait until the `initialize` request has been written.

**Columns.**
- **Union?** says whether the subtype is in the 0.3.311 d.ts `SDKControlRequestInner` union, or only exists in sdk.mjs.
- **ack** means the TS method ignores the payload: `{}` or absent. The CLI may still send fields; see 5.1.

| Method | TS signature | Doc (first sentence) | Subtype sent | Request fields (TS param → wire) | Response | Union? |
|---|---|---|---|---|---|---|
| `interrupt` | interrupt(): Promise<SDKControlInterruptResponse \| undefined> | Interrupt the current query execution. | interrupt | (none). The implementation also takes a hidden `{cancelQueued}` → `cancel_queued: true` | `InterruptResponse` {still_queued: string[], cancelled?: string[]}. Returns undefined when `still_queued` is not an array. | yes |
| `setPermissionMode` | setPermissionMode(mode: PermissionMode): Promise<void> | Change the permission mode for the current session. | set_permission_mode | mode → `mode` | ack. The CLI answers `{mode}` (observed). | yes |
| `setMcpPermissionModeOverride` | setMcpPermissionModeOverride(serverName: string, mode: 'default' \| 'auto' \| null): Promise<{warning?: string}> | Pin (or clear, with mode:null) a per-MCP-server permission-mode override. | set_mcp_permission_mode_override | serverName → `serverName` (camelCase on the wire), mode → `mode` | `{warning?: string}` (`response ?? {}`) | **no** (sdk.mjs only) |
| `setModel` | setModel(model?: string): Promise<void> | Change the model used for subsequent responses. | set_model | model → `model` (undefined → omitted; resets to default) | ack | yes |
| `setMaxThinkingTokens` | setMaxThinkingTokens(maxThinkingTokens: number \| null, thinkingDisplay?: 'summarized' \| 'omitted' \| 'highlights' \| null): Promise<void> — **deprecated** | Set the maximum number of thinking tokens the model is allowed to use when generating its response. | set_max_thinking_tokens | maxThinkingTokens → `max_thinking_tokens`, thinkingDisplay → `thinking_display` | ack | yes |
| `applyFlagSettings` | applyFlagSettings(settings: {[K in keyof Settings]?: Settings[K] \| null}): Promise<void> | Merge settings into the flag settings layer. | apply_flag_settings | settings → `settings` (null deletes a key from the flag layer) | ack | yes |
| `updateSettings` | updateSettings(source: 'localSettings' \| 'userSettings', settings: Record<string, unknown>): Promise<void> | Merge settings into a settings FILE through the CLI's own writer — the same path /config uses (canonical store root, gitignore upkeep, hardened write) — and live-apply them. | update_settings | source → `source`, settings → `settings` | ack | yes |
| `initializationResult` | initializationResult(): Promise<SDKControlInitializeResponse> | Get the full initialization result, including supported commands, models, account info, and output style configuration: the first-connect answer, or, on a Query returned by a pre-w | — (no request) | — | The stored answer to the first `initialize` (or the latest `reinitialize`): `InitializeResponse` | n/a |
| `reinitialize` | reinitialize(): Promise<SDKControlInitializeResponse> | Re-send the `initialize` control request to an already-running CLI. | initialize | The full initialize request is rebuilt from options (same fields as the first one, section 7.6) | `InitializeResponse` | yes |
| `supportedCommands` | supportedCommands(): Promise<SlashCommand[]> | Get the list of available skills for the current session. | — (no request) | — | `initialize.commands`, replaced by the latest `system/commands_changed` list | n/a |
| `supportedModels` | supportedModels(): Promise<ModelInfo[]> | Get the list of available models. | — (no request) | — | `initialize.models` | n/a |
| `supportedAgents` | supportedAgents(): Promise<AgentInfo[]> | Get the list of available subagents for the current session. | — (no request) | — | `initialize.agents` | n/a |
| `mcpServerStatus` | mcpServerStatus(): Promise<McpServerStatus[]> | Get the current status of all configured MCP servers. | mcp_status | (none) | `response.mcpServers`: `McpServerStatus[]` (Go: `McpStatusResponse`) | yes |
| `getContextUsage` | getContextUsage(opts?: {detail?: 'summary' \| 'full'}): Promise<SDKControlGetContextUsageResponse> | Get a breakdown of current context window usage by category (system prompt, tools, messages, MCP tools, memory files, etc.). | get_context_usage | opts spread: detail → `detail` | `GetContextUsageResponse` (camelCase fields) | yes |
| `usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET` | usage_EXPERIMENTAL…(opts?: {skipBehaviors?: boolean}): Promise<SDKControlGetUsageResponse> | Get the structured data behind the `/usage` command: session cost and token usage totals plus claude.ai plan rate-limit utilization windows (5-hour, 7-day, per-model) when availabl | get_usage | skipBehaviors → `skip_behaviors: true` (only when true) | `GetUsageResponse` | yes |
| `readFile` | readFile(path: string, options?: {maxBytes?: number; encoding?: 'utf-8' \| 'base64'}): Promise<SDKControlReadFileResponse \| null> | Read a file from the session's filesystem for the remote sidebar viewer. | read_file | path → `path`, maxBytes → `max_bytes`, encoding → `encoding` | `ReadFileResponse` {contents, absPath, truncated?, encoding?}. **Any error resolves to null.** | yes |
| `reloadPlugins` | reloadPlugins(options?: {holdOnCacheImpact?: boolean; addPlugins?: SdkPluginConfig[]}): Promise<SDKControlReloadPluginsResponse> | Reload plugins from disk and return the refreshed commands, agents, plugins, and MCP server status. | reload_plugins | holdOnCacheImpact → `hold_on_cache_impact: true` (only when true); addPlugins → `add_plugins` (only when non-empty). 0.3.325 adds refuseIfBusy → `refuse_if_busy: true` (**since 0.3.325**) | `ReloadPluginsResponse` | yes |
| `reloadSkills` | reloadSkills(): Promise<SDKControlReloadSkillsResponse> | Reload skills from disk and return the refreshed skill list. | reload_skills | (none) | `ReloadSkillsResponse` {skills: SlashCommand[]} | yes |
| `reloadOutputStyles` | reloadOutputStyles(): Promise<SDKControlReloadOutputStylesResponse> | Re-read the output-style directories from disk and return the refreshed style names. | reload_output_styles | (none) | `ReloadOutputStylesResponse` {available_output_styles} | yes |
| `accountInfo` | accountInfo(): Promise<AccountInfo> | Get information about the authenticated account. | — (no request) | — | `initialize.account` | n/a |
| `rewindFiles` | rewindFiles(userMessageId: string, options?: {dryRun?: boolean}): Promise<RewindFilesResult> | Rewind tracked files to their state at a specific user message. | rewind_files | userMessageId → `user_message_id`, dryRun → `dry_run` | `RewindFilesResult` {canRewind, error?, filesChanged?, insertions?, deletions?, skippedLinks?} | yes |
| `seedReadState` | seedReadState(path: string, mtime: number): Promise<void> | Seed the CLI's readFileState cache with a path+mtime entry. | seed_read_state | path → `path`, mtime → `mtime` | ack | yes |
| `reconnectMcpServer` | reconnectMcpServer(serverName: string): Promise<void> | Reconnect an MCP server by name. | mcp_reconnect | serverName → `serverName` (camelCase) | ack | yes |
| `toggleMcpServer` | toggleMcpServer(serverName: string, enabled: boolean): Promise<void> | Enable or disable an MCP server by name. | mcp_toggle | serverName → `serverName`, enabled → `enabled` | ack | yes |
| `readMcpResource` | readMcpResource(serverName: string, uri: string): Promise<SDKControlMcpReadResourceResponse> — *alpha* | Read one MCP Apps (SEP-1865) UI resource from a connected MCP server the CLI itself dialed, so the host can render a tool's widget. | mcp_read_resource | serverName → `serverName`, uri → `uri` | `McpReadResourceResponse` {contents: {uri, mimeType?, text?, blob?, _meta?}[]} | yes |
| `setMcpServers` | setMcpServers(servers: Record<string, McpServerConfig>): Promise<McpSetServersResult> | Dynamically set the MCP servers for this session. | mcp_set_servers | servers → `servers`. Before sending, the SDK connects in-process (`type:"sdk"` with `instance`) servers itself, disconnects removed ones, and replaces each with `{type:"sdk", name, timeout?, tools?}` | `McpSetServersResult` {added, removed, errors} | yes |
| `streamInput` | streamInput(stream: AsyncIterable<SDKUserMessage>): Promise<void> | Stream input messages to the query. | — (no control request) | Writes each SDKUserMessage as one stdin JSON line (adds `client_composed: true` when `verbatimPrompts`). Afterwards it closes stdin, or first waits for the run to end when hooks, canUseTool or SDK MCP servers need the channel. | — | n/a |
| `stopTask` | stopTask(taskId: string): Promise<void> | Stop a running task. | stop_task | taskId → `task_id` | ack | yes |
| `backgroundTasks` | backgroundTasks(toolUseId?: string): Promise<boolean> | Background in-flight foreground tasks (Bash commands and subagents). | background_tasks | toolUseId → `tool_use_id` | `response.backgrounded ?? true` (Go: `BackgroundTasksResponse`) | yes |
| `close` | close(): void | Close the query and terminate the underlying process. | — (no request) | — | Cleanup: aborts in-flight host handlers, ends stdin, SIGTERM after 2 s if still alive, SIGKILL 5 s later | n/a |


There are **30 methods** (counting `close`, `streamInput` and the four cached accessors). The method set and the request field names are identical in 0.3.311 and 0.3.325, except that `reloadPlugins` gains `refuseIfBusy`.

### 1.1 Query methods present in sdk.mjs but stripped from the d.ts (0.3.311 and 0.3.325)

None of these subtypes is in the d.ts control-request union. They are listed so a Go client can mirror the wire if it needs to.

| Method (internal) | Subtype | Request fields |
|---|---|---|
| `getSettings()` | get_settings | —; response observed as `{effective, sources, applied}` (Go `GetSettingsResponse`). The subtype *is* in the union; only the method is hidden. |
| `getHooksListing()` | get_hooks_listing | — (in union) |
| `getTaskOutput(taskId)` | get_task_output | `task_id` (in union) |
| `listPermissionRules()` | list_permission_rules | — (in union) |
| `cancelAsyncMessage(uuid)` | cancel_async_message | `message_uuid` (in union); returns `response.cancelled` |
| `setCwd(path, {trustAccepted?, trustedDirectory?})` | set_cwd | `path`, `trust_accepted`, `trusted_directory` |
| `claimSession(p)` | claim_session | spread p + `include_initialize` (default true) |
| `enableRemoteControl(enabled, name?, opts?)` | remote_control | `enabled`, `name`, `reattach_session_id`, `keep_session_on_exit`, `work_secret` |
| `submitFeedback(description, opts?)` | submit_feedback | `description`, `surface`, `draft_id`, `type`, `title`, `area`, `attach_transcript`, `save_locally` |
| `submitSurveyFeedback(p)` | submit_survey_feedback | spread p |
| `renameSession(title, sessionId?)` | rename_session | `title`, `source:"host"`, `session_id` (in union) |
| `generateSessionTitle(description, {persist?})` | generate_session_title | `description`, `persist` → returns `response.title` |
| `askSideQuestion(question, {history?, signal?})` | side_question | `question`, `history`; progress arrives as `system/control_request_progress` |
| `launchUltrareview(args, {confirm?})` | ultrareview_launch | `args`, `confirm` |
| `messageRated(p)` | message_rated | `messageUuid`, `sentiment`, `surface`, `cleared` |
| `setChromeBrowserHints(p)` | set_chrome_browser_hints | spread p |
| `setPromptSuggestionsPaused(paused)` | set_prompt_suggestions_paused | `paused` |
| `enableChannel(serverName)` | channel_enable | `serverName` |
| `mcpAuthenticate(serverName, redirectUri, {redirectUriOnly?})` | mcp_authenticate | `serverName`, `redirectUri`, `redirectUriOnly` |
| `mcpClearAuth(serverName)` | mcp_clear_auth | `serverName` |
| `mcpSubmitOAuthCallbackUrl(serverName, callbackUrl)` | mcp_oauth_callback_url | `serverName`, `callbackUrl` |
| `claudeAuthenticate(loginWithClaudeAi, {consoleProfile?})` | claude_authenticate | `loginWithClaudeAi`, `consoleProfile` |
| `claudeOAuthCallback(code, state)` | claude_oauth_callback | `authorizationCode`, `state` |
| `claudeOAuthWaitForCompletion()` | claude_oauth_wait_for_completion | — |
| `getMemoryDialog() / getSkillsDialog() / getStatus() / exportConversation() / getChromeDialog() / getChromeBrowsers() / getSandboxDialog() / getPlan()` | get_memory_dialog / get_skills_dialog / get_status / export_conversation / get_chrome_dialog / get_chrome_browsers / get_sandbox_dialog / get_plan | — |
| `selectChromeBrowser(deviceId)` | select_chrome_browser | `device_id` |


## 2. Options

`Options` has **69 fields** in 0.3.311; the set is unchanged in 0.3.325.

**Categories.**
- `argv`: becomes a CLI flag.
- `initialize`: sent in the `initialize` control request.
- `env`: becomes a child-process environment variable.
- `host-only callback`: a function, or store, that runs in the host.
- `transport`: affects how the process is spawned or how stdin is written.

**Fixed flags.** Every launch gets `--output-format stream-json --verbose --input-format stream-json`. Flags with values use the `--flag=value` form, except where noted.

**In Go.** `Options` is a data-only struct. The function-valued options `canUseTool`, `onElicitation`, `onUserDialog` and `stderr` are booleans. `hooks` holds callback ids, `HookCallbackMatcher.HookCallbackIDs`. `abortController`, `sessionStore` and `spawnClaudeCodeProcess` are omitted.

| Field | Type | Doc (first sentence) | Default | Category | Wire mapping (sdk.mjs 0.3.311) |
|---|---|---|---|---|---|
| `abortController?` | `AbortController` | Controller for cancelling the query. |  | transport | Aborting kills the process. In Go: context.Context. |
| `additionalDirectories?` | `string[]` | Additional directories Claude can access beyond the current working directory. |  | argv | `--add-dir <dir>` per entry |
| `projectConfigRoot?` | `string` | The trusted checkout `cwd` is a worktree of. |  | argv | `--project-config-root` |
| `agent?` | `string` | Agent name for the main thread. |  | argv | `--agent` |
| `agents?` | `Record<string, AgentDefinition>` | Programmatically define custom subagents that can be invoked via the Agent tool. |  | initialize | `initialize.agents` |
| `allowedTools?` | `string[]` | List of tool names that are auto-allowed without prompting for permission. |  | argv | `--allowedTools a,b` (plus `Skill(...)` entries derived from `skills`) |
| `canUseTool?` | `CanUseTool` | Custom permission handler for controlling tool usage. |  | host-only callback | Adds `--permission-prompt-tool=stdio`; answers `can_use_tool`. Throws if combined with permissionPromptToolName. |
| `continue?` | `boolean` | Continue the most recent conversation in the current directory instead of starting a new one. |  | argv | `--continue` (with sessionStore: resolves the latest session via the store instead) |
| `cwd?` | `string` | Current working directory for the session. | `process.cwd()` | transport | Spawn cwd |
| `disallowedTools?` | `string[]` | List of tool names that are disallowed. |  | argv | `--disallowedTools a,b` |
| `toolAliases?` | `Record<string, string>` | Map of tool-name aliases applied before name resolution. |  | initialize | `initialize.toolAliases` |
| `tools?` | `string[] \| { type: 'preset'; preset: 'claude_code'; }` | Specify the base set of available built-in tools. - `string[]` - Array of specific tool names (e.g., `['Bash', 'Read', 'Edit']`) - `[]` (empty array) - Disable all built-in tools - `{ type: 'preset';  |  | argv | `--tools a,b`; the preset object → `--tools=default` |
| `env?` | `{ [envVar: string]: string \| undefined; }` | Environment variables for the Claude Code process. | `process.env` | env | The child environment (default: `process.env`). The SDK always sets `CLAUDE_CODE_ENTRYPOINT=sdk-ts` and `CLAUDE_AGENT_SDK_VERSION` if missing, and `CLAUDE_CODE_SDK_READS_SESSION_STATE=1`. It deletes `NODE_OPTIONS`, and `DEBUG` unless `DEBUG_CLAUDE_AGENT_SDK`. |
| `executable?` | `'bun' \| 'deno' \| 'node'` | JavaScript runtime to use for executing Claude Code. | `'bun'` under Bun, else `'node'` | transport | Runtime used when the executable path is a .js file (default bun if running under bun, else node) |
| `executableArgs?` | `string[]` | Additional arguments to pass to the JavaScript runtime executable. |  | transport | Prepended runtime args |
| `extraArgs?` | `Record<string, string \| null>` | Additional CLI arguments to pass to Claude Code. |  | argv | `--<key>=<value>`, or `--<key>` for null |
| `fallbackModel?` | `string` | Fallback model(s) to use if the primary model is overloaded or unavailable. |  | argv | `--fallback-model` (throws if equal to model) |
| `enableFileCheckpointing?` | `boolean` | Enable file checkpointing to track file changes during the session. |  | env | `CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true` |
| `toolConfig?` | `ToolConfig` | Per-tool configuration for built-in tools. |  | env | `CLAUDE_CODE_QUESTION_PREVIEW_FORMAT` (+ `CLAUDE_CODE_QUESTION_EXTENDED`, `…_OPTIONAL_DESCRIPTIONS` for hidden sub-keys) |
| `forkSession?` | `boolean` | When true, resumed sessions will fork to a new session ID rather than continuing the previous session. |  | argv | `--fork-session` |
| `betas?` | `SdkBeta[]` | Enable beta features. |  | argv | `--betas a,b` |
| `hooks?` | `Partial<Record<HookEvent, HookCallbackMatcher[]>>` | Hook callbacks for responding to various events during execution. |  | host-only callback | Callbacks get ids `hook_<n>`. `initialize.hooks` = `{Event: [{matcher, hookCallbackIds, timeout}]}`. The CLI calls back with `hook_callback`. |
| `onElicitation?` | `OnElicitation` | Callback for handling MCP elicitation requests. |  | host-only callback | Answers `elicitation` (default when absent: `{action:"decline"}`) |
| `onUserDialog?` | `OnUserDialog` | Callback for handling `request_user_dialog` control requests — blocking dialogs the CLI asks the host to render. |  | host-only callback | Answers `request_user_dialog` (when absent the request is left unanswered) |
| `supportedDialogKinds?` | `string[]` | Dialog kinds this consumer's `onUserDialog` can actually render (`request_user_dialog` `dialog_kind` values, e.g. 'refusal_fallback_prompt'). |  | initialize | `initialize.supportedDialogKinds` (validated against onUserDialog) |
| `perTaskStopAffordance?` | `boolean` | Declares that this consumer renders a per-task stop control wired to the `stop_task` control request, so the user can stop an individual background task. |  | initialize | `initialize.perTaskStopAffordance` |
| `persistSession?` | `boolean` | When false, disables session persistence to disk. | true | argv | false → `--no-session-persistence` |
| `sessionStore?` | `SessionStore` | Mirror session transcripts to an external store. *(alpha)* |  | host-only callback | Store methods. Adds `--session-mirror`; the CLI emits transcript_mirror frames that the SDK appends to the store; resume materialises the store into a temp CLAUDE_CONFIG_DIR |
| `sessionStoreFlush?` | `SessionStoreFlush` | Controls how aggressively transcript entries are flushed to . *(alpha)* | `'batched'` | host-only callback | Batching of sessionStore.append |
| `loadTimeoutMs?` | `number` | Timeout for each `sessionStore.load()` / `sessionStore.listSubkeys()` call during resume materialization. *(alpha)* | 60_000 | host-only callback | Timeout for store load/listSubkeys on resume |
| `includeHookEvents?` | `boolean` | Include hook lifecycle events in the output stream. | false | argv | `--include-hook-events` |
| `includePartialMessages?` | `boolean` | Include partial/streaming message events in the output. |  | argv | `--include-partial-messages` |
| `forwardSubagentText?` | `boolean` | Forward subagent text and thinking blocks as assistant/user messages with `parent_tool_use_id` set. |  | initialize | `initialize.forwardSubagentText` |
| `verbatimPrompts?` | `boolean` | Send every user message with `client_composed: true`, so the CLI delivers the prompt text as written: no `@path` file-mention expansion and no slash-command dispatch. | false | transport | Stamps `client_composed: true` on every user message written to stdin |
| `thinking?` | `ThinkingConfig` | Controls Claude's thinking/reasoning behavior. - `{ type: 'adaptive' }` — Claude decides when and how much to think (Opus 4.6+). |  | argv | adaptive → `--thinking=adaptive`; enabled → `--max-thinking-tokens=N` (no budget → adaptive); disabled → `--thinking=disabled`; display → `--thinking-display` |
| `effort?` | `EffortLevel` | Controls how much effort Claude puts into its response. |  | argv | `--effort` |
| `maxThinkingTokens?` | `number` | **Deprecated.** Maximum number of tokens the model can use for its thinking/reasoning process. |  | argv | When thinking is unset: 0 → disabled, else enabled with budget |
| `maxTurns?` | `number` | Maximum number of conversation turns before the query stops. |  | argv | `--max-turns` |
| `maxBudgetUsd?` | `number` | Maximum budget in USD for the query. |  | argv | `--max-budget-usd` |
| `taskBudget?` | `{ total: number; }` | API-side task budget in tokens. *(alpha)* |  | argv | `--task-budget <total>` |
| `mcpServers?` | `Record<string, McpServerConfig>` | MCP (Model Context Protocol) server configurations. |  | argv | Non-SDK servers → `--mcp-config {"mcpServers":{…}}`. `type:"sdk"` servers with an instance stay in process: their names go in `initialize.sdkMcpServers` and their timeout/tools in `initialize.sdkMcpServerConfigs`. Initialize also carries `sdkMcpServerManifests` (captured initialize + tools/list). |
| `model?` | `string` | Claude model to use. | CLI default | argv | `--model` |
| `outputFormat?` | `OutputFormat` | Output format configuration for structured responses. |  | argv | `--json-schema <json>` and `initialize.jsonSchema` |
| `pathToClaudeCodeExecutable?` | `string` | Path to the Claude Code executable. |  | transport | Executable to spawn (default: the bundled native binary for platform/arch) |
| `permissionMode?` | `PermissionMode` | Permission mode for the session. |  | argv | `--permission-mode` |
| `planModeInstructions?` | `string` | Custom workflow instructions for plan mode. |  | initialize | `initialize.planModeInstructions` |
| `allowDangerouslySkipPermissions?` | `boolean` | Must be set to `true` when using `permissionMode: 'bypassPermissions'`. | false | argv | `--allow-dangerously-skip-permissions` |
| `permissionPromptToolName?` | `string` | MCP tool name to use for permission prompts. |  | argv | `--permission-prompt-tool <name>` |
| `permissionPrompts?` | `'host' \| 'none'` | Who answers permission prompts. | `'host'` | argv | `--permission-prompts host\|none` |
| `plugins?` | `SdkPluginConfig[]` | Load plugins for this session. |  | argv | pluginDelivery 'argv': `--plugin-dir <path>` / `--plugin-dir-no-mcp <path>`. 'initialize': `--await-initialize` + `initialize.plugins` |
| `pluginDelivery?` | `'argv' \| 'initialize'` | How `plugins` reach the Claude Code process. - `'argv'` (default) - One `--plugin-dir <path>` flag per plugin. | `'argv'` | argv | Chooses how plugins travel (see plugins) |
| `promptSuggestions?` | `boolean` | Enable prompt suggestions. |  | initialize | `initialize.promptSuggestions` |
| `agentProgressSummaries?` | `boolean` | Enable periodic AI-generated progress summaries for running subagents. | false | initialize | `initialize.agentProgressSummaries` |
| `resume?` | `string` | Session ID to resume. |  | argv | `--resume <id>` |
| `sessionId?` | `string` | Use a specific session ID for the conversation instead of an auto-generated one. |  | argv | `--session-id` |
| `resumeSessionAt?` | `string` | When resuming, only resume messages up to and including the message with this UUID. |  | argv | `--resume-session-at` |
| `resumeDropsTurn?` | `string` | With `resumeSessionAt`: declares the prompt UUID of the turn this truncating resume intends to discard. |  | argv | `--resume-drops-turn` |
| `sandbox?` | `SandboxSettings` | Sandbox settings for command execution isolation. | `true` unless this option or an inline `settings` sandbox bl | argv | Merged into the `settings` object as `settings.sandbox` and sent with `--settings`. `enabled:true` defaults `failIfUnavailable:true`. Throws if `settings` is a file path |
| `settings?` | `string \| Settings` | Additional settings to apply. |  | argv | `--settings <path\|json>` |
| `managedSettings?` | `Settings` | Policy-tier settings supplied by the spawning parent process. |  | argv | `--managed-settings <json>` |
| `settingSources?` | `SettingSource[]` | Control which filesystem settings to load. - `'user'` - Global user settings (`~/.claude/settings.json`) - `'project'` - Project settings (`.claude/settings.json`) - `'local'` - Local settings (`.clau | all sources | argv | `--setting-sources a,b` |
| `skills?` | `string[] \| 'all'` | Skills to enable for the main session. |  | argv | Adds `Skill` ('all') or `Skill(name)` entries to `--allowedTools`; list also → `initialize.skills` |
| `debug?` | `boolean` | Enable debug mode for the Claude Code process. |  | argv | `--debug` |
| `debugFile?` | `string` | Write debug logs to a specific file path. |  | argv | `--debug-file` |
| `stderr?` | `(data: string) => void` | Callback for stderr output from the Claude Code process. |  | host-only callback | Receives child stderr chunks |
| `strictMcpConfig?` | `boolean` | Only use MCP servers passed via the `mcpServers` option (and servers declared by explicitly-passed agent definitions in `agents`), ignoring all other MCP configurations: project `.mcp.json`, user sett |  | argv | `--strict-mcp-config` |
| `systemPrompt?` | `string \| string[] \| { type: 'custom'; prompt: string \| string[]; snapshot?: boolean; } \| { type: 'preset'; preset: 'clau…` | System prompt configuration. - `string` - Use a custom system prompt - `string[]` - Use a custom system prompt as an array of blocks; include `SYSTEM_PROMPT_DYNAMIC_BOUNDARY` as a standalone element t |  | initialize | string → `initialize.systemPrompt:[s]`; string[] → as is; custom → `systemPrompt` + `systemPromptSnapshot`; preset → `appendSystemPrompt`, `excludeDynamicSections`, `systemPromptSnapshot`; omitted → `""` |
| `title?` | `string` | Custom title for a new session. |  | initialize | `initialize.title` |
| `spawnClaudeCodeProcess?` | `(options: SpawnOptions) => SpawnedProcess` | Custom function to spawn the Claude Code process. |  | host-only callback | Replaces the local spawn (receives {command,args,cwd,env,signal}) |


**Not in the d.ts but read by the 0.3.311 `query()`.** These are internal options: `appendSubagentSystemPrompt`, `codeRunningMcpTools`, `webSearchIsolationExemptMcpServers`, `rapidFollowupPreempt`, `forwardNestedSubagentToolCalls` and `workspaceTrust`, all → initialize; `channels` → `--channels`; `workload` → `--workload`; `getOAuthToken` and `getHostAuthToken`, host callbacks for the `oauth_token_refresh` and `host_auth_token_refresh` requests, which also set `CLAUDE_CODE_SDK_HAS_OAUTH_REFRESH=1` and `CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH=1`; and `onProcessSpawned`.

**First message.** A string `prompt` is written as `{"type":"user","session_id":"","message":{"role":"user","content":[{"type":"text","text":…}]},"parent_tool_use_id":null}`. Stdin then closes after the first result, which makes it a single-turn query. An `AsyncIterable` prompt goes through `streamInput`.

---

## 3. Messages

`SDKMessage` is the union the `Query` generator yields. `StdoutMessage` (not exported) is everything the CLI writes to stdout in stream-json mode, one JSON object per line. It is `SDKMessage` plus `SDKActiveGoalMessage` (`active_goal`), `SDKControlResponse`, `SDKControlRequest`, `SDKControlCancelRequest` and `SDKKeepAliveMessage` (`keep_alive`).

In Go:
- `StdoutMessage` and `Message` are unions keyed on `type` and `subtype`.
- `user` and `user`+`isReplay:true` share a discriminator. `Decode` picks `UserMessageReplay` when `isReplay` is true.
- `StdoutMessage.SDKMessage()` narrows a `StdoutMessage` to a `Message`.

| type:subtype | TS type | Go type | Union | Doc |
|---|---|---|---|---|
| `assistant` | SDKAssistantMessage | AssistantMessage | SDKMessage | An assistant message. |
| `user` | SDKUserMessage | UserMessage | SDKMessage | A user-role message. |
| `user (isReplay: true)` | SDKUserMessageReplay | UserMessageReplay | SDKMessage |  |
| `result:success` | SDKResultSuccess | ResultSuccess | SDKMessage |  |
| `result:error_during_execution\|error_max_turns\|error_max_budget_usd\|error_max_structured_output_retries` | SDKResultError | ResultError | SDKMessage |  |
| `system:init` | SDKSystemMessage | SystemInitMessage | SDKMessage | Session metadata the CLI emits at the start of each turn, normally ahead of every other message of that turn: session_id, model, working… |
| `stream_event` | SDKPartialAssistantMessage | PartialAssistantMessage | SDKMessage | An incremental streaming event for the assistant message being generated, emitted only when partial messages are requested… |
| `system:compact_boundary` | SDKCompactBoundaryMessage | CompactBoundaryMessage | SDKMessage |  |
| `system:status` | SDKStatusMessage | StatusMessage | SDKMessage |  |
| `system:api_retry` | SDKAPIRetryMessage | APIRetryMessage | SDKMessage | Emitted when an API request fails with a retryable error and will be retried after a delay. error_status is null for connection errors… |
| `system:control_request_progress` | SDKControlRequestProgressMessage | RequestProgressMessage | SDKMessage | Progress for a long-running client-originated control_request (currently only side_question), correlated by request_id. status 'started'… |
| `system:model_refusal_fallback` | SDKModelRefusalFallbackMessage | ModelRefusalFallbackMessage | SDKMessage | Emitted when the primary model ends the stream with stop_reason "refusal" and the turn is retried once on a fallback model (direction:… |
| `system:model_refusal_no_fallback` | SDKModelRefusalNoFallbackMessage | ModelRefusalNoFallbackMessage | SDKMessage | Emitted when the model ends the stream with stop_reason "refusal" and no retry runs: no fallback model is configured, or per-category… |
| `system:local_command_output` | SDKLocalCommandOutputMessage | LocalCommandOutputMessage | SDKMessage | Output from a local slash command (e.g. /voice, /usage). |
| `system:hook_started` | SDKHookStartedMessage | HookStartedMessage | SDKMessage |  |
| `system:hook_progress` | SDKHookProgressMessage | HookProgressMessage | SDKMessage |  |
| `system:hook_response` | SDKHookResponseMessage | HookResponseMessage | SDKMessage |  |
| `system:plugin_install` | SDKPluginInstallMessage | PluginInstallMessage | SDKMessage | Headless plugin installation progress (CLAUDE_CODE_SYNC_PLUGIN_INSTALL). started/completed bracket the whole install; installed/failed… |
| `tool_progress` | SDKToolProgressMessage | ToolProgressMessage | SDKMessage |  |
| `auth_status` | SDKAuthStatusMessage | AuthStatusMessage | SDKMessage |  |
| `system:task_notification` | SDKTaskNotificationMessage | TaskNotificationMessage | SDKMessage |  |
| `system:task_started` | SDKTaskStartedMessage | TaskStartedMessage | SDKMessage |  |
| `system:task_updated` | SDKTaskUpdatedMessage | TaskUpdatedMessage | SDKMessage |  |
| `system:task_progress` | SDKTaskProgressMessage | TaskProgressMessage | SDKMessage |  |
| `system:background_tasks_changed` | SDKBackgroundTasksChangedMessage | BackgroundTasksChangedMessage | SDKMessage | The full set of live background tasks, emitted whenever membership changes (start, completion, kill, a foreground agent being… |
| `system:thinking_tokens` | SDKThinkingTokensMessage | ThinkingTokensMessage | SDKMessage | Live thinking-token estimate, digested from thinking_delta.estimated_tokens during the redacted-thinking phase (where the API otherwise… |
| `system:session_state_changed` | SDKSessionStateChangedMessage | SessionStateChangedMessage | SDKMessage | Mirrors notifySessionStateChanged. |
| `system:worker_shutting_down` | SDKWorkerShuttingDownMessage | WorkerShuttingDownMessage | SDKMessage | Emitted by the bridge on opt-in graceful worker teardown (only when the teardown caller supplied a reason), before the heartbeat stops,… |
| `system:commands_changed` | SDKCommandsChangedMessage | CommandsChangedMessage | SDKMessage | Fire-and-forget push of the full slash-command list after a mid-session change (e.g. skills discovered dynamically as the agent works in… |
| `system:notification` | SDKNotificationMessage | NotificationMessage | SDKMessage | Loop-side text notification. |
| `system:files_persisted` | SDKFilesPersistedEvent | FilesPersistedEvent | SDKMessage |  |
| `tool_use_summary` | SDKToolUseSummaryMessage | ToolUseSummaryMessage | SDKMessage |  |
| `system:memory_recall` | SDKMemoryRecallMessage | MemoryRecallMessage | SDKMessage | Emitted when the memory recall supervisor surfaces relevant memories into the turn. |
| `rate_limit_event` | SDKRateLimitEvent | RateLimitEvent | SDKMessage | Rate limit event emitted when rate limit info changes. |
| `system:elicitation_complete` | SDKElicitationCompleteMessage | ElicitationCompleteMessage | SDKMessage | Emitted when an MCP server confirms that a URL-mode elicitation is complete. |
| `system:permission_denied` | SDKPermissionDeniedMessage | PermissionDeniedMessage | SDKMessage | Emitted when a tool call is auto-denied without an interactive permission prompt (e.g. auto-mode classifier, dontAsk mode,… |
| `prompt_suggestion` | SDKPromptSuggestionMessage | PromptSuggestionMessage | SDKMessage | Predicted next user prompt, emitted after each turn when promptSuggestions is enabled. |
| `system:mirror_error` | SDKMirrorErrorMessage | MirrorErrorMessage | SDKMessage | Emitted when SessionStore.append() rejects or times out for a transcript-mirror batch after bounded retry (3 attempts with short… |
| `system:informational` | SDKInformationalMessage | InformationalMessage | SDKMessage | Generic text banner emitted by the loop — non-error status lines, hook feedback (e.g. a UserPromptSubmit hook's block reason),… |
| `conversation_reset` | SDKConversationResetMessage | ConversationResetMessage | SDKMessage | Emitted by /clear, plan-mode exit, fresh-session, and onboarding flows. |
| `system:session_writes_lost` | SDKSessionWritesLostMessage | SessionWritesLostMessage | SDKMessage — **since 0.3.317 (not in 0.3.311)** | The CLI could not save lines of a session transcript, changes to ~/.claude.json, or both, because the disk was full or its quota was… |
| `active_goal` | SDKActiveGoalMessage | ActiveGoalMessage | StdoutMessage only | Emitted when the user's /goal Stop hook reports met (clears) or not-yet-met (bumps iterations + last_reason). |
| `control_request` | SDKControlRequest | ControlRequest | StdoutMessage only | Envelope for a control-protocol request, sent by either side on the same stream as the messages. |
| `control_response` | SDKControlResponse | ControlResponse | StdoutMessage only | Envelope for the single reply to a control_request, sent by whichever side received the request. |
| `control_cancel_request` | SDKControlCancelRequest | ControlCancelRequest | StdoutMessage only | Tells the other side that the sender no longer needs the answer to one of its own in-flight control_requests (for example a pending… |
| `keep_alive` | SDKKeepAliveMessage | KeepAliveMessage | StdoutMessage only | Liveness heartbeat. |


**Counts:** 40 SDKMessage variants in 0.3.311, 41 in 0.3.325; StdoutMessage adds 5 more. All four result error subtypes share `SDKResultError`.

### assistant — `SDKAssistantMessage` → Go `AssistantMessage`

An assistant message.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'assistant'` | req |  |
| `message` | `BetaMessage` | req | Shaped like an Anthropic Messages API Message object (role "assistant"): id, model, content blocks (text, thinking, tool_use, ...), stop_reason… |
| `parent_tool_use_id` | `string \| null` | req |  |
| `error` | `SDKAssistantMessageError` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `request_id` | `string` | opt |  |
| `user_message_uuid` | `string` | opt | Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on an assistant message each time that send changes —… |
| `user_message_uuids` | `string[]` | opt | Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host… |
| `resume_reason` | `string` | opt | Why this frame's turn continues one that a worker restart interrupted. |
| `resumed_from_incomplete_thinking` | `true` | opt | This turn continued the preceding truncated assistant turn inside its trailing signed thinking block (max-output-tokens recovery). |
| `supersedes` | `UUID[]` | opt | Wire uuids of previously-delivered messages that this message replaces (refusal-fallback supersede). |
| `aborted` | `true` | opt | True when this assistant message was truncated by an interrupt/abort before the stream completed: stop_reason was never received and the content… |
| `agent_id` | `string` | opt | ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed. |
| `subagent_type` | `string` | opt | Subagent type that produced this message. |
| `task_description` | `string` | opt | Description of the subagent task that produced this message. |
| `timestamp` | `string` | opt | ISO timestamp of when this content block finished on the originating process. |
| `context_usage` | `SDKContextUsage` | opt | Structured twin of the /context report, carried on the synthetic assistant message that delivers the markdown table. |
| `usage_report` | `SDKUsageReport` | opt | Structured twin of the /usage report, carried on the synthetic assistant message that delivers its text: the session totals, the plan's usage rows… |

### user — `SDKUserMessage` → Go `UserMessage`

A user-role message.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'user'` | req |  |
| `message` | `MessageParam` | req | An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image,… |
| `parent_tool_use_id` | `string \| null` | req |  |
| `isSynthetic` | `boolean` | opt |  |
| `tool_use_result` | `unknown` | opt | Structured tool output — the tool's full Output object, not the string content sent to the model. |
| `priority` | `'now' \| 'next' \| 'later'` | opt |  |
| `origin` | `SDKMessageOrigin` | opt |  |
| `shouldQuery` | `boolean` | opt | When false, the message is appended to the transcript without triggering an assistant turn. |
| `timestamp` | `string` | opt | ISO timestamp when the message was created on the originating process. |
| `client_composed` | `true` | opt | The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and… |
| `uuid` | `UUID` | opt |  |
| `pasted_content` | `MessageParam['content'][]` | opt | Content the user pasted into the prompt rather than typed: each entry a string or an array of content blocks. |
| `inline_pastes` | `string[]` | opt | Text the user pasted that is still in `message.content` where they put it: each entry one paste. |
| `session_id` | `string` | opt |  |
| `agent_id` | `string` | opt | ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed. |
| `subagent_type` | `string` | opt | Subagent type that produced this message. |
| `task_description` | `string` | opt | Description of the subagent task that produced this message. |

### user (isReplay: true) — `SDKUserMessageReplay` → Go `UserMessageReplay`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'user'` | req |  |
| `message` | `MessageParam` | req | An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image,… |
| `parent_tool_use_id` | `string \| null` | req |  |
| `isSynthetic` | `boolean` | opt |  |
| `tool_use_result` | `unknown` | opt | Structured tool output — the tool's full Output object, not the string content sent to the model. |
| `priority` | `'now' \| 'next' \| 'later'` | opt |  |
| `origin` | `SDKMessageOrigin` | opt |  |
| `shouldQuery` | `boolean` | opt | When false, the message is appended to the transcript without triggering an assistant turn. |
| `timestamp` | `string` | opt | ISO timestamp when the message was created on the originating process. |
| `client_composed` | `true` | opt | The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and… |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `isReplay` | `true` | req |  |
| `file_attachments` | `unknown[]` | opt |  |

### result:success — `SDKResultSuccess` → Go `ResultSuccess`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'result'` | req |  |
| `subtype` | `'success'` | req |  |
| `duration_ms` | `number` | req |  |
| `duration_api_ms` | `number` | req |  |
| `ttft_ms` | `number` | opt |  |
| `ttft_stream_ms` | `number` | opt |  |
| `time_to_request_ms` | `number` | opt |  |
| `user_message_uuid` | `string` | opt |  |
| `user_message_uuids` | `string[]` | opt |  |
| `resume_reason` | `string` | opt |  |
| `local_command` | `string` | opt |  |
| `request_sent_wall_ms` | `number` | opt |  |
| `first_content_frame_ms` | `number` | opt |  |
| `first_stream_post_ms` | `number` | opt |  |
| `first_stream_post_ack_ms` | `number` | opt |  |
| `first_stream_post_queue_wait_ms` | `number` | opt |  |
| `first_stream_post_queued_behind` | `'durable_post' \| 'ephemeral_post' \| 'retry_backoff' \| 'hold' \| 'none'` | opt |  |
| `first_stream_post_wall_ms` | `number` | opt |  |
| `first_text_post_ms` | `number` | opt |  |
| `first_text_post_queue_wait_ms` | `number` | opt |  |
| `first_text_post_queued_behind` | `'durable_post' \| 'ephemeral_post' \| 'retry_backoff' \| 'hold' \| 'none'` | opt |  |
| `first_text_post_wall_ms` | `number` | opt |  |
| `time_to_request_from_spawn_ms` | `number` | opt |  |
| `warm_spare_claimed` | `boolean` | opt |  |
| `time_origin_ms` | `number` | opt |  |
| `is_error` | `boolean` | req |  |
| `api_error_status` | `number \| null` | opt |  |
| `num_turns` | `number` | req |  |
| `result` | `string` | req |  |
| `stop_reason` | `string \| null` | req |  |
| `total_cost_usd` | `number` | req | Cumulative estimated cost in USD for this query() call, covering the same query-pipeline calls as modelUsage and sharing its lifecycle: cumulative… |
| `usage` | `NonNullableUsage` | req | MAIN AGENT LOOP ONLY — excludes Task subagent, sidechain, and auxiliary model calls, and is per-turn in streaming-input sessions. |
| `modelUsage` | `Record<string, ModelUsage>` | req | Per-model totals for every model call made through the query pipeline during this query() call — main loop, Task subagents, sidechains, and… |
| `permission_denials` | `SDKPermissionDenial[]` | req |  |
| `queued_turn_count` | `number` | opt | User-initiated sends still waiting in the command queue when this result was produced. |
| `structured_output` | `unknown` | opt |  |
| `deferred_tool_use` | `SDKDeferredToolUse` | opt |  |
| `terminal_reason` | `TerminalReason` | opt |  |
| `result_index` | `number` | opt | Delivery sequence of this result within the run: how many results the run numbered before this one, starting at 0, in the order the process writes… |
| `fast_mode_state` | `FastModeState` | opt |  |
| `fast_mode_disabled_reason` | `FastModeDisabledReason` | opt |  |
| `origin` | `SDKMessageOrigin` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `credential_expires_at` | `string` | opt | **since 0.3.318 (not in 0.3.311).** When the claude.ai credential this process's requests send ends for good, as an ISO 8601 instant, read as the result is written. |

### result:error_during_execution|error_max_turns|error_max_budget_usd|error_max_structured_output_retries — `SDKResultError` → Go `ResultError`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'result'` | req |  |
| `subtype` | `'error_during_execution' \| 'error_max_turns' \| 'error_max_budget_usd' \| 'error_max_structured_output_retries'` | req |  |
| `duration_ms` | `number` | req |  |
| `duration_api_ms` | `number` | req |  |
| `is_error` | `boolean` | req |  |
| `num_turns` | `number` | req |  |
| `stop_reason` | `string \| null` | req |  |
| `total_cost_usd` | `number` | req | Cumulative estimated cost in USD for this query() call, covering the same query-pipeline calls as modelUsage and sharing its lifecycle: cumulative… |
| `usage` | `NonNullableUsage` | req | MAIN AGENT LOOP ONLY — excludes Task subagent, sidechain, and auxiliary model calls, and is per-turn in streaming-input sessions. |
| `modelUsage` | `Record<string, ModelUsage>` | req | Per-model totals for every model call made through the query pipeline during this query() call — main loop, Task subagents, sidechains, and… |
| `permission_denials` | `SDKPermissionDenial[]` | req |  |
| `queued_turn_count` | `number` | opt | User-initiated sends still waiting in the command queue when this result was produced. |
| `errors` | `string[]` | req |  |
| `startup_failure_reason` | `SDKStartupFailureReason` | opt | Set on the zeroed error_during_execution result a stream-json run writes before exiting on a known startup failure; errors carries the same text… |
| `user_message_uuid` | `string` | opt | Client uuid of the user message that triggered this turn (submitMessage options.uuid), echoed back so a consumer can link this error result to the… |
| `user_message_uuids` | `string[]` | opt | Client uuids of every user message whose prompt this turn consumed, in consumption order — all members of a prompt batch the host merged into this… |
| `resume_reason` | `string` | opt | Why this turn continued one that a worker restart interrupted. |
| `terminal_reason` | `TerminalReason` | opt |  |
| `result_index` | `number` | opt | Delivery sequence of this result within the run: how many results the run numbered before this one, starting at 0, in the order the process writes… |
| `fast_mode_state` | `FastModeState` | opt |  |
| `fast_mode_disabled_reason` | `FastModeDisabledReason` | opt |  |
| `origin` | `SDKMessageOrigin` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `invalid_config_file` | {…} | opt | **since 0.3.325 (not in 0.3.311).** Set when startup_failure_reason is config_file_invalid: the config file that did not parse and its backup, as separate fields. |
| `invalid_config_file.path` | `string` | req | The file that is not valid JSON. |
| `invalid_config_file.is_global` | `boolean` | req | Whether this is Claude Code's global config file. |
| `invalid_config_file.backup_searched` | `boolean` | req | Whether Claude Code knows the result of its search for a backup of the file. |
| `invalid_config_file.backup_path` | `string` | opt | The backup that the message in errors names. |
| `credential_expires_at` | `string` | opt | **since 0.3.318 (not in 0.3.311).** When the claude.ai credential this process's requests send ends for good, as an ISO 8601 instant, read as the result is written. |

### system:init — `SDKSystemMessage` → Go `SystemInitMessage`

Session metadata the CLI emits at the start of each turn, normally ahead of every other message of that turn: session_id, model, working directory, tools, MCP servers, slash commands, permission mode, and the capabilities list for feature detection.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'init'` | req |  |
| `agents` | `string[]` | opt |  |
| `apiKeySource` | `ApiKeySource` | req | Where the credential used for API requests came from: 'ANTHROPIC_API_KEY' (environment variable), 'apiKeyHelper' (the configured helper command),… |
| `betas` | `string[]` | opt |  |
| `claude_code_version` | `string` | req |  |
| `cwd` | `string` | req |  |
| `tools` | `string[]` | req |  |
| `mcp_servers` | {…}[] | req |  |
| `mcp_servers[].name` | `string` | req |  |
| `mcp_servers[].status` | `string` | req |  |
| `mcp_servers[].source` | `string` | opt | Where the server definition came from — same values as McpServerStatus.source (sdk \| plugin \| a config scope). |
| `model` | `string` | req |  |
| `permissionMode` | `PermissionMode` | req | Permission mode for controlling how tool executions are handled. |
| `slash_commands` | `string[]` | req |  |
| `terminal_slash_commands` | `string[]` | opt | Subset of slash_commands whose UX is bound to the local terminal (e.g. exit, statusline). |
| `output_style` | `string` | req |  |
| `skills` | `string[]` | req |  |
| `plugins` | {…}[] | req |  |
| `plugins[].name` | `string` | req |  |
| `plugins[].path` | `string` | req |  |
| `plugins[].version` | `string` | opt | The plugin's version as declared in its plugin.json manifest, emitted verbatim (plugin-author-controlled — validate before trusting). |
| `plugin_errors` | {…}[] | opt | Plugin load-time errors. |
| `plugin_errors[].plugin` | `string` | req |  |
| `plugin_errors[].type` | `string` | req |  |
| `plugin_errors[].message` | `string` | req |  |
| `plugin_errors[].path` | `string` | opt | Present only when a --plugin-dir, SDK `plugins` or synced directory entry did not load at all: the path of that entry, resolved against the cwd… |
| `plugin_errors[].managed_lock` | {…} | opt | **since 0.3.325 (not in 0.3.311).** Present only when a managed-settings `enabledPlugins` entry dropped this --plugin-dir, SDK `plugins`, skills-directory or synced copy: the two… |
| `plugin_errors[].managed_lock.plugin_enabled` | `boolean` | req |  |
| `plugin_errors[].managed_lock.name` | `string` | req |  |
| `plugin_errors[].managed_lock.marketplace` | `string` | opt |  |
| `fast_mode_state` | `FastModeState` | opt |  |
| `fast_mode_disabled_reason` | `FastModeDisabledReason` | opt |  |
| `effort` | `('low' \| 'medium' \| 'high' \| 'xhigh' \| 'max') \| null` | opt | The effort level the session will send on its next request — after env overrides, session state, org caps and model-support downgrades; the same… |
| `view_mode` | `'focus' \| 'default'` | opt | Whether the session's transcript is in focus view ('focus': the model is told the user sees only its final message per turn, so clients may… |
| `capabilities` | `string[]` | opt | Protocol capabilities this CLI supports, so SDK consumers can feature-detect instead of version-sniffing. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### stream_event — `SDKPartialAssistantMessage` → Go `PartialAssistantMessage`

An incremental streaming event for the assistant message being generated, emitted only when partial messages are requested (--include-partial-messages).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'stream_event'` | req |  |
| `event` | `BetaRawMessageStreamEvent` | req | One Anthropic Messages API streaming event (message_start, content_block_start, content_block_delta, content_block_stop, message_delta,… |
| `parent_tool_use_id` | `string \| null` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `ttft_ms` | `number` | opt |  |
| `user_message_uuid` | `string` | opt | Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on a non-ping stream event each time that send… |
| `user_message_uuids` | `string[]` | opt | Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host… |
| `resume_reason` | `string` | opt | Why this frame's turn continues one that a worker restart interrupted. |

### system:compact_boundary — `SDKCompactBoundaryMessage` → Go `CompactBoundaryMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'compact_boundary'` | req |  |
| `compact_metadata` | {…} | req |  |
| `compact_metadata.trigger` | `'manual' \| 'auto'` | req |  |
| `compact_metadata.pre_tokens` | `number` | req |  |
| `compact_metadata.post_tokens` | `number` | opt |  |
| `compact_metadata.duration_ms` | `number` | opt |  |
| `compact_metadata.preserved_segment` | {…} | opt | Relink info for messagesToKeep. |
| `compact_metadata.preserved_segment.head_uuid` | `UUID` | req |  |
| `compact_metadata.preserved_segment.anchor_uuid` | `UUID` | req |  |
| `compact_metadata.preserved_segment.tail_uuid` | `UUID` | req |  |
| `compact_metadata.preserved_messages` | {…} | opt | Ordered messagesToKeep UUIDs. |
| `compact_metadata.preserved_messages.anchor_uuid` | `UUID` | req |  |
| `compact_metadata.preserved_messages.uuids` | `UUID[]` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:status — `SDKStatusMessage` → Go `StatusMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'status'` | req |  |
| `status` | `SDKStatus` | req |  |
| `permissionMode` | `PermissionMode` | opt |  |
| `compact_result` | `'success' \| 'failed'` | opt |  |
| `compact_error` | `string` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:api_retry — `SDKAPIRetryMessage` → Go `APIRetryMessage`

Emitted when an API request fails with a retryable error and will be retried after a delay. error_status is null for connection errors (e.g. timeouts) that had no HTTP response.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'api_retry'` | req |  |
| `attempt` | `number` | req |  |
| `max_retries` | `number` | req |  |
| `retry_delay_ms` | `number` | req |  |
| `error_status` | `number \| null` | req |  |
| `error` | `SDKAssistantMessageError` | req |  |
| `no_response` | {…} | opt | Present only when the API sent no response headers within the first-byte window (CLAUDE_STREAM_FIRST_BYTE_TIMEOUT_MS): waited_ms is how long the… |
| `no_response.waited_ms` | `number` | req |  |
| `no_response.retry_wait_ms` | `number` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:control_request_progress — `SDKControlRequestProgressMessage` → Go `RequestProgressMessage`

Progress for a long-running client-originated control_request (currently only side_question), correlated by request_id. status 'started' means the worker accepted the request and launched the work; 'api_retry' carries the same retry counters as SDKAPIRetryMessage and is present only for that status.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'control_request_progress'` | req |  |
| `request_id` | `string` | req | request_id of the in-flight control_request this progress belongs to. |
| `status` | `'started' \| 'api_retry'` | req |  |
| `attempt` | `number` | opt |  |
| `max_retries` | `number` | opt |  |
| `retry_delay_ms` | `number` | opt |  |
| `error_status` | `number \| null` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:model_refusal_fallback — `SDKModelRefusalFallbackMessage` → Go `ModelRefusalFallbackMessage`

Emitted when the primary model ends the stream with stop_reason "refusal" and the turn is retried once on a fallback model (direction: "retry").

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'model_refusal_fallback'` | req |  |
| `trigger` | `'refusal'` | req |  |
| `direction` | `'retry' \| 'revert' \| 'sticky'` | req |  |
| `scope` | `'session' \| 'local'` | opt | 'session': the main thread fell back and the session model is swapped. |
| `original_model` | `string` | req |  |
| `fallback_model` | `string` | req |  |
| `request_id` | `string \| null` | req |  |
| `api_refusal_category` | `string \| null` | opt | The refusal category ('cyber', 'bio', …): stop_details.category from the refused API response (client lane), or the fallback block's server-gated… |
| `api_refusal_explanation` | `string \| null` | opt | stop_details.explanation from the refused API response (client lane only — the server-lane trigger carries no explanation). |
| `retracted_message_uuids` | `string[]` | opt | Wire uuids of the messages this fallback retracted — the refused partial as the consumer received it (one uuid per normalized SDK message;… |
| `refused_user_message_uuid` | `string \| null` | opt | UUID of the user message the refused request was for — the rewind target and composer prefill for edit-and-retry. |
| `content` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:model_refusal_no_fallback — `SDKModelRefusalNoFallbackMessage` → Go `ModelRefusalNoFallbackMessage`

Emitted when the model ends the stream with stop_reason "refusal" and no retry runs: no fallback model is configured, or per-category routing declined the retry (the mapped fallback target is unresolvable, or CLAUDE_CODE_REFUSAL_FALLBACK_CATCH_ALL is explicitly disabled and the refusal category has no fallback map entry).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'model_refusal_no_fallback'` | req |  |
| `original_model` | `string` | req |  |
| `request_id` | `string \| null` | req |  |
| `api_refusal_category` | `string \| null` | opt |  |
| `api_refusal_explanation` | `string \| null` | opt |  |
| `refused_user_message_uuid` | `string \| null` | opt |  |
| `content` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:local_command_output — `SDKLocalCommandOutputMessage` → Go `LocalCommandOutputMessage`

Output from a local slash command (e.g. /voice, /usage).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'local_command_output'` | req |  |
| `content` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:hook_started — `SDKHookStartedMessage` → Go `HookStartedMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'hook_started'` | req |  |
| `hook_id` | `string` | req |  |
| `hook_name` | `string` | req |  |
| `hook_event` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:hook_progress — `SDKHookProgressMessage` → Go `HookProgressMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'hook_progress'` | req |  |
| `hook_id` | `string` | req |  |
| `hook_name` | `string` | req |  |
| `hook_event` | `string` | req |  |
| `stdout` | `string` | req |  |
| `stderr` | `string` | req |  |
| `output` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:hook_response — `SDKHookResponseMessage` → Go `HookResponseMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'hook_response'` | req |  |
| `hook_id` | `string` | req |  |
| `hook_name` | `string` | req |  |
| `hook_event` | `string` | req |  |
| `output` | `string` | req |  |
| `stdout` | `string` | req |  |
| `stderr` | `string` | req |  |
| `exit_code` | `number` | opt |  |
| `outcome` | `'success' \| 'error' \| 'cancelled'` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:plugin_install — `SDKPluginInstallMessage` → Go `PluginInstallMessage`

Headless plugin installation progress (CLAUDE_CODE_SYNC_PLUGIN_INSTALL). started/completed bracket the whole install; installed/failed carry a per-marketplace name.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'plugin_install'` | req |  |
| `status` | `'started' \| 'installed' \| 'failed' \| 'completed'` | req |  |
| `name` | `string` | opt |  |
| `error` | `string` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### tool_progress — `SDKToolProgressMessage` → Go `ToolProgressMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'tool_progress'` | req |  |
| `tool_use_id` | `string` | req |  |
| `tool_name` | `string` | req |  |
| `parent_tool_use_id` | `string \| null` | req |  |
| `elapsed_time_seconds` | `number` | req |  |
| `task_id` | `string` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `heartbeat` | `boolean` | opt |  |
| `subagent_type` | `string` | opt |  |
| `subagent_retry` | {…} | opt |  |
| `subagent_retry.agent_id` | `string` | req |  |
| `subagent_retry.attempt` | `number` | req |  |
| `subagent_retry.max_retries` | `number` | req |  |
| `subagent_retry.retry_delay_ms` | `number` | req |  |
| `subagent_retry.error_status` | `number \| null` | req |  |
| `subagent_retry.error_category` | `string` | req |  |

### auth_status — `SDKAuthStatusMessage` → Go `AuthStatusMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'auth_status'` | req |  |
| `isAuthenticating` | `boolean` | req |  |
| `output` | `string[]` | req |  |
| `error` | `string` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:task_notification — `SDKTaskNotificationMessage` → Go `TaskNotificationMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'task_notification'` | req |  |
| `task_id` | `string` | req |  |
| `run_id` | `string` | opt | Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run. |
| `tool_use_id` | `string` | opt |  |
| `status` | `'completed' \| 'failed' \| 'stopped'` | req |  |
| `reason` | `'worker_restart'` | opt | Machine-readable cause, set only when the task did not end through an ordinary completion, failure, or stop. |
| `output_file` | `string` | req |  |
| `summary` | `string` | req |  |
| `usage` | {…} | opt |  |
| `usage.total_tokens` | `number` | req |  |
| `usage.tool_uses` | `number` | req |  |
| `usage.duration_ms` | `number` | req |  |
| `resource_links` | `SDKMcpResourceLink[]` | opt | CLI-owned: for a backgrounded MCP task (task_type mcp_task) that completed, the `resource_link` content blocks of its final result — the files it… |
| `skip_transcript` | `boolean` | opt |  |
| `ambient` | `boolean` | opt | True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude… |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:task_started — `SDKTaskStartedMessage` → Go `TaskStartedMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'task_started'` | req |  |
| `task_id` | `string` | req |  |
| `run_id` | `string` | opt | Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run. |
| `tool_use_id` | `string` | opt |  |
| `description` | `string` | req |  |
| `subagent_type` | `string` | opt | Subagent type for Task tool subagents. |
| `is_backgrounded` | `boolean` | opt | Whether the task was registered in the background (true) or in the foreground with the spawning tool call blocking on it (false). |
| `spawn_depth` | `number` | opt | Nesting depth of a spawned subagent (local_agent) task: 1 for a top-level spawn, N+1 when spawned from inside a depth-N agent. |
| `parent_task_id` | `string` | opt | task_id of the subagent (local_agent) task whose agent launched this subagent, shell, monitor or workflow task. |
| `task_type` | `string` | opt |  |
| `workflow_name` | `string` | opt | meta.name from the workflow script (e.g. |
| `prompt` | `string` | opt |  |
| `skip_transcript` | `boolean` | opt | Ambient/housekeeping task. |
| `ambient` | `boolean` | opt | True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude… |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:task_updated — `SDKTaskUpdatedMessage` → Go `TaskUpdatedMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'task_updated'` | req |  |
| `task_id` | `string` | req |  |
| `run_id` | `string` | opt | Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run. |
| `patch` | {…} | req | Wire-safe subset of TaskState fields that changed. |
| `patch.status` | `'pending' \| 'running' \| 'completed' \| 'failed' \| 'killed' \| 'paused'` | opt |  |
| `patch.description` | `string` | opt |  |
| `patch.end_time` | `number` | opt |  |
| `patch.total_paused_ms` | `number` | opt |  |
| `patch.error` | `string` | opt |  |
| `patch.is_backgrounded` | `boolean` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:task_progress — `SDKTaskProgressMessage` → Go `TaskProgressMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'task_progress'` | req |  |
| `task_id` | `string` | req |  |
| `run_id` | `string` | opt | Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run. |
| `tool_use_id` | `string` | opt |  |
| `description` | `string` | req |  |
| `subagent_type` | `string` | opt | Subagent type for Task tool subagents. |
| `usage` | {…} | req |  |
| `usage.total_tokens` | `number` | req |  |
| `usage.tool_uses` | `number` | req |  |
| `usage.duration_ms` | `number` | req |  |
| `last_tool_name` | `string` | opt |  |
| `summary` | `string` | opt | A one-line status for the task's row. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:background_tasks_changed — `SDKBackgroundTasksChangedMessage` → Go `BackgroundTasksChangedMessage`

The full set of live background tasks, emitted whenever membership changes (start, completion, kill, a foreground agent being backgrounded) or an entry's `ambient` flag or `parent_task_id` changes.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'background_tasks_changed'` | req |  |
| `tasks` | {…}[] | req | Every live background task after the change. |
| `tasks[].task_id` | `string` | req |  |
| `tasks[].run_id` | `string` | opt | Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run. |
| `tasks[].task_type` | `string` | req |  |
| `tasks[].subagent_type` | `string` | opt | Agent type of a local_agent task, such as "general-purpose" or a custom agent's name; "main-session" for a backgrounded main session. |
| `tasks[].description` | `string` | req |  |
| `tasks[].parent_task_id` | `string` | opt | task_id of the subagent (local_agent) task whose agent launched this subagent, shell, monitor or workflow task. |
| `tasks[].ambient` | `boolean` | opt | True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude… |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:thinking_tokens — `SDKThinkingTokensMessage` → Go `ThinkingTokensMessage`

Live thinking-token estimate, digested from thinking_delta.estimated_tokens during the redacted-thinking phase (where the API otherwise streams only pings). estimated_tokens is the running total for the current thinking block; estimated_tokens_delta is the increment carried by this frame.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'thinking_tokens'` | req |  |
| `estimated_tokens` | `number` | req |  |
| `estimated_tokens_delta` | `number` | req |  |
| `user_message_uuid` | `string` | opt | Client uuid of the user message that triggered this turn (submitMessage options.uuid), stamped on every thinking_tokens frame of a headless… |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:session_state_changed — `SDKSessionStateChangedMessage` → Go `SessionStateChangedMessage`

Mirrors notifySessionStateChanged.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'session_state_changed'` | req |  |
| `state` | `'idle' \| 'running' \| 'requires_action'` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:worker_shutting_down — `SDKWorkerShuttingDownMessage` → Go `WorkerShuttingDownMessage`

Emitted by the bridge on opt-in graceful worker teardown (only when the teardown caller supplied a reason), before the heartbeat stops, so remote clients can show why the worker went away instead of waiting for heartbeat timeout.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'worker_shutting_down'` | req |  |
| `reason` | `string` | req | Short snake_case reason set by the host CLI (not user input), e.g. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:commands_changed — `SDKCommandsChangedMessage` → Go `CommandsChangedMessage`

Fire-and-forget push of the full slash-command list after a mid-session change (e.g. skills discovered dynamically as the agent works in a subdirectory).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'commands_changed'` | req |  |
| `commands` | `SlashCommand[]` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:notification — `SDKNotificationMessage` → Go `NotificationMessage`

Loop-side text notification.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'notification'` | req |  |
| `key` | `string` | req |  |
| `text` | `string` | req |  |
| `priority` | `'low' \| 'medium' \| 'high' \| 'immediate'` | req |  |
| `color` | `string` | opt |  |
| `timeout_ms` | `number` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:files_persisted — `SDKFilesPersistedEvent` → Go `FilesPersistedEvent`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'files_persisted'` | req |  |
| `files` | {…}[] | req |  |
| `files[].filename` | `string` | req |  |
| `files[].file_id` | `string` | req |  |
| `failed` | {…}[] | req |  |
| `failed[].filename` | `string` | req |  |
| `failed[].error` | `string` | req |  |
| `processed_at` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### tool_use_summary — `SDKToolUseSummaryMessage` → Go `ToolUseSummaryMessage`



| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'tool_use_summary'` | req |  |
| `summary` | `string` | req |  |
| `preceding_tool_use_ids` | `string[]` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:memory_recall — `SDKMemoryRecallMessage` → Go `MemoryRecallMessage`

Emitted when the memory recall supervisor surfaces relevant memories into the turn.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'memory_recall'` | req |  |
| `mode` | `'select' \| 'synthesize'` | req | How memories were surfaced: 'select' returns full file bodies chosen by the parallel selector; 'synthesize' returns a Sonnet-authored paragraph… |
| `memories` | {…}[] | req |  |
| `memories[].path` | `string` | req | Absolute path to the memory file, a synthesis sentinel of the form `<synthesis:DIR>` when mode is 'synthesize', or an https URL when scope is… |
| `memories[].scope` | `'personal' \| 'team' \| 'organization'` | req |  |
| `memories[].content` | `string` | opt | The surfaced memory body. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### rate_limit_event — `SDKRateLimitEvent` → Go `RateLimitEvent`

Rate limit event emitted when rate limit info changes.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'rate_limit_event'` | req |  |
| `rate_limit_info` | `SDKRateLimitInfo` | req | Rate limit information for claude.ai subscription users. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:elicitation_complete — `SDKElicitationCompleteMessage` → Go `ElicitationCompleteMessage`

Emitted when an MCP server confirms that a URL-mode elicitation is complete.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'elicitation_complete'` | req |  |
| `mcp_server_name` | `string` | req |  |
| `elicitation_id` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:permission_denied — `SDKPermissionDeniedMessage` → Go `PermissionDeniedMessage`

Emitted when a tool call is auto-denied without an interactive permission prompt (e.g. auto-mode classifier, dontAsk mode, headless-agent auto-deny, or a deny rule).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'permission_denied'` | req |  |
| `tool_name` | `string` | req |  |
| `tool_use_id` | `string` | req |  |
| `agent_id` | `string` | opt | Subagent ID when the denied tool call originated inside a subagent. |
| `decision_reason_type` | `string` | opt | Discriminator from PermissionDecisionReason (e.g. |
| `decision_reason` | `string` | opt | Human-readable reason from the deciding component, when available. |
| `message` | `string` | req | The rejection message returned to the model in the tool_result. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### prompt_suggestion — `SDKPromptSuggestionMessage` → Go `PromptSuggestionMessage`

Predicted next user prompt, emitted after each turn when promptSuggestions is enabled.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'prompt_suggestion'` | req |  |
| `suggestion` | `string` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:mirror_error — `SDKMirrorErrorMessage` → Go `MirrorErrorMessage`

Emitted when SessionStore.append() rejects or times out for a transcript-mirror batch after bounded retry (3 attempts with short backoff; timeouts are not retried).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'mirror_error'` | req |  |
| `error` | `string` | req |  |
| `key` | {…} | req |  |
| `key.projectKey` | `string` | req |  |
| `key.sessionId` | `string` | req |  |
| `key.subpath` | `string` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### system:informational — `SDKInformationalMessage` → Go `InformationalMessage`

Generic text banner emitted by the loop — non-error status lines, hook feedback (e.g. a UserPromptSubmit hook's block reason), slash-command output.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req |  |
| `subtype` | `'informational'` | req |  |
| `content` | `string` | req |  |
| `level` | `'info' \| 'notice' \| 'suggestion' \| 'warning'` | req | Render level. |
| `tool_use_id` | `string` | opt | Dedupes progress messages for the same tool use. |
| `prevent_continuation` | `boolean` | opt | When true, execution stops after this message (e.g. a Stop hook denied continuation). |
| `tag` | `string` | opt | Opaque feature tag on a line a host may treat specially; absent on ordinary lines. |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### conversation_reset — `SDKConversationResetMessage` → Go `ConversationResetMessage`

Emitted by /clear, plan-mode exit, fresh-session, and onboarding flows.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'conversation_reset'` | req |  |
| `new_conversation_id` | `UUID` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `trigger` | `'clear' \| 'plan_mode_exit' \| 'fresh_session' \| 'onboarding'` | opt | What discarded the conversation: 'clear' is the /clear command (or its /reset and /new aliases), 'plan_mode_exit' is leaving plan mode with the… |
| `user_message_uuid` | `string` | opt | Only with trigger 'clear': the uuid of the user message whose /clear was executed (the client's own uuid for that message when it came from a… |
| `timestamp` | `string` | opt | When the reset happened, as an ISO 8601 string in UTC read from the clock of the process that performed it. |

### system:session_writes_lost — `SDKSessionWritesLostMessage` → Go `SessionWritesLostMessage` (**since 0.3.317, not in 0.3.311**)

The CLI could not save lines of a session transcript, changes to ~/.claude.json, or both, because the disk was full or its quota was used up.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'system'` | req | **since 0.3.312-0.3.325 (not in 0.3.311).**  |
| `subtype` | `'session_writes_lost'` | req | **since 0.3.312-0.3.325 (not in 0.3.311).**  |
| `error_code` | `'ENOSPC' \| 'EDQUOT'` | req | **since 0.3.312-0.3.325 (not in 0.3.311).** Why the writes failed. |
| `lost` | `('transcript' \| 'global_config')[]` | req | **since 0.3.312-0.3.325 (not in 0.3.311).** What was not saved. |
| `transcript_session_ids` | `string[]` | opt | **since 0.3.312-0.3.325 (not in 0.3.311).** The sessions whose transcripts lost lines, in the order each first lost one. |
| `uuid` | `UUID` | req | **since 0.3.312-0.3.325 (not in 0.3.311).**  |
| `session_id` | `string` | req | **since 0.3.312-0.3.325 (not in 0.3.311).**  |

### active_goal — `SDKActiveGoalMessage` → Go `ActiveGoalMessage`

Emitted when the user's /goal Stop hook reports met (clears) or not-yet-met (bumps iterations + last_reason).

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'active_goal'` | req |  |
| `value` | {…} \| null | req |  |
| `value.condition` | `string` | req |  |
| `value.iterations` | `number` | req |  |
| `value.set_at` | `number` | req |  |
| `value.tokens_at_start` | `number` | req |  |
| `value.last_reason` | `string` | opt |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |

### keep_alive — `SDKKeepAliveMessage` → Go `KeepAliveMessage`

Liveness heartbeat.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'keep_alive'` | req |  |

---

## 4. Supporting types

### 4.1 Assistant message body and content blocks

`SDKAssistantMessage.message` is an Anthropic Messages API `BetaMessage` (from `@anthropic-ai/sdk`, peer dependency >=0.93.0, not declared in sdk.d.ts). `SDKUserMessage.message` is a `MessageParam`, whose `content` is a string or an array of content blocks.

`types.go` models these by hand as `APIMessage`, `MessageParam`, `Usage` and the `ContentBlock` union. It merges response blocks and request block-params into one union keyed by `type`. Unmodelled block types decode as unknown, with `Raw` kept.

| Field | Type | Req | Notes |
|---|---|---|---|
| `id` | `string` | req |  |
| `type` | `'message'` | req |  |
| `role` | `'assistant'` | req |  |
| `model` | `string` | req |  |
| `content` | `BetaContentBlock[]` | req |  |
| `stop_reason` | `string \| null` | req |  |
| `stop_sequence` | `string \| null` | req |  |
| `usage` | `NonNullableUsage` | req |  |
| `container` | {…} \| null | opt |  |
| `container.id` | `string` | req |  |
| `container.expires_at` | `string` | req |  |
| `context_management` | `Record<string, unknown> \| null` | opt |  |


**`Usage`** (`NonNullableUsage`: BetaUsage with nulls removed; result messages carry it):

_(no fields)_


| Block `type` | Go type | Where | Fields |
|---|---|---|---|
| `text` | TextBlock | assistant, user | text, citations? |
| `thinking` | ThinkingBlock | assistant | thinking, signature |
| `redacted_thinking` | RedactedThinkingBlock | assistant | data |
| `tool_use` | ToolUseBlock | assistant | id, name, input |
| `server_tool_use` | ServerToolUseBlock | assistant | id, name, input |
| `tool_result` | ToolResultBlock | user | tool_use_id, content? (string \| blocks), is_error? |
| `image` | ImageBlock | user | source {type: base64\|url\|file …} |
| `document` | DocumentBlock | user | source, title?, context?, citations? |
| `search_result` | SearchResultBlock | user | source, title, content |
| `mcp_tool_use / mcp_tool_result` | MCPToolUseBlock / MCPToolResultBlock | assistant | id, name, server_name, input / tool_use_id, is_error, content |
| `container_upload` | ContainerUploadBlock | assistant | file_id |
| `web_search_tool_result, web_fetch_tool_result, code_execution_tool_result, bash_code_execution_tool_result, text_editor_code_execution_tool_result` | …ToolResultBlock | assistant | tool_use_id, content (kept raw) |


### 4.2 `SDKPartialAssistantMessage` (`stream_event`)

This message is emitted only with `includePartialMessages` / `--include-partial-messages`. `event` is one `BetaRawMessageStreamEvent`; Go: the `StreamEvent` union.

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'stream_event'` | req |  |
| `event` | `BetaRawMessageStreamEvent` | req | One Anthropic Messages API streaming event (message_start, content_block_start, content_block_delta, content_block_stop, message_delta,… |
| `parent_tool_use_id` | `string \| null` | req |  |
| `uuid` | `UUID` | req |  |
| `session_id` | `string` | req |  |
| `ttft_ms` | `number` | opt |  |
| `user_message_uuid` | `string` | opt | Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on a non-ping stream event each time that send… |
| `user_message_uuids` | `string[]` | opt | Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host… |
| `resume_reason` | `string` | opt | Why this frame's turn continues one that a worker restart interrupted. |


| event.type | Go type | Fields |
|---|---|---|
| `message_start` | MessageStartEvent | message: APIMessage |
| `message_delta` | MessageDeltaEvent | delta {stop_reason, stop_sequence}, usage: MessageDeltaUsage, context_management? |
| `message_stop` | MessageStopEvent | — |
| `content_block_start` | ContentBlockStartEvent | index, content_block: ContentBlock |
| `content_block_delta` | ContentBlockDeltaEvent | index, delta: ContentBlockDelta (text_delta.text \| input_json_delta.partial_json \| citations_delta.citation \| thinking_delta.thinking \| signature_delta.signature) |
| `content_block_stop` | ContentBlockStopEvent | index |


### 4.3 Result variants (`SDKResultMessage = SDKResultSuccess | SDKResultError`)

- `result:success` has `result` (text), `structured_output?`, `deferred_tool_use?`, `api_error_status?` and timing fields.
- `result:error_during_execution | error_max_turns | error_max_budget_usd | error_max_structured_output_retries` has `errors: string[]` and `startup_failure_reason?` (`SDKStartupFailureReason`). In 0.3.325 it also has `invalid_config_file?` (**since 0.3.325**).
- Both carry `usage`, `modelUsage` (Record<model, ModelUsage>), `permission_denials`, `total_cost_usd`, `num_turns`, `terminal_reason?`, `result_index?`, `fast_mode_*` and `origin?`.
- 0.3.318 adds `credential_expires_at?`.
- Field tables are in section 3.

In Go, `Message.ResultMessage()` gives the `ResultMessage` union, with `AsResultSuccess` and `AsResultError`.

**ModelUsage**

| Field | Type | Req | Notes |
|---|---|---|---|
| `inputTokens` | `number` | req |  |
| `outputTokens` | `number` | req |  |
| `thinkingTokens` | `number` | opt | Thinking tokens, already counted inside outputTokens. |
| `cacheReadInputTokens` | `number` | req |  |
| `cacheCreationInputTokens` | `number` | req |  |
| `cacheCreationEphemeral5mInputTokens` | `number` | opt | Cache writes with a 5-minute lifetime, already counted inside cacheCreationInputTokens. |
| `cacheCreationEphemeral1hInputTokens` | `number` | opt | Cache writes with a 1-hour lifetime, already counted inside cacheCreationInputTokens. costUSD prices them at the 1-hour rate when the model's… |
| `webSearchRequests` | `number` | req |  |
| `costUSD` | `number` | req |  |
| `contextWindow` | `number` | req |  |
| `maxOutputTokens` | `number` | req |  |
| `canonicalModel` | `string` | opt | Canonical model id used for the pricing lookup (e.g. |
| `provider` | `string` | opt | API provider that served this model (e.g. |
| `costBasis` | `'list' \| 'managed' \| 'unknown'` | opt | Which price table the most recent request for this model was priced at: Claude Code's built-in list prices ('list'), the organization's… |


**SDKPermissionDenial**

| Field | Type | Req | Notes |
|---|---|---|---|
| `tool_name` | `string` | req |  |
| `tool_use_id` | `string` | req |  |
| `tool_input` | `Record<string, unknown>` | req |  |


**SDKMessageOrigin** (union keyed by `kind`): `human`, `channel`, `peer`, `task-notification`, `coordinator`, `unclassified`, `observer`, `auto-continuation`, `observer-activity`. In Go it is `MessageOrigin` with `MessageOrigin<Kind>` variants.

---

## 5. Control protocol

### 5.0 Envelopes

**SDKControlRequest** (either direction):

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'control_request'` | req |  |
| `request_id` | `string` | req | Chosen by the sender, unique among its in-flight requests; the control_response (and any control_cancel_request) for this request echoes it. |
| `request` | `SDKControlRequestInner` | req |  |


**SDKControlResponse**: `{type:"control_response", response: ControlResponse | ControlErrorResponse}`. In Go, `ControlResponse.Response` is the `ControlResponseResponse` union keyed by `subtype`.

**ControlResponse** (`subtype:"success"`), Go `ControlSuccessResponse`:

| Field | Type | Req | Notes |
|---|---|---|---|
| `subtype` | `'success'` | req |  |
| `request_id` | `string` | req | The request_id of the control_request this answers. |
| `response` | `Record<string, unknown>` | opt | The success payload, shaped as documented for the answered request's subtype; absent or {} for requests that are merely acknowledged. |
| `pending_permission_requests` | `SDKControlRequest[]` | opt | can_use_tool requests this CLI process has issued and not yet resolved, so a client joining an already-initialized session learns about in-flight… |
| `pending_user_dialog_requests` | `SDKControlRequest[]` | opt | request_user_dialog requests this CLI process has issued and not yet resolved (sibling of pending_permission_requests, with the same… |
| `pending_elicitation_requests` | `SDKControlRequest[]` | opt | **since 0.3.312-0.3.325 (not in 0.3.311).** elicitation requests that this CLI process sent and that are not yet answered or withdrawn (sibling of pending_permission_requests), each as it… |


**ControlErrorResponse** (`subtype:"error"`). The CLI also sends `error_code` (e.g. `reload_plugins_busy`), which the d.ts strips; Go reads it with `ControlErrorResponse.ErrorCode()`.

| Field | Type | Req | Notes |
|---|---|---|---|
| `subtype` | `'error'` | req |  |
| `request_id` | `string` | req | The request_id of the control_request this answers. |
| `error` | `string` | req | Human-readable failure description. |
| `pending_permission_requests` | `SDKControlRequest[]` | opt | can_use_tool requests this CLI process has issued and not yet resolved, so a client joining an already-initialized session learns about in-flight… |
| `pending_user_dialog_requests` | `SDKControlRequest[]` | opt | request_user_dialog requests this CLI process has issued and not yet resolved (sibling of pending_permission_requests, with the same… |
| `pending_elicitation_requests` | `SDKControlRequest[]` | opt | **since 0.3.312-0.3.325 (not in 0.3.311).** elicitation requests that this CLI process sent and that are not yet answered or withdrawn (sibling of pending_permission_requests), each as it… |


**SDKControlCancelRequest**: `{type:"control_cancel_request", request_id}`. Either side sends it to withdraw its own in-flight request. When it arrives for a host handler, the SDK aborts that handler's AbortSignal.

**SDKKeepAliveMessage**: `{type:"keep_alive"}`.

### 5.1 Control-request subtypes in the d.ts union

There are **40 subtypes in 0.3.311** and 41 in 0.3.325. "Direction" means who sends the request.

| Subtype | TS type → Go type | Direction | Request fields | Success response |
|---|---|---|---|---|
| `interrupt` | SDKControlInterruptRequest → InterruptRequest | host → CLI | cancel_queued? | `SDKControlInterruptResponse` |
| `can_use_tool` | SDKControlPermissionRequest → CanUseToolRequest | CLI → host | tool_name, mcp_server?, input, permission_suggestions?, blocked_path?, decision_reason?, decision_reason_type?, classifier_approvable?, suppress_always_allow_rule?, default_to_no?, matched_ask_rule?, title?, display_name?, tool_use_id, agent_id?, description?, requires_user_interaction? | `PermissionResult` + `toolUseID` |
| `initialize` | SDKControlInitializeRequest → InitializeRequest | host → CLI | hooks?, sdkMcpServers?, sdkMcpServerConfigs?, sdkMcpServerManifests?, sdkMcpServerManifestsOrigin?, jsonSchema?, systemPrompt?, appendSystemPrompt?, systemPromptSnapshot?, planModeInstructions?, toolAliases?, excludeDynamicSections?, agents?, title?, skills?, promptSuggestions?, agentProgressSummaries?, forwardSubagentText?, supportedDialogKinds?, perTaskStopAffordance?, plugins? | `SDKControlInitializeResponse` (7.6) |
| `set_permission_mode` | SDKControlSetPermissionModeRequest → SetPermissionModeRequest | host → CLI | mode | ack (observed `{mode}`) |
| `set_model` | SDKControlSetModelRequest → SetModelRequest | host → CLI | model? | ack |
| `set_max_thinking_tokens` | SDKControlSetMaxThinkingTokensRequest → SetMaxThinkingTokensRequest | host → CLI | max_thinking_tokens?, thinking_display? | ack |
| `rename_session` | SDKControlRenameSessionRequest → RenameSessionRequest | host → CLI | title, source?, session_id? | ack |
| `set_color` | SDKControlSetColorRequest → SetColorRequest | host → CLI | color | ack |
| `mcp_status` | SDKControlMcpStatusRequest → McpStatusRequest | host → CLI | liveReply? | `{mcpServers: McpServerStatus[]}` (sdk.mjs) |
| `get_context_usage` | SDKControlGetContextUsageRequest → GetContextUsageRequest | host → CLI | detail? | `SDKControlGetContextUsageResponse` |
| `get_session_cost` | SDKControlGetSessionCostRequest → GetSessionCostRequest | host → CLI | — | undeclared; observed `{text}` |
| `list_models` | SDKControlListModelsRequest → ListModelsRequest | host → CLI | — | undeclared; observed `{models: ModelInfo[]}` |
| `get_usage` | SDKControlGetUsageRequest → GetUsageRequest | host → CLI | skip_behaviors? | `SDKControlGetUsageResponse` |
| `get_binary_version` | SDKControlGetBinaryVersionRequest → GetBinaryVersionRequest | host → CLI | — | undeclared |
| `mcp_call` | SDKControlMcpCallRequest → McpCallRequest | host → CLI | tool, arguments?, expires_at?, timeout_ms?, result_not_for_model?, input_files?, output_files? | undeclared (MCP tools/call result) |
| `file_suggestions` | SDKControlFileSuggestionsRequest → FileSuggestionsRequest | host → CLI | query | undeclared |
| `hook_callback` | SDKHookCallbackRequest → HookCallbackRequest | CLI → host | callback_id, input, tool_use_id? | `HookJSONOutput` |
| `mcp_message` | SDKControlMcpMessageRequest → McpMessageRequest | CLI → host | server_name, message | `{mcp_response: JSONRPCMessage}` |
| `rewind_files` | SDKControlRewindFilesRequest → RewindFilesRequest | host → CLI | user_message_id, dry_run? | `RewindFilesResult` |
| `cancel_async_message` | SDKControlCancelAsyncMessageRequest → CancelAsyncMessageRequest | host → CLI | message_uuid | `{cancelled}` (sdk.mjs) |
| `read_file` | SDKControlReadFileRequest → ReadFileRequest | host → CLI | path, max_bytes?, encoding? | `SDKControlReadFileResponse` |
| `seed_read_state` | SDKControlSeedReadStateRequest → SeedReadStateRequest | host → CLI | path, mtime | ack |
| `mcp_set_servers` | SDKControlMcpSetServersRequest → McpSetServersRequest | host → CLI | servers | `McpSetServersResult` |
| `register_repo_root` | SDKControlRegisterRepoRootRequest → RegisterRepoRootRequest | host → CLI | directory, reload_claude_md?, reload_plugins?, reload_skills? | undeclared |
| `reload_plugins` | SDKControlReloadPluginsRequest → ReloadPluginsRequest | host → CLI | hold_on_cache_impact?, add_plugins? | `SDKControlReloadPluginsResponse` |
| `reload_skills` | SDKControlReloadSkillsRequest → ReloadSkillsRequest | host → CLI | — | `SDKControlReloadSkillsResponse` |
| `reload_output_styles` | SDKControlReloadOutputStylesRequest → ReloadOutputStylesRequest | host → CLI | — | `SDKControlReloadOutputStylesResponse` |
| `mcp_reconnect` | SDKControlMcpReconnectRequest → McpReconnectRequest | host → CLI | serverName | ack |
| `mcp_toggle` | SDKControlMcpToggleRequest → McpToggleRequest | host → CLI | serverName, enabled | ack |
| `stop_task` | SDKControlStopTaskRequest → StopTaskRequest | host → CLI | task_id | ack |
| `background_tasks` | SDKControlBackgroundTasksRequest → BackgroundTasksRequest | host → CLI | tool_use_id? | `{backgrounded?: boolean}` (sdk.mjs) |
| `get_task_output` | SDKControlGetTaskOutputRequest → GetTaskOutputRequest | host → CLI | task_id | `SDKControlGetTaskOutputResponse` |
| `apply_flag_settings` | SDKControlApplyFlagSettingsRequest → ApplyFlagSettingsRequest | host → CLI | settings | ack |
| `get_settings` | SDKControlGetSettingsRequest → GetSettingsRequest | host → CLI | — | undeclared; observed `{effective, sources, applied}` |
| `get_hooks_listing` | SDKControlGetHooksListingRequest → GetHooksListingRequest | host → CLI | — | `SDKControlGetHooksListingResponse` |
| `update_settings` | SDKControlUpdateSettingsRequest → UpdateSettingsRequest | host → CLI | source, settings | ack |
| `elicitation` | SDKControlElicitationRequest → ElicitationControlRequest | CLI → host | mcp_server_name, message, mode?, url?, elicitation_id?, requested_schema?, property_order?, title?, display_name?, description? | `ElicitResult` {action, content?} |
| `request_user_dialog` | SDKControlRequestUserDialogRequest → UserDialogControlRequest | CLI → host | dialog_kind, payload, tool_use_id? | `UserDialogResult` {behavior:"completed", result} \| {behavior:"cancelled"} |
| `list_permission_rules` | SDKControlListPermissionRulesRequest → ListPermissionRulesRequest | host → CLI | — | `SDKControlListPermissionRulesResponse` |
| `mcp_read_resource` | SDKControlMcpReadResourceRequest → McpReadResourceRequest | host → CLI | serverName, uri | `SDKControlMcpReadResourceResponse` |
| `set_has_subscribers` | SDKControlSetHasSubscribersRequest → SetHasSubscribersRequest **(since 0.3.312-0.3.325, not in 0.3.311)** | host → CLI | has_subscribers | ack |


### 5.2 CLI → host requests that the SDK handles but the d.ts union omits

| Subtype | Handled by | Answer |
|---|---|---|
| `oauth_token_refresh` | internal option `getOAuthToken` | `{accessToken, reason?}` |
| `host_auth_token_refresh` | internal option `getHostAuthToken` | `{authToken}` |
| `remote_control_work_secret` | enableRemoteControl refresh callback | `{work_secret}` |
| `ui_copy / ui_prompt_read / ui_prompt_fill / ui_prompt_paint / ui_prompt_suggest / ui_read_selection` | internal `setUiHost` | host-defined |


Any other unknown subtype is answered with an error.

### 5.3 `can_use_tool` (CLI → host)

| Field | Type | Req | Notes |
|---|---|---|---|
| `subtype` | `'can_use_tool'` | req |  |
| `tool_name` | `string` | req |  |
| `mcp_server` | `McpServerProvenance` | opt |  |
| `input` | `Record<string, unknown>` | req |  |
| `permission_suggestions` | `PermissionUpdate[]` | opt |  |
| `blocked_path` | `string` | opt |  |
| `decision_reason` | `string` | opt | Human-readable reason the ask escalated, for the consent line of the host's dialog. |
| `decision_reason_type` | `'rule' \| 'mode' \| 'subcommandResults' \| 'permissionPromptTool' \| 'hook' \| 'asyncAgent' \| 'sandboxOverride' …` | opt | Structured discriminator for why auto-mode escalated. |
| `classifier_approvable` | `boolean` | opt | Set when a safetyCheck is present anywhere in the decision reason (including nested inside subcommandResults for compound bash). false = at least… |
| `suppress_always_allow_rule` | `boolean` | opt | True when the dialog must not offer the persistent "don't ask again" row for this ask: accepting it would write a whole-tool allow rule broader… |
| `default_to_no` | `boolean` | opt | True when the ask must not be approvable by a single stray keystroke (PermissionAskDecision.defaultToNo): a terminal-style prompt opens on its… |
| `matched_ask_rule` | {…} | opt | Set when a user-configured ask RULE (permissions.ask) forced this prompt but the ask carries the tool's own decision_reason — the ask-rule… |
| `matched_ask_rule.source` | `string` | req |  |
| `matched_ask_rule.tool_name` | `string` | req |  |
| `matched_ask_rule.rule_content` | `string` | opt |  |
| `title` | `string` | opt |  |
| `display_name` | `string` | opt |  |
| `tool_use_id` | `string` | req |  |
| `agent_id` | `string` | opt |  |
| `description` | `string` | opt |  |
| `requires_user_interaction` | `boolean` | opt | True when one-tap Approve/Deny must not be offered: the tool's approval card IS the user-interaction surface (Tool.requiresUserInteraction(): the… |


**How the SDK calls `canUseTool`.** It calls `canUseTool(tool_name, input, opts)` with these opts:

| opts field | From |
|---|---|
| `signal` | aborted by `control_cancel_request` / close |
| `suggestions` | `permission_suggestions` |
| `blockedPath` | `blocked_path` |
| `mcpServer` | `mcp_server` |
| `decisionReason` | `decision_reason` |
| `decisionReasonType` | `decision_reason_type` |
| `title` | `title` |
| `displayName` | `display_name` |
| `description` | `description` |
| `defaultToNo` | `default_to_no` |
| `suppressAlwaysAllowRule` | `suppress_always_allow_rule` |
| `toolUseID` | `tool_use_id` |
| `agentID` | `agent_id` |
| `requestId` | envelope `request_id` |
| `requiresUserInteraction` | `requires_user_interaction` |
| `matchedAskRule` | `matched_ask_rule`: {source, toolName, ruleContent?} |
| `(hidden)` | `decision_reason_code`, `server_policy_ask_only`, `plugin`, `server_prompt`, `computer_folder` are also forwarded |


**The reply.** It is `{subtype:"success", request_id, response: {...PermissionResult, toolUseID: tool_use_id}}`. A `null` from `canUseTool` means the host answered out-of-band, so the SDK writes nothing.

**PermissionResult** (union on `behavior`; Go `PermissionResult` with `PermissionResultAllow` / `PermissionResultDeny`):

| behavior | Fields |
|---|---|
| `allow` | updatedInput?: Record<string, unknown>, updatedPermissions?: PermissionUpdate[], toolUseID?, decisionClassification? |
| `deny` | message: string, interrupt?: boolean, toolUseID?, decisionClassification? |


Here `decisionClassification` is `'user_temporary' | 'user_permanent' | 'user_reject'`.

**PermissionUpdate** (union on `type`; Go `PermissionUpdate`):

| type | Go variant | Fields |
|---|---|---|
| `addRules` | PermissionUpdateAddRules | rules: {toolName, ruleContent?}[], behavior: allow\|deny\|ask, destination |
| `replaceRules` | PermissionUpdateReplaceRules | rules, behavior, destination |
| `removeRules` | PermissionUpdateRemoveRules | rules, behavior, destination |
| `setMode` | PermissionUpdateSetMode | mode: PermissionMode, destination |
| `addDirectories` | PermissionUpdateAddDirectories | directories: string[], destination |
| `removeDirectories` | PermissionUpdateRemoveDirectories | directories: string[], destination |


**PermissionUpdateDestination:** `userSettings | projectSettings | localSettings | session | cliArg`.

**PermissionBehavior:** `allow | deny | ask`.

**PermissionMode:** `default | acceptEdits | bypassPermissions | plan | dontAsk | auto`. `bypassPermissions` requires `allowDangerouslySkipPermissions`. In Go these are the `PermissionMode*` constants.

### 5.4 Other CLI → host requests

- **hook_callback** `{callback_id, input: HookInput, tool_use_id?}` → `HookJSONOutput` (section 6). The SDK looks up `callback_id` (`hook_<n>`) and calls the callback with `(input, tool_use_id, {signal})`.
- **mcp_message** `{server_name, message: JSONRPCMessage}` → `{mcp_response}`. A request is routed to the in-process server's transport and its reply returned. A notification is delivered and answered `{jsonrpc:"2.0", result:{}, id:0}`.
- **elicitation**: section 7.1.
- **request_user_dialog** `{dialog_kind, payload, tool_use_id?}` → `onUserDialog({dialogKind, payload, toolUseID}, {signal, requestId})` → `UserDialogResult`. With no handler the request stays unanswered.

---

## 6. Hooks

**HookEvent:** the 33 events below; the list is identical in 0.3.311 and 0.3.325. Every `*HookInput` is `BaseHookInput & {hook_event_name: <event>, …}`. In Go each input is a flat struct, and the `HookInput` union is keyed by `hook_event_name`.

**BaseHookInput**

| Field | Type | Req | Notes |
|---|---|---|---|
| `session_id` | `string` | req |  |
| `transcript_path` | `string` | req |  |
| `cwd` | `string` | req |  |
| `prompt_id` | `string` | opt | UUID correlating a user prompt with all subsequent events until the next prompt. |
| `permission_mode` | `string` | opt |  |
| `agent_id` | `string` | opt | Subagent identifier. |
| `agent_type` | `string` | opt | Agent type name (e.g., "general-purpose", "code-reviewer"). |
| `effort` | {…} | opt | Reasoning effort applied to the current turn. |
| `effort.level` | `string` | req | Active effort level for the current turn (e.g., "low", "medium", "high", "xhigh", "max"), after any silent downgrade for the selected model. |


| HookEvent | Input type | Event-specific input fields | hookSpecificOutput type |
|---|---|---|---|
| `PreToolUse` | PreToolUseHookInput | tool_name: string; tool_input: unknown; tool_use_id: string; mcp_server?: McpServerProvenance | PreToolUseHookSpecificOutput |
| `PostToolUse` | PostToolUseHookInput | tool_name: string; tool_input: unknown; tool_response: unknown; tool_use_id: string; duration_ms?: number; mcp_server?: McpServerProvenance | PostToolUseHookSpecificOutput |
| `PostToolUseFailure` | PostToolUseFailureHookInput | tool_name: string; tool_input: unknown; tool_use_id: string; error: string; is_interrupt?: boolean; duration_ms?: number; mcp_server?: McpServerProvenance | PostToolUseFailureHookSpecificOutput |
| `PostToolBatch` | PostToolBatchHookInput | tool_calls: PostToolBatchToolCall[] | PostToolBatchHookSpecificOutput |
| `PermissionDenied` | PermissionDeniedHookInput | tool_name: string; tool_input: unknown; tool_use_id: string; reason: string; mcp_server?: McpServerProvenance | PermissionDeniedHookSpecificOutput |
| `Notification` | NotificationHookInput | message: string; title?: string; notification_type: string | NotificationHookSpecificOutput |
| `UserPromptSubmit` | UserPromptSubmitHookInput | prompt: string; source?: 'user' \| 'sdk' \| 'system' \| 'loop_wakeup' \| 'schedule_wakeup; session_title?: string | UserPromptSubmitHookSpecificOutput |
| `UserPromptExpansion` | UserPromptExpansionHookInput | expansion_type: 'slash_command' \| 'mcp_prompt'; command_name: string; command_args: string; command_source?: string; prompt: string | UserPromptExpansionHookSpecificOutput |
| `SessionStart` | SessionStartHookInput | source: 'startup' \| 'resume' \| 'clear' \| 'compact' \| 'fork'; model?: string; session_title?: string; seconds_since_last_response?: number; context_tokens?: number; prompt_cache_likely_expired?: boolean; estimated_cache_write_usd?: number | SessionStartHookSpecificOutput |
| `SessionEnd` | SessionEndHookInput | reason: ExitReason | — |
| `Stop` | StopHookInput | stop_hook_active: boolean; last_assistant_message?: string; background_tasks?: BackgroundTaskSummary[]; session_crons?: SessionCronSummary[] | StopHookSpecificOutput |
| `StopFailure` | StopFailureHookInput | error: SDKAssistantMessageError; error_details?: string; last_assistant_message?: string | — |
| `SubagentStart` | SubagentStartHookInput |  | SubagentStartHookSpecificOutput |
| `SubagentStop` | SubagentStopHookInput | stop_hook_active: boolean; agent_transcript_path: string; last_assistant_message?: string; background_tasks?: BackgroundTaskSummary[]; session_crons?: SessionCronSummary[] | SubagentStopHookSpecificOutput |
| `PreCompact` | PreCompactHookInput | trigger: 'manual' \| 'auto'; custom_instructions: string \| null | — |
| `PostCompact` | PostCompactHookInput | trigger: 'manual' \| 'auto'; compact_summary: string | — |
| `PreModelSwitch` | PreModelSwitchHookInput | from_model: string; to_model: string; requested_model: string \| null; source: 'command' \| 'picker' \| 'sdk'; context_tokens: number; prompt_cache_warm: boolean; cache_ttl: '5m' \| '1h'; estimated_cache_write_usd: number; pricing: 'configured' \| 'catalog' \| 'default' | PreModelSwitchHookSpecificOutput |
| `PostModelSwitch` | PostModelSwitchHookInput | from_model: string; to_model: string; requested_model: string \| null; source: 'command' \| 'picker' \| 'sdk' \| 'auto' \| 'resume'; context_tokens: number; prompt_cache_warm: boolean; cache_ttl: '5m' \| '1h'; estimated_cache_write_usd: number; pricing: 'configured' \| 'catalog' \| 'default' | PostModelSwitchHookSpecificOutput |
| `PermissionRequest` | PermissionRequestHookInput | tool_name: string; tool_input: unknown; permission_suggestions?: PermissionUpdate[]; mcp_server?: McpServerProvenance | PermissionRequestHookSpecificOutput |
| `Setup` | SetupHookInput | trigger: 'init' \| 'maintenance' | SetupHookSpecificOutput |
| `TeammateIdle` | TeammateIdleHookInput | teammate_name: string; team_name: string | — |
| `TaskCreated` | TaskCreatedHookInput | task_id: string; task_subject: string; task_description?: string; teammate_name?: string; team_name?: string | — |
| `TaskCompleted` | TaskCompletedHookInput | task_id: string; task_subject: string; task_description?: string; teammate_name?: string; team_name?: string | — |
| `Elicitation` | ElicitationHookInput | mcp_server_name: string; message: string; mode?: 'form' \| 'url'; url?: string; elicitation_id?: string; requested_schema?: Record<string, unknown> | ElicitationHookSpecificOutput |
| `ElicitationResult` | ElicitationResultHookInput | mcp_server_name: string; elicitation_id?: string; mode?: 'form' \| 'url'; action: 'accept' \| 'decline' \| 'cancel'; content?: Record<string, unknown> | ElicitationResultHookSpecificOutput |
| `ConfigChange` | ConfigChangeHookInput | source: 'user_settings' \| 'project_settings' \| 'local_settings' \| 'p; file_path?: string | — |
| `InstructionsLoaded` | InstructionsLoadedHookInput | file_path: string; memory_type: 'User' \| 'Project' \| 'Local' \| 'Managed'; load_reason: 'session_start' \| 'nested_traversal' \| 'path_glob_match' \| '; globs?: string[]; trigger_file_path?: string; parent_file_path?: string | — |
| `WorktreeCreate` | WorktreeCreateHookInput | name: string | WorktreeCreateHookSpecificOutput |
| `WorktreeRemove` | WorktreeRemoveHookInput | worktree_path: string | — |
| `CwdChanged` | CwdChangedHookInput | old_cwd: string; new_cwd: string | CwdChangedHookSpecificOutput |
| `FileChanged` | FileChangedHookInput | file_path: string; event: 'change' \| 'add' \| 'unlink' | FileChangedHookSpecificOutput |
| `DirectoryAdded` | DirectoryAddedHookInput | directory: string; source: 'slash_command' \| 'register_repo_root' | — |
| `MessageDisplay` | MessageDisplayHookInput | turn_id: string; message_id: string; index: number; final: boolean; delta: string | MessageDisplayHookSpecificOutput |


**HookJSONOutput** = `AsyncHookJSONOutput | SyncHookJSONOutput`. Go `HookJSONOutput` is a union discriminated by the boolean `async`.

**AsyncHookJSONOutput**

| Field | Type | Req | Notes |
|---|---|---|---|
| `async` | `true` | req |  |
| `asyncTimeout` | `number` | opt |  |


**SyncHookJSONOutput**

| Field | Type | Req | Notes |
|---|---|---|---|
| `continue` | `boolean` | opt |  |
| `suppressOutput` | `boolean` | opt |  |
| `stopReason` | `string` | opt |  |
| `decision` | `'approve' \| 'block'` | opt |  |
| `systemMessage` | `string` | opt |  |
| `terminalSequence` | `string` | opt | A terminal escape sequence (e.g. |
| `reason` | `string` | opt |  |
| `hookSpecificOutput` | `PreToolUseHookSpecificOutput \| UserPromptSubmitHookSpecificOutput \| UserPromptExpansionHookSpecificOutput \|…` | opt |  |


`hookSpecificOutput` is a union keyed by `hookEventName`; Go `SyncHookJSONOutputHookSpecificOutput`. The event-specific outputs:

| Type | Fields |
|---|---|
| `CwdChangedHookSpecificOutput` | hookEventName, watchPaths? |
| `ElicitationHookSpecificOutput` | hookEventName, action?, content? |
| `ElicitationResultHookSpecificOutput` | hookEventName, action?, content? |
| `FileChangedHookSpecificOutput` | hookEventName, watchPaths? |
| `MessageDisplayHookSpecificOutput` | hookEventName, displayContent? |
| `NotificationHookSpecificOutput` | hookEventName, additionalContext? |
| `PermissionDeniedHookSpecificOutput` | hookEventName, retry? |
| `PermissionRequestHookSpecificOutput` | hookEventName, decision |
| `PostModelSwitchHookSpecificOutput` | hookEventName, additionalContext? |
| `PostToolBatchHookSpecificOutput` | hookEventName, additionalContext? |
| `PostToolUseFailureHookSpecificOutput` | hookEventName, additionalContext? |
| `PostToolUseHookSpecificOutput` | hookEventName, additionalContext?, classifierContext?, updatedToolOutput?, updatedMCPToolOutput? |
| `PreModelSwitchHookSpecificOutput` | hookEventName, permissionDecision?, permissionDecisionReason? |
| `PreToolUseHookSpecificOutput` | hookEventName, permissionDecision?, permissionDecisionReason?, updatedInput?, additionalContext? |
| `SessionStartHookSpecificOutput` | hookEventName, additionalContext?, initialUserMessage?, sessionTitle?, watchPaths?, reloadSkills? |
| `SetupHookSpecificOutput` | hookEventName, additionalContext? |
| `StopHookSpecificOutput` | hookEventName, additionalContext? |
| `SubagentStartHookSpecificOutput` | hookEventName, additionalContext? |
| `SubagentStopHookSpecificOutput` | hookEventName, additionalContext? |
| `UserPromptExpansionHookSpecificOutput` | hookEventName, additionalContext?, suppressOriginalPrompt? |
| `UserPromptSubmitHookSpecificOutput` | hookEventName, additionalContext?, sessionTitle?, suppressOriginalPrompt? |
| `WorktreeCreateHookSpecificOutput` | hookEventName, worktreePath |


**Hook registration on the wire** (`initialize.hooks`): `Partial<Record<HookEvent, SDKHookCallbackMatcher[]>>`. Go uses `map[HookEvent][]HookCallbackMatcher`.

| Field | Type | Req | Notes |
|---|---|---|---|
| `matcher` | `string` | opt |  |
| `hookCallbackIds` | `string[]` | req | Opaque ids chosen by the client, one per hook function it registered for this matcher. |
| `timeout` | `number` | opt |  |
| `onFailure` | `'continue' \| 'block'` | opt | **since 0.3.317 (not in 0.3.311).** What a failure of these callbacks does: the client answered the hook_callback request with an error, or with an answer that fails validation. |


---

## 7. MCP, elicitation, user messages, agents, initialize pieces

### 7.1 Elicitation

**Request (CLI → host)**, `elicitation`, Go `ElicitationControlRequest`:

| Field | Type | Req | Notes |
|---|---|---|---|
| `subtype` | `'elicitation'` | req |  |
| `mcp_server_name` | `string` | req |  |
| `message` | `string` | req |  |
| `mode` | `'form' \| 'url'` | opt |  |
| `url` | `string` | opt |  |
| `elicitation_id` | `string` | opt |  |
| `requested_schema` | `Record<string, unknown>` | opt |  |
| `property_order` | `string[]` | opt | requested_schema.properties' top-level keys in the order the MCP server listed them; absent for URL mode and when the schema has no properties object. |
| `title` | `string` | opt | Permission-display title from the MCP server's _meta['anthropic/permissionDisplay']. |
| `display_name` | `string` | opt | Short tool/server label from _meta['anthropic/permissionDisplay'].displayName. |
| `description` | `string` | opt | Permission-display subtitle from _meta['anthropic/permissionDisplay'].description. |


The SDK maps it to the host-facing `ElicitationRequest`: `serverName, message, mode?, url?, elicitationId?, requestedSchema?, title?, displayName?, description?`. It calls `onElicitation(request, {signal, requestId})`.

**Response:** `ElicitResult` `{action: 'accept'|'decline'|'cancel', content?: Record<string, string|number|boolean|string[]>, _meta?}`. With no handler the SDK answers `{action:"decline"}`; `null` means answered out-of-band. Completion of a URL-mode elicitation arrives as `system/elicitation_complete`.

### 7.2 McpServerConfig variants

`McpServerConfig` is `McpStdioServerConfig | McpSSEServerConfig | McpHttpServerConfig | McpSdkServerConfigWithInstance`.

`McpServerConfigForProcessTransport` (the wire form, used by `--mcp-config`, `mcp_set_servers` and `AgentDefinition.mcpServers`) has `McpSdkServerConfig` instead of the instance form. `McpServerStatusConfig` adds `McpClaudeAIProxyServerConfig`.

In Go, `McpServerConfig` is the wire union keyed by `type`; a missing `type` means stdio.

**McpStdioServerConfig**

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'stdio'` | opt |  |
| `command` | `string` | req |  |
| `args` | `string[]` | opt |  |
| `env` | `Record<string, string>` | opt |  |
| `timeout` | `number` | opt | Per-server tool-call timeout in milliseconds. |
| `alwaysLoad` | `boolean` | opt | When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself… |


**McpSSEServerConfig**

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'sse'` | req |  |
| `url` | `string` | req |  |
| `headers` | `Record<string, string>` | opt |  |
| `tools` | `McpServerToolPolicy[]` | opt |  |
| `timeout` | `number` | opt | Per-server tool-call timeout in milliseconds. |
| `alwaysLoad` | `boolean` | opt | When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself… |


**McpHttpServerConfig**

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'http'` | req |  |
| `url` | `string` | req |  |
| `headers` | `Record<string, string>` | opt |  |
| `tools` | `McpServerToolPolicy[]` | opt |  |
| `timeout` | `number` | opt | Per-server tool-call timeout in milliseconds. |
| `alwaysLoad` | `boolean` | opt | When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself… |


**McpSdkServerConfig**

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'sdk'` | req |  |
| `name` | `string` | req |  |
| `timeout` | `number` | opt | Per-server tool-call timeout in milliseconds. |
| `tools` | `McpServerToolPolicy[]` | opt | Per-tool permission policy the host declares for this server's tools (not the tool definitions passed to createSdkMcpServer). always_ask: the CLI… |


**McpClaudeAIProxyServerConfig**

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'claudeai-proxy'` | req |  |
| `url` | `string` | req |  |
| `id` | `string` | req |  |
| `timeout` | `number` | opt | Per-server tool-call timeout in milliseconds. |


**McpServerToolPolicy**

| Field | Type | Req | Notes |
|---|---|---|---|
| `name` | `string` | req |  |
| `permission_policy` | `'always_allow' \| 'always_ask' \| 'always_deny'` | opt |  |
| `org_max_permission` | `'allow' \| 'ask' \| 'blocked'` | opt | Org admin's per-tool ceiling. |


**McpServerStatus**

| Field | Type | Req | Notes |
|---|---|---|---|
| `name` | `string` | req | Server name as configured |
| `status` | `'connected' \| 'failed' \| 'needs-auth' \| 'pending' \| 'disabled'` | req | Current connection status |
| `serverInfo` | {…} | opt | Server information (available when connected) |
| `serverInfo.name` | `string` | req |  |
| `serverInfo.version` | `string` | req |  |
| `error` | `string` | opt | Error message (available when status is 'failed') |
| `config` | `McpServerStatusConfig` | opt | Server configuration (includes URL for HTTP/SSE servers) |
| `scope` | `string` | opt | Configuration scope (e.g., project, user, local, claudeai, managed) |
| `source` | `string` | opt | Where the server definition came from: sdk (an in-process server the SDK host registered — only the host can register one), plugin (a server a… |
| `tools` | {…}[] | opt | Tools provided by this server (available when connected) |
| `tools[].name` | `string` | req |  |
| `tools[].description` | `string` | opt |  |
| `tools[].annotations` | {…} | opt |  |
| `tools[].annotations.readOnly` | `boolean` | opt |  |
| `tools[].annotations.destructive` | `boolean` | opt |  |
| `tools[].annotations.openWorld` | `boolean` | opt |  |
| `tools[]._meta` | `Record<string, unknown>` | opt | The MCP Apps (SEP-1865) members of the tool's `_meta`, for a host that renders the tool's `ui://` resource, under the keys the server used: `ui`… |


**McpSetServersResult**

| Field | Type | Req | Notes |
|---|---|---|---|
| `added` | `string[]` | req | Names of servers that were added |
| `removed` | `string[]` | req | Names of servers that were removed |
| `errors` | `Record<string, string>` | req | Map of server names to error messages for servers that failed to connect |


### 7.3 SDKUserMessage (host → CLI on stdin; also echoed back)

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'user'` | req |  |
| `message` | `MessageParam` | req | An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image,… |
| `parent_tool_use_id` | `string \| null` | req |  |
| `isSynthetic` | `boolean` | opt |  |
| `tool_use_result` | `unknown` | opt | Structured tool output — the tool's full Output object, not the string content sent to the model. |
| `priority` | `'now' \| 'next' \| 'later'` | opt |  |
| `origin` | `SDKMessageOrigin` | opt |  |
| `shouldQuery` | `boolean` | opt | When false, the message is appended to the transcript without triggering an assistant turn. |
| `timestamp` | `string` | opt | ISO timestamp when the message was created on the originating process. |
| `client_composed` | `true` | opt | The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and… |
| `uuid` | `UUID` | opt |  |
| `pasted_content` | `MessageParam['content'][]` | opt | Content the user pasted into the prompt rather than typed: each entry a string or an array of content blocks. |
| `inline_pastes` | `string[]` | opt | Text the user pasted that is still in `message.content` where they put it: each entry one paste. |
| `session_id` | `string` | opt |  |
| `agent_id` | `string` | opt | ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed. |
| `subagent_type` | `string` | opt | Subagent type that produced this message. |
| `task_description` | `string` | opt | Description of the subagent task that produced this message. |


`SDKUserMessageReplay` has the same fields, with `uuid` and `session_id` required, plus `isReplay: true` and `file_attachments?`.

### 7.4 AgentDefinition (`Options.agents` / `initialize.agents`)

| Field | Type | Req | Notes |
|---|---|---|---|
| `description` | `string` | req | Natural language description of when to use this agent |
| `tools` | `string[]` | opt | Array of allowed tool names. |
| `disallowedTools` | `string[]` | opt | Array of tool names to explicitly disallow for this agent. |
| `prompt` | `string` | req | The agent's system prompt |
| `model` | `string` | opt | Model alias (e.g. |
| `mcpServers` | `AgentMcpServerSpec[]` | opt |  |
| `criticalSystemReminder_EXPERIMENTAL` | `string` | opt | Experimental: Critical reminder added to system prompt |
| `skills` | `string[]` | opt | Array of skill names to preload into the agent context |
| `initialPrompt` | `string` | opt | Auto-submitted as the first user turn when this agent is the main thread agent. |
| `maxTurns` | `number` | opt | Maximum number of agentic turns (API round-trips) before stopping |
| `autoCompactWindow` | `number` | opt | Token count at which this agent compacts its own conversation when it runs as a subagent. |
| `background` | `boolean` | opt | Run this agent as a background task (non-blocking, fire-and-forget) when invoked |
| `omitClaudeMd` | `boolean` | opt | Run this agent without the user, project and local CLAUDE.md instruction files when it runs as a subagent; managed policy files are kept. |
| `memory` | `'user' \| 'project' \| 'local'` | opt | Scope for auto-loading agent memory files. |
| `effort` | `('low' \| 'medium' \| 'high' \| 'xhigh' \| 'max') \| number` | opt | Reasoning effort level for this agent. |
| `permissionMode` | `PermissionMode` | opt | Permission mode controlling how tool executions are handled |
| `observer` | `string` | opt | Agent type auto-spawned as a background observer whenever this agent runs. |
| `observerMessage` | `string` | opt | Supplemental postamble appended (after the harness-owned default) to each activity digest sent to the observer. |


`AgentMcpServerSpec = string | Record<string, McpServerConfigForProcessTransport>`; Go keeps it as raw JSON.

### 7.5 SettingSource

`'user' | 'project' | 'local'`. `ConfigScope` is the same set; `ResolvedSettingSource` adds `'managed' | 'flag'`.

### 7.6 initialize

**Request** (host → CLI, always the first control request), Go `InitializeRequest`. sdk.mjs 0.3.311 also sends the undeclared keys `appendSubagentSystemPrompt`, `codeRunningMcpTools`, `webSearchIsolationExemptMcpServers`, `rapidFollowupPreempt`, `forwardNestedSubagentToolCalls` and `workspaceTrust` when set.

| Field | Type | Req | Notes |
|---|---|---|---|
| `subtype` | `'initialize'` | req |  |
| `hooks` | `Partial<Record<HookEvent, SDKHookCallbackMatcher[]>>` | opt |  |
| `sdkMcpServers` | `string[]` | opt |  |
| `sdkMcpServerConfigs` | `Record<string, { timeout?: number; tools?: McpServerToolPolicy[]; }>` | opt | Settings for the SDK-hosted MCP servers named in sdkMcpServers, keyed by server name. |
| `sdkMcpServerManifests` | `Record<string, { initializeResult: Record<string, unknown>; toolsListResult?: Record<string, unknown>; }>` | opt | Optional, keyed by sdk server name (each key should also appear in sdkMcpServers; other keys are ignored). |
| `sdkMcpServerManifestsOrigin` | `'host_per_send'` | opt | The host's statement that it composed this request's sdkMcpServerManifests for this very send, from its servers' current state. |
| `jsonSchema` | `Record<string, unknown>` | opt |  |
| `systemPrompt` | `string[]` | opt |  |
| `appendSystemPrompt` | `string` | opt |  |
| `systemPromptSnapshot` | `boolean` | opt | Record the conversation's system prompt once and reuse it verbatim on every later request and resume. |
| `planModeInstructions` | `string` | opt | Custom workflow body for the plan-mode system reminder. |
| `toolAliases` | `Record<string, string>` | opt | Map of tool-name aliases applied before name resolution. |
| `excludeDynamicSections` | `boolean` | opt | When true, omit per-user dynamic sections (working directory, auto-memory path) from the cached system prompt and re-inject them as the first user… |
| `agents` | `Record<string, AgentDefinition>` | opt |  |
| `title` | `string` | opt | Custom session title. |
| `skills` | `string[]` | opt | When provided, only skills whose names match an entry are loaded into the main session system prompt, matching the exact canonical name (e.g. |
| `promptSuggestions` | `boolean` | opt |  |
| `agentProgressSummaries` | `boolean` | opt |  |
| `forwardSubagentText` | `boolean` | opt |  |
| `supportedDialogKinds` | `string[]` | opt | Dialog kinds (request_user_dialog `dialog_kind` values) this consumer's onUserDialog can actually render. |
| `perTaskStopAffordance` | `boolean` | opt | Declares that this consumer renders a per-task stop control wired to the `stop_task` control request, so the user can stop an individual… |
| `plugins` | `SdkPluginConfig[]` | opt | Plugins to load for the session, in the same shape as the SDK `plugins` option: the stdin form of one --plugin-dir flag per entry… |


**Response**, Go `InitializeResponse`:

| Field | Type | Req | Notes |
|---|---|---|---|
| `commands` | `SlashCommand[]` | req |  |
| `agents` | `AgentInfo[]` | req |  |
| `output_style` | `string` | req |  |
| `available_output_styles` | `string[]` | req |  |
| `models` | `ModelInfo[]` | req |  |
| `account` | `AccountInfo` | req | Information about the logged in user's account. |
| `hooks_applied` | `boolean` | opt | Whether the `hooks` this initialize carried were registered: true on a session's first initialize, and on a repeated initialize from the process… |
| `plugins_applied` | `boolean` | opt | Whether every plugin this initialize listed is loaded: true when each one is among the plugins the process loaded at launch (from `plugins` under… |
| `sdk_mcp_manifests_parked` | `Record<string, 'parked' \| 'already_connected' \| 'protocol_version_mismatch' \| 'malformed' \| 'not_honoured'>` | opt | What became of each entry of this initialize's sdkMcpServerManifests, keyed by sdk server name, for the entries whose name is in sdkMcpServers. |
| `claude_code_version` | `string` | opt | Claude Code version of the process that runs this session's turns (the same value its system/init carries). |
| `fast_mode_state` | `FastModeState` | opt |  |
| `fast_mode_disabled_reason` | `FastModeDisabledReason` | opt |  |
| `hook_on_failure_supported` | `boolean` | opt | **since 0.3.317 (not in 0.3.311).** True on every initialize reply from a CLI that honors `onFailure` on hook matchers. |


**ModelInfo**

| Field | Type | Req | Notes |
|---|---|---|---|
| `value` | `string` | req | Model identifier to use in API calls |
| `resolvedModel` | `string` | opt | Canonical wire model id this row's `value` resolves to (e.g. |
| `displayName` | `string` | req | Human-readable display name |
| `description` | `string` | req | Description of the model's capabilities |
| `supportsEffort` | `boolean` | opt | Whether this model supports effort levels |
| `supportedEffortLevels` | `('low' \| 'medium' \| 'high' \| 'xhigh' \| 'max')[]` | opt | Available effort levels for this model |
| `supportsAdaptiveThinking` | `boolean` | opt | Whether this model supports adaptive thinking (Claude decides when and how much to think) |
| `supportsFastMode` | `boolean` | opt | Whether this model supports fast mode |
| `supportsAutoMode` | `boolean` | opt | Whether this model supports auto mode |


**SlashCommand**

| Field | Type | Req | Notes |
|---|---|---|---|
| `name` | `string` | req | Skill name (without the leading slash) |
| `description` | `string` | req | Description of what the skill does |
| `argumentHint` | `string` | req | Hint for skill arguments (e.g., "<file>") |
| `aliases` | `string[]` | opt | Alternate names that resolve to this command (e.g., /cost and /stats both resolve to /usage) |
| `builtin` | `boolean` | opt | True when the command is Claude Code's own; absent for a command defined by a user, project, plugin or MCP server. |


**AccountInfo**

| Field | Type | Req | Notes |
|---|---|---|---|
| `email` | `string` | opt |  |
| `organization` | `string` | opt |  |
| `subscriptionType` | `string` | opt |  |
| `tokenSource` | `string` | opt |  |
| `apiKeySource` | `string` | opt |  |
| `apiProvider` | `'firstParty' \| 'bedrock' \| 'vertex' \| 'foundry' \| 'anthropicAws' \| 'anthropicGoogleCloud' \| 'mantle' \| 'gat…` | opt | Active API backend. |


**AgentInfo**

| Field | Type | Req | Notes |
|---|---|---|---|
| `name` | `string` | req | Agent type identifier (e.g., "Explore") |
| `description` | `string` | req | Description of when to use this agent |
| `model` | `string` | opt | Model this agent uses: an alias or model ID, or 'inherit' for the parent's model. |


**SdkPluginConfig**

| Field | Type | Req | Notes |
|---|---|---|---|
| `type` | `'local'` | req | Plugin type. |
| `path` | `string` | req | Absolute or relative path to the plugin directory |
| `skipMcpDiscovery` | `boolean` | opt | When true, the engine loads skills/hooks/agents/commands from this plugin but does NOT read its .mcp.json or manifest mcpServers. |


---

## 8. Exported functions (0.3.311)

The paths below assume the local filesystem; with `sessionStore` every function goes through the store API instead. The session-helper behaviour comes from reading the minified sdk.mjs. It was traced in 0.3.325, and its key constants are confirmed present in 0.3.311. Differences from 0.3.311 are noted.

| Export | Signature | What it does |
|---|---|---|
| `query` | query({prompt: string \| AsyncIterable<SDKUserMessage>, options?: Options}): Query | Spawns the CLI (section 2) and writes `initialize` first, then the prompt. Returns the Query generator (section 1). |
| `startup` | startup({options?, initializeTimeoutMs?}?): Promise<WarmQuery> | Spawns the CLI and waits for the initialize response. Returns a WarmQuery whose single `query(prompt)` writes the prompt; `close()` kills it. |
| `prewarm *(alpha)*` | prewarm({options?, initializeTimeoutMs?}?): Promise<SpareProcess> | Starts a parked spare with `--await-claim` (cwd defaults to a mkdtemp under `<config>/spares/`; resume, continue and forkSession are rejected). `claim({prompt, options: ClaimOptions})` sends `claim_session {cwd, permission_mode?, env?, additional_directories?, append_system_prompt?, title?, agents?, sdk_mcp_servers, include_initialize:true}`, then the settings overlay and the prompt. |
| `tool` | tool(name, description, inputSchema: ZodRawShape, handler, extras?: {annotations?, searchHint?, alwaysLoad?}): SdkMcpToolDefinition | Returns `{name, description, inputSchema, handler, annotations, _meta}`. `_meta` holds `"anthropic/searchHint"` and/or `"anthropic/alwaysLoad": true`. |
| `createSdkMcpServer` | createSdkMcpServer({name, version?, instructions?, tools?, alwaysLoad?, timeout?}): McpSdkServerConfigWithInstance | Creates an in-process `McpServer({name, version ?? "1.0.0"}, {capabilities: {tools}, instructions})` and registers each tool. The zod schema becomes JSON Schema draft-7 (input side). Server-level alwaysLoad stamps `anthropic/alwaysLoad` on every tool. Returns `{type:"sdk", name, instance, timeout?}`. |
| `listSessions` | listSessions({dir?, limit?, offset?, includeWorktrees?, includeProgrammatic?, sessionStore?}?): Promise<SDKSessionInfo[]> | Scans `<CLAUDE_CONFIG_DIR or ~/.claude>/projects/<sanitized dir>/*.jsonl` (all projects when `dir` is omitted; with `dir`, also its git worktrees via `git worktree list --porcelain`). Reads a 64 KiB head and a 64 KiB tail of each file, skips sidechains and sessions continued in another file, and derives the SessionInfo fields. Sorts by lastModified desc and dedupes by sessionId. See 8.1. |
| `getSessionInfo` | getSessionInfo(sessionId, {dir?, sessionStore?}?): Promise<SDKSessionInfo \| undefined> | Same derivation for a single file. Undefined if the file is missing, a sidechain, or has no summary. |
| `getSessionMessages` | getSessionMessages(sessionId, {dir?, limit?, offset?, includeSystemMessages?, sessionStore?}?): Promise<SessionMessage[]> | Parses the JSONL and rebuilds the main conversation chain through `parentUuid` from the newest main-chain leaf. Returns user/assistant entries in order (system ones only with includeSystemMessages), then applies offset/limit. Files over 5 MiB skip everything before the last plain compact boundary. |
| `getSubagentMessages` | getSubagentMessages(sessionId, agentId, {dir?, limit?, offset?, sessionStore?}?): Promise<SessionMessage[]> | Reads `<projdir>/<sessionId>/subagents/agent-<agentId>.jsonl`, plus the `agent-<id>.meta.json` sidecar for `parent_tool_use_id` and `parent_agent_id`. |
| `listSubagents` | listSubagents(sessionId, {dir?, sessionStore?}?): Promise<string[]> | Returns the ids of `agent-<id>.jsonl` files under `<sessionId>/subagents/`, recursively. |
| `renameSession` | renameSession(sessionId, title, {dir?, sessionStore?}?): Promise<void> | Appends `{"type":"custom-title","customTitle":…,"sessionId":…}` to the transcript. |
| `tagSession` | tagSession(sessionId, tag \| null, opts?): Promise<void> | Appends `{"type":"tag","tag":…,"sessionId":…}`; null writes `""` and clears the tag. Non-ASCII tags are NFKC-normalized and stripped of format characters. |
| `deleteSession` | deleteSession(sessionId, opts?): Promise<void> | Removes `<id>.jsonl`, then the `<id>/` directory. |
| `forkSession` | forkSession(sessionId, {dir?, upToMessageId?, title?, sessionStore?}?): Promise<{sessionId}> | Copies the transcript to `<newId>.jsonl` with fresh UUIDs and a remapped parentUuid chain, optionally cut at upToMessageId. Each entry gets `forkedFrom`. Appends a custom-title entry; the default title is `"<title> (fork)"`. |
| `importSessionToStore *(alpha)*` | importSessionToStore(sessionId, store, {dir?, includeSubagents?, batchSize?}?) | Streams a local transcript into `store.append` in batches. |
| `resolveSettings / filterEscalatingDefaultMode *(alpha)*` | resolveSettings(opts?): Promise<ResolvedSettings> | Runs the CLI settings-merge engine in process, without spawning the CLI. |
| `foldSessionSummary *(alpha)*` | (prev, key, entries, {mtime?}) => SessionSummaryEntry | Helper for SessionStore implementations. |
| `InMemorySessionStore *(alpha)*, AbortError` | classes | A test store, and the error thrown on abort. |
| `Constants` | HOOK_EVENTS, EXIT_REASONS, BUILTIN_TOOL_NAMES *(alpha)*, LEGACY_TOOL_NAME_ALIASES *(alpha)*, SYSTEM_PROMPT_DYNAMIC_BOUNDARY = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__", USAGE_LIMIT_ERROR_PREFIXES / USAGE_TRANSITION_PREFIXES / USAGE_WARNING_PREFIXES / ORG_POLICY_LIMIT_PREFIXES *(alpha)* | Lists. 0.3.325 adds "Now using credits" to USAGE_TRANSITION_PREFIXES. |


The runtime bundle also exports names that the d.ts does not declare: `CommandMatcher`, `DirectConnectError`, `DirectConnectTransport`, `FileIndex` and `parseDirectConnectUrl` (seen in 0.3.325).

### 8.1 Session files: details for a Go port

**Project directory name**
- Start from `realpath(dir)`, NFC-normalized on macOS.
- Replace every character outside `[a-zA-Z0-9]` with `-`.
- If the result is longer than 200 characters, keep the first 200 and append `"-" + base36(|javaHash(path)|)`. The hash is the Java-style 32-bit `h = 31*h + utf16 code unit`.
- Readers also accept any directory starting with that 200-character prefix plus `-` whose transcript's `cwd` sanitizes to the same name.
- When `CLAUDE_CONFIG_DIR` is set, `CLAUDE_CODE_PROJECT_DIR_NAME` (matching `^[A-Za-z0-9_-]{1,64}$`) overrides the name.

**Session files**
- A session file is `<uuid>.jsonl`.
- A file whose first line has `"isSidechain":true` is a subagent transcript and is skipped.
- Empty files count as missing.

**SessionInfo fields.** Each comes from the 64 KiB head and tail windows:
- `customTitle`: the last `{type:"custom-title"}` entry in the tail, else the sidecar `<id>/custom-title.json`, else the same entry in the head.
  - 0.3.325 also falls back to `ai-title` entries; 0.3.311 does not.
- `summary`: `customTitle`, else the last `"lastPrompt"` value, else the last `"summary"` value, else `firstPrompt`.
- `firstPrompt`: the first real user text in the head.
  - Lines with `tool_result`, `isMeta` or `isCompactSummary` are skipped, as are texts that start with an XML-like tag or "[Request interrupted by user".
  - `<command-name>` and `<bash-input>` are special-cased.
  - The text is capped at 200 characters plus `…`.
- `gitBranch`, `cwd` and `createdAt` come from the first or last raw `"key":"value"` occurrence. For `createdAt` that is the first `timestamp`.
- `tag`: the last `{type:"tag"}` entry in the tail.
- `lastModified`: the file mtime in milliseconds.
- `fileSize`: the file size in bytes.

**Filtering.** `includeProgrammatic:false` drops sessions whose `entrypoint` is `sdk-cli`, `sdk-ts` or `sdk-py`, or whose `sessionKind` is `daemon` or `daemon-worker`.

**Pagination.** With limit or offset, files are sorted by mtime before parsing and evaluated in batches of 32.

---

## 9. Not in 0.3.311

These items appear in 0.3.325 but not in 0.3.311. Every one is additive. `types.go` keeps each of them, marked `// since … (not in 0.3.311)`. A 2.1.311 CLI never sends them, and a 0.3.311 host must not rely on them.

| Kind | Item | Since |
|---|---|---|
| `field` | HookCallbackMatcher.onFailure | 0.3.317 |
| `field` | McpCallRequest.check_permissions | 0.3.312-0.3.325 |
| `field` | ReloadPluginsRequest.refuse_if_busy | 0.3.325 |
| `type` | SDKControlSetHasSubscribersRequest | 0.3.312-0.3.325 |
| `union variant` | ControlRequestInner.SDKControlSetHasSubscribersRequest | 0.3.312-0.3.325 |
| `field` | ControlErrorResponse.pending_elicitation_requests | 0.3.312-0.3.325 |
| `field` | ControlSuccessResponse.pending_elicitation_requests | 0.3.312-0.3.325 |
| `type` | DeniedModelsMatch | 0.3.323 |
| `field` | ResolvedSettings.machinePolicy | 0.3.323 |
| `field` | InitializeResponse.hook_on_failure_supported | 0.3.317 |
| `field` | PermissionRulesState.withheld | 0.3.312-0.3.325 |
| `field` | ReadFileResponse.lossy | 0.3.317 |
| `field` | ResultSuccess.credential_expires_at | 0.3.318 |
| `enum value` | StartupFailureReason.sandbox_required_unavailable | 0.3.314 |
| `enum value` | StartupFailureReason.initialize_not_first | 0.3.317 |
| `enum value` | StartupFailureReason.config_file_invalid | 0.3.318 |
| `enum value` | StartupFailureReason.managed_settings_not_delivered | 0.3.312-0.3.325 |
| `field` | ResultError.invalid_config_file | 0.3.325 |
| `field` | ResultError.credential_expires_at | 0.3.318 |
| `field` | SystemInitMessagePluginError.managed_lock | 0.3.325 |
| `type` | SDKSessionWritesLostMessage | 0.3.317 |
| `union variant` | Message.SDKSessionWritesLostMessage | 0.3.317 |
| `field` | SessionInfo.archived | 0.3.325 |
| `union variant` | StdoutMessage.SDKSessionWritesLostMessage | 0.3.317 |


**Grouped by surface**

- **Message variants:** `system:session_writes_lost` (`SDKSessionWritesLostMessage`), since 0.3.317.
- **Message fields:**
  - `credential_expires_at` on all result messages (0.3.318)
  - `invalid_config_file` on error results (0.3.325)
  - `plugin_errors[].managed_lock` on `system:init` (0.3.325)
- **Control subtypes:** `set_has_subscribers` (host → CLI; `has_subscribers: boolean | null`), 0.3.312-0.3.325.
- **Control request fields:**
  - `mcp_call.check_permissions` (0.3.312-0.3.325)
  - `reload_plugins.refuse_if_busy` (0.3.325)
  - `initialize.hooks[Event][].onFailure` (0.3.317)
- **Control response fields:**
  - `initialize` → `hook_on_failure_supported` (0.3.317)
  - `read_file` → `lossy` (0.3.317)
  - `list_permission_rules` → `state.withheld` (0.3.312-0.3.325)
  - the control_response envelope → `pending_elicitation_requests` (0.3.312-0.3.325)
- **Enum values:** `SDKStartupFailureReason` gains `sandbox_required_unavailable` (0.3.314), `initialize_not_first` (0.3.317), `config_file_invalid` (0.3.318) and `managed_settings_not_delivered` (0.3.312-0.3.325).
- **Query methods:** `reloadPlugins({refuseIfBusy})` option (0.3.325). No method is new.
- **Options:** none new in the d.ts.
  - `HookCallbackMatcher.onFailure` (0.3.317) is new on the hook matcher.
  - With `onFailure:"block"`, sdk.mjs 0.3.325 checks `hook_on_failure_supported` and `hooks_applied` and fails the query if they are missing. 0.3.311 has neither.
- **Exports:** `archiveSession` and `unarchiveSession` (0.3.325), `matchDeniedModels` and `DeniedModelsMatch` (0.3.323). Also new: `SDKSessionInfo.archived` (0.3.325) and `ResolvedSettings.machinePolicy` (0.3.323).
- **Behaviour in sdk.mjs:**
  - 0.3.325 adds an internal `managedSettingsDelivery: 'initialize'` (`--managed-settings-in-initialize`, `initialize.managedSettings`).
  - It adds `--await-initialize` on resume or continue with the bundled binary (0.3.321).
  - It stops auto-approving sandboxed Bash unless `autoAllowBashIfSandboxed: true` (0.3.325).
  - It falls back to `ai-title` for session titles.
- **Settings keys:**
  - Added: `autoMode`, `disableReplyCheck`, `disableWindowsBashCommandSplit`, `skipAutoPermissionPrompt` and `useAutoModeDuringPlan`.
  - Added inside `permissions`: `blockWritesOutsideWorkingDirectories`, `blockReadsExemptPaths`, `workingDirectoriesAsGiven` and `disableAutoMode`.
  - Added on marketplace sources: `engines`.
  - Go keeps Settings as a map, so nothing changes there.
- **sdk-tools.d.ts** (tool input/output schemas; not modelled in types.go):
  - ArtifactInput `+page_token`; BashOutput `+backgroundWakesOwner`; CronDeleteOutput `+endedLoopWakeup`; GlobOutput and GrepOutput `+timedOut`.
  - Shape changes in ArtifactOutput, AskUserQuestionOutput.annotations, FileReadOutput, ProjectsOutput, ProposeSkillsInput.proposals, PushNotificationInput.status and TaskCreateOutput.task.

---

## 10. Go mapping notes

**Unions.** There is one pattern everywhere: a struct with the discriminator string(s) and `Raw`, plus `As<Variant>()`, `Decode()` and `New<Union>(v)`. This covers:
- `Message` and `StdoutMessage` (type, subtype)
- `ControlRequestInner` and `ControlResponseResponse` (subtype)
- `ContentBlock`, `StreamEvent` and `ContentBlockDelta` (type)
- `PermissionResult` and `UserDialogResult` (behavior)
- `PermissionUpdate`, `McpServerConfig`, `McpServerStatusConfig`, `ThinkingConfig` and `OutputFormat` (type)
- `MessageOrigin` (kind)
- `HookInput` (hook_event_name)
- the hookSpecificOutput union (hookEventName)
- `HookJSONOutput`, the one hand-written union, discriminated by the boolean `async`

**Lossless JSON.**
- Every wire struct keeps `Raw`. MarshalJSON merges back undeclared keys, and restores declared keys whose value is unchanged (explicit null, "", []).
- Decoding tolerates JSON type mismatches.
- The round trip was checked on 942 captured CLI frames (2.1.278, 2.1.311 and 2.1.325 runs, both directions) and on typed control-response payloads. No frame lost data.

**What does not map cleanly**
- **TS `number`** becomes `float64`. Fields that are integers by name become `int`: tokens, counts, indexes, bytes, sizes, turns and attempts. A fractional value in an `int` field decodes as 0 and keeps its original in Raw.
- **Mixed unions keep raw JSON:**
  - `string | string[] | {…}`: `systemPrompt`, `tools`, `settings`, `skills`, `MessageParam.content`, `AgentDefinition.effort`, `AgentMcpServerSpec`, tool_result content
  - `unknown`: `tool_input`, `tool_response`, `structured_output`, `tool_use_result`
  - Helpers: `MessageParam.Text()` and `Blocks()`.
- **`user` vs replay:** `SDKUserMessage` and `SDKUserMessageReplay` share `type:"user"`. Replay is detected by `isReplay:true`.
- **Results:** `SDKResultError` covers four subtypes in one struct.
- **Stdio type:** a `McpStdioServerConfig` with no `type` is stdio.
- **External types are hand-modelled:** BetaMessage, BetaRawMessageStreamEvent, MessageParam, BetaUsage, ElicitResult, CallToolResult, ToolAnnotations, and SandboxSettings (a zod schema).
- **Settings** (about 180 optional keys) is `map[string]any`.
- **Callbacks** cannot live in the types: canUseTool, hooks, onElicitation, onUserDialog, stderr, spawnClaudeCodeProcess, sessionStore, tool handlers, abortController.
  - Options carries booleans or callback ids instead.
  - The host answers the corresponding control requests (5.3–5.4, 7.1) with the response types here.
- **Generic and conditional TS types** (`applyFlagSettings`' mapped type, `InferShape`, zod schemas) have no Go equivalent: `map[string]any` or omitted.
- **Undeclared shapes** use Go types marked "observed" or "sdk.mjs": `error_code`, hidden initialize keys, and responses for mcp_status, background_tasks, list_models, get_session_cost, get_settings, set_permission_mode and set_mcp_permission_mode_override.
- **Undocumented responses:** get_binary_version, file_suggestions, mcp_call and register_repo_root have no declared or observed response shape. Read them as raw JSON.

