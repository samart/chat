import type { BetaMessage } from '@anthropic-ai/sdk/resources/beta/messages/messages.mjs';
import type { BetaRawMessageStreamEvent } from '@anthropic-ai/sdk/resources/beta/messages/messages.mjs';
import type { BetaUsage } from '@anthropic-ai/sdk/resources/beta/messages/messages.mjs';
import type { CallToolResult } from '@modelcontextprotocol/sdk/types.js';
import type { ElicitResult } from '@modelcontextprotocol/sdk/types.js';
import type { JSONRPCMessage } from '@modelcontextprotocol/sdk/types.js';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { MessageParam } from '@anthropic-ai/sdk/resources';
import type { Readable } from 'stream';
import type { ToolAnnotations } from '@modelcontextprotocol/sdk/types.js';
import type { UUID } from 'crypto';
import type { Writable } from 'stream';
import * as z from 'zod/v4';
import type { ZodRawShape } from 'zod/v3';
import type { ZodRawShape as ZodRawShape_2 } from 'zod/v4';

export declare class AbortError extends Error {
}

// Information about the logged in user's account.
export declare type AccountInfo = {
    email?: string;
    organization?: string;
    subscriptionType?: string;
    tokenSource?: string;
    apiKeySource?: string;
    // Active API backend.
    apiProvider?: 'firstParty' | 'bedrock' | 'vertex' | 'foundry' | 'anthropicAws' | 'anthropicGoogleCloud' | 'mantle' | 'gateway';
};

// Definition for a custom subagent that can be invoked via the Agent tool.
export declare type AgentDefinition = {
    // Natural language description of when to use this agent
    description: string;
    // Array of allowed tool names.
    tools?: string[];
    // Array of tool names to explicitly disallow for this agent.
    disallowedTools?: string[];
    // The agent's system prompt
    prompt: string;
    // Model alias (e.g.
    model?: string;
    mcpServers?: AgentMcpServerSpec[];
    // Experimental: Critical reminder added to system prompt
    criticalSystemReminder_EXPERIMENTAL?: string;
    // Array of skill names to preload into the agent context
    skills?: string[];
    // Auto-submitted as the first user turn when this agent is the main thread agent.
    initialPrompt?: string;
    // Maximum number of agentic turns (API round-trips) before stopping
    maxTurns?: number;
    // Token count at which this agent compacts its own conversation when it runs as a subagent.
    autoCompactWindow?: number;
    // Run this agent as a background task (non-blocking, fire-and-forget) when invoked
    background?: boolean;
    // Run this agent without the user, project and local CLAUDE.md instruction files when it runs as a subagent; managed policy files are kept.
    omitClaudeMd?: boolean;
    // Scope for auto-loading agent memory files.
    memory?: 'user' | 'project' | 'local';
    // Reasoning effort level for this agent.
    effort?: ('low' | 'medium' | 'high' | 'xhigh' | 'max') | number;
    // Permission mode controlling how tool executions are handled
    permissionMode?: PermissionMode;
    // Agent type auto-spawned as a background observer whenever this agent runs, or a comma-separated list of up to four, none of whose names has a comma.
    observer?: string;
    // Supplemental postamble appended (after the harness-owned default) to each activity digest sent to the observer.
    observerMessage?: string;
};

// Information about an available subagent that can be invoked via the Task tool.
export declare type AgentInfo = {
    // Agent type identifier (e.g., "Explore")
    name: string;
    // Description of when to use this agent
    description: string;
    // Model this agent uses: an alias or model ID, or 'inherit' for the parent's model.
    model?: string;
};

export declare type AgentMcpServerSpec = string | Record<string, McpServerConfigForProcessTransport>;

export declare type AnyZodRawShape = ZodRawShape | ZodRawShape_2;

// Where the credential used for API requests came from: 'ANTHROPIC_API_KEY' (environment variable), 'apiKeyHelper' (the configured helper command), '/login managed key' (an API key created and stored by /login with an Anthropic Console account), or 'none' (no API key in use - e.g.
export declare type ApiKeySource = 'ANTHROPIC_API_KEY' | 'apiKeyHelper' | '/login managed key' | 'none' | 'user' | 'project' | 'org' | 'temporary' | 'oauth';

// Archive a session.
export declare function archiveSession(_sessionId: string, _options?: SessionMutationOptions): Promise<void>;

export declare type AsyncHookJSONOutput = {
    async: true;
    asyncTimeout?: number;
};

export declare type BackgroundTaskSummary = {
    id: string;
    // Friendly task-type label (e.g.
    type: string;
    status: string;
    // Free-text description.
    description: string;
    // Shell command line.
    command?: string;
    // Subagent type name.
    agent_type?: string;
    // MCP server name.
    server?: string;
    // MCP tool name.
    tool?: string;
    // Workflow name.
    name?: string;
};

export declare type BaseHookInput = {
    session_id: string;
    transcript_path: string;
    cwd: string;
    // UUID correlating a user prompt with all subsequent events until the next prompt.
    prompt_id?: string;
    permission_mode?: string;
    // Subagent identifier.
    agent_id?: string;
    // Agent type name (e.g., "general-purpose", "code-reviewer").
    agent_type?: string;
    // Reasoning effort applied to the current turn.
    effort?: {
        // Active effort level for the current turn (e.g., "low", "medium", "high", "xhigh", "max"), after any silent downgrade for the selected model.
        level: string;
    };
};

export declare type BaseOutputFormat = {
    type: OutputFormatType;
};

// Names of built-in tools, as permission rules, hook matchers, `tools` and `disallowedTools` spell them. [@alpha]
export declare const BUILTIN_TOOL_NAMES: readonly ['Agent', 'Artifact', 'AskUserQuestion', 'Bash', 'ClaudeDesign', 'CronCreate', 'CronDelete', 'CronList', 'Edit', 'EndConversation', 'EnterPlanMode', 'EnterWorktree', 'ExitPlanMode', 'ExitWorktree', 'Glob', 'Grep', 'ListAgents', 'ListMcpResourcesTool', 'LSP', 'Monitor', 'NotebookEdit', 'OfferChromeSetup', 'PowerShell', 'Projects', 'propose_skills', 'ProposeGoal', 'PushNotification', 'Read', 'ReadMcpResourceDirTool', 'ReadMcpResourceTool', 'ReadNotifications', 'RemoteTrigger', 'ReportFindings', 'ScheduleWakeup', 'SendFeedback', 'SendMessage', 'SendUserFile', 'SendUserMessage', 'ShareOnboardingGuide', 'ShowOnboardingRolePicker', 'Skill', 'SubagentHandback', 'TaskCreate', 'TaskGet', 'TaskList', 'TaskStop', 'TaskUpdate', 'TodoWrite', 'ToolSearch', 'WaitForMcpServers', 'WebFetch', 'WebSearch', 'Workflow', 'Write'];

// Permission callback function for controlling tool usage.
export declare type CanUseTool = (toolName: string, input: Record<string, unknown>, options: {
    // Signaled if the operation should be aborted.
    signal: AbortSignal;
    // Suggestions for updating permissions so that the user will not be prompted again for this tool during this session.
    suggestions?: PermissionUpdate[];
    // The file path that triggered the permission request, if applicable.
    blockedPath?: string;
    // For `mcp__*` tools: the MCP server serving the tool and where its definition came from.
    mcpServer?: {
        name: string;
        source: string;
    };
    // Explains why this permission request was triggered.
    decisionReason?: string;

    // Full permission prompt sentence rendered by the bridge (e.g.
    title?: string;
    // Short noun phrase for the tool action (e.g.
    displayName?: string;
    // Human-readable subtitle from the bridge (e.g.
    description?: string;
    // The ask must not be approvable by a single stray keystroke: open the prompt on its decline option and offer no one-key approve shortcut.
    defaultToNo?: boolean;
    // The ask must not offer a persistent "don't ask again" choice: its rule grants more than this ask's action, or a plugin started this call.
    suppressAlwaysAllowRule?: boolean;
    // Unique identifier for this specific tool call within the assistant message.
    toolUseID: string;
    // If running within the context of a sub-agent, the sub-agent's ID.
    agentID?: string;

    // The control_request envelope's `request_id`.
    requestId: string;
    // Set when a user-configured ask RULE (permissions.ask) forced this prompt while the ask carries the tool's own decisionReason, or when an ask rule reaches an MCP tool only under another name of its server (such as a connector's id or its claude.ai name), not the one given here.
    matchedAskRule?: {
        source: string;
        toolName: string;
        ruleContent?: string;
    };

}) => Promise<PermissionResult | null>;

// The per-session half of {@link Options}: what a host only knows once the user has picked a folder and started the session. [@alpha]
export declare type ClaimOptions = {
    // The session's working directory (required).
    cwd: string;
    // Per-session environment additions.
    env?: Record<string, string>;
    // As `--add-dir`; relative entries resolve against `cwd`, missing ones are skipped.
    additionalDirectories?: string[];
    model?: string;
    // As `Options.permissionMode`, sent as part of the claim itself: a mode the process cannot take (`bypassPermissions` on a spare prewarmed without `allowDangerouslySkipPermissions`, one the folder's settings disable) refuses the claim — `claimed` rejects with `permission_mode_not_claimable` (or `claim_failed`) and the prompt gets the `not_claimed` error result — so a session never runs under a different mode than the one asked for.
    permissionMode?: PermissionMode;
    maxThinkingTokens?: number | null;
    // Flag-tier settings overlay (e.g.
    settings?: Parameters<Query['applyFlagSettings']>[0];
    appendSystemPrompt?: string;
    title?: string;
    agents?: Record<string, AgentDefinition>;

};

export declare type ConfigChangeHookInput = BaseHookInput & {
    hook_event_name: 'ConfigChange';
    source: 'user_settings' | 'project_settings' | 'local_settings' | 'policy_settings' | 'skills';
    file_path?: string;
};

// Config scope for settings.
export declare type ConfigScope = 'local' | 'user' | 'project';

// The request failed or was rejected (unknown subtype, invalid arguments, or an error while handling it).
declare type ControlErrorResponse = {
    subtype: 'error';
    // The request_id of the control_request this answers.
    request_id: string;
    // Human-readable failure description.
    error: string;

    // can_use_tool requests this CLI process has issued and not yet resolved, so a client joining an already-initialized session learns about in-flight prompts.
    pending_permission_requests?: SDKControlRequest[];
    // request_user_dialog requests this CLI process has issued and not yet resolved (sibling of pending_permission_requests, with the same inherited-prompt caveat), so a client joining an already-initialized session can re-arm in-flight dialogs.
    pending_user_dialog_requests?: SDKControlRequest[];
    // elicitation requests that this CLI process sent and that are not yet answered or withdrawn (sibling of pending_permission_requests), each as it was sent, so a client that joins after an MCP server asked can still show the question.
    pending_elicitation_requests?: SDKControlRequest[];
};

// The request was handled.
declare type ControlResponse = {
    subtype: 'success';
    // The request_id of the control_request this answers.
    request_id: string;
    // The success payload, shaped as documented for the answered request's subtype; absent or {} for requests that are merely acknowledged.
    response?: Record<string, unknown>;
    // can_use_tool requests this CLI process has issued and not yet resolved, so a client joining an already-initialized session learns about in-flight prompts.
    pending_permission_requests?: SDKControlRequest[];
    // request_user_dialog requests this CLI process has issued and not yet resolved (sibling of pending_permission_requests, with the same inherited-prompt caveat), so a client joining an already-initialized session can re-arm in-flight dialogs.
    pending_user_dialog_requests?: SDKControlRequest[];
    // elicitation requests that this CLI process sent and that are not yet answered or withdrawn (sibling of pending_permission_requests), each as it was sent, so a client that joins after an MCP server asked can still show the question.
    pending_elicitation_requests?: SDKControlRequest[];
};

declare namespace coreTypes {
    export {
        SandboxCredentialsConfig,
        SandboxFilesystemConfig,
        SandboxIgnoreViolations,
        SandboxNetworkConfig,
        SandboxSettings,
        NonNullableUsage,
        HOOK_EVENTS,
        EXIT_REASONS,
        SYSTEM_PROMPT_DYNAMIC_BOUNDARY,
        AccountInfo,
        AgentDefinition,
        AgentInfo,
        AgentMcpServerSpec,
        ApiKeySource,
        AsyncHookJSONOutput,
        BackgroundTaskSummary,
        BaseHookInput,
        BaseOutputFormat,
        ConfigChangeHookInput,
        ConfigScope,
        CwdChangedHookInput,
        CwdChangedHookSpecificOutput,
        DirectoryAddedHookInput,
        ElicitationHookInput,
        ElicitationHookSpecificOutput,
        ElicitationResultHookInput,
        ElicitationResultHookSpecificOutput,
        ExitReason,
        FastModeDisabledReason,
        FastModeState,
        FileChangedHookInput,
        FileChangedHookSpecificOutput,
        HookEvent,
        HookInput,
        HookJSONOutput,
        HookPermissionDecision,
        InstructionsLoadedHookInput,
        JsonSchemaOutputFormat,
        McpClaudeAIProxyServerConfig,
        McpHttpServerConfig,
        McpSSEServerConfig,
        McpSdkServerConfig,
        McpServerConfigForProcessTransport,
        McpServerProvenance,
        McpServerStatusConfig,
        McpServerStatus,
        McpServerToolPolicy,
        McpSetServersResult,
        McpStdioServerConfig,
        MessageDisplayHookInput,
        MessageDisplayHookSpecificOutput,
        ModelInfo,
        ModelUsage,
        NotificationHookInput,
        NotificationHookSpecificOutput,
        OutputFormat,
        OutputFormatType,
        PermissionBehavior,
        PermissionDecisionClassification,
        PermissionDeniedHookInput,
        PermissionDeniedHookSpecificOutput,
        PermissionMode,
        PermissionRequestHookInput,
        PermissionRequestHookSpecificOutput,
        PermissionResult,
        PermissionRuleValue,
        PermissionUpdateDestination,
        PermissionUpdate,
        PostCompactHookInput,
        PostModelSwitchHookInput,
        PostModelSwitchHookSpecificOutput,
        PostToolBatchHookInput,
        PostToolBatchHookSpecificOutput,
        PostToolBatchToolCall,
        PostToolUseFailureHookInput,
        PostToolUseFailureHookSpecificOutput,
        PostToolUseHookInput,
        PostToolUseHookSpecificOutput,
        PreCompactHookInput,
        PreModelSwitchHookInput,
        PreModelSwitchHookSpecificOutput,
        PreToolUseHookInput,
        PreToolUseHookSpecificOutput,
        RewindFilesResult,
        SDKAPIRetryMessage,
        SDKActiveGoalMessage,
        SDKAssistantMessageError,
        SDKAssistantMessage,
        SDKAuthStatusMessage,
        SDKBackgroundTasksChangedMessage,
        SDKCommandsChangedMessage,
        SDKCompactBoundaryMessage,
        SDKContextUsageCategory,
        SDKContextUsage,
        SDKControlRequestProgressMessage,
        SDKConversationResetMessage,
        SDKDeferredToolUse,
        SDKElicitationCompleteMessage,
        SDKFilesPersistedEvent,
        SDKHookProgressMessage,
        SDKHookResponseMessage,
        SDKHookStartedMessage,
        SDKInformationalMessage,
        SDKLocalCommandOutputMessage,
        SDKMcpResourceLink,
        SDKMemoryRecallMessage,
        SDKMessageOrigin,
        SDKMessage,
        SDKMirrorErrorMessage,
        SDKModelRefusalFallbackMessage,
        SDKModelRefusalNoFallbackMessage,
        SDKNotificationMessage,
        SDKPartialAssistantMessage,
        SDKPermissionDenial,
        SDKPermissionDeniedMessage,
        SDKPluginInstallMessage,
        SDKPromptSuggestionMessage,
        SDKRateLimitEvent,
        SDKRateLimitInfo,
        SDKResultError,
        SDKResultMessage,
        SDKResultSuccess,
        SDKSessionInfo,
        SDKSessionStateChangedMessage,
        SDKSessionWritesLostMessage,
        SDKSettingsParseError,
        SDKStartupFailureReason,
        SDKStatusMessage,
        SDKStatus,
        SDKSystemMessage,
        SDKTaskNotificationMessage,
        SDKTaskProgressMessage,
        SDKTaskStartedMessage,
        SDKTaskUpdatedMessage,
        SDKThinkingTokensMessage,
        SDKToolProgressMessage,
        SDKToolUseSummaryMessage,
        SDKUsageReport,
        SDKUserMessageReplay,
        SDKUserMessage,
        SDKWorkerShuttingDownMessage,
        SdkBeta,
        SdkPluginConfig,
        SessionCronSummary,
        SessionEndHookInput,
        SessionStartHookInput,
        SessionStartHookSpecificOutput,
        SettingSource,
        SetupHookInput,
        SetupHookSpecificOutput,
        SlashCommand,
        StopFailureHookInput,
        StopHookInput,
        StopHookSpecificOutput,
        SubagentStartHookInput,
        SubagentStartHookSpecificOutput,
        SubagentStopHookInput,
        SubagentStopHookSpecificOutput,
        SyncHookJSONOutput,
        TaskCompletedHookInput,
        TaskCreatedHookInput,
        TeammateIdleHookInput,
        TerminalReason,
        ThinkingAdaptive,
        ThinkingConfig,
        ThinkingDisabled,
        ThinkingEnabled,
        UserPromptExpansionHookInput,
        UserPromptExpansionHookSpecificOutput,
        UserPromptSubmitHookInput,
        UserPromptSubmitHookSpecificOutput,
        WorktreeCreateHookInput,
        WorktreeCreateHookSpecificOutput,
        WorktreeRemoveHookInput
    }
}

// Creates an MCP server instance that can be used with the SDK transport.
export declare function createSdkMcpServer(_options: CreateSdkMcpServerOptions): McpSdkServerConfigWithInstance;

declare type CreateSdkMcpServerOptions = {
    name: string;
    version?: string;
    // Server instructions returned from `initialize` and surfaced to the model as an MCP instructions block.
    instructions?: string;
    tools?: Array<SdkMcpToolDefinition<any>>;
    // When true, all tools from this server are always included in the prompt and never deferred behind tool search.
    alwaysLoad?: boolean;
    // Per-server tool-call timeout in milliseconds.
    timeout?: number;
};

export declare type CwdChangedHookInput = BaseHookInput & {
    hook_event_name: 'CwdChanged';
    old_cwd: string;
    new_cwd: string;
};

export declare type CwdChangedHookSpecificOutput = {
    hookEventName: 'CwdChanged';
    watchPaths?: string[];
};

// Delete a session.
export declare function deleteSession(_sessionId: string, _options?: SessionMutationOptions): Promise<void>;

// What {@link matchDeniedModels} answers. [@alpha]
export declare type DeniedModelsMatch = {
    // For each model asked about, in order: whether an entry blocks it.
    denied: boolean[];
    // The CLI's notes on entries: why one was ignored, or what one blocks where that may surprise.
    warnings: string[];
};

export declare type DirectoryAddedHookInput = BaseHookInput & {
    hook_event_name: 'DirectoryAdded';
    // Absolute path of the directory that was added.
    directory: string;
    // How the directory was added: "slash_command" for /add-dir, "register_repo_root" for the SDK control_request.
    source: 'slash_command' | 'register_repo_root';
};

// Effort level for controlling how much thinking/reasoning Claude applies.
export declare type EffortLevel = 'low' | 'medium' | 'high' | 'xhigh' | 'max';

// Hook input for the Elicitation event.
export declare type ElicitationHookInput = BaseHookInput & {
    hook_event_name: 'Elicitation';
    mcp_server_name: string;
    message: string;
    mode?: 'form' | 'url';
    url?: string;
    elicitation_id?: string;
    requested_schema?: Record<string, unknown>;
};

// Hook-specific output for the Elicitation event.
export declare type ElicitationHookSpecificOutput = {
    hookEventName: 'Elicitation';
    action?: 'accept' | 'decline' | 'cancel';
    content?: Record<string, unknown>;
};

// Elicitation request from an MCP server, asking the SDK consumer for user input.
export declare type ElicitationRequest = {
    // Name of the MCP server requesting elicitation
    serverName: string;
    // Message to display to the user
    message: string;
    // Elicitation mode: 'form' for structured input, 'url' for browser-based auth
    mode?: 'form' | 'url';
    // URL to open (only for 'url' mode)
    url?: string;
    // Elicitation ID for correlating URL elicitations with completion notifications (URL mode only)
    elicitationId?: string;
    // JSON Schema for the requested input (only for 'form' mode)
    requestedSchema?: Record<string, unknown>;
    // Permission-display title from MCP `_meta['anthropic/permissionDisplay']` — header for elicitation-driven permission prompts
    title?: string;
    // Short tool/server label from MCP `_meta['anthropic/permissionDisplay'].displayName`
    displayName?: string;
    // Permission-display subtitle from MCP `_meta['anthropic/permissionDisplay'].description`
    description?: string;
};

// Elicitation response from the SDK consumer.
export declare type ElicitationResult = ElicitResult;

// Hook input for the ElicitationResult event.
export declare type ElicitationResultHookInput = BaseHookInput & {
    hook_event_name: 'ElicitationResult';
    mcp_server_name: string;
    elicitation_id?: string;
    mode?: 'form' | 'url';
    action: 'accept' | 'decline' | 'cancel';
    content?: Record<string, unknown>;
};

// Hook-specific output for the ElicitationResult event.
export declare type ElicitationResultHookSpecificOutput = {
    hookEventName: 'ElicitationResult';
    action?: 'accept' | 'decline' | 'cancel';
    content?: Record<string, unknown>;
};

export declare const EXIT_REASONS: readonly ['clear', 'resume', 'logout', 'prompt_input_exit', 'other'];

export declare type ExitReason = 'clear' | 'resume' | 'logout' | 'prompt_input_exit' | 'other';

// Why fast mode can't serve right now.
export declare type FastModeDisabledReason = 'free' | 'preference' | 'extra_usage_disabled' | 'network_error' | 'unknown' | 'not_first_party' | 'disabled_by_env' | 'model_not_allowed' | 'sdk_opt_in_required' | 'pending';

// Fast mode state: off, in cooldown after rate limit, or actively enabled.
export declare type FastModeState = 'off' | 'cooldown' | 'on';

export declare type FileChangedHookInput = BaseHookInput & {
    hook_event_name: 'FileChanged';
    file_path: string;
    event: 'change' | 'add' | 'unlink';
};

export declare type FileChangedHookSpecificOutput = {
    hookEventName: 'FileChanged';
    watchPaths?: string[];
};

// Apply the same trust-tier filter the CLI applies before honoring escalating permission modes from settings: if `permissions.defaultMode` is escalating (`bypassPermissions`/`auto`/`acceptEdits`) AND was set by a repo-committed tier (`project`), drop it from the returned `effective`. [@alpha]
export declare function filterEscalatingDefaultMode(_resolved: ResolvedSettings): Settings;

// Fold a batch of appended entries into the running summary for `key`. [@alpha]
export declare function foldSessionSummary(prev: SessionSummaryEntry | undefined, key: SessionKey, entries: SessionStoreEntry[], options?: {
    mtime?: number;
}): SessionSummaryEntry;

// Fork a session into a new branch with fresh UUIDs.
export declare function forkSession(_sessionId: string, _options?: ForkSessionOptions): Promise<ForkSessionResult>;

// Options for forking a session into a new branch.
export declare type ForkSessionOptions = SessionMutationOptions & {
    // Slice transcript up to the message whose `uuid` field equals this value (inclusive).
    upToMessageId?: string;
    // Custom title for the fork.
    title?: string;
};

// Result of a fork operation.
export declare type ForkSessionResult = {
    // New session UUID.
    sessionId: string;
};

// Reads metadata for a single session by ID.
export declare function getSessionInfo(_sessionId: string, _options?: GetSessionInfoOptions): Promise<SDKSessionInfo | undefined>;

// Options for getSessionInfo.
export declare type GetSessionInfoOptions = {
    // Project directory path (same semantics as `listSessions({ dir })`).
    dir?: string;
    // When provided, load session info from this store instead of the local filesystem. [@alpha]
    sessionStore?: SessionStore;
};

// Reads a session's conversation messages from its JSONL transcript file.
export declare function getSessionMessages(_sessionId: string, _options?: GetSessionMessagesOptions): Promise<SessionMessage[]>;

// Options for retrieving session messages.
export declare type GetSessionMessagesOptions = {
    // Project directory to find the session in.
    dir?: string;
    // Maximum number of messages to return.
    limit?: number;
    // Number of messages to skip from the start.
    offset?: number;
    // When true, include system messages (e.g., compact boundaries, informational notices) in the returned list alongside user/assistant messages.
    includeSystemMessages?: boolean;
    // When provided, load session messages from this store instead of the local filesystem. [@alpha]
    sessionStore?: SessionStore;
};

// Reads a subagent's conversation messages from its JSONL transcript file.
export declare function getSubagentMessages(_sessionId: string, _agentId: string, _options?: GetSubagentMessagesOptions): Promise<SessionMessage[]>;

// Options for retrieving subagent messages.
export declare type GetSubagentMessagesOptions = {
    // Project directory to find the session in.
    dir?: string;
    // Maximum number of messages to return.
    limit?: number;
    // Number of messages to skip from the start.
    offset?: number;
    // When provided, load subagent messages from this store instead of the local filesystem. [@alpha]
    sessionStore?: SessionStore;
};

export declare const HOOK_EVENTS: readonly ['PreToolUse', 'PostToolUse', 'PostToolUseFailure', 'PostToolBatch', 'Notification', 'UserPromptSubmit', 'UserPromptExpansion', 'SessionStart', 'SessionEnd', 'Stop', 'StopFailure', 'SubagentStart', 'SubagentStop', 'PreCompact', 'PostCompact', 'PreModelSwitch', 'PostModelSwitch', 'PermissionRequest', 'PermissionDenied', 'Setup', 'TeammateIdle', 'TaskCreated', 'TaskCompleted', 'Elicitation', 'ElicitationResult', 'ConfigChange', 'WorktreeCreate', 'WorktreeRemove', 'InstructionsLoaded', 'CwdChanged', 'FileChanged', 'DirectoryAdded', 'MessageDisplay'];

// Hook callback function for responding to events during execution.
export declare type HookCallback = (input: HookInput, toolUseID: string | undefined, options: {
    signal: AbortSignal;
}) => Promise<HookJSONOutput>;

// Hook callback matcher containing hook callbacks and optional pattern matching.
export declare interface HookCallbackMatcher {
    matcher?: string;
    hooks: HookCallback[];
    // Timeout in seconds for all hooks in this matcher
    timeout?: number;
    // What a failure of these hooks does: one throws, or returns output that fails validation.
    onFailure?: 'continue' | 'block';
}

export declare type HookEvent = 'PreToolUse' | 'PostToolUse' | 'PostToolUseFailure' | 'PostToolBatch' | 'Notification' | 'UserPromptSubmit' | 'UserPromptExpansion' | 'SessionStart' | 'SessionEnd' | 'Stop' | 'StopFailure' | 'SubagentStart' | 'SubagentStop' | 'PreCompact' | 'PostCompact' | 'PreModelSwitch' | 'PostModelSwitch' | 'PermissionRequest' | 'PermissionDenied' | 'Setup' | 'TeammateIdle' | 'TaskCreated' | 'TaskCompleted' | 'Elicitation' | 'ElicitationResult' | 'ConfigChange' | 'WorktreeCreate' | 'WorktreeRemove' | 'InstructionsLoaded' | 'CwdChanged' | 'FileChanged' | 'DirectoryAdded' | 'MessageDisplay';

export declare type HookInput = PreToolUseHookInput | PostToolUseHookInput | PostToolUseFailureHookInput | PostToolBatchHookInput | PermissionDeniedHookInput | NotificationHookInput | UserPromptSubmitHookInput | UserPromptExpansionHookInput | SessionStartHookInput | SessionEndHookInput | StopHookInput | StopFailureHookInput | SubagentStartHookInput | SubagentStopHookInput | PreCompactHookInput | PostCompactHookInput | PreModelSwitchHookInput | PostModelSwitchHookInput | PermissionRequestHookInput | SetupHookInput | TeammateIdleHookInput | TaskCreatedHookInput | TaskCompletedHookInput | ElicitationHookInput | ElicitationResultHookInput | ConfigChangeHookInput | InstructionsLoadedHookInput | WorktreeCreateHookInput | WorktreeRemoveHookInput | CwdChangedHookInput | FileChangedHookInput | DirectoryAddedHookInput | MessageDisplayHookInput;

export declare type HookJSONOutput = AsyncHookJSONOutput | SyncHookJSONOutput;

export declare type HookPermissionDecision = 'allow' | 'deny' | 'ask' | 'defer';

// Copy a local JSONL session into a SessionStore. [@alpha]
export declare function importSessionToStore(_sessionId: string, _store: SessionStore, _options?: ImportSessionToStoreOptions): Promise<void>;

// Options for importing a local JSONL session into a SessionStore. [@alpha]
export declare type ImportSessionToStoreOptions = {
    // Project directory path (same semantics as `listSessions({ dir })`).
    dir?: string;
    // If true, also import subagent transcripts.
    includeSubagents?: boolean;
    // Maximum entries per `store.append()` call.
    batchSize?: number;
};

export declare type InferShape<T extends AnyZodRawShape> = {
    [K in keyof T]: T[K] extends {
        _output: infer O;
    } ? O : never;
} & {};

