package claude

import (
	"context"
	"encoding/json"
	"time"
)

// Version targets. The client is written against Claude Code CLI 2.1.311,
// which ships with Agent SDK 0.3.311 (SDK 0.3.N <-> CLI 2.1.N).
const (
	TargetCLIVersion = "2.1.311"
	TargetSDKVersion = "0.3.311"
	// DefaultEntrypoint is sent as CLAUDE_CODE_ENTRYPOINT when the environment
	// does not set one. The TypeScript SDK sends "sdk-ts". The CLI keeps a
	// closed list of known entrypoints and separately treats any value that
	// starts with "sdk-" as an SDK client, so "sdk-go" is honest and still
	// SDK-class. Set Options.Entrypoint to "sdk-ts" for strict parity.
	DefaultEntrypoint = "sdk-go"
)

// Options mirrors the Agent SDK's Options for everything that reaches the
// CLI. Field docs name the SDK option and where it goes: argv, env, or the
// initialize request. Zero values mean "not set" unless stated otherwise.
type Options struct {
	// ---- Process -------------------------------------------------------

	// PathToClaudeCodeExecutable is the CLI to run (SDK: pathToClaudeCodeExecutable).
	// Required: unlike the npm package, this client bundles no binary.
	// A path ending in .js/.mjs/.ts/.tsx/.jsx is run through Executable.
	PathToClaudeCodeExecutable string
	// Executable runs a JS entrypoint (SDK: executable). Default "node".
	Executable string
	// ExecutableArgs are prepended to argv, also for a native binary.
	ExecutableArgs []string
	// Cwd is the child's working directory (SDK: cwd).
	Cwd string
	// Env replaces the inherited environment when non-nil (SDK: env). The
	// SDK's own additions and removals still apply; see buildEnv.
	Env map[string]string
	// ExtraArgs are appended after every built-in flag, in order (SDK: extraArgs).
	ExtraArgs []ExtraArg
	// Stderr receives raw stderr chunks (SDK: stderr).
	Stderr func(chunk string)
	// OnProcessSpawned is called with the child pid (SDK: onProcessSpawned).
	OnProcessSpawned func(pid int)

	// Entrypoint and SDKVersion are the values for CLAUDE_CODE_ENTRYPOINT and
	// CLAUDE_AGENT_SDK_VERSION when the environment has none.
	Entrypoint string
	SDKVersion string

	// MaxLineBytes caps one stdout line; 0 = unlimited (the SDK's behaviour).
	MaxLineBytes int
	// InitializeTimeout bounds the wait for the initialize response. Zero
	// means no timeout, like the SDK's query(). The SDK's startup() uses 60 s
	// plus time spent in SessionStart/Setup hooks; 60 s is a sensible value
	// for an interactive host.
	InitializeTimeout time.Duration

	// ---- Session -------------------------------------------------------

	Resume          string // --resume=<id>
	Continue        bool   // --continue
	ForkSession     bool   // --fork-session
	ResumeSessionAt string // --resume-session-at=<uuid>: keep messages up to and including this entry
	ResumeDropsTurn *bool  // --resume-drops-turn=<true|false>
	SessionID       string // --session-id=<uuid>
	// NoSessionPersistence maps to SDK persistSession:false.
	NoSessionPersistence bool

	// ---- Model ---------------------------------------------------------

	Model             string          // --model=<m>
	FallbackModel     string          // --fallback-model=<m>; must differ from Model
	Thinking          *ThinkingConfig // wins over MaxThinkingTokens
	MaxThinkingTokens *int            // 0 -> --thinking=disabled; N -> --max-thinking-tokens=N
	Effort            string          // --effort=<level>
	MaxTurns          int             // --max-turns=<n>; 0 emits nothing
	MaxBudgetUSD      *float64        // --max-budget-usd=<n>; 0 IS emitted
	TaskBudgetTotal   int             // --task-budget=<total>
	Agent             string          // --agent=<name>
	Betas             []string        // --betas=<a,b>

	// ---- Tools and permissions -----------------------------------------

	AllowedTools    []string     // --allowedTools=<a,b>
	DisallowedTools []string     // --disallowedTools=<a,b>
	Tools           *ToolsOption // --tools=<a,b> | --tools= | --tools=default
	Skills          *SkillsOption
	PermissionMode  string // --permission-mode=<mode>
	// AllowDangerouslySkipPermissions unlocks bypassPermissions.
	AllowDangerouslySkipPermissions bool
	PermissionPromptToolName        string // --permission-prompt-tool=<name>; not with CanUseTool
	PermissionPrompts               string // --permission-prompts=<v>
	// CanUseTool answers can_use_tool requests. Setting it adds
	// --permission-prompt-tool=stdio.
	CanUseTool CanUseToolFunc

	// ---- Configuration -------------------------------------------------

	// SettingSources: nil emits no flag; a non-nil empty slice emits the
	// literal "--setting-sources=" (load no settings files).
	SettingSources        []string
	Settings              string   // --settings <path or JSON> (sent in the extra-args section)
	AdditionalDirectories []string // --add-dir=<d>, repeated
	// McpServers are external servers (stdio, sse, http) for --mcp-config.
	McpServers []McpServerConfig
	// SdkMcpServers are served by this process over the control channel.
	SdkMcpServers           []SdkMcpServer
	StrictMcpConfig         bool
	Plugins                 []PluginConfig
	ProjectConfigRoot       *string
	EnableFileCheckpointing bool // env CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true; needed for rewind_files
	// AskUserQuestionPreviewFormat is SDK toolConfig.askUserQuestion.previewFormat
	// (env CLAUDE_CODE_QUESTION_PREVIEW_FORMAT), e.g. "markdown".
	AskUserQuestionPreviewFormat string

	// ---- Output --------------------------------------------------------

	IncludePartialMessages bool            // --include-partial-messages (stream_event frames)
	IncludeHookEvents      bool            // --include-hook-events
	JSONSchema             json.RawMessage // SDK outputFormat{type:"json_schema"}: argv and initialize
	Debug                  bool            // --debug
	DebugFile              string          // --debug-file=<path>; wins over Debug

	// ---- Initialize request --------------------------------------------

	// SystemPrompt: nil sends systemPrompt:[""] (an EMPTY prompt, as the SDK
	// does). Use PresetSystemPrompt for Claude Code's own prompt.
	SystemPrompt               *SystemPrompt
	PlanModeInstructions       string
	AppendSubagentSystemPrompt string
	Agents                     json.RawMessage // {"name": AgentDefinition}
	Title                      string
	Hooks                      []HookEventConfig
	PromptSuggestions          *bool
	AgentProgressSummaries     *bool
	ForwardSubagentText        *bool
	SupportedDialogKinds       []string // requires OnUserDialog
	PerTaskStopAffordance      *bool
	RapidFollowupPreempt       *bool
	// ForwardNestedSubagentToolCalls makes nested subagents' tool calls visible.
	ForwardNestedSubagentToolCalls *bool
	WorkspaceTrust                 json.RawMessage
	// InitializeExtra adds or overrides initialize fields this client does not
	// model yet. Keys are appended in order after the known fields.
	InitializeExtra []InitField

	// OnElicitation answers MCP elicitation requests. Without it the reply is
	// {"action":"decline"}.
	OnElicitation ElicitationFunc
	// OnUserDialog answers request_user_dialog. Without it no reply is sent.
	OnUserDialog UserDialogFunc
}

