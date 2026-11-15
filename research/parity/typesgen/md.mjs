// Renders api-surface.md. Tables come from the 0.3.311 sdk.d.ts AST (with 0.3.325 additions marked);
// behaviour/wire mappings are hand-written from reading sdk.mjs 0.3.311.
// Usage: node md.mjs <ts-dir> <base sdk.d.ts> <newer sdk.d.ts> <extern.d.ts> <since.json> <options311.json> > api-surface.md
import { createRequire } from 'node:module';
import fs from 'node:fs';
const require = createRequire(process.argv[2] + '/');
const ts = require('typescript');
const [, , , baseFile, newFile, externFile, sinceFile, optFile] = process.argv;
const parse = (f) => ts.createSourceFile(f, fs.readFileSync(f, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
const sfBase = parse(baseFile), sfNew = parse(newFile), sfExt = parse(externFile);
const index = (sf) => { const m = new Map(); sf.forEachChild((n) => { if ((ts.isTypeAliasDeclaration(n) || ts.isInterfaceDeclaration(n)) && n.name && !m.has(n.name.text)) m.set(n.name.text, n); }); return m; };
const B = index(sfBase), N = index(sfNew), X = index(sfExt);
const sinceLog = JSON.parse(fs.readFileSync(sinceFile, 'utf8'));
const options = JSON.parse(fs.readFileSync(optFile, 'utf8'));
const refName = (t) => t.typeName.getText().replace(/^coreTypes\./, '');
const txt = (n) => n.getText().replace(/\/\*\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ').trim();
const strip = (t) => { while (t && ts.isParenthesizedTypeNode(t)) t = t.type; return t; };
const isNullish = (t) => { t = strip(t); return (ts.isLiteralTypeNode(t) && t.literal.kind === ts.SyntaxKind.NullKeyword) || t.kind === ts.SyntaxKind.UndefinedKeyword; };
const members = (t) => { t = strip(t); return ts.isUnionTypeNode(t) ? t.types.flatMap(members) : [t]; };
const pname = (p) => p.name.getText().replace(/^['"]|['"]$/g, '');
const look = (n, ctx) => { const sf = ctx?.getSourceFile?.(); return sf === sfNew ? (N.get(n) ?? B.get(n)) : sf === sfExt ? (X.get(n) ?? B.get(n)) : (B.get(n) ?? N.get(n) ?? X.get(n)); };
const SINCE_EXACT = Object.fromEntries(sinceLog.map((s) => [s.name, s.version]));
const sinceTxt = (k) => `since ${SINCE_EXACT[k] || '0.3.312-0.3.325'}`;
const RENAME = {
  SDKMessage: 'Message', SDKSystemMessage: 'SystemInitMessage', SDKPartialAssistantMessage: 'PartialAssistantMessage', SDKResultMessage: 'ResultMessage',
  SDKControlRequest: 'ControlRequest', SDKControlResponse: 'ControlResponse', SDKControlRequestInner: 'ControlRequestInner', ControlResponse: 'ControlSuccessResponse',
  ControlErrorResponse: 'ControlErrorResponse', SDKControlCancelRequest: 'ControlCancelRequest', SDKKeepAliveMessage: 'KeepAliveMessage',
  SDKControlPermissionRequest: 'CanUseToolRequest', SDKHookCallbackRequest: 'HookCallbackRequest', SDKHookCallbackMatcher: 'HookCallbackMatcher',
  SDKControlElicitationRequest: 'ElicitationControlRequest', SDKControlRequestUserDialogRequest: 'UserDialogControlRequest', SDKControlInitializeRequest: 'InitializeRequest',
  SDKControlInitializeResponse: 'InitializeResponse', SDKUserMessage: 'UserMessage', SDKUserMessageReplay: 'UserMessageReplay', SDKAssistantMessage: 'AssistantMessage',
  SDKSessionInfo: 'SessionInfo', SDKMessageOrigin: 'MessageOrigin', SDKStatus: 'Status', SdkBeta: 'Beta', SdkPluginConfig: 'PluginConfig', BetaMessage: 'APIMessage',
  BetaRawMessageStreamEvent: 'StreamEvent', NonNullableUsage: 'Usage', BetaContentBlock: 'ContentBlock', ElicitResult: 'ElicitationResult',
};
const goName = (n) => RENAME[n] || n.replace(/^SDKControl/, '').replace(/^SDK/, '').replace(/^Sdk/, '');
const esc = (s) => String(s ?? '').replace(/\\\|/g, '|').replace(/\|/g, '\\|').replace(/\n/g, ' ');
function docOf(node, max = 200) {
  const jd = node?.jsDoc?.at(-1);
  if (!jd) return '';
  let c = typeof jd.comment === 'string' ? jd.comment : (jd.comment || []).map((x) => x.text || '').join('');
  c = (c || '').replace(/\s+/g, ' ').trim();
  let first = c.split(/(?<=[.!?])\s(?=[A-Z`'"(])/)[0] || '';
  if (first.length > max) first = first.slice(0, max - 3).replace(/\s+\S*$/, '') + '…';
  for (const t of jd.tags || []) { if (t.tagName.text === 'deprecated') first = '**Deprecated.** ' + first; if (t.tagName.text === 'alpha') first += ' *(alpha)*'; }
  return first;
}
function propsOf(t, seen = new Set()) {
  t = strip(t);
  if (!t) return null;
  if (ts.isTypeLiteralNode(t)) return t.members.filter((m) => ts.isPropertySignature(m));
  if (ts.isIntersectionTypeNode(t)) { const all = []; for (const x of t.types) { const p = propsOf(x, seen); if (!p) return null; for (const pp of p) { const i = all.findIndex((q) => pname(q) === pname(pp)); if (i >= 0) all[i] = pp; else all.push(pp); } } return all; }
  if (ts.isTypeReferenceNode(t)) { const n = refName(t); const d = look(n, t); if (!d || seen.has(n)) return null; seen.add(n); return ts.isInterfaceDeclaration(d) ? d.members.filter((m) => ts.isPropertySignature(m)) : propsOf(d.type, seen); }
  return null;
}
const nonNull1 = (t) => { if (!t) return null; const ms = members(t).filter((m) => !isNullish(m)); return ms.length === 1 ? strip(ms[0]) : strip(t); };
function inlineObj(t) { t = nonNull1(t); if (!t) return null; if (ts.isTypeLiteralNode(t)) return { arr: '', node: t }; if (ts.isArrayTypeNode(t) && ts.isTypeLiteralNode(strip(t.elementType))) return { arr: '[]', node: strip(t.elementType) }; return null; }
function typeStr(t) {
  const io = inlineObj(t);
  const nul = members(t).some(isNullish) ? ' \\| null' : '';
  if (io) return '{…}' + io.arr + nul;
  let s = txt(t).replace(/coreTypes\./g, '');
  if (s.length > 110) s = s.slice(0, 107) + '…';
  return '`' + s.replace(/`/g, "'") + '`';
}
// rows for a type: [path, type, opt, note]
function rows(baseProps, newProps, prefix = '', depth = 0) {
  const out = [];
  const bn = new Set((baseProps || []).map(pname));
  const emit = (m, nm, isNew) => {
    const name = prefix + pname(m);
    let note = docOf(m, 150);
    if (isNew) note = `**${sinceTxt(pname(m))} (not in 0.3.311).** ` + note;
    out.push([name, typeStr(m.type), m.questionToken ? 'opt' : 'req', note]);
    const io = inlineObj(m.type);
    if (io && depth < 2) {
      const nio = nm ? inlineObj(nm.type) : null;
      out.push(...rows(io.node.members.filter((x) => ts.isPropertySignature(x)), nio ? nio.node.members.filter((x) => ts.isPropertySignature(x)) : null, name + io.arr + '.', depth + 1));
    }
  };
  for (const m of baseProps || []) emit(m, (newProps || []).find((x) => pname(x) === pname(m)), false);
  for (const m of newProps || []) if (!bn.has(pname(m))) emit(m, null, true);
  return out;
}
function tableFor(name, { skip = [] } = {}) {
  const b = B.get(name) || X.get(name), n = N.get(name);
  const bp = b ? (ts.isInterfaceDeclaration(b) ? b.members.filter((m) => ts.isPropertySignature(m)) : propsOf(b.type)) : null;
  const np = n ? (ts.isInterfaceDeclaration(n) ? n.members.filter((m) => ts.isPropertySignature(m)) : propsOf(n.type)) : null;
  const r = b ? rows(bp, np) : rows([], np);
  return table(['Field', 'Type', 'Req', 'Notes'], r.filter((x) => !skip.includes(x[0])));
}
function table(head, rs) {
  if (!rs.length) return '_(no fields)_\n';
  return `| ${head.join(' | ')} |\n|${head.map(() => '---').join('|')}|\n` + rs.map((r) => '| ' + r.map((c, i) => (i === 0 ? '`' + esc(c) + '`' : esc(c))).join(' | ') + ' |').join('\n') + '\n';
}
function unionList(name, file = 'base') {
  const d = (file === 'new' ? N : B).get(name);
  const out = [];
  const flat = (t) => members(t).filter((m) => !isNullish(m)).flatMap((m) => { m = strip(m); if (ts.isTypeReferenceNode(m)) { const r = refName(m); const dd = look(r, m); if (dd && ts.isTypeAliasDeclaration(dd) && ts.isUnionTypeNode(strip(dd.type)) && /Message$|Result/.test(r) && r !== 'PermissionResult') return flat(dd.type); return [r]; } return [txt(m)]; });
  return d ? flat(d.type) : out;
}
function disc(name, keys = ['type', 'subtype']) {
  const d = look(name); if (!d) return '';
  const p = ts.isInterfaceDeclaration(d) ? d.members : propsOf(d.type) || [];
  return keys.map((k) => { const q = p.find((x) => x.name && pname(x) === k); if (!q) return null; const v = members(q.type).map((m) => { m = strip(m); return ts.isLiteralTypeNode(m) && ts.isStringLiteral(m.literal) ? m.literal.text : null; }).filter(Boolean); return v.join('|'); }).filter((x) => x !== null).join(':');
}

const L = [];
const P = (...a) => L.push(...a);

// =====================================================================
P(`# Claude Agent SDK API surface (target 0.3.311)

**Target:** \`@anthropic-ai/claude-agent-sdk\` **0.3.311** with Claude Code CLI **2.1.311**.

**Sources:**
- \`sdk.d.ts\` from the published 0.3.311 tarball, cross-checked against 0.3.325.
- \`sdk.mjs\` 0.3.311 for the wire behaviour of each method and option.
- Captured CLI 2.1.311 control traffic, for undeclared response shapes. Rows that come from the captures say "observed".

The companion Go file is \`types.go\` (\`package claude\`). It names each Go type after its TS source and uses the TS doc first sentence as the comment.

**Version markers.** Everything below is the 0.3.311 surface. Anything that 0.3.325 adds is still listed, and marked **since 0.3.3xx (not in 0.3.311)**. "0.3.312-0.3.325" means the changelog does not name the release. Section 9 collects all of them.

Nothing that exists in 0.3.311 was removed or reshaped by 0.3.325: every change between them is additive. The 0.3.311 form is therefore also a valid subset of 0.3.325.

**Hidden fields.** The published d.ts strips \`@internal\` members. Roughly 78 blank "holes" in the 0.3.325 declarations show where. The wire therefore carries fields this document cannot list, and \`types.go\` keeps them in \`Raw\`.

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
`);

// =====================================================================
P('## 1. Query methods', '',
  `\`query({prompt, options})\` returns a \`Query\`. A \`Query\` is an \`AsyncGenerator<SDKMessage>\` plus the methods below.`, '',
  `**How control requests are sent.** Every control request is written as \`{"type":"control_request","request_id":<random base36, 11-13 chars>,"request":{"subtype":…,…}}\`. The reply is \`{"type":"control_response","response":{"subtype":"success","request_id":…,"response":{…}}}\` or \`{…"subtype":"error","error":"…","error_code"?:…}\`.`, '',
  `- An error reply rejects the promise.`,
  `- A request whose AbortSignal fires is followed by \`{"type":"control_cancel_request","request_id":…}\`.`,
  `- Requests wait until the \`initialize\` request has been written.`, '',
  `**Columns.**`,
  `- **Union?** says whether the subtype is in the 0.3.311 d.ts \`SDKControlRequestInner\` union, or only exists in sdk.mjs.`,
  `- **ack** means the TS method ignores the payload: \`{}\` or absent. The CLI may still send fields; see 5.1.`, '');
const Q = [
  ['interrupt', 'interrupt(): Promise<SDKControlInterruptResponse \\| undefined>', 'interrupt', '(none). The implementation also takes a hidden `{cancelQueued}` → `cancel_queued: true`', '`InterruptResponse` {still_queued: string[], cancelled?: string[]}. Returns undefined when `still_queued` is not an array.', 'yes'],
  ['setPermissionMode', 'setPermissionMode(mode: PermissionMode): Promise<void>', 'set_permission_mode', 'mode → `mode`', 'ack. The CLI answers `{mode}` (observed).', 'yes'],
  ['setMcpPermissionModeOverride', "setMcpPermissionModeOverride(serverName: string, mode: 'default' \\| 'auto' \\| null): Promise<{warning?: string}>", 'set_mcp_permission_mode_override', 'serverName → `serverName` (camelCase on the wire), mode → `mode`', '`{warning?: string}` (`response ?? {}`)', '**no** (sdk.mjs only)'],
  ['setModel', 'setModel(model?: string): Promise<void>', 'set_model', 'model → `model` (undefined → omitted; resets to default)', 'ack', 'yes'],
  ['setMaxThinkingTokens', "setMaxThinkingTokens(maxThinkingTokens: number \\| null, thinkingDisplay?: 'summarized' \\| 'omitted' \\| 'highlights' \\| null): Promise<void> — **deprecated**", 'set_max_thinking_tokens', 'maxThinkingTokens → `max_thinking_tokens`, thinkingDisplay → `thinking_display`', 'ack', 'yes'],
  ['applyFlagSettings', 'applyFlagSettings(settings: {[K in keyof Settings]?: Settings[K] \\| null}): Promise<void>', 'apply_flag_settings', 'settings → `settings` (null deletes a key from the flag layer)', 'ack', 'yes'],
  ['updateSettings', "updateSettings(source: 'localSettings' \\| 'userSettings', settings: Record<string, unknown>): Promise<void>", 'update_settings', 'source → `source`, settings → `settings`', 'ack', 'yes'],
  ['initializationResult', 'initializationResult(): Promise<SDKControlInitializeResponse>', '— (no request)', '—', 'The stored answer to the first `initialize` (or the latest `reinitialize`): `InitializeResponse`', 'n/a'],
  ['reinitialize', 'reinitialize(): Promise<SDKControlInitializeResponse>', 'initialize', 'The full initialize request is rebuilt from options (same fields as the first one, section 7.6)', '`InitializeResponse`', 'yes'],
  ['supportedCommands', 'supportedCommands(): Promise<SlashCommand[]>', '— (no request)', '—', '`initialize.commands`, replaced by the latest `system/commands_changed` list', 'n/a'],
  ['supportedModels', 'supportedModels(): Promise<ModelInfo[]>', '— (no request)', '—', '`initialize.models`', 'n/a'],
  ['supportedAgents', 'supportedAgents(): Promise<AgentInfo[]>', '— (no request)', '—', '`initialize.agents`', 'n/a'],
  ['mcpServerStatus', 'mcpServerStatus(): Promise<McpServerStatus[]>', 'mcp_status', '(none)', '`response.mcpServers`: `McpServerStatus[]` (Go: `McpStatusResponse`)', 'yes'],
  ['getContextUsage', "getContextUsage(opts?: {detail?: 'summary' \\| 'full'}): Promise<SDKControlGetContextUsageResponse>", 'get_context_usage', 'opts spread: detail → `detail`', '`GetContextUsageResponse` (camelCase fields)', 'yes'],
  ['usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET', 'usage_EXPERIMENTAL…(opts?: {skipBehaviors?: boolean}): Promise<SDKControlGetUsageResponse>', 'get_usage', 'skipBehaviors → `skip_behaviors: true` (only when true)', '`GetUsageResponse`', 'yes'],
  ['readFile', "readFile(path: string, options?: {maxBytes?: number; encoding?: 'utf-8' \\| 'base64'}): Promise<SDKControlReadFileResponse \\| null>", 'read_file', 'path → `path`, maxBytes → `max_bytes`, encoding → `encoding`', '`ReadFileResponse` {contents, absPath, truncated?, encoding?}. **Any error resolves to null.**', 'yes'],
  ['reloadPlugins', 'reloadPlugins(options?: {holdOnCacheImpact?: boolean; addPlugins?: SdkPluginConfig[]}): Promise<SDKControlReloadPluginsResponse>', 'reload_plugins', 'holdOnCacheImpact → `hold_on_cache_impact: true` (only when true); addPlugins → `add_plugins` (only when non-empty). 0.3.325 adds refuseIfBusy → `refuse_if_busy: true` (**since 0.3.325**)', '`ReloadPluginsResponse`', 'yes'],
  ['reloadSkills', 'reloadSkills(): Promise<SDKControlReloadSkillsResponse>', 'reload_skills', '(none)', '`ReloadSkillsResponse` {skills: SlashCommand[]}', 'yes'],
  ['reloadOutputStyles', 'reloadOutputStyles(): Promise<SDKControlReloadOutputStylesResponse>', 'reload_output_styles', '(none)', '`ReloadOutputStylesResponse` {available_output_styles}', 'yes'],
  ['accountInfo', 'accountInfo(): Promise<AccountInfo>', '— (no request)', '—', '`initialize.account`', 'n/a'],
  ['rewindFiles', 'rewindFiles(userMessageId: string, options?: {dryRun?: boolean}): Promise<RewindFilesResult>', 'rewind_files', 'userMessageId → `user_message_id`, dryRun → `dry_run`', '`RewindFilesResult` {canRewind, error?, filesChanged?, insertions?, deletions?, skippedLinks?}', 'yes'],
  ['seedReadState', 'seedReadState(path: string, mtime: number): Promise<void>', 'seed_read_state', 'path → `path`, mtime → `mtime`', 'ack', 'yes'],
  ['reconnectMcpServer', 'reconnectMcpServer(serverName: string): Promise<void>', 'mcp_reconnect', 'serverName → `serverName` (camelCase)', 'ack', 'yes'],
  ['toggleMcpServer', 'toggleMcpServer(serverName: string, enabled: boolean): Promise<void>', 'mcp_toggle', 'serverName → `serverName`, enabled → `enabled`', 'ack', 'yes'],
  ['readMcpResource', 'readMcpResource(serverName: string, uri: string): Promise<SDKControlMcpReadResourceResponse> — *alpha*', 'mcp_read_resource', 'serverName → `serverName`, uri → `uri`', '`McpReadResourceResponse` {contents: {uri, mimeType?, text?, blob?, _meta?}[]}', 'yes'],
  ['setMcpServers', 'setMcpServers(servers: Record<string, McpServerConfig>): Promise<McpSetServersResult>', 'mcp_set_servers', 'servers → `servers`. Before sending, the SDK connects in-process (`type:"sdk"` with `instance`) servers itself, disconnects removed ones, and replaces each with `{type:"sdk", name, timeout?, tools?}`', '`McpSetServersResult` {added, removed, errors}', 'yes'],
  ['streamInput', 'streamInput(stream: AsyncIterable<SDKUserMessage>): Promise<void>', '— (no control request)', 'Writes each SDKUserMessage as one stdin JSON line (adds `client_composed: true` when `verbatimPrompts`). Afterwards it closes stdin, or first waits for the run to end when hooks, canUseTool or SDK MCP servers need the channel.', '—', 'n/a'],
  ['stopTask', 'stopTask(taskId: string): Promise<void>', 'stop_task', 'taskId → `task_id`', 'ack', 'yes'],
  ['backgroundTasks', 'backgroundTasks(toolUseId?: string): Promise<boolean>', 'background_tasks', 'toolUseId → `tool_use_id`', '`response.backgrounded ?? true` (Go: `BackgroundTasksResponse`)', 'yes'],
  ['close', 'close(): void', '— (no request)', '—', 'Cleanup: aborts in-flight host handlers, ends stdin, SIGTERM after 2 s if still alive, SIGKILL 5 s later', 'n/a'],
];
const qdocs = Object.fromEntries(JSON.parse(fs.readFileSync(optFile.replace('options311', 'query311'), 'utf8')).map((q) => [q.name, q.doc.split(/(?<=[.!?])\s(?=[A-Z`])/)[0]]));
P(table(['Method', 'TS signature', 'Doc (first sentence)', 'Subtype sent', 'Request fields (TS param → wire)', 'Response', 'Union?'],
  Q.map(([m, sig, sub, req, resp, u]) => [m, sig, esc(qdocs[m] || '').slice(0, 180), sub, req, resp, u])));
P('', `There are **30 methods** (counting \`close\`, \`streamInput\` and the four cached accessors). The method set and the request field names are identical in 0.3.311 and 0.3.325, except that \`reloadPlugins\` gains \`refuseIfBusy\`.`, '',
  `### 1.1 Query methods present in sdk.mjs but stripped from the d.ts (0.3.311 and 0.3.325)`, '',
  `None of these subtypes is in the d.ts control-request union. They are listed so a Go client can mirror the wire if it needs to.`, '',
  table(['Method (internal)', 'Subtype', 'Request fields'], [
    ['getSettings()', 'get_settings', '—; response observed as `{effective, sources, applied}` (Go `GetSettingsResponse`). The subtype *is* in the union; only the method is hidden.'],
    ['getHooksListing()', 'get_hooks_listing', '— (in union)'], ['getTaskOutput(taskId)', 'get_task_output', '`task_id` (in union)'], ['listPermissionRules()', 'list_permission_rules', '— (in union)'],
    ['cancelAsyncMessage(uuid)', 'cancel_async_message', '`message_uuid` (in union); returns `response.cancelled`'],
    ['setCwd(path, {trustAccepted?, trustedDirectory?})', 'set_cwd', '`path`, `trust_accepted`, `trusted_directory`'],
    ['claimSession(p)', 'claim_session', 'spread p + `include_initialize` (default true)'],
    ['enableRemoteControl(enabled, name?, opts?)', 'remote_control', '`enabled`, `name`, `reattach_session_id`, `keep_session_on_exit`, `work_secret`'],
    ['submitFeedback(description, opts?)', 'submit_feedback', '`description`, `surface`, `draft_id`, `type`, `title`, `area`, `attach_transcript`, `save_locally`'],
    ['submitSurveyFeedback(p)', 'submit_survey_feedback', 'spread p'], ['renameSession(title, sessionId?)', 'rename_session', '`title`, `source:"host"`, `session_id` (in union)'],
    ['generateSessionTitle(description, {persist?})', 'generate_session_title', '`description`, `persist` → returns `response.title`'],
    ['askSideQuestion(question, {history?, signal?})', 'side_question', '`question`, `history`; progress arrives as `system/control_request_progress`'],
    ['launchUltrareview(args, {confirm?})', 'ultrareview_launch', '`args`, `confirm`'], ['messageRated(p)', 'message_rated', '`messageUuid`, `sentiment`, `surface`, `cleared`'],
    ['setChromeBrowserHints(p)', 'set_chrome_browser_hints', 'spread p'], ['setPromptSuggestionsPaused(paused)', 'set_prompt_suggestions_paused', '`paused`'],
    ['enableChannel(serverName)', 'channel_enable', '`serverName`'], ['mcpAuthenticate(serverName, redirectUri, {redirectUriOnly?})', 'mcp_authenticate', '`serverName`, `redirectUri`, `redirectUriOnly`'],
    ['mcpClearAuth(serverName)', 'mcp_clear_auth', '`serverName`'], ['mcpSubmitOAuthCallbackUrl(serverName, callbackUrl)', 'mcp_oauth_callback_url', '`serverName`, `callbackUrl`'],
    ['claudeAuthenticate(loginWithClaudeAi, {consoleProfile?})', 'claude_authenticate', '`loginWithClaudeAi`, `consoleProfile`'],
    ['claudeOAuthCallback(code, state)', 'claude_oauth_callback', '`authorizationCode`, `state`'], ['claudeOAuthWaitForCompletion()', 'claude_oauth_wait_for_completion', '—'],
    ['getMemoryDialog() / getSkillsDialog() / getStatus() / exportConversation() / getChromeDialog() / getChromeBrowsers() / getSandboxDialog() / getPlan()', 'get_memory_dialog / get_skills_dialog / get_status / export_conversation / get_chrome_dialog / get_chrome_browsers / get_sandbox_dialog / get_plan', '—'],
    ['selectChromeBrowser(deviceId)', 'select_chrome_browser', '`device_id`'],
  ]), '');

// =====================================================================
const CAT = {
  abortController: ['transport', 'Aborting kills the process. In Go: context.Context.'],
  additionalDirectories: ['argv', '`--add-dir <dir>` per entry'], projectConfigRoot: ['argv', '`--project-config-root`'], agent: ['argv', '`--agent`'],
  agents: ['initialize', '`initialize.agents`'], allowedTools: ['argv', '`--allowedTools a,b` (plus `Skill(...)` entries derived from `skills`)'],
  canUseTool: ['host-only callback', 'Adds `--permission-prompt-tool=stdio`; answers `can_use_tool`. Throws if combined with permissionPromptToolName.'],
  continue: ['argv', '`--continue` (with sessionStore: resolves the latest session via the store instead)'], cwd: ['transport', 'Spawn cwd'],
  disallowedTools: ['argv', '`--disallowedTools a,b`'], toolAliases: ['initialize', '`initialize.toolAliases`'],
  tools: ['argv', "`--tools a,b`; the preset object → `--tools=default`"], env: ['env', 'The child environment (default: `process.env`). The SDK always sets `CLAUDE_CODE_ENTRYPOINT=sdk-ts` and `CLAUDE_AGENT_SDK_VERSION` if missing, and `CLAUDE_CODE_SDK_READS_SESSION_STATE=1`. It deletes `NODE_OPTIONS`, and `DEBUG` unless `DEBUG_CLAUDE_AGENT_SDK`.'],
  executable: ['transport', 'Runtime used when the executable path is a .js file (default bun if running under bun, else node)'], executableArgs: ['transport', 'Prepended runtime args'],
  extraArgs: ['argv', '`--<key>=<value>`, or `--<key>` for null'], fallbackModel: ['argv', '`--fallback-model` (throws if equal to model)'],
  enableFileCheckpointing: ['env', '`CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true`'],
  toolConfig: ['env', '`CLAUDE_CODE_QUESTION_PREVIEW_FORMAT` (+ `CLAUDE_CODE_QUESTION_EXTENDED`, `…_OPTIONAL_DESCRIPTIONS` for hidden sub-keys)'],
  forkSession: ['argv', '`--fork-session`'], betas: ['argv', '`--betas a,b`'],
  hooks: ['host-only callback', 'Callbacks get ids `hook_<n>`. `initialize.hooks` = `{Event: [{matcher, hookCallbackIds, timeout}]}`. The CLI calls back with `hook_callback`.'],
  onElicitation: ['host-only callback', 'Answers `elicitation` (default when absent: `{action:"decline"}`)'],
  onUserDialog: ['host-only callback', 'Answers `request_user_dialog` (when absent the request is left unanswered)'],
  supportedDialogKinds: ['initialize', '`initialize.supportedDialogKinds` (validated against onUserDialog)'], perTaskStopAffordance: ['initialize', '`initialize.perTaskStopAffordance`'],
  persistSession: ['argv', 'false → `--no-session-persistence`'], sessionStore: ['host-only callback', 'Store methods. Adds `--session-mirror`; the CLI emits transcript_mirror frames that the SDK appends to the store; resume materialises the store into a temp CLAUDE_CONFIG_DIR'],
  sessionStoreFlush: ['host-only callback', 'Batching of sessionStore.append'], loadTimeoutMs: ['host-only callback', 'Timeout for store load/listSubkeys on resume'],
  includeHookEvents: ['argv', '`--include-hook-events`'], includePartialMessages: ['argv', '`--include-partial-messages`'], forwardSubagentText: ['initialize', '`initialize.forwardSubagentText`'],
  verbatimPrompts: ['transport', 'Stamps `client_composed: true` on every user message written to stdin'],
  thinking: ['argv', "adaptive → `--thinking=adaptive`; enabled → `--max-thinking-tokens=N` (no budget → adaptive); disabled → `--thinking=disabled`; display → `--thinking-display`"],
  effort: ['argv', '`--effort`'], maxThinkingTokens: ['argv', 'When thinking is unset: 0 → disabled, else enabled with budget'], maxTurns: ['argv', '`--max-turns`'],
  maxBudgetUsd: ['argv', '`--max-budget-usd`'], taskBudget: ['argv', '`--task-budget <total>`'],
  mcpServers: ['argv', 'Non-SDK servers → `--mcp-config {"mcpServers":{…}}`. `type:"sdk"` servers with an instance stay in process: their names go in `initialize.sdkMcpServers` and their timeout/tools in `initialize.sdkMcpServerConfigs`. Initialize also carries `sdkMcpServerManifests` (captured initialize + tools/list).'],
  model: ['argv', '`--model`'], outputFormat: ['argv', '`--json-schema <json>` and `initialize.jsonSchema`'], pathToClaudeCodeExecutable: ['transport', 'Executable to spawn (default: the bundled native binary for platform/arch)'],
  permissionMode: ['argv', '`--permission-mode`'], planModeInstructions: ['initialize', '`initialize.planModeInstructions`'], allowDangerouslySkipPermissions: ['argv', '`--allow-dangerously-skip-permissions`'],
  permissionPromptToolName: ['argv', '`--permission-prompt-tool <name>`'], permissionPrompts: ['argv', '`--permission-prompts host|none`'],
  plugins: ['argv', "pluginDelivery 'argv': `--plugin-dir <path>` / `--plugin-dir-no-mcp <path>`. 'initialize': `--await-initialize` + `initialize.plugins`"],
  pluginDelivery: ['argv', 'Chooses how plugins travel (see plugins)'], promptSuggestions: ['initialize', '`initialize.promptSuggestions`'], agentProgressSummaries: ['initialize', '`initialize.agentProgressSummaries`'],
  resume: ['argv', '`--resume <id>`'], sessionId: ['argv', '`--session-id`'], resumeSessionAt: ['argv', '`--resume-session-at`'], resumeDropsTurn: ['argv', '`--resume-drops-turn`'],
  sandbox: ['argv', "Merged into the `settings` object as `settings.sandbox` and sent with `--settings`. `enabled:true` defaults `failIfUnavailable:true`. Throws if `settings` is a file path"],
  settings: ['argv', '`--settings <path|json>`'], managedSettings: ['argv', '`--managed-settings <json>`'], settingSources: ['argv', '`--setting-sources a,b`'],
  skills: ['argv', "Adds `Skill` ('all') or `Skill(name)` entries to `--allowedTools`; list also → `initialize.skills`"],
  debug: ['argv', '`--debug`'], debugFile: ['argv', '`--debug-file`'], stderr: ['host-only callback', 'Receives child stderr chunks'], strictMcpConfig: ['argv', '`--strict-mcp-config`'],
  systemPrompt: ['initialize', 'string → `initialize.systemPrompt:[s]`; string[] → as is; custom → `systemPrompt` + `systemPromptSnapshot`; preset → `appendSystemPrompt`, `excludeDynamicSections`, `systemPromptSnapshot`; omitted → `""`'],
  title: ['initialize', '`initialize.title`'], spawnClaudeCodeProcess: ['host-only callback', 'Replaces the local spawn (receives {command,args,cwd,env,signal})'],
};
P('## 2. Options', '',
  `\`Options\` has **${options.length} fields** in 0.3.311; the set is unchanged in 0.3.325.`, '',
  `**Categories.**`,
  `- \`argv\`: becomes a CLI flag.`,
  `- \`initialize\`: sent in the \`initialize\` control request.`,
  `- \`env\`: becomes a child-process environment variable.`,
  `- \`host-only callback\`: a function, or store, that runs in the host.`,
  `- \`transport\`: affects how the process is spawned or how stdin is written.`, '',
  `**Fixed flags.** Every launch gets \`--output-format stream-json --verbose --input-format stream-json\`. Flags with values use the \`--flag=value\` form, except where noted.`, '',
  `**In Go.** \`Options\` is a data-only struct. The function-valued options \`canUseTool\`, \`onElicitation\`, \`onUserDialog\` and \`stderr\` are booleans. \`hooks\` holds callback ids, \`HookCallbackMatcher.HookCallbackIDs\`. \`abortController\`, \`sessionStore\` and \`spawnClaudeCodeProcess\` are omitted.`, '');
P(table(['Field', 'Type', 'Doc (first sentence)', 'Default', 'Category', 'Wire mapping (sdk.mjs 0.3.311)'], options.map((o) => {
  const d = o.doc.split(/(?<=[.!?])\s(?=[A-Z`*])/)[0];
  let def = o.tags.default ? String(o.tags.default) : '';
  const dm = o.doc.match(/[Dd]efaults? to ([^.;]+)/); if (!def && dm) def = dm[1].replace(/\s+/g, ' ').slice(0, 60);
  if (!def && /\(default\)/.test(o.doc)) def = (o.doc.match(/`'?([^`']+)'?`\s*\(default\)/) || [])[1] || '';
  const extra = { executable: "`'bun'` under Bun, else `'node'`", permissionPrompts: "`'host'`", pluginDelivery: "`'argv'`", verbatimPrompts: 'false', allowDangerouslySkipPermissions: 'false', settingSources: 'all sources', cwd: '`process.cwd()`', env: '`process.env`', model: 'CLI default' };
  if (extra[o.name]) def = extra[o.name];
  const [cat, map] = CAT[o.name] || ['?', ''];
  return [o.name + (o.optional ? '?' : ''), '`' + o.type.replace(/`/g, "'").slice(0, 120) + (o.type.length > 120 ? '…' : '') + '`', (o.tags.deprecated ? '**Deprecated.** ' : '') + d.slice(0, 200) + (o.tags.alpha ? ' *(alpha)*' : ''), def, cat, map];
})), '');
P(`**Not in the d.ts but read by the 0.3.311 \`query()\`.** These are internal options: \`appendSubagentSystemPrompt\`, \`codeRunningMcpTools\`, \`webSearchIsolationExemptMcpServers\`, \`rapidFollowupPreempt\`, \`forwardNestedSubagentToolCalls\` and \`workspaceTrust\`, all → initialize; \`channels\` → \`--channels\`; \`workload\` → \`--workload\`; \`getOAuthToken\` and \`getHostAuthToken\`, host callbacks for the \`oauth_token_refresh\` and \`host_auth_token_refresh\` requests, which also set \`CLAUDE_CODE_SDK_HAS_OAUTH_REFRESH=1\` and \`CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH=1\`; and \`onProcessSpawned\`.`, '',
  `**First message.** A string \`prompt\` is written as \`{"type":"user","session_id":"","message":{"role":"user","content":[{"type":"text","text":…}]},"parent_tool_use_id":null}\`. Stdin then closes after the first result, which makes it a single-turn query. An \`AsyncIterable\` prompt goes through \`streamInput\`.`, '', '---', '');

// =====================================================================
P('## 3. Messages', '',
  `\`SDKMessage\` is the union the \`Query\` generator yields. \`StdoutMessage\` (not exported) is everything the CLI writes to stdout in stream-json mode, one JSON object per line. It is \`SDKMessage\` plus \`SDKActiveGoalMessage\` (\`active_goal\`), \`SDKControlResponse\`, \`SDKControlRequest\`, \`SDKControlCancelRequest\` and \`SDKKeepAliveMessage\` (\`keep_alive\`).`, '',
  `In Go:`,
  `- \`StdoutMessage\` and \`Message\` are unions keyed on \`type\` and \`subtype\`.`,
  `- \`user\` and \`user\`+\`isReplay:true\` share a discriminator. \`Decode\` picks \`UserMessageReplay\` when \`isReplay\` is true.`,
  `- \`StdoutMessage.SDKMessage()\` narrows a \`StdoutMessage\` to a \`Message\`.`, '');
const msgVariants311 = unionList('SDKMessage');
const msgVariants325 = unionList('SDKMessage', 'new').filter((n) => !msgVariants311.includes(n));
const allMsg = [...msgVariants311, ...msgVariants325, 'SDKActiveGoalMessage', 'SDKControlRequest', 'SDKControlResponse', 'SDKControlCancelRequest', 'SDKKeepAliveMessage'];
const discOf = (n) => (n === 'SDKUserMessageReplay' ? 'user (isReplay: true)' : disc(n));
P(table(['type:subtype', 'TS type', 'Go type', 'Union', 'Doc'], allMsg.map((n) => [discOf(n), n, goName(n), msgVariants325.includes(n) ? `SDKMessage — **${sinceTxt(n)} (not in 0.3.311)**` : msgVariants311.includes(n) ? 'SDKMessage' : 'StdoutMessage only', docOf(look(n), 140)])), '');
P(`**Counts:** ${msgVariants311.length} SDKMessage variants in 0.3.311, ${msgVariants311.length + msgVariants325.length} in 0.3.325; StdoutMessage adds 5 more. All four result error subtypes share \`SDKResultError\`.`, '');
for (const n of allMsg) {
  if (['SDKControlRequest', 'SDKControlResponse', 'SDKControlCancelRequest'].includes(n)) continue;
  P(`### ${discOf(n)} — \`${n}\` → Go \`${goName(n)}\`${msgVariants325.includes(n) ? ` (**${sinceTxt(n)}, not in 0.3.311**)` : ''}`, '', docOf(look(n), 400), '', tableFor(n));
}
P('---', '');

// =====================================================================
P('## 4. Supporting types', '',
  `### 4.1 Assistant message body and content blocks`, '',
  `\`SDKAssistantMessage.message\` is an Anthropic Messages API \`BetaMessage\` (from \`@anthropic-ai/sdk\`, peer dependency >=0.93.0, not declared in sdk.d.ts). \`SDKUserMessage.message\` is a \`MessageParam\`, whose \`content\` is a string or an array of content blocks.`, '',
  `\`types.go\` models these by hand as \`APIMessage\`, \`MessageParam\`, \`Usage\` and the \`ContentBlock\` union. It merges response blocks and request block-params into one union keyed by \`type\`. Unmodelled block types decode as unknown, with \`Raw\` kept.`, '',
  tableFor('BetaMessage'), '', `**\`Usage\`** (\`NonNullableUsage\`: BetaUsage with nulls removed; result messages carry it):`, '', tableFor('NonNullableUsage'), '',
  table(['Block `type`', 'Go type', 'Where', 'Fields'], [
    ['text', 'TextBlock', 'assistant, user', 'text, citations?'], ['thinking', 'ThinkingBlock', 'assistant', 'thinking, signature'], ['redacted_thinking', 'RedactedThinkingBlock', 'assistant', 'data'],
    ['tool_use', 'ToolUseBlock', 'assistant', 'id, name, input'], ['server_tool_use', 'ServerToolUseBlock', 'assistant', 'id, name, input'],
    ['tool_result', 'ToolResultBlock', 'user', 'tool_use_id, content? (string \\| blocks), is_error?'], ['image', 'ImageBlock', 'user', 'source {type: base64\\|url\\|file …}'],
    ['document', 'DocumentBlock', 'user', 'source, title?, context?, citations?'], ['search_result', 'SearchResultBlock', 'user', 'source, title, content'],
    ['mcp_tool_use / mcp_tool_result', 'MCPToolUseBlock / MCPToolResultBlock', 'assistant', 'id, name, server_name, input / tool_use_id, is_error, content'],
    ['container_upload', 'ContainerUploadBlock', 'assistant', 'file_id'],
    ['web_search_tool_result, web_fetch_tool_result, code_execution_tool_result, bash_code_execution_tool_result, text_editor_code_execution_tool_result', '…ToolResultBlock', 'assistant', 'tool_use_id, content (kept raw)'],
  ]), '',
  `### 4.2 \`SDKPartialAssistantMessage\` (\`stream_event\`)`, '',
  `This message is emitted only with \`includePartialMessages\` / \`--include-partial-messages\`. \`event\` is one \`BetaRawMessageStreamEvent\`; Go: the \`StreamEvent\` union.`, '', tableFor('SDKPartialAssistantMessage'), '',
  table(['event.type', 'Go type', 'Fields'], [['message_start', 'MessageStartEvent', 'message: APIMessage'], ['message_delta', 'MessageDeltaEvent', 'delta {stop_reason, stop_sequence}, usage: MessageDeltaUsage, context_management?'],
    ['message_stop', 'MessageStopEvent', '—'], ['content_block_start', 'ContentBlockStartEvent', 'index, content_block: ContentBlock'],
    ['content_block_delta', 'ContentBlockDeltaEvent', 'index, delta: ContentBlockDelta (text_delta.text \\| input_json_delta.partial_json \\| citations_delta.citation \\| thinking_delta.thinking \\| signature_delta.signature)'],
    ['content_block_stop', 'ContentBlockStopEvent', 'index']]), '',
  `### 4.3 Result variants (\`SDKResultMessage = SDKResultSuccess | SDKResultError\`)`, '',
  `- \`result:success\` has \`result\` (text), \`structured_output?\`, \`deferred_tool_use?\`, \`api_error_status?\` and timing fields.`,
  `- \`result:error_during_execution | error_max_turns | error_max_budget_usd | error_max_structured_output_retries\` has \`errors: string[]\` and \`startup_failure_reason?\` (\`SDKStartupFailureReason\`). In 0.3.325 it also has \`invalid_config_file?\` (**since 0.3.325**).`,
  `- Both carry \`usage\`, \`modelUsage\` (Record<model, ModelUsage>), \`permission_denials\`, \`total_cost_usd\`, \`num_turns\`, \`terminal_reason?\`, \`result_index?\`, \`fast_mode_*\` and \`origin?\`.`,
  `- 0.3.318 adds \`credential_expires_at?\`.`,
  `- Field tables are in section 3.`, '',
  `In Go, \`Message.ResultMessage()\` gives the \`ResultMessage\` union, with \`AsResultSuccess\` and \`AsResultError\`.`, '',
  `**ModelUsage**`, '', tableFor('ModelUsage'), '', `**SDKPermissionDenial**`, '', tableFor('SDKPermissionDenial'), '',
  `**SDKMessageOrigin** (union keyed by \`kind\`): ${unionList('SDKMessageOrigin').map((m) => m.match(/kind: '([^']+)'/)?.[1]).filter(Boolean).map((k) => '`' + k + '`').join(', ')}. In Go it is \`MessageOrigin\` with \`MessageOrigin<Kind>\` variants.`, '', '---', '');

// =====================================================================
const inner311 = unionList('SDKControlRequestInner');
const inner325 = unionList('SDKControlRequestInner', 'new');
const innerAll = [...inner311, ...inner325.filter((n) => !inner311.includes(n))];
const cliToHost = new Set(['SDKControlPermissionRequest', 'SDKHookCallbackRequest', 'SDKControlMcpMessageRequest', 'SDKControlElicitationRequest', 'SDKControlRequestUserDialogRequest']);
const RESP = {
  initialize: '`SDKControlInitializeResponse` (7.6)', interrupt: '`SDKControlInterruptResponse`', set_permission_mode: 'ack (observed `{mode}`)', set_model: 'ack', set_max_thinking_tokens: 'ack', rename_session: 'ack',
  set_color: 'ack', mcp_status: '`{mcpServers: McpServerStatus[]}` (sdk.mjs)', get_context_usage: '`SDKControlGetContextUsageResponse`', get_session_cost: 'undeclared; observed `{text}`',
  list_models: 'undeclared; observed `{models: ModelInfo[]}`', get_usage: '`SDKControlGetUsageResponse`', get_binary_version: 'undeclared', mcp_call: 'undeclared (MCP tools/call result)',
  file_suggestions: 'undeclared', rewind_files: '`RewindFilesResult`', cancel_async_message: '`{cancelled}` (sdk.mjs)', read_file: '`SDKControlReadFileResponse`', seed_read_state: 'ack',
  mcp_set_servers: '`McpSetServersResult`', register_repo_root: 'undeclared', reload_plugins: '`SDKControlReloadPluginsResponse`', reload_skills: '`SDKControlReloadSkillsResponse`',
  reload_output_styles: '`SDKControlReloadOutputStylesResponse`', mcp_reconnect: 'ack', mcp_toggle: 'ack', set_has_subscribers: 'ack', stop_task: 'ack',
  background_tasks: '`{backgrounded?: boolean}` (sdk.mjs)', get_task_output: '`SDKControlGetTaskOutputResponse`', apply_flag_settings: 'ack', get_settings: 'undeclared; observed `{effective, sources, applied}`',
  get_hooks_listing: '`SDKControlGetHooksListingResponse`', update_settings: 'ack', list_permission_rules: '`SDKControlListPermissionRulesResponse`', mcp_read_resource: '`SDKControlMcpReadResourceResponse`',
  can_use_tool: '`PermissionResult` + `toolUseID`', hook_callback: '`HookJSONOutput`', mcp_message: '`{mcp_response: JSONRPCMessage}`', elicitation: '`ElicitResult` {action, content?}',
  request_user_dialog: '`UserDialogResult` {behavior:"completed", result} \\| {behavior:"cancelled"}',
};
P('## 5. Control protocol', '',
  `### 5.0 Envelopes`, '', `**SDKControlRequest** (either direction):`, '', tableFor('SDKControlRequest'), '',
  `**SDKControlResponse**: \`{type:"control_response", response: ControlResponse | ControlErrorResponse}\`. In Go, \`ControlResponse.Response\` is the \`ControlResponseResponse\` union keyed by \`subtype\`.`, '',
  `**ControlResponse** (\`subtype:"success"\`), Go \`ControlSuccessResponse\`:`, '', tableFor('ControlResponse'), '',
  `**ControlErrorResponse** (\`subtype:"error"\`). The CLI also sends \`error_code\` (e.g. \`reload_plugins_busy\`), which the d.ts strips; Go reads it with \`ControlErrorResponse.ErrorCode()\`.`, '', tableFor('ControlErrorResponse'), '',
  `**SDKControlCancelRequest**: \`{type:"control_cancel_request", request_id}\`. Either side sends it to withdraw its own in-flight request. When it arrives for a host handler, the SDK aborts that handler's AbortSignal.`, '',
  `**SDKKeepAliveMessage**: \`{type:"keep_alive"}\`.`, '',
  `### 5.1 Control-request subtypes in the d.ts union`, '',
  `There are **${inner311.length} subtypes in 0.3.311** and ${innerAll.length} in 0.3.325. "Direction" means who sends the request.`, '',
  table(['Subtype', 'TS type → Go type', 'Direction', 'Request fields', 'Success response'], innerAll.map((n) => {
    const sub = disc(n, ['subtype']);
    const ps = propsOf(look(n).type) || [];
    const f = ps.filter((p) => pname(p) !== 'subtype').map((p) => pname(p) + (p.questionToken ? '?' : '')).join(', ');
    return [sub, `${n} → ${goName(n)}` + (inner311.includes(n) ? '' : ` **(${sinceTxt(n)}, not in 0.3.311)**`), cliToHost.has(n) ? 'CLI → host' : 'host → CLI', f || '—', RESP[sub] || '?'];
  })), '',
  `### 5.2 CLI → host requests that the SDK handles but the d.ts union omits`, '',
  table(['Subtype', 'Handled by', 'Answer'], [
    ['oauth_token_refresh', 'internal option `getOAuthToken`', '`{accessToken, reason?}`'], ['host_auth_token_refresh', 'internal option `getHostAuthToken`', '`{authToken}`'],
    ['remote_control_work_secret', 'enableRemoteControl refresh callback', '`{work_secret}`'], ['ui_copy / ui_prompt_read / ui_prompt_fill / ui_prompt_paint / ui_prompt_suggest / ui_read_selection', 'internal `setUiHost`', 'host-defined'],
  ]), '', `Any other unknown subtype is answered with an error.`, '',
  `### 5.3 \`can_use_tool\` (CLI → host)`, '', tableFor('SDKControlPermissionRequest'), '',
  `**How the SDK calls \`canUseTool\`.** It calls \`canUseTool(tool_name, input, opts)\` with these opts:`, '',
  table(['opts field', 'From'], [
    ['signal', 'aborted by `control_cancel_request` / close'], ['suggestions', '`permission_suggestions`'], ['blockedPath', '`blocked_path`'],
    ['mcpServer', '`mcp_server`'], ['decisionReason', '`decision_reason`'], ['decisionReasonType', '`decision_reason_type`'],
    ['title', '`title`'], ['displayName', '`display_name`'], ['description', '`description`'], ['defaultToNo', '`default_to_no`'],
    ['suppressAlwaysAllowRule', '`suppress_always_allow_rule`'], ['toolUseID', '`tool_use_id`'], ['agentID', '`agent_id`'], ['requestId', 'envelope `request_id`'],
    ['requiresUserInteraction', '`requires_user_interaction`'], ['matchedAskRule', '`matched_ask_rule`: {source, toolName, ruleContent?}'],
    ['(hidden)', '`decision_reason_code`, `server_policy_ask_only`, `plugin`, `server_prompt`, `computer_folder` are also forwarded'],
  ]), '',
  `**The reply.** It is \`{subtype:"success", request_id, response: {...PermissionResult, toolUseID: tool_use_id}}\`. A \`null\` from \`canUseTool\` means the host answered out-of-band, so the SDK writes nothing.`, '',
  `**PermissionResult** (union on \`behavior\`; Go \`PermissionResult\` with \`PermissionResultAllow\` / \`PermissionResultDeny\`):`, '',
  table(['behavior', 'Fields'], [['allow', 'updatedInput?: Record<string, unknown>, updatedPermissions?: PermissionUpdate[], toolUseID?, decisionClassification?'], ['deny', 'message: string, interrupt?: boolean, toolUseID?, decisionClassification?']]), '',
  `Here \`decisionClassification\` is \`'user_temporary' | 'user_permanent' | 'user_reject'\`.`, '',
  `**PermissionUpdate** (union on \`type\`; Go \`PermissionUpdate\`):`, '',
  table(['type', 'Go variant', 'Fields'], [['addRules', 'PermissionUpdateAddRules', 'rules: {toolName, ruleContent?}[], behavior: allow\\|deny\\|ask, destination'], ['replaceRules', 'PermissionUpdateReplaceRules', 'rules, behavior, destination'],
    ['removeRules', 'PermissionUpdateRemoveRules', 'rules, behavior, destination'], ['setMode', 'PermissionUpdateSetMode', 'mode: PermissionMode, destination'],
    ['addDirectories', 'PermissionUpdateAddDirectories', 'directories: string[], destination'], ['removeDirectories', 'PermissionUpdateRemoveDirectories', 'directories: string[], destination']]), '',
  `**PermissionUpdateDestination:** \`userSettings | projectSettings | localSettings | session | cliArg\`.`, '',
  `**PermissionBehavior:** \`allow | deny | ask\`.`, '',
  `**PermissionMode:** \`default | acceptEdits | bypassPermissions | plan | dontAsk | auto\`. \`bypassPermissions\` requires \`allowDangerouslySkipPermissions\`. In Go these are the \`PermissionMode*\` constants.`, '',
  `### 5.4 Other CLI → host requests`, '',
  `- **hook_callback** \`{callback_id, input: HookInput, tool_use_id?}\` → \`HookJSONOutput\` (section 6). The SDK looks up \`callback_id\` (\`hook_<n>\`) and calls the callback with \`(input, tool_use_id, {signal})\`.`,
  `- **mcp_message** \`{server_name, message: JSONRPCMessage}\` → \`{mcp_response}\`. A request is routed to the in-process server's transport and its reply returned. A notification is delivered and answered \`{jsonrpc:"2.0", result:{}, id:0}\`.`,
  `- **elicitation**: section 7.1.`,
  `- **request_user_dialog** \`{dialog_kind, payload, tool_use_id?}\` → \`onUserDialog({dialogKind, payload, toolUseID}, {signal, requestId})\` → \`UserDialogResult\`. With no handler the request stays unanswered.`, '', '---', '');

// =====================================================================
const HOOK_IN = unionList('HookInput');
P('## 6. Hooks', '',
  `**HookEvent:** the 33 events below; the list is identical in 0.3.311 and 0.3.325. Every \`*HookInput\` is \`BaseHookInput & {hook_event_name: <event>, …}\`. In Go each input is a flat struct, and the \`HookInput\` union is keyed by \`hook_event_name\`.`, '',
  `**BaseHookInput**`, '', tableFor('BaseHookInput'), '');
P(table(['HookEvent', 'Input type', 'Event-specific input fields', 'hookSpecificOutput type'], HOOK_IN.map((n) => {
  const ev = disc(n, ['hook_event_name']);
  const base = new Set((propsOf(B.get('BaseHookInput').type) || []).map(pname));
  const ps = (propsOf(look(n).type) || []).filter((p) => !base.has(pname(p)) && pname(p) !== 'hook_event_name').map((p) => pname(p) + (p.questionToken ? '?' : '') + ': ' + (inlineObj(p.type) ? '{…}' : txt(p.type).replace(/coreTypes\./g, '').slice(0, 60)));
  const out = B.has(ev + 'HookSpecificOutput') ? ev + 'HookSpecificOutput' : '—';
  return [ev, n, ps.join('; '), out];
})), '');
P(`**HookJSONOutput** = \`AsyncHookJSONOutput | SyncHookJSONOutput\`. Go \`HookJSONOutput\` is a union discriminated by the boolean \`async\`.`, '',
  `**AsyncHookJSONOutput**`, '', tableFor('AsyncHookJSONOutput'), '', `**SyncHookJSONOutput**`, '', tableFor('SyncHookJSONOutput'), '',
  `\`hookSpecificOutput\` is a union keyed by \`hookEventName\`; Go \`SyncHookJSONOutputHookSpecificOutput\`. The event-specific outputs:`, '',
  table(['Type', 'Fields'], [...B.keys()].filter((k) => /HookSpecificOutput$/.test(k)).sort().map((k) => [k, (propsOf(B.get(k).type) || []).map((p) => pname(p) + (p.questionToken ? '?' : '')).join(', ')])), '',
  `**Hook registration on the wire** (\`initialize.hooks\`): \`Partial<Record<HookEvent, SDKHookCallbackMatcher[]>>\`. Go uses \`map[HookEvent][]HookCallbackMatcher\`.`, '', tableFor('SDKHookCallbackMatcher'), '', '---', '');

// =====================================================================
P('## 7. MCP, elicitation, user messages, agents, initialize pieces', '',
  `### 7.1 Elicitation`, '', `**Request (CLI → host)**, \`elicitation\`, Go \`ElicitationControlRequest\`:`, '', tableFor('SDKControlElicitationRequest'), '',
  `The SDK maps it to the host-facing \`ElicitationRequest\`: \`serverName, message, mode?, url?, elicitationId?, requestedSchema?, title?, displayName?, description?\`. It calls \`onElicitation(request, {signal, requestId})\`.`, '',
  `**Response:** \`ElicitResult\` \`{action: 'accept'|'decline'|'cancel', content?: Record<string, string|number|boolean|string[]>, _meta?}\`. With no handler the SDK answers \`{action:"decline"}\`; \`null\` means answered out-of-band. Completion of a URL-mode elicitation arrives as \`system/elicitation_complete\`.`, '',
  `### 7.2 McpServerConfig variants`, '',
  `\`McpServerConfig\` is \`McpStdioServerConfig | McpSSEServerConfig | McpHttpServerConfig | McpSdkServerConfigWithInstance\`.`, '',
  `\`McpServerConfigForProcessTransport\` (the wire form, used by \`--mcp-config\`, \`mcp_set_servers\` and \`AgentDefinition.mcpServers\`) has \`McpSdkServerConfig\` instead of the instance form. \`McpServerStatusConfig\` adds \`McpClaudeAIProxyServerConfig\`.`, '',
  `In Go, \`McpServerConfig\` is the wire union keyed by \`type\`; a missing \`type\` means stdio.`, '');
for (const n of ['McpStdioServerConfig', 'McpSSEServerConfig', 'McpHttpServerConfig', 'McpSdkServerConfig', 'McpClaudeAIProxyServerConfig', 'McpServerToolPolicy', 'McpServerStatus', 'McpSetServersResult']) P(`**${n}**`, '', tableFor(n), '');
P(`### 7.3 SDKUserMessage (host → CLI on stdin; also echoed back)`, '', tableFor('SDKUserMessage'), '',
  `\`SDKUserMessageReplay\` has the same fields, with \`uuid\` and \`session_id\` required, plus \`isReplay: true\` and \`file_attachments?\`.`, '',
  `### 7.4 AgentDefinition (\`Options.agents\` / \`initialize.agents\`)`, '', tableFor('AgentDefinition'), '',
  `\`AgentMcpServerSpec = string | Record<string, McpServerConfigForProcessTransport>\`; Go keeps it as raw JSON.`, '',
  `### 7.5 SettingSource`, '', `\`'user' | 'project' | 'local'\`. \`ConfigScope\` is the same set; \`ResolvedSettingSource\` adds \`'managed' | 'flag'\`.`, '',
  `### 7.6 initialize`, '',
  `**Request** (host → CLI, always the first control request), Go \`InitializeRequest\`. sdk.mjs 0.3.311 also sends the undeclared keys \`appendSubagentSystemPrompt\`, \`codeRunningMcpTools\`, \`webSearchIsolationExemptMcpServers\`, \`rapidFollowupPreempt\`, \`forwardNestedSubagentToolCalls\` and \`workspaceTrust\` when set.`, '', tableFor('SDKControlInitializeRequest'), '',
  `**Response**, Go \`InitializeResponse\`:`, '', tableFor('SDKControlInitializeResponse'), '');
for (const n of ['ModelInfo', 'SlashCommand', 'AccountInfo', 'AgentInfo', 'SdkPluginConfig']) P(`**${n}**`, '', tableFor(n), '');
P('---', '');

// =====================================================================
P('## 8. Exported functions (0.3.311)', '',
  `The paths below assume the local filesystem; with \`sessionStore\` every function goes through the store API instead. The session-helper behaviour comes from reading the minified sdk.mjs. It was traced in 0.3.325, and its key constants are confirmed present in 0.3.311. Differences from 0.3.311 are noted.`, '',
  table(['Export', 'Signature', 'What it does'], [
    ['query', 'query({prompt: string | AsyncIterable<SDKUserMessage>, options?: Options}): Query', 'Spawns the CLI (section 2) and writes `initialize` first, then the prompt. Returns the Query generator (section 1).'],
    ['startup', 'startup({options?, initializeTimeoutMs?}?): Promise<WarmQuery>', 'Spawns the CLI and waits for the initialize response. Returns a WarmQuery whose single `query(prompt)` writes the prompt; `close()` kills it.'],
    ['prewarm *(alpha)*', 'prewarm({options?, initializeTimeoutMs?}?): Promise<SpareProcess>', 'Starts a parked spare with `--await-claim` (cwd defaults to a mkdtemp under `<config>/spares/`; resume, continue and forkSession are rejected). `claim({prompt, options: ClaimOptions})` sends `claim_session {cwd, permission_mode?, env?, additional_directories?, append_system_prompt?, title?, agents?, sdk_mcp_servers, include_initialize:true}`, then the settings overlay and the prompt.'],
    ['tool', 'tool(name, description, inputSchema: ZodRawShape, handler, extras?: {annotations?, searchHint?, alwaysLoad?}): SdkMcpToolDefinition', 'Returns `{name, description, inputSchema, handler, annotations, _meta}`. `_meta` holds `"anthropic/searchHint"` and/or `"anthropic/alwaysLoad": true`.'],
    ['createSdkMcpServer', 'createSdkMcpServer({name, version?, instructions?, tools?, alwaysLoad?, timeout?}): McpSdkServerConfigWithInstance', 'Creates an in-process `McpServer({name, version ?? "1.0.0"}, {capabilities: {tools}, instructions})` and registers each tool. The zod schema becomes JSON Schema draft-7 (input side). Server-level alwaysLoad stamps `anthropic/alwaysLoad` on every tool. Returns `{type:"sdk", name, instance, timeout?}`.'],
    ['listSessions', 'listSessions({dir?, limit?, offset?, includeWorktrees?, includeProgrammatic?, sessionStore?}?): Promise<SDKSessionInfo[]>', 'Scans `<CLAUDE_CONFIG_DIR or ~/.claude>/projects/<sanitized dir>/*.jsonl` (all projects when `dir` is omitted; with `dir`, also its git worktrees via `git worktree list --porcelain`). Reads a 64 KiB head and a 64 KiB tail of each file, skips sidechains and sessions continued in another file, and derives the SessionInfo fields. Sorts by lastModified desc and dedupes by sessionId. See 8.1.'],
    ['getSessionInfo', 'getSessionInfo(sessionId, {dir?, sessionStore?}?): Promise<SDKSessionInfo | undefined>', 'Same derivation for a single file. Undefined if the file is missing, a sidechain, or has no summary.'],
    ['getSessionMessages', 'getSessionMessages(sessionId, {dir?, limit?, offset?, includeSystemMessages?, sessionStore?}?): Promise<SessionMessage[]>', 'Parses the JSONL and rebuilds the main conversation chain through `parentUuid` from the newest main-chain leaf. Returns user/assistant entries in order (system ones only with includeSystemMessages), then applies offset/limit. Files over 5 MiB skip everything before the last plain compact boundary.'],
    ['getSubagentMessages', 'getSubagentMessages(sessionId, agentId, {dir?, limit?, offset?, sessionStore?}?): Promise<SessionMessage[]>', 'Reads `<projdir>/<sessionId>/subagents/agent-<agentId>.jsonl`, plus the `agent-<id>.meta.json` sidecar for `parent_tool_use_id` and `parent_agent_id`.'],
    ['listSubagents', 'listSubagents(sessionId, {dir?, sessionStore?}?): Promise<string[]>', 'Returns the ids of `agent-<id>.jsonl` files under `<sessionId>/subagents/`, recursively.'],
    ['renameSession', 'renameSession(sessionId, title, {dir?, sessionStore?}?): Promise<void>', 'Appends `{"type":"custom-title","customTitle":…,"sessionId":…}` to the transcript.'],
    ['tagSession', 'tagSession(sessionId, tag | null, opts?): Promise<void>', 'Appends `{"type":"tag","tag":…,"sessionId":…}`; null writes `""` and clears the tag. Non-ASCII tags are NFKC-normalized and stripped of format characters.'],
    ['deleteSession', 'deleteSession(sessionId, opts?): Promise<void>', 'Removes `<id>.jsonl`, then the `<id>/` directory.'],
    ['forkSession', 'forkSession(sessionId, {dir?, upToMessageId?, title?, sessionStore?}?): Promise<{sessionId}>', 'Copies the transcript to `<newId>.jsonl` with fresh UUIDs and a remapped parentUuid chain, optionally cut at upToMessageId. Each entry gets `forkedFrom`. Appends a custom-title entry; the default title is `"<title> (fork)"`.'],
    ['importSessionToStore *(alpha)*', 'importSessionToStore(sessionId, store, {dir?, includeSubagents?, batchSize?}?)', 'Streams a local transcript into `store.append` in batches.'],
    ['resolveSettings / filterEscalatingDefaultMode *(alpha)*', 'resolveSettings(opts?): Promise<ResolvedSettings>', 'Runs the CLI settings-merge engine in process, without spawning the CLI.'],
    ['foldSessionSummary *(alpha)*', '(prev, key, entries, {mtime?}) => SessionSummaryEntry', 'Helper for SessionStore implementations.'],
    ['InMemorySessionStore *(alpha)*, AbortError', 'classes', 'A test store, and the error thrown on abort.'],
    ['Constants', 'HOOK_EVENTS, EXIT_REASONS, BUILTIN_TOOL_NAMES *(alpha)*, LEGACY_TOOL_NAME_ALIASES *(alpha)*, SYSTEM_PROMPT_DYNAMIC_BOUNDARY = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__", USAGE_LIMIT_ERROR_PREFIXES / USAGE_TRANSITION_PREFIXES / USAGE_WARNING_PREFIXES / ORG_POLICY_LIMIT_PREFIXES *(alpha)*', 'Lists. 0.3.325 adds "Now using credits" to USAGE_TRANSITION_PREFIXES.'],
  ]), '',
  `The runtime bundle also exports names that the d.ts does not declare: \`CommandMatcher\`, \`DirectConnectError\`, \`DirectConnectTransport\`, \`FileIndex\` and \`parseDirectConnectUrl\` (seen in 0.3.325).`, '',
  `### 8.1 Session files: details for a Go port`, '',
  `**Project directory name**`,
  `- Start from \`realpath(dir)\`, NFC-normalized on macOS.`,
  `- Replace every character outside \`[a-zA-Z0-9]\` with \`-\`.`,
  `- If the result is longer than 200 characters, keep the first 200 and append \`"-" + base36(|javaHash(path)|)\`. The hash is the Java-style 32-bit \`h = 31*h + utf16 code unit\`.`,
  `- Readers also accept any directory starting with that 200-character prefix plus \`-\` whose transcript's \`cwd\` sanitizes to the same name.`,
  `- When \`CLAUDE_CONFIG_DIR\` is set, \`CLAUDE_CODE_PROJECT_DIR_NAME\` (matching \`^[A-Za-z0-9_-]{1,64}$\`) overrides the name.`, '',
  `**Session files**`,
  `- A session file is \`<uuid>.jsonl\`.`,
  `- A file whose first line has \`"isSidechain":true\` is a subagent transcript and is skipped.`,
  `- Empty files count as missing.`, '',
  `**SessionInfo fields.** Each comes from the 64 KiB head and tail windows:`,
  `- \`customTitle\`: the last \`{type:"custom-title"}\` entry in the tail, else the sidecar \`<id>/custom-title.json\`, else the same entry in the head.`,
  `  - 0.3.325 also falls back to \`ai-title\` entries; 0.3.311 does not.`,
  `- \`summary\`: \`customTitle\`, else the last \`"lastPrompt"\` value, else the last \`"summary"\` value, else \`firstPrompt\`.`,
  `- \`firstPrompt\`: the first real user text in the head.`,
  `  - Lines with \`tool_result\`, \`isMeta\` or \`isCompactSummary\` are skipped, as are texts that start with an XML-like tag or "[Request interrupted by user".`,
  `  - \`<command-name>\` and \`<bash-input>\` are special-cased.`,
  `  - The text is capped at 200 characters plus \`…\`.`,
  `- \`gitBranch\`, \`cwd\` and \`createdAt\` come from the first or last raw \`"key":"value"\` occurrence. For \`createdAt\` that is the first \`timestamp\`.`,
  `- \`tag\`: the last \`{type:"tag"}\` entry in the tail.`,
  `- \`lastModified\`: the file mtime in milliseconds.`,
  `- \`fileSize\`: the file size in bytes.`, '',
  `**Filtering.** \`includeProgrammatic:false\` drops sessions whose \`entrypoint\` is \`sdk-cli\`, \`sdk-ts\` or \`sdk-py\`, or whose \`sessionKind\` is \`daemon\` or \`daemon-worker\`.`, '',
  `**Pagination.** With limit or offset, files are sorted by mtime before parsing and evaluated in batches of 32.`, '', '---', '');

// =====================================================================
P('## 9. Not in 0.3.311', '',
  `These items appear in 0.3.325 but not in 0.3.311. Every one is additive. \`types.go\` keeps each of them, marked \`// since … (not in 0.3.311)\`. A 2.1.311 CLI never sends them, and a 0.3.311 host must not rely on them.`, '');
const grouped = {};
for (const s of sinceLog) { const k = s.kind; (grouped[k] ||= []).push(s); }
P(table(['Kind', 'Item', 'Since'], sinceLog.filter((s, i, a) => a.findIndex((x) => x.name === s.name && x.owner === s.owner) === i).map((s) => [s.kind, (s.owner ? s.owner + '.' : '') + s.name, s.version])), '',
  `**Grouped by surface**`, '',
  `- **Message variants:** \`system:session_writes_lost\` (\`SDKSessionWritesLostMessage\`), since 0.3.317.`,
  `- **Message fields:**`,
  `  - \`credential_expires_at\` on all result messages (0.3.318)`,
  `  - \`invalid_config_file\` on error results (0.3.325)`,
  `  - \`plugin_errors[].managed_lock\` on \`system:init\` (0.3.325)`,
  `- **Control subtypes:** \`set_has_subscribers\` (host → CLI; \`has_subscribers: boolean | null\`), 0.3.312-0.3.325.`,
  `- **Control request fields:**`,
  `  - \`mcp_call.check_permissions\` (0.3.312-0.3.325)`,
  `  - \`reload_plugins.refuse_if_busy\` (0.3.325)`,
  `  - \`initialize.hooks[Event][].onFailure\` (0.3.317)`,
  `- **Control response fields:**`,
  `  - \`initialize\` → \`hook_on_failure_supported\` (0.3.317)`,
  `  - \`read_file\` → \`lossy\` (0.3.317)`,
  `  - \`list_permission_rules\` → \`state.withheld\` (0.3.312-0.3.325)`,
  `  - the control_response envelope → \`pending_elicitation_requests\` (0.3.312-0.3.325)`,
  `- **Enum values:** \`SDKStartupFailureReason\` gains \`sandbox_required_unavailable\` (0.3.314), \`initialize_not_first\` (0.3.317), \`config_file_invalid\` (0.3.318) and \`managed_settings_not_delivered\` (0.3.312-0.3.325).`,
  `- **Query methods:** \`reloadPlugins({refuseIfBusy})\` option (0.3.325). No method is new.`,
  `- **Options:** none new in the d.ts.`,
  `  - \`HookCallbackMatcher.onFailure\` (0.3.317) is new on the hook matcher.`,
  `  - With \`onFailure:"block"\`, sdk.mjs 0.3.325 checks \`hook_on_failure_supported\` and \`hooks_applied\` and fails the query if they are missing. 0.3.311 has neither.`,
  `- **Exports:** \`archiveSession\` and \`unarchiveSession\` (0.3.325), \`matchDeniedModels\` and \`DeniedModelsMatch\` (0.3.323). Also new: \`SDKSessionInfo.archived\` (0.3.325) and \`ResolvedSettings.machinePolicy\` (0.3.323).`,
  `- **Behaviour in sdk.mjs:**`,
  `  - 0.3.325 adds an internal \`managedSettingsDelivery: 'initialize'\` (\`--managed-settings-in-initialize\`, \`initialize.managedSettings\`).`,
  `  - It adds \`--await-initialize\` on resume or continue with the bundled binary (0.3.321).`,
  `  - It stops auto-approving sandboxed Bash unless \`autoAllowBashIfSandboxed: true\` (0.3.325).`,
  `  - It falls back to \`ai-title\` for session titles.`,
  `- **Settings keys:**`,
  `  - Added: \`autoMode\`, \`disableReplyCheck\`, \`disableWindowsBashCommandSplit\`, \`skipAutoPermissionPrompt\` and \`useAutoModeDuringPlan\`.`,
  `  - Added inside \`permissions\`: \`blockWritesOutsideWorkingDirectories\`, \`blockReadsExemptPaths\`, \`workingDirectoriesAsGiven\` and \`disableAutoMode\`.`,
  `  - Added on marketplace sources: \`engines\`.`,
  `  - Go keeps Settings as a map, so nothing changes there.`,
  `- **sdk-tools.d.ts** (tool input/output schemas; not modelled in types.go):`,
  `  - ArtifactInput \`+page_token\`; BashOutput \`+backgroundWakesOwner\`; CronDeleteOutput \`+endedLoopWakeup\`; GlobOutput and GrepOutput \`+timedOut\`.`,
  `  - Shape changes in ArtifactOutput, AskUserQuestionOutput.annotations, FileReadOutput, ProjectsOutput, ProposeSkillsInput.proposals, PushNotificationInput.status and TaskCreateOutput.task.`, '', '---', '');

// =====================================================================
P('## 10. Go mapping notes', '',
  `**Unions.** There is one pattern everywhere: a struct with the discriminator string(s) and \`Raw\`, plus \`As<Variant>()\`, \`Decode()\` and \`New<Union>(v)\`. This covers:`,
  `- \`Message\` and \`StdoutMessage\` (type, subtype)`,
  `- \`ControlRequestInner\` and \`ControlResponseResponse\` (subtype)`,
  `- \`ContentBlock\`, \`StreamEvent\` and \`ContentBlockDelta\` (type)`,
  `- \`PermissionResult\` and \`UserDialogResult\` (behavior)`,
  `- \`PermissionUpdate\`, \`McpServerConfig\`, \`McpServerStatusConfig\`, \`ThinkingConfig\` and \`OutputFormat\` (type)`,
  `- \`MessageOrigin\` (kind)`,
  `- \`HookInput\` (hook_event_name)`,
  `- the hookSpecificOutput union (hookEventName)`,
  `- \`HookJSONOutput\`, the one hand-written union, discriminated by the boolean \`async\``, '',
  `**Lossless JSON.**`,
  `- Every wire struct keeps \`Raw\`. MarshalJSON merges back undeclared keys, and restores declared keys whose value is unchanged (explicit null, "", []).`,
  `- Decoding tolerates JSON type mismatches.`,
  `- The round trip was checked on 942 captured CLI frames (2.1.278, 2.1.311 and 2.1.325 runs, both directions) and on typed control-response payloads. No frame lost data.`, '',
  `**What does not map cleanly**`,
  `- **TS \`number\`** becomes \`float64\`. Fields that are integers by name become \`int\`: tokens, counts, indexes, bytes, sizes, turns and attempts. A fractional value in an \`int\` field decodes as 0 and keeps its original in Raw.`,
  `- **Mixed unions keep raw JSON:**`,
  `  - \`string | string[] | {…}\`: \`systemPrompt\`, \`tools\`, \`settings\`, \`skills\`, \`MessageParam.content\`, \`AgentDefinition.effort\`, \`AgentMcpServerSpec\`, tool_result content`,
  `  - \`unknown\`: \`tool_input\`, \`tool_response\`, \`structured_output\`, \`tool_use_result\``,
  `  - Helpers: \`MessageParam.Text()\` and \`Blocks()\`.`,
  `- **\`user\` vs replay:** \`SDKUserMessage\` and \`SDKUserMessageReplay\` share \`type:"user"\`. Replay is detected by \`isReplay:true\`.`,
  `- **Results:** \`SDKResultError\` covers four subtypes in one struct.`,
  `- **Stdio type:** a \`McpStdioServerConfig\` with no \`type\` is stdio.`,
  `- **External types are hand-modelled:** BetaMessage, BetaRawMessageStreamEvent, MessageParam, BetaUsage, ElicitResult, CallToolResult, ToolAnnotations, and SandboxSettings (a zod schema).`,
  `- **Settings** (about 180 optional keys) is \`map[string]any\`.`,
  `- **Callbacks** cannot live in the types: canUseTool, hooks, onElicitation, onUserDialog, stderr, spawnClaudeCodeProcess, sessionStore, tool handlers, abortController.`,
  `  - Options carries booleans or callback ids instead.`,
  `  - The host answers the corresponding control requests (5.3–5.4, 7.1) with the response types here.`,
  `- **Generic and conditional TS types** (\`applyFlagSettings\`' mapped type, \`InferShape\`, zod schemas) have no Go equivalent: \`map[string]any\` or omitted.`,
  `- **Undeclared shapes** use Go types marked "observed" or "sdk.mjs": \`error_code\`, hidden initialize keys, and responses for mcp_status, background_tasks, list_models, get_session_cost, get_settings, set_permission_mode and set_mcp_permission_mode_override.`,
  `- **Undocumented responses:** get_binary_version, file_suggestions, mcp_call and register_repo_root have no declared or observed response shape. Read them as raw JSON.`, '');

process.stdout.write(L.join('\n') + '\n');
