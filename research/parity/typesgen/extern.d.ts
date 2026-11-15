// Hand-written TS stand-ins for types sdk.d.ts imports from other packages
// (@anthropic-ai/sdk, @modelcontextprotocol/sdk, zod) or declares with
// non-wire constructs. Fed to gen.mjs alongside sdk.d.ts; these override
// same-named declarations there.

/** Shaped like an Anthropic Messages API Message object (role "assistant") as the CLI emits it on `assistant.message`. Hand-modelled from @anthropic-ai/sdk BetaMessage; unknown fields survive in Raw. */
export type BetaMessage = {
    id: string;
    type: 'message';
    role: 'assistant';
    model: string;
    content: BetaContentBlock[];
    stop_reason: string | null;
    stop_sequence: string | null;
    usage: NonNullableUsage;
    container?: {
        id: string;
        expires_at: string;
    } | null;
    context_management?: Record<string, unknown> | null;
};

/** Token usage (BetaUsage with nulls removed, as the CLI reports it on result messages and assistant messages). */
export type NonNullableUsage = {
    input_tokens: number;
    output_tokens: number;
    cache_creation_input_tokens: number;
    cache_read_input_tokens: number;
    cache_creation?: {
        ephemeral_1h_input_tokens: number;
        ephemeral_5m_input_tokens: number;
    } | null;
    server_tool_use?: {
        web_search_requests: number;
        web_fetch_requests?: number;
    } | null;
    service_tier?: string | null;
};

/** Usage delta carried by a message_delta stream event. */
export type MessageDeltaUsage = {
    input_tokens?: number | null;
    output_tokens: number;
    cache_creation_input_tokens?: number | null;
    cache_read_input_tokens?: number | null;
    server_tool_use?: {
        web_search_requests: number;
        web_fetch_requests?: number;
    } | null;
};

/** One content block of an assistant or user message (Messages API content blocks and block params, merged into one union keyed by "type"). */
export type BetaContentBlock = TextBlock | ThinkingBlock | RedactedThinkingBlock | ToolUseBlock | ServerToolUseBlock | ToolResultBlock | ImageBlock | DocumentBlock | SearchResultBlock | MCPToolUseBlock | MCPToolResultBlock | ContainerUploadBlock | WebSearchToolResultBlock | WebFetchToolResultBlock | CodeExecutionToolResultBlock | BashCodeExecutionToolResultBlock | TextEditorCodeExecutionToolResultBlock;

/** Text content block. */
export type TextBlock = {
    type: 'text';
    text: string;
    citations?: unknown[] | null;
};
/** Extended-thinking block. */
export type ThinkingBlock = {
    type: 'thinking';
    thinking: string;
    signature: string;
};
/** Redacted thinking block (encrypted). */
export type RedactedThinkingBlock = {
    type: 'redacted_thinking';
    data: string;
};
/** A client tool call made by the model. */
export type ToolUseBlock = {
    type: 'tool_use';
    id: string;
    name: string;
    input: unknown;
};
/** A server-side tool call (web_search, web_fetch, code_execution, ...). */
export type ServerToolUseBlock = {
    type: 'server_tool_use';
    id: string;
    name: string;
    input: unknown;
};
/** A tool result (appears in user messages). `content` is a string or an array of text/image/document/search_result blocks. */
export type ToolResultBlock = {
    type: 'tool_result';
    tool_use_id: string;
    content?: unknown;
    is_error?: boolean;
};
/** Image block; `source` is {type:'base64',media_type,data} | {type:'url',url} | {type:'file',file_id}. */
export type ImageBlock = {
    type: 'image';
    source: Record<string, unknown>;
};
/** Document block (PDF / plain text / content / URL / file source). */
export type DocumentBlock = {
    type: 'document';
    source: Record<string, unknown>;
    title?: string | null;
    context?: string | null;
    citations?: Record<string, unknown> | null;
};
/** Search-result block (user content). */
export type SearchResultBlock = {
    type: 'search_result';
    source: string;
    title: string;
    content: unknown[];
};
/** MCP connector tool call (server-side MCP). */
export type MCPToolUseBlock = {
    type: 'mcp_tool_use';
    id: string;
    name: string;
    server_name: string;
    input: unknown;
};
/** MCP connector tool result. `content` is a string or an array of text blocks. */
export type MCPToolResultBlock = {
    type: 'mcp_tool_result';
    tool_use_id: string;
    is_error: boolean;
    content: unknown;
};
/** Container upload block. */
export type ContainerUploadBlock = {
    type: 'container_upload';
    file_id: string;
};
/** Web search server-tool result. */
export type WebSearchToolResultBlock = {
    type: 'web_search_tool_result';
    tool_use_id: string;
    content: unknown;
};
/** Web fetch server-tool result. */
export type WebFetchToolResultBlock = {
    type: 'web_fetch_tool_result';
    tool_use_id: string;
    content: unknown;
};
/** Code execution server-tool result. */
export type CodeExecutionToolResultBlock = {
    type: 'code_execution_tool_result';
    tool_use_id: string;
    content: unknown;
};
/** Bash code execution server-tool result. */
export type BashCodeExecutionToolResultBlock = {
    type: 'bash_code_execution_tool_result';
    tool_use_id: string;
    content: unknown;
};
/** Text editor code execution server-tool result. */
export type TextEditorCodeExecutionToolResultBlock = {
    type: 'text_editor_code_execution_tool_result';
    tool_use_id: string;
    content: unknown;
};