// ExtraArg is one entry of SDK extraArgs. A nil Value emits a bare "--key".
type ExtraArg struct {
	Key   string
	Value *string
}

// Flag returns an ExtraArg with no value ("--key").
func Flag(key string) ExtraArg { return ExtraArg{Key: key} }

// Arg returns an ExtraArg with a value ("--key value").
func Arg(key, value string) ExtraArg { return ExtraArg{Key: key, Value: &value} }

// ThinkingConfig is SDK thinking. Type is "enabled", "adaptive" or "disabled".
type ThinkingConfig struct {
	Type         string
	BudgetTokens *int   // only with "enabled"
	Display      string // --thinking-display=<v>, ignored when disabled
}

// ToolsOption is SDK tools: a list (possibly empty) or the default preset.
type ToolsOption struct {
	Preset bool // --tools=default
	List   []string
}

// SkillsOption is SDK skills: 'all' or a list of names.
type SkillsOption struct {
	All   bool
	Names []string
}

// McpServerConfig is one external MCP server. Config is the server object as
// it appears in --mcp-config, e.g. {"type":"stdio","command":"…","args":[…]}.
type McpServerConfig struct {
	Name   string
	Config json.RawMessage
}

// PluginConfig is SDK plugins[]. Only type "local" exists.
type PluginConfig struct {
	Path             string
	SkipMcpDiscovery bool // --plugin-dir-no-mcp instead of --plugin-dir
}

// SystemPrompt is SDK systemPrompt.
type SystemPrompt struct {
	// Preset selects Claude Code's built-in prompt: systemPrompt is omitted
	// from initialize and Append is sent as appendSystemPrompt.
	Preset                 bool
	Append                 string
	ExcludeDynamicSections *bool
	// Custom is the full prompt (one or more blocks) when Preset is false.
	Custom []string
	// Snapshot is SDK systemPrompt.snapshot (systemPromptSnapshot).
	Snapshot *bool
}

// PresetSystemPrompt is {type:"preset", preset:"claude_code", append}.
func PresetSystemPrompt(appendText string) *SystemPrompt {
	return &SystemPrompt{Preset: true, Append: appendText}
}