// In-memory SessionStore implementation for testing and development. [@alpha]
export declare class InMemorySessionStore implements SessionStore {
    private store;
    private mtimes;
    private summaries;
    private lastMtime;
    private keyToString;
    append(key: SessionKey, entries: SessionStoreEntry[]): Promise<void>;
    load(key: SessionKey): Promise<SessionStoreEntry[] | null>;
    listSessions(projectKey: string): Promise<Array<{
        sessionId: string;
        mtime: number;
    }>>;
    listSessionSummaries(projectKey: string): Promise<SessionSummaryEntry[]>;
    delete(key: SessionKey): Promise<void>;
    listSubkeys(key: {
        projectKey: string;
        sessionId: string;
    }): Promise<string[]>;
    // Test helper -- get all entries for a key
    getEntries(key: SessionKey): SessionStoreEntry[];
    // Test helper -- number of stored sessions (main transcripts only)
    get size(): number;
    // Test helper -- clear all stored data
    clear(): void;
}

export declare type InstructionsLoadedHookInput = BaseHookInput & {
    hook_event_name: 'InstructionsLoaded';
    file_path: string;
    memory_type: 'User' | 'Project' | 'Local' | 'Managed';
    load_reason: 'session_start' | 'nested_traversal' | 'path_glob_match' | 'include' | 'compact';
    globs?: string[];
    trigger_file_path?: string;
    parent_file_path?: string;
};

export declare type JsonSchemaOutputFormat = {
    type: 'json_schema';
    schema: Record<string, unknown>;
};

// Former names that permission rules, hook matchers, `tools` and `disallowedTools` still accept, each mapped to the tool's current name. [@alpha]
export declare const LEGACY_TOOL_NAME_ALIASES: Readonly<Record<string, string>>;

// List sessions with metadata.
export declare function listSessions(_options?: ListSessionsOptions): Promise<SDKSessionInfo[]>;

// Options for listing sessions.
export declare type ListSessionsOptions = {
    // Directory to list sessions for.
    dir?: string;
    // Maximum number of sessions to return.
    limit?: number;
    // Number of sessions to skip from the start of the sorted result set.
    offset?: number;
    // When `dir` is provided and the directory is inside a git repository, include sessions from all git worktree paths.
    includeWorktrees?: boolean;
    // Include programmatic/headless sessions (SDK entrypoints `sdk-cli`, `sdk-ts`, `sdk-py`) and daemon/daemon-worker sessions.
    includeProgrammatic?: boolean;
    // When provided, list sessions from this store instead of the local filesystem. [@alpha]
    sessionStore?: SessionStore;
};

// Lists subagent IDs for a given session by scanning the subagents directory.
export declare function listSubagents(_sessionId: string, _options?: ListSubagentsOptions): Promise<string[]>;

// Options for listing subagents.
export declare type ListSubagentsOptions = {
    // Project directory to find the session in.
    dir?: string;
    // When provided, list subagents from this store instead of the local filesystem. [@alpha]
    sessionStore?: SessionStore;
};

// Say which of `models` a managed `deniedModels` list blocks, without spawning the Claude CLI. [@alpha]
export declare function matchDeniedModels(_models: readonly string[], _deniedModels: readonly string[] | undefined): Promise<DeniedModelsMatch>;

export declare type McpClaudeAIProxyServerConfig = {
    type: 'claudeai-proxy';
    url: string;
    id: string;
    // Per-server tool-call timeout in milliseconds.
    timeout?: number;
};

export declare type McpHttpServerConfig = {
    type: 'http';
    url: string;
    headers?: Record<string, string>;
    tools?: McpServerToolPolicy[];
    // Per-server tool-call timeout in milliseconds.
    timeout?: number;

    // When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself lists with _meta anthropic/alwaysLoad set to false.
    alwaysLoad?: boolean;

};

export declare type McpSdkServerConfig = {
    type: 'sdk';
    name: string;
    // Per-server tool-call timeout in milliseconds.
    timeout?: number;

    // Per-tool permission policy the host declares for this server's tools (not the tool definitions passed to createSdkMcpServer).
    tools?: McpServerToolPolicy[];
};

// MCP SDK server config with an actual McpServer instance.
export declare type McpSdkServerConfigWithInstance = McpSdkServerConfig & {
    instance: McpServer;
};

// Union of all MCP server config types, including those with non-serializable instances.
export declare type McpServerConfig = McpStdioServerConfig | McpSSEServerConfig | McpHttpServerConfig | McpSdkServerConfigWithInstance;

export declare type McpServerConfigForProcessTransport = McpStdioServerConfig | McpSSEServerConfig | McpHttpServerConfig | McpSdkServerConfig;

// The MCP server serving this tool, for `mcp__*` tools: `name` is the server's config key (for `source: "sdk"`, exactly the name the SDK host registered in `sdkMcpServers` / `mcp_set_servers`; for any other source, the key as authored in that configuration — untrusted text, the same value `mcp_status` and system/init report, to be escaped before display), `source` is where its definition came from — `sdk` (an in-process server the SDK host runs; only the host can register one, so a configured server of the same name never reads `sdk`), `plugin` (a server a plugin ships or registers at runtime), or a config scope (`user`, `project`, `local`, `dynamic` for --mcp-config / `mcp_set_servers` process servers, `managed`, `enterprise`, `claudeai`, `agent`).
export declare type McpServerProvenance = {
    name: string;
    // sdk | plugin | user | project | local | dynamic | managed | enterprise | claudeai | agent — an open set; treat unknown values as an unrecognized configured source, never as sdk.
    source: string;
};

// Status information for an MCP server connection.
export declare type McpServerStatus = {
    // Server name as configured
    name: string;
    // Current connection status
    status: 'connected' | 'failed' | 'needs-auth' | 'pending' | 'disabled';
    // Server information (available when connected)
    serverInfo?: {
        name: string;
        version: string;
    };
    // Error message (available when status is 'failed')
    error?: string;

    // Server configuration (includes URL for HTTP/SSE servers)
    config?: McpServerStatusConfig;

    // Configuration scope (e.g., project, user, local, claudeai, managed)
    scope?: string;
    // Where the server definition came from: sdk (an in-process server the SDK host registered — only the host can register one), plugin (a server a plugin ships or registers at runtime), or the config scope (user, project, local, dynamic, managed, enterprise, claudeai, agent).
    source?: string;
    // Tools provided by this server (available when connected)
    tools?: {
        name: string;
        description?: string;
        annotations?: {
            readOnly?: boolean;
            destructive?: boolean;
            openWorld?: boolean;
        };
        // The MCP Apps (SEP-1865) members of the tool's `_meta`, for a host that renders the tool's `ui://` resource, under the keys the server used: `ui` (an object: `resourceUri`, a `ui://` string; `visibility`, an array of 'model' | 'app'; and any other member the server sent) and the deprecated flat `ui/resourceUri` (a `ui://` string).
        _meta?: Record<string, unknown>;
    }[];

};

export declare type McpServerStatusConfig = McpServerConfigForProcessTransport | McpClaudeAIProxyServerConfig;

// Per-tool permission policy for an MCP server: carried on mcp_set_servers for remote (http/sse) servers, and on mcp_set_servers or initialize (sdkMcpServerConfigs) for host-registered sdk servers, where only always_ask and always_deny take effect.
export declare type McpServerToolPolicy = {
    name: string;
    permission_policy?: 'always_allow' | 'always_ask' | 'always_deny';
    // Org admin's per-tool ceiling.
    org_max_permission?: 'allow' | 'ask' | 'blocked';
};

// Result of a setMcpServers operation.
export declare type McpSetServersResult = {
    // Names of servers that were added
    added: string[];
    // Names of servers that were removed
    removed: string[];
    // Map of server names to error messages for servers that failed to connect
    errors: Record<string, string>;
};

export declare type McpSSEServerConfig = {
    type: 'sse';
    url: string;
    headers?: Record<string, string>;
    tools?: McpServerToolPolicy[];
    // Per-server tool-call timeout in milliseconds.
    timeout?: number;

    // When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself lists with _meta anthropic/alwaysLoad set to false.
    alwaysLoad?: boolean;

};

export declare type McpStdioServerConfig = {
    type?: 'stdio';
    command: string;
    args?: string[];
    env?: Record<string, string>;
    // Per-server tool-call timeout in milliseconds.
    timeout?: number;
    // When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself lists with _meta anthropic/alwaysLoad set to false.
    alwaysLoad?: boolean;

};

// Hook input for the MessageDisplay event.
export declare type MessageDisplayHookInput = BaseHookInput & {
    hook_event_name: 'MessageDisplay';
    // UUID of the current turn.
    turn_id: string;
    // UUID of the assistant message being displayed.
    message_id: string;
    // Zero-based index of this delta within the message.
    index: number;
    // True on the message's last flush.
    final: boolean;
    // The newly completed lines since the prior flush.
    delta: string;
};

// Hook-specific output for the MessageDisplay event.
export declare type MessageDisplayHookSpecificOutput = {
    hookEventName: 'MessageDisplay';
    // Text displayed in place of the delta.
    displayContent?: string;
};

// Information about an available model.
export declare type ModelInfo = {
    // Model identifier to use in API calls
    value: string;
    // Canonical wire model id this row's `value` resolves to (e.g.
    resolvedModel?: string;
    // Human-readable display name
    displayName: string;
    // Description of the model's capabilities
    description: string;
    // Whether this model supports effort levels
    supportsEffort?: boolean;
    // Available effort levels for this model
    supportedEffortLevels?: ('low' | 'medium' | 'high' | 'xhigh' | 'max')[];
    // Whether this model supports adaptive thinking (Claude decides when and how much to think)
    supportsAdaptiveThinking?: boolean;
    // Whether this model supports fast mode
    supportsFastMode?: boolean;
    // Whether this model supports auto mode
    supportsAutoMode?: boolean;

};

export declare type ModelUsage = {
    inputTokens: number;
    outputTokens: number;
    // Thinking tokens, already counted inside outputTokens.
    thinkingTokens?: number;
    cacheReadInputTokens: number;
    cacheCreationInputTokens: number;
    // Cache writes with a 5-minute lifetime, already counted inside cacheCreationInputTokens.
    cacheCreationEphemeral5mInputTokens?: number;
    // Cache writes with a 1-hour lifetime, already counted inside cacheCreationInputTokens.
    cacheCreationEphemeral1hInputTokens?: number;
    webSearchRequests: number;
    costUSD: number;
    contextWindow: number;
    maxOutputTokens: number;
    // Canonical model id used for the pricing lookup (e.g.
    canonicalModel?: string;
    // API provider that served this model (e.g.
    provider?: string;
    // Which price table the most recent request for this model was priced at: Claude Code's built-in list prices ('list'), the organization's managed-settings modelPricing rates or multiplier ('managed'), or neither ('unknown' — no pricing row and no built-in price matched the model ID, so costUSD is a guess at the default model's rate).
    costBasis?: 'list' | 'managed' | 'unknown';
};

export declare type NonNullableUsage = {
    [K in keyof BetaUsage]: K extends 'fallback_credit' ? BetaUsage[K] : NonNullable<BetaUsage[K]>;
};

export declare type NotificationHookInput = BaseHookInput & {
    hook_event_name: 'Notification';
    message: string;
    title?: string;
    notification_type: string;
};

export declare type NotificationHookSpecificOutput = {
    hookEventName: 'Notification';
    additionalContext?: string;
};

// Callback for handling MCP elicitation requests.
export declare type OnElicitation = (request: ElicitationRequest, options: {
    signal: AbortSignal;
    // The control_request envelope's `request_id`.
    requestId: string;
}) => Promise<ElicitationResult | null>;

// Callback for handling `request_user_dialog` control requests.
export declare type OnUserDialog = (request: UserDialogRequest, options: {
    signal: AbortSignal;
    // The control_request envelope's `request_id`.
    requestId: string;
}) => Promise<UserDialogResult | null>;

// Options for the query function.
export declare type Options = {
    // Controller for cancelling the query.
    abortController?: AbortController;
    // Additional directories Claude can access beyond the current working directory.
    additionalDirectories?: string[];
    // The trusted checkout `cwd` is a worktree of.
    projectConfigRoot?: string;
    // Agent name for the main thread.
    agent?: string;
    // Programmatically define custom subagents that can be invoked via the Agent tool.
    agents?: Record<string, AgentDefinition>;
    // List of tool names that are auto-allowed without prompting for permission.
    allowedTools?: string[];
    // Custom permission handler for controlling tool usage.
    canUseTool?: CanUseTool;
    // Continue the most recent conversation in the current directory instead of starting a new one.
    continue?: boolean;
    // Current working directory for the session.
    cwd?: string;
    // List of tool names that are disallowed.
    disallowedTools?: string[];
    // Map of tool-name aliases applied before name resolution.
    toolAliases?: Record<string, string>;

    // Specify the base set of available built-in tools.
    tools?: string[] | {
        type: 'preset';
        preset: 'claude_code';
    };
    // Environment variables for the Claude Code process.
    env?: {
        [envVar: string]: string | undefined;
    };
    // JavaScript runtime to use for executing Claude Code.
    executable?: 'bun' | 'deno' | 'node';
    // Additional arguments to pass to the JavaScript runtime executable.
    executableArgs?: string[];
    // Additional CLI arguments to pass to Claude Code.
    extraArgs?: Record<string, string | null>;
    // Fallback model(s) to use if the primary model is overloaded or unavailable.
    fallbackModel?: string;
    // Enable file checkpointing to track file changes during the session.
    enableFileCheckpointing?: boolean;
    // Per-tool configuration for built-in tools.
    toolConfig?: ToolConfig;
    // When true, resumed sessions will fork to a new session ID rather than continuing the previous session.
    forkSession?: boolean;
    // Enable beta features.
    betas?: SdkBeta[];
    // Hook callbacks for responding to various events during execution.
    hooks?: Partial<Record<HookEvent, HookCallbackMatcher[]>>;
    // Callback for handling MCP elicitation requests.
    onElicitation?: OnElicitation;
    // Callback for handling `request_user_dialog` control requests — blocking dialogs the CLI asks the host to render.
    onUserDialog?: OnUserDialog;
    // Dialog kinds this consumer's `onUserDialog` can actually render (`request_user_dialog` `dialog_kind` values, e.g.
    supportedDialogKinds?: string[];
    // Declares that this consumer renders a per-task stop control wired to the `stop_task` control request, so the user can stop an individual background task.
    perTaskStopAffordance?: boolean;

    // When false, disables session persistence to disk. [@default]
    persistSession?: boolean;
    // Mirror session transcripts to an external store. [@alpha]
    sessionStore?: SessionStore;
    // Controls how aggressively transcript entries are flushed to {@link Options.sessionStore}. [@alpha]
    sessionStoreFlush?: SessionStoreFlush;
    // Timeout for each `sessionStore.load()` / `sessionStore.listSubkeys()` call during resume materialization. [@alpha,@default]
    loadTimeoutMs?: number;
    // Include hook lifecycle events in the output stream. [@default]
    includeHookEvents?: boolean;
    // Include partial/streaming message events in the output.
    includePartialMessages?: boolean;
    // Forward subagent text and thinking blocks as assistant/user messages with `parent_tool_use_id` set.
    forwardSubagentText?: boolean;
    // Send every user message with `client_composed: true`, so the CLI delivers the prompt text as written: no `@path` file-mention expansion and no slash-command dispatch. [@default]
    verbatimPrompts?: boolean;
    // Controls Claude's thinking/reasoning behavior.
    thinking?: ThinkingConfig;
    // Controls how much effort Claude puts into its response.
    effort?: EffortLevel;
    // Maximum number of tokens the model can use for its thinking/reasoning process. [@deprecated]
    maxThinkingTokens?: number;
    // Maximum number of conversation turns before the query stops.
    maxTurns?: number;
    // Maximum budget in USD for the query.
    maxBudgetUsd?: number;
    // API-side task budget in tokens. [@alpha]
    taskBudget?: {
        total: number;
    };
    // MCP (Model Context Protocol) server configurations.
    mcpServers?: Record<string, McpServerConfig>;
    // Claude model to use.
    model?: string;
    // Output format configuration for structured responses.
    outputFormat?: OutputFormat;
    // Path to the Claude Code executable.
    pathToClaudeCodeExecutable?: string;
    // Permission mode for the session.
    permissionMode?: PermissionMode;
    // Custom workflow instructions for plan mode.
    planModeInstructions?: string;
    // Must be set to `true` when using `permissionMode: 'bypassPermissions'`.
    allowDangerouslySkipPermissions?: boolean;
    // MCP tool name to use for permission prompts.
    permissionPromptToolName?: string;
    // Who answers permission prompts.
    permissionPrompts?: 'host' | 'none';
    // Load plugins for this session.
    plugins?: SdkPluginConfig[];
    // How `plugins` reach the Claude Code process.
    pluginDelivery?: 'argv' | 'initialize';

    // Enable prompt suggestions.
    promptSuggestions?: boolean;
    // Enable periodic AI-generated progress summaries for running subagents.
    agentProgressSummaries?: boolean;

    // Session ID to resume.
    resume?: string;
    // Use a specific session ID for the conversation instead of an auto-generated one.
    sessionId?: string;
    // When resuming, only resume messages up to and including the message with this UUID.
    resumeSessionAt?: string;
    // With `resumeSessionAt`: declares the prompt UUID of the turn this truncating resume intends to discard.
    resumeDropsTurn?: string;
    // Sandbox settings for command execution isolation.
    sandbox?: SandboxSettings;
    // Additional settings to apply.
    settings?: string | Settings;
    // Policy-tier settings supplied by the spawning parent process.
    managedSettings?: Settings;

    // Control which filesystem settings to load.
    settingSources?: SettingSource[];
    // Skills to enable for the main session.
    skills?: string[] | 'all';
    // Enable debug mode for the Claude Code process.
    debug?: boolean;
    // Write debug logs to a specific file path.
    debugFile?: string;
    // Callback for stderr output from the Claude Code process.
    stderr?: (data: string) => void;
    // Only use MCP servers passed via the `mcpServers` option (and servers declared by explicitly-passed agent definitions in `agents`), ignoring all other MCP configurations: project `.mcp.json`, user settings, plugins, and on-disk agent frontmatter — including subagent frontmatter MCP.
    strictMcpConfig?: boolean;
    // System prompt configuration.
    systemPrompt?: string | string[] | {
        type: 'custom';
        prompt: string | string[];
        snapshot?: boolean;
    } | {
        type: 'preset';
        preset: 'claude_code';
        append?: string;
        excludeDynamicSections?: boolean;
        snapshot?: boolean;
    };
    // Custom title for a new session.
    title?: string;

    // Custom function to spawn the Claude Code process.
    spawnClaudeCodeProcess?: (options: SpawnOptions) => SpawnedProcess;
};

// Emitted on the same severity:'error' path as the usage-limit bucket, but the condition is an org policy, not an exhausted limit — consumers route these to org-disabled presentation, never the usage-limit card. [@alpha]
export declare const ORG_POLICY_LIMIT_PREFIXES: readonly ['This service is disabled for your org'];

export declare type OutputFormat = JsonSchemaOutputFormat;

export declare type OutputFormatType = 'json_schema';

export declare type PermissionBehavior = 'allow' | 'deny' | 'ask';

// Classification of this permission decision for telemetry.
export declare type PermissionDecisionClassification = 'user_temporary' | 'user_permanent' | 'user_reject';

export declare type PermissionDeniedHookInput = BaseHookInput & {
    hook_event_name: 'PermissionDenied';
    tool_name: string;
    tool_input: unknown;
    tool_use_id: string;
    reason: string;
    mcp_server?: McpServerProvenance;
};

export declare type PermissionDeniedHookSpecificOutput = {
    hookEventName: 'PermissionDenied';
    retry?: boolean;
};

// Permission mode for controlling how tool executions are handled.
export declare type PermissionMode = 'default' | 'acceptEdits' | 'bypassPermissions' | 'plan' | 'dontAsk' | 'auto';

export declare type PermissionRequestHookInput = BaseHookInput & {
    hook_event_name: 'PermissionRequest';
    tool_name: string;
    tool_input: unknown;
    permission_suggestions?: PermissionUpdate[];
    mcp_server?: McpServerProvenance;

};

export declare type PermissionRequestHookSpecificOutput = {
    hookEventName: 'PermissionRequest';
    decision: {
        behavior: 'allow';
        updatedInput?: Record<string, unknown>;
        updatedPermissions?: PermissionUpdate[];
    } | {
        behavior: 'deny';
        message?: string;
        interrupt?: boolean;
    };
};

export declare type PermissionResult = {
    behavior: 'allow';
    updatedInput?: Record<string, unknown>;
    updatedPermissions?: PermissionUpdate[];
    toolUseID?: string;
    decisionClassification?: PermissionDecisionClassification;
} | {
    behavior: 'deny';
    message: string;
    interrupt?: boolean;
    toolUseID?: string;
    decisionClassification?: PermissionDecisionClassification;
};

export declare type PermissionRuleValue = {
    toolName: string;
    ruleContent?: string;
};

export declare type PermissionUpdate = {
    type: 'addRules';
    rules: PermissionRuleValue[];
    behavior: PermissionBehavior;
    destination: PermissionUpdateDestination;
} | {
    type: 'replaceRules';
    rules: PermissionRuleValue[];
    behavior: PermissionBehavior;
    destination: PermissionUpdateDestination;
} | {
    type: 'removeRules';
    rules: PermissionRuleValue[];
    behavior: PermissionBehavior;
    destination: PermissionUpdateDestination;
} | {
    type: 'setMode';
    mode: PermissionMode;
    destination: PermissionUpdateDestination;
} | {
    type: 'addDirectories';
    directories: string[];
    destination: PermissionUpdateDestination;
} | {
    type: 'removeDirectories';
    directories: string[];
    destination: PermissionUpdateDestination;
};

export declare type PermissionUpdateDestination = 'userSettings' | 'projectSettings' | 'localSettings' | 'session' | 'cliArg';

// Which policy sub-source supplied a `'managed'` value. [@alpha]
export declare type PolicySettingsOrigin = 'helper' | 'remote' | 'plist' | 'hklm' | 'file' | 'parent' | 'hkcu';

export declare type PostCompactHookInput = BaseHookInput & {
    hook_event_name: 'PostCompact';
    trigger: 'manual' | 'auto';
    // The conversation summary produced by compaction
    compact_summary: string;
};

export declare type PostModelSwitchHookInput = (BaseHookInput & {
    hook_event_name: 'PostModelSwitch';
}) & {
    // Resolved model id the session was running before the switch
    from_model: string;
    // Resolved model id the session runs after the switch
    to_model: string;
    // What was asked for (alias such as "opus", a full id, or null for "default")
    requested_model: string | null;
    // command: /model <name>, the /config Model row, or enabling fast mode when that promotes the model; picker: an interactive model picker; sdk: headless set_model (SDK, Remote Control, IDE); auto: automatic fallback or other programmatic change; resume: model restored while resuming a session
    source: 'command' | 'picker' | 'sdk' | 'auto' | 'resume';
    // Prompt tokens the next request re-sends: the last main-thread response's input + cache_read + cache_creation + output tokens (0 before the first response; for a server-side tool loop, its last iteration's window, not the summed totals)
    context_tokens: number;
    // Whether the current model's prompt cache is likely still warm (a switch then forfeits it)
    prompt_cache_warm: boolean;
    cache_ttl: '5m' | '1h';
    // Estimated cost of re-caching context_tokens on to_model at its cache-write rate — the managed modelPricing when set, otherwise list price; excludes the response
    estimated_cache_write_usd: number;
    // configured: priced at the managed modelPricing setting; catalog: list price; default: to_model unknown, the default tier was assumed
    pricing: 'configured' | 'catalog' | 'default';
};

export declare type PostModelSwitchHookSpecificOutput = {
    hookEventName: 'PostModelSwitch';
    // Reaches the model with the next request the new model serves
    additionalContext?: string;
};

// Hook input for the PostToolBatch event.
export declare type PostToolBatchHookInput = BaseHookInput & {
    hook_event_name: 'PostToolBatch';
    tool_calls: PostToolBatchToolCall[];
};

export declare type PostToolBatchHookSpecificOutput = {
    hookEventName: 'PostToolBatch';
    additionalContext?: string;
};

export declare type PostToolBatchToolCall = {
    tool_name: string;
    tool_input: unknown;
    tool_use_id: string;
    tool_response?: unknown;
};

export declare type PostToolUseFailureHookInput = BaseHookInput & {
    hook_event_name: 'PostToolUseFailure';
    tool_name: string;
    tool_input: unknown;
    tool_use_id: string;
    error: string;
    is_interrupt?: boolean;
    // Tool execution time in milliseconds.
    duration_ms?: number;
    mcp_server?: McpServerProvenance;
};

export declare type PostToolUseFailureHookSpecificOutput = {
    hookEventName: 'PostToolUseFailure';
    additionalContext?: string;
};

export declare type PostToolUseHookInput = BaseHookInput & {
    hook_event_name: 'PostToolUse';
    tool_name: string;
    tool_input: unknown;
    tool_response: unknown;
    tool_use_id: string;
    // Tool execution time in milliseconds.
    duration_ms?: number;
    mcp_server?: McpServerProvenance;
};

export declare type PostToolUseHookSpecificOutput = {
    hookEventName: 'PostToolUse';
    additionalContext?: string;
    // Host-asserted context shown to the auto-mode permission classifier alongside this tool call's result.
    classifierContext?: string;
    // Replaces the tool output before it is sent to the model
    updatedToolOutput?: unknown;
    // Replaces the output for MCP tools only.
    updatedMCPToolOutput?: unknown;
};

export declare type PreCompactHookInput = BaseHookInput & {
    hook_event_name: 'PreCompact';
    trigger: 'manual' | 'auto';
    custom_instructions: string | null;
};

export declare type PreModelSwitchHookInput = (BaseHookInput & {
    hook_event_name: 'PreModelSwitch';
}) & {
    // Resolved model id the session was running before the switch
    from_model: string;
    // Resolved model id the session runs after the switch
    to_model: string;
    // What was asked for (alias such as "opus", a full id, or null for "default")
    requested_model: string | null;
    // command: /model <name>, the /config Model row, or enabling fast mode when that promotes the model; picker: an interactive model picker; sdk: headless set_model (SDK, Remote Control, IDE)
    source: 'command' | 'picker' | 'sdk';
    // Prompt tokens the next request re-sends: the last main-thread response's input + cache_read + cache_creation + output tokens (0 before the first response; for a server-side tool loop, its last iteration's window, not the summed totals)
    context_tokens: number;
    // Whether the current model's prompt cache is likely still warm (a switch then forfeits it)
    prompt_cache_warm: boolean;
    cache_ttl: '5m' | '1h';
    // Estimated cost of re-caching context_tokens on to_model at its cache-write rate — the managed modelPricing when set, otherwise list price; excludes the response
    estimated_cache_write_usd: number;
    // configured: priced at the managed modelPricing setting; catalog: list price; default: to_model unknown, the default tier was assumed
    pricing: 'configured' | 'catalog' | 'default';
};

export declare type PreModelSwitchHookSpecificOutput = {
    hookEventName: 'PreModelSwitch';
    // Same contract as PreToolUse: allow proceeds (skipping the interactive cache-miss confirm), deny cancels the switch, ask asks the user to confirm (a headless session refuses instead)
    permissionDecision?: 'allow' | 'deny' | 'ask';
    permissionDecisionReason?: string;
};

export declare type PreToolUseHookInput = BaseHookInput & {
    hook_event_name: 'PreToolUse';
    tool_name: string;
    tool_input: unknown;
    tool_use_id: string;
    mcp_server?: McpServerProvenance;
};

export declare type PreToolUseHookSpecificOutput = {
    hookEventName: 'PreToolUse';
    permissionDecision?: HookPermissionDecision;
    permissionDecisionReason?: string;
    updatedInput?: Record<string, unknown>;
    additionalContext?: string;

};

// Start a Claude Code process as a parked **spare** before the session it will serve is known, and bind it later with {@link SpareProcess.claim}. [@alpha]
export declare function prewarm(_params?: {
    options?: Options;
    // The most time, in milliseconds, to wait for the spare process to initialize; it counts throughout.
    initializeTimeoutMs?: number;
}): Promise<SpareProcess>;

// Per-key provenance entry. [@alpha]
export declare type ProvenanceEntry = {
    source: ResolvedSettingSource;
    // Absolute path to the settings file, for filesystem-backed sources.
    path?: string;
    // Which policy sub-source supplied the value, when `source === 'managed'`.
    policyOrigin?: PolicySettingsOrigin;
};

// Query interface with methods for controlling query execution.
export declare interface Query extends AsyncGenerator<SDKMessage, void> {
    // Control Requests The following methods are control requests, and are only supported when streaming input/output is used.
    // Interrupt the current query execution.
    interrupt(): Promise<SDKControlInterruptResponse | undefined>;
    // Change the permission mode for the current session.
    setPermissionMode(mode: PermissionMode): Promise<void>;
    // Pin (or clear, with mode:null) a per-MCP-server permission-mode override.
    setMcpPermissionModeOverride(serverName: string, mode: 'default' | 'auto' | null): Promise<{
        warning?: string;
    }>;

    // Change the model used for subsequent responses.
    setModel(model?: string): Promise<void>;
    // Set the maximum number of thinking tokens the model is allowed to use when generating its response. [@deprecated]
    setMaxThinkingTokens(maxThinkingTokens: number | null, thinkingDisplay?: 'summarized' | 'omitted' | 'highlights' | null): Promise<void>;
    // Merge settings into the flag settings layer.
    applyFlagSettings(settings: {
        [K in keyof Settings]?: K extends 'effortLevel' ? EffortLevel | null : Settings[K] | null;
    }): Promise<void>;
    // Merge settings into a settings FILE through the CLI's own writer — the same path /config uses (canonical store root, gitignore upkeep, hardened write) — and live-apply them.
    updateSettings(source: 'localSettings' | 'userSettings', settings: Record<string, unknown>): Promise<void>;
    // Get the full initialization result, including supported commands, models, account info, and output style configuration: the first-connect answer, or, on a Query returned by a pre-warmed spare's `claim()`, the claimed session's.
    initializationResult(): Promise<SDKControlInitializeResponse>;
    // Re-send the `initialize` control request to an already-running CLI.
    reinitialize(): Promise<SDKControlInitializeResponse>;