/** An Anthropic Messages API user message: role "user"; content is a string or an array of content blocks (kept raw: use MessageParam.Text / MessageParam.Blocks). */
export type MessageParam = {
    role: 'user' | 'assistant';
    content: string | BetaContentBlock[];
};

/** One Anthropic Messages API streaming event (BetaRawMessageStreamEvent), as carried on stream_event messages. */
export type BetaRawMessageStreamEvent = MessageStartEvent | MessageDeltaEvent | MessageStopEvent | ContentBlockStartEvent | ContentBlockDeltaEvent | ContentBlockStopEvent;
export type MessageStartEvent = {
    type: 'message_start';
    message: BetaMessage;
};
export type MessageDeltaEvent = {
    type: 'message_delta';
    delta: {
        stop_reason: string | null;
        stop_sequence: string | null;
    };
    usage: MessageDeltaUsage;
    context_management?: Record<string, unknown> | null;
};
export type MessageStopEvent = {
    type: 'message_stop';
};
export type ContentBlockStartEvent = {
    type: 'content_block_start';
    index: number;
    content_block: BetaContentBlock;
};
export type ContentBlockDeltaEvent = {
    type: 'content_block_delta';
    index: number;
    delta: ContentBlockDelta;
};
export type ContentBlockStopEvent = {
    type: 'content_block_stop';
    index: number;
};
/** Delta of a content_block_delta event. */
export type ContentBlockDelta = TextDelta | InputJSONDelta | CitationsDelta | ThinkingDelta | SignatureDelta;
export type TextDelta = {
    type: 'text_delta';
    text: string;
};
export type InputJSONDelta = {
    type: 'input_json_delta';
    partial_json: string;
};
export type CitationsDelta = {
    type: 'citations_delta';
    citation: Record<string, unknown>;
};
export type ThinkingDelta = {
    type: 'thinking_delta';
    thinking: string;
};
export type SignatureDelta = {
    type: 'signature_delta';
    signature: string;
};

/** MCP elicitation result (ElicitResult from @modelcontextprotocol/sdk): the answer to an `elicitation` control request. */
export type ElicitResult = {
    action: 'accept' | 'decline' | 'cancel';
    content?: Record<string, unknown>;
    _meta?: Record<string, unknown>;
};

/** MCP tool annotations (ToolAnnotations from @modelcontextprotocol/sdk). */
export type ToolAnnotations = {
    title?: string;
    readOnlyHint?: boolean;
    destructiveHint?: boolean;
    idempotentHint?: boolean;
    openWorldHint?: boolean;
};

/** MCP tools/call result (CallToolResult from @modelcontextprotocol/sdk), returned by an SDK MCP tool handler. */
export type CallToolResult = {
    content: unknown[];
    structuredContent?: Record<string, unknown>;
    isError?: boolean;
    _meta?: Record<string, unknown>;
};

/** Union of MCP server configs a host can pass. In Go the in-process ("sdk") variant carries only the name: the host runs the server itself and answers mcp_message requests. */
export type McpServerConfig = McpStdioServerConfig | McpSSEServerConfig | McpHttpServerConfig | McpSdkServerConfig;
export type McpServerConfigForProcessTransport = McpServerConfig;

/** Sandbox settings (z.infer of SandboxSettingsSchema; a loose object, so unknown keys survive in Raw). */
export type SandboxSettings = {
    enabled?: boolean;
    failIfUnavailable?: boolean;
    autoAllowBashIfSandboxed?: boolean;
    allowUnsandboxedCommands?: boolean;
    network?: {
        allowedDomains?: string[];
        deniedDomains?: string[];
        strictAllowlist?: boolean;
        allowManagedDomainsOnly?: boolean;
        allowUnixSockets?: string[];
        allowAllUnixSockets?: boolean;
        allowLocalBinding?: boolean;
        allowMachLookup?: string[];
        httpProxyPort?: number;
        socksProxyPort?: number;
        tlsTerminate?: {
            caCertPath?: string;
            caKeyPath?: string;
        };
    };
    filesystem?: {
        allowWrite?: string[];
        denyWrite?: string[];
        denyRead?: string[];
        allowRead?: string[];
        allowManagedReadPathsOnly?: boolean;
        disabled?: boolean;
    };
    credentials?: Record<string, unknown>;
    ignoreViolations?: Record<string, string[]>;
    enableWeakerNestedSandbox?: boolean;
    allowNestedUserNamespaces?: boolean;
    enableWeakerNetworkIsolation?: boolean;
    allowAppleEvents?: boolean;
    excludedCommands?: string[];
    ripgrep?: {
        command: string;
        args?: string[];
    };
    bwrapPath?: string;
    socatPath?: string;
};