// CustomSystemPrompt replaces the system prompt entirely.
func CustomSystemPrompt(blocks ...string) *SystemPrompt { return &SystemPrompt{Custom: blocks} }

// InitField is one extra initialize field.
type InitField struct {
	Key   string
	Value any
}

// ---- Callbacks ---------------------------------------------------------

// PermissionRequest is a can_use_tool control request. Raw holds the full
// request object, so fields added by newer CLIs are never lost.
type PermissionRequest struct {
	RequestID             string          `json:"-"` // envelope request_id
	ToolName              string          `json:"tool_name"`
	Input                 json.RawMessage `json:"input"`
	ToolUseID             string          `json:"tool_use_id,omitempty"`
	AgentID               string          `json:"agent_id,omitempty"` // set when a subagent asks
	PermissionSuggestions json.RawMessage `json:"permission_suggestions,omitempty"`
	BlockedPath           string          `json:"blocked_path,omitempty"`
	DecisionReason        string          `json:"decision_reason,omitempty"`
	Title                 string          `json:"title,omitempty"`
	DisplayName           string          `json:"display_name,omitempty"`
	Description           string          `json:"description,omitempty"`
	Raw                   json.RawMessage `json:"-"`
}

// PermissionResult is the answer to a can_use_tool request (SDK PermissionResult).
// The client adds toolUseID from the request, as the SDK does.
type PermissionResult struct {
	Behavior string // "allow" or "deny"
	// UpdatedInput is the input to run (allow). Send the original input back
	// unchanged, or the user's edited version. Optional in the 0.3.311 types
	// but always send it: older CLIs require it.
	UpdatedInput json.RawMessage
	// UpdatedPermissions are PermissionUpdate objects (allow). Only send
	// entries taken from the request's PermissionSuggestions.
	UpdatedPermissions json.RawMessage
	Message            string // deny: fed back to Claude as the tool error
	Interrupt          bool   // deny: also stop the turn
	// Raw, when set, is sent instead of the fields above (plus toolUseID).
	// Use it to forward a result built elsewhere, e.g. by a gRPC client.
	Raw json.RawMessage
}

// Allow approves a tool call with the given input (normally req.Input).
func Allow(input json.RawMessage) *PermissionResult {
	return &PermissionResult{Behavior: "allow", UpdatedInput: input}
}

// Deny rejects a tool call; message is returned to Claude.
func Deny(message string) *PermissionResult {
	return &PermissionResult{Behavior: "deny", Message: message}
}

// CanUseToolFunc decides one tool call. ctx is cancelled when the CLI sends
// control_cancel_request (for example after an interrupt) or the query
// closes. Returning (nil, nil) sends no reply, like a null result in the SDK.
type CanUseToolFunc func(ctx context.Context, req PermissionRequest) (*PermissionResult, error)

// HookEventConfig lists the matchers for one hook event, e.g. "PreToolUse".
// Order matters: callback ids hook_0, hook_1, … are assigned across events,
// matchers and hooks in the order given.
type HookEventConfig struct {
	Event    string
	Matchers []HookMatcher
}

// HookMatcher is SDK HookCallbackMatcher.
type HookMatcher struct {
	Matcher   *string // tool-name pattern, e.g. "Edit|Write"; nil matches all
	Hooks     []HookFunc
	Timeout   *int   // seconds, enforced by the CLI
	OnFailure string // "", "continue" or "block"
}

// HookFunc handles one hook invocation. input is the event's HookInput; the
// return value is marshalled as the HookJSONOutput (nil sends no "response").
type HookFunc func(ctx context.Context, input json.RawMessage, toolUseID string) (any, error)

// SdkMcpServer is an MCP server hosted by this process (SDK createSdkMcpServer).
// The CLI talks to it through mcp_message control requests.
type SdkMcpServer struct {
	Name string
	// Handle receives one JSON-RPC message from the CLI. For a request
	// (method + non-null id) it returns the JSON-RPC response with the same
	// id. For a notification it returns nil.
	Handle func(ctx context.Context, message json.RawMessage) (json.RawMessage, error)
	// Settings is sent as initialize sdkMcpServerConfigs[name] when non-empty
	// (timeout, tools, …).
	Settings json.RawMessage
}

// ElicitationFunc answers an MCP elicitation request. request is the full
// control request object. Return {"action":"accept"|"decline"|"cancel",…};
// (nil, nil) sends no reply.
type ElicitationFunc func(ctx context.Context, requestID string, request json.RawMessage) (any, error)

// UserDialogFunc answers request_user_dialog. (nil, nil) sends no reply.
type UserDialogFunc func(ctx context.Context, requestID string, request json.RawMessage) (any, error)