    // Get the list of available skills for the current session.
    supportedCommands(): Promise<SlashCommand[]>;
    // Get the list of available models.
    supportedModels(): Promise<ModelInfo[]>;
    // Get the list of available subagents for the current session.
    supportedAgents(): Promise<AgentInfo[]>;
    // Get the current status of all configured MCP servers.
    mcpServerStatus(): Promise<McpServerStatus[]>;
    // Get a breakdown of current context window usage by category (system prompt, tools, messages, MCP tools, memory files, etc.).
    getContextUsage(opts?: {
        detail?: 'summary' | 'full';
    }): Promise<SDKControlGetContextUsageResponse>;
    // Get the structured data behind the `/usage` command: session cost and token usage totals plus claude.ai plan rate-limit utilization windows (5-hour, 7-day, per-model) when available.
    usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET(opts?: {
        skipBehaviors?: boolean;
    }): Promise<SDKControlGetUsageResponse>;

    // Read a file from the session's filesystem for the remote sidebar viewer.
    readFile(path: string, options?: {
        maxBytes?: number;
        encoding?: 'utf-8' | 'base64';
    }): Promise<SDKControlReadFileResponse | null>;

    // Reload plugins from disk and return the refreshed commands, agents, plugins, and MCP server status.
    reloadPlugins(options?: {
        holdOnCacheImpact?: boolean;
        addPlugins?: SdkPluginConfig[];
        refuseIfBusy?: boolean;
    }): Promise<SDKControlReloadPluginsResponse>;
    // Reload skills from disk and return the refreshed skill list.
    reloadSkills(): Promise<SDKControlReloadSkillsResponse>;
    // Re-read the output-style directories from disk and return the refreshed style names.
    reloadOutputStyles(): Promise<SDKControlReloadOutputStylesResponse>;
    // Get information about the authenticated account.
    accountInfo(): Promise<AccountInfo>;
    // Rewind tracked files to their state at a specific user message.
    rewindFiles(userMessageId: string, options?: {
        dryRun?: boolean;
    }): Promise<RewindFilesResult>;
    // Seed the CLI's readFileState cache with a path+mtime entry.
    seedReadState(path: string, mtime: number): Promise<void>;

    // Reconnect an MCP server by name.
    reconnectMcpServer(serverName: string): Promise<void>;
    // Enable or disable an MCP server by name.
    toggleMcpServer(serverName: string, enabled: boolean): Promise<void>;
    // Read one MCP Apps (SEP-1865) UI resource from a connected MCP server the CLI itself dialed, so the host can render a tool's widget. [@alpha]
    readMcpResource(serverName: string, uri: string): Promise<SDKControlMcpReadResourceResponse>;

    // Dynamically set the MCP servers for this session.
    setMcpServers(servers: Record<string, McpServerConfig>): Promise<McpSetServersResult>;
    // Stream input messages to the query.
    streamInput(stream: AsyncIterable<SDKUserMessage>): Promise<void>;
    // Stop a running task.
    stopTask(taskId: string): Promise<void>;
    // Background in-flight foreground tasks (Bash commands and subagents).
    backgroundTasks(toolUseId?: string): Promise<boolean>;
    // Close the query and terminate the underlying process.
    close(): void;
}

export declare function query(_params: {
    prompt: string | AsyncIterable<SDKUserMessage>;
    options?: Options;
}): Query;

// Rename a session.
export declare function renameSession(_sessionId: string, _title: string, _options?: SessionMutationOptions): Promise<void>;

// Result of {@link resolveSettings}. [@alpha]
export declare type ResolvedSettings = {
    // Merged settings after applying all enabled sources in precedence order.
    effective: Settings;
    // For each top-level key in `effective`, which source supplied the value.
    provenance: Partial<Record<keyof Settings, ProvenanceEntry>>;
    // Per-source raw settings, low→high precedence.
    sources: Array<{
        source: ResolvedSettingSource;
        settings: Settings;
        path?: string;
        policyOrigin?: PolicySettingsOrigin;
    }>;
    // What became of this machine's own policy sources, in the host's own read (not a Claude Code session's).
    machinePolicy?: {
        // `false`, or the kinds of source that may be there and did not load: `'profile'` (a macOS MDM profile), `'registry'` (the Windows HKLM policy value; under WSL also the Windows folder's managed-settings files while WSL does not inherit them), `'file'` (the managed-settings file or a drop-in).
        didNotLoad: false | Array<'profile' | 'registry' | 'file'>;
    };
};

// Source that contributed an effective setting value. [@alpha]
export declare type ResolvedSettingSource = SettingSource | 'managed' | 'flag';

// Resolve the effective Claude Code settings for the given options using the same merge engine as the CLI, without spawning the Claude CLI. [@alpha]
export declare function resolveSettings(_opts?: ResolveSettingsOptions): Promise<ResolvedSettings>;

// Options for {@link resolveSettings}. [@alpha]
export declare type ResolveSettingsOptions = {
    // Directory to resolve project/local settings relative to.
    cwd?: string;
    // Which filesystem settings sources to load.
    settingSources?: SettingSource[];
    // Restrictive policy-tier settings — equivalent to `Options.managedSettings` on `query()`.
    managedSettings?: Settings;
    // Server-managed settings payload (the result of fetching `/api/claude_code/settings`).
    serverManagedSettings?: Settings;
};

// Result of a rewindFiles operation.
export declare type RewindFilesResult = {
    canRewind: boolean;
    error?: string;
    filesChanged?: string[];
    insertions?: number;
    deletions?: number;
    // Count of tracked files NOT restored or deleted because a symlink, hard link, or other non-regular file was detected at the tracked path, its parent directory no longer resolves to where it pointed when the checkpoint was taken, or its backup could not be safely read.
    skippedLinks?: number;
};

export declare type SandboxCredentialsConfig = NonNullable<z.infer<ReturnType<typeof SandboxCredentialsConfigSchema>>>;

declare const SandboxCredentialsConfigSchema: () => z.ZodOptional<z.ZodObject<{
    files: z.ZodOptional<z.ZodArray<z.ZodPreprocess<z.ZodObject<{
        path: z.ZodString;
        mode: z.ZodEnum<{
            deny: "deny";
            mask: "mask";
        }>;
        extract: z.ZodOptional<z.ZodString>;
        onExtractNoMatch: z.ZodOptional<z.ZodEnum<{
            deny: "deny";
            error: "error";
            warn: "warn";
        }>>;
        decode: z.ZodOptional<z.ZodEnum<{
            jwt: "jwt";
        }>>;
        maskClaims: z.ZodOptional<z.ZodArray<z.ZodString>>;
        maskDuplicates: z.ZodOptional<z.ZodBoolean>;
        injectHosts: z.ZodOptional<z.ZodArray<z.ZodString>>;
    }, z.core.$strip>, unknown>>>;
    envVars: z.ZodOptional<z.ZodArray<z.ZodPreprocess<z.ZodObject<{
        name: z.ZodString;
        mode: z.ZodEnum<{
            deny: "deny";
            mask: "mask";
        }>;
        extract: z.ZodOptional<z.ZodString>;
        onExtractNoMatch: z.ZodOptional<z.ZodEnum<{
            deny: "deny";
            error: "error";
            warn: "warn";
        }>>;
        decode: z.ZodOptional<z.ZodEnum<{
            jwt: "jwt";
        }>>;
        maskClaims: z.ZodOptional<z.ZodArray<z.ZodString>>;
        injectHosts: z.ZodOptional<z.ZodArray<z.ZodString>>;
    }, z.core.$strip>, unknown>>>;
    allowPlaintextInject: z.ZodOptional<z.ZodBoolean>;
    awsPairs: z.ZodOptional<z.ZodArray<z.ZodObject<{
        accessKeyIdVar: z.ZodString;
        secretAccessKeyVar: z.ZodString;
        sessionTokenVar: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>>>;
    sigv4: z.ZodOptional<z.ZodObject<{
        streaming: z.ZodOptional<z.ZodEnum<{
            deny: "deny";
            passthrough: "passthrough";
        }>>;
        presigned: z.ZodOptional<z.ZodEnum<{
            deny: "deny";
            passthrough: "passthrough";
        }>>;
        sigv4a: z.ZodOptional<z.ZodEnum<{
            deny: "deny";
            passthrough: "passthrough";
        }>>;
    }, z.core.$strip>>;
}, z.core.$strip>>;

export declare type SandboxFilesystemConfig = NonNullable<z.infer<ReturnType<typeof SandboxFilesystemConfigSchema>>>;

// Filesystem configuration schema for sandbox.
declare const SandboxFilesystemConfigSchema: () => z.ZodOptional<z.ZodObject<{
    allowWrite: z.ZodOptional<z.ZodArray<z.ZodString>>;
    denyWrite: z.ZodOptional<z.ZodArray<z.ZodString>>;
    denyRead: z.ZodOptional<z.ZodArray<z.ZodString>>;
    allowRead: z.ZodOptional<z.ZodArray<z.ZodString>>;
    allowManagedReadPathsOnly: z.ZodOptional<z.ZodBoolean>;
    disabled: z.ZodOptional<z.ZodBoolean>;
}, z.core.$strip>>;

export declare type SandboxIgnoreViolations = NonNullable<SandboxSettings['ignoreViolations']>;

export declare type SandboxNetworkConfig = NonNullable<z.infer<ReturnType<typeof SandboxNetworkConfigSchema>>>;

// Network configuration schema for sandbox.
declare const SandboxNetworkConfigSchema: () => z.ZodOptional<z.ZodObject<{
    allowedDomains: z.ZodOptional<z.ZodArray<z.ZodString>>;
    deniedDomains: z.ZodOptional<z.ZodArray<z.ZodString>>;
    strictAllowlist: z.ZodOptional<z.ZodBoolean>;
    allowManagedDomainsOnly: z.ZodOptional<z.ZodBoolean>;
    allowUnixSockets: z.ZodOptional<z.ZodArray<z.ZodString>>;
    allowAllUnixSockets: z.ZodOptional<z.ZodBoolean>;
    allowLocalBinding: z.ZodOptional<z.ZodBoolean>;
    allowMachLookup: z.ZodOptional<z.ZodArray<z.ZodString>>;
    httpProxyPort: z.ZodOptional<z.ZodNumber>;
    socksProxyPort: z.ZodOptional<z.ZodNumber>;
    tlsTerminate: z.ZodOptional<z.ZodObject<{
        caCertPath: z.ZodOptional<z.ZodString>;
        caKeyPath: z.ZodOptional<z.ZodString>;
    }, z.core.$strip>>;
}, z.core.$strip>>;

export declare type SandboxSettings = z.infer<ReturnType<typeof SandboxSettingsSchema>>;

// Sandbox settings schema.
declare const SandboxSettingsSchema: () => z.ZodObject<{
    enabled: z.ZodOptional<z.ZodBoolean>;
    failIfUnavailable: z.ZodOptional<z.ZodBoolean>;
    autoAllowBashIfSandboxed: z.ZodOptional<z.ZodBoolean>;
    allowUnsandboxedCommands: z.ZodOptional<z.ZodBoolean>;
    network: z.ZodOptional<z.ZodObject<{
        allowedDomains: z.ZodOptional<z.ZodArray<z.ZodString>>;
        deniedDomains: z.ZodOptional<z.ZodArray<z.ZodString>>;
        strictAllowlist: z.ZodOptional<z.ZodBoolean>;
        allowManagedDomainsOnly: z.ZodOptional<z.ZodBoolean>;
        allowUnixSockets: z.ZodOptional<z.ZodArray<z.ZodString>>;
        allowAllUnixSockets: z.ZodOptional<z.ZodBoolean>;
        allowLocalBinding: z.ZodOptional<z.ZodBoolean>;
        allowMachLookup: z.ZodOptional<z.ZodArray<z.ZodString>>;
        httpProxyPort: z.ZodOptional<z.ZodNumber>;
        socksProxyPort: z.ZodOptional<z.ZodNumber>;
        tlsTerminate: z.ZodOptional<z.ZodObject<{
            caCertPath: z.ZodOptional<z.ZodString>;
            caKeyPath: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>>;
    }, z.core.$strip>>;
    filesystem: z.ZodOptional<z.ZodObject<{
        allowWrite: z.ZodOptional<z.ZodArray<z.ZodString>>;
        denyWrite: z.ZodOptional<z.ZodArray<z.ZodString>>;
        denyRead: z.ZodOptional<z.ZodArray<z.ZodString>>;
        allowRead: z.ZodOptional<z.ZodArray<z.ZodString>>;
        allowManagedReadPathsOnly: z.ZodOptional<z.ZodBoolean>;
        disabled: z.ZodOptional<z.ZodBoolean>;
    }, z.core.$strip>>;
    credentials: z.ZodOptional<z.ZodObject<{
        files: z.ZodOptional<z.ZodArray<z.ZodPreprocess<z.ZodObject<{
            path: z.ZodString;
            mode: z.ZodEnum<{
                deny: "deny";
                mask: "mask";
            }>;
            extract: z.ZodOptional<z.ZodString>;
            onExtractNoMatch: z.ZodOptional<z.ZodEnum<{
                deny: "deny";
                error: "error";
                warn: "warn";
            }>>;
            decode: z.ZodOptional<z.ZodEnum<{
                jwt: "jwt";
            }>>;
            maskClaims: z.ZodOptional<z.ZodArray<z.ZodString>>;
            maskDuplicates: z.ZodOptional<z.ZodBoolean>;
            injectHosts: z.ZodOptional<z.ZodArray<z.ZodString>>;
        }, z.core.$strip>, unknown>>>;
        envVars: z.ZodOptional<z.ZodArray<z.ZodPreprocess<z.ZodObject<{
            name: z.ZodString;
            mode: z.ZodEnum<{
                deny: "deny";
                mask: "mask";
            }>;
            extract: z.ZodOptional<z.ZodString>;
            onExtractNoMatch: z.ZodOptional<z.ZodEnum<{
                deny: "deny";
                error: "error";
                warn: "warn";
            }>>;
            decode: z.ZodOptional<z.ZodEnum<{
                jwt: "jwt";
            }>>;
            maskClaims: z.ZodOptional<z.ZodArray<z.ZodString>>;
            injectHosts: z.ZodOptional<z.ZodArray<z.ZodString>>;
        }, z.core.$strip>, unknown>>>;
        allowPlaintextInject: z.ZodOptional<z.ZodBoolean>;
        awsPairs: z.ZodOptional<z.ZodArray<z.ZodObject<{
            accessKeyIdVar: z.ZodString;
            secretAccessKeyVar: z.ZodString;
            sessionTokenVar: z.ZodOptional<z.ZodString>;
        }, z.core.$strip>>>;
        sigv4: z.ZodOptional<z.ZodObject<{
            streaming: z.ZodOptional<z.ZodEnum<{
                deny: "deny";
                passthrough: "passthrough";
            }>>;
            presigned: z.ZodOptional<z.ZodEnum<{
                deny: "deny";
                passthrough: "passthrough";
            }>>;
            sigv4a: z.ZodOptional<z.ZodEnum<{
                deny: "deny";
                passthrough: "passthrough";
            }>>;
        }, z.core.$strip>>;
    }, z.core.$strip>>;
    ignoreViolations: z.ZodOptional<z.ZodRecord<z.ZodString, z.ZodArray<z.ZodString>>>;
    enableWeakerNestedSandbox: z.ZodOptional<z.ZodBoolean>;
    allowNestedUserNamespaces: z.ZodOptional<z.ZodBoolean>;
    enableWeakerNetworkIsolation: z.ZodOptional<z.ZodBoolean>;
    allowAppleEvents: z.ZodOptional<z.ZodBoolean>;
    excludedCommands: z.ZodOptional<z.ZodArray<z.ZodString>>;
    ripgrep: z.ZodOptional<z.ZodObject<{
        command: z.ZodString;
        args: z.ZodOptional<z.ZodArray<z.ZodString>>;
    }, z.core.$strip>>;
    bwrapPath: z.ZodCatch<z.ZodOptional<z.ZodPreprocess<z.ZodString, unknown>>>;
    socatPath: z.ZodCatch<z.ZodOptional<z.ZodPreprocess<z.ZodString, unknown>>>;
}, z.core.$loose>;

// Emitted when the user's /goal Stop hook reports met (clears) or not-yet-met (bumps iterations + last_reason).
export declare type SDKActiveGoalMessage = {
    type: 'active_goal';
    value: {
        condition: string;
        iterations: number;
        set_at: number;
        tokens_at_start: number;
        last_reason?: string;
    } | null;
    uuid: UUID;
    session_id: string;
};

// Emitted when an API request fails with a retryable error and will be retried after a delay.
export declare type SDKAPIRetryMessage = {
    type: 'system';
    subtype: 'api_retry';
    attempt: number;
    max_retries: number;
    retry_delay_ms: number;
    error_status: number | null;
    error: SDKAssistantMessageError;
    // Present only when the API sent no response headers within the first-byte window (CLAUDE_STREAM_FIRST_BYTE_TIMEOUT_MS): waited_ms is how long the failed attempt waited for headers, retry_wait_ms how long the retry will wait for them.
    no_response?: {
        waited_ms: number;
        retry_wait_ms: number;
    };
    uuid: UUID;
    session_id: string;
};

// An assistant message.
export declare type SDKAssistantMessage = {
    type: 'assistant';
    // Shaped like an Anthropic Messages API Message object (role "assistant"): id, model, content blocks (text, thinking, tool_use, ...), stop_reason and usage.
    message: BetaMessage;
    parent_tool_use_id: string | null;
    error?: SDKAssistantMessageError;
    uuid: UUID;

    session_id: string;
    request_id?: string;

    // Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on an assistant message each time that send changes — the turn's FIRST top-level assistant message (which may carry only a thinking block, or be a synthetic API-error message), and then, for a turn started by a synthetic (meta) prompt, the first assistant message after each queued user message folded in mid-turn (the fold takes the echo over); with --include-partial-messages the turn's first non-ping stream event is stamped too, independently (see SDKPartialAssistantMessage), so the same uuid may appear on both — either binds the reply to the send it answers without waiting for the result; the server keeps the first stamp it sees per uuid.
    user_message_uuid?: string;
    // Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host merged into this one turn (several messages sent close together run as one turn whose user_message_uuid is the LAST member's), then any user message folded into the turn before this frame — so a consumer that sent any of them can bind this reply to its own send by finding its uuid anywhere in the list.
    user_message_uuids?: string[];
    // Why this frame's turn continues one that a worker restart interrupted.
    resume_reason?: string;
    // This turn continued the preceding truncated assistant turn inside its trailing signed thinking block (max-output-tokens recovery).
    resumed_from_incomplete_thinking?: true;
    // Uuids of earlier messages that this message replaces: after a refusal fallback, or after a response whose tool call could not be parsed.
    supersedes?: UUID[];
    // True when this assistant message was truncated by an interrupt/abort before the stream completed: stop_reason was never received and the content may end mid-word.
    aborted?: true;
    // ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed.
    agent_id?: string;
    // Subagent type that produced this message.
    subagent_type?: string;
    // Description of the subagent task that produced this message.
    task_description?: string;

    // ISO timestamp of when this content block finished on the originating process.
    timestamp?: string;

    // Structured twin of the /context report, carried on the synthetic assistant message that delivers the markdown table.
    context_usage?: SDKContextUsage;
    // Structured twin of the /usage report, carried on the synthetic assistant message that delivers its text: the session totals, the plan's usage rows and extra-usage spend, for remote clients that render a card from data.
    usage_report?: SDKUsageReport;

};

export declare type SDKAssistantMessageError = 'authentication_failed' | 'oauth_org_not_allowed' | 'account_on_hold' | 'verification_required' | 'billing_error' | 'rate_limit' | 'overloaded' | 'invalid_request' | 'model_not_found' | 'server_error' | 'unknown' | 'max_output_tokens' | 'cloud_credential_error';

export declare type SDKAuthStatusMessage = {
    type: 'auth_status';
    isAuthenticating: boolean;
    output: string[];
    error?: string;
    uuid: UUID;
    session_id: string;
};

// The full set of live background tasks, emitted whenever membership changes (start, completion, kill, a foreground agent being backgrounded) or an entry's `ambient` flag or `parent_task_id` changes.
export declare type SDKBackgroundTasksChangedMessage = {
    type: 'system';
    subtype: 'background_tasks_changed';
    // Every live background task after the change.
    tasks: {
        task_id: string;
        // Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
        run_id?: string;
        task_type: string;
        // Agent type of a local_agent task, such as "general-purpose" or a custom agent's name; "main-session" for a backgrounded main session.
        subagent_type?: string;
        description: string;
        // task_id of the subagent (local_agent) task whose agent launched this subagent, shell, monitor or workflow task.
        parent_task_id?: string;
        // True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude them from activity indicators.
        ambient?: boolean;

    }[];
    uuid: UUID;
    session_id: string;
};

export declare type SdkBeta = 'context-1m-2025-08-07';

// Fire-and-forget push of the full slash-command list after a mid-session change (e.g.
export declare type SDKCommandsChangedMessage = {
    type: 'system';
    subtype: 'commands_changed';
    commands: SlashCommand[];
    uuid: UUID;
    session_id: string;
};

export declare type SDKCompactBoundaryMessage = {
    type: 'system';
    subtype: 'compact_boundary';
    compact_metadata: {
        trigger: 'manual' | 'auto';
        pre_tokens: number;
        post_tokens?: number;

        duration_ms?: number;

        // Relink info for messagesToKeep.
        preserved_segment?: {
            head_uuid: UUID;
            anchor_uuid: UUID;
            tail_uuid: UUID;
        };
        // Ordered messagesToKeep UUIDs.
        preserved_messages?: {
            anchor_uuid: UUID;
            uuids: UUID[];

        };
    };

    uuid: UUID;
    session_id: string;

};

// Structured twin of the /context report — the data a client needs to render the context-usage card without parsing the markdown table.
export declare type SDKContextUsage = {
    // Main-loop model the usage was computed for.
    model: string;
    // Estimated tokens in use, unclamped — may exceed raw_max_tokens when over limit.
    total_tokens: number;
    // The window usage is measured against: the resolved autocompact window — the model's believed limit, or a smaller compaction-policy window (a configured value, or e.g.
    raw_max_tokens: number;
    // Rounded total_tokens / raw_max_tokens, 0-100+.
    percentage: number;
    // Present when total_tokens exceeds raw_max_tokens.
    over_limit?: {
        tokens_over: number;
        kind: 'hard_limit' | 'compaction_window';
    };
    categories: SDKContextUsageCategory[];
    mcp_tools: {
        // Wire name, e.g.
        name: string;
        server_name: string;
        tokens: number;
    }[];
    memory_files: {
        path: string;
        // Display label of the memory-file source, e.g.
        type: string;
        tokens: number;
    }[];
    agents: {
        agent_type: string;
        // Raw source identifier, e.g.
        source: string;
        tokens: number;
    }[];
    // Omitted when no skills contribute tokens.
    skills?: {
        name: string;
        // Raw source identifier, e.g.
        source: string;
        plugin_name?: string;
        tokens: number;
    }[];
};

// One row of the /context usage-by-category breakdown.
export declare type SDKContextUsageCategory = {
    // Display name of the row as the CLI renders it, e.g.
    name: string;
    tokens: number;
    // What the row is: 'used' content occupies the window; 'free' is the remaining window; 'buffer' is the compaction reserve (autocompact or manual); 'deferred' rows are out-of-window tool schemas — listed for awareness, excluded from usage math.
    kind: 'used' | 'free' | 'buffer' | 'deferred';
};

// Merges the provided settings into the flag settings layer, updating the active configuration.
declare type SDKControlApplyFlagSettingsRequest = {
    subtype: 'apply_flag_settings';
    settings: Record<string, unknown>;
};

// Backgrounds in-flight foreground tasks (Bash commands and subagents).
declare type SDKControlBackgroundTasksRequest = {
    subtype: 'background_tasks';
    // When set, backgrounds only the task whose originating tool_use block has this id.
    tool_use_id?: string;
};

// Drops a pending async user message from the command queue by uuid.
declare type SDKControlCancelAsyncMessageRequest = {
    subtype: 'cancel_async_message';
    message_uuid: string;
};

// Tells the other side that the sender no longer needs the answer to one of its own in-flight control_requests (for example a pending can_use_tool prompt after the turn was interrupted, or one that another client already answered).
declare type SDKControlCancelRequest = {
    type: 'control_cancel_request';
    // The request_id of the control_request being withdrawn.
    request_id: string;
};

// Requests the SDK consumer to handle an MCP elicitation (user input request).
declare type SDKControlElicitationRequest = {
    subtype: 'elicitation';
    mcp_server_name: string;
    message: string;
    mode?: 'form' | 'url';
    url?: string;
    elicitation_id?: string;
    requested_schema?: Record<string, unknown>;
    // requested_schema.properties' top-level keys in the order the MCP server listed them; absent for URL mode and when the schema has no properties object.
    property_order?: string[];
    // Permission-display title from the MCP server's _meta['anthropic/permissionDisplay'].
    title?: string;
    // Short tool/server label from _meta['anthropic/permissionDisplay'].displayName.
    display_name?: string;
    // Permission-display subtitle from _meta['anthropic/permissionDisplay'].description.
    description?: string;
};

// Requests at-mention file autocomplete suggestions for a partial path prefix.
declare type SDKControlFileSuggestionsRequest = {
    subtype: 'file_suggestions';
    query: string;
};

// Requests the responder's CLI binary version.
declare type SDKControlGetBinaryVersionRequest = {
    subtype: 'get_binary_version';
};

// Requests a breakdown of current context window usage by category.
declare type SDKControlGetContextUsageRequest = {
    subtype: 'get_context_usage';
    // 'full' counts each category with the token-count API; 'summary' answers from the last response's usage and local estimates without the per-category token-count calls.
    detail?: 'summary' | 'full';
};

// Breakdown of current context window usage by category (system prompt, tools, messages, etc.).
export declare type SDKControlGetContextUsageResponse = {
    categories: {
        // Which row this is.
        id?: 'system_prompt' | 'system_tools' | 'mcp_tools' | 'mcp_instructions' | 'mcp_tools_deferred' | 'system_tools_deferred' | 'custom_agents' | 'memory_files' | 'skills' | 'messages' | 'autocompact_buffer' | 'compact_buffer' | 'free_space';
        name: string;
        tokens: number;
        color: string;
        isDeferred?: boolean;
        // What the row is, the same classification the /context result's context_usage rows carry: 'used' content occupies the window; 'free' is the remaining window; 'buffer' is the compaction reserve; 'deferred' rows are out-of-window tool schemas.
        kind: 'used' | 'free' | 'buffer' | 'deferred';
    }[];
    totalTokens: number;
    maxTokens: number;
    rawMaxTokens: number;
    percentage: number;
    gridRows: {
        color: string;
        isFilled: boolean;
        categoryName: string;
        tokens: number;
        percentage: number;
        squareFullness: number;
    }[][];
    model: string;
    memoryFiles: {
        path: string;
        type: string;
        tokens: number;
    }[];
    mcpTools: {
        name: string;
        serverName: string;
        tokens: number;
        isLoaded?: boolean;
    }[];
    // The 'mcp_instructions' row by server.
    mcpInstructions?: {
        // Server name, as mcp_status reports it and mcp_toggle takes it (not normalized, unlike mcpTools[].serverName).
        serverName: string;
        tokens: number;
    }[];
    deferredBuiltinTools?: {
        name: string;
        tokens: number;
        isLoaded: boolean;
    }[];
    systemTools?: {
        name: string;
        tokens: number;
    }[];
    systemPromptSections?: {
        name: string;
        tokens: number;
    }[];
    agents: {
        agentType: string;
        source: string;
        // Id of the providing plugin (its name and marketplace), when there is one.
        pluginId?: string;
        // Absolute path of the agent's file on the machine the CLI runs on, which over SSH or WSL is not the host's.
        path?: string;
        tokens: number;
    }[];
    slashCommands?: {
        totalCommands: number;
        includedCommands: number;
        tokens: number;
    };
    skills?: {
        totalSkills: number;
        includedSkills: number;
        tokens: number;
        skillFrontmatter: {
            name: string;
            source: string;
            // Name of the providing plugin, when there is one.
            pluginName?: string;
            // Id of the providing plugin (its name and marketplace), when there is one.
            pluginId?: string;
            // Absolute path of the skill's file on the machine the CLI runs on, which over SSH or WSL is not the host's.
            path?: string;
            tokens: number;
        }[];
    };
    autoCompactThreshold?: number;
    isAutoCompactEnabled: boolean;
    messageBreakdown?: {
        toolCallTokens: number;
        toolResultTokens: number;
        attachmentTokens: number;
        assistantMessageTokens: number;
        userMessageTokens: number;
        redirectedContextTokens: number;
        unattributedTokens: number;
        toolCallsByType: {
            name: string;
            callTokens: number;
            resultTokens: number;
        }[];
        attachmentsByType: {
            name: string;
            tokens: number;
        }[];
    };
    apiUsage: {
        input_tokens: number;
        output_tokens: number;
        cache_creation_input_tokens: number;
        cache_read_input_tokens: number;
    } | null;

};

// Returns the hooks listing the CLI's read-only /hooks menu renders: settings-file, session, and plugin hooks grouped by event and matcher, with display-ready strings (control characters revealed) and the policy and safe-mode state the menu banners on.
declare type SDKControlGetHooksListingRequest = {
    subtype: 'get_hooks_listing';
};

// The hooks listing the CLI's /hooks menu renders, with display-ready strings.
declare type SDKControlGetHooksListingResponse = {
    // Events that have at least one listed hook, in the /hooks menu’s lifecycle order.
    events: {
        // Hook event name.
        name: string;
        // One-line summary, as the /hooks event list shows.
        summary: string;
        // Whether hooks on this event can carry a matcher.
        supportsMatcher: boolean;
        // Number of listed hooks.
        hookCount: number;
    }[];
    // One row per listed hook: events in lifecycle order, matchers in the menu’s priority order.
    hooks: {
        event: string;
        // Matcher with control characters revealed; '' when the entry has none.
        matcher: string;
        // Raw source name (userSettings, sessionHook, pluginHook, …).
        source: string;
        // User-facing source description.
        sourceLabel: string;
        pluginName?: string;
        // Hook type (command, prompt, agent, http, mcp_tool, …).
        type: string;
        // List-row label: statusMessage when set, else the identity text; one line, control characters revealed.
        displayText: string;
        // Identity text — the literal command/prompt/URL that runs, control characters revealed.
        commandText: string;
        // Label for commandText (Command, Prompt, URL, …).
        contentLabel: string;
        // If-condition, revealed, when set.
        condition?: string;
        // Timeout in seconds.
        timeout?: number;
        statusMessage?: string;
        runsOnce?: boolean;
        runsInBackground?: boolean;
        // True when the session's mode or policy keeps this hook from running (the policy block and safeMode/bareMode say why); absent on rows that run.
        disabled?: true;
        // The entry as stored (raw matcher, '' when none, and raw hook object; no display escaping) for a host's edit form and as the target it names to `claude edit-hook`.
        editable?: {
            matcher: string;
            config: Record<string, unknown>;
            headersRedacted?: true;
        };
    }[];
    // Every hook event in lifecycle order (name, the /hooks summary, whether its hooks take a matcher), for an add-hook form.
    eventCatalog: {
        name: string;
        summary: string;
        supportsMatcher: boolean;
    }[];
    policy: {
        // Managed disableAllHooks — nothing runs at all.
        disabledByPolicy: boolean;
        // Managed allowManagedHooksOnly — non-managed hooks are blocked and managed hooks are intentionally not listed.
        managedOnly: boolean;
        // Managed strictPluginOnlyCustomization locks the hooks surface.
        pluginOnly: boolean;
        // Effective disableAllHooks, whatever source set it.
        allDisabled: boolean;
        // Hooks configured in managed settings (they run even under a non-managed disableAllHooks).
        policyHookCount: number;
        // Present when a managed settings source exists but could not be read: what the organization configured is unknown, so edit_hook refuses every edit (fail-closed) and a host locks its editing controls, as it does under the three locks above.
        policyUnreadable?: true;
    };
    // Present only when the session runs under --safe-mode.
    safeMode?: {
        managedHooksStillApply: boolean;
        // How to leave safe mode, per its activation source.
        exitHint: string;
    };
    // Present only under --bare / CLAUDE_CODE_SIMPLE with the hooks surface gated off: settings-file, flag, policy, and plugin hooks never fire there; session hooks still run.
    bareMode?: {
        // How to leave bare mode, per its activation source.
        exitHint: string;
    };
    // Settings files skipped by the merge — their hooks are neither listed nor running.
    errors?: coreTypes.SDKSettingsParseError[];
};

// Requests the formatted session cost summary (the same text /usage prints in non-interactive mode).
declare type SDKControlGetSessionCostRequest = {
    subtype: 'get_session_cost';
};

// Returns the effective merged settings and the raw per-source settings.
declare type SDKControlGetSettingsRequest = {
    subtype: 'get_settings';
};

// Reads the end of one background shell or Monitor task's output: at most the last 8 KiB the command wrote, the same tail the terminal's /tasks detail view reads.
declare type SDKControlGetTaskOutputRequest = {
    subtype: 'get_task_output';
    // A shell or Monitor task of this session, running or ended: the task_id from task_started or background_tasks_changed.
    task_id: string;
};

// Success payload answering get_task_output.
declare type SDKControlGetTaskOutputResponse = {
    // The end of the output, decoded as UTF-8; empty when the command has written nothing yet.
    output: string;
    // The size of the whole output in bytes.
    total_bytes: number;
    // Whether output holds only the end of a longer output (the last 8 KiB).
    truncated: boolean;
};

// Requests the structured /usage data: session cost/usage totals plus claude.ai plan rate-limit utilization when available.
declare type SDKControlGetUsageRequest = {
    subtype: 'get_usage';
    // Skip the scan of local transcripts that produces the response's behaviors section (it is null in the answer).
    skip_behaviors?: boolean;
};

// Structured /usage data: session cost/usage totals plus claude.ai plan rate-limit utilization.
export declare type SDKControlGetUsageResponse = {
    // Cost and usage accumulated by the current session.
    session: {
        total_cost_usd: number;
        total_api_duration_ms: number;
        total_duration_ms: number;
        total_lines_added: number;
        total_lines_removed: number;
        model_usage: Record<string, coreTypes.ModelUsage>;
    };
    // Claude.ai subscription type ('pro', 'max', 'team', 'enterprise') or null for API key / 3P provider sessions.
    subscription_type: string | null;
    // False when plan rate limits do not apply (API key, Bedrock, Vertex, or missing profile scope) — rate_limits will be null.
    rate_limits_available: boolean;
    // Plan rate-limit utilization windows from the claude.ai usage endpoint, or null when unavailable.
    rate_limits: {
        five_hour?: {
            // Percentage of the window used, 0-100.
            utilization: number | null;
            // ISO 8601 timestamp when the window resets.
            resets_at: string | null;
        } | null;
        seven_day?: {
            // Percentage of the window used, 0-100.
            utilization: number | null;
            // ISO 8601 timestamp when the window resets.
            resets_at: string | null;
        } | null;
        seven_day_oauth_apps?: {
            // Percentage of the window used, 0-100.
            utilization: number | null;
            // ISO 8601 timestamp when the window resets.
            resets_at: string | null;
        } | null;
        seven_day_opus?: {
            // Percentage of the window used, 0-100.
            utilization: number | null;
            // ISO 8601 timestamp when the window resets.
            resets_at: string | null;
        } | null;
        seven_day_sonnet?: {
            // Percentage of the window used, 0-100.
            utilization: number | null;
            // ISO 8601 timestamp when the window resets.
            resets_at: string | null;
        } | null;
        // Per-model weekly windows from the server limits[] array, filtered by the overage-included-models allowlist.
        model_scoped?: {
            // Server-supplied label for the model bucket (e.g.
            display_name: string;
            utilization: number | null;
            resets_at: string | null;
        }[];
        extra_usage?: {
            is_enabled: boolean;
            monthly_limit: number | null;
            used_credits: number | null;
            utilization: number | null;
            currency?: string | null;
        } | null;
    } | null;
    // What's contributing to limits usage, from a scan of local transcripts on this machine (the same data the /usage dialog renders): behavioral characteristics plus per-skill/agent/plugin/MCP-server attribution.
    behaviors: {
        // Last 24 hours.
        day: {
            // API requests found in local transcripts for this window.
            request_count: number;
            // Distinct sessions observed in this window.
            session_count: number;
            // Behavioral characteristics of local usage.
            behaviors: {
                key: 'cache_miss' | 'long_context' | 'subagent_heavy' | 'high_parallel' | 'cron' | 'max_effort' | 'workflow' | 'always_on_context' | 'large_tool_results';
                // Share of the weighted local usage attributed to this behavior, 0-100.
                pct: number;
                // Requests in this window exhibiting the behavior.
                count: number;
            }[];
            // One recommended change per notable behavior or source, largest share first, worded for display.
            recommendations?: {
                // The behavior key, or 'agent', 'skill', 'plugin' or 'mcp_server' for a named source.
                key: string;
                // Share of the weighted local usage this touches, 0-100.
                pct: number;
                // What to do, verb first.
                title: string;
                // A few words: what the scan saw, or the command to run.
                detail?: string;
            }[];
            // What one "fix" action does.
            fix?: {
                // The recommendations it covers.
                keys: string[];
                // User messages to send for the person, one each and in order, at priority 'later' so a running turn cannot reorder them: any /effort commands first, then one prompt for the rest, which goes with client_composed set.
                messages: string[];
            };
            agents: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
            skills: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
            plugins: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
            mcp_servers: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
        };
        // Last 7 days.
        week: {
            // API requests found in local transcripts for this window.
            request_count: number;
            // Distinct sessions observed in this window.
            session_count: number;
            // Behavioral characteristics of local usage.
            behaviors: {
                key: 'cache_miss' | 'long_context' | 'subagent_heavy' | 'high_parallel' | 'cron' | 'max_effort' | 'workflow' | 'always_on_context' | 'large_tool_results';
                // Share of the weighted local usage attributed to this behavior, 0-100.
                pct: number;
                // Requests in this window exhibiting the behavior.
                count: number;
            }[];
            // One recommended change per notable behavior or source, largest share first, worded for display.
            recommendations?: {
                // The behavior key, or 'agent', 'skill', 'plugin' or 'mcp_server' for a named source.
                key: string;
                // Share of the weighted local usage this touches, 0-100.
                pct: number;
                // What to do, verb first.
                title: string;
                // A few words: what the scan saw, or the command to run.
                detail?: string;
            }[];
            // What one "fix" action does.
            fix?: {
                // The recommendations it covers.
                keys: string[];
                // User messages to send for the person, one each and in order, at priority 'later' so a running turn cannot reorder them: any /effort commands first, then one prompt for the rest, which goes with client_composed set.
                messages: string[];
            };
            agents: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
            skills: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
            plugins: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
            mcp_servers: {
                name: string;
                // Share of the weighted local usage attributed to this item, 0-100.
                pct: number;
            }[];
        };
    } | null;
};

// Initializes the SDK session with hooks, MCP servers, and agent configuration.
declare type SDKControlInitializeRequest = {
    subtype: 'initialize';
    hooks?: Partial<Record<coreTypes.HookEvent, SDKHookCallbackMatcher[]>>;
    sdkMcpServers?: string[];
    // Settings for the SDK-hosted MCP servers named in sdkMcpServers, keyed by server name.
    sdkMcpServerConfigs?: Record<string, {
        // Per-server tool-call timeout in milliseconds.
        timeout?: number;

        // Per-tool permission policy the host declares for this server's tools (not the tool definitions passed to createSdkMcpServer).
        tools?: coreTypes.McpServerToolPolicy[];
    }>;
    // Optional, keyed by sdk server name (each key should also appear in sdkMcpServers; other keys are ignored).
    sdkMcpServerManifests?: Record<string, {
        // The server's verbatim JSON-RPC `initialize` result object (protocolVersion, capabilities, serverInfo, instructions, ...), exactly as the in-process server produced it when the host initialized it with no client capabilities — not re-serialized or schema-parsed by the host.
        initializeResult: Record<string, unknown>;
        // The server's verbatim JSON-RPC `tools/list` result object.
        toolsListResult?: Record<string, unknown>;
    }>;
    // The host's statement that it composed this request's sdkMcpServerManifests for this very send, from its servers' current state.
    sdkMcpServerManifestsOrigin?: 'host_per_send';
    jsonSchema?: Record<string, unknown>;
    systemPrompt?: string[];
    appendSystemPrompt?: string;
    // Record the conversation's system prompt once and reuse it verbatim on every later request and resume.
    systemPromptSnapshot?: boolean;
    // Custom workflow body for the plan-mode system reminder.
    planModeInstructions?: string;

    // Map of tool-name aliases applied before name resolution.
    toolAliases?: Record<string, string>;

    // When true, omit per-user dynamic sections (working directory, auto-memory path) from the cached system prompt and re-inject them as the first user message.
    excludeDynamicSections?: boolean;
    agents?: Record<string, coreTypes.AgentDefinition>;
    // Custom session title.
    title?: string;
    // When provided, only skills whose names match an entry are loaded into the main session system prompt, matching the exact canonical name (e.g.
    skills?: string[];

    promptSuggestions?: boolean;
    agentProgressSummaries?: boolean;
    forwardSubagentText?: boolean;
    // Dialog kinds (request_user_dialog `dialog_kind` values) this consumer's onUserDialog can actually render.
    supportedDialogKinds?: string[];

    // Declares that this consumer renders a per-task stop control wired to the `stop_task` control request, so the user can stop an individual background task.
    perTaskStopAffordance?: boolean;

    // Plugins to load for the session, in the same shape as the SDK `plugins` option: the stdin form of one --plugin-dir flag per entry (--plugin-dir-no-mcp when skipMcpDiscovery is set), so the launch command line does not grow with the plugin count.
    plugins?: coreTypes.SdkPluginConfig[];

};

// Response from session initialization with available commands, models, and account info.
export declare type SDKControlInitializeResponse = {
    commands: coreTypes.SlashCommand[];
    agents: coreTypes.AgentInfo[];
    output_style: string;
    available_output_styles: string[];

    models: coreTypes.ModelInfo[];

    // Information about the logged in user's account.
    account: coreTypes.AccountInfo;

    // Whether the `hooks` this initialize carried were registered: true on a session's first initialize, and on a repeated initialize from the process that owns the CLI's stdin (its set replaces the one registered earlier); false when a repeated initialize's hooks were ignored (a client joining a remote session another client configured).
    hooks_applied?: boolean;
    // True on every initialize reply from a CLI that honors `onFailure` on hook matchers.
    hook_on_failure_supported?: boolean;

    // Whether every plugin this initialize listed is loaded: true when each one is among the plugins the process loaded at launch (from `plugins` under --await-initialize, or from --plugin-dir), so a re-sent initialize naming the launch set also reads true; false otherwise.
    plugins_applied?: boolean;

    // What became of each entry of this initialize's sdkMcpServerManifests, keyed by sdk server name, for the entries whose name is in sdkMcpServers.
    sdk_mcp_manifests_parked?: Record<string, 'parked' | 'already_connected' | 'protocol_version_mismatch' | 'malformed' | 'not_honoured'>;

    // Claude Code version of the process that runs this session's turns (the same value its system/init carries).
    claude_code_version?: string;
    fast_mode_state?: coreTypes.FastModeState;
    fast_mode_disabled_reason?: coreTypes.FastModeDisabledReason;

};

// Interrupts the currently running conversation turn.
declare type SDKControlInterruptRequest = {
    subtype: 'interrupt';

    // When true, the interrupt also cancels every uuid-stamped main-thread command still in the queue or already dequeued for the imminent turn but not yet reachable by the abort (the first-command prewait window) — the same set the response would otherwise list under `still_queued`.
    cancel_queued?: boolean;

};

// Result of an interrupt operation.
export declare type SDKControlInterruptResponse = {
    // Uuids of async user messages that survive this interrupt: commands still in the queue, plus any batch already dequeued for the imminent turn but not yet reachable by the abort, plus any user message this process has read but not yet queued, from its own input stream (held, for instance, while a model change is still being checked or behind start-up work; a message a Remote Control client sent that is still being ingested is not listed).
    still_queued: string[];

    // Present only when the request set cancel_queued:true, with one exception that the last sentence names — uuids of main-thread commands cancelled by this interrupt: every survivor that would otherwise have appeared under `still_queued`, including any uuid that was mid-fold at the interrupt instant (this request also aborts, so the fold never delivers it).
    cancelled?: string[];
};

// Requests the worker's selectable model catalog.
declare type SDKControlListModelsRequest = {
    subtype: 'list_models';
};

// Requests the session's live permission rules and workspace directories — the same data /permissions lists in the terminal: rules from settings files plus session-only approvals, slash-command grants, and --allowedTools flag rules, each with its source.
declare type SDKControlListPermissionRulesRequest = {
    subtype: 'list_permission_rules';
};

// Success payload of list_permission_rules.
export declare type SDKControlListPermissionRulesResponse = {
    // The session's live permission rules state, as list_permission_rules reports it.
    state: SDKControlPermissionRulesState;
};

// Invokes an MCP tool via the subprocess MCP client without a model turn.
declare type SDKControlMcpCallRequest = {
    subtype: 'mcp_call';
    // Fully-qualified MCP tool name, e.g.
    tool: string;
    // Tool arguments.
    arguments?: Record<string, unknown>;
    // RFC3339 deadline, REQUIRED when output_files are declared (a stale buffered drain must not overwrite rows written since).
    expires_at?: string;
    // Tool-execution timeout (staging and collection have their own transport timeouts); with check_permissions it also bounds hooks and the wait for an answer.
    timeout_ms?: number;

    // The caller hands the result to a program (a host that renders it, a script), not to a model, and bounds it itself.
    result_not_for_model?: boolean;
    // The call is someone else's (a page a host renders), so it is decided as the model's own call to this tool is, except that a slash command's allowed-tools and connector card answers never allow it, even during a turn: permission rules under every spelling, the permission mode, requiresUserInteraction, managed settings, and PreToolUse and PostToolUse hooks, which see the arguments and the result.
    check_permissions?: boolean;
    // Declaring input_files or output_files makes this a STAGED call: rows are fetched from the synced-file lane into a private per-request temp dir the WORKER chooses (random, per-UID — the caller never sees or computes paths; it references files via tokens in arguments), the tool runs, and declared outputs are written back as lane rows (durable-at-ack PUT).
    input_files?: {
        // Handle referenced from arguments as "{{in:NAME}}".
        name: string;
        // Synced-file lane row to stage, e.g.
        lane_path: string;
    }[];
    output_files?: {
        // Handle referenced from arguments as "{{out:NAME}}" — the tool is told (via the substituted path) where to write; the worker collects from exactly that path.
        name: string;
        // Synced-file lane row to write, e.g.
        lane_path: string;
        // Opaque lane etag the output row must still carry for the write to land (CAS).
        if_match?: string;
    }[];
};

// Carries one MCP JSON-RPC message for an SDK-hosted MCP server (one named in initialize.sdkMcpServers or added later with mcp_set_servers).
declare type SDKControlMcpMessageRequest = {
    subtype: 'mcp_message';
    server_name: string;
    // A JSON-RPC 2.0 message as defined by the Model Context Protocol (request, notification, or response object).
    message: JSONRPCMessage;
};

// Reads one MCP Apps (SEP-1865) UI resource — a `ui://` URI, typically the `_meta.ui.resourceUri` a tool declares — from a connected MCP server the CLI itself dialed, with `resources/read`, for a host that renders it.
declare type SDKControlMcpReadResourceRequest = {
    subtype: 'mcp_read_resource';
    // Server name, as mcp_status reports it (not normalized).
    serverName: string;
    // The resource to read.
    uri: string;
};

// The server's resources/read result for an mcp_read_resource request.
export declare type SDKControlMcpReadResourceResponse = {
    contents: {
        uri: string;
        mimeType?: string;
        text?: string;
        // Base64, for a binary item.
        blob?: string;
        // The content item's own `_meta` as the server sent it (MCP Apps puts the resource's `ui.csp` and `ui.permissions` here), minus keys under the CLI-reserved `com.anthropic/` prefix and `io.modelcontextprotocol/files` download targets: from a claude.ai connector, one that holds the download service's one-time token (every such target while connector downloads or their removal are on); from a server on a cloud session's MCP route, every such target.
        _meta?: Record<string, unknown>;
    }[];
};

// Reconnects a disconnected or failed MCP server.
declare type SDKControlMcpReconnectRequest = {
    subtype: 'mcp_reconnect';
    serverName: string;
};

// Replaces the set of dynamically managed MCP servers.
declare type SDKControlMcpSetServersRequest = {
    subtype: 'mcp_set_servers';
    servers: Record<string, coreTypes.McpServerConfigForProcessTransport>;

};

// Requests the current status of all MCP server connections.
declare type SDKControlMcpStatusRequest = {
    subtype: 'mcp_status';
    // When true and the CLI writes to the session through CCR (a cloud session's worker, or a Remote Control worker), the success reply is published only to the clients attached at that moment, as a control_response marked live_only: true on the session's unsequenced event stream, and is not stored with the session's events; a cloud session's worker that does not advertise `mcp_status_no_config_v1` embeds each server's configuration (URLs, headers, OAuth settings) in its rows.
    liveReply?: boolean;
};

// Enables or disables an MCP server.
declare type SDKControlMcpToggleRequest = {
    subtype: 'mcp_toggle';
    serverName: string;
    enabled: boolean;
};

// Requests permission to use a tool with the given input.
declare type SDKControlPermissionRequest = {
    subtype: 'can_use_tool';
    tool_name: string;
    mcp_server?: coreTypes.McpServerProvenance;
    input: Record<string, unknown>;
    permission_suggestions?: coreTypes.PermissionUpdate[];
    blocked_path?: string;
    // Human-readable reason the ask escalated, for the consent line of the host's dialog.
    decision_reason?: string;
    // Structured discriminator for why auto-mode escalated.
    decision_reason_type?: 'rule' | 'mode' | 'subcommandResults' | 'permissionPromptTool' | 'hook' | 'asyncAgent' | 'sandboxOverride' | 'workingDir' | 'safetyCheck' | 'classifier' | 'other';

    // Set when a safetyCheck is present anywhere in the decision reason (including nested inside subcommandResults for compound bash).
    classifier_approvable?: boolean;
    // True when the dialog must not offer the persistent "don't ask again" row for this ask: accepting it would write a whole-tool allow rule broader than the ask's own verb (PermissionAskDecision.suppressAlwaysAllowRule), or the ask comes from a run a plugin started, where Claude Code saves no rule from the answer, not even for the session.
    suppress_always_allow_rule?: boolean;
    // True when the ask must not be approvable by a single stray keystroke (PermissionAskDecision.defaultToNo): a terminal-style prompt opens on its decline option and takes no digit shortcut.
    default_to_no?: boolean;
    // Set when a user-configured ask RULE (permissions.ask) forced this prompt but the ask carries the tool's own decision_reason — the ask-rule substitution keeps the richer tool-minted ask, so the rule rides here instead of decision_reason_type 'rule'.
    matched_ask_rule?: {
        source: string;
        tool_name: string;
        rule_content?: string;
    };
    title?: string;
    display_name?: string;
    tool_use_id: string;
    agent_id?: string;

    description?: string;
    // True when one-tap Approve/Deny must not be offered: the tool's approval card IS the user-interaction surface (Tool.requiresUserInteraction(): the user responds on the card itself), OR the pending ask is localDisplayOnly (its consent disclosure cannot ride this wire and only the local dialog renders it), OR the auto-mode classifier blocked the call and it was handed to a person (decision_reason_type 'classifier' on a person-only ask), which no stored grant or automatic approval may answer, OR a grant keyed on the tool name must not answer it: the MCP server's name spells a claude.ai connector's key and the server came from neither claude.ai nor whoever launched the session, or, while Claude Code enforces grants, two tools of the server get that name, OR the SDK host worded the ask (title and description are its consent copy, which a push cannot carry whole), OR the ask stands in for the classifier's verdict on a call to Claude Desktop's Cowork shell, which no stored grant or automatic approval may answer.
    requires_user_interaction?: boolean;

};

// The session's live permission rules state, as list_permission_rules reports it.
export declare type SDKControlPermissionRulesState = {
    rules: SDKPermissionRuleEntry[];
    workspaceDirectories: SDKPermissionWorkspaceDirectory[];
    // The session's original working directory.
    originalCwd: string;
    // True when enterprise managed settings pin allowManagedPermissionRulesOnly: the session applies policy rules only, and rules from other settings files appear with notInEffect set.
    managedOnly: boolean;

    // Settings parse and validation errors, as get_settings reports them.
    errors?: coreTypes.SDKSettingsParseError[];
    // True when this answer was cut because the connection stores its answers (a cloud or self-hosted session): `rules` holds only rows from settings sources and `mcpServerPolicy`, `workspaceDirectories` only folders from settings sources, each entry of `errors` only a fixed `message` and an empty `path`, and `parentSettingsIncluded` is absent.
    withheld?: boolean;
};

// Read a file from the session filesystem for the remote sidebar viewer.
declare type SDKControlReadFileRequest = {
    subtype: 'read_file';
    path: string;
    max_bytes?: number;
    // How to encode the bytes in `contents`.
    encoding?: 'utf-8' | 'base64';

};

// File contents for the remote sidebar viewer.
export declare type SDKControlReadFileResponse = {
    contents: string;
    absPath: string;
    truncated?: boolean;
    // Set when the request asked for base64.
    encoding?: 'base64';
    // Present on every utf-8 answer.
    lossy?: boolean;

};

// Add a directory as a working-directory root and optionally reload CLAUDE.md, skills, and plugins.
declare type SDKControlRegisterRepoRootRequest = {
    subtype: 'register_repo_root';
    directory: string;
    reload_claude_md?: boolean;
    reload_plugins?: boolean;
    reload_skills?: boolean;
};

// Re-reads the output-style directories from disk (a style file written mid-session is otherwise invisible until the next session) and returns the refreshed style names.
declare type SDKControlReloadOutputStylesRequest = {
    subtype: 'reload_output_styles';
};

// Refreshed output style names after reload, built-in and custom, in the order the initialize response lists them.
export declare type SDKControlReloadOutputStylesResponse = {
    available_output_styles: string[];
};

// Reloads plugins from disk and returns the refreshed session components.
declare type SDKControlReloadPluginsRequest = {
    subtype: 'reload_plugins';
    // When true, the reload is not applied if applying it would change the session's tool list while the conversation's prompt cache depends on that list (the same check the interactive /reload-plugins makes before it asks for --force): the response then carries held: true and cache_impact, and the session keeps its current plugins.
    hold_on_cache_impact?: boolean;
    // Plugins for the reload to load as well, in the same shape as the SDK `plugins` option: each entry joins the session as one more --plugin-dir would (--plugin-dir-no-mcp when skipMcpDiscovery is set).
    add_plugins?: coreTypes.SdkPluginConfig[];
    // When true, a request that finds the session busy reloads nothing and gets an error whose error_code is reload_plugins_busy; ask again once the session is idle.
    refuse_if_busy?: boolean;
};

// Refreshed commands, agents, plugins, and MCP server status after reload.
export declare type SDKControlReloadPluginsResponse = {
    commands: coreTypes.SlashCommand[];
    agents: coreTypes.AgentInfo[];
    plugins: {
        name: string;
        path: string;
        source?: string;
        // The plugin's version as declared in its plugin.json manifest, emitted verbatim (plugin-author-controlled — validate before trusting).
        version?: string;
    }[];
    mcpServers: coreTypes.McpServerStatus[];
    error_count: number;
    // Present only when the request asked to hold on cache impact and this CLI ran the check.
    held?: boolean;
    // What applying the held reload would change in the session's tool list: plugin MCP servers it would register or drop (scoped plugin:<plugin>:<server> names, plugin-authored — validate before showing) and whether it would add or remove the LSP tool (the may- forms mean the preview could not fully see the pending plugin set).
    cache_impact?: {
        mcp_servers_added: string[];
        mcp_servers_removed: string[];
        lsp_tool_change: ('adds' | 'may-add' | 'removes' | 'may-remove') | null;
    };
};

// Reloads skills from disk and returns the refreshed skill list.
declare type SDKControlReloadSkillsRequest = {
    subtype: 'reload_skills';
};

// Refreshed skill commands after reload.
export declare type SDKControlReloadSkillsResponse = {
    skills: coreTypes.SlashCommand[];
};

// Sets the user-facing title for the current session.
declare type SDKControlRenameSessionRequest = {
    subtype: 'rename_session';
    title: string;
    // Who chose the title: 'remote' (the default) for a rename made on claude.ai and relayed to this process, 'host' for one the user made in the hosting application (an IDE), which the CLI counts as a user rename.
    source?: 'remote' | 'host';
    // The session the title is for.
    session_id?: string;

};

// Envelope for a control-protocol request, sent by either side on the same stream as the messages.
export declare type SDKControlRequest = {
    type: 'control_request';
    // Chosen by the sender, unique among its in-flight requests; the control_response (and any control_cancel_request) for this request echoes it.
    request_id: string;
    request: SDKControlRequestInner;

};

declare type SDKControlRequestInner = SDKControlInterruptRequest | SDKControlPermissionRequest | SDKControlInitializeRequest | SDKControlSetPermissionModeRequest | SDKControlSetModelRequest | SDKControlSetMaxThinkingTokensRequest | SDKControlRenameSessionRequest | SDKControlSetColorRequest | SDKControlMcpStatusRequest | SDKControlGetContextUsageRequest | SDKControlGetSessionCostRequest | SDKControlListModelsRequest | SDKControlGetUsageRequest | SDKControlGetBinaryVersionRequest | SDKControlMcpCallRequest | SDKControlFileSuggestionsRequest | SDKHookCallbackRequest | SDKControlMcpMessageRequest | SDKControlRewindFilesRequest | SDKControlCancelAsyncMessageRequest | SDKControlReadFileRequest | SDKControlSeedReadStateRequest | SDKControlMcpSetServersRequest | SDKControlRegisterRepoRootRequest | SDKControlReloadPluginsRequest | SDKControlReloadSkillsRequest | SDKControlReloadOutputStylesRequest | SDKControlMcpReconnectRequest | SDKControlMcpToggleRequest | SDKControlSetHasSubscribersRequest | SDKControlStopTaskRequest | SDKControlBackgroundTasksRequest | SDKControlGetTaskOutputRequest | SDKControlApplyFlagSettingsRequest | SDKControlGetSettingsRequest | SDKControlGetHooksListingRequest | SDKControlUpdateSettingsRequest | SDKControlElicitationRequest | SDKControlRequestUserDialogRequest | SDKControlListPermissionRulesRequest | SDKControlMcpReadResourceRequest;

// Progress for a long-running client-originated control_request (currently only side_question), correlated by request_id.
export declare type SDKControlRequestProgressMessage = {
    type: 'system';
    subtype: 'control_request_progress';
    // request_id of the in-flight control_request this progress belongs to.
    request_id: string;
    status: 'started' | 'api_retry';
    attempt?: number;
    max_retries?: number;
    retry_delay_ms?: number;
    error_status?: number | null;
    uuid: UUID;
    session_id: string;
};

// Requests the SDK consumer to render a tool-driven blocking dialog and return the user choice.
declare type SDKControlRequestUserDialogRequest = {
    subtype: 'request_user_dialog';
    // Identifier for the dialog the host should render.
    dialog_kind: string;
    // Dialog-specific data passed to the host renderer.
    payload: Record<string, unknown>;
    tool_use_id?: string;
};

// Envelope for the single reply to a control_request, sent by whichever side received the request.
export declare type SDKControlResponse = {
    type: 'control_response';
    response: ControlResponse | ControlErrorResponse;

};

// Rewinds file changes made since a specific user message.
declare type SDKControlRewindFilesRequest = {
    subtype: 'rewind_files';
    user_message_id: string;
    dry_run?: boolean;
};

// Seeds the readFileState cache with a path+mtime entry.
declare type SDKControlSeedReadStateRequest = {
    subtype: 'seed_read_state';
    path: string;
    mtime: number;
};

// Sets the session accent color.
declare type SDKControlSetColorRequest = {
    subtype: 'set_color';
    color: string;
};

// Tells the CLI whether anybody is watching the output that the host reads from stream-json stdout and passes on.
declare type SDKControlSetHasSubscribersRequest = {
    subtype: 'set_has_subscribers';
    // false: nobody watches now.
    has_subscribers: boolean | null;
};

// Sets the maximum number of thinking tokens for extended thinking.
declare type SDKControlSetMaxThinkingTokensRequest = {
    subtype: 'set_max_thinking_tokens';
    max_thinking_tokens?: number | null;
    thinking_display?: ('summarized' | 'omitted' | 'highlights') | null;
};

// Sets the model to use for subsequent conversation turns.
declare type SDKControlSetModelRequest = {
    subtype: 'set_model';
    // Model to switch to.
    model?: string | null;

};

// Sets the permission mode for tool execution handling.
declare type SDKControlSetPermissionModeRequest = {
    subtype: 'set_permission_mode';
    // Permission mode for controlling how tool executions are handled.
    mode: coreTypes.PermissionMode;

};

// Stops a running task.
declare type SDKControlStopTaskRequest = {
    subtype: 'stop_task';
    task_id: string;
};

// Writes settings through the CLI's own writer.
declare type SDKControlUpdateSettingsRequest = {
    subtype: 'update_settings';
    // Which settings file to write: the project's local settings file, where /config's writes land, or the user's settings file, which takes effortLevel only.
    source: 'localSettings' | 'userSettings';
    settings: Record<string, unknown>;
};

// Emitted by /clear, plan-mode exit, fresh-session, and onboarding flows.
export declare type SDKConversationResetMessage = {
    type: 'conversation_reset';
    new_conversation_id: UUID;
    uuid: UUID;
    session_id: string;
    // What discarded the conversation: 'clear' is the /clear command (or its /reset and /new aliases), 'plan_mode_exit' is leaving plan mode with the clear-context option, 'fresh_session' is a flow that starts a fresh session to implement an approved plan, 'onboarding' is an onboarding flow re-run inside an existing session.
    trigger?: 'clear' | 'plan_mode_exit' | 'fresh_session' | 'onboarding';
    // Only with trigger 'clear': the uuid of the user message whose /clear was executed (the client's own uuid for that message when it came from a Remote Control or stream-json client, otherwise the uuid the CLI assigned to the typed command).
    user_message_uuid?: string;
    // When the reset happened, as an ISO 8601 string in UTC read from the clock of the process that performed it.
    timestamp?: string;
};

export declare type SDKDeferredToolUse = {
    id: string;
    name: string;
    input: Record<string, unknown>;
};

// Emitted when an MCP server confirms that a URL-mode elicitation is complete.
export declare type SDKElicitationCompleteMessage = {
    type: 'system';
    subtype: 'elicitation_complete';
    mcp_server_name: string;
    elicitation_id: string;
    uuid: UUID;
    session_id: string;
};

export declare type SDKFilesPersistedEvent = {
    type: 'system';
    subtype: 'files_persisted';
    files: {
        filename: string;
        file_id: string;
    }[];
    failed: {
        filename: string;
        error: string;
    }[];
    processed_at: string;
    uuid: UUID;
    session_id: string;
};

// Configuration for matching and routing hook callbacks.
declare type SDKHookCallbackMatcher = {
    matcher?: string;
    // Opaque ids chosen by the client, one per hook function it registered for this matcher.
    hookCallbackIds: string[];
    timeout?: number;
    // What a failure of these callbacks does: the client answered the hook_callback request with an error, or with an answer that fails validation.
    onFailure?: 'continue' | 'block';
};

// Delivers a hook callback with its input data.
declare type SDKHookCallbackRequest = {
    subtype: 'hook_callback';
    callback_id: string;
    input: coreTypes.HookInput;
    tool_use_id?: string;

};

export declare type SDKHookProgressMessage = {
    type: 'system';
    subtype: 'hook_progress';
    hook_id: string;
    hook_name: string;
    hook_event: string;
    stdout: string;
    stderr: string;
    output: string;
    uuid: UUID;
    session_id: string;
};

export declare type SDKHookResponseMessage = {
    type: 'system';
    subtype: 'hook_response';
    hook_id: string;
    hook_name: string;
    hook_event: string;
    output: string;
    stdout: string;
    stderr: string;
    exit_code?: number;
    outcome: 'success' | 'error' | 'cancelled';
    uuid: UUID;
    session_id: string;
};

export declare type SDKHookStartedMessage = {
    type: 'system';
    subtype: 'hook_started';
    hook_id: string;
    hook_name: string;
    hook_event: string;
    uuid: UUID;
    session_id: string;
};

// Generic text banner emitted by the loop — non-error status lines, hook feedback (e.g.
export declare type SDKInformationalMessage = {
    type: 'system';
    subtype: 'informational';
    content: string;
    // Render level.
    level: 'info' | 'notice' | 'suggestion' | 'warning';
    // Dedupes progress messages for the same tool use.
    tool_use_id?: string;
    // When true, execution stops after this message (e.g.
    prevent_continuation?: boolean;
    // Opaque feature tag on a line a host may treat specially; absent on ordinary lines.
    tag?: string;

    uuid: UUID;
    session_id: string;
};

// Liveness heartbeat.
declare type SDKKeepAliveMessage = {
    type: 'keep_alive';
};

// Output from a local slash command (e.g.
export declare type SDKLocalCommandOutputMessage = {
    type: 'system';
    subtype: 'local_command_output';
    content: string;
    uuid: UUID;
    session_id: string;
};

export declare type SDKMcpResourceLink = {
    uri: string;
    name: string;
    title?: string;
    description?: string;
    mimeType?: string;
    size?: number;
    annotations?: Record<string, unknown>;
};

// MCP tool definition for SDK servers.
export declare type SdkMcpToolDefinition<Schema extends AnyZodRawShape = AnyZodRawShape> = {
    name: string;
    description: string;
    inputSchema: Schema;
    annotations?: ToolAnnotations;
    _meta?: Record<string, unknown>;
    handler: (args: InferShape<Schema>, extra: unknown) => Promise<CallToolResult>;
};

// Emitted when the memory recall supervisor surfaces relevant memories into the turn.
export declare type SDKMemoryRecallMessage = {
    type: 'system';
    subtype: 'memory_recall';
    // How memories were surfaced: 'select' returns full file bodies chosen by the parallel selector; 'synthesize' returns a Sonnet-authored paragraph distilled from many tiny memories.
    mode: 'select' | 'synthesize';
    memories: {
        // Absolute path to the memory file, a synthesis sentinel of the form `<synthesis:DIR>` when mode is 'synthesize', or an https URL when scope is 'organization'.
        path: string;
        scope: 'personal' | 'team' | 'organization';
        // The surfaced memory body.
        content?: string;
    }[];
    uuid: UUID;
    session_id: string;
};

// Every conversational and informational message the CLI emits on its output stream, discriminated by type (and subtype for system/result messages).
export declare type SDKMessage = SDKAssistantMessage | SDKUserMessage | SDKUserMessageReplay | SDKResultMessage | SDKSystemMessage | SDKPartialAssistantMessage | SDKCompactBoundaryMessage | SDKStatusMessage | SDKAPIRetryMessage | SDKControlRequestProgressMessage | SDKModelRefusalFallbackMessage | SDKModelRefusalNoFallbackMessage | SDKLocalCommandOutputMessage | SDKHookStartedMessage | SDKHookProgressMessage | SDKHookResponseMessage | SDKPluginInstallMessage | SDKToolProgressMessage | SDKAuthStatusMessage | SDKTaskNotificationMessage | SDKTaskStartedMessage | SDKTaskUpdatedMessage | SDKTaskProgressMessage | SDKBackgroundTasksChangedMessage | SDKThinkingTokensMessage | SDKSessionStateChangedMessage | SDKWorkerShuttingDownMessage | SDKCommandsChangedMessage | SDKNotificationMessage | SDKFilesPersistedEvent | SDKToolUseSummaryMessage | SDKMemoryRecallMessage | SDKRateLimitEvent | SDKElicitationCompleteMessage | SDKPermissionDeniedMessage | SDKPromptSuggestionMessage | SDKMirrorErrorMessage | SDKInformationalMessage | SDKConversationResetMessage | SDKSessionWritesLostMessage;

// Provenance of a user-role message (peer session, team lead, channel).
export declare type SDKMessageOrigin = {
    kind: 'human';
} | {
    kind: 'channel';
    server: string;
} | {
    kind: 'peer';
    from: string;
    // The SENDING session's permission class as declared by the host that injects this message on local stdin ('bypass' for sessions that run tools without asking, 'prompting' otherwise).
    fromMode?: 'bypass' | 'prompting';
    // Sender display name, normalized by the harness: Unicode control, format, surrogate, and line/paragraph-separator code points stripped (categories Cc/Cf/Cs/Zl/Zp — covers bidi controls, zero-width characters, and tag characters), trimmed, at most 64 code points (+ ellipsis, never splitting a surrogate pair).
    name?: string;
    // The sender's host-openable session id (the envelope's `from-session` attribute — e.g.
    fromSession?: string;

    // Task id of the in-process background subagent that sent this message, stamped by the harness from the sending loop (never from tool input).
    senderTaskId?: string;
    // Decoded message body with the peer envelope stripped — byte-exact with what the model sees.
    body?: string;
    // Kernel-verified pid of the process that connected to this session's cross-session messaging socket, read from the connection (SO_PEERCRED / LOCAL_PEERPID) — never from the payload.
    verifiedPeerPid?: number;

} | {
    kind: 'task-notification';
    // Present when the delivery is the fired stored prompt of a scheduled task ('scheduled-trigger', stamped from server-asserted provenance, or declared by a local host for its own scheduled runs; the schedule attests storage, not authorship) or a coordinator co-member SendMessage delivery ('peer-send-message': model-authored text from another of the same user's sessions, verified by the server-stamped receiver co-membership — task-notification for prompt authority, but distinguishable so the receive-side crossSessionInbound setting can apply to it).
    subkind?: 'scheduled-trigger' | 'peer-send-message' | 'projects-relay' | 'session-inbox';
    // On a 'scheduled-trigger' delivery, why it fired: a short lowercase token such as 'scheduled', 'manual', 'retry', 'catch_up' or 'api'.
    fireReason?: string;

    // On a background task's own notification: the run_id its task_* events carried, as of when the notification was queued.
    runId?: string;

} | {
    kind: 'coordinator';
} | {
    kind: 'unclassified';
} | {
    kind: 'observer';
    from: string;
    senderTaskId: string;
} | {
    kind: 'auto-continuation';
} | {
    kind: 'observer-activity';
};

// Emitted when SessionStore.append() rejects or times out for a transcript-mirror batch after bounded retry (3 attempts with short backoff; timeouts are not retried).
export declare type SDKMirrorErrorMessage = {
    type: 'system';
    subtype: 'mirror_error';
    error: string;
    key: {
        projectKey: string;
        sessionId: string;
        subpath?: string;
    };
    uuid: UUID;
    session_id: string;
};

// Emitted when the primary model ends the stream with stop_reason "refusal" and the turn is retried once on a fallback model (direction: "retry").
export declare type SDKModelRefusalFallbackMessage = {
    type: 'system';
    subtype: 'model_refusal_fallback';
    trigger: 'refusal';
    direction: 'retry' | 'revert' | 'sticky';
    // 'session': the main thread fell back and the session model is swapped.
    scope?: 'session' | 'local';
    original_model: string;
    fallback_model: string;
    request_id: string | null;
    // The refusal category ('cyber', 'bio', …): stop_details.category from the refused API response (client lane), or the fallback block's server-gated trigger.category (server lane).
    api_refusal_category?: string | null;

    // stop_details.explanation from the refused API response (client lane only — the server-lane trigger carries no explanation).
    api_refusal_explanation?: string | null;

    // Wire uuids of the messages this fallback retracted — the refused partial as the consumer received it (one uuid per normalized SDK message; multi-block messages carry per-block derived uuids) plus any tombstoned tool_results.
    retracted_message_uuids?: string[];
    // UUID of the user message the refused request was for — the rewind target and composer prefill for edit-and-retry.
    refused_user_message_uuid?: string | null;
    content: string;
    uuid: UUID;
    session_id: string;
};

// Emitted when the model ends the stream with stop_reason "refusal" and no retry runs: no fallback model is configured, or per-category routing declined the retry (the mapped fallback target is unresolvable, or CLAUDE_CODE_REFUSAL_FALLBACK_CATCH_ALL is explicitly disabled and the refusal category has no fallback map entry).
export declare type SDKModelRefusalNoFallbackMessage = {
    type: 'system';
    subtype: 'model_refusal_no_fallback';
    original_model: string;
    request_id: string | null;
    api_refusal_category?: string | null;
    api_refusal_explanation?: string | null;
    refused_user_message_uuid?: string | null;
    content: string;
    uuid: UUID;
    session_id: string;
};

// Loop-side text notification.
export declare type SDKNotificationMessage = {
    type: 'system';
    subtype: 'notification';
    key: string;
    text: string;
    priority: 'low' | 'medium' | 'high' | 'immediate';
    color?: string;
    timeout_ms?: number;
    uuid: UUID;
    session_id: string;
};

// An incremental streaming event for the assistant message being generated, emitted only when partial messages are requested (--include-partial-messages).
export declare type SDKPartialAssistantMessage = {
    type: 'stream_event';
    // One Anthropic Messages API streaming event (message_start, content_block_start, content_block_delta, content_block_stop, message_delta, message_stop) as defined for the streaming Messages API.
    event: BetaRawMessageStreamEvent;
    parent_tool_use_id: string | null;
    uuid: UUID;
    session_id: string;
    ttft_ms?: number;

    // Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on a non-ping stream event each time that send changes: the turn's FIRST non-ping stream event (normally the frame that triggers the turn's initial ack), and, for a turn started by a synthetic (meta) prompt, the first non-ping stream event after each queued user message folded in mid-turn takes the echo over (see SDKAssistantMessage.user_message_uuid for the rule) — so a consumer can bind the reply stream to the send it answers without waiting for the result.
    user_message_uuid?: string;
    // Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host merged into this one turn (several messages sent close together run as one turn whose user_message_uuid is the LAST member's), then any user message folded into the turn before this frame — so a consumer that sent any of them can bind this reply to its own send by finding its uuid anywhere in the list.
    user_message_uuids?: string[];
    // Why this frame's turn continues one that a worker restart interrupted.
    resume_reason?: string;

};

export declare type SDKPermissionDenial = {
    tool_name: string;
    tool_use_id: string;
    tool_input: Record<string, unknown>;
};

// Emitted when a tool call is auto-denied without an interactive permission prompt (e.g.
export declare type SDKPermissionDeniedMessage = {
    type: 'system';
    subtype: 'permission_denied';
    tool_name: string;
    tool_use_id: string;
    // Subagent ID when the denied tool call originated inside a subagent.
    agent_id?: string;
    // Discriminator from PermissionDecisionReason (e.g.
    decision_reason_type?: string;

    // Human-readable reason from the deciding component, when available.
    decision_reason?: string;
    // The rejection message returned to the model in the tool_result.
    message: string;
    uuid: UUID;
    session_id: string;
};

// The CLI's plain-language reading of a rule (e.g.
export declare type SDKPermissionRuleDescription = {
    prefix: string;
    emphasis?: string;
    suffix?: string;
};

// One permission rule with its provenance and where it lives.
export declare type SDKPermissionRuleEntry = {
    behavior: 'allow' | 'deny' | 'ask';
    // Where the rule comes from.
    source: 'userSettings' | 'projectSettings' | 'localSettings' | 'flagSettings' | 'policySettings' | 'cliArg' | 'command' | 'session' | 'mcpMount' | 'toolsNarrowing' | 'mcpServerPolicy' | 'hostCredential';
    // The stored rule string VERBATIM, exactly as the session holds it; where the state's `withheld` is true, the text may be cut.
    rule: string;
    // Plain-language reading of the rule; absent where the terminal shows no subtitle either.
    description?: SDKPermissionRuleDescription;
    // Where the rule lives: 'persistent' (userSettings/projectSettings/localSettings — saved in a settings file), 'session' (cliArg/session — in memory only, for the rest of this session), or 'readonly' (policySettings/flagSettings/command and every other source — the set the terminal's /permissions treats as read-only).
    editability: 'persistent' | 'session' | 'readonly';
    // Present (true) when enterprise managed settings pin allowManagedPermissionRulesOnly and this rule, read from a non-policy settings file, is ignored by the session.
    notInEffect?: boolean;
};

// One additional working directory in the permission scope.
export declare type SDKPermissionWorkspaceDirectory = {
    path: string;
    // Where the directory grant came from: a settings source (e.g.
    source: string;
};

// Configuration for loading a plugin.
export declare type SdkPluginConfig = {
    // Plugin type.
    type: 'local';
    // Absolute or relative path to the plugin directory
    path: string;
    // When true, the engine loads skills/hooks/agents/commands from this plugin but does NOT read its .mcp.json or manifest mcpServers.
    skipMcpDiscovery?: boolean;
};

// Headless plugin installation progress.
export declare type SDKPluginInstallMessage = {
    type: 'system';
    subtype: 'plugin_install';
    status: 'started' | 'installed' | 'failed' | 'completed';
    name?: string;
    error?: string;
    uuid: UUID;
    session_id: string;
};

// Predicted next user prompt, emitted after each turn when promptSuggestions is enabled.
export declare type SDKPromptSuggestionMessage = {
    type: 'prompt_suggestion';
    suggestion: string;
    uuid: UUID;
    session_id: string;
};

// Rate limit event emitted when rate limit info changes.
export declare type SDKRateLimitEvent = {
    type: 'rate_limit_event';
    // Rate limit information for claude.ai subscription users.
    rate_limit_info: SDKRateLimitInfo;
    uuid: UUID;
    session_id: string;
};

// Rate limit information for claude.ai subscription users.
export declare type SDKRateLimitInfo = {
    status: 'allowed' | 'allowed_warning' | 'rejected';
    resetsAt?: number;
    rateLimitType?: 'five_hour' | 'seven_day' | 'seven_day_opus' | 'seven_day_sonnet' | 'seven_day_overage_included' | 'overage';
    utilization?: number;

    overageStatus?: 'allowed' | 'allowed_warning' | 'rejected';
    overageResetsAt?: number;
    overageDisabledReason?: 'overage_not_provisioned' | 'org_level_disabled' | 'org_level_disabled_until' | 'out_of_credits' | 'seat_tier_level_disabled' | 'member_level_disabled' | 'seat_tier_zero_credit_limit' | 'group_zero_credit_limit' | 'member_zero_credit_limit' | 'org_service_level_disabled' | 'no_limits_configured' | 'fetch_error' | 'unknown';
    isUsingOverage?: boolean;
    overageInUse?: boolean;
    overageEnabled?: boolean;
    surpassedThreshold?: number;

    // Which spend limit blocked the request when it is not the member's own cap: 'group_pool' means a pooled group budget shared by the member's team is used up (the denial otherwise looks like the member's own monthly cap).
    limitScope?: 'service' | 'channel' | 'group_pool';
    errorCode?: 'credits_required';
    canUserPurchaseCredits?: boolean;
    hasChargeableSavedPaymentMethod?: boolean;

};

export declare type SDKResultError = {
    type: 'result';
    subtype: 'error_during_execution' | 'error_max_turns' | 'error_max_budget_usd' | 'error_max_structured_output_retries';
    duration_ms: number;
    duration_api_ms: number;
    is_error: boolean;
    num_turns: number;
    stop_reason: string | null;
    // Cumulative estimated cost in USD for this query() call, covering the same query-pipeline calls as modelUsage and sharing its lifecycle: cumulative across turns in streaming-input sessions — each result carries the running total so far, so read the latest result rather than summing across results.
    total_cost_usd: number;
    // MAIN AGENT LOOP ONLY — excludes Task subagent, sidechain, and auxiliary model calls, and is per-turn in streaming-input sessions.
    usage: NonNullableUsage;
    // Per-model totals for every model call made through the query pipeline during this query() call — main loop, Task subagents, sidechains, and internal calls such as compaction and Workflow agents.
    modelUsage: Record<string, ModelUsage>;

    permission_denials: SDKPermissionDenial[];
    // User-initiated sends still waiting in the command queue when this result was produced.
    queued_turn_count?: number;
    errors: string[];

    // Set on the zeroed error_during_execution result a stream-json run writes before exiting on a known startup failure; errors carries the failure message.
    startup_failure_reason?: SDKStartupFailureReason;
    // Set when startup_failure_reason is config_file_invalid: the config file that did not parse and its backup, as separate fields.
    invalid_config_file?: {
        // The file that is not valid JSON.
        path: string;
        // Whether this is Claude Code's global config file.
        is_global: boolean;
        // Whether Claude Code knows the result of its search for a backup of the file.
        backup_searched: boolean;
        // The backup that the message in errors names.
        backup_path?: string;
    };
    // Client uuid of the user message that triggered this turn (submitMessage options.uuid), echoed back so a consumer can link this error result to the send it answers — the same join key the success variant echoes, carried alone (error turns have no request_sent_wall_ms to report).
    user_message_uuid?: string;
    // Client uuids of every user message whose prompt this turn consumed, in consumption order — all members of a prompt batch the host merged into this one turn (several messages sent close together run as one turn whose user_message_uuid is the LAST member's), then any queued user message folded into the running turn between tool rounds, once taken off the queue — so a consumer that sent any of them can bind this result to its own send by finding its uuid anywhere in the list.
    user_message_uuids?: string[];
    // Why this turn continued one that a worker restart interrupted.
    resume_reason?: string;

    terminal_reason?: TerminalReason;

    // Delivery sequence of this result within the run: how many results the run numbered before this one, starting at 0, in the order the process writes them.
    result_index?: number;
    // When the claude.ai credential this process's requests send ends for good, as an ISO 8601 instant, read as the result is written.
    credential_expires_at?: string;
    fast_mode_state?: FastModeState;
    fast_mode_disabled_reason?: FastModeDisabledReason;
    origin?: SDKMessageOrigin;
    uuid: UUID;
    session_id: string;
};

// The outcome of a turn.
export declare type SDKResultMessage = SDKResultSuccess | SDKResultError;

export declare type SDKResultSuccess = {
    type: 'result';
    subtype: 'success';
    duration_ms: number;
    duration_api_ms: number;
    ttft_ms?: number;
    ttft_stream_ms?: number;
    time_to_request_ms?: number;

    user_message_uuid?: string;
    user_message_uuids?: string[];
    resume_reason?: string;
    local_command?: string;
    request_sent_wall_ms?: number;

    first_content_frame_ms?: number;
    first_stream_post_ms?: number;
    first_stream_post_ack_ms?: number;
    first_stream_post_queue_wait_ms?: number;
    first_stream_post_queued_behind?: 'durable_post' | 'ephemeral_post' | 'retry_backoff' | 'hold' | 'none';
    first_stream_post_wall_ms?: number;

    first_text_post_ms?: number;
    first_text_post_queue_wait_ms?: number;
    first_text_post_queued_behind?: 'durable_post' | 'ephemeral_post' | 'retry_backoff' | 'hold' | 'none';
    first_text_post_wall_ms?: number;

    time_to_request_from_spawn_ms?: number;
    warm_spare_claimed?: boolean;

    time_origin_ms?: number;

    is_error: boolean;
    api_error_status?: number | null;

    num_turns: number;
    result: string;
    stop_reason: string | null;
    // Cumulative estimated cost in USD for this query() call, covering the same query-pipeline calls as modelUsage and sharing its lifecycle: cumulative across turns in streaming-input sessions — each result carries the running total so far, so read the latest result rather than summing across results.
    total_cost_usd: number;
    // MAIN AGENT LOOP ONLY — excludes Task subagent, sidechain, and auxiliary model calls, and is per-turn in streaming-input sessions.
    usage: NonNullableUsage;
    // Per-model totals for every model call made through the query pipeline during this query() call — main loop, Task subagents, sidechains, and internal calls such as compaction and Workflow agents.
    modelUsage: Record<string, ModelUsage>;

    permission_denials: SDKPermissionDenial[];
    // User-initiated sends still waiting in the command queue when this result was produced.
    queued_turn_count?: number;
    structured_output?: unknown;
    deferred_tool_use?: SDKDeferredToolUse;
    terminal_reason?: TerminalReason;

    // Delivery sequence of this result within the run: how many results the run numbered before this one, starting at 0, in the order the process writes them.
    result_index?: number;
    // When the claude.ai credential this process's requests send ends for good, as an ISO 8601 instant, read as the result is written.
    credential_expires_at?: string;
    fast_mode_state?: FastModeState;
    fast_mode_disabled_reason?: FastModeDisabledReason;
    origin?: SDKMessageOrigin;
    uuid: UUID;
    session_id: string;
};

// Session metadata returned by listSessions and getSessionInfo.
export declare type SDKSessionInfo = {
    // Unique session identifier (UUID).
    sessionId: string;
    // Display title for the session: custom title, auto-generated summary, or first prompt.
    summary: string;
    // Last modified time in integer milliseconds since epoch.
    lastModified: number;
    // File size in bytes.
    fileSize?: number;
    // User-set session title via /rename.
    customTitle?: string;
    // First meaningful user prompt in the session.
    firstPrompt?: string;
    // Git branch at the end of the session.
    gitBranch?: string;
    // Working directory for the session.
    cwd?: string;
    // User-set session tag.
    tag?: string;
    // Creation time in integer milliseconds since epoch, extracted from the first entry's timestamp.
    createdAt?: number;
    // Whether the session is archived: false if it was and has been unarchived, or a prompt was sent in it since.
    archived?: boolean;
};

// Mirrors notifySessionStateChanged.
export declare type SDKSessionStateChangedMessage = {
    type: 'system';
    subtype: 'session_state_changed';
    state: 'idle' | 'running' | 'requires_action';
    uuid: UUID;
    session_id: string;
};

// The CLI could not save lines of a session transcript, changes to ~/.claude.json, or both, because the disk was full or its quota was used up.
export declare type SDKSessionWritesLostMessage = {
    type: 'system';
    subtype: 'session_writes_lost';
    // Why the writes failed.
    error_code: 'ENOSPC' | 'EDQUOT';
    // What was not saved.
    lost: ('transcript' | 'global_config')[];
    // The sessions whose transcripts lost lines, in the order each first lost one.
    transcript_session_ids?: string[];
    uuid: UUID;
    session_id: string;
};

// A settings file parse or validation error.
export declare type SDKSettingsParseError = {
    // Path to the settings file that failed to parse or validate.
    file?: string;
    // Dot-notation path to the field with the error, or empty string for whole-file errors.
    path: string;
    // Human-readable error message.
    message: string;
};

// Why Claude Code refused to start, so a host can offer the fix instead of a retry.
export declare type SDKStartupFailureReason = 'org_pin_api_key_conflict' | 'provider_not_allowed' | 'org_verify_failed' | 'org_pin_mismatch' | 'managed_settings_invalid' | 'remote_settings_required_unavailable' | 'gateway_signin_required' | 'gateway_access_denied' | 'proxy_invalid' | 'temp_dir_unusable' | 'cwd_unavailable' | 'shell_tool_missing' | 'session_held_by_background' | 'worktree_resume_refused' | 'worktree_unverified' | 'cli_version_too_old' | 'bypass_root' | 'org_config_required_unavailable' | 'org_config_refused' | 'sandbox_required_unavailable' | 'initialize_not_first' | 'config_file_invalid' | 'managed_settings_not_delivered';

export declare type SDKStatus = 'compacting' | 'requesting' | null;

export declare type SDKStatusMessage = {
    type: 'system';
    subtype: 'status';
    status: SDKStatus;
    permissionMode?: PermissionMode;
    compact_result?: 'success' | 'failed';
    compact_error?: string;

    uuid: UUID;
    session_id: string;
};

// Session metadata the CLI emits at the start of each turn, normally ahead of every other message of that turn: session_id, model, working directory, tools, MCP servers, slash commands, permission mode, and the capabilities list for feature detection.
export declare type SDKSystemMessage = {
    type: 'system';
    subtype: 'init';
    agents?: string[];
    // Where the credential used for API requests came from: 'ANTHROPIC_API_KEY' (environment variable), 'apiKeyHelper' (the configured helper command), '/login managed key' (an API key created and stored by /login with an Anthropic Console account), or 'none' (no API key in use - e.g.
    apiKeySource: ApiKeySource;

    betas?: string[];
    claude_code_version: string;
    cwd: string;
    tools: string[];
    mcp_servers: {
        name: string;
        status: string;
        // Where the server definition came from — same values as McpServerStatus.source (sdk | plugin | a config scope).
        source?: string;
    }[];
    model: string;
    // Permission mode for controlling how tool executions are handled.
    permissionMode: PermissionMode;
    slash_commands: string[];
    // Subset of slash_commands whose UX is bound to the local terminal (e.g.
    terminal_slash_commands?: string[];
    output_style: string;
    skills: string[];
    plugins: {
        name: string;
        path: string;

        // The plugin's version as declared in its plugin.json manifest, emitted verbatim (plugin-author-controlled — validate before trusting).
        version?: string;
    }[];
    // Plugin load-time errors.
    plugin_errors?: {
        plugin: string;
        type: string;
        message: string;
        // Present only when a --plugin-dir, SDK `plugins` or synced directory entry did not load at all: the path of that entry, resolved against the cwd (for a directory, the value its `plugins[]` row would have carried; for a .zip, the archive; a --plugin-url entry carries none).
        path?: string;
        // Present only when a managed-settings `enabledPlugins` entry dropped this --plugin-dir, SDK `plugins`, skills-directory or synced copy: the two halves of that entry, `name` and `marketplace`, and its value.
        managed_lock?: {
            plugin_enabled: boolean;
            name: string;
            marketplace?: string;
        };
    }[];

    fast_mode_state?: FastModeState;
    fast_mode_disabled_reason?: FastModeDisabledReason;

    // The effort level set for the session's next message — after env overrides, session state, org caps and model-support downgrades; the same value get_settings reports as applied.effort.
    effort?: ('low' | 'medium' | 'high' | 'xhigh' | 'max') | null;

    // Whether the session's transcript is in focus view ('focus': the model is told the user sees only its final message per turn, so clients may collapse each turn to the prompt and the final response; 'default' otherwise).
    view_mode?: 'focus' | 'default';
    // Protocol capabilities this CLI supports, so SDK consumers can feature-detect instead of version-sniffing.
    capabilities?: string[];

    uuid: UUID;
    session_id: string;
};

export declare type SDKTaskNotificationMessage = {
    type: 'system';
    subtype: 'task_notification';
    task_id: string;
    // Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
    run_id?: string;
    tool_use_id?: string;
    status: 'completed' | 'failed' | 'stopped';
    // Machine-readable cause, set only when the task did not end through an ordinary completion, failure, or stop.
    reason?: 'worker_restart';
    output_file: string;
    summary: string;
    usage?: {
        total_tokens: number;
        tool_uses: number;
        duration_ms: number;
    };
    // CLI-owned: for a backgrounded MCP task (task_type mcp_task) that completed, the `resource_link` content blocks of its final result — the files it returned by reference — collected from the raw result before the CLI renders it as the text the model reads.
    resource_links?: SDKMcpResourceLink[];

    skip_transcript?: boolean;
    // True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude them from activity indicators.
    ambient?: boolean;
    uuid: UUID;
    session_id: string;
};

export declare type SDKTaskProgressMessage = {
    type: 'system';
    subtype: 'task_progress';
    task_id: string;
    // Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
    run_id?: string;
    tool_use_id?: string;
    description: string;
    // Subagent type for Task tool subagents.
    subagent_type?: string;
    usage: {
        total_tokens: number;
        tool_uses: number;
        duration_ms: number;
    };
    last_tool_name?: string;
    // A one-line status for the task's row.
    summary?: string;

    uuid: UUID;
    session_id: string;
};

export declare type SDKTaskStartedMessage = {
    type: 'system';
    subtype: 'task_started';
    task_id: string;
    // Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
    run_id?: string;
    tool_use_id?: string;
    description: string;
    // Subagent type for Task tool subagents.
    subagent_type?: string;
    // Whether the task was registered in the background (true) or in the foreground with the spawning tool call blocking on it (false).
    is_backgrounded?: boolean;

    // Nesting depth of a spawned subagent (local_agent) task: 1 for a top-level spawn, N+1 when spawned from inside a depth-N agent.
    spawn_depth?: number;
    // task_id of the subagent (local_agent) task whose agent launched this subagent, shell, monitor or workflow task.
    parent_task_id?: string;
    task_type?: string;
    // meta.name from the workflow script (e.g.
    workflow_name?: string;
    prompt?: string;
    // Ambient/housekeeping task.
    skip_transcript?: boolean;
    // True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude them from activity indicators.
    ambient?: boolean;
    uuid: UUID;
    session_id: string;
};

export declare type SDKTaskUpdatedMessage = {
    type: 'system';
    subtype: 'task_updated';
    task_id: string;
    // Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
    run_id?: string;
    // Wire-safe subset of TaskState fields that changed.
    patch: {
        status?: 'pending' | 'running' | 'completed' | 'failed' | 'killed' | 'paused';
        description?: string;
        end_time?: number;
        total_paused_ms?: number;
        error?: string;
        is_backgrounded?: boolean;
    };
    uuid: UUID;
    session_id: string;
};

// Live thinking-token estimate, digested from thinking_delta.estimated_tokens during the redacted-thinking phase (where the API otherwise streams only pings).
export declare type SDKThinkingTokensMessage = {
    type: 'system';
    subtype: 'thinking_tokens';
    estimated_tokens: number;
    estimated_tokens_delta: number;
    // Client uuid of the user message that triggered this turn (submitMessage options.uuid), stamped on every thinking_tokens frame of a headless (stream-json / Agent SDK) turn so a consumer can attribute thinking progress to the send it answers before any reply frame arrives.
    user_message_uuid?: string;
    uuid: UUID;
    session_id: string;
};

export declare type SDKToolProgressMessage = {
    type: 'tool_progress';
    tool_use_id: string;
    tool_name: string;
    parent_tool_use_id: string | null;
    elapsed_time_seconds: number;
    task_id?: string;
    uuid: UUID;
    session_id: string;
    heartbeat?: boolean;
    subagent_type?: string;
    subagent_retry?: {
        agent_id: string;
        attempt: number;
        max_retries: number;
        retry_delay_ms: number;
        error_status: number | null;
        error_category: string;
    };
};

export declare type SDKToolUseSummaryMessage = {
    type: 'tool_use_summary';
    summary: string;
    preceding_tool_use_ids: string[];
    uuid: UUID;
    session_id: string;

};

// Structured twin of a /usage result, carried beside its text: the session's totals, the plan's usage rows as the server sent them and the extra-usage spend, and nothing else from the usage body (the get_usage control reply carries the rest).
export declare type SDKUsageReport = {
    // Cost and usage accumulated by the current session.
    session: {
        total_cost_usd: number;
        total_api_duration_ms: number;
        total_duration_ms: number;
        total_lines_added: number;
        total_lines_removed: number;
        model_usage: Record<string, ModelUsage>;
    };
    // The plan's usage rows and extra-usage spend from the claude.ai usage endpoint; null when the CLI could not fetch them (no plan on this lane, or a token without the profile scope).
    rate_limits: {
        // The server's usage rows (the usage endpoint's limits[]), as sent: which meters apply, their scope, labels, severity and order are the server's, so a client renders them verbatim and a new meter needs no client release.
        limits: {
            // The server's meter kind, e.g.
            kind: string;
            // The server's row group, e.g.
            group: string;
            // Share of the window used, 0-100.
            percent: number;
            // ISO 8601 timestamp when the window resets.
            resets_at: string | null;
            // What a scoped row is for, a model or a surface, with the server's display label.
            scope?: {
                model?: {
                    display_name: string;
                } | null;
                surface?: {
                    display_name: string;
                } | null;
            } | null;
            // The server's reading of the row for a meter's colour, e.g.
            severity: string;
            // The server's headline pick: the row a single-value indicator shows.
            is_active: boolean;
        }[] | null;
        // Extra-usage (overage) spend for the billing period, when the plan has it.
        extra_usage?: {
            is_enabled: boolean;
            monthly_limit: number | null;
            used_credits: number | null;
            utilization: number | null;
            currency?: string | null;
        } | null;
    } | null;
};

// A user-role message.
export declare type SDKUserMessage = {
    type: 'user';
    // An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image, document, tool_result, ...).
    message: MessageParam;
    parent_tool_use_id: string | null;
    isSynthetic?: boolean;
    // Structured tool output — the tool's full Output object, not the string content sent to the model.
    tool_use_result?: unknown;
    priority?: 'now' | 'next' | 'later';

    origin?: SDKMessageOrigin;

    // When false, the message is appended to the transcript without triggering an assistant turn.
    shouldQuery?: boolean;
    // ISO timestamp when the message was created on the originating process.
    timestamp?: string;

    // The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and no slash-command dispatch.
    client_composed?: true;

    uuid?: UUID;
    // Content the user pasted into the prompt rather than typed: each entry a string or an array of content blocks.
    pasted_content?: MessageParam['content'][];
    // Text the user pasted that is still in `message.content` where they put it: each entry one paste.
    inline_pastes?: string[];

    session_id?: string;
    // ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed.
    agent_id?: string;
    // Subagent type that produced this message.
    subagent_type?: string;
    // Description of the subagent task that produced this message.
    task_description?: string;

};

export declare type SDKUserMessageReplay = {
    type: 'user';
    // An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image, document, tool_result, ...).
    message: MessageParam;
    parent_tool_use_id: string | null;
    isSynthetic?: boolean;
    // Structured tool output — the tool's full Output object, not the string content sent to the model.
    tool_use_result?: unknown;
    priority?: 'now' | 'next' | 'later';

    origin?: SDKMessageOrigin;

    // When false, the message is appended to the transcript without triggering an assistant turn.
    shouldQuery?: boolean;
    // ISO timestamp when the message was created on the originating process.
    timestamp?: string;

    // The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and no slash-command dispatch.
    client_composed?: true;

    uuid: UUID;
    session_id: string;
    isReplay: true;
    file_attachments?: unknown[];
};

// Emitted by the bridge on opt-in graceful worker teardown (only when the teardown caller supplied a reason), before the heartbeat stops, so remote clients can show why the worker went away instead of waiting for heartbeat timeout.
export declare type SDKWorkerShuttingDownMessage = {
    type: 'system';
    subtype: 'worker_shutting_down';
    // Short snake_case reason set by the host CLI (not user input), e.g.
    reason: string;
    uuid: UUID;
    session_id: string;
};

export declare type SessionCronSummary = {
    id: string;
    // Cron expression, e.g.
    schedule: string;
    // False for one-shot wakeups whose cron field encodes a single fire time; true for tasks that re-fire on every match.
    recurring: boolean;
    // Prompt text submitted when the cron fires.
    prompt: string;
};

export declare type SessionEndHookInput = BaseHookInput & {
    hook_event_name: 'SessionEnd';
    reason: ExitReason;
};

// Identifies a session transcript or subagent transcript in the store. [@alpha]
export declare type SessionKey = {
    // Caller-defined scope.
    projectKey: string;
    sessionId: string;
    // Undefined = main transcript.
    subpath?: string;
};

// A message from a session transcript.
export declare type SessionMessage = {
    type: 'user' | 'assistant' | 'system';
    uuid: string;
    session_id: string;
    message: unknown;
    parent_tool_use_id: string | null;
    // agentId of the subagent that spawned this subagent, or null when this message belongs to a depth-1 subagent (spawned by the main loop) or to the main session itself.
    parent_agent_id: string | null;
};

// Options shared by session mutation functions (renameSession, tagSession, archiveSession, unarchiveSession, deleteSession, forkSession).
export declare type SessionMutationOptions = {
    // Project directory path (same semantics as `listSessions({ dir })`).
    dir?: string;
    // When provided, read/write session data via this store instead of the local filesystem. [@alpha]
    sessionStore?: SessionStore;
};

export declare type SessionStartHookInput = BaseHookInput & {
    hook_event_name: 'SessionStart';
    source: 'startup' | 'resume' | 'clear' | 'compact' | 'fork';
    agent_type?: string;
    model?: string;
    session_title?: string;
    // resume/fork: seconds since the resumed transcript's last assistant response
    seconds_since_last_response?: number;
    // resume/fork: the resumed transcript's last response input + cache_read + cache_creation + output tokens (for a server-side tool loop, its last iteration's window, not the summed totals)
    context_tokens?: number;
    // resume/fork: seconds_since_last_response exceeds the prompt-cache TTL, so the first request re-caches context_tokens
    prompt_cache_likely_expired?: boolean;
    // resume/fork: estimated cost of re-caching context_tokens on the session model — the managed modelPricing when set, otherwise list price; excludes the response
    estimated_cache_write_usd?: number;
};

export declare type SessionStartHookSpecificOutput = {
    hookEventName: 'SessionStart';
    additionalContext?: string;
    initialUserMessage?: string;
    sessionTitle?: string;
    watchPaths?: string[];
    // Re-scan skill and command directories after SessionStart hooks complete, so skills installed by the hook are available in the same session
    reloadSkills?: boolean;
};

// Adapter for mirroring session transcripts to external storage. [@alpha]
export declare type SessionStore = {
    // Mirror a batch of transcript entries.
    append(key: SessionKey, entries: SessionStoreEntry[]): Promise<void>;
    // Load a full session for resume.
    load(key: SessionKey): Promise<SessionStoreEntry[] | null>;
    // List sessions for a projectKey.
    listSessions?(projectKey: string): Promise<Array<{
        sessionId: string;
        mtime: number;
    }>>;
    // Return incrementally-maintained summaries for all sessions in one call. [@alpha]
    listSessionSummaries?(projectKey: string): Promise<SessionSummaryEntry[]>;
    // Delete a session.
    delete?(key: SessionKey): Promise<void>;
    // List all subpath keys under a session (e.g., subagent transcripts).
    listSubkeys?(key: {
        projectKey: string;
        sessionId: string;
    }): Promise<string[]>;
};

// One JSONL transcript line as observed by a {@link SessionStore} adapter. [@alpha]
export declare type SessionStoreEntry = {
    type: string;
    uuid?: string;
    timestamp?: string;
    [k: string]: unknown;
};

// Flush strategy for {@link Options.sessionStore} transcript mirroring. [@alpha]
export declare type SessionStoreFlush = 'batched' | 'eager';

// Incrementally-maintained session summary. [@alpha]
export declare type SessionSummaryEntry = {
    sessionId: string;
    // Storage write time of the sidecar on the adapter.
    mtime: number;
    // Opaque SDK-owned state.
    data: Record<string, unknown>;
};

// AUTO-GENERATED - DO NOT EDIT This file is auto-generated from the settings JSON schema.
export declare interface Settings {
    // JSON Schema reference for Claude Code settings
    $schema?: string;
    // Path to a script that outputs authentication values
    apiKeyHelper?: string;
    // Shell command that outputs a Proxy-Authorization header value (EAP)
    proxyAuthHelper?: string;
    // Path to a script that exports AWS credentials
    awsCredentialExport?: string;
    // Path to a script that refreshes AWS authentication
    awsAuthRefresh?: string;
    // Command to refresh GCP authentication (e.g., gcloud auth application-default login)
    gcpAuthRefresh?: string;
    // Corporate launcher argv prefix for the background-agent supervisor, the sessions and workers it hosts, and the other covered background processes listed in the Claude Code corporate-launcher documentation.
    processWrapper?: string;
    // Executable that computes managed settings at startup.
    policyHelper?: {
        // Absolute path to the helper executable
        path: string;
        timeoutMs?: number;
        refreshIntervalMs?: 0 | number;
    };

    // Custom file suggestion configuration for \@ mentions
    fileSuggestion?: {
        type: 'command';
        command: string;
    };
    // Whether file picker should respect .gitignore files (default: true).
    respectGitignore?: boolean;

    // Number of days to retain chat transcripts before automatic cleanup (default: 30).
    cleanupPeriodDays?: number;
    // Retention ceiling in days for session transcripts created or last written by a desktop-host surface (Claude Desktop, Cowork), which are otherwise exempt from the cleanupPeriodDays sweep.
    desktopSessionCleanupPeriodDays?: number;
    // Set to false to turn off syncing of the skills you have enabled on claude.ai.
    syncClaudeAiSkills?: boolean;
    // Set to false to turn off syncing of the plugins you have enabled on claude.ai.
    syncClaudeAiPlugins?: boolean;
    // Per-skill description character cap in the skill listing sent to Claude (default: 1536).
    skillListingMaxDescChars?: number;
    // Fraction of the context window (in characters) reserved for the skill listing sent to Claude (default: 0.01 = 1%).
    skillListingBudgetFraction?: number;
    // When set to true in either admin-only Windows source — the HKLM SOFTWARE/Policies/ClaudeCode registry key or C:/Program Files/ClaudeCode/managed-settings.json — WSL reads managed settings from the full Windows policy chain (HKLM, C:/Program Files/ClaudeCode via DrvFs, HKCU) in addition to /etc/claude-code.
    wslInheritsWindowsSettings?: boolean;
    // Environment variables to set for Claude Code sessions
    env?: {
        [k: string]: string;
    };
    // Customize attribution text for commits and PRs.
    attribution?: boolean | {
        // Attribution text for git commits, including any trailers.
        commit?: string;
        // Attribution text for pull request descriptions.
        pr?: string;
        // Whether to append the claude.ai session link to commits and PRs created from web or Remote Control sessions (default: true).
        sessionUrl?: boolean;
        [k: string]: unknown;
    };
    // The claude.ai Project this session names for your organization's usage reports: the Project's ID, starting with "claude_proj_" or "chan_", or the UUID of a "claude_proj_" Project.
    attributionProject?: string;
    // Your organization's default for attributionProject.
    attributionProjectPolicy?: {
        // The Project used when neither CLAUDE_CODE_ATTRIBUTION_PROJECT nor attributionProject names one: its ID, starting with "claude_proj_" or "chan_", or the UUID of a "claude_proj_" Project.
        default?: string;
    };
    // Deprecated: Use attribution instead.
    includeCoAuthoredBy?: boolean;
    // Include built-in commit and PR workflow instructions in Claude's system prompt (default: true)
    includeGitInstructions?: boolean;
    // Tool usage permissions configuration
    permissions?: {
        // List of permission rules for allowed operations
        allow?: string[];
        // List of permission rules for denied operations
        deny?: string[];
        // List of permission rules that should always prompt for confirmation
        ask?: string[];
        // Default permission mode when Claude Code needs access ('manual' is accepted as an alias for 'default')
        defaultMode?: 'acceptEdits' | 'auto' | 'bypassPermissions' | 'default' | 'dontAsk' | 'plan';
        // Disable the ability to bypass permission prompts
        disableBypassPermissionsMode?: 'disable';
        // Refuse file-tool reads (Read, Grep, Glob, LSP) outside the working directories in every permission mode; true in any settings source wins.
        blockReadsOutsideWorkingDirectories?: boolean;

        // Refuse file-tool writes (Write, Edit, NotebookEdit) outside the working directories in every permission mode; true in any settings source wins.
        blockWritesOutsideWorkingDirectories?: boolean;
        // For the program that starts Claude Code: read only from --settings and from settings that program sends while the session runs; ignored in user, project, local and managed settings.
        blockReadsExemptPaths?: string[];
        // For the program that starts Claude Code: read only from --settings and from settings that program sends while the session runs; ignored in user, project, local and managed settings.
        workingDirectoriesAsGiven?: string[];
        // Disable auto mode
        disableAutoMode?: 'disable';
        // Never wait for a person to answer a tool permission prompt: a call that would prompt is denied at once, the model is told, and the turn carries on.
        denyInsteadOfPrompting?: boolean;
        // Additional directories to include in the permission scope
        additionalDirectories?: string[];
        [k: string]: unknown;
    };
    // Override the default model used by Claude Code
    model?: string;
    // Fallback model(s) tried in order when the primary model is overloaded or unavailable.
    fallbackModel?: string[];
    // Allowlist of models that users can select.
    availableModels?: string[];
    // When true and availableModels is a non-empty array, the Default model selection is also constrained: if the default model for the user tier is not in availableModels, Default resolves to the first allowed availableModels entry instead.
    enforceAvailableModels?: boolean;
    // How availableModels entries match model IDs.
    availableModelsMatch?: 'prefix' | 'exact';
    // Models users cannot select, even when availableModels allows them.
    deniedModels?: string[];
    // Override mapping from Anthropic model ID (e.g.
    modelOverrides?: {
        [k: string]: string;
    };
    // Curate the /model picker: an ordered list of models with your own labels, independent of the built-in lineup and of Claude Code releases.
    modelPicker?: {
        // Rows to show in the /model picker, in order.
        options: {
            // Model to select, taken verbatim: an alias ("opus"), an Anthropic model ID, or a provider-format ID (Vertex, Bedrock, gateway).
            model: string;
            // Row title.
            label?: string;
            // Row subtitle.
            description?: string;
            // For a model this version of Claude Code does not know: the ID of a model it does know (e.g.
            behavesAs?: string;
        }[];
        // When true, the picker shows only the Default row and these options — the built-in lineup, gateway-discovered models and ANTHROPIC_CUSTOM_MODEL_OPTION are hidden.
        replaceBuiltInOptions?: boolean;
    };
    // Price usage at your organization's contracted rates instead of list price.
    modelPricing?: {
        multiplier?: number;
        overrides?: {
            [k: string]: {
                input: number;
                output: number;
                cacheRead: number;
                cacheWrite: number;
            };
        };
    };
    // Whether to automatically approve all MCP servers in the project
    enableAllProjectMcpServers?: boolean;
    // List of approved MCP servers from .mcp.json
    enabledMcpjsonServers?: string[];
    // List of rejected MCP servers from .mcp.json
    disabledMcpjsonServers?: string[];
    // When true in any settings source, claude.ai MCP cloud connectors are not auto-fetched or connected, and a claudeai-proxy server passed explicitly (e.g.
    disableClaudeAiConnectors?: boolean;
    // Per-skill listing overrides keyed by skill name.
    skillOverrides?: {
        [k: string]: 'on' | 'name-only' | 'user-invocable-only' | 'off';
    };
    // Disable the skills and workflows that ship with Claude Code: bundled skills and workflows are removed entirely; built-in slash commands stay typable but are hidden from the model.
    disableBundledSkills?: boolean;
    // MCP servers the organization provides to every user, keyed by server name, each with the .mcp.json entry shape; only "http" and "sse" servers are accepted (nothing that names a program to run, no ${VAR} references).
    managedMcpServers?: {
        [k: string]: {
            [k: string]: unknown;
        };
    };
    // Enterprise allowlist of the MCP servers users may use.
    allowedMcpServers?: {
        // Name of the MCP server that users are allowed to configure
        serverName?: string;
        // Command array [command, ...args] to match exactly for allowed stdio servers \@minItems 1
        serverCommand?: [string, ...string[]];
        // URL pattern with wildcard support (e.g., "https://*.example.com/*") for allowed remote MCP servers
        serverUrl?: string;
    }[];
    // Enterprise denylist of MCP servers that are explicitly blocked.
    deniedMcpServers?: {
        // Name of the MCP server that is explicitly blocked
        serverName?: string;
        // Command array [command, ...args] to match exactly for blocked stdio servers \@minItems 1
        serverCommand?: [string, ...string[]];
        // URL pattern with wildcard support (e.g., "https://*.example.com/*") for blocked remote MCP servers
        serverUrl?: string;
    }[];
    // Custom commands to run before/after tool executions
    hooks?: {
        [k: string]: {
            // String pattern to match (e.g.
            matcher?: string;
            // List of hooks to execute when the matcher matches
            hooks: ({
                // Shell command hook type
                type: 'command';
                // Shell command to execute
                command: string;
                // Argument list for exec form.
                args?: string[];
                // Permission rule syntax to filter when this hook runs (e.g., "Bash(git *)").
                if?: string;
                // Shell interpreter.
                shell?: 'bash' | 'powershell';
                // Timeout in seconds for this specific command
                timeout?: number;
                // What a failure of this hook does: it could not start (a missing script or plugin directory), timed out, exited with a code other than 0 or 2, or printed JSON that is invalid or fails validation.
                onFailure?: 'continue' | 'block';
                // Custom status message to display in spinner while hook runs
                statusMessage?: string;
                // If true, hook runs once and is removed after execution
                once?: boolean;
                // If true, hook runs in background without blocking
                async?: boolean;
                // If true, hook runs in background and wakes the model on exit code 2 (blocking error).
                asyncRewake?: boolean;

            } | {
                // LLM prompt hook type
                type: 'prompt';
                // Prompt to evaluate with LLM.
                prompt: string;
                // Permission rule syntax to filter when this hook runs (e.g., "Bash(git *)").
                if?: string;
                // Timeout in seconds for this specific prompt evaluation
                timeout?: number;
                // Model to use for this prompt hook (e.g., "claude-sonnet-5").
                model?: string;
                // Sets the continue value for the decision:"block" produced when ok is false.
                continueOnBlock?: boolean;
                // Custom status message to display in spinner while hook runs
                statusMessage?: string;
                // If true, hook runs once and is removed after execution
                once?: boolean;
            } | {
                // Agentic verifier hook type
                type: 'agent';
                // Prompt describing what to verify (e.g.
                prompt: string;
                // Permission rule syntax to filter when this hook runs (e.g., "Bash(git *)").
                if?: string;
                // Timeout in seconds for agent execution (default 60)
                timeout?: number;
                // Model to use for this agent hook (e.g., "claude-sonnet-5").
                model?: string;
                // Custom status message to display in spinner while hook runs
                statusMessage?: string;
                // If true, hook runs once and is removed after execution
                once?: boolean;
            } | {
                // HTTP hook type
                type: 'http';
                // URL to POST the hook input JSON to
                url: string;
                // Permission rule syntax to filter when this hook runs (e.g., "Bash(git *)").
                if?: string;
                // Timeout in seconds for this specific request
                timeout?: number;
                // What a failure of this hook does: it could not start (a missing script or plugin directory), timed out, exited with a code other than 0 or 2, or printed JSON that is invalid or fails validation.
                onFailure?: 'continue' | 'block';
                // Additional headers to include in the request.
                headers?: {
                    [k: string]: string;
                };
                // Explicit list of environment variable names that may be interpolated in header values.
                allowedEnvVars?: string[];
                // Custom status message to display in spinner while hook runs
                statusMessage?: string;
                // If true, hook runs once and is removed after execution
                once?: boolean;

            } | {
                // MCP tool hook type
                type: 'mcp_tool';
                // Name of an already-configured MCP server to invoke
                server: string;
                // Name of the tool on that server to call
                tool: string;
                // Arguments passed to the MCP tool.
                input?: {
                    [k: string]: unknown;
                };
                // Permission rule syntax to filter when this hook runs (e.g., "Bash(git *)").
                if?: string;
                // Timeout in seconds for this specific tool call
                timeout?: number;
                // Custom status message to display in spinner while hook runs
                statusMessage?: string;
                // If true, hook runs once and is removed after execution
                once?: boolean;
            })[];
        }[];
    };
    // Git worktree configuration: the CLI --worktree flag, EnterWorktree and agent isolation, plus the location Claude Code Desktop uses for SSH-session worktrees on this machine.
    worktree?: {
        // Directories to symlink from main repository to worktrees to avoid disk bloat.
        symlinkDirectories?: string[];
        // Directories to include when creating worktrees, via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Which ref new worktrees branch from.
        baseRef?: 'fresh' | 'head';
        // Isolation mode for background sessions in this repo.
        bgIsolation?: 'worktree' | 'none';
        // Directory under which Claude Code Desktop creates the worktrees of SSH sessions that run on this machine (an absolute path or one starting with ~/), instead of <project>/.claude/worktrees.
        location?: string;
    };
    // Disable all hooks and statusLine execution: the hooks defined in settings files and by installed plugins.
    disableAllHooks?: boolean;
    // Disable the reply check in Claude Tag sessions on a self-hosted runner.
    disableReplyCheck?: boolean;
    // Disable agent view (`claude agents`, `--bg`, /background, the on-demand daemon).
    disableAgentView?: boolean;
    // Disable Remote Control (claude.ai/code, `claude remote-control`, `--remote-control`/`--rc`, auto-start, and the in-session toggle).
    disableRemoteControl?: boolean;
    // Disable the Workflows feature.
    disableWorkflows?: boolean;
    // Deprecated: use enableArtifact: false.
    disableArtifact?: boolean;
    // Turn the Artifact tool on or off.
    enableArtifact?: boolean;
    // Enable or disable the Workflows feature for this user.
    enableWorkflows?: boolean;
    // Advisory size guideline for the dynamic workflows Claude writes: "small" aims for fewer than 5 agents, "medium" fewer than 10, "large" fewer than 50, and "unrestricted" sends no guideline.
    workflowSizeGuideline?: 'unrestricted' | 'small' | 'medium' | 'large';
    // Enable the "ultracode" keyword trigger: including the keyword in a prompt opts that turn into the Workflow tool.
    workflowKeywordTriggerEnabled?: boolean;
    // Disable inline shell execution in skills and custom slash commands from user, project, or plugin sources.
    disableSkillShellExecution?: boolean;
    // Windows only.
    disableWindowsBashCommandSplit?: boolean;
    // Default shell for input-box !
    defaultShell?: 'bash' | 'powershell';
    // Whether the Bash tool shows a diff of the files a Bash command changed (PostToolUse Bash hooks get the changed-file list in tool_response).
    bashEditDiffEnabled?: boolean;
    // How many characters of a successful Bash or PowerShell command's output Claude receives inline (default 30000; values clamp to 4000-128000).
    bashOutputMaxChars?: number;
    // Deprecated: no longer has any effect (the TaskOutput tool was removed).
    taskOutputMaxChars?: number;
    // Whether Claude responds after an input-box !
    respondToBashCommands?: boolean;
    // When true (and set in managed settings), only hooks from managed settings and from plugins that managed settings enable run.
    allowManagedHooksOnly?: boolean;
    // Allowlist of URL patterns that HTTP hooks may target.
    allowedHttpHookUrls?: string[];
    // Allowlist of environment variable names HTTP hooks may interpolate into headers.
    httpHookAllowedEnvVars?: string[];
    // When true (and set in managed settings), permission rules from user, project, local, and --settings files and allow rules from --allowedTools are ignored; only managed settings can add allow rules through settings.
    allowManagedPermissionRulesOnly?: boolean;
    // When true (and set in managed settings), allowedMcpServers is only read from managed settings.
    allowManagedMcpServersOnly?: boolean;
    // When true (and set in managed settings), claude.ai cloud MCP connectors load alongside managed-mcp.json instead of being suppressed by its exclusive-control lockdown.
    allowAllClaudeAiMcps?: boolean;
    // When true (and set in device managed settings: MDM, the managed-settings.json file, or a policy helper those configure), the built-in Claude in Chrome MCP server can run alongside managed-mcp.json instead of being blocked by its exclusive-control lockdown.
    allowClaudeInChromeWithManagedMcp?: boolean;
    // When set in managed settings, blocks non-plugin customization sources for the listed surfaces.
    strictPluginOnlyCustomization?: boolean | ('skills' | 'agents' | 'hooks' | 'mcp')[];
    // Custom status line display configuration
    statusLine?: {
        type: 'command';
        command: string;
        padding?: number;
        // Re-run the status line command every N seconds in addition to event-driven updates
        refreshInterval?: number;
        // Hide the built-in `-- INSERT --` / `-- VISUAL --` indicator below the prompt.
        hideVimModeIndicator?: boolean;
    };
    // URL template for PR links in the footer link badges and inline messages.
    prUrlTemplate?: string;
    // Extra clickable footer badges that appear when a regex matches turn output (tool results and assistant responses).
    footerLinksRegexes?: ({
        // Config variant.
        type: 'regex';
        // Regex matched against turn output (tool results and assistant text)
        pattern: string;
        // Link target.
        url: string;
        // Badge text.
        label?: string;
        [k: string]: unknown;
    } | {
        // Config variant discriminator for entries this client does not understand; the entry is preserved as-is and skipped at runtime.
        type: string;
        [k: string]: unknown;
    })[];
    // Deprecated: no longer has any effect.
    subagentStatusLine?: {
        type: 'command';
        command: string;
    };
    // Enabled plugins using plugin-id\@marketplace-id format.
    enabledPlugins?: {
        [k: string]: string[] | boolean | {
            [k: string]: unknown;
        };
    };
    // Managed plugins (plugin\@marketplace ids that managed enabledPlugins sets true) whose hooks run first, outermost, in the listed order: the first id listed sees every event before any other plugin and every result after it.
    prependPlugins?: string[];
    // Managed plugins (plugin\@marketplace ids that managed enabledPlugins sets true) whose hooks run last among plugins, innermost, in the listed order: the last id listed sits just above the built-in plugins and sees each event as every other plugin left it.
    appendPlugins?: string[];
    // Additional marketplaces to make available for this repository.
    extraKnownMarketplaces?: {
        [k: string]: {
            // Where to fetch the marketplace from
            source: {
                source: 'url';
                // Direct URL to marketplace.json file
                url: string;
                // Custom HTTP headers (e.g., for authentication)
                headers?: {
                    [k: string]: string;
                };
                // Command that prints a JSON object of HTTP headers (e.g.
                headersHelper?: string;
            } | {
                source: 'github';
                // GitHub repository in owner/repo format.
                repo: string;
                // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
                path?: string;
                // Directories to include via git sparse-checkout (cone mode).
                sparsePaths?: string[];
                // Has no effect; accepted so existing settings keep working.
                skipLfs?: boolean;
            } | {
                source: 'git';
                // Full git repository URL
                url: string;
                // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
                path?: string;
                // Directories to include via git sparse-checkout (cone mode).
                sparsePaths?: string[];
                // Has no effect; accepted so existing settings keep working.
                skipLfs?: boolean;
            } | {
                source: 'npm';
                // npm package containing marketplace.json (e.g.
                package: string;
                // Version or range to fetch (e.g.
                version?: string;
                // Registry URL.
                registry?: string;
            } | {
                source: 'file';
                // Local file path to marketplace.json
                path: string;
            } | {
                source: 'directory';
                // Local directory containing .claude-plugin/marketplace.json
                path: string;
            } | {
                source: 'skills-dir';
            } | {
                source: 'hostPattern';
                // Regex pattern to match the host/domain extracted from any marketplace source type.
                hostPattern: string;
            } | {
                source: 'pathPattern';
                // Regex pattern matched against the .path field of file and directory sources.
                pathPattern: string;
            } | {
                source: 'settings';
                // Marketplace name.
                name: string;
                // Plugin entries declared inline in settings.json
                plugins: {
                    // Plugin name as it appears in the target repository
                    name: string;
                    // Where to fetch the plugin from.
                    source: string | {
                        source: 'npm';
                        // Package name (or url, or local path, or anything else that can be passed to `npm` as a package)
                        package: string;
                        // Specific version or version range (e.g., ^1.0.0, ~2.1.0)
                        version?: string;
                        // Custom NPM registry URL (defaults to using system default, likely npmjs.org)
                        registry?: string;
                    } | {
                        source: 'url';
                        // Full git repository URL (https:// or git\@)
                        url: string;
                        // Git branch or tag to use (e.g., "main", "v1.0.0").
                        ref?: string;
                        // Specific commit SHA to use
                        sha?: string;
                    } | {
                        source: 'github';
                        // GitHub repository in owner/repo format
                        repo: string;
                        // Git branch or tag to use (e.g., "main", "v1.0.0").
                        ref?: string;
                        // Specific commit SHA to use
                        sha?: string;
                    } | {
                        source: 'git-subdir';
                        // Git repository: GitHub owner/repo shorthand, https://, or git\@ URL
                        url: string;
                        // Subdirectory within the repo containing the plugin (e.g., "tools/claude-plugin").
                        path: string;
                        // Git branch or tag to use (e.g., "main", "v1.0.0").
                        ref?: string;
                        // Specific commit SHA to use
                        sha?: string;
                    } | {
                        source: 'archive';
                        // HTTPS URL of a zip archive containing the plugin.
                        url: string;
                        // SHA-256 digest of the archive.
                        sha256?: string;
                    } | {
                        source: 'command';
                        // Shell command that prints the absolute path of the plugin directory on stdout (exactly one line) and exits 0.
                        command: string;
                        // Seconds to wait for the command before giving up (default: 60)
                        timeout?: number;
                        // copy (default): the printed directory is copied into the plugin cache and content-hashed, so it may be deleted afterwards.
                        mode?: 'copy' | 'link';
                    } | {
                        source: 'unsupported';
                        error?: string;
                    };
                    description?: string;
                    version?: string;
                    strict?: boolean;
                    // Versions of its hosts this plugin runs on.
                    engines?: {
                        [k: string]: unknown;
                    };
                    // HTTP headers sent when downloading this entry's `archive` source.
                    headers?: {
                        [k: string]: string;
                    };
                    // Command that prints a JSON object of HTTP headers for downloading this entry's `archive` source.
                    headersHelper?: string;
                }[];
                owner?: {
                    // Display name of the plugin author or organization
                    name: string;
                    // Contact email for support or feedback
                    email?: string;
                    // Website, GitHub profile, or organization URL
                    url?: string;
                };
            };
            // Local cache path where marketplace manifest is stored (auto-generated if not provided)
            installLocation?: string;
            // Whether to automatically update this marketplace and its installed plugins on startup
            autoUpdate?: boolean;
            // Deprecated and ignored: settings that hold it still load.
            pluginDigests?: {
                [k: string]: string[];
            };
        };
    };
    // Alias for extraKnownMarketplaces: this key is read exactly as if it were spelled extraKnownMarketplaces.
    additionalMarketplaces?: {
        [k: string]: {
            // Where to fetch the marketplace from
            source: {
                source: 'url';
                // Direct URL to marketplace.json file
                url: string;
                // Custom HTTP headers (e.g., for authentication)
                headers?: {
                    [k: string]: string;
                };
                // Command that prints a JSON object of HTTP headers (e.g.
                headersHelper?: string;
            } | {
                source: 'github';
                // GitHub repository in owner/repo format.
                repo: string;
                // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
                path?: string;
                // Directories to include via git sparse-checkout (cone mode).
                sparsePaths?: string[];
                // Has no effect; accepted so existing settings keep working.
                skipLfs?: boolean;
            } | {
                source: 'git';
                // Full git repository URL
                url: string;
                // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
                path?: string;
                // Directories to include via git sparse-checkout (cone mode).
                sparsePaths?: string[];
                // Has no effect; accepted so existing settings keep working.
                skipLfs?: boolean;
            } | {
                source: 'npm';
                // npm package containing marketplace.json (e.g.
                package: string;
                // Version or range to fetch (e.g.
                version?: string;
                // Registry URL.
                registry?: string;
            } | {
                source: 'file';
                // Local file path to marketplace.json
                path: string;
            } | {
                source: 'directory';
                // Local directory containing .claude-plugin/marketplace.json
                path: string;
            } | {
                source: 'skills-dir';
            } | {
                source: 'hostPattern';
                // Regex pattern to match the host/domain extracted from any marketplace source type.
                hostPattern: string;
            } | {
                source: 'pathPattern';
                // Regex pattern matched against the .path field of file and directory sources.
                pathPattern: string;
            } | {
                source: 'settings';
                // Marketplace name.
                name: string;
                // Plugin entries declared inline in settings.json
                plugins: {
                    // Plugin name as it appears in the target repository
                    name: string;
                    // Where to fetch the plugin from.
                    source: string | {
                        source: 'npm';
                        // Package name (or url, or local path, or anything else that can be passed to `npm` as a package)
                        package: string;
                        // Specific version or version range (e.g., ^1.0.0, ~2.1.0)
                        version?: string;
                        // Custom NPM registry URL (defaults to using system default, likely npmjs.org)
                        registry?: string;
                    } | {
                        source: 'url';
                        // Full git repository URL (https:// or git\@)
                        url: string;
                        // Git branch or tag to use (e.g., "main", "v1.0.0").
                        ref?: string;
                        // Specific commit SHA to use
                        sha?: string;
                    } | {
                        source: 'github';
                        // GitHub repository in owner/repo format
                        repo: string;
                        // Git branch or tag to use (e.g., "main", "v1.0.0").
                        ref?: string;
                        // Specific commit SHA to use
                        sha?: string;
                    } | {
                        source: 'git-subdir';
                        // Git repository: GitHub owner/repo shorthand, https://, or git\@ URL
                        url: string;
                        // Subdirectory within the repo containing the plugin (e.g., "tools/claude-plugin").
                        path: string;
                        // Git branch or tag to use (e.g., "main", "v1.0.0").
                        ref?: string;
                        // Specific commit SHA to use
                        sha?: string;
                    } | {
                        source: 'archive';
                        // HTTPS URL of a zip archive containing the plugin.
                        url: string;
                        // SHA-256 digest of the archive.
                        sha256?: string;
                    } | {
                        source: 'command';
                        // Shell command that prints the absolute path of the plugin directory on stdout (exactly one line) and exits 0.
                        command: string;
                        // Seconds to wait for the command before giving up (default: 60)
                        timeout?: number;
                        // copy (default): the printed directory is copied into the plugin cache and content-hashed, so it may be deleted afterwards.
                        mode?: 'copy' | 'link';
                    } | {
                        source: 'unsupported';
                        error?: string;
                    };
                    description?: string;
                    version?: string;
                    strict?: boolean;
                    // Versions of its hosts this plugin runs on.
                    engines?: {
                        [k: string]: unknown;
                    };
                    // HTTP headers sent when downloading this entry's `archive` source.
                    headers?: {
                        [k: string]: string;
                    };
                    // Command that prints a JSON object of HTTP headers for downloading this entry's `archive` source.
                    headersHelper?: string;
                }[];
                owner?: {
                    // Display name of the plugin author or organization
                    name: string;
                    // Contact email for support or feedback
                    email?: string;
                    // Website, GitHub profile, or organization URL
                    url?: string;
                };
            };
            // Local cache path where marketplace manifest is stored (auto-generated if not provided)
            installLocation?: string;
            // Whether to automatically update this marketplace and its installed plugins on startup
            autoUpdate?: boolean;
            // Deprecated and ignored: settings that hold it still load.
            pluginDigests?: {
                [k: string]: string[];
            };
        };
    };
    // Enterprise strict list of allowed marketplace sources.
    strictKnownMarketplaces?: ({
        source: 'url';
        // Direct URL to marketplace.json file
        url: string;
        // Custom HTTP headers (e.g., for authentication)
        headers?: {
            [k: string]: string;
        };
        // Command that prints a JSON object of HTTP headers (e.g.
        headersHelper?: string;
    } | {
        source: 'github';
        // GitHub repository in owner/repo format.
        repo: string;
        // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
        ref?: string;
        // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
        path?: string;
        // Directories to include via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Has no effect; accepted so existing settings keep working.
        skipLfs?: boolean;
    } | {
        source: 'git';
        // Full git repository URL
        url: string;
        // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
        ref?: string;
        // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
        path?: string;
        // Directories to include via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Has no effect; accepted so existing settings keep working.
        skipLfs?: boolean;
    } | {
        source: 'npm';
        // npm package containing marketplace.json (e.g.
        package: string;
        // Version or range to fetch (e.g.
        version?: string;
        // Registry URL.
        registry?: string;
    } | {
        source: 'file';
        // Local file path to marketplace.json
        path: string;
    } | {
        source: 'directory';
        // Local directory containing .claude-plugin/marketplace.json
        path: string;
    } | {
        source: 'skills-dir';
    } | {
        source: 'hostPattern';
        // Regex pattern to match the host/domain extracted from any marketplace source type.
        hostPattern: string;
    } | {
        source: 'pathPattern';
        // Regex pattern matched against the .path field of file and directory sources.
        pathPattern: string;
    } | {
        source: 'settings';
        // Marketplace name.
        name: string;
        // Plugin entries declared inline in settings.json
        plugins: {
            // Plugin name as it appears in the target repository
            name: string;
            // Where to fetch the plugin from.
            source: string | {
                source: 'npm';
                // Package name (or url, or local path, or anything else that can be passed to `npm` as a package)
                package: string;
                // Specific version or version range (e.g., ^1.0.0, ~2.1.0)
                version?: string;
                // Custom NPM registry URL (defaults to using system default, likely npmjs.org)
                registry?: string;
            } | {
                source: 'url';
                // Full git repository URL (https:// or git\@)
                url: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'github';
                // GitHub repository in owner/repo format
                repo: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'git-subdir';
                // Git repository: GitHub owner/repo shorthand, https://, or git\@ URL
                url: string;
                // Subdirectory within the repo containing the plugin (e.g., "tools/claude-plugin").
                path: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'archive';
                // HTTPS URL of a zip archive containing the plugin.
                url: string;
                // SHA-256 digest of the archive.
                sha256?: string;
            } | {
                source: 'command';
                // Shell command that prints the absolute path of the plugin directory on stdout (exactly one line) and exits 0.
                command: string;
                // Seconds to wait for the command before giving up (default: 60)
                timeout?: number;
                // copy (default): the printed directory is copied into the plugin cache and content-hashed, so it may be deleted afterwards.
                mode?: 'copy' | 'link';
            } | {
                source: 'unsupported';
                error?: string;
            };
            description?: string;
            version?: string;
            strict?: boolean;
            // Versions of its hosts this plugin runs on.
            engines?: {
                [k: string]: unknown;
            };
            // HTTP headers sent when downloading this entry's `archive` source.
            headers?: {
                [k: string]: string;
            };
            // Command that prints a JSON object of HTTP headers for downloading this entry's `archive` source.
            headersHelper?: string;
        }[];
        owner?: {
            // Display name of the plugin author or organization
            name: string;
            // Contact email for support or feedback
            email?: string;
            // Website, GitHub profile, or organization URL
            url?: string;
        };
    })[];
    // Alias for strictKnownMarketplaces (managed settings only): this key is read exactly as if it were spelled strictKnownMarketplaces.
    allowedMarketplaces?: ({
        source: 'url';
        // Direct URL to marketplace.json file
        url: string;
        // Custom HTTP headers (e.g., for authentication)
        headers?: {
            [k: string]: string;
        };
        // Command that prints a JSON object of HTTP headers (e.g.
        headersHelper?: string;
    } | {
        source: 'github';
        // GitHub repository in owner/repo format.
        repo: string;
        // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
        ref?: string;
        // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
        path?: string;
        // Directories to include via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Has no effect; accepted so existing settings keep working.
        skipLfs?: boolean;
    } | {
        source: 'git';
        // Full git repository URL
        url: string;
        // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
        ref?: string;
        // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
        path?: string;
        // Directories to include via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Has no effect; accepted so existing settings keep working.
        skipLfs?: boolean;
    } | {
        source: 'npm';
        // npm package containing marketplace.json (e.g.
        package: string;
        // Version or range to fetch (e.g.
        version?: string;
        // Registry URL.
        registry?: string;
    } | {
        source: 'file';
        // Local file path to marketplace.json
        path: string;
    } | {
        source: 'directory';
        // Local directory containing .claude-plugin/marketplace.json
        path: string;
    } | {
        source: 'skills-dir';
    } | {
        source: 'hostPattern';
        // Regex pattern to match the host/domain extracted from any marketplace source type.
        hostPattern: string;
    } | {
        source: 'pathPattern';
        // Regex pattern matched against the .path field of file and directory sources.
        pathPattern: string;
    } | {
        source: 'settings';
        // Marketplace name.
        name: string;
        // Plugin entries declared inline in settings.json
        plugins: {
            // Plugin name as it appears in the target repository
            name: string;
            // Where to fetch the plugin from.
            source: string | {
                source: 'npm';
                // Package name (or url, or local path, or anything else that can be passed to `npm` as a package)
                package: string;
                // Specific version or version range (e.g., ^1.0.0, ~2.1.0)
                version?: string;
                // Custom NPM registry URL (defaults to using system default, likely npmjs.org)
                registry?: string;
            } | {
                source: 'url';
                // Full git repository URL (https:// or git\@)
                url: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'github';
                // GitHub repository in owner/repo format
                repo: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'git-subdir';
                // Git repository: GitHub owner/repo shorthand, https://, or git\@ URL
                url: string;
                // Subdirectory within the repo containing the plugin (e.g., "tools/claude-plugin").
                path: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'archive';
                // HTTPS URL of a zip archive containing the plugin.
                url: string;
                // SHA-256 digest of the archive.
                sha256?: string;
            } | {
                source: 'command';
                // Shell command that prints the absolute path of the plugin directory on stdout (exactly one line) and exits 0.
                command: string;
                // Seconds to wait for the command before giving up (default: 60)
                timeout?: number;
                // copy (default): the printed directory is copied into the plugin cache and content-hashed, so it may be deleted afterwards.
                mode?: 'copy' | 'link';
            } | {
                source: 'unsupported';
                error?: string;
            };
            description?: string;
            version?: string;
            strict?: boolean;
            // Versions of its hosts this plugin runs on.
            engines?: {
                [k: string]: unknown;
            };
            // HTTP headers sent when downloading this entry's `archive` source.
            headers?: {
                [k: string]: string;
            };
            // Command that prints a JSON object of HTTP headers for downloading this entry's `archive` source.
            headersHelper?: string;
        }[];
        owner?: {
            // Display name of the plugin author or organization
            name: string;
            // Contact email for support or feedback
            email?: string;
            // Website, GitHub profile, or organization URL
            url?: string;
        };
    })[];
    // Enterprise blocklist of marketplace sources.
    blockedMarketplaces?: ({
        source: 'url';
        // Direct URL to marketplace.json file
        url: string;
        // Custom HTTP headers (e.g., for authentication)
        headers?: {
            [k: string]: string;
        };
        // Command that prints a JSON object of HTTP headers (e.g.
        headersHelper?: string;
    } | {
        source: 'github';
        // GitHub repository in owner/repo format.
        repo: string;
        // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
        ref?: string;
        // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
        path?: string;
        // Directories to include via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Has no effect; accepted so existing settings keep working.
        skipLfs?: boolean;
    } | {
        source: 'git';
        // Full git repository URL
        url: string;
        // Git branch, tag or full commit id to use (e.g., "main", "v1.0.0").
        ref?: string;
        // Path to marketplace.json within repo (defaults to .claude-plugin/marketplace.json)
        path?: string;
        // Directories to include via git sparse-checkout (cone mode).
        sparsePaths?: string[];
        // Has no effect; accepted so existing settings keep working.
        skipLfs?: boolean;
    } | {
        source: 'npm';
        // npm package containing marketplace.json (e.g.
        package: string;
        // Version or range to fetch (e.g.
        version?: string;
        // Registry URL.
        registry?: string;
    } | {
        source: 'file';
        // Local file path to marketplace.json
        path: string;
    } | {
        source: 'directory';
        // Local directory containing .claude-plugin/marketplace.json
        path: string;
    } | {
        source: 'skills-dir';
    } | {
        source: 'hostPattern';
        // Regex pattern to match the host/domain extracted from any marketplace source type.
        hostPattern: string;
    } | {
        source: 'pathPattern';
        // Regex pattern matched against the .path field of file and directory sources.
        pathPattern: string;
    } | {
        source: 'settings';
        // Marketplace name.
        name: string;
        // Plugin entries declared inline in settings.json
        plugins: {
            // Plugin name as it appears in the target repository
            name: string;
            // Where to fetch the plugin from.
            source: string | {
                source: 'npm';
                // Package name (or url, or local path, or anything else that can be passed to `npm` as a package)
                package: string;
                // Specific version or version range (e.g., ^1.0.0, ~2.1.0)
                version?: string;
                // Custom NPM registry URL (defaults to using system default, likely npmjs.org)
                registry?: string;
            } | {
                source: 'url';
                // Full git repository URL (https:// or git\@)
                url: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'github';
                // GitHub repository in owner/repo format
                repo: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'git-subdir';
                // Git repository: GitHub owner/repo shorthand, https://, or git\@ URL
                url: string;
                // Subdirectory within the repo containing the plugin (e.g., "tools/claude-plugin").
                path: string;
                // Git branch or tag to use (e.g., "main", "v1.0.0").
                ref?: string;
                // Specific commit SHA to use
                sha?: string;
            } | {
                source: 'archive';
                // HTTPS URL of a zip archive containing the plugin.
                url: string;
                // SHA-256 digest of the archive.
                sha256?: string;
            } | {
                source: 'command';
                // Shell command that prints the absolute path of the plugin directory on stdout (exactly one line) and exits 0.
                command: string;
                // Seconds to wait for the command before giving up (default: 60)
                timeout?: number;
                // copy (default): the printed directory is copied into the plugin cache and content-hashed, so it may be deleted afterwards.
                mode?: 'copy' | 'link';
            } | {
                source: 'unsupported';
                error?: string;
            };
            description?: string;
            version?: string;
            strict?: boolean;
            // Versions of its hosts this plugin runs on.
            engines?: {
                [k: string]: unknown;
            };
            // HTTP headers sent when downloading this entry's `archive` source.
            headers?: {
                [k: string]: string;
            };
            // Command that prints a JSON object of HTTP headers for downloading this entry's `archive` source.
            headersHelper?: string;
        }[];
        owner?: {
            // Display name of the plugin author or organization
            name: string;
            // Contact email for support or feedback
            email?: string;
            // Website, GitHub profile, or organization URL
            url?: string;
        };
    })[];
    // Controls the `command` plugin source, whose plugin directory is produced by running a marketplace-declared command on this machine.
    disableCommandPluginSources?: boolean;
    // When true (and set in managed settings), rejects the --plugin-dir, --plugin-url, --agents, and non-sdk --mcp-config CLI flags at startup.
    disableSideloadFlags?: boolean;
    // Marketplace names whose plugins may surface as contextual install suggestions (relevance-based tips).
    pluginSuggestionMarketplaces?: string[];
    // Force a specific login method: "claudeai" for a Claude subscription (individual, Team, or Enterprise plans), "console" for Console billing, "gateway" for the Cloud gateway OIDC device flow
    forceLoginMethod?: 'claudeai' | 'console' | 'gateway';
    // Cloud gateway URL to pre-fill during login, alongside forceLoginMethod: "gateway".
    forceLoginGatewayUrl?: string;
    // IPv4 CIDR blocks (at most 4, each /8 to /32, not overlapping) your Cloud gateway sits in: the public block your organization numbers its internal network from, which lets /login reach a gateway there.
    gatewayInternalNetworks?: string[];
    // Controls whether the SDK parent tier (Options.managedSettings / --managed-settings) layers under this admin tier.
    parentSettingsBehavior?: 'first-wins' | 'merge';
    // Controls how the managed settings sources compose.
    managedSourcesBehavior?: 'first-wins' | 'merge';
    // Organization UUID to require for OAuth login.
    forceLoginOrgUUID?: string | string[];
    // Managed settings only (managed-settings.json, MDM, or server-managed).
    allowedProviders?: ('anthropic' | 'customEndpoint' | 'bedrock' | 'vertex' | 'foundry' | 'anthropicAws' | 'mantle' | 'gateway')[];
    // When set in managed settings, the CLI blocks startup until remote managed settings are freshly fetched, and exits if the fetch fails
    forceRemoteSettingsRefresh?: boolean;
    // Path to a script that outputs OpenTelemetry headers
    otelHeadersHelper?: string;
    // Controls the output style for assistant responses
    outputStyle?: string;
    // Default transcript view mode on startup
    viewMode?: 'default' | 'verbose' | 'focus';
    // Preferred language for Claude responses and voice dictation (e.g., "japanese", "spanish")
    language?: string;
    // Skip the WebFetch blocklist check for enterprise environments with restrictive security policies
    skipWebFetchPreflight?: boolean;
    sandbox?: {
        // Run Bash commands inside the sandbox.
        enabled?: boolean;
        // Exit with an error at startup if sandbox.enabled is true but the sandbox cannot start (missing dependencies or unsupported platform).
        failIfUnavailable?: boolean;
        // macOS and Linux/WSL only: approve a Bash command that will run inside the sandbox without a permission prompt, unless a deny rule, a command-specific ask rule or a safety check stops it.
        autoAllowBashIfSandboxed?: boolean;
        // Allow commands to run outside the sandbox via the dangerouslyDisableSandbox parameter.
        allowUnsandboxedCommands?: boolean;
        network?: {
            // Domains sandboxed commands may reach without a prompt (wildcards such as *.example.com supported).
            allowedDomains?: string[];
            // Domains that are always blocked, even if matched by allowedDomains.
            deniedDomains?: string[];
            // When true, the sandbox runtime deterministically denies hosts not in allowedDomains instead of prompting.
            strictAllowlist?: boolean;
            // When true (and set in managed settings), only allowedDomains and WebFetch(domain:...) allow rules from managed settings are respected.
            allowManagedDomainsOnly?: boolean;
            // macOS only: Unix socket paths to allow.
            allowUnixSockets?: string[];
            // If true, allow all Unix sockets (disables blocking on both platforms).
            allowAllUnixSockets?: boolean;
            // macOS only: If true, sandboxed commands can bind to localhost ports.
            allowLocalBinding?: boolean;
            // macOS only: Additional XPC/Mach service names to allow looking up.
            allowMachLookup?: string[];
            // Local TCP port of your own HTTP proxy for sandboxed traffic, used instead of the proxy Claude Code runs.
            httpProxyPort?: number;
            // Local TCP port of your own SOCKS5 proxy for sandboxed traffic, used instead of the proxy Claude Code runs.
            socksProxyPort?: number;
            // [EXPERIMENTAL] Enable in-process TLS termination so the per-request filter can see HTTPS request bodies.
            tlsTerminate?: {
                caCertPath?: string;
                caKeyPath?: string;
            };
        };
        filesystem?: {
            // Additional paths to allow writing within the sandbox.
            allowWrite?: string[];
            // Additional paths to deny writing within the sandbox.
            denyWrite?: string[];
            // Additional paths to deny reading within the sandbox.
            denyRead?: string[];
            // Paths to re-allow reading within denyRead regions.
            allowRead?: string[];
            // When true (set in managed settings), only allowRead paths from policySettings are used.
            allowManagedReadPathsOnly?: boolean;
            // macOS and Linux/WSL only: skip filesystem isolation entirely while keeping network and seccomp isolation.
            disabled?: boolean;
        };
        credentials?: {
            // Credential files or directories to protect.
            files?: {
                // Path to a credential file or directory.
                path: string;
                // Access mode for this path.
                mode: 'deny' | 'mask';
                // Optional regex for structured masking when mode is `mask`.
                extract?: string;
                // What to do when `extract` matches nothing in the file — or, with `decode`, when no candidate survives verification.
                onExtractNoMatch?: 'warn' | 'deny' | 'error';
                // Optional encoded-credential format for `mask` mode.
                decode?: 'jwt';
                // Names of top-level payload claims to mask inside each decoded value, instead of replacing the whole token.
                maskClaims?: string[];
                // If true, verbatim occurrences of each captured credential value outside the regex-matched spans are also replaced with the corresponding sentinel — for a secret repeated where the regex does not reach (e.g.
                maskDuplicates?: boolean;
                // Optional narrowing of where the proxy substitutes this credential.
                injectHosts?: string[];
            }[];
            // Environment variables to protect.
            envVars?: {
                // Environment variable name.
                name: string;
                // Access mode for this environment variable.
                mode: 'deny' | 'mask';
                // Optional regex for structured masking when mode is `mask`.
                extract?: string;
                // What to do when `extract` matches nothing in the value.
                onExtractNoMatch?: 'warn' | 'deny' | 'error';
                // Optional encoded-credential format for `mask` mode.
                decode?: 'jwt';
                // Names of top-level payload claims to mask inside the decoded value, instead of replacing the whole token.
                maskClaims?: string[];
                // Optional narrowing of where the proxy substitutes this credential.
                injectHosts?: string[];
            }[];
            // Allow sentinel→real substitution on the plain-HTTP proxy path.
            allowPlaintextInject?: boolean;
            // Explicit groupings of masked env vars into AWS credential pairs for SigV4 re-signing, for non-standard variable names.
            awsPairs?: {
                // Name of the masked env var holding the AWS access key id.
                accessKeyIdVar: string;
                // Name of the masked env var holding the AWS secret access key.
                secretAccessKeyVar: string;
                // Optional name of the masked env var holding the AWS session token (temporary credentials).
                sessionTokenVar?: string;
            }[];
            // Policies for AWS SigV4 request shapes the proxy cannot re-sign (streaming, presigned, sigv4a) when they reference a masked credential pair: `deny` (default) or `passthrough`.
            sigv4?: {
                // Policy for aws-chunked streaming uploads (x-amz-content-sha256: STREAMING-*): per-chunk signatures chain off the seed signature, so re-signing would require rewriting the body.
                streaming?: 'deny' | 'passthrough';
                // Policy for presigned URLs (X-Amz-Algorithm/X-Amz-Signature in the query, no Authorization header): the signature lives in the URL itself.
                presigned?: 'deny' | 'passthrough';
                // Policy for SigV4A (AWS4-ECDSA-P256-SHA256) asymmetric signatures: there is no shared-key HMAC to recompute.
                sigv4a?: 'deny' | 'passthrough';
            };
        };
        // Sandbox violations to leave unreported: a map of command patterns ("*" for every command) to the filesystem paths whose violations are ignored.
        ignoreViolations?: {
            [k: string]: string[];
        };
        // Linux only: Run without the fresh /proc mount, for hosts such as unprivileged Docker containers that cannot create one.
        enableWeakerNestedSandbox?: boolean;
        // Linux/WSL only: Let sandboxed commands create user namespaces of their own, which the sandbox refuses by default.
        allowNestedUserNamespaces?: boolean;
        // macOS only: Allow access to com.apple.trustd.agent in the sandbox.
        enableWeakerNetworkIsolation?: boolean;
        // macOS only: Allow sandboxed commands to send Apple Events (and look up the appleeventsd Mach service).
        allowAppleEvents?: boolean;
        // Command patterns (Bash permission-rule syntax) for commands to run outside the sandbox.
        excludedCommands?: string[];
        // Custom ripgrep configuration for bundled ripgrep support.
        ripgrep?: {
            command: string;
            args?: string[];
        };
        // Linux/WSL only: Absolute path to the bwrap (bubblewrap) binary.
        bwrapPath?: string;
        // Linux/WSL only: Absolute path to the socat binary used for the sandbox network proxy.
        socatPath?: string;
        [k: string]: unknown;
    };
    // Probability (0–1) that the session quality survey appears when eligible.
    feedbackSurveyRate?: number;
    // Model-drafted feedback (the SendFeedback tool).
    feedbackDrafts?: 'notify' | 'quiet' | 'off';
    // Whether to show tips in the spinner
    spinnerTipsEnabled?: boolean;

    // Customize spinner verbs.
    spinnerVerbs?: {
        mode: 'append' | 'replace';
        verbs: string[];
    };
    // Add your organization's own tips to the spinner tip rotation.
    spinnerTipsOverride?: {
        excludeDefault?: boolean;
        tips?: (string | {
            [k: string]: unknown;
        })[];
        // Absolute or ~/ local path to a JSON file holding an array of tips (same shapes as `tips`); honored from user, --settings and on-disk managed settings only.
        tipsFile?: string;
        // Prefix shown before your tips in the spinner (default "Tip")
        label?: string;
        [k: string]: unknown;
    };
    // Whether to disable syntax highlighting in diffs
    syntaxHighlightingDisabled?: boolean;
    // Maximum width, in terminal columns, of the prose in Claude's responses (paragraphs, headings, lists, blockquotes).
    maxProseWidth?: number;
    // Underline misspelled words in the prompt input as you type, using an installed aspell, hunspell or ispell (off unless "enabled" is true; does nothing if none is installed).
    spellcheck?: {
        // Turn on spell checking of the prompt input (default: false)
        enabled?: boolean;
        // Which spell checker to run: "aspell", "hunspell", "ispell", or "auto" (default) for the first of those found on PATH
        checker?: string;
        // Dictionary to use, passed to the checker as-is (aspell --lang, hunspell -d, ispell -d), e.g.
        language?: string;
        // Color of misspelled words (they are also underlined): a terminal color name such as "red" or "magenta", "#rrggbb", "rgb(r,g,b)", "ansi256(n)" or "ansi:<name>".
        color?: string;
        [k: string]: unknown;
    };
    // Whether /rename updates the terminal tab title (defaults to true).
    terminalTitleFromRename?: boolean;
    // Prompt cache TTL for the main conversation (interactive, -p and SDK turns, plus the helpers that run inline with it): "5m" or "1h".
    promptCacheTtl?: '5m' | '1h';
    // Prompt cache TTL for everything outside the main conversation — subagents, workflows, background and helper requests: "5m" or "1h".
    subagentPromptCacheTtl?: '5m' | '1h';
    // When false, thinking is disabled.
    alwaysThinkingEnabled?: boolean;
    // Persisted effort level for supported models.
    effortLevel?: 'low' | 'medium' | 'high' | 'xhigh';
    // Maximum effort level.
    maxEffortLevel?: 'low' | 'medium' | 'high' | 'xhigh' | 'max';
    // Per-model settings keyed by canonical model name.
    modelSettings?: {
        [k: string]: {
            // Persisted effort level for this model.
            effortLevel?: 'low' | 'medium' | 'high' | 'xhigh';
            // Maximum effort level for this model.
            maxEffortLevel?: 'low' | 'medium' | 'high' | 'xhigh' | 'max';
            // Auto-compact window for this model, in tokens (100000 to 1000000), or "auto" for the window tuned for the model.
            autoCompactWindow?: 'auto' | number;
            [k: string]: unknown;
        };
    };
    // Enable ultracode for the session: standing dynamic-workflow orchestration at any effort level.
    ultracode?: boolean;
    // Auto-compact window size
    autoCompactWindow?: number;
    // Advisor model for the server-side advisor tool.
    advisorModel?: string;
    // When true, fast mode is enabled.
    fastMode?: boolean;
    // When true, fast mode does not persist across sessions.
    fastModePerSessionOptIn?: boolean;
    // When false, prompt suggestions are disabled.
    promptSuggestionEnabled?: boolean;
    // When false, the :emoji: shortcode typeahead (the suggestion popup and the :name: inline replacement) is disabled.
    emojiCompletionEnabled?: boolean;

    // When true, the plan-approval dialog offers a "clear context" option.
    showClearContextOnPlanAccept?: boolean;
    // Idle time before Claude's questions in the terminal auto-continue with any answers selected so far.
    askUserQuestionTimeout?: '60s' | '5m' | '10m' | 'never';
    // Max time a permission/user dialog forwarded to a remote client stays parked awaiting an answer, and how long a HELD cross-session message awaits approval, before either resolves to its safe no-action default (cancelled / dropped-with-denial).
    dialogExpiry?: '60s' | '5m' | '10m' | 'never';
    // Name of an agent (built-in or custom) to use for the main thread.
    agent?: string;

    // Company announcements to display at startup (one will be randomly selected if multiple are provided)
    companyAnnouncements?: string[];
    // Per-plugin configuration including MCP server user configs, keyed by plugin ID (plugin\@marketplace format)
    pluginConfigs?: {
        [k: string]: {
            // User configuration values for MCP servers keyed by server name
            mcpServers?: {
                [k: string]: {
                    [k: string]: string | number | boolean | string[];
                };
            };
            // Non-sensitive option values from plugin manifest userConfig, keyed by option name.
            options?: {
                [k: string]: string | number | boolean | string[];
            };
        } | {
            [k: string]: unknown;
        };
    };
    // Cloud session configuration
    remote?: {
        // Default environment ID to use for cloud sessions
        defaultEnvironmentId?: string;
    };
    // Release channel for auto-updates (latest or stable)
    autoUpdatesChannel?: 'latest' | 'stable' | 'rc';
    // Minimum version to stay on - prevents downgrades when switching to stable channel
    minimumVersion?: string;
    // Minimum Claude Code version required to start.
    requiredMinimumVersion?: string;
    // Maximum Claude Code version allowed to start.
    requiredMaximumVersion?: string;
    // Custom directory for plan files, relative to project root.
    plansDirectory?: string;
    // Terminal UI renderer.
    tui?: 'default' | 'fullscreen';
    // Voice mode settings (hold-to-talk / tap-to-toggle dictation)
    voice?: {
        enabled?: boolean;
        // 'hold' (default): hold to talk.
        mode?: 'hold' | 'tap';
        // Submit the prompt when hold-to-talk is released (hold mode only)
        autoSubmit?: boolean;
    };
    // Managed-org opt-in for channel notifications (MCP servers with the claude/channel capability pushing inbound messages).
    channelsEnabled?: boolean;
    // Managed-org allowlist of channel plugins.
    allowedChannelPlugins?: {
        marketplace: string;
        plugin: string;
    }[];
    // Reduce or disable animations for accessibility (spinner shimmer, flash effects, etc.)
    prefersReducedMotion?: boolean;
    // Clock format for times shown in the UI: "auto" (default, follows the locale), "12-hour", "24-hour", "24-hour-utc" ("18:05Z"), or a strftime pattern such as "%H:%M" (any value containing "%"; other values read as "auto").
    timeFormat?: ('auto' | '12-hour' | '24-hour' | '24-hour-utc') | string;
    // IANA time zone for times shown in the UI, e.g.
    timeZone?: string;

    // Enable auto-memory for this project.
    autoMemoryEnabled?: boolean;
    // Custom directory path for auto-memory storage.
    autoMemoryDirectory?: string;
    // Enable background memory consolidation (auto-dream).
    autoDreamEnabled?: boolean;
    // Request API-side thinking summaries and show them in the conversation and in the transcript view (ctrl+o).
    showThinkingSummaries?: boolean;
    // Whether the user has accepted the bypass permissions mode dialog
    skipDangerousModePermissionPrompt?: boolean;

    // Disable auto mode
    disableAutoMode?: 'disable';

    // SSH connection configurations for remote environments.
    sshConfigs?: {
        // Unique identifier for this SSH config.
        id: string;
        // Display name for the SSH connection
        name: string;
        // SSH host in format "user\@hostname" or "hostname", or a host alias from ~/.ssh/config
        sshHost: string;
        // SSH port (default: 22)
        sshPort?: number;
        // Path to SSH identity file (private key)
        sshIdentityFile?: string;
        // Default working directory on the remote host.
        startDirectory?: string;
    }[];
    // SSH connections in Claude Desktop whose host comes from a command.
    sshHostCommands?: unknown[];
    // CLAUDE.md-style instructions injected as organization-managed memory.
    claudeMd?: string;
    // Glob patterns or absolute paths of CLAUDE.md files to exclude from loading.
    claudeMdExcludes?: string[];
    // Custom message to append to the plugin trust warning shown before installation.
    pluginTrustMessage?: string;
    // Color theme for the UI
    theme?: ('auto' | 'dark' | 'light' | 'light-daltonized' | 'dark-daltonized' | 'light-ansi' | 'dark-ansi') | string;
    // Key binding mode for the prompt input
    editorMode?: 'normal' | 'vim';
    // Deprecated: no longer has any effect.
    keybindingFlavor?: 'classic' | 'readline';
    // Vim INSERT-mode key-sequence remaps, e.g.
    vimInsertModeRemaps?: {
        [k: string]: unknown;
    };
    // Show full tool output instead of truncated summaries
    verbose?: boolean;
    // Preferred OS notification channel
    preferredNotifChannel?: 'auto' | 'iterm2' | 'terminal_bell' | 'iterm2_with_bell' | 'kitty' | 'ghostty' | 'notifications_disabled';
    // Automatically compact conversation when context fills
    autoCompactEnabled?: boolean;
    // Precompute the compaction summary in the background before it is needed.
    precomputeCompactionEnabled?: boolean;

    // Set to false to stop Claude Code from compacting a long conversation while the session is idle.
    idleCompaction?: boolean;
    // When safeguards flag a message, automatically switch to a different model to keep chatting.
    switchModelsOnFlag?: boolean;
    // When a claude.ai usage limit stops your session, wait for the limit to reset and continue the task automatically.
    autoContinueAtUsageLimit?: boolean;
    // Auto-scroll the conversation view to bottom (fullscreen mode only)
    autoScrollEnabled?: boolean;
    // Ramp mouse-wheel scroll speed during fast scrolls (fullscreen mode only)
    wheelScrollAccelerationEnabled?: boolean;
    // Snapshot files before edits so /rewind can restore them
    fileCheckpointingEnabled?: boolean;
    // Show "Cooked for Nm Ns" after each assistant turn
    showTurnDuration?: boolean;
    // Stamp each message with its arrival time
    showMessageTimestamps?: boolean;
    // Emit OSC 9;4 progress sequences during long operations
    terminalProgressBarEnabled?: boolean;
    // Enable the todo / task tracking panel
    todoFeatureEnabled?: boolean;
    // How spawned teammates execute (tmux, iterm2, in-process, auto)
    teammateMode?: 'auto' | 'tmux' | 'iterm2' | 'in-process';
    // Start Remote Control bridge automatically each session
    remoteControlAtStartup?: boolean;

    // Require explicit approval before SendMessage can reach a peer session on another machine via Remote Control
    isolatePeerMachines?: boolean;
    // When no background service is running: 'transient' spawns one for this login session; 'ask' offers to install it persistently
    daemonColdStart?: 'transient' | 'ask';
    // Inbound cross-session peer messages (SendMessage from your other sessions): 'accept' delivers them, 'hold' parks them for your review without letting Claude act, 'refuse' opts this session out.
    crossSessionInbound?: 'accept' | 'hold' | 'refuse';
    // Mirror local sessions to claude.ai as view-only (no remote control)
    autoUploadSessions?: boolean;
    // Push to mobile when a permission prompt or question is waiting
    inputNeededNotifEnabled?: boolean;
    // Allow Claude to push proactive mobile notifications
    agentPushNotifEnabled?: boolean;
    // Whether the user has accepted the auto mode opt-in dialog
    skipAutoPermissionPrompt?: boolean;
    // Whether plan mode uses auto mode semantics when auto mode is available (default: true)
    useAutoModeDuringPlan?: boolean;
    // Auto mode classifier prompt customization
    autoMode?: {
        // Rules for the auto mode classifier allow section.
        allow?: string[];
        // Rules for the auto mode classifier SOFT BLOCK section — destructive/irreversible actions that user intent can clear.
        soft_deny?: string[];
        // Rules for the auto mode classifier HARD BLOCK section — security boundaries that user intent does NOT clear.
        hard_deny?: string[];
        // Entries for the auto mode classifier environment section.
        environment?: string[];
        // When true, every Bash/PowerShell allow rule is suspended while auto mode is active so all shell commands are routed through the classifier (higher safety, more classifier calls).
        classifyAllShell?: boolean;
    };
    // Prevent claude-cli:// protocol handler registration with the OS
    disableDeepLinkRegistration?: 'disable';
    // Enable voice mode (hold-to-talk dictation)
    voiceEnabled?: boolean;
    // Default transcript view: chat (SendUserMessage checkpoints only) or transcript (full)
    defaultView?: 'chat' | 'transcript';
    [k: string]: unknown;
}

// Source for loading filesystem-based settings.
export declare type SettingSource = 'user' | 'project' | 'local';

export declare type SetupHookInput = BaseHookInput & {
    hook_event_name: 'Setup';
    trigger: 'init' | 'maintenance';
};

export declare type SetupHookSpecificOutput = {
    hookEventName: 'Setup';
    additionalContext?: string;
};

// Information about an available skill (invoked via /command syntax).
export declare type SlashCommand = {
    // Skill name (without the leading slash)
    name: string;
    // Description of what the skill does
    description: string;
    // Hint for skill arguments (e.g., "<file>")
    argumentHint: string;
    // Alternate names that resolve to this command (e.g., /cost and /stats both resolve to /usage)
    aliases?: string[];
    // True when the command is Claude Code's own; absent for a command defined by a user, project, plugin or MCP server.
    builtin?: boolean;

};

// A parked, fully started Claude Code process that is not yet any session (see {@link prewarm}). [@alpha]
export declare interface SpareProcess extends AsyncDisposable {
    // Bind the spare to a session and send its first message.
    claim(params: {
        prompt: string | AsyncIterable<SDKUserMessage>;
        options: ClaimOptions;
    }): Query;
    // Settles once the process has answered the claim AND the per-session settings sent with it: resolves with the canonical working directory, the session id and how long the spare was parked.
    readonly claimed: Promise<{
        cwd: string;
        sessionId: string;
        parkedMs?: number;
        sdkMcpSettled: boolean;
    }>;
    // Settles when the spare's process exits, claimed or not — a parked spare that dies should be replaced.
    readonly exited: Promise<void>;
    // Terminate the process.
    close(): void;
}

// Represents a spawned process with stdin/stdout streams and lifecycle management.
export declare interface SpawnedProcess {
    // Writable stream for sending data to the process stdin
    stdin: Writable;
    // Readable stream for receiving data from the process stdout
    stdout: Readable;
    // Whether the process has been killed
    readonly killed: boolean;
    // Exit code if the process has exited, null otherwise
    readonly exitCode: number | null;
    // Signal that terminated the process, if any.
    readonly signalCode?: NodeJS.Signals | null;
    // Kill the process with the given signal @param signal - The signal to send (e.g., 'SIGTERM', 'SIGKILL')
    kill(signal: NodeJS.Signals): boolean;
    // Register a callback for when the process exits @param event - Must be 'exit' @param listener - Callback receiving exit code and signal ProcessTransport's built-in local spawn delivers this only after the child's stderr has also closed (bounded by a short grace), so exit consumers see a complete stderr tail in exit errors.
    on(event: 'exit', listener: (code: number | null, signal: NodeJS.Signals | null) => void): void;
    // Register a callback for process errors @param event - Must be 'error' @param listener - Callback receiving the error
    on(event: 'error', listener: (error: Error) => void): void;
    // Register a one-time callback for when the process exits
    once(event: 'exit', listener: (code: number | null, signal: NodeJS.Signals | null) => void): void;
    once(event: 'error', listener: (error: Error) => void): void;
    // Remove an event listener
    off(event: 'exit', listener: (code: number | null, signal: NodeJS.Signals | null) => void): void;
    off(event: 'error', listener: (error: Error) => void): void;
}

// Options passed to the spawn function.
export declare interface SpawnOptions {
    // Command to execute
    command: string;
    // Arguments to pass to the command
    args: string[];
    // Working directory
    cwd?: string;
    // Environment variables
    env: {
        [envVar: string]: string | undefined;
    };
    // Abort signal for cancellation.
    signal: AbortSignal;
}

// Pre-warms the CLI subprocess so the first `query()` resolves immediately.
export declare function startup(_params?: {
    options?: Options;
    // The most time, in milliseconds, that startup() counts while it waits for the subprocess to initialize; when that time is used up, the promise rejects with a timeout error.
    initializeTimeoutMs?: number;
}): Promise<WarmQuery>;

// Everything the CLI writes to its output stream (stdout in stream-json mode): exactly one StdoutMessage per line, as a single JSON object.
declare type StdoutMessage = coreTypes.SDKMessage | coreTypes.SDKActiveGoalMessage | SDKControlResponse | SDKControlRequest | SDKControlCancelRequest | SDKKeepAliveMessage;

export declare type StopFailureHookInput = BaseHookInput & {
    hook_event_name: 'StopFailure';
    error: SDKAssistantMessageError;
    error_details?: string;
    last_assistant_message?: string;
};

export declare type StopHookInput = BaseHookInput & {
    hook_event_name: 'Stop';
    stop_hook_active: boolean;
    // Text content of the last assistant message before stopping.
    last_assistant_message?: string;
    // In-flight background work (running/pending + backgrounded) registered in this session.
    background_tasks?: BackgroundTaskSummary[];
    // Session-scoped cron tasks (CronCreate, ScheduleWakeup, /loop) that will wake this session later.
    session_crons?: SessionCronSummary[];

};

// Hook-specific output for the Stop event.
export declare type StopHookSpecificOutput = {
    hookEventName: 'Stop';
    additionalContext?: string;
};

export declare type SubagentStartHookInput = BaseHookInput & {
    hook_event_name: 'SubagentStart';
    agent_id: string;
    agent_type: string;
};

export declare type SubagentStartHookSpecificOutput = {
    hookEventName: 'SubagentStart';
    additionalContext?: string;
};

export declare type SubagentStopHookInput = BaseHookInput & {
    hook_event_name: 'SubagentStop';
    stop_hook_active: boolean;
    agent_id: string;
    agent_transcript_path: string;
    agent_type: string;
    // Text content of the last assistant message before stopping.
    last_assistant_message?: string;
    // In-flight background work (running/pending + backgrounded) registered in this session.
    background_tasks?: BackgroundTaskSummary[];
    // Session-scoped cron tasks (CronCreate, ScheduleWakeup, /loop) that will wake this session later.
    session_crons?: SessionCronSummary[];

};

// Hook-specific output for the SubagentStop event.
export declare type SubagentStopHookSpecificOutput = {
    hookEventName: 'SubagentStop';
    additionalContext?: string;
};

export declare type SyncHookJSONOutput = {
    continue?: boolean;
    suppressOutput?: boolean;
    stopReason?: string;
    decision?: 'approve' | 'block';
    systemMessage?: string;
    // A terminal escape sequence (e.g.
    terminalSequence?: string;
    reason?: string;

    hookSpecificOutput?: PreToolUseHookSpecificOutput | UserPromptSubmitHookSpecificOutput | UserPromptExpansionHookSpecificOutput | SessionStartHookSpecificOutput | SetupHookSpecificOutput | PreModelSwitchHookSpecificOutput | PostModelSwitchHookSpecificOutput | SubagentStartHookSpecificOutput | PostToolUseHookSpecificOutput | PostToolUseFailureHookSpecificOutput | PostToolBatchHookSpecificOutput | StopHookSpecificOutput | SubagentStopHookSpecificOutput | PermissionDeniedHookSpecificOutput | NotificationHookSpecificOutput | PermissionRequestHookSpecificOutput | ElicitationHookSpecificOutput | ElicitationResultHookSpecificOutput | CwdChangedHookSpecificOutput | FileChangedHookSpecificOutput | WorktreeCreateHookSpecificOutput | MessageDisplayHookSpecificOutput;
};

// Marker string that splits a custom `systemPrompt` into a static prefix (eligible for cross-session prompt caching) and a dynamic suffix (session-specific, not globally cached).
export declare const SYSTEM_PROMPT_DYNAMIC_BOUNDARY = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__";

// Tag a session.
export declare function tagSession(_sessionId: string, _tag: string | null, _options?: SessionMutationOptions): Promise<void>;

export declare type TaskCompletedHookInput = BaseHookInput & {
    hook_event_name: 'TaskCompleted';
    task_id: string;
    task_subject: string;
    task_description?: string;
    teammate_name?: string;
    // @deprecated Sessions have a single implicit team; this carries the session-derived team name and will be removed in a future release. [@deprecated]
    team_name?: string;
};

export declare type TaskCreatedHookInput = BaseHookInput & {
    hook_event_name: 'TaskCreated';
    task_id: string;
    task_subject: string;
    task_description?: string;
    teammate_name?: string;
    // @deprecated Sessions have a single implicit team; this carries the session-derived team name and will be removed in a future release. [@deprecated]
    team_name?: string;
};

export declare type TeammateIdleHookInput = BaseHookInput & {
    hook_event_name: 'TeammateIdle';
    teammate_name: string;
    // @deprecated Sessions have a single implicit team; this carries the session-derived team name and will be removed in a future release. [@deprecated]
    team_name: string;
};

// Why the query loop terminated.
export declare type TerminalReason = 'blocking_limit' | 'rapid_refill_breaker' | 'prompt_too_long' | 'image_error' | 'model_error' | 'api_error' | 'malformed_tool_use_exhausted' | 'aborted_streaming' | 'aborted_tools' | 'stop_hook_prevented' | 'hook_stopped' | 'tool_deferred' | 'max_turns' | 'background_requested' | 'completed' | 'budget_exhausted' | 'structured_output_retry_exhausted' | 'tool_deferred_unavailable' | 'turn_setup_failed';

// Claude decides when and how much to think (Opus 4.6+).
export declare type ThinkingAdaptive = {
    type: 'adaptive';
    display?: 'summarized' | 'omitted';
};

// Controls Claude's thinking/reasoning behavior.
export declare type ThinkingConfig = ThinkingAdaptive | ThinkingEnabled | ThinkingDisabled;

// No extended thinking
export declare type ThinkingDisabled = {
    type: 'disabled';
};

// Fixed thinking token budget (older models)
export declare type ThinkingEnabled = {
    type: 'enabled';
    budgetTokens?: number;
    display?: 'summarized' | 'omitted';
};

export declare function tool<Schema extends AnyZodRawShape>(_name: string, _description: string, _inputSchema: Schema, _handler: (args: InferShape<Schema>, extra: unknown) => Promise<CallToolResult>, _extras?: {
    annotations?: ToolAnnotations;
    searchHint?: string;
    alwaysLoad?: boolean;
}): SdkMcpToolDefinition<Schema>;

// Per-tool configuration for built-in tools.
export declare type ToolConfig = {
    askUserQuestion?: {
        // Content format for the `preview` field on question options. [@default]
        previewFormat?: 'markdown' | 'html';

    };
};

// Transport interface for Claude Code SDK communication Abstracts the communication layer to support both process and WebSocket transports
export declare interface Transport {
    // Write data to the transport May be async for network-based transports
    write(data: string): void | Promise<void>;
    // Close the transport connection and clean up resources This also closes stdin if still open (eliminating need for endInput)
    close(): void;
    // Check if transport is ready for communication
    isReady(): boolean;
    // Read and parse messages from the transport Each transport handles its own protocol and error checking
    readMessages(): AsyncGenerator<StdoutMessage, void, unknown>;
    // Register a request_id whose control_response the caller will await out-of-band via Query.awaitControlResponse.
    expectControlResponse?(requestId: string): void;
    // Called by Query immediately before it yields `message` to the SDK consumer.
    markDelivered?(message: object): void;
    // End the input stream
    endInput(): void;
    // Await the underlying subprocess's exit.
    waitForExit?(): Promise<void>;
    // Optional Disposable support.
    [Symbol.dispose]?(): void;
}

// Unarchive a session, so that `listSessions` returns it with `archived: false`.
export declare function unarchiveSession(_sessionId: string, _options?: SessionMutationOptions): Promise<void>;

// Messages meaning "a usage limit was genuinely reached" — the error-path outputs of getLimitReachedText (rateLimitMessages.ts) and getUsageCreditsOnlyModelRejectionContent (api/errors.ts). [@alpha]
export declare const USAGE_LIMIT_ERROR_PREFIXES: readonly ["You've hit your", "You've reached your", "You're out of usage credits", 'Your org is out of usage · add funds to continue', 'Your org is out of usage · contact your admin', "Your seat type doesn't include usage credits", "Your seat type doesn't include usage", 'Your usage allocation has been disabled by your admin', "Your group's usage limit is set to $0", 'Fable 5 requires usage credits', "You're out of extra usage", "Your seat type doesn't include extra usage"];

// Overage-transition notifications ("now drawing from credits"). [@alpha]
export declare const USAGE_TRANSITION_PREFIXES: readonly ["You're now using usage credits", "You're now using your usage allocation", 'Now using your usage allocation', 'Now using usage credits', 'Now using credits', "You're now using extra usage", 'Now using extra usage'];

// Approaching-limit warnings (severity:'warning'). [@alpha]
export declare const USAGE_WARNING_PREFIXES: readonly ["You've used", "You're close to"];

// A `request_user_dialog` control request from the CLI, asking the SDK consumer to render a blocking dialog and return the user's choice.
export declare type UserDialogRequest = {
    // Identifier for the dialog the host should render.
    dialogKind: string;
    // Dialog-specific data for the host renderer; shape is defined per dialogKind.
    payload: Record<string, unknown>;
    // Present when the dialog is tied to a specific tool invocation.
    toolUseID?: string;
};

// The host's answer to a {@link UserDialogRequest}.
export declare type UserDialogResult = {
    behavior: 'completed';
    result: unknown;
} | {
    behavior: 'cancelled';
};

export declare type UserPromptExpansionHookInput = BaseHookInput & {
    hook_event_name: 'UserPromptExpansion';
    expansion_type: 'slash_command' | 'mcp_prompt';
    command_name: string;
    command_args: string;
    command_source?: string;
    prompt: string;
};

export declare type UserPromptExpansionHookSpecificOutput = {
    hookEventName: 'UserPromptExpansion';
    additionalContext?: string;
    // When decision is "block", omit the original prompt from the block message
    suppressOriginalPrompt?: boolean;
};

export declare type UserPromptSubmitHookInput = BaseHookInput & {
    hook_event_name: 'UserPromptSubmit';
    prompt: string;
    // Who authored/injected the prompt: `user` = submitted from the interactive composer, `sdk` = non-interactive entrypoint (`-p` / Agent SDK), `loop_wakeup` = dynamic /loop wakeup, `schedule_wakeup` = scheduled-task fire (CronCreate or a cloud scheduled task), `system` = other machine-injected turns (peer/channel messages, task notifications, auto-continuation), `poll_event` = the poll-event channel enqueue-time pass (the hook fires when the host submits an event, before its delivery ack exists — a blocking verdict rejects the event).
    source?: 'user' | 'sdk' | 'system' | 'loop_wakeup' | 'schedule_wakeup' | 'poll_event';
    session_title?: string;
};

export declare type UserPromptSubmitHookSpecificOutput = {
    hookEventName: 'UserPromptSubmit';
    additionalContext?: string;
    sessionTitle?: string;
    // When decision is "block", omit the original prompt from the block message
    suppressOriginalPrompt?: boolean;
};

// A pre-warmed query handle returned by `startup()`.
export declare interface WarmQuery extends AsyncDisposable {
    // Send a prompt to the pre-warmed subprocess and return the Query.
    query(prompt: string | AsyncIterable<SDKUserMessage>): Query;
    // Close the subprocess without sending a prompt.
    close(): void;
}

export declare type WorktreeCreateHookInput = BaseHookInput & {
    hook_event_name: 'WorktreeCreate';
    name: string;
};

// Hook-specific output for the WorktreeCreate event.
export declare type WorktreeCreateHookSpecificOutput = {
    hookEventName: 'WorktreeCreate';
    worktreePath: string;
};

export declare type WorktreeRemoveHookInput = BaseHookInput & {
    hook_event_name: 'WorktreeRemove';
    worktree_path: string;
};

export { }
