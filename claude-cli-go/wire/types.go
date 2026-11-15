// Package wire holds the wire types and API-surface types of the Claude Agent
// SDK (TypeScript @anthropic-ai/claude-agent-sdk), re-expressed in Go.
//
// Target: Agent SDK 0.3.311 / Claude Code CLI 2.1.311. Declarations added after
// 0.3.311 (up to 0.3.325) are kept so a newer CLI decodes fully; each one is
// marked "since 0.3.3xx (not in 0.3.311)". Where the changelog does not name
// the release, the marker gives the range 0.3.312-0.3.325.
//
// Conventions
//
//   - JSON tags are the exact wire names, which mix snake_case and camelCase
//     exactly as the CLI does.
//   - Optional TS fields carry omitempty. Optional numbers, booleans and
//     objects are pointers, so a zero value is not confused with "absent".
//     A required TS field typed `T | null` is a pointer without omitempty.
//   - TS `number` is float64, except fields that are always integers (token
//     counts, counts, indexes, byte sizes), which are int.
//   - TS `unknown` and the unions that do not map to a Go type are json.RawMessage.
//     `Record<string, unknown>` is map[string]any.
//   - Unions use one pattern throughout. A union is a struct that holds its
//     discriminator(s) (for example Type and Subtype) and the complete Raw JSON.
//     Typed As<Variant>() accessors and Decode() decode Raw on demand, and
//     New<Union>(v) wraps a variant for sending. A union whose discriminator
//     is not known to this package still decodes, and Raw keeps it intact.
//   - Every wire struct keeps the JSON it was decoded from in Raw (`json:"-"`).
//     MarshalJSON writes the declared fields, then re-adds every key from Raw
//     that the struct does not declare. It also re-adds a declared key that
//     omitempty dropped but whose value is unchanged since decoding, such as
//     an explicit null, "" or []. Decode then encode is therefore lossless,
//     though key order is not kept. A newer CLI's extra fields are never
//     dropped.
//   - Decoding is tolerant. A field whose JSON type does not match its Go type
//     is left zero, and the rest of the value still decodes, with Raw intact.
//   - Wire structs whose discriminator is a fixed string literal fill it in
//     on MarshalJSON when it is left empty.
//   - No type here holds a function. TS callbacks (canUseTool, hooks,
//     onElicitation, onUserDialog, MCP tool handlers, spawnClaudeCodeProcess,
//     stderr) become data. Each one is either a boolean flag in Options or a
//     callback id (HookCallbackMatcher.HookCallbackIDs), and the matching
//     control_request / control_response payloads are types in this package.
//     A gRPC host can forward them over the network.
//
// The bulk of this file is generated from sdk.d.ts. The section before the
// "generated" banner is written by hand.
package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// SDKVersion and CLIVersion are the versions these types target.
const (
	SDKVersion = "0.3.311"
	CLIVersion = "2.1.311"
)

// ---------------------------------------------------------------------------
// Union and raw-preservation helpers
// ---------------------------------------------------------------------------

// ErrVariantMismatch is returned by an As<Variant> accessor whose discriminator does not match.
var ErrVariantMismatch = errors.New("claude: union variant mismatch")

// ErrUnknownVariant is returned by Decode for a discriminator this package does not model.
var ErrUnknownVariant = errors.New("claude: unknown union variant")

func asVariant[T any](raw json.RawMessage, ok bool, union, variant string) (*T, error) {
	if !ok {
		return nil, fmt.Errorf("%w: %s is not %s", ErrVariantMismatch, union, variant)
	}
	v := new(T)
	if len(raw) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return nil, err
	}
	return v, nil
}

func unknownVariant(union string, raw json.RawMessage) error {
	head := raw
	if len(head) > 120 {
		head = head[:120]
	}
	return fmt.Errorf("%w: %s %s", ErrUnknownVariant, union, head)
}

// unmarshalUnion records the complete JSON and peeks the string discriminators.
// It never fails on a well-formed value: a non-object or a non-string
// discriminator just leaves the discriminator fields empty.
func unmarshalUnion(b []byte, raw *json.RawMessage, keys []string, dst ...*string) error {
	for _, d := range dst {
		*d = ""
	}
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		*raw = nil
		return nil
	}
	if !json.Valid(b) {
		return fmt.Errorf("claude: invalid JSON in union value")
	}
	*raw = append(json.RawMessage(nil), b...)
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	for i, k := range keys {
		if v, ok := m[k]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				*dst[i] = s
			}
		}
	}
	return nil
}

func marshalUnion(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	return raw, nil
}

func wrapUnion(v any, u json.Unmarshaler) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return u.UnmarshalJSON(b)
}

func peekBool(raw json.RawMessage, key string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	var b bool
	return json.Unmarshal(m[key], &b) == nil && b
}

// unmarshalKeep decodes b into dst (a pointer to a method-less copy of the
// struct type) and stores b in raw. JSON type mismatches are tolerated.
func unmarshalKeep(b []byte, dst any, raw *json.RawMessage) error {
	if err := json.Unmarshal(b, dst); err != nil {
		var te *json.UnmarshalTypeError
		if !errors.As(err, &te) {
			return err
		}
	}
	*raw = append(json.RawMessage(nil), b...)
	return nil
}

var fieldIndexCache sync.Map // reflect.Type -> map[string][]int

func jsonFieldIndex(t reflect.Type) map[string][]int {
	if m, ok := fieldIndexCache.Load(t); ok {
		return m.(map[string][]int)
	}
	m := map[string][]int{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" || !f.IsExported() {
			continue
		}
		name := f.Name
		if tag != "" {
			if j := bytes.IndexByte([]byte(tag), ','); j >= 0 {
				tag = tag[:j]
			}
			if tag != "" {
				name = tag
			}
		}
		m[name] = f.Index
	}
	fieldIndexCache.Store(t, m)
	return m
}

// marshalKeep encodes v (a method-less copy of a wire struct) and merges back
// keys from raw: undeclared keys always; declared keys when the field still
// holds exactly what raw decodes to (the original bytes are kept), or when the
// raw value never fit the field's Go type and the field is still zero.
func marshalKeep(v any, raw json.RawMessage) ([]byte, error) {
	out, err := json.Marshal(v)
	if err != nil || len(raw) == 0 {
		return out, err
	}
	var rawObj map[string]json.RawMessage
	if json.Unmarshal(raw, &rawObj) != nil || rawObj == nil {
		return out, nil
	}
	var outObj map[string]json.RawMessage
	if err := json.Unmarshal(out, &outObj); err != nil {
		return nil, err
	}
	rv := reflect.ValueOf(v)
	idx := jsonFieldIndex(rv.Type())
	changed := false
	for k, val := range rawObj {
		cur, present := outObj[k]
		if present && bytes.Equal(cur, val) {
			continue
		}
		fi, known := idx[k]
		if !known {
			outObj[k], changed = val, true
			continue
		}
		fv := rv.FieldByIndex(fi)
		probe := reflect.New(fv.Type())
		perr := json.Unmarshal(val, probe.Interface())
		switch {
		case perr == nil && reflect.DeepEqual(probe.Elem().Interface(), fv.Interface()):
			// Unchanged since decoding: keep the original bytes (explicit null, "", [], number formatting).
			outObj[k], changed = val, true
		case perr != nil && fv.IsZero():
			// The JSON value never fit the Go type (tolerated on decode) and was not set since: keep it.
			outObj[k], changed = val, true
		}
	}
	if !changed {
		return out, nil
	}
	return json.Marshal(outObj)
}

// ---------------------------------------------------------------------------
// Hand-written types
// ---------------------------------------------------------------------------

// Settings is a Claude Code settings object (TS: Settings, the generated
// settings.json interface: about 180 keys, every one optional). It is kept as
// a generic map. Hosts pass it through and do not interpret it.
type Settings = map[string]any

// HookJSONOutput is what a hook callback answers. On the wire it is the
// `response` payload of the control_response to a hook_callback request (TS:
// HookJSONOutput = AsyncHookJSONOutput | SyncHookJSONOutput). The
// discriminator is the boolean "async". Async is true for
// AsyncHookJSONOutput and false for SyncHookJSONOutput.
type HookJSONOutput struct {
	// Async is the "async" discriminator (true only for AsyncHookJSONOutput).
	Async bool `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *HookJSONOutput) UnmarshalJSON(b []byte) error {
	if err := unmarshalUnion(b, &u.Raw, nil); err != nil {
		return err
	}
	u.Async = peekBool(u.Raw, "async")
	return nil
}
func (u HookJSONOutput) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewHookJSONOutput wraps an AsyncHookJSONOutput or SyncHookJSONOutput.
func NewHookJSONOutput(v any) (HookJSONOutput, error) {
	var u HookJSONOutput
	err := wrapUnion(v, &u)
	return u, err
}

// AsAsyncHookJSONOutput decodes u as AsyncHookJSONOutput.
func (u HookJSONOutput) AsAsyncHookJSONOutput() (*AsyncHookJSONOutput, error) {
	return asVariant[AsyncHookJSONOutput](u.Raw, u.Async, "HookJSONOutput", "AsyncHookJSONOutput")
}

// AsSyncHookJSONOutput decodes u as SyncHookJSONOutput.
func (u HookJSONOutput) AsSyncHookJSONOutput() (*SyncHookJSONOutput, error) {
	return asVariant[SyncHookJSONOutput](u.Raw, !u.Async, "HookJSONOutput", "SyncHookJSONOutput")
}

// Decode returns *AsyncHookJSONOutput or *SyncHookJSONOutput.
func (u HookJSONOutput) Decode() (any, error) {
	if u.Async {
		return u.AsAsyncHookJSONOutput()
	}
	return u.AsSyncHookJSONOutput()
}

// Text returns the content of a MessageParam whose content is a plain string.
func (m MessageParam) Text() (string, bool) {
	var s string
	if json.Unmarshal(m.Content, &s) != nil {
		return "", false
	}
	return s, true
}

// Blocks returns the content of a MessageParam as content blocks. A
// string content comes back as one TextBlock.
func (m MessageParam) Blocks() ([]ContentBlock, error) {
	if s, ok := m.Text(); ok {
		cb, err := NewContentBlock(TextBlock{Type: "text", Text: s})
		return []ContentBlock{cb}, err
	}
	var bs []ContentBlock
	err := json.Unmarshal(m.Content, &bs)
	return bs, err
}

// SDKMessage returns u as a Message when it is one of the SDKMessage
// variants, so neither a control frame, keep_alive nor active_goal.
func (u StdoutMessage) SDKMessage() (Message, bool) {
	switch u.Type {
	case "control_request", "control_response", "control_cancel_request", "keep_alive", "active_goal":
		return Message{}, false
	}
	return Message{Type: u.Type, Subtype: u.Subtype, Raw: u.Raw}, true
}

// ResultMessage returns m as a ResultMessage union when it is a result
// message (TS: SDKResultMessage = SDKResultSuccess | SDKResultError).
func (m Message) ResultMessage() (ResultMessage, bool) {
	if m.Type != "result" {
		return ResultMessage{}, false
	}
	return ResultMessage{Type: m.Type, Subtype: m.Subtype, Raw: m.Raw}, true
}

// --- Control payloads that sdk.mjs 0.3.311 sends or reads but sdk.d.ts does not declare ---

// SetMcpPermissionModeOverrideRequest is sent by Query.setMcpPermissionModeOverride.
// The subtype is "set_mcp_permission_mode_override". It is not in the d.ts
// control-request union, and its shape comes from sdk.mjs 0.3.311.
type SetMcpPermissionModeOverrideRequest struct {
	// Subtype is always "set_mcp_permission_mode_override".
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
	// Mode is "default", "auto" or null (clear the override).
	Mode *string `json:"mode"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *SetMcpPermissionModeOverrideRequest) UnmarshalJSON(b []byte) error {
	type plain SetMcpPermissionModeOverrideRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetMcpPermissionModeOverrideRequest) MarshalJSON() ([]byte, error) {
	type plain SetMcpPermissionModeOverrideRequest
	if v.Subtype == "" {
		v.Subtype = "set_mcp_permission_mode_override"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SetMcpPermissionModeOverrideResponse is the success payload for
// set_mcp_permission_mode_override. Its shape comes from sdk.d.ts
// Query.setMcpPermissionModeOverride.
type SetMcpPermissionModeOverrideResponse struct {
	Warning string `json:"warning,omitempty"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *SetMcpPermissionModeOverrideResponse) UnmarshalJSON(b []byte) error {
	type plain SetMcpPermissionModeOverrideResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetMcpPermissionModeOverrideResponse) MarshalJSON() ([]byte, error) {
	type plain SetMcpPermissionModeOverrideResponse
	return marshalKeep(plain(v), v.Raw)
}

// McpStatusResponse is the success payload for mcp_status. Query.mcpServerStatus
// returns response.mcpServers.
type McpStatusResponse struct {
	McpServers []McpServerStatus `json:"mcpServers"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *McpStatusResponse) UnmarshalJSON(b []byte) error {
	type plain McpStatusResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpStatusResponse) MarshalJSON() ([]byte, error) {
	type plain McpStatusResponse
	return marshalKeep(plain(v), v.Raw)
}

// BackgroundTasksResponse is the success payload for background_tasks.
// Query.backgroundTasks returns `backgrounded ?? true`.
type BackgroundTasksResponse struct {
	Backgrounded *bool `json:"backgrounded,omitempty"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *BackgroundTasksResponse) UnmarshalJSON(b []byte) error {
	type plain BackgroundTasksResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BackgroundTasksResponse) MarshalJSON() ([]byte, error) {
	type plain BackgroundTasksResponse
	return marshalKeep(plain(v), v.Raw)
}

// McpMessageResponse is the success payload the host returns for an
// mcp_message request: the in-process MCP server's JSON-RPC reply. For a
// notification it is {"jsonrpc":"2.0","result":{},"id":0}.
type McpMessageResponse struct {
	McpResponse json.RawMessage `json:"mcp_response"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *McpMessageResponse) UnmarshalJSON(b []byte) error {
	type plain McpMessageResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpMessageResponse) MarshalJSON() ([]byte, error) {
	type plain McpMessageResponse
	return marshalKeep(plain(v), v.Raw)
}

// CanUseToolResponse is the success payload the host returns for a
// can_use_tool request. It is the PermissionResult with toolUseID set to the
// request's tool_use_id: the SDK sends {...result, toolUseID}.
type CanUseToolResponse = PermissionResult

// --- Success payloads that sdk.d.ts does not declare. These shapes were seen in captured CLI 2.1.311 control_responses. ---

// ListModelsResponse is the success payload for list_models (observed; not declared in sdk.d.ts).
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *ListModelsResponse) UnmarshalJSON(b []byte) error {
	type plain ListModelsResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ListModelsResponse) MarshalJSON() ([]byte, error) {
	type plain ListModelsResponse
	return marshalKeep(plain(v), v.Raw)
}

// GetSessionCostResponse is the success payload for get_session_cost: the
// formatted /usage text (observed; not declared in sdk.d.ts).
type GetSessionCostResponse struct {
	Text string `json:"text"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *GetSessionCostResponse) UnmarshalJSON(b []byte) error {
	type plain GetSessionCostResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetSessionCostResponse) MarshalJSON() ([]byte, error) {
	type plain GetSessionCostResponse
	return marshalKeep(plain(v), v.Raw)
}

// GetSettingsResponse is the success payload for get_settings: the effective
// merged settings, the raw per-source settings, and the values in force
// (observed; not declared in sdk.d.ts).
type GetSettingsResponse struct {
	Effective Settings          `json:"effective"`
	Sources   []json.RawMessage `json:"sources"`
	Applied   map[string]any    `json:"applied,omitempty"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *GetSettingsResponse) UnmarshalJSON(b []byte) error {
	type plain GetSettingsResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetSettingsResponse) MarshalJSON() ([]byte, error) {
	type plain GetSettingsResponse
	return marshalKeep(plain(v), v.Raw)
}

// SetPermissionModeResponse is the success payload for set_permission_mode:
// the mode now in force (observed; not declared in sdk.d.ts, and the TS SDK
// ignores it).
type SetPermissionModeResponse struct {
	Mode PermissionMode `json:"mode"`
	// Raw is the JSON this value was decoded from.
	Raw json.RawMessage `json:"-"`
}

func (v *SetPermissionModeResponse) UnmarshalJSON(b []byte) error {
	type plain SetPermissionModeResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetPermissionModeResponse) MarshalJSON() ([]byte, error) {
	type plain SetPermissionModeResponse
	return marshalKeep(plain(v), v.Raw)
}

// ErrorCode returns the machine-readable error_code of a control error
// (e.g. "reload_plugins_busy"). The CLI sends it and sdk.mjs reads it, but
// sdk.d.ts does not declare it, so it is read from Raw.
func (v ControlErrorResponse) ErrorCode() string {
	var m struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(v.Raw, &m)
	return m.ErrorCode
}

// ControlRequestSubtypes lists every control-request subtype in the
// 0.3.311 sdk.d.ts union (ControlRequestInner), by direction. "host" means
// the SDK host sends it to the CLI, and "cli" means the CLI sends it to the
// host. set_has_subscribers is a later addition and is not listed.
var ControlRequestSubtypes = map[string]string{
	"initialize": "host", "interrupt": "host", "set_permission_mode": "host", "set_model": "host",
	"set_max_thinking_tokens": "host", "rename_session": "host", "set_color": "host", "mcp_status": "host",
	"get_context_usage": "host", "get_session_cost": "host", "list_models": "host", "get_usage": "host",
	"get_binary_version": "host", "mcp_call": "host", "file_suggestions": "host", "rewind_files": "host",
	"cancel_async_message": "host", "read_file": "host", "seed_read_state": "host", "mcp_set_servers": "host",
	"register_repo_root": "host", "reload_plugins": "host", "reload_skills": "host", "reload_output_styles": "host",
	"mcp_reconnect": "host", "mcp_toggle": "host", "stop_task": "host", "background_tasks": "host",
	"get_task_output": "host", "apply_flag_settings": "host", "get_settings": "host", "get_hooks_listing": "host",
	"update_settings": "host", "list_permission_rules": "host", "mcp_read_resource": "host",
	"can_use_tool": "cli", "hook_callback": "cli", "mcp_message": "cli", "elicitation": "cli", "request_user_dialog": "cli",
}

// ---------------------------------------------------------------------------
// generated from sdk.d.ts below this line
// ---------------------------------------------------------------------------

// AccountInfo - Information about the logged in user's account. (TS: AccountInfo)
type AccountInfo struct {
	Email            string `json:"email,omitempty"`
	Organization     string `json:"organization,omitempty"`
	SubscriptionType string `json:"subscriptionType,omitempty"`
	TokenSource      string `json:"tokenSource,omitempty"`
	APIKeySource     string `json:"apiKeySource,omitempty"`
	// Active API backend. (one of "firstParty", "bedrock", "vertex", "foundry", "anthropicAws", "anthropicGoogleCloud", "mantle", "gateway")
	APIProvider string `json:"apiProvider,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *AccountInfo) UnmarshalJSON(b []byte) error {
	type plain AccountInfo
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v AccountInfo) MarshalJSON() ([]byte, error) {
	type plain AccountInfo
	return marshalKeep(plain(v), v.Raw)
}

// AgentMcpServerSpec is kept as raw JSON. TS: string | Record<string, McpServerConfigForProcessTransport>. (TS: AgentMcpServerSpec)
type AgentMcpServerSpec = json.RawMessage

// PermissionMode is a string enum. Permission mode for controlling how tool executions are handled. (TS: PermissionMode)
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeDontAsk           PermissionMode = "dontAsk"
	PermissionModeAuto              PermissionMode = "auto"
)

// AgentDefinition - Definition for a custom subagent that can be invoked via the Agent tool. (TS: AgentDefinition)
type AgentDefinition struct {
	// Natural language description of when to use this agent
	Description string `json:"description"`
	// Array of allowed tool names.
	Tools []string `json:"tools,omitempty"`
	// Array of tool names to explicitly disallow for this agent.
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	// The agent's system prompt
	Prompt string `json:"prompt"`
	// Model alias (e.g.
	Model      string               `json:"model,omitempty"`
	MCPServers []AgentMcpServerSpec `json:"mcpServers,omitempty"`
	// Experimental: Critical reminder added to system prompt
	CriticalSystemReminderEXPERIMENTAL string `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"`
	// Array of skill names to preload into the agent context
	Skills []string `json:"skills,omitempty"`
	// Auto-submitted as the first user turn when this agent is the main thread agent.
	InitialPrompt string `json:"initialPrompt,omitempty"`
	// Maximum number of agentic turns (API round-trips) before stopping
	MaxTurns *int `json:"maxTurns,omitempty"`
	// Token count at which this agent compacts its own conversation when it runs as a subagent.
	AutoCompactWindow *float64 `json:"autoCompactWindow,omitempty"`
	// Run this agent as a background task (non-blocking, fire-and-forget) when invoked
	Background *bool `json:"background,omitempty"`
	// Run this agent without the user, project and local CLAUDE.md instruction files when it runs as a subagent; managed policy files are kept.
	OmitClaudeMd *bool `json:"omitClaudeMd,omitempty"`
	// Scope for auto-loading agent memory files. (one of "user", "project", "local")
	Memory string `json:"memory,omitempty"`
	// Reasoning effort level for this agent. (TS: ('low' | 'medium' | 'high' | 'xhigh' | 'max') | number)
	Effort json.RawMessage `json:"effort,omitempty"`
	// Permission mode controlling how tool executions are handled
	PermissionMode PermissionMode `json:"permissionMode,omitempty"`
	// Agent type auto-spawned as a background observer whenever this agent runs.
	Observer string `json:"observer,omitempty"`
	// Supplemental postamble appended (after the harness-owned default) to each activity digest sent to the observer.
	ObserverMessage string `json:"observerMessage,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *AgentDefinition) UnmarshalJSON(b []byte) error {
	type plain AgentDefinition
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v AgentDefinition) MarshalJSON() ([]byte, error) {
	type plain AgentDefinition
	return marshalKeep(plain(v), v.Raw)
}

// AgentInfo - Information about an available subagent that can be invoked via the Task tool. (TS: AgentInfo)
type AgentInfo struct {
	// Agent type identifier (e.g., "Explore")
	Name string `json:"name"`
	// Description of when to use this agent
	Description string `json:"description"`
	// Model this agent uses: an alias or model ID, or 'inherit' for the parent's model.
	Model string `json:"model,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *AgentInfo) UnmarshalJSON(b []byte) error {
	type plain AgentInfo
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v AgentInfo) MarshalJSON() ([]byte, error) {
	type plain AgentInfo
	return marshalKeep(plain(v), v.Raw)
}

// ApiKeySource is a string enum. Where the credential used for API requests came from: 'ANTHROPIC_API_KEY' (environment variable), 'apiKeyHelper' (the configured helper command), '/login managed key' (an API key created and stored by /login with an... (TS: ApiKeySource)
type ApiKeySource string

const (
	ApiKeySourceANTHROPICAPIKEY      ApiKeySource = "ANTHROPIC_API_KEY"
	ApiKeySourceApiKeyHelper         ApiKeySource = "apiKeyHelper"
	ApiKeySourceSlashLoginManagedKey ApiKeySource = "/login managed key"
	ApiKeySourceNone                 ApiKeySource = "none"
	ApiKeySourceUser                 ApiKeySource = "user"
	ApiKeySourceProject              ApiKeySource = "project"
	ApiKeySourceOrg                  ApiKeySource = "org"
	ApiKeySourceTemporary            ApiKeySource = "temporary"
	ApiKeySourceOauth                ApiKeySource = "oauth"
)

// AsyncHookJSONOutput (TS: AsyncHookJSONOutput)
type AsyncHookJSONOutput struct {
	// (TS literal `true`)
	Async        bool     `json:"async"`
	AsyncTimeout *float64 `json:"asyncTimeout,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *AsyncHookJSONOutput) UnmarshalJSON(b []byte) error {
	type plain AsyncHookJSONOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v AsyncHookJSONOutput) MarshalJSON() ([]byte, error) {
	type plain AsyncHookJSONOutput
	return marshalKeep(plain(v), v.Raw)
}

// BackgroundTaskSummary (TS: BackgroundTaskSummary)
type BackgroundTaskSummary struct {
	ID string `json:"id"`
	// Friendly task-type label (e.g.
	Type   string `json:"type"`
	Status string `json:"status"`
	// Free-text description.
	Description string `json:"description"`
	// Shell command line.
	Command string `json:"command,omitempty"`
	// Subagent type name.
	AgentType string `json:"agent_type,omitempty"`
	// MCP server name.
	Server string `json:"server,omitempty"`
	// MCP tool name.
	Tool string `json:"tool,omitempty"`
	// Workflow name.
	Name string `json:"name,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BackgroundTaskSummary) UnmarshalJSON(b []byte) error {
	type plain BackgroundTaskSummary
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BackgroundTaskSummary) MarshalJSON() ([]byte, error) {
	type plain BackgroundTaskSummary
	return marshalKeep(plain(v), v.Raw)
}

// BaseHookInputEffort - inline object (field effort of BaseHookInput)
type BaseHookInputEffort struct {
	// Active effort level for the current turn (e.g., "low", "medium", "high", "xhigh", "max"), after any silent downgrade for the selected model.
	Level string `json:"level"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BaseHookInputEffort) UnmarshalJSON(b []byte) error {
	type plain BaseHookInputEffort
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BaseHookInputEffort) MarshalJSON() ([]byte, error) {
	type plain BaseHookInputEffort
	return marshalKeep(plain(v), v.Raw)
}

// BaseHookInput (TS: BaseHookInput)
type BaseHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BaseHookInput) UnmarshalJSON(b []byte) error {
	type plain BaseHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BaseHookInput) MarshalJSON() ([]byte, error) {
	type plain BaseHookInput
	return marshalKeep(plain(v), v.Raw)
}

// OutputFormatType is a string enum. (TS: OutputFormatType)
type OutputFormatType string

const (
	OutputFormatTypeJsonSchema OutputFormatType = "json_schema"
)

// BaseOutputFormat (TS: BaseOutputFormat)
type BaseOutputFormat struct {
	Type OutputFormatType `json:"type"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BaseOutputFormat) UnmarshalJSON(b []byte) error {
	type plain BaseOutputFormat
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BaseOutputFormat) MarshalJSON() ([]byte, error) {
	type plain BaseOutputFormat
	return marshalKeep(plain(v), v.Raw)
}

// BashCodeExecutionToolResultBlock - Bash code execution server-tool result. (hand-modelled stand-in for TS BashCodeExecutionToolResultBlock)
type BashCodeExecutionToolResultBlock struct {
	// (always "bash_code_execution_tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BashCodeExecutionToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain BashCodeExecutionToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BashCodeExecutionToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain BashCodeExecutionToolResultBlock
	if v.Type == "" {
		v.Type = "bash_code_execution_tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TextBlock - Text content block. (hand-modelled stand-in for TS TextBlock)
type TextBlock struct {
	// (always "text")
	Type string `json:"type"`
	Text string `json:"text"`
	// (optional, nullable)
	Citations []json.RawMessage `json:"citations,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TextBlock) UnmarshalJSON(b []byte) error {
	type plain TextBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TextBlock) MarshalJSON() ([]byte, error) {
	type plain TextBlock
	if v.Type == "" {
		v.Type = "text"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ThinkingBlock - Extended-thinking block. (hand-modelled stand-in for TS ThinkingBlock)
type ThinkingBlock struct {
	// (always "thinking")
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ThinkingBlock) UnmarshalJSON(b []byte) error {
	type plain ThinkingBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ThinkingBlock) MarshalJSON() ([]byte, error) {
	type plain ThinkingBlock
	if v.Type == "" {
		v.Type = "thinking"
	}
	return marshalKeep(plain(v), v.Raw)
}

// RedactedThinkingBlock - Redacted thinking block (encrypted). (hand-modelled stand-in for TS RedactedThinkingBlock)
type RedactedThinkingBlock struct {
	// (always "redacted_thinking")
	Type string `json:"type"`
	Data string `json:"data"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RedactedThinkingBlock) UnmarshalJSON(b []byte) error {
	type plain RedactedThinkingBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RedactedThinkingBlock) MarshalJSON() ([]byte, error) {
	type plain RedactedThinkingBlock
	if v.Type == "" {
		v.Type = "redacted_thinking"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ToolUseBlock - A client tool call made by the model. (hand-modelled stand-in for TS ToolUseBlock)
type ToolUseBlock struct {
	// (always "tool_use")
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolUseBlock) UnmarshalJSON(b []byte) error {
	type plain ToolUseBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolUseBlock) MarshalJSON() ([]byte, error) {
	type plain ToolUseBlock
	if v.Type == "" {
		v.Type = "tool_use"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ServerToolUseBlock - A server-side tool call (web_search, web_fetch, code_execution, ...). (hand-modelled stand-in for TS ServerToolUseBlock)
type ServerToolUseBlock struct {
	// (always "server_tool_use")
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ServerToolUseBlock) UnmarshalJSON(b []byte) error {
	type plain ServerToolUseBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ServerToolUseBlock) MarshalJSON() ([]byte, error) {
	type plain ServerToolUseBlock
	if v.Type == "" {
		v.Type = "server_tool_use"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ToolResultBlock - A tool result (appears in user messages). (hand-modelled stand-in for TS ToolResultBlock)
type ToolResultBlock struct {
	// (always "tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   *bool           `json:"is_error,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain ToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain ToolResultBlock
	if v.Type == "" {
		v.Type = "tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ImageBlock - Image block; `source` is {type:'base64',media_type,data} | {type:'url',url} | {type:'file',file_id}. (hand-modelled stand-in for TS ImageBlock)
type ImageBlock struct {
	// (always "image")
	Type   string         `json:"type"`
	Source map[string]any `json:"source"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ImageBlock) UnmarshalJSON(b []byte) error {
	type plain ImageBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ImageBlock) MarshalJSON() ([]byte, error) {
	type plain ImageBlock
	if v.Type == "" {
		v.Type = "image"
	}
	return marshalKeep(plain(v), v.Raw)
}

// DocumentBlock - Document block (PDF / plain text / content / URL / file source). (hand-modelled stand-in for TS DocumentBlock)
type DocumentBlock struct {
	// (always "document")
	Type   string         `json:"type"`
	Source map[string]any `json:"source"`
	// (optional, nullable)
	Title *string `json:"title,omitempty"`
	// (optional, nullable)
	Context *string `json:"context,omitempty"`
	// (optional, nullable)
	Citations map[string]any `json:"citations,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *DocumentBlock) UnmarshalJSON(b []byte) error {
	type plain DocumentBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v DocumentBlock) MarshalJSON() ([]byte, error) {
	type plain DocumentBlock
	if v.Type == "" {
		v.Type = "document"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SearchResultBlock - Search-result block (user content). (hand-modelled stand-in for TS SearchResultBlock)
type SearchResultBlock struct {
	// (always "search_result")
	Type    string            `json:"type"`
	Source  string            `json:"source"`
	Title   string            `json:"title"`
	Content []json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SearchResultBlock) UnmarshalJSON(b []byte) error {
	type plain SearchResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SearchResultBlock) MarshalJSON() ([]byte, error) {
	type plain SearchResultBlock
	if v.Type == "" {
		v.Type = "search_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MCPToolUseBlock - MCP connector tool call (server-side MCP). (hand-modelled stand-in for TS MCPToolUseBlock)
type MCPToolUseBlock struct {
	// (always "mcp_tool_use")
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	ServerName string          `json:"server_name"`
	Input      json.RawMessage `json:"input"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MCPToolUseBlock) UnmarshalJSON(b []byte) error {
	type plain MCPToolUseBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MCPToolUseBlock) MarshalJSON() ([]byte, error) {
	type plain MCPToolUseBlock
	if v.Type == "" {
		v.Type = "mcp_tool_use"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MCPToolResultBlock - MCP connector tool result. (hand-modelled stand-in for TS MCPToolResultBlock)
type MCPToolResultBlock struct {
	// (always "mcp_tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
	Content   json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MCPToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain MCPToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MCPToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain MCPToolResultBlock
	if v.Type == "" {
		v.Type = "mcp_tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ContainerUploadBlock - Container upload block. (hand-modelled stand-in for TS ContainerUploadBlock)
type ContainerUploadBlock struct {
	// (always "container_upload")
	Type   string `json:"type"`
	FileID string `json:"file_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContainerUploadBlock) UnmarshalJSON(b []byte) error {
	type plain ContainerUploadBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContainerUploadBlock) MarshalJSON() ([]byte, error) {
	type plain ContainerUploadBlock
	if v.Type == "" {
		v.Type = "container_upload"
	}
	return marshalKeep(plain(v), v.Raw)
}

// WebSearchToolResultBlock - Web search server-tool result. (hand-modelled stand-in for TS WebSearchToolResultBlock)
type WebSearchToolResultBlock struct {
	// (always "web_search_tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *WebSearchToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain WebSearchToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v WebSearchToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain WebSearchToolResultBlock
	if v.Type == "" {
		v.Type = "web_search_tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// WebFetchToolResultBlock - Web fetch server-tool result. (hand-modelled stand-in for TS WebFetchToolResultBlock)
type WebFetchToolResultBlock struct {
	// (always "web_fetch_tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *WebFetchToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain WebFetchToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v WebFetchToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain WebFetchToolResultBlock
	if v.Type == "" {
		v.Type = "web_fetch_tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CodeExecutionToolResultBlock - Code execution server-tool result. (hand-modelled stand-in for TS CodeExecutionToolResultBlock)
type CodeExecutionToolResultBlock struct {
	// (always "code_execution_tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CodeExecutionToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain CodeExecutionToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CodeExecutionToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain CodeExecutionToolResultBlock
	if v.Type == "" {
		v.Type = "code_execution_tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TextEditorCodeExecutionToolResultBlock - Text editor code execution server-tool result. (hand-modelled stand-in for TS TextEditorCodeExecutionToolResultBlock)
type TextEditorCodeExecutionToolResultBlock struct {
	// (always "text_editor_code_execution_tool_result")
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TextEditorCodeExecutionToolResultBlock) UnmarshalJSON(b []byte) error {
	type plain TextEditorCodeExecutionToolResultBlock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TextEditorCodeExecutionToolResultBlock) MarshalJSON() ([]byte, error) {
	type plain TextEditorCodeExecutionToolResultBlock
	if v.Type == "" {
		v.Type = "text_editor_code_execution_tool_result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UsageCacheCreation - inline object (field cache_creation of Usage)
type UsageCacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageCacheCreation) UnmarshalJSON(b []byte) error {
	type plain UsageCacheCreation
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageCacheCreation) MarshalJSON() ([]byte, error) {
	type plain UsageCacheCreation
	return marshalKeep(plain(v), v.Raw)
}

// UsageServerToolUse - inline object (field server_tool_use of Usage)
type UsageServerToolUse struct {
	WebSearchRequests int  `json:"web_search_requests"`
	WebFetchRequests  *int `json:"web_fetch_requests,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageServerToolUse) UnmarshalJSON(b []byte) error {
	type plain UsageServerToolUse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageServerToolUse) MarshalJSON() ([]byte, error) {
	type plain UsageServerToolUse
	return marshalKeep(plain(v), v.Raw)
}

// Usage - Token usage (BetaUsage with nulls removed, as the CLI reports it on result messages and assistant messages). (hand-modelled stand-in for TS NonNullableUsage)
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	// (optional, nullable)
	CacheCreation *UsageCacheCreation `json:"cache_creation,omitempty"`
	// (optional, nullable)
	ServerToolUse *UsageServerToolUse `json:"server_tool_use,omitempty"`
	// (optional, nullable)
	ServiceTier *string `json:"service_tier,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *Usage) UnmarshalJSON(b []byte) error {
	type plain Usage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v Usage) MarshalJSON() ([]byte, error) { type plain Usage; return marshalKeep(plain(v), v.Raw) }

// APIMessageContainer - inline object (field container of APIMessage)
type APIMessageContainer struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *APIMessageContainer) UnmarshalJSON(b []byte) error {
	type plain APIMessageContainer
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v APIMessageContainer) MarshalJSON() ([]byte, error) {
	type plain APIMessageContainer
	return marshalKeep(plain(v), v.Raw)
}

// APIMessage - Shaped like an Anthropic Messages API Message object (role "assistant") as the CLI emits it on `assistant.message`. (hand-modelled stand-in for TS BetaMessage)
type APIMessage struct {
	ID string `json:"id"`
	// (always "message")
	Type string `json:"type"`
	// (always "assistant")
	Role    string         `json:"role"`
	Model   string         `json:"model"`
	Content []ContentBlock `json:"content"`
	// (nullable)
	StopReason *string `json:"stop_reason"`
	// (nullable)
	StopSequence *string `json:"stop_sequence"`
	Usage        Usage   `json:"usage"`
	// (optional, nullable)
	Container *APIMessageContainer `json:"container,omitempty"`
	// (optional, nullable)
	ContextManagement map[string]any `json:"context_management,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *APIMessage) UnmarshalJSON(b []byte) error {
	type plain APIMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v APIMessage) MarshalJSON() ([]byte, error) {
	type plain APIMessage
	if v.Type == "" {
		v.Type = "message"
	}
	if v.Role == "" {
		v.Role = "assistant"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageStartEvent (hand-modelled stand-in for TS MessageStartEvent)
type MessageStartEvent struct {
	// (always "message_start")
	Type    string     `json:"type"`
	Message APIMessage `json:"message"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageStartEvent) UnmarshalJSON(b []byte) error {
	type plain MessageStartEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageStartEvent) MarshalJSON() ([]byte, error) {
	type plain MessageStartEvent
	if v.Type == "" {
		v.Type = "message_start"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageDeltaEventDelta - inline object (field delta of MessageDeltaEvent)
type MessageDeltaEventDelta struct {
	// (nullable)
	StopReason *string `json:"stop_reason"`
	// (nullable)
	StopSequence *string `json:"stop_sequence"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageDeltaEventDelta) UnmarshalJSON(b []byte) error {
	type plain MessageDeltaEventDelta
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageDeltaEventDelta) MarshalJSON() ([]byte, error) {
	type plain MessageDeltaEventDelta
	return marshalKeep(plain(v), v.Raw)
}

// MessageDeltaUsage - Usage delta carried by a message_delta stream event. (hand-modelled stand-in for TS MessageDeltaUsage)
type MessageDeltaUsage struct {
	// (optional, nullable)
	InputTokens  *int `json:"input_tokens,omitempty"`
	OutputTokens int  `json:"output_tokens"`
	// (optional, nullable)
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens,omitempty"`
	// (optional, nullable)
	CacheReadInputTokens *int `json:"cache_read_input_tokens,omitempty"`
	// (optional, nullable)
	ServerToolUse *UsageServerToolUse `json:"server_tool_use,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageDeltaUsage) UnmarshalJSON(b []byte) error {
	type plain MessageDeltaUsage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageDeltaUsage) MarshalJSON() ([]byte, error) {
	type plain MessageDeltaUsage
	return marshalKeep(plain(v), v.Raw)
}

// MessageDeltaEvent (hand-modelled stand-in for TS MessageDeltaEvent)
type MessageDeltaEvent struct {
	// (always "message_delta")
	Type  string                 `json:"type"`
	Delta MessageDeltaEventDelta `json:"delta"`
	Usage MessageDeltaUsage      `json:"usage"`
	// (optional, nullable)
	ContextManagement map[string]any `json:"context_management,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageDeltaEvent) UnmarshalJSON(b []byte) error {
	type plain MessageDeltaEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageDeltaEvent) MarshalJSON() ([]byte, error) {
	type plain MessageDeltaEvent
	if v.Type == "" {
		v.Type = "message_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageStopEvent (hand-modelled stand-in for TS MessageStopEvent)
type MessageStopEvent struct {
	// (always "message_stop")
	Type string `json:"type"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageStopEvent) UnmarshalJSON(b []byte) error {
	type plain MessageStopEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageStopEvent) MarshalJSON() ([]byte, error) {
	type plain MessageStopEvent
	if v.Type == "" {
		v.Type = "message_stop"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ContentBlockStartEvent (hand-modelled stand-in for TS ContentBlockStartEvent)
type ContentBlockStartEvent struct {
	// (always "content_block_start")
	Type         string       `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContentBlockStartEvent) UnmarshalJSON(b []byte) error {
	type plain ContentBlockStartEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContentBlockStartEvent) MarshalJSON() ([]byte, error) {
	type plain ContentBlockStartEvent
	if v.Type == "" {
		v.Type = "content_block_start"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TextDelta (hand-modelled stand-in for TS TextDelta)
type TextDelta struct {
	// (always "text_delta")
	Type string `json:"type"`
	Text string `json:"text"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TextDelta) UnmarshalJSON(b []byte) error {
	type plain TextDelta
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TextDelta) MarshalJSON() ([]byte, error) {
	type plain TextDelta
	if v.Type == "" {
		v.Type = "text_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// InputJSONDelta (hand-modelled stand-in for TS InputJSONDelta)
type InputJSONDelta struct {
	// (always "input_json_delta")
	Type        string `json:"type"`
	PartialJSON string `json:"partial_json"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InputJSONDelta) UnmarshalJSON(b []byte) error {
	type plain InputJSONDelta
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InputJSONDelta) MarshalJSON() ([]byte, error) {
	type plain InputJSONDelta
	if v.Type == "" {
		v.Type = "input_json_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CitationsDelta (hand-modelled stand-in for TS CitationsDelta)
type CitationsDelta struct {
	// (always "citations_delta")
	Type     string         `json:"type"`
	Citation map[string]any `json:"citation"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CitationsDelta) UnmarshalJSON(b []byte) error {
	type plain CitationsDelta
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CitationsDelta) MarshalJSON() ([]byte, error) {
	type plain CitationsDelta
	if v.Type == "" {
		v.Type = "citations_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ThinkingDelta (hand-modelled stand-in for TS ThinkingDelta)
type ThinkingDelta struct {
	// (always "thinking_delta")
	Type     string `json:"type"`
	Thinking string `json:"thinking"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ThinkingDelta) UnmarshalJSON(b []byte) error {
	type plain ThinkingDelta
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ThinkingDelta) MarshalJSON() ([]byte, error) {
	type plain ThinkingDelta
	if v.Type == "" {
		v.Type = "thinking_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SignatureDelta (hand-modelled stand-in for TS SignatureDelta)
type SignatureDelta struct {
	// (always "signature_delta")
	Type      string `json:"type"`
	Signature string `json:"signature"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SignatureDelta) UnmarshalJSON(b []byte) error {
	type plain SignatureDelta
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SignatureDelta) MarshalJSON() ([]byte, error) {
	type plain SignatureDelta
	if v.Type == "" {
		v.Type = "signature_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ContentBlockDeltaEvent (hand-modelled stand-in for TS ContentBlockDeltaEvent)
type ContentBlockDeltaEvent struct {
	// (always "content_block_delta")
	Type  string            `json:"type"`
	Index int               `json:"index"`
	Delta ContentBlockDelta `json:"delta"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContentBlockDeltaEvent) UnmarshalJSON(b []byte) error {
	type plain ContentBlockDeltaEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContentBlockDeltaEvent) MarshalJSON() ([]byte, error) {
	type plain ContentBlockDeltaEvent
	if v.Type == "" {
		v.Type = "content_block_delta"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ContentBlockStopEvent (hand-modelled stand-in for TS ContentBlockStopEvent)
type ContentBlockStopEvent struct {
	// (always "content_block_stop")
	Type  string `json:"type"`
	Index int    `json:"index"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContentBlockStopEvent) UnmarshalJSON(b []byte) error {
	type plain ContentBlockStopEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContentBlockStopEvent) MarshalJSON() ([]byte, error) {
	type plain ContentBlockStopEvent
	if v.Type == "" {
		v.Type = "content_block_stop"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CallToolResult - MCP tools/call result (CallToolResult from (hand-modelled stand-in for TS CallToolResult)
type CallToolResult struct {
	Content           []json.RawMessage `json:"content"`
	StructuredContent map[string]any    `json:"structuredContent,omitempty"`
	IsError           *bool             `json:"isError,omitempty"`
	XMeta             map[string]any    `json:"_meta,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CallToolResult) UnmarshalJSON(b []byte) error {
	type plain CallToolResult
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CallToolResult) MarshalJSON() ([]byte, error) {
	type plain CallToolResult
	return marshalKeep(plain(v), v.Raw)
}

// ConfigChangeHookInput (TS: ConfigChangeHookInput)
type ConfigChangeHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "ConfigChange")
	HookEventName string `json:"hook_event_name"`
	// (one of "user_settings", "project_settings", "local_settings", "policy_settings", "skills")
	Source   string `json:"source"`
	FilePath string `json:"file_path,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ConfigChangeHookInput) UnmarshalJSON(b []byte) error {
	type plain ConfigChangeHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ConfigChangeHookInput) MarshalJSON() ([]byte, error) {
	type plain ConfigChangeHookInput
	if v.HookEventName == "" {
		v.HookEventName = "ConfigChange"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ConfigScope is a string enum. Config scope for settings. (TS: ConfigScope)
type ConfigScope string

const (
	ConfigScopeLocal   ConfigScope = "local"
	ConfigScopeUser    ConfigScope = "user"
	ConfigScopeProject ConfigScope = "project"
)

// InterruptRequest - Interrupts the currently running conversation turn. (TS: SDKControlInterruptRequest, not exported (wire type))
type InterruptRequest struct {
	// (always "interrupt")
	Subtype string `json:"subtype"`
	// When true, the interrupt also cancels every uuid-stamped main-thread command still in the queue or already dequeued for the imminent turn but not yet reachable by the abort (the first-command prewait window) — the...
	CancelQueued *bool `json:"cancel_queued,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InterruptRequest) UnmarshalJSON(b []byte) error {
	type plain InterruptRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InterruptRequest) MarshalJSON() ([]byte, error) {
	type plain InterruptRequest
	if v.Subtype == "" {
		v.Subtype = "interrupt"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpServerProvenance - The MCP server serving this tool, for `mcp__*` tools: `name` is the server's config key (for `source: "sdk"`, exactly the name the SDK host registered in `sdkMcpServers` / `mcp_set_servers`; for any other source, the... (TS: McpServerProvenance)
type McpServerProvenance struct {
	Name string `json:"name"`
	// sdk | plugin | user | project | local | dynamic | managed | enterprise | claudeai | agent — an open set; treat unknown values as an unrecognized configured source, never as sdk.
	Source string `json:"source"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpServerProvenance) UnmarshalJSON(b []byte) error {
	type plain McpServerProvenance
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpServerProvenance) MarshalJSON() ([]byte, error) {
	type plain McpServerProvenance
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRuleValue (TS: PermissionRuleValue)
type PermissionRuleValue struct {
	ToolName    string `json:"toolName"`
	RuleContent string `json:"ruleContent,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRuleValue) UnmarshalJSON(b []byte) error {
	type plain PermissionRuleValue
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRuleValue) MarshalJSON() ([]byte, error) {
	type plain PermissionRuleValue
	return marshalKeep(plain(v), v.Raw)
}

// PermissionBehavior is a string enum. (TS: PermissionBehavior)
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination is a string enum. (TS: PermissionUpdateDestination)
type PermissionUpdateDestination string

const (
	PermissionUpdateDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionUpdateDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionUpdateDestinationSession         PermissionUpdateDestination = "session"
	PermissionUpdateDestinationCliArg          PermissionUpdateDestination = "cliArg"
)

// PermissionUpdateAddRules - variant of PermissionUpdate
type PermissionUpdateAddRules struct {
	// (always "addRules")
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionUpdateAddRules) UnmarshalJSON(b []byte) error {
	type plain PermissionUpdateAddRules
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionUpdateAddRules) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateAddRules
	if v.Type == "" {
		v.Type = "addRules"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionUpdateReplaceRules - variant of PermissionUpdate
type PermissionUpdateReplaceRules struct {
	// (always "replaceRules")
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionUpdateReplaceRules) UnmarshalJSON(b []byte) error {
	type plain PermissionUpdateReplaceRules
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionUpdateReplaceRules) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateReplaceRules
	if v.Type == "" {
		v.Type = "replaceRules"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionUpdateRemoveRules - variant of PermissionUpdate
type PermissionUpdateRemoveRules struct {
	// (always "removeRules")
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionUpdateRemoveRules) UnmarshalJSON(b []byte) error {
	type plain PermissionUpdateRemoveRules
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionUpdateRemoveRules) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateRemoveRules
	if v.Type == "" {
		v.Type = "removeRules"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionUpdateSetMode - variant of PermissionUpdate
type PermissionUpdateSetMode struct {
	// (always "setMode")
	Type        string                      `json:"type"`
	Mode        PermissionMode              `json:"mode"`
	Destination PermissionUpdateDestination `json:"destination"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionUpdateSetMode) UnmarshalJSON(b []byte) error {
	type plain PermissionUpdateSetMode
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionUpdateSetMode) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateSetMode
	if v.Type == "" {
		v.Type = "setMode"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionUpdateAddDirectories - variant of PermissionUpdate
type PermissionUpdateAddDirectories struct {
	// (always "addDirectories")
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionUpdateAddDirectories) UnmarshalJSON(b []byte) error {
	type plain PermissionUpdateAddDirectories
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionUpdateAddDirectories) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateAddDirectories
	if v.Type == "" {
		v.Type = "addDirectories"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionUpdateRemoveDirectories - variant of PermissionUpdate
type PermissionUpdateRemoveDirectories struct {
	// (always "removeDirectories")
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionUpdateRemoveDirectories) UnmarshalJSON(b []byte) error {
	type plain PermissionUpdateRemoveDirectories
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionUpdateRemoveDirectories) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateRemoveDirectories
	if v.Type == "" {
		v.Type = "removeDirectories"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CanUseToolRequestMatchedAskRule - inline object (field matched_ask_rule of CanUseToolRequest)
type CanUseToolRequestMatchedAskRule struct {
	Source      string `json:"source"`
	ToolName    string `json:"tool_name"`
	RuleContent string `json:"rule_content,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CanUseToolRequestMatchedAskRule) UnmarshalJSON(b []byte) error {
	type plain CanUseToolRequestMatchedAskRule
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CanUseToolRequestMatchedAskRule) MarshalJSON() ([]byte, error) {
	type plain CanUseToolRequestMatchedAskRule
	return marshalKeep(plain(v), v.Raw)
}

// CanUseToolRequest - Requests permission to use a tool with the given input. (TS: SDKControlPermissionRequest, not exported (wire type))
type CanUseToolRequest struct {
	// (always "can_use_tool")
	Subtype               string               `json:"subtype"`
	ToolName              string               `json:"tool_name"`
	MCPServer             *McpServerProvenance `json:"mcp_server,omitempty"`
	Input                 map[string]any       `json:"input"`
	PermissionSuggestions []PermissionUpdate   `json:"permission_suggestions,omitempty"`
	BlockedPath           string               `json:"blocked_path,omitempty"`
	// Human-readable reason the ask escalated, for the consent line of the host's dialog.
	DecisionReason string `json:"decision_reason,omitempty"`
	// Structured discriminator for why auto-mode escalated. (one of "rule", "mode", "subcommandResults", "permissionPromptTool", "hook", "asyncAgent", "sandboxOverride", "workingDir", "safetyCheck", "classifier", "other")
	DecisionReasonType string `json:"decision_reason_type,omitempty"`
	// Set when a safetyCheck is present anywhere in the decision reason (including nested inside subcommandResults for compound bash). false = at least one safety check requires manual approval (e.g.
	ClassifierApprovable *bool `json:"classifier_approvable,omitempty"`
	// True when the dialog must not offer the persistent "don't ask again" row for this ask: accepting it would write a whole-tool allow rule broader than the ask's own verb (PermissionAskDecision.suppressAlwaysAllowRule).
	SuppressAlwaysAllowRule *bool `json:"suppress_always_allow_rule,omitempty"`
	// True when the ask must not be approvable by a single stray keystroke (PermissionAskDecision.defaultToNo): a terminal-style prompt opens on its decline option and takes no digit shortcut.
	DefaultToNo *bool `json:"default_to_no,omitempty"`
	// Set when a user-configured ask RULE (permissions.ask) forced this prompt but the ask carries the tool's own decision_reason — the ask-rule substitution keeps the richer tool-minted ask, so the rule rides here instead...
	MatchedAskRule *CanUseToolRequestMatchedAskRule `json:"matched_ask_rule,omitempty"`
	Title          string                           `json:"title,omitempty"`
	DisplayName    string                           `json:"display_name,omitempty"`
	ToolUseID      string                           `json:"tool_use_id"`
	AgentID        string                           `json:"agent_id,omitempty"`
	Description    string                           `json:"description,omitempty"`
	// True when one-tap Approve/Deny must not be offered: the tool's approval card IS the user-interaction surface (Tool.requiresUserInteraction(): the user responds on the card itself), OR the pending ask is...
	RequiresUserInteraction *bool `json:"requires_user_interaction,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CanUseToolRequest) UnmarshalJSON(b []byte) error {
	type plain CanUseToolRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CanUseToolRequest) MarshalJSON() ([]byte, error) {
	type plain CanUseToolRequest
	if v.Subtype == "" {
		v.Subtype = "can_use_tool"
	}
	return marshalKeep(plain(v), v.Raw)
}

// HookCallbackMatcher - Configuration for matching and routing hook callbacks. (TS: SDKHookCallbackMatcher, not exported (wire type))
type HookCallbackMatcher struct {
	Matcher string `json:"matcher,omitempty"`
	// Opaque ids chosen by the client, one per hook function it registered for this matcher.
	HookCallbackIDs []string `json:"hookCallbackIds"`
	Timeout         *float64 `json:"timeout,omitempty"`
	// since 0.3.317 (not in 0.3.311)
	// What a failure of these callbacks does: the client answered the hook_callback request with an error, or with an answer that fails validation. (one of "continue", "block")
	OnFailure string `json:"onFailure,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *HookCallbackMatcher) UnmarshalJSON(b []byte) error {
	type plain HookCallbackMatcher
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v HookCallbackMatcher) MarshalJSON() ([]byte, error) {
	type plain HookCallbackMatcher
	return marshalKeep(plain(v), v.Raw)
}

// McpServerToolPolicy - Per-tool permission policy for an MCP server: carried on mcp_set_servers for remote (http/sse) servers, and on mcp_set_servers or initialize (sdkMcpServerConfigs) for host-registered sdk servers, where only... (TS: McpServerToolPolicy)
type McpServerToolPolicy struct {
	Name string `json:"name"`
	// (one of "always_allow", "always_ask", "always_deny")
	PermissionPolicy string `json:"permission_policy,omitempty"`
	// Org admin's per-tool ceiling. (one of "allow", "ask", "blocked")
	OrgMaxPermission string `json:"org_max_permission,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpServerToolPolicy) UnmarshalJSON(b []byte) error {
	type plain McpServerToolPolicy
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpServerToolPolicy) MarshalJSON() ([]byte, error) {
	type plain McpServerToolPolicy
	return marshalKeep(plain(v), v.Raw)
}

// InitializeRequestSdkMCPServerConfig - inline object (field sdkMcpServerConfig of InitializeRequest)
type InitializeRequestSdkMCPServerConfig struct {
	// Per-server tool-call timeout in milliseconds.
	Timeout *float64 `json:"timeout,omitempty"`
	// Per-tool permission policy the host declares for this server's tools (not the tool definitions passed to createSdkMcpServer). always_ask: the CLI asks the host (canUseTool or the permission prompt tool) before each...
	Tools []McpServerToolPolicy `json:"tools,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InitializeRequestSdkMCPServerConfig) UnmarshalJSON(b []byte) error {
	type plain InitializeRequestSdkMCPServerConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InitializeRequestSdkMCPServerConfig) MarshalJSON() ([]byte, error) {
	type plain InitializeRequestSdkMCPServerConfig
	return marshalKeep(plain(v), v.Raw)
}

// InitializeRequestSdkMCPServerManifest - inline object (field sdkMcpServerManifest of InitializeRequest)
type InitializeRequestSdkMCPServerManifest struct {
	// The server's verbatim JSON-RPC `initialize` result object (protocolVersion, capabilities, serverInfo, instructions, ...), exactly as the in-process server produced it when the host initialized it with no client...
	InitializeResult map[string]any `json:"initializeResult"`
	// The server's verbatim JSON-RPC `tools/list` result object.
	ToolsListResult map[string]any `json:"toolsListResult,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InitializeRequestSdkMCPServerManifest) UnmarshalJSON(b []byte) error {
	type plain InitializeRequestSdkMCPServerManifest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InitializeRequestSdkMCPServerManifest) MarshalJSON() ([]byte, error) {
	type plain InitializeRequestSdkMCPServerManifest
	return marshalKeep(plain(v), v.Raw)
}

// PluginConfig - Configuration for loading a plugin. (TS: SdkPluginConfig)
type PluginConfig struct {
	// Plugin type. (always "local")
	Type string `json:"type"`
	// Absolute or relative path to the plugin directory
	Path string `json:"path"`
	// When true, the engine loads skills/hooks/agents/commands from this plugin but does NOT read its .mcp.json or manifest mcpServers.
	SkipMCPDiscovery *bool `json:"skipMcpDiscovery,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PluginConfig) UnmarshalJSON(b []byte) error {
	type plain PluginConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PluginConfig) MarshalJSON() ([]byte, error) {
	type plain PluginConfig
	if v.Type == "" {
		v.Type = "local"
	}
	return marshalKeep(plain(v), v.Raw)
}

// InitializeRequest - Initializes the SDK session with hooks, MCP servers, and agent configuration. (TS: SDKControlInitializeRequest, not exported (wire type))
type InitializeRequest struct {
	// (always "initialize")
	Subtype       string                              `json:"subtype"`
	Hooks         map[HookEvent][]HookCallbackMatcher `json:"hooks,omitempty"`
	SdkMCPServers []string                            `json:"sdkMcpServers,omitempty"`
	// Settings for the SDK-hosted MCP servers named in sdkMcpServers, keyed by server name.
	SdkMCPServerConfigs map[string]InitializeRequestSdkMCPServerConfig `json:"sdkMcpServerConfigs,omitempty"`
	// Optional, keyed by sdk server name (each key should also appear in sdkMcpServers; other keys are ignored).
	SdkMCPServerManifests map[string]InitializeRequestSdkMCPServerManifest `json:"sdkMcpServerManifests,omitempty"`
	// The host's statement that it composed this request's sdkMcpServerManifests for this very send, from its servers' current state. (always "host_per_send")
	SdkMCPServerManifestsOrigin string         `json:"sdkMcpServerManifestsOrigin,omitempty"`
	JSONSchema                  map[string]any `json:"jsonSchema,omitempty"`
	SystemPrompt                []string       `json:"systemPrompt,omitempty"`
	AppendSystemPrompt          string         `json:"appendSystemPrompt,omitempty"`
	// Record the conversation's system prompt once and reuse it verbatim on every later request and resume.
	SystemPromptSnapshot *bool `json:"systemPromptSnapshot,omitempty"`
	// Custom workflow body for the plan-mode system reminder.
	PlanModeInstructions string `json:"planModeInstructions,omitempty"`
	// Map of tool-name aliases applied before name resolution.
	ToolAliases map[string]string `json:"toolAliases,omitempty"`
	// When true, omit per-user dynamic sections (working directory, auto-memory path) from the cached system prompt and re-inject them as the first user message.
	ExcludeDynamicSections *bool                      `json:"excludeDynamicSections,omitempty"`
	Agents                 map[string]AgentDefinition `json:"agents,omitempty"`
	// Custom session title.
	Title string `json:"title,omitempty"`
	// When provided, only skills whose names match an entry are loaded into the main session system prompt, matching the exact canonical name (e.g.
	Skills                 []string `json:"skills,omitempty"`
	PromptSuggestions      *bool    `json:"promptSuggestions,omitempty"`
	AgentProgressSummaries *bool    `json:"agentProgressSummaries,omitempty"`
	ForwardSubagentText    *bool    `json:"forwardSubagentText,omitempty"`
	// Dialog kinds (request_user_dialog `dialog_kind` values) this consumer's onUserDialog can actually render.
	SupportedDialogKinds []string `json:"supportedDialogKinds,omitempty"`
	// Declares that this consumer renders a per-task stop control wired to the `stop_task` control request, so the user can stop an individual background task.
	PerTaskStopAffordance *bool `json:"perTaskStopAffordance,omitempty"`
	// Plugins to load for the session, in the same shape as the SDK `plugins` option: the stdin form of one --plugin-dir flag per entry (--plugin-dir-no-mcp when skipMcpDiscovery is set), so the launch command line does...
	Plugins []PluginConfig `json:"plugins,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InitializeRequest) UnmarshalJSON(b []byte) error {
	type plain InitializeRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InitializeRequest) MarshalJSON() ([]byte, error) {
	type plain InitializeRequest
	if v.Subtype == "" {
		v.Subtype = "initialize"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SetPermissionModeRequest - Sets the permission mode for tool execution handling. (TS: SDKControlSetPermissionModeRequest, not exported (wire type))
type SetPermissionModeRequest struct {
	// (always "set_permission_mode")
	Subtype string `json:"subtype"`
	// Permission mode for controlling how tool executions are handled.
	Mode PermissionMode `json:"mode"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetPermissionModeRequest) UnmarshalJSON(b []byte) error {
	type plain SetPermissionModeRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetPermissionModeRequest) MarshalJSON() ([]byte, error) {
	type plain SetPermissionModeRequest
	if v.Subtype == "" {
		v.Subtype = "set_permission_mode"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SetModelRequest - Sets the model to use for subsequent conversation turns. (TS: SDKControlSetModelRequest, not exported (wire type))
type SetModelRequest struct {
	// (always "set_model")
	Subtype string `json:"subtype"`
	// Model to switch to. (optional, nullable)
	Model *string `json:"model,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetModelRequest) UnmarshalJSON(b []byte) error {
	type plain SetModelRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetModelRequest) MarshalJSON() ([]byte, error) {
	type plain SetModelRequest
	if v.Subtype == "" {
		v.Subtype = "set_model"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SetMaxThinkingTokensRequest - Sets the maximum number of thinking tokens for extended thinking. (TS: SDKControlSetMaxThinkingTokensRequest, not exported (wire type))
type SetMaxThinkingTokensRequest struct {
	// (always "set_max_thinking_tokens")
	Subtype string `json:"subtype"`
	// (optional, nullable)
	MaxThinkingTokens *int `json:"max_thinking_tokens,omitempty"`
	// (one of "summarized", "omitted", "highlights"; optional, nullable)
	ThinkingDisplay *string `json:"thinking_display,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetMaxThinkingTokensRequest) UnmarshalJSON(b []byte) error {
	type plain SetMaxThinkingTokensRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetMaxThinkingTokensRequest) MarshalJSON() ([]byte, error) {
	type plain SetMaxThinkingTokensRequest
	if v.Subtype == "" {
		v.Subtype = "set_max_thinking_tokens"
	}
	return marshalKeep(plain(v), v.Raw)
}

// RenameSessionRequest - Sets the user-facing title for the current session. (TS: SDKControlRenameSessionRequest, not exported (wire type))
type RenameSessionRequest struct {
	// (always "rename_session")
	Subtype string `json:"subtype"`
	Title   string `json:"title"`
	// Who chose the title: 'remote' (the default) for a rename made on claude.ai and relayed to this process, 'host' for one the user made in the hosting application (an IDE), which the CLI counts as a user rename. (one of "remote", "host")
	Source string `json:"source,omitempty"`
	// The session the title is for.
	SessionID string `json:"session_id,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RenameSessionRequest) UnmarshalJSON(b []byte) error {
	type plain RenameSessionRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RenameSessionRequest) MarshalJSON() ([]byte, error) {
	type plain RenameSessionRequest
	if v.Subtype == "" {
		v.Subtype = "rename_session"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SetColorRequest - Sets the session accent color. (TS: SDKControlSetColorRequest, not exported (wire type))
type SetColorRequest struct {
	// (always "set_color")
	Subtype string `json:"subtype"`
	Color   string `json:"color"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetColorRequest) UnmarshalJSON(b []byte) error {
	type plain SetColorRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetColorRequest) MarshalJSON() ([]byte, error) {
	type plain SetColorRequest
	if v.Subtype == "" {
		v.Subtype = "set_color"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpStatusRequest - Requests the current status of all MCP server connections. (TS: SDKControlMcpStatusRequest, not exported (wire type))
type McpStatusRequest struct {
	// (always "mcp_status")
	Subtype string `json:"subtype"`
	// When true and the CLI writes to the session through CCR (a cloud session's worker, or a Remote Control worker), the success reply is published only to the clients attached at that moment, as a control_response marked...
	LiveReply *bool `json:"liveReply,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpStatusRequest) UnmarshalJSON(b []byte) error {
	type plain McpStatusRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpStatusRequest) MarshalJSON() ([]byte, error) {
	type plain McpStatusRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_status"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageRequest - Requests a breakdown of current context window usage by category. (TS: SDKControlGetContextUsageRequest, not exported (wire type))
type GetContextUsageRequest struct {
	// (always "get_context_usage")
	Subtype string `json:"subtype"`
	// 'full' counts each category with the token-count API; 'summary' answers from the last response's usage and local estimates without the per-category token-count calls. (one of "summary", "full")
	Detail string `json:"detail,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageRequest) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageRequest) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageRequest
	if v.Subtype == "" {
		v.Subtype = "get_context_usage"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetSessionCostRequest - Requests the formatted session cost summary (the same text /usage prints in non-interactive mode). (TS: SDKControlGetSessionCostRequest, not exported (wire type))
type GetSessionCostRequest struct {
	// (always "get_session_cost")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetSessionCostRequest) UnmarshalJSON(b []byte) error {
	type plain GetSessionCostRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetSessionCostRequest) MarshalJSON() ([]byte, error) {
	type plain GetSessionCostRequest
	if v.Subtype == "" {
		v.Subtype = "get_session_cost"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ListModelsRequest - Requests the worker's selectable model catalog. (TS: SDKControlListModelsRequest, not exported (wire type))
type ListModelsRequest struct {
	// (always "list_models")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ListModelsRequest) UnmarshalJSON(b []byte) error {
	type plain ListModelsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ListModelsRequest) MarshalJSON() ([]byte, error) {
	type plain ListModelsRequest
	if v.Subtype == "" {
		v.Subtype = "list_models"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageRequest - Requests the structured /usage data: session cost/usage totals plus claude.ai plan rate-limit utilization when available. (TS: SDKControlGetUsageRequest, not exported (wire type))
type GetUsageRequest struct {
	// (always "get_usage")
	Subtype string `json:"subtype"`
	// Skip the scan of local transcripts that produces the response's behaviors section (it is null in the answer).
	SkipBehaviors *bool `json:"skip_behaviors,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageRequest) UnmarshalJSON(b []byte) error {
	type plain GetUsageRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageRequest) MarshalJSON() ([]byte, error) {
	type plain GetUsageRequest
	if v.Subtype == "" {
		v.Subtype = "get_usage"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetBinaryVersionRequest - Requests the responder's CLI binary version. (TS: SDKControlGetBinaryVersionRequest, not exported (wire type))
type GetBinaryVersionRequest struct {
	// (always "get_binary_version")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetBinaryVersionRequest) UnmarshalJSON(b []byte) error {
	type plain GetBinaryVersionRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetBinaryVersionRequest) MarshalJSON() ([]byte, error) {
	type plain GetBinaryVersionRequest
	if v.Subtype == "" {
		v.Subtype = "get_binary_version"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpCallRequestInputFile - inline object (field input_file of McpCallRequest)
type McpCallRequestInputFile struct {
	// Handle referenced from arguments as "{{in:NAME}}".
	Name string `json:"name"`
	// Synced-file lane row to stage, e.g. /working/.cowork/originals/a.docx.
	LanePath string `json:"lane_path"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpCallRequestInputFile) UnmarshalJSON(b []byte) error {
	type plain McpCallRequestInputFile
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpCallRequestInputFile) MarshalJSON() ([]byte, error) {
	type plain McpCallRequestInputFile
	return marshalKeep(plain(v), v.Raw)
}

// McpCallRequestOutputFile - inline object (field output_file of McpCallRequest)
type McpCallRequestOutputFile struct {
	// Handle referenced from arguments as "{{out:NAME}}" — the tool is told (via the substituted path) where to write; the worker collects from exactly that path.
	Name string `json:"name"`
	// Synced-file lane row to write, e.g. /working/report.cd.
	LanePath string `json:"lane_path"`
	// Opaque lane etag the output row must still carry for the write to land (CAS).
	IfMatch string `json:"if_match,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpCallRequestOutputFile) UnmarshalJSON(b []byte) error {
	type plain McpCallRequestOutputFile
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpCallRequestOutputFile) MarshalJSON() ([]byte, error) {
	type plain McpCallRequestOutputFile
	return marshalKeep(plain(v), v.Raw)
}

// McpCallRequest - Invokes an MCP tool via the subprocess MCP client without a model turn. (TS: SDKControlMcpCallRequest, not exported (wire type))
type McpCallRequest struct {
	// (always "mcp_call")
	Subtype string `json:"subtype"`
	// Fully-qualified MCP tool name, e.g. mcp__server__tool_name.
	Tool string `json:"tool"`
	// Tool arguments.
	Arguments map[string]any `json:"arguments,omitempty"`
	// RFC3339 deadline, REQUIRED when output_files are declared (a stale buffered drain must not overwrite rows written since).
	ExpiresAt string `json:"expires_at,omitempty"`
	// Tool-execution timeout (staging and collection have their own transport timeouts).
	TimeoutMs *float64 `json:"timeout_ms,omitempty"`
	// The caller hands the result to a program (a host that renders it, a script), not to a model, and bounds it itself. content is then the server's own content blocks as MCP defines them (the MCP SDK drops members it...
	ResultNotForModel *bool `json:"result_not_for_model,omitempty"`
	// Declaring input_files or output_files makes this a STAGED call: rows are fetched from the synced-file lane into a private per-request temp dir the WORKER chooses (random, per-UID — the caller never sees or computes...
	InputFiles  []McpCallRequestInputFile  `json:"input_files,omitempty"`
	OutputFiles []McpCallRequestOutputFile `json:"output_files,omitempty"`
	// since 0.3.312-0.3.325 (not in 0.3.311)
	// The call is someone else's (a page a host renders), so it is decided as the model's own call to this tool is, except that a slash command's allowed-tools and connector card answers never allow it, even during a turn:...
	CheckPermissions *bool `json:"check_permissions,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpCallRequest) UnmarshalJSON(b []byte) error {
	type plain McpCallRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpCallRequest) MarshalJSON() ([]byte, error) {
	type plain McpCallRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_call"
	}
	return marshalKeep(plain(v), v.Raw)
}

// FileSuggestionsRequest - Requests at-mention file autocomplete suggestions for a partial path prefix. (TS: SDKControlFileSuggestionsRequest, not exported (wire type))
type FileSuggestionsRequest struct {
	// (always "file_suggestions")
	Subtype string `json:"subtype"`
	Query   string `json:"query"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *FileSuggestionsRequest) UnmarshalJSON(b []byte) error {
	type plain FileSuggestionsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v FileSuggestionsRequest) MarshalJSON() ([]byte, error) {
	type plain FileSuggestionsRequest
	if v.Subtype == "" {
		v.Subtype = "file_suggestions"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PreToolUseHookInput (TS: PreToolUseHookInput)
type PreToolUseHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PreToolUse")
	HookEventName string               `json:"hook_event_name"`
	ToolName      string               `json:"tool_name"`
	ToolInput     json.RawMessage      `json:"tool_input"`
	ToolUseID     string               `json:"tool_use_id"`
	MCPServer     *McpServerProvenance `json:"mcp_server,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PreToolUseHookInput) UnmarshalJSON(b []byte) error {
	type plain PreToolUseHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PreToolUseHookInput) MarshalJSON() ([]byte, error) {
	type plain PreToolUseHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PreToolUse"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostToolUseHookInput (TS: PostToolUseHookInput)
type PostToolUseHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PostToolUse")
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolResponse  json.RawMessage `json:"tool_response"`
	ToolUseID     string          `json:"tool_use_id"`
	// Tool execution time in milliseconds.
	DurationMs *float64             `json:"duration_ms,omitempty"`
	MCPServer  *McpServerProvenance `json:"mcp_server,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolUseHookInput) UnmarshalJSON(b []byte) error {
	type plain PostToolUseHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolUseHookInput) MarshalJSON() ([]byte, error) {
	type plain PostToolUseHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PostToolUse"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostToolUseFailureHookInput (TS: PostToolUseFailureHookInput)
type PostToolUseFailureHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PostToolUseFailure")
	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolUseID     string          `json:"tool_use_id"`
	Error         string          `json:"error"`
	IsInterrupt   *bool           `json:"is_interrupt,omitempty"`
	// Tool execution time in milliseconds.
	DurationMs *float64             `json:"duration_ms,omitempty"`
	MCPServer  *McpServerProvenance `json:"mcp_server,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolUseFailureHookInput) UnmarshalJSON(b []byte) error {
	type plain PostToolUseFailureHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolUseFailureHookInput) MarshalJSON() ([]byte, error) {
	type plain PostToolUseFailureHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PostToolUseFailure"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostToolBatchToolCall (TS: PostToolBatchToolCall)
type PostToolBatchToolCall struct {
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolUseID    string          `json:"tool_use_id"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolBatchToolCall) UnmarshalJSON(b []byte) error {
	type plain PostToolBatchToolCall
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolBatchToolCall) MarshalJSON() ([]byte, error) {
	type plain PostToolBatchToolCall
	return marshalKeep(plain(v), v.Raw)
}

// PostToolBatchHookInput - Hook input for the PostToolBatch event. (TS: PostToolBatchHookInput)
type PostToolBatchHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PostToolBatch")
	HookEventName string                  `json:"hook_event_name"`
	ToolCalls     []PostToolBatchToolCall `json:"tool_calls"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolBatchHookInput) UnmarshalJSON(b []byte) error {
	type plain PostToolBatchHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolBatchHookInput) MarshalJSON() ([]byte, error) {
	type plain PostToolBatchHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PostToolBatch"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionDeniedHookInput (TS: PermissionDeniedHookInput)
type PermissionDeniedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PermissionDenied")
	HookEventName string               `json:"hook_event_name"`
	ToolName      string               `json:"tool_name"`
	ToolInput     json.RawMessage      `json:"tool_input"`
	ToolUseID     string               `json:"tool_use_id"`
	Reason        string               `json:"reason"`
	MCPServer     *McpServerProvenance `json:"mcp_server,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionDeniedHookInput) UnmarshalJSON(b []byte) error {
	type plain PermissionDeniedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionDeniedHookInput) MarshalJSON() ([]byte, error) {
	type plain PermissionDeniedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PermissionDenied"
	}
	return marshalKeep(plain(v), v.Raw)
}

// NotificationHookInput (TS: NotificationHookInput)
type NotificationHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "Notification")
	HookEventName    string `json:"hook_event_name"`
	Message          string `json:"message"`
	Title            string `json:"title,omitempty"`
	NotificationType string `json:"notification_type"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *NotificationHookInput) UnmarshalJSON(b []byte) error {
	type plain NotificationHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v NotificationHookInput) MarshalJSON() ([]byte, error) {
	type plain NotificationHookInput
	if v.HookEventName == "" {
		v.HookEventName = "Notification"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserPromptSubmitHookInput (TS: UserPromptSubmitHookInput)
type UserPromptSubmitHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "UserPromptSubmit")
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
	// Who authored/injected the prompt: `user` = submitted from the interactive composer, `sdk` = non-interactive entrypoint (`-p` / Agent SDK), `loop_wakeup` = dynamic /loop wakeup, `schedule_wakeup` = scheduled-task fire... (one of "user", "sdk", "system", "loop_wakeup", "schedule_wakeup", "poll_event")
	Source       string `json:"source,omitempty"`
	SessionTitle string `json:"session_title,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserPromptSubmitHookInput) UnmarshalJSON(b []byte) error {
	type plain UserPromptSubmitHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserPromptSubmitHookInput) MarshalJSON() ([]byte, error) {
	type plain UserPromptSubmitHookInput
	if v.HookEventName == "" {
		v.HookEventName = "UserPromptSubmit"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserPromptExpansionHookInput (TS: UserPromptExpansionHookInput)
type UserPromptExpansionHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "UserPromptExpansion")
	HookEventName string `json:"hook_event_name"`
	// (one of "slash_command", "mcp_prompt")
	ExpansionType string `json:"expansion_type"`
	CommandName   string `json:"command_name"`
	CommandArgs   string `json:"command_args"`
	CommandSource string `json:"command_source,omitempty"`
	Prompt        string `json:"prompt"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserPromptExpansionHookInput) UnmarshalJSON(b []byte) error {
	type plain UserPromptExpansionHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserPromptExpansionHookInput) MarshalJSON() ([]byte, error) {
	type plain UserPromptExpansionHookInput
	if v.HookEventName == "" {
		v.HookEventName = "UserPromptExpansion"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SessionStartHookInput (TS: SessionStartHookInput)
type SessionStartHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID   string `json:"agent_id,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "SessionStart")
	HookEventName string `json:"hook_event_name"`
	// (one of "startup", "resume", "clear", "compact", "fork")
	Source       string `json:"source"`
	Model        string `json:"model,omitempty"`
	SessionTitle string `json:"session_title,omitempty"`
	// resume/fork: seconds since the resumed transcript's last assistant response
	SecondsSinceLastResponse *float64 `json:"seconds_since_last_response,omitempty"`
	// resume/fork: the resumed transcript's last response input + cache_read + cache_creation + output tokens (for a server-side tool loop, its last iteration's window, not the summed totals)
	ContextTokens *int `json:"context_tokens,omitempty"`
	// resume/fork: seconds_since_last_response exceeds the prompt-cache TTL, so the first request re-caches context_tokens
	PromptCacheLikelyExpired *bool `json:"prompt_cache_likely_expired,omitempty"`
	// resume/fork: estimated cost of re-caching context_tokens on the session model — the managed modelPricing when set, otherwise list price; excludes the response
	EstimatedCacheWriteUSD *float64 `json:"estimated_cache_write_usd,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionStartHookInput) UnmarshalJSON(b []byte) error {
	type plain SessionStartHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionStartHookInput) MarshalJSON() ([]byte, error) {
	type plain SessionStartHookInput
	if v.HookEventName == "" {
		v.HookEventName = "SessionStart"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ExitReason is a string enum. (TS: ExitReason)
type ExitReason string

const (
	ExitReasonClear           ExitReason = "clear"
	ExitReasonResume          ExitReason = "resume"
	ExitReasonLogout          ExitReason = "logout"
	ExitReasonPromptInputExit ExitReason = "prompt_input_exit"
	ExitReasonOther           ExitReason = "other"
)

// SessionEndHookInput (TS: SessionEndHookInput)
type SessionEndHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "SessionEnd")
	HookEventName string     `json:"hook_event_name"`
	Reason        ExitReason `json:"reason"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionEndHookInput) UnmarshalJSON(b []byte) error {
	type plain SessionEndHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionEndHookInput) MarshalJSON() ([]byte, error) {
	type plain SessionEndHookInput
	if v.HookEventName == "" {
		v.HookEventName = "SessionEnd"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SessionCronSummary (TS: SessionCronSummary)
type SessionCronSummary struct {
	ID string `json:"id"`
	// Cron expression, e.g.
	Schedule string `json:"schedule"`
	// False for one-shot wakeups whose cron field encodes a single fire time; true for tasks that re-fire on every match.
	Recurring bool `json:"recurring"`
	// Prompt text submitted when the cron fires.
	Prompt string `json:"prompt"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionCronSummary) UnmarshalJSON(b []byte) error {
	type plain SessionCronSummary
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionCronSummary) MarshalJSON() ([]byte, error) {
	type plain SessionCronSummary
	return marshalKeep(plain(v), v.Raw)
}

// StopHookInput (TS: StopHookInput)
type StopHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "Stop")
	HookEventName  string `json:"hook_event_name"`
	StopHookActive bool   `json:"stop_hook_active"`
	// Text content of the last assistant message before stopping.
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
	// In-flight background work (running/pending + backgrounded) registered in this session.
	BackgroundTasks []BackgroundTaskSummary `json:"background_tasks,omitempty"`
	// Session-scoped cron tasks (CronCreate, ScheduleWakeup, /loop) that will wake this session later.
	SessionCrons []SessionCronSummary `json:"session_crons,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *StopHookInput) UnmarshalJSON(b []byte) error {
	type plain StopHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v StopHookInput) MarshalJSON() ([]byte, error) {
	type plain StopHookInput
	if v.HookEventName == "" {
		v.HookEventName = "Stop"
	}
	return marshalKeep(plain(v), v.Raw)
}

// AssistantMessageError is a string enum. (TS: SDKAssistantMessageError)
type AssistantMessageError string

const (
	AssistantMessageErrorAuthenticationFailed AssistantMessageError = "authentication_failed"
	AssistantMessageErrorOauthOrgNotAllowed   AssistantMessageError = "oauth_org_not_allowed"
	AssistantMessageErrorAccountOnHold        AssistantMessageError = "account_on_hold"
	AssistantMessageErrorVerificationRequired AssistantMessageError = "verification_required"
	AssistantMessageErrorBillingError         AssistantMessageError = "billing_error"
	AssistantMessageErrorRateLimit            AssistantMessageError = "rate_limit"
	AssistantMessageErrorOverloaded           AssistantMessageError = "overloaded"
	AssistantMessageErrorInvalidRequest       AssistantMessageError = "invalid_request"
	AssistantMessageErrorModelNotFound        AssistantMessageError = "model_not_found"
	AssistantMessageErrorServerError          AssistantMessageError = "server_error"
	AssistantMessageErrorUnknown              AssistantMessageError = "unknown"
	AssistantMessageErrorMaxOutputTokens      AssistantMessageError = "max_output_tokens"
	AssistantMessageErrorCloudCredentialError AssistantMessageError = "cloud_credential_error"
)

// StopFailureHookInput (TS: StopFailureHookInput)
type StopFailureHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "StopFailure")
	HookEventName        string                `json:"hook_event_name"`
	Error                AssistantMessageError `json:"error"`
	ErrorDetails         string                `json:"error_details,omitempty"`
	LastAssistantMessage string                `json:"last_assistant_message,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *StopFailureHookInput) UnmarshalJSON(b []byte) error {
	type plain StopFailureHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v StopFailureHookInput) MarshalJSON() ([]byte, error) {
	type plain StopFailureHookInput
	if v.HookEventName == "" {
		v.HookEventName = "StopFailure"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SubagentStartHookInput (TS: SubagentStartHookInput)
type SubagentStartHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	AgentID        string `json:"agent_id"`
	AgentType      string `json:"agent_type"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "SubagentStart")
	HookEventName string `json:"hook_event_name"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SubagentStartHookInput) UnmarshalJSON(b []byte) error {
	type plain SubagentStartHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SubagentStartHookInput) MarshalJSON() ([]byte, error) {
	type plain SubagentStartHookInput
	if v.HookEventName == "" {
		v.HookEventName = "SubagentStart"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SubagentStopHookInput (TS: SubagentStopHookInput)
type SubagentStopHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	AgentID        string `json:"agent_id"`
	AgentType      string `json:"agent_type"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "SubagentStop")
	HookEventName       string `json:"hook_event_name"`
	StopHookActive      bool   `json:"stop_hook_active"`
	AgentTranscriptPath string `json:"agent_transcript_path"`
	// Text content of the last assistant message before stopping.
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
	// In-flight background work (running/pending + backgrounded) registered in this session.
	BackgroundTasks []BackgroundTaskSummary `json:"background_tasks,omitempty"`
	// Session-scoped cron tasks (CronCreate, ScheduleWakeup, /loop) that will wake this session later.
	SessionCrons []SessionCronSummary `json:"session_crons,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SubagentStopHookInput) UnmarshalJSON(b []byte) error {
	type plain SubagentStopHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SubagentStopHookInput) MarshalJSON() ([]byte, error) {
	type plain SubagentStopHookInput
	if v.HookEventName == "" {
		v.HookEventName = "SubagentStop"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PreCompactHookInput (TS: PreCompactHookInput)
type PreCompactHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PreCompact")
	HookEventName string `json:"hook_event_name"`
	// (one of "manual", "auto")
	Trigger string `json:"trigger"`
	// (nullable)
	CustomInstructions *string `json:"custom_instructions"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PreCompactHookInput) UnmarshalJSON(b []byte) error {
	type plain PreCompactHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PreCompactHookInput) MarshalJSON() ([]byte, error) {
	type plain PreCompactHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PreCompact"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostCompactHookInput (TS: PostCompactHookInput)
type PostCompactHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PostCompact")
	HookEventName string `json:"hook_event_name"`
	// (one of "manual", "auto")
	Trigger string `json:"trigger"`
	// The conversation summary produced by compaction
	CompactSummary string `json:"compact_summary"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostCompactHookInput) UnmarshalJSON(b []byte) error {
	type plain PostCompactHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostCompactHookInput) MarshalJSON() ([]byte, error) {
	type plain PostCompactHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PostCompact"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PreModelSwitchHookInput (TS: PreModelSwitchHookInput)
type PreModelSwitchHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PreModelSwitch")
	HookEventName string `json:"hook_event_name"`
	// Resolved model id the session was running before the switch
	FromModel string `json:"from_model"`
	// Resolved model id the session runs after the switch
	ToModel string `json:"to_model"`
	// What was asked for (alias such as "opus", a full id, or null for "default") (nullable)
	RequestedModel *string `json:"requested_model"`
	// command: /model <name>, the /config Model row, or enabling fast mode when that promotes the model; picker: an interactive model picker; sdk: headless set_model (SDK, Remote Control, IDE) (one of "command", "picker", "sdk")
	Source string `json:"source"`
	// Prompt tokens the next request re-sends: the last main-thread response's input + cache_read + cache_creation + output tokens (0 before the first response; for a server-side tool loop, its last iteration's window, not...
	ContextTokens int `json:"context_tokens"`
	// Whether the current model's prompt cache is likely still warm (a switch then forfeits it)
	PromptCacheWarm bool `json:"prompt_cache_warm"`
	// (one of "5m", "1h")
	CacheTtl string `json:"cache_ttl"`
	// Estimated cost of re-caching context_tokens on to_model at its cache-write rate — the managed modelPricing when set, otherwise list price; excludes the response
	EstimatedCacheWriteUSD float64 `json:"estimated_cache_write_usd"`
	// configured: priced at the managed modelPricing setting; catalog: list price; default: to_model unknown, the default tier was assumed (one of "configured", "catalog", "default")
	Pricing string `json:"pricing"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PreModelSwitchHookInput) UnmarshalJSON(b []byte) error {
	type plain PreModelSwitchHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PreModelSwitchHookInput) MarshalJSON() ([]byte, error) {
	type plain PreModelSwitchHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PreModelSwitch"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostModelSwitchHookInput (TS: PostModelSwitchHookInput)
type PostModelSwitchHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PostModelSwitch")
	HookEventName string `json:"hook_event_name"`
	// Resolved model id the session was running before the switch
	FromModel string `json:"from_model"`
	// Resolved model id the session runs after the switch
	ToModel string `json:"to_model"`
	// What was asked for (alias such as "opus", a full id, or null for "default") (nullable)
	RequestedModel *string `json:"requested_model"`
	// command: /model <name>, the /config Model row, or enabling fast mode when that promotes the model; picker: an interactive model picker; sdk: headless set_model (SDK, Remote Control, IDE); auto: automatic fallback or... (one of "command", "picker", "sdk", "auto", "resume")
	Source string `json:"source"`
	// Prompt tokens the next request re-sends: the last main-thread response's input + cache_read + cache_creation + output tokens (0 before the first response; for a server-side tool loop, its last iteration's window, not...
	ContextTokens int `json:"context_tokens"`
	// Whether the current model's prompt cache is likely still warm (a switch then forfeits it)
	PromptCacheWarm bool `json:"prompt_cache_warm"`
	// (one of "5m", "1h")
	CacheTtl string `json:"cache_ttl"`
	// Estimated cost of re-caching context_tokens on to_model at its cache-write rate — the managed modelPricing when set, otherwise list price; excludes the response
	EstimatedCacheWriteUSD float64 `json:"estimated_cache_write_usd"`
	// configured: priced at the managed modelPricing setting; catalog: list price; default: to_model unknown, the default tier was assumed (one of "configured", "catalog", "default")
	Pricing string `json:"pricing"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostModelSwitchHookInput) UnmarshalJSON(b []byte) error {
	type plain PostModelSwitchHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostModelSwitchHookInput) MarshalJSON() ([]byte, error) {
	type plain PostModelSwitchHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PostModelSwitch"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRequestHookInput (TS: PermissionRequestHookInput)
type PermissionRequestHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "PermissionRequest")
	HookEventName         string               `json:"hook_event_name"`
	ToolName              string               `json:"tool_name"`
	ToolInput             json.RawMessage      `json:"tool_input"`
	PermissionSuggestions []PermissionUpdate   `json:"permission_suggestions,omitempty"`
	MCPServer             *McpServerProvenance `json:"mcp_server,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRequestHookInput) UnmarshalJSON(b []byte) error {
	type plain PermissionRequestHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRequestHookInput) MarshalJSON() ([]byte, error) {
	type plain PermissionRequestHookInput
	if v.HookEventName == "" {
		v.HookEventName = "PermissionRequest"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SetupHookInput (TS: SetupHookInput)
type SetupHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "Setup")
	HookEventName string `json:"hook_event_name"`
	// (one of "init", "maintenance")
	Trigger string `json:"trigger"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetupHookInput) UnmarshalJSON(b []byte) error {
	type plain SetupHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetupHookInput) MarshalJSON() ([]byte, error) {
	type plain SetupHookInput
	if v.HookEventName == "" {
		v.HookEventName = "Setup"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TeammateIdleHookInput (TS: TeammateIdleHookInput)
type TeammateIdleHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "TeammateIdle")
	HookEventName string `json:"hook_event_name"`
	TeammateName  string `json:"teammate_name"`
	// Deprecated:
	TeamName string `json:"team_name"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TeammateIdleHookInput) UnmarshalJSON(b []byte) error {
	type plain TeammateIdleHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TeammateIdleHookInput) MarshalJSON() ([]byte, error) {
	type plain TeammateIdleHookInput
	if v.HookEventName == "" {
		v.HookEventName = "TeammateIdle"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TaskCreatedHookInput (TS: TaskCreatedHookInput)
type TaskCreatedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "TaskCreated")
	HookEventName   string `json:"hook_event_name"`
	TaskID          string `json:"task_id"`
	TaskSubject     string `json:"task_subject"`
	TaskDescription string `json:"task_description,omitempty"`
	TeammateName    string `json:"teammate_name,omitempty"`
	// Deprecated:
	TeamName string `json:"team_name,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskCreatedHookInput) UnmarshalJSON(b []byte) error {
	type plain TaskCreatedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskCreatedHookInput) MarshalJSON() ([]byte, error) {
	type plain TaskCreatedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "TaskCreated"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TaskCompletedHookInput (TS: TaskCompletedHookInput)
type TaskCompletedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "TaskCompleted")
	HookEventName   string `json:"hook_event_name"`
	TaskID          string `json:"task_id"`
	TaskSubject     string `json:"task_subject"`
	TaskDescription string `json:"task_description,omitempty"`
	TeammateName    string `json:"teammate_name,omitempty"`
	// Deprecated:
	TeamName string `json:"team_name,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskCompletedHookInput) UnmarshalJSON(b []byte) error {
	type plain TaskCompletedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskCompletedHookInput) MarshalJSON() ([]byte, error) {
	type plain TaskCompletedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "TaskCompleted"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ElicitationHookInput - Hook input for the Elicitation event. (TS: ElicitationHookInput)
type ElicitationHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "Elicitation")
	HookEventName string `json:"hook_event_name"`
	MCPServerName string `json:"mcp_server_name"`
	Message       string `json:"message"`
	// (one of "form", "url")
	Mode            string         `json:"mode,omitempty"`
	URL             string         `json:"url,omitempty"`
	ElicitationID   string         `json:"elicitation_id,omitempty"`
	RequestedSchema map[string]any `json:"requested_schema,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationHookInput) UnmarshalJSON(b []byte) error {
	type plain ElicitationHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationHookInput) MarshalJSON() ([]byte, error) {
	type plain ElicitationHookInput
	if v.HookEventName == "" {
		v.HookEventName = "Elicitation"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ElicitationResultHookInput - Hook input for the ElicitationResult event. (TS: ElicitationResultHookInput)
type ElicitationResultHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "ElicitationResult")
	HookEventName string `json:"hook_event_name"`
	MCPServerName string `json:"mcp_server_name"`
	ElicitationID string `json:"elicitation_id,omitempty"`
	// (one of "form", "url")
	Mode string `json:"mode,omitempty"`
	// (one of "accept", "decline", "cancel")
	Action  string         `json:"action"`
	Content map[string]any `json:"content,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationResultHookInput) UnmarshalJSON(b []byte) error {
	type plain ElicitationResultHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationResultHookInput) MarshalJSON() ([]byte, error) {
	type plain ElicitationResultHookInput
	if v.HookEventName == "" {
		v.HookEventName = "ElicitationResult"
	}
	return marshalKeep(plain(v), v.Raw)
}

// InstructionsLoadedHookInput (TS: InstructionsLoadedHookInput)
type InstructionsLoadedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "InstructionsLoaded")
	HookEventName string `json:"hook_event_name"`
	FilePath      string `json:"file_path"`
	// (one of "User", "Project", "Local", "Managed")
	MemoryType string `json:"memory_type"`
	// (one of "session_start", "nested_traversal", "path_glob_match", "include", "compact")
	LoadReason      string   `json:"load_reason"`
	Globs           []string `json:"globs,omitempty"`
	TriggerFilePath string   `json:"trigger_file_path,omitempty"`
	ParentFilePath  string   `json:"parent_file_path,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InstructionsLoadedHookInput) UnmarshalJSON(b []byte) error {
	type plain InstructionsLoadedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InstructionsLoadedHookInput) MarshalJSON() ([]byte, error) {
	type plain InstructionsLoadedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "InstructionsLoaded"
	}
	return marshalKeep(plain(v), v.Raw)
}

// WorktreeCreateHookInput (TS: WorktreeCreateHookInput)
type WorktreeCreateHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "WorktreeCreate")
	HookEventName string `json:"hook_event_name"`
	Name          string `json:"name"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *WorktreeCreateHookInput) UnmarshalJSON(b []byte) error {
	type plain WorktreeCreateHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v WorktreeCreateHookInput) MarshalJSON() ([]byte, error) {
	type plain WorktreeCreateHookInput
	if v.HookEventName == "" {
		v.HookEventName = "WorktreeCreate"
	}
	return marshalKeep(plain(v), v.Raw)
}

// WorktreeRemoveHookInput (TS: WorktreeRemoveHookInput)
type WorktreeRemoveHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "WorktreeRemove")
	HookEventName string `json:"hook_event_name"`
	WorktreePath  string `json:"worktree_path"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *WorktreeRemoveHookInput) UnmarshalJSON(b []byte) error {
	type plain WorktreeRemoveHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v WorktreeRemoveHookInput) MarshalJSON() ([]byte, error) {
	type plain WorktreeRemoveHookInput
	if v.HookEventName == "" {
		v.HookEventName = "WorktreeRemove"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CwdChangedHookInput (TS: CwdChangedHookInput)
type CwdChangedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "CwdChanged")
	HookEventName string `json:"hook_event_name"`
	OldCwd        string `json:"old_cwd"`
	NewCwd        string `json:"new_cwd"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CwdChangedHookInput) UnmarshalJSON(b []byte) error {
	type plain CwdChangedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CwdChangedHookInput) MarshalJSON() ([]byte, error) {
	type plain CwdChangedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "CwdChanged"
	}
	return marshalKeep(plain(v), v.Raw)
}

// FileChangedHookInput (TS: FileChangedHookInput)
type FileChangedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "FileChanged")
	HookEventName string `json:"hook_event_name"`
	FilePath      string `json:"file_path"`
	// (one of "change", "add", "unlink")
	Event string `json:"event"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *FileChangedHookInput) UnmarshalJSON(b []byte) error {
	type plain FileChangedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v FileChangedHookInput) MarshalJSON() ([]byte, error) {
	type plain FileChangedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "FileChanged"
	}
	return marshalKeep(plain(v), v.Raw)
}

// DirectoryAddedHookInput (TS: DirectoryAddedHookInput)
type DirectoryAddedHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "DirectoryAdded")
	HookEventName string `json:"hook_event_name"`
	// Absolute path of the directory that was added.
	Directory string `json:"directory"`
	// How the directory was added: "slash_command" for /add-dir, "register_repo_root" for the SDK control_request. (one of "slash_command", "register_repo_root")
	Source string `json:"source"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *DirectoryAddedHookInput) UnmarshalJSON(b []byte) error {
	type plain DirectoryAddedHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v DirectoryAddedHookInput) MarshalJSON() ([]byte, error) {
	type plain DirectoryAddedHookInput
	if v.HookEventName == "" {
		v.HookEventName = "DirectoryAdded"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageDisplayHookInput - Hook input for the MessageDisplay event. (TS: MessageDisplayHookInput)
type MessageDisplayHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	// UUID correlating a user prompt with all subsequent events until the next prompt.
	PromptID       string `json:"prompt_id,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	// Subagent identifier.
	AgentID string `json:"agent_id,omitempty"`
	// Agent type name (e.g., "general-purpose", "code-reviewer").
	AgentType string `json:"agent_type,omitempty"`
	// Reasoning effort applied to the current turn.
	Effort *BaseHookInputEffort `json:"effort,omitempty"`
	// (always "MessageDisplay")
	HookEventName string `json:"hook_event_name"`
	// UUID of the current turn.
	TurnID string `json:"turn_id"`
	// UUID of the assistant message being displayed.
	MessageID string `json:"message_id"`
	// Zero-based index of this delta within the message.
	Index int `json:"index"`
	// True on the message's last flush.
	Final bool `json:"final"`
	// The newly completed lines since the prior flush.
	Delta string `json:"delta"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageDisplayHookInput) UnmarshalJSON(b []byte) error {
	type plain MessageDisplayHookInput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageDisplayHookInput) MarshalJSON() ([]byte, error) {
	type plain MessageDisplayHookInput
	if v.HookEventName == "" {
		v.HookEventName = "MessageDisplay"
	}
	return marshalKeep(plain(v), v.Raw)
}

// HookCallbackRequest - Delivers a hook callback with its input data. (TS: SDKHookCallbackRequest, not exported (wire type))
type HookCallbackRequest struct {
	// (always "hook_callback")
	Subtype    string    `json:"subtype"`
	CallbackID string    `json:"callback_id"`
	Input      HookInput `json:"input"`
	ToolUseID  string    `json:"tool_use_id,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *HookCallbackRequest) UnmarshalJSON(b []byte) error {
	type plain HookCallbackRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v HookCallbackRequest) MarshalJSON() ([]byte, error) {
	type plain HookCallbackRequest
	if v.Subtype == "" {
		v.Subtype = "hook_callback"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpMessageRequest - Carries one MCP JSON-RPC message for an SDK-hosted MCP server (one named in initialize.sdkMcpServers or added later with mcp_set_servers). (TS: SDKControlMcpMessageRequest, not exported (wire type))
type McpMessageRequest struct {
	// (always "mcp_message")
	Subtype    string `json:"subtype"`
	ServerName string `json:"server_name"`
	// A JSON-RPC 2.0 message as defined by the Model Context Protocol (request, notification, or response object).
	Message json.RawMessage `json:"message"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpMessageRequest) UnmarshalJSON(b []byte) error {
	type plain McpMessageRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpMessageRequest) MarshalJSON() ([]byte, error) {
	type plain McpMessageRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_message"
	}
	return marshalKeep(plain(v), v.Raw)
}

// RewindFilesRequest - Rewinds file changes made since a specific user message. (TS: SDKControlRewindFilesRequest, not exported (wire type))
type RewindFilesRequest struct {
	// (always "rewind_files")
	Subtype       string `json:"subtype"`
	UserMessageID string `json:"user_message_id"`
	DryRun        *bool  `json:"dry_run,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RewindFilesRequest) UnmarshalJSON(b []byte) error {
	type plain RewindFilesRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RewindFilesRequest) MarshalJSON() ([]byte, error) {
	type plain RewindFilesRequest
	if v.Subtype == "" {
		v.Subtype = "rewind_files"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CancelAsyncMessageRequest - Drops a pending async user message from the command queue by uuid. (TS: SDKControlCancelAsyncMessageRequest, not exported (wire type))
type CancelAsyncMessageRequest struct {
	// (always "cancel_async_message")
	Subtype     string `json:"subtype"`
	MessageUUID string `json:"message_uuid"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CancelAsyncMessageRequest) UnmarshalJSON(b []byte) error {
	type plain CancelAsyncMessageRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CancelAsyncMessageRequest) MarshalJSON() ([]byte, error) {
	type plain CancelAsyncMessageRequest
	if v.Subtype == "" {
		v.Subtype = "cancel_async_message"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ReadFileRequest - Read a file from the session filesystem for the remote sidebar viewer. (TS: SDKControlReadFileRequest, not exported (wire type))
type ReadFileRequest struct {
	// (always "read_file")
	Subtype  string `json:"subtype"`
	Path     string `json:"path"`
	MaxBytes *int   `json:"max_bytes,omitempty"`
	// How to encode the bytes in `contents`. (one of "utf-8", "base64")
	Encoding string `json:"encoding,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReadFileRequest) UnmarshalJSON(b []byte) error {
	type plain ReadFileRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReadFileRequest) MarshalJSON() ([]byte, error) {
	type plain ReadFileRequest
	if v.Subtype == "" {
		v.Subtype = "read_file"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SeedReadStateRequest - Seeds the readFileState cache with a path+mtime entry. (TS: SDKControlSeedReadStateRequest, not exported (wire type))
type SeedReadStateRequest struct {
	// (always "seed_read_state")
	Subtype string `json:"subtype"`
	Path    string `json:"path"`
	Mtime   int    `json:"mtime"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SeedReadStateRequest) UnmarshalJSON(b []byte) error {
	type plain SeedReadStateRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SeedReadStateRequest) MarshalJSON() ([]byte, error) {
	type plain SeedReadStateRequest
	if v.Subtype == "" {
		v.Subtype = "seed_read_state"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpStdioServerConfig (TS: McpStdioServerConfig)
type McpStdioServerConfig struct {
	// (always "stdio")
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// Per-server tool-call timeout in milliseconds.
	Timeout *float64 `json:"timeout,omitempty"`
	// When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself lists with _meta anthropic/alwaysLoad set to false.
	AlwaysLoad *bool `json:"alwaysLoad,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpStdioServerConfig) UnmarshalJSON(b []byte) error {
	type plain McpStdioServerConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpStdioServerConfig) MarshalJSON() ([]byte, error) {
	type plain McpStdioServerConfig
	return marshalKeep(plain(v), v.Raw)
}

// McpSSEServerConfig (TS: McpSSEServerConfig)
type McpSSEServerConfig struct {
	// (always "sse")
	Type    string                `json:"type"`
	URL     string                `json:"url"`
	Headers map[string]string     `json:"headers,omitempty"`
	Tools   []McpServerToolPolicy `json:"tools,omitempty"`
	// Per-server tool-call timeout in milliseconds.
	Timeout *float64 `json:"timeout,omitempty"`
	// When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself lists with _meta anthropic/alwaysLoad set to false.
	AlwaysLoad *bool `json:"alwaysLoad,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpSSEServerConfig) UnmarshalJSON(b []byte) error {
	type plain McpSSEServerConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpSSEServerConfig) MarshalJSON() ([]byte, error) {
	type plain McpSSEServerConfig
	if v.Type == "" {
		v.Type = "sse"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpHttpServerConfig (TS: McpHttpServerConfig)
type McpHttpServerConfig struct {
	// (always "http")
	Type    string                `json:"type"`
	URL     string                `json:"url"`
	Headers map[string]string     `json:"headers,omitempty"`
	Tools   []McpServerToolPolicy `json:"tools,omitempty"`
	// Per-server tool-call timeout in milliseconds.
	Timeout *float64 `json:"timeout,omitempty"`
	// When true, all tools from this server are always included in the prompt and never deferred behind tool search, except a tool the server itself lists with _meta anthropic/alwaysLoad set to false.
	AlwaysLoad *bool `json:"alwaysLoad,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpHttpServerConfig) UnmarshalJSON(b []byte) error {
	type plain McpHttpServerConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpHttpServerConfig) MarshalJSON() ([]byte, error) {
	type plain McpHttpServerConfig
	if v.Type == "" {
		v.Type = "http"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpSdkServerConfig (TS: McpSdkServerConfig)
type McpSdkServerConfig struct {
	// (always "sdk")
	Type string `json:"type"`
	Name string `json:"name"`
	// Per-server tool-call timeout in milliseconds.
	Timeout *float64 `json:"timeout,omitempty"`
	// Per-tool permission policy the host declares for this server's tools (not the tool definitions passed to createSdkMcpServer). always_ask: the CLI asks the host (canUseTool or the permission prompt tool) before each...
	Tools []McpServerToolPolicy `json:"tools,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpSdkServerConfig) UnmarshalJSON(b []byte) error {
	type plain McpSdkServerConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpSdkServerConfig) MarshalJSON() ([]byte, error) {
	type plain McpSdkServerConfig
	if v.Type == "" {
		v.Type = "sdk"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpServerConfigForProcessTransport aliases McpServerConfig. (hand-modelled stand-in for TS McpServerConfigForProcessTransport)
type McpServerConfigForProcessTransport = McpServerConfig

// McpSetServersRequest - Replaces the set of dynamically managed MCP servers. (TS: SDKControlMcpSetServersRequest, not exported (wire type))
type McpSetServersRequest struct {
	// (always "mcp_set_servers")
	Subtype string                                        `json:"subtype"`
	Servers map[string]McpServerConfigForProcessTransport `json:"servers"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpSetServersRequest) UnmarshalJSON(b []byte) error {
	type plain McpSetServersRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpSetServersRequest) MarshalJSON() ([]byte, error) {
	type plain McpSetServersRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_set_servers"
	}
	return marshalKeep(plain(v), v.Raw)
}

// RegisterRepoRootRequest - Add a directory as a working-directory root and optionally reload CLAUDE.md, skills, and plugins. (TS: SDKControlRegisterRepoRootRequest, not exported (wire type))
type RegisterRepoRootRequest struct {
	// (always "register_repo_root")
	Subtype        string `json:"subtype"`
	Directory      string `json:"directory"`
	ReloadClaudeMd *bool  `json:"reload_claude_md,omitempty"`
	ReloadPlugins  *bool  `json:"reload_plugins,omitempty"`
	ReloadSkills   *bool  `json:"reload_skills,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RegisterRepoRootRequest) UnmarshalJSON(b []byte) error {
	type plain RegisterRepoRootRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RegisterRepoRootRequest) MarshalJSON() ([]byte, error) {
	type plain RegisterRepoRootRequest
	if v.Subtype == "" {
		v.Subtype = "register_repo_root"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ReloadPluginsRequest - Reloads plugins from disk and returns the refreshed session components. (TS: SDKControlReloadPluginsRequest, not exported (wire type))
type ReloadPluginsRequest struct {
	// (always "reload_plugins")
	Subtype string `json:"subtype"`
	// When true, the reload is not applied if applying it would change the session's tool list while the conversation's prompt cache depends on that list (the same check the interactive /reload-plugins makes before it asks...
	HoldOnCacheImpact *bool `json:"hold_on_cache_impact,omitempty"`
	// Plugins for the reload to load as well, in the same shape as the SDK `plugins` option: each entry joins the session as one more --plugin-dir would (--plugin-dir-no-mcp when skipMcpDiscovery is set).
	AddPlugins []PluginConfig `json:"add_plugins,omitempty"`
	// since 0.3.325 (not in 0.3.311)
	// When true, a request that finds the session busy reloads nothing and gets an error whose error_code is reload_plugins_busy; ask again once the session is idle.
	RefuseIfBusy *bool `json:"refuse_if_busy,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadPluginsRequest) UnmarshalJSON(b []byte) error {
	type plain ReloadPluginsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadPluginsRequest) MarshalJSON() ([]byte, error) {
	type plain ReloadPluginsRequest
	if v.Subtype == "" {
		v.Subtype = "reload_plugins"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ReloadSkillsRequest - Reloads skills from disk and returns the refreshed skill list. (TS: SDKControlReloadSkillsRequest, not exported (wire type))
type ReloadSkillsRequest struct {
	// (always "reload_skills")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadSkillsRequest) UnmarshalJSON(b []byte) error {
	type plain ReloadSkillsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadSkillsRequest) MarshalJSON() ([]byte, error) {
	type plain ReloadSkillsRequest
	if v.Subtype == "" {
		v.Subtype = "reload_skills"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ReloadOutputStylesRequest - Re-reads the output-style directories from disk (a style file written mid-session is otherwise invisible until the next session) and returns the refreshed style names. (TS: SDKControlReloadOutputStylesRequest, not exported (wire type))
type ReloadOutputStylesRequest struct {
	// (always "reload_output_styles")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadOutputStylesRequest) UnmarshalJSON(b []byte) error {
	type plain ReloadOutputStylesRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadOutputStylesRequest) MarshalJSON() ([]byte, error) {
	type plain ReloadOutputStylesRequest
	if v.Subtype == "" {
		v.Subtype = "reload_output_styles"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpReconnectRequest - Reconnects a disconnected or failed MCP server. (TS: SDKControlMcpReconnectRequest, not exported (wire type))
type McpReconnectRequest struct {
	// (always "mcp_reconnect")
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpReconnectRequest) UnmarshalJSON(b []byte) error {
	type plain McpReconnectRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpReconnectRequest) MarshalJSON() ([]byte, error) {
	type plain McpReconnectRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_reconnect"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpToggleRequest - Enables or disables an MCP server. (TS: SDKControlMcpToggleRequest, not exported (wire type))
type McpToggleRequest struct {
	// (always "mcp_toggle")
	Subtype    string `json:"subtype"`
	ServerName string `json:"serverName"`
	Enabled    bool   `json:"enabled"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpToggleRequest) UnmarshalJSON(b []byte) error {
	type plain McpToggleRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpToggleRequest) MarshalJSON() ([]byte, error) {
	type plain McpToggleRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_toggle"
	}
	return marshalKeep(plain(v), v.Raw)
}

// StopTaskRequest - Stops a running task. (TS: SDKControlStopTaskRequest, not exported (wire type))
type StopTaskRequest struct {
	// (always "stop_task")
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *StopTaskRequest) UnmarshalJSON(b []byte) error {
	type plain StopTaskRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v StopTaskRequest) MarshalJSON() ([]byte, error) {
	type plain StopTaskRequest
	if v.Subtype == "" {
		v.Subtype = "stop_task"
	}
	return marshalKeep(plain(v), v.Raw)
}

// BackgroundTasksRequest - Backgrounds in-flight foreground tasks (Bash commands and subagents). (TS: SDKControlBackgroundTasksRequest, not exported (wire type))
type BackgroundTasksRequest struct {
	// (always "background_tasks")
	Subtype string `json:"subtype"`
	// When set, backgrounds only the task whose originating tool_use block has this id.
	ToolUseID string `json:"tool_use_id,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BackgroundTasksRequest) UnmarshalJSON(b []byte) error {
	type plain BackgroundTasksRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BackgroundTasksRequest) MarshalJSON() ([]byte, error) {
	type plain BackgroundTasksRequest
	if v.Subtype == "" {
		v.Subtype = "background_tasks"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetTaskOutputRequest - Reads the end of one background shell or Monitor task's output: at most the last 8 KiB the command wrote, the same tail the terminal's /tasks detail view reads. (TS: SDKControlGetTaskOutputRequest, not exported (wire type))
type GetTaskOutputRequest struct {
	// (always "get_task_output")
	Subtype string `json:"subtype"`
	// A shell or Monitor task of this session, running or ended: the task_id from task_started or background_tasks_changed.
	TaskID string `json:"task_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetTaskOutputRequest) UnmarshalJSON(b []byte) error {
	type plain GetTaskOutputRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetTaskOutputRequest) MarshalJSON() ([]byte, error) {
	type plain GetTaskOutputRequest
	if v.Subtype == "" {
		v.Subtype = "get_task_output"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ApplyFlagSettingsRequest - Merges the provided settings into the flag settings layer, updating the active configuration. (TS: SDKControlApplyFlagSettingsRequest, not exported (wire type))
type ApplyFlagSettingsRequest struct {
	// (always "apply_flag_settings")
	Subtype  string         `json:"subtype"`
	Settings map[string]any `json:"settings"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ApplyFlagSettingsRequest) UnmarshalJSON(b []byte) error {
	type plain ApplyFlagSettingsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ApplyFlagSettingsRequest) MarshalJSON() ([]byte, error) {
	type plain ApplyFlagSettingsRequest
	if v.Subtype == "" {
		v.Subtype = "apply_flag_settings"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetSettingsRequest - Returns the effective merged settings and the raw per-source settings. (TS: SDKControlGetSettingsRequest, not exported (wire type))
type GetSettingsRequest struct {
	// (always "get_settings")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetSettingsRequest) UnmarshalJSON(b []byte) error {
	type plain GetSettingsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetSettingsRequest) MarshalJSON() ([]byte, error) {
	type plain GetSettingsRequest
	if v.Subtype == "" {
		v.Subtype = "get_settings"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingRequest - Returns the hooks listing the CLI's read-only /hooks menu renders: settings-file, session, and plugin hooks grouped by event and matcher, with display-ready strings (control characters revealed) and the policy and... (TS: SDKControlGetHooksListingRequest, not exported (wire type))
type GetHooksListingRequest struct {
	// (always "get_hooks_listing")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingRequest) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingRequest) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingRequest
	if v.Subtype == "" {
		v.Subtype = "get_hooks_listing"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UpdateSettingsRequest - Writes settings through the CLI's own writer. (TS: SDKControlUpdateSettingsRequest, not exported (wire type))
type UpdateSettingsRequest struct {
	// (always "update_settings")
	Subtype string `json:"subtype"`
	// Which settings file to write: the project's local settings file, where /config's writes land, or the user's settings file, which takes effortLevel only. (one of "localSettings", "userSettings")
	Source   string         `json:"source"`
	Settings map[string]any `json:"settings"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UpdateSettingsRequest) UnmarshalJSON(b []byte) error {
	type plain UpdateSettingsRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UpdateSettingsRequest) MarshalJSON() ([]byte, error) {
	type plain UpdateSettingsRequest
	if v.Subtype == "" {
		v.Subtype = "update_settings"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ElicitationControlRequest - Requests the SDK consumer to handle an MCP elicitation (user input request). (TS: SDKControlElicitationRequest, not exported (wire type))
type ElicitationControlRequest struct {
	// (always "elicitation")
	Subtype       string `json:"subtype"`
	MCPServerName string `json:"mcp_server_name"`
	Message       string `json:"message"`
	// (one of "form", "url")
	Mode            string         `json:"mode,omitempty"`
	URL             string         `json:"url,omitempty"`
	ElicitationID   string         `json:"elicitation_id,omitempty"`
	RequestedSchema map[string]any `json:"requested_schema,omitempty"`
	// requested_schema.properties' top-level keys in the order the MCP server listed them; absent for URL mode and when the schema has no properties object.
	PropertyOrder []string `json:"property_order,omitempty"`
	// Permission-display title from the MCP server's _meta['anthropic/permissionDisplay'].
	Title string `json:"title,omitempty"`
	// Short tool/server label from _meta['anthropic/permissionDisplay'].displayName.
	DisplayName string `json:"display_name,omitempty"`
	// Permission-display subtitle from _meta['anthropic/permissionDisplay'].description.
	Description string `json:"description,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationControlRequest) UnmarshalJSON(b []byte) error {
	type plain ElicitationControlRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationControlRequest) MarshalJSON() ([]byte, error) {
	type plain ElicitationControlRequest
	if v.Subtype == "" {
		v.Subtype = "elicitation"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserDialogControlRequest - Requests the SDK consumer to render a tool-driven blocking dialog and return the user choice. (TS: SDKControlRequestUserDialogRequest, not exported (wire type))
type UserDialogControlRequest struct {
	// (always "request_user_dialog")
	Subtype string `json:"subtype"`
	// Identifier for the dialog the host should render.
	DialogKind string `json:"dialog_kind"`
	// Dialog-specific data passed to the host renderer.
	Payload   map[string]any `json:"payload"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserDialogControlRequest) UnmarshalJSON(b []byte) error {
	type plain UserDialogControlRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserDialogControlRequest) MarshalJSON() ([]byte, error) {
	type plain UserDialogControlRequest
	if v.Subtype == "" {
		v.Subtype = "request_user_dialog"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ListPermissionRulesRequest - Requests the session's live permission rules and workspace directories — the same data /permissions lists in the terminal: rules from settings files plus session-only approvals, slash-command grants, and... (TS: SDKControlListPermissionRulesRequest, not exported (wire type))
type ListPermissionRulesRequest struct {
	// (always "list_permission_rules")
	Subtype string `json:"subtype"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ListPermissionRulesRequest) UnmarshalJSON(b []byte) error {
	type plain ListPermissionRulesRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ListPermissionRulesRequest) MarshalJSON() ([]byte, error) {
	type plain ListPermissionRulesRequest
	if v.Subtype == "" {
		v.Subtype = "list_permission_rules"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpReadResourceRequest - Reads one MCP Apps (SEP-1865) UI resource — a `ui://` URI, typically the `_meta.ui.resourceUri` a tool declares — from a connected MCP server the CLI itself dialed, with `resources/read`, for a host that renders it. (TS: SDKControlMcpReadResourceRequest, not exported (wire type))
type McpReadResourceRequest struct {
	// (always "mcp_read_resource")
	Subtype string `json:"subtype"`
	// Server name, as mcp_status reports it (not normalized).
	ServerName string `json:"serverName"`
	// The resource to read.
	URI string `json:"uri"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpReadResourceRequest) UnmarshalJSON(b []byte) error {
	type plain McpReadResourceRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpReadResourceRequest) MarshalJSON() ([]byte, error) {
	type plain McpReadResourceRequest
	if v.Subtype == "" {
		v.Subtype = "mcp_read_resource"
	}
	return marshalKeep(plain(v), v.Raw)
}

// since 0.3.312-0.3.325 (not in 0.3.311)
// SetHasSubscribersRequest - Tells the CLI whether anybody is watching the output that the host reads from stream-json stdout and passes on. (TS: SDKControlSetHasSubscribersRequest, not exported (wire type))
type SetHasSubscribersRequest struct {
	// (always "set_has_subscribers")
	Subtype string `json:"subtype"`
	// false: nobody watches now. true: somebody does. null: the host does not know, which counts as true. (nullable)
	HasSubscribers *bool `json:"has_subscribers"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetHasSubscribersRequest) UnmarshalJSON(b []byte) error {
	type plain SetHasSubscribersRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetHasSubscribersRequest) MarshalJSON() ([]byte, error) {
	type plain SetHasSubscribersRequest
	if v.Subtype == "" {
		v.Subtype = "set_has_subscribers"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ControlRequest - Envelope for a control-protocol request, sent by either side on the same stream as the messages. (TS: SDKControlRequest)
type ControlRequest struct {
	// (always "control_request")
	Type string `json:"type"`
	// Chosen by the sender, unique among its in-flight requests; the control_response (and any control_cancel_request) for this request echoes it.
	RequestID string              `json:"request_id"`
	Request   ControlRequestInner `json:"request"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ControlRequest) UnmarshalJSON(b []byte) error {
	type plain ControlRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ControlRequest) MarshalJSON() ([]byte, error) {
	type plain ControlRequest
	if v.Type == "" {
		v.Type = "control_request"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ControlErrorResponse - The request failed or was rejected (unknown subtype, invalid arguments, or an error while handling it). (TS: ControlErrorResponse, not exported (wire type))
type ControlErrorResponse struct {
	// (always "error")
	Subtype string `json:"subtype"`
	// The request_id of the control_request this answers.
	RequestID string `json:"request_id"`
	// Human-readable failure description.
	Error string `json:"error"`
	// can_use_tool requests this CLI process has issued and not yet resolved, so a client joining an already-initialized session learns about in-flight prompts.
	PendingPermissionRequests []ControlRequest `json:"pending_permission_requests,omitempty"`
	// request_user_dialog requests this CLI process has issued and not yet resolved (sibling of pending_permission_requests, with the same inherited-prompt caveat), so a client joining an already-initialized session can...
	PendingUserDialogRequests []ControlRequest `json:"pending_user_dialog_requests,omitempty"`
	// since 0.3.312-0.3.325 (not in 0.3.311)
	// elicitation requests that this CLI process sent and that are not yet answered or withdrawn (sibling of pending_permission_requests), each as it was sent, so a client that joins after an MCP server asked can still...
	PendingElicitationRequests []ControlRequest `json:"pending_elicitation_requests,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ControlErrorResponse) UnmarshalJSON(b []byte) error {
	type plain ControlErrorResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ControlErrorResponse) MarshalJSON() ([]byte, error) {
	type plain ControlErrorResponse
	if v.Subtype == "" {
		v.Subtype = "error"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ControlSuccessResponse - The request was handled. (TS: ControlResponse, not exported (wire type))
type ControlSuccessResponse struct {
	// (always "success")
	Subtype string `json:"subtype"`
	// The request_id of the control_request this answers.
	RequestID string `json:"request_id"`
	// The success payload, shaped as documented for the answered request's subtype; absent or {} for requests that are merely acknowledged.
	Response map[string]any `json:"response,omitempty"`
	// can_use_tool requests this CLI process has issued and not yet resolved, so a client joining an already-initialized session learns about in-flight prompts.
	PendingPermissionRequests []ControlRequest `json:"pending_permission_requests,omitempty"`
	// request_user_dialog requests this CLI process has issued and not yet resolved (sibling of pending_permission_requests, with the same inherited-prompt caveat), so a client joining an already-initialized session can...
	PendingUserDialogRequests []ControlRequest `json:"pending_user_dialog_requests,omitempty"`
	// since 0.3.312-0.3.325 (not in 0.3.311)
	// elicitation requests that this CLI process sent and that are not yet answered or withdrawn (sibling of pending_permission_requests), each as it was sent, so a client that joins after an MCP server asked can still...
	PendingElicitationRequests []ControlRequest `json:"pending_elicitation_requests,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ControlSuccessResponse) UnmarshalJSON(b []byte) error {
	type plain ControlSuccessResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ControlSuccessResponse) MarshalJSON() ([]byte, error) {
	type plain ControlSuccessResponse
	if v.Subtype == "" {
		v.Subtype = "success"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CwdChangedHookSpecificOutput (TS: CwdChangedHookSpecificOutput)
type CwdChangedHookSpecificOutput struct {
	// (always "CwdChanged")
	HookEventName string   `json:"hookEventName"`
	WatchPaths    []string `json:"watchPaths,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CwdChangedHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain CwdChangedHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CwdChangedHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain CwdChangedHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "CwdChanged"
	}
	return marshalKeep(plain(v), v.Raw)
}

// since 0.3.323 (not in 0.3.311)
// DeniedModelsMatch - What answers. (alpha) (TS: DeniedModelsMatch)
type DeniedModelsMatch struct {
	// For each model asked about, in order: whether an entry blocks it.
	Denied []bool `json:"denied"`
	// The CLI's notes on entries: why one was ignored, or what one blocks where that may surprise.
	Warnings []string `json:"warnings"`
}

// EffortLevel is a string enum. Effort level for controlling how much thinking/reasoning Claude applies. (TS: EffortLevel)
type EffortLevel string

const (
	EffortLevelLow    EffortLevel = "low"
	EffortLevelMedium EffortLevel = "medium"
	EffortLevelHigh   EffortLevel = "high"
	EffortLevelXhigh  EffortLevel = "xhigh"
	EffortLevelMax    EffortLevel = "max"
)

// ElicitationResult - MCP elicitation result (ElicitResult from (hand-modelled stand-in for TS ElicitResult)
type ElicitationResult struct {
	// (one of "accept", "decline", "cancel")
	Action  string         `json:"action"`
	Content map[string]any `json:"content,omitempty"`
	XMeta   map[string]any `json:"_meta,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationResult) UnmarshalJSON(b []byte) error {
	type plain ElicitationResult
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationResult) MarshalJSON() ([]byte, error) {
	type plain ElicitationResult
	return marshalKeep(plain(v), v.Raw)
}

// ElicitationHookSpecificOutput - Hook-specific output for the Elicitation event. (TS: ElicitationHookSpecificOutput)
type ElicitationHookSpecificOutput struct {
	// (always "Elicitation")
	HookEventName string `json:"hookEventName"`
	// (one of "accept", "decline", "cancel")
	Action  string         `json:"action,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain ElicitationHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain ElicitationHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "Elicitation"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ElicitationRequest - Elicitation request from an MCP server, asking the SDK consumer for user input. (TS: ElicitationRequest)
type ElicitationRequest struct {
	// Name of the MCP server requesting elicitation
	ServerName string `json:"serverName"`
	// Message to display to the user
	Message string `json:"message"`
	// Elicitation mode: 'form' for structured input, 'url' for browser-based auth (one of "form", "url")
	Mode string `json:"mode,omitempty"`
	// URL to open (only for 'url' mode)
	URL string `json:"url,omitempty"`
	// Elicitation ID for correlating URL elicitations with completion notifications (URL mode only)
	ElicitationID string `json:"elicitationId,omitempty"`
	// JSON Schema for the requested input (only for 'form' mode)
	RequestedSchema map[string]any `json:"requestedSchema,omitempty"`
	// Permission-display title from MCP `_meta['anthropic/permissionDisplay']` — header for elicitation-driven permission prompts
	Title string `json:"title,omitempty"`
	// Short tool/server label from MCP `_meta['anthropic/permissionDisplay'].displayName`
	DisplayName string `json:"displayName,omitempty"`
	// Permission-display subtitle from MCP `_meta['anthropic/permissionDisplay'].description`
	Description string `json:"description,omitempty"`
}

// ElicitationResultHookSpecificOutput - Hook-specific output for the ElicitationResult event. (TS: ElicitationResultHookSpecificOutput)
type ElicitationResultHookSpecificOutput struct {
	// (always "ElicitationResult")
	HookEventName string `json:"hookEventName"`
	// (one of "accept", "decline", "cancel")
	Action  string         `json:"action,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationResultHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain ElicitationResultHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationResultHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain ElicitationResultHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "ElicitationResult"
	}
	return marshalKeep(plain(v), v.Raw)
}

// FastModeDisabledReason is a string enum. Why fast mode can't serve right now. (TS: FastModeDisabledReason)
type FastModeDisabledReason string

const (
	FastModeDisabledReasonFree               FastModeDisabledReason = "free"
	FastModeDisabledReasonPreference         FastModeDisabledReason = "preference"
	FastModeDisabledReasonExtraUsageDisabled FastModeDisabledReason = "extra_usage_disabled"
	FastModeDisabledReasonNetworkError       FastModeDisabledReason = "network_error"
	FastModeDisabledReasonUnknown            FastModeDisabledReason = "unknown"
	FastModeDisabledReasonNotFirstParty      FastModeDisabledReason = "not_first_party"
	FastModeDisabledReasonDisabledByEnv      FastModeDisabledReason = "disabled_by_env"
	FastModeDisabledReasonModelNotAllowed    FastModeDisabledReason = "model_not_allowed"
	FastModeDisabledReasonSdkOptInRequired   FastModeDisabledReason = "sdk_opt_in_required"
	FastModeDisabledReasonPending            FastModeDisabledReason = "pending"
)

// FastModeState is a string enum. Fast mode state: off, in cooldown after rate limit, or actively enabled. (TS: FastModeState)
type FastModeState string

const (
	FastModeStateOff      FastModeState = "off"
	FastModeStateCooldown FastModeState = "cooldown"
	FastModeStateOn       FastModeState = "on"
)

// FileChangedHookSpecificOutput (TS: FileChangedHookSpecificOutput)
type FileChangedHookSpecificOutput struct {
	// (always "FileChanged")
	HookEventName string   `json:"hookEventName"`
	WatchPaths    []string `json:"watchPaths,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *FileChangedHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain FileChangedHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v FileChangedHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain FileChangedHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "FileChanged"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ForkSessionOptions - Options for forking a session into a new branch. (TS: ForkSessionOptions)
type ForkSessionOptions struct {
	// Project directory path (same semantics as `listSessions({ dir })`).
	Dir string `json:"dir,omitempty"`
	// Slice transcript up to the message whose `uuid` field equals this value (inclusive).
	UpToMessageID string `json:"upToMessageId,omitempty"`
	// Custom title for the fork.
	Title string `json:"title,omitempty"`
}

// ForkSessionResult - Result of a fork operation. (TS: ForkSessionResult)
type ForkSessionResult struct {
	// New session UUID.
	SessionID string `json:"sessionId"`
}

// GetSessionInfoOptions - Options for getSessionInfo. (TS: GetSessionInfoOptions)
type GetSessionInfoOptions struct {
	// Project directory path (same semantics as `listSessions({ dir })`).
	Dir string `json:"dir,omitempty"`
}

// GetSessionMessagesOptions - Options for retrieving session messages. (TS: GetSessionMessagesOptions)
type GetSessionMessagesOptions struct {
	// Project directory to find the session in.
	Dir string `json:"dir,omitempty"`
	// Maximum number of messages to return.
	Limit *int `json:"limit,omitempty"`
	// Number of messages to skip from the start.
	Offset *int `json:"offset,omitempty"`
	// When true, include system messages (e.g., compact boundaries, informational notices) in the returned list alongside user/assistant messages.
	IncludeSystemMessages *bool `json:"includeSystemMessages,omitempty"`
}

// GetSubagentMessagesOptions - Options for retrieving subagent messages. (TS: GetSubagentMessagesOptions)
type GetSubagentMessagesOptions struct {
	// Project directory to find the session in.
	Dir string `json:"dir,omitempty"`
	// Maximum number of messages to return.
	Limit *int `json:"limit,omitempty"`
	// Number of messages to skip from the start.
	Offset *int `json:"offset,omitempty"`
}

// HookEvent is a string enum. (TS: HookEvent)
type HookEvent string

const (
	HookEventPreToolUse          HookEvent = "PreToolUse"
	HookEventPostToolUse         HookEvent = "PostToolUse"
	HookEventPostToolUseFailure  HookEvent = "PostToolUseFailure"
	HookEventPostToolBatch       HookEvent = "PostToolBatch"
	HookEventNotification        HookEvent = "Notification"
	HookEventUserPromptSubmit    HookEvent = "UserPromptSubmit"
	HookEventUserPromptExpansion HookEvent = "UserPromptExpansion"
	HookEventSessionStart        HookEvent = "SessionStart"
	HookEventSessionEnd          HookEvent = "SessionEnd"
	HookEventStop                HookEvent = "Stop"
	HookEventStopFailure         HookEvent = "StopFailure"
	HookEventSubagentStart       HookEvent = "SubagentStart"
	HookEventSubagentStop        HookEvent = "SubagentStop"
	HookEventPreCompact          HookEvent = "PreCompact"
	HookEventPostCompact         HookEvent = "PostCompact"
	HookEventPreModelSwitch      HookEvent = "PreModelSwitch"
	HookEventPostModelSwitch     HookEvent = "PostModelSwitch"
	HookEventPermissionRequest   HookEvent = "PermissionRequest"
	HookEventPermissionDenied    HookEvent = "PermissionDenied"
	HookEventSetup               HookEvent = "Setup"
	HookEventTeammateIdle        HookEvent = "TeammateIdle"
	HookEventTaskCreated         HookEvent = "TaskCreated"
	HookEventTaskCompleted       HookEvent = "TaskCompleted"
	HookEventElicitation         HookEvent = "Elicitation"
	HookEventElicitationResult   HookEvent = "ElicitationResult"
	HookEventConfigChange        HookEvent = "ConfigChange"
	HookEventWorktreeCreate      HookEvent = "WorktreeCreate"
	HookEventWorktreeRemove      HookEvent = "WorktreeRemove"
	HookEventInstructionsLoaded  HookEvent = "InstructionsLoaded"
	HookEventCwdChanged          HookEvent = "CwdChanged"
	HookEventFileChanged         HookEvent = "FileChanged"
	HookEventDirectoryAdded      HookEvent = "DirectoryAdded"
	HookEventMessageDisplay      HookEvent = "MessageDisplay"
)

// HookPermissionDecision is a string enum. (TS: HookPermissionDecision)
type HookPermissionDecision string

const (
	HookPermissionDecisionAllow HookPermissionDecision = "allow"
	HookPermissionDecisionDeny  HookPermissionDecision = "deny"
	HookPermissionDecisionAsk   HookPermissionDecision = "ask"
	HookPermissionDecisionDefer HookPermissionDecision = "defer"
)

// ImportSessionToStoreOptions - Options for importing a local JSONL session into a SessionStore. (alpha) (TS: ImportSessionToStoreOptions)
type ImportSessionToStoreOptions struct {
	// Project directory path (same semantics as `listSessions({ dir })`).
	Dir string `json:"dir,omitempty"`
	// If true, also import subagent transcripts.
	IncludeSubagents *bool `json:"includeSubagents,omitempty"`
	// Maximum entries per `store.append()` call.
	BatchSize *int `json:"batchSize,omitempty"`
}

// JsonSchemaOutputFormat (TS: JsonSchemaOutputFormat)
type JsonSchemaOutputFormat struct {
	// (always "json_schema")
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *JsonSchemaOutputFormat) UnmarshalJSON(b []byte) error {
	type plain JsonSchemaOutputFormat
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v JsonSchemaOutputFormat) MarshalJSON() ([]byte, error) {
	type plain JsonSchemaOutputFormat
	if v.Type == "" {
		v.Type = "json_schema"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ListSessionsOptions - Options for listing sessions. (TS: ListSessionsOptions)
type ListSessionsOptions struct {
	// Directory to list sessions for.
	Dir string `json:"dir,omitempty"`
	// Maximum number of sessions to return.
	Limit *int `json:"limit,omitempty"`
	// Number of sessions to skip from the start of the sorted result set.
	Offset *int `json:"offset,omitempty"`
	// When `dir` is provided and the directory is inside a git repository, include sessions from all git worktree paths.
	IncludeWorktrees *bool `json:"includeWorktrees,omitempty"`
	// Include programmatic/headless sessions (SDK entrypoints `sdk-cli`, `sdk-ts`, `sdk-py`) and daemon/daemon-worker sessions.
	IncludeProgrammatic *bool `json:"includeProgrammatic,omitempty"`
}

// ListSubagentsOptions - Options for listing subagents. (TS: ListSubagentsOptions)
type ListSubagentsOptions struct {
	// Project directory to find the session in.
	Dir string `json:"dir,omitempty"`
}

// McpClaudeAIProxyServerConfig (TS: McpClaudeAIProxyServerConfig)
type McpClaudeAIProxyServerConfig struct {
	// (always "claudeai-proxy")
	Type string `json:"type"`
	URL  string `json:"url"`
	ID   string `json:"id"`
	// Per-server tool-call timeout in milliseconds.
	Timeout *float64 `json:"timeout,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpClaudeAIProxyServerConfig) UnmarshalJSON(b []byte) error {
	type plain McpClaudeAIProxyServerConfig
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpClaudeAIProxyServerConfig) MarshalJSON() ([]byte, error) {
	type plain McpClaudeAIProxyServerConfig
	if v.Type == "" {
		v.Type = "claudeai-proxy"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpServerStatusServerInfo - inline object (field serverInfo of McpServerStatus)
type McpServerStatusServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpServerStatusServerInfo) UnmarshalJSON(b []byte) error {
	type plain McpServerStatusServerInfo
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpServerStatusServerInfo) MarshalJSON() ([]byte, error) {
	type plain McpServerStatusServerInfo
	return marshalKeep(plain(v), v.Raw)
}

// McpServerStatusConfig is kept as raw JSON. TS: McpServerConfigForProcessTransport | McpClaudeAIProxyServerConfig. (TS: McpServerStatusConfig)
type McpServerStatusConfig = json.RawMessage

// McpServerStatusToolAnnotations - inline object (field annotations of McpServerStatusTool)
type McpServerStatusToolAnnotations struct {
	ReadOnly    *bool `json:"readOnly,omitempty"`
	Destructive *bool `json:"destructive,omitempty"`
	OpenWorld   *bool `json:"openWorld,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpServerStatusToolAnnotations) UnmarshalJSON(b []byte) error {
	type plain McpServerStatusToolAnnotations
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpServerStatusToolAnnotations) MarshalJSON() ([]byte, error) {
	type plain McpServerStatusToolAnnotations
	return marshalKeep(plain(v), v.Raw)
}

// McpServerStatusTool - inline object (field tool of McpServerStatus)
type McpServerStatusTool struct {
	Name        string                          `json:"name"`
	Description string                          `json:"description,omitempty"`
	Annotations *McpServerStatusToolAnnotations `json:"annotations,omitempty"`
	// The MCP Apps (SEP-1865) members of the tool's `_meta`, for a host that renders the tool's `ui://` resource, under the keys the server used: `ui` (an object: `resourceUri`, a `ui://` string; `visibility`, an array of...
	XMeta map[string]any `json:"_meta,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpServerStatusTool) UnmarshalJSON(b []byte) error {
	type plain McpServerStatusTool
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpServerStatusTool) MarshalJSON() ([]byte, error) {
	type plain McpServerStatusTool
	return marshalKeep(plain(v), v.Raw)
}

// McpServerStatus - Status information for an MCP server connection. (TS: McpServerStatus)
type McpServerStatus struct {
	// Server name as configured
	Name string `json:"name"`
	// Current connection status (one of "connected", "failed", "needs-auth", "pending", "disabled")
	Status string `json:"status"`
	// Server information (available when connected)
	ServerInfo *McpServerStatusServerInfo `json:"serverInfo,omitempty"`
	// Error message (available when status is 'failed')
	Error string `json:"error,omitempty"`
	// Server configuration (includes URL for HTTP/SSE servers)
	Config McpServerStatusConfig `json:"config,omitempty"`
	// Configuration scope (e.g., project, user, local, claudeai, managed)
	Scope string `json:"scope,omitempty"`
	// Where the server definition came from: sdk (an in-process server the SDK host registered — only the host can register one), plugin (a server a plugin ships or registers at runtime), or the config scope (user,...
	Source string `json:"source,omitempty"`
	// Tools provided by this server (available when connected)
	Tools []McpServerStatusTool `json:"tools,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpServerStatus) UnmarshalJSON(b []byte) error {
	type plain McpServerStatus
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpServerStatus) MarshalJSON() ([]byte, error) {
	type plain McpServerStatus
	return marshalKeep(plain(v), v.Raw)
}

// McpSetServersResult - Result of a setMcpServers operation. (TS: McpSetServersResult)
type McpSetServersResult struct {
	// Names of servers that were added
	Added []string `json:"added"`
	// Names of servers that were removed
	Removed []string `json:"removed"`
	// Map of server names to error messages for servers that failed to connect
	Errors map[string]string `json:"errors"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpSetServersResult) UnmarshalJSON(b []byte) error {
	type plain McpSetServersResult
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpSetServersResult) MarshalJSON() ([]byte, error) {
	type plain McpSetServersResult
	return marshalKeep(plain(v), v.Raw)
}

// MessageDisplayHookSpecificOutput - Hook-specific output for the MessageDisplay event. (TS: MessageDisplayHookSpecificOutput)
type MessageDisplayHookSpecificOutput struct {
	// (always "MessageDisplay")
	HookEventName string `json:"hookEventName"`
	// Text displayed in place of the delta.
	DisplayContent string `json:"displayContent,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageDisplayHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain MessageDisplayHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageDisplayHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain MessageDisplayHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "MessageDisplay"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageParam - An Anthropic Messages API user message: role "user"; content is a string or an array of content blocks (kept raw: use MessageParam.Text / MessageParam.Blocks). (hand-modelled stand-in for TS MessageParam)
type MessageParam struct {
	// (one of "user", "assistant")
	Role string `json:"role"`
	// (TS: string | BetaContentBlock[])
	Content json.RawMessage `json:"content"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageParam) UnmarshalJSON(b []byte) error {
	type plain MessageParam
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageParam) MarshalJSON() ([]byte, error) {
	type plain MessageParam
	return marshalKeep(plain(v), v.Raw)
}

// ModelInfo - Information about an available model. (TS: ModelInfo)
type ModelInfo struct {
	// Model identifier to use in API calls
	Value string `json:"value"`
	// Canonical wire model id this row's `value` resolves to (e.g.
	ResolvedModel string `json:"resolvedModel,omitempty"`
	// Human-readable display name
	DisplayName string `json:"displayName"`
	// Description of the model's capabilities
	Description string `json:"description"`
	// Whether this model supports effort levels
	SupportsEffort *bool `json:"supportsEffort,omitempty"`
	// Available effort levels for this model (one of "low", "medium", "high", "xhigh", "max")
	SupportedEffortLevels []string `json:"supportedEffortLevels,omitempty"`
	// Whether this model supports adaptive thinking (Claude decides when and how much to think)
	SupportsAdaptiveThinking *bool `json:"supportsAdaptiveThinking,omitempty"`
	// Whether this model supports fast mode
	SupportsFastMode *bool `json:"supportsFastMode,omitempty"`
	// Whether this model supports auto mode
	SupportsAutoMode *bool `json:"supportsAutoMode,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ModelInfo) UnmarshalJSON(b []byte) error {
	type plain ModelInfo
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ModelInfo) MarshalJSON() ([]byte, error) {
	type plain ModelInfo
	return marshalKeep(plain(v), v.Raw)
}

// ModelUsage (TS: ModelUsage)
type ModelUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	// Thinking tokens, already counted inside outputTokens.
	ThinkingTokens           *int `json:"thinkingTokens,omitempty"`
	CacheReadInputTokens     int  `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int  `json:"cacheCreationInputTokens"`
	// Cache writes with a 5-minute lifetime, already counted inside cacheCreationInputTokens.
	CacheCreationEphemeral5mInputTokens *int `json:"cacheCreationEphemeral5mInputTokens,omitempty"`
	// Cache writes with a 1-hour lifetime, already counted inside cacheCreationInputTokens. costUSD prices them at the 1-hour rate when the model's price table has one.
	CacheCreationEphemeral1hInputTokens *int    `json:"cacheCreationEphemeral1hInputTokens,omitempty"`
	WebSearchRequests                   int     `json:"webSearchRequests"`
	CostUSD                             float64 `json:"costUSD"`
	ContextWindow                       float64 `json:"contextWindow"`
	MaxOutputTokens                     int     `json:"maxOutputTokens"`
	// Canonical model id used for the pricing lookup (e.g.
	CanonicalModel string `json:"canonicalModel,omitempty"`
	// API provider that served this model (e.g.
	Provider string `json:"provider,omitempty"`
	// Which price table the most recent request for this model was priced at: Claude Code's built-in list prices ('list'), the organization's managed-settings modelPricing rates or multiplier ('managed'), or neither... (one of "list", "managed", "unknown")
	CostBasis string `json:"costBasis,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ModelUsage) UnmarshalJSON(b []byte) error {
	type plain ModelUsage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ModelUsage) MarshalJSON() ([]byte, error) {
	type plain ModelUsage
	return marshalKeep(plain(v), v.Raw)
}

// NotificationHookSpecificOutput (TS: NotificationHookSpecificOutput)
type NotificationHookSpecificOutput struct {
	// (always "Notification")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *NotificationHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain NotificationHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v NotificationHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain NotificationHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "Notification"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ToolConfigAskUserQuestion - inline object (field askUserQuestion of ToolConfig)
type ToolConfigAskUserQuestion struct {
	// Content format for the `preview` field on question options. Default: 'markdown'. (one of "markdown", "html")
	PreviewFormat string `json:"previewFormat,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolConfigAskUserQuestion) UnmarshalJSON(b []byte) error {
	type plain ToolConfigAskUserQuestion
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolConfigAskUserQuestion) MarshalJSON() ([]byte, error) {
	type plain ToolConfigAskUserQuestion
	return marshalKeep(plain(v), v.Raw)
}

// ToolConfig - Per-tool configuration for built-in tools. (TS: ToolConfig)
type ToolConfig struct {
	AskUserQuestion *ToolConfigAskUserQuestion `json:"askUserQuestion,omitempty"`
}

// Beta is a string enum. (TS: SdkBeta)
type Beta string

const (
	BetaContext1m20250807 Beta = "context-1m-2025-08-07"
)

// SessionStoreFlush is a string enum. Flush strategy for transcript mirroring. - `'batched'` (default): buffer transcript_mirror frames and flush at end-of-turn or when pending thresholds are exceeded. - `'eager'`: schedule a background flush after every... (alpha) (TS: SessionStoreFlush)
type SessionStoreFlush string

const (
	SessionStoreFlushBatched SessionStoreFlush = "batched"
	SessionStoreFlushEager   SessionStoreFlush = "eager"
)

// ThinkingAdaptive - Claude decides when and how much to think (Opus 4.6+). (TS: ThinkingAdaptive)
type ThinkingAdaptive struct {
	// (always "adaptive")
	Type string `json:"type"`
	// (one of "summarized", "omitted")
	Display string `json:"display,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ThinkingAdaptive) UnmarshalJSON(b []byte) error {
	type plain ThinkingAdaptive
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ThinkingAdaptive) MarshalJSON() ([]byte, error) {
	type plain ThinkingAdaptive
	if v.Type == "" {
		v.Type = "adaptive"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ThinkingEnabled - Fixed thinking token budget (older models) (TS: ThinkingEnabled)
type ThinkingEnabled struct {
	// (always "enabled")
	Type         string `json:"type"`
	BudgetTokens *int   `json:"budgetTokens,omitempty"`
	// (one of "summarized", "omitted")
	Display string `json:"display,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ThinkingEnabled) UnmarshalJSON(b []byte) error {
	type plain ThinkingEnabled
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ThinkingEnabled) MarshalJSON() ([]byte, error) {
	type plain ThinkingEnabled
	if v.Type == "" {
		v.Type = "enabled"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ThinkingDisabled - No extended thinking (TS: ThinkingDisabled)
type ThinkingDisabled struct {
	// (always "disabled")
	Type string `json:"type"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ThinkingDisabled) UnmarshalJSON(b []byte) error {
	type plain ThinkingDisabled
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ThinkingDisabled) MarshalJSON() ([]byte, error) {
	type plain ThinkingDisabled
	if v.Type == "" {
		v.Type = "disabled"
	}
	return marshalKeep(plain(v), v.Raw)
}

// OptionsTaskBudget - inline object (field taskBudget of Options)
type OptionsTaskBudget struct {
	Total float64 `json:"total"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *OptionsTaskBudget) UnmarshalJSON(b []byte) error {
	type plain OptionsTaskBudget
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v OptionsTaskBudget) MarshalJSON() ([]byte, error) {
	type plain OptionsTaskBudget
	return marshalKeep(plain(v), v.Raw)
}

// OutputFormat (TS: OutputFormat)
type OutputFormat struct {
	// (always "json_schema")
	Type   string         `json:"type"`
	Schema map[string]any `json:"schema"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *OutputFormat) UnmarshalJSON(b []byte) error {
	type plain OutputFormat
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v OutputFormat) MarshalJSON() ([]byte, error) {
	type plain OutputFormat
	if v.Type == "" {
		v.Type = "json_schema"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SandboxSettingsNetworkTlsTerminate - inline object (field tlsTerminate of SandboxSettingsNetwork)
type SandboxSettingsNetworkTlsTerminate struct {
	CaCertPath string `json:"caCertPath,omitempty"`
	CaKeyPath  string `json:"caKeyPath,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SandboxSettingsNetworkTlsTerminate) UnmarshalJSON(b []byte) error {
	type plain SandboxSettingsNetworkTlsTerminate
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SandboxSettingsNetworkTlsTerminate) MarshalJSON() ([]byte, error) {
	type plain SandboxSettingsNetworkTlsTerminate
	return marshalKeep(plain(v), v.Raw)
}

// SandboxSettingsNetwork - inline object (field network of SandboxSettings)
type SandboxSettingsNetwork struct {
	AllowedDomains          []string                            `json:"allowedDomains,omitempty"`
	DeniedDomains           []string                            `json:"deniedDomains,omitempty"`
	StrictAllowlist         *bool                               `json:"strictAllowlist,omitempty"`
	AllowManagedDomainsOnly *bool                               `json:"allowManagedDomainsOnly,omitempty"`
	AllowUnixSockets        []string                            `json:"allowUnixSockets,omitempty"`
	AllowAllUnixSockets     *bool                               `json:"allowAllUnixSockets,omitempty"`
	AllowLocalBinding       *bool                               `json:"allowLocalBinding,omitempty"`
	AllowMachLookup         []string                            `json:"allowMachLookup,omitempty"`
	HttpProxyPort           *float64                            `json:"httpProxyPort,omitempty"`
	SocksProxyPort          *float64                            `json:"socksProxyPort,omitempty"`
	TlsTerminate            *SandboxSettingsNetworkTlsTerminate `json:"tlsTerminate,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SandboxSettingsNetwork) UnmarshalJSON(b []byte) error {
	type plain SandboxSettingsNetwork
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SandboxSettingsNetwork) MarshalJSON() ([]byte, error) {
	type plain SandboxSettingsNetwork
	return marshalKeep(plain(v), v.Raw)
}

// SandboxSettingsFilesystem - inline object (field filesystem of SandboxSettings)
type SandboxSettingsFilesystem struct {
	AllowWrite                []string `json:"allowWrite,omitempty"`
	DenyWrite                 []string `json:"denyWrite,omitempty"`
	DenyRead                  []string `json:"denyRead,omitempty"`
	AllowRead                 []string `json:"allowRead,omitempty"`
	AllowManagedReadPathsOnly *bool    `json:"allowManagedReadPathsOnly,omitempty"`
	Disabled                  *bool    `json:"disabled,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SandboxSettingsFilesystem) UnmarshalJSON(b []byte) error {
	type plain SandboxSettingsFilesystem
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SandboxSettingsFilesystem) MarshalJSON() ([]byte, error) {
	type plain SandboxSettingsFilesystem
	return marshalKeep(plain(v), v.Raw)
}

// SandboxSettingsRipgrep - inline object (field ripgrep of SandboxSettings)
type SandboxSettingsRipgrep struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SandboxSettingsRipgrep) UnmarshalJSON(b []byte) error {
	type plain SandboxSettingsRipgrep
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SandboxSettingsRipgrep) MarshalJSON() ([]byte, error) {
	type plain SandboxSettingsRipgrep
	return marshalKeep(plain(v), v.Raw)
}

// SandboxSettings - Sandbox settings (z.infer of SandboxSettingsSchema; a loose object, so unknown keys survive in Raw). (hand-modelled stand-in for TS SandboxSettings)
type SandboxSettings struct {
	Enabled                      *bool                      `json:"enabled,omitempty"`
	FailIfUnavailable            *bool                      `json:"failIfUnavailable,omitempty"`
	AutoAllowBashIfSandboxed     *bool                      `json:"autoAllowBashIfSandboxed,omitempty"`
	AllowUnsandboxedCommands     *bool                      `json:"allowUnsandboxedCommands,omitempty"`
	Network                      *SandboxSettingsNetwork    `json:"network,omitempty"`
	Filesystem                   *SandboxSettingsFilesystem `json:"filesystem,omitempty"`
	Credentials                  map[string]any             `json:"credentials,omitempty"`
	IgnoreViolations             map[string][]string        `json:"ignoreViolations,omitempty"`
	EnableWeakerNestedSandbox    *bool                      `json:"enableWeakerNestedSandbox,omitempty"`
	AllowNestedUserNamespaces    *bool                      `json:"allowNestedUserNamespaces,omitempty"`
	EnableWeakerNetworkIsolation *bool                      `json:"enableWeakerNetworkIsolation,omitempty"`
	AllowAppleEvents             *bool                      `json:"allowAppleEvents,omitempty"`
	ExcludedCommands             []string                   `json:"excludedCommands,omitempty"`
	Ripgrep                      *SandboxSettingsRipgrep    `json:"ripgrep,omitempty"`
	BwrapPath                    string                     `json:"bwrapPath,omitempty"`
	SocatPath                    string                     `json:"socatPath,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SandboxSettings) UnmarshalJSON(b []byte) error {
	type plain SandboxSettings
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SandboxSettings) MarshalJSON() ([]byte, error) {
	type plain SandboxSettings
	return marshalKeep(plain(v), v.Raw)
}

// SettingSource is a string enum. Source for loading filesystem-based settings. (TS: SettingSource)
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// Options - Options for the query function. (TS: Options)
type Options struct {
	// Additional directories Claude can access beyond the current working directory.
	AdditionalDirectories []string `json:"additionalDirectories,omitempty"`
	// The trusted checkout `cwd` is a worktree of.
	ProjectConfigRoot string `json:"projectConfigRoot,omitempty"`
	// Agent name for the main thread.
	Agent string `json:"agent,omitempty"`
	// Programmatically define custom subagents that can be invoked via the Agent tool.
	Agents map[string]AgentDefinition `json:"agents,omitempty"`
	// List of tool names that are auto-allowed without prompting for permission.
	AllowedTools []string `json:"allowedTools,omitempty"`
	// True when the host answers can_use_tool control requests (TS: the canUseTool callback). Implies --permission-prompt-tool=stdio.
	CanUseTool bool `json:"canUseTool,omitempty"`
	// Continue the most recent conversation in the current directory instead of starting a new one.
	Continue *bool `json:"continue,omitempty"`
	// Current working directory for the session.
	Cwd string `json:"cwd,omitempty"`
	// List of tool names that are disallowed.
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	// Map of tool-name aliases applied before name resolution.
	ToolAliases map[string]string `json:"toolAliases,omitempty"`
	// Specify the base set of available built-in tools. - `string[]` - Array of specific tool names (e.g., `['Bash', 'Read', 'Edit']`) - `[]` (empty array) - Disable all built-in tools - `{ type: 'preset'; preset:... (TS: string[] | { type: 'preset'; preset: 'claude_code'; })
	Tools json.RawMessage `json:"tools,omitempty"`
	// Environment variables for the Claude Code process.
	Env map[string]string `json:"env,omitempty"`
	// JavaScript runtime to use for executing Claude Code. (one of "bun", "deno", "node")
	Executable string `json:"executable,omitempty"`
	// Additional arguments to pass to the JavaScript runtime executable.
	ExecutableArgs []string `json:"executableArgs,omitempty"`
	// Additional CLI arguments to pass to Claude Code.
	ExtraArgs map[string]*string `json:"extraArgs,omitempty"`
	// Fallback model(s) to use if the primary model is overloaded or unavailable.
	FallbackModel string `json:"fallbackModel,omitempty"`
	// Enable file checkpointing to track file changes during the session.
	EnableFileCheckpointing *bool `json:"enableFileCheckpointing,omitempty"`
	// Per-tool configuration for built-in tools.
	ToolConfig *ToolConfig `json:"toolConfig,omitempty"`
	// When true, resumed sessions will fork to a new session ID rather than continuing the previous session.
	ForkSession *bool `json:"forkSession,omitempty"`
	// Enable beta features.
	Betas []Beta `json:"betas,omitempty"`
	// Hook callbacks for responding to various events during execution.
	Hooks map[HookEvent][]HookCallbackMatcher `json:"hooks,omitempty"`
	// True when the host answers elicitation control requests (TS: the onElicitation callback). Otherwise the SDK answers {action:"decline"}.
	OnElicitation bool `json:"onElicitation,omitempty"`
	// True when the host answers request_user_dialog control requests (TS: the onUserDialog callback). Otherwise the request is left unanswered.
	OnUserDialog bool `json:"onUserDialog,omitempty"`
	// Dialog kinds this consumer's `onUserDialog` can actually render (`request_user_dialog` `dialog_kind` values, e.g.
	SupportedDialogKinds []string `json:"supportedDialogKinds,omitempty"`
	// Declares that this consumer renders a per-task stop control wired to the `stop_task` control request, so the user can stop an individual background task.
	PerTaskStopAffordance *bool `json:"perTaskStopAffordance,omitempty"`
	// When false, disables session persistence to disk. Default: true.
	PersistSession *bool `json:"persistSession,omitempty"`
	// Controls how aggressively transcript entries are flushed to . (alpha)
	SessionStoreFlush SessionStoreFlush `json:"sessionStoreFlush,omitempty"`
	// Timeout for each `sessionStore.load()` / `sessionStore.listSubkeys()` call during resume materialization. Default: 60_000. (alpha)
	LoadTimeoutMs *float64 `json:"loadTimeoutMs,omitempty"`
	// Include hook lifecycle events in the output stream. Default: false.
	IncludeHookEvents *bool `json:"includeHookEvents,omitempty"`
	// Include partial/streaming message events in the output.
	IncludePartialMessages *bool `json:"includePartialMessages,omitempty"`
	// Forward subagent text and thinking blocks as assistant/user messages with `parent_tool_use_id` set.
	ForwardSubagentText *bool `json:"forwardSubagentText,omitempty"`
	// Send every user message with `client_composed: true`, so the CLI delivers the prompt text as written: no `@path` file-mention expansion and no slash-command dispatch. Default: false.
	VerbatimPrompts *bool `json:"verbatimPrompts,omitempty"`
	// Controls Claude's thinking/reasoning behavior. - `{ type: 'adaptive' }` — Claude decides when and how much to think (Opus 4.6+).
	Thinking *ThinkingConfig `json:"thinking,omitempty"`
	// Controls how much effort Claude puts into its response.
	Effort EffortLevel `json:"effort,omitempty"`
	// Deprecated: Maximum number of tokens the model can use for its thinking/reasoning process.
	MaxThinkingTokens *int `json:"maxThinkingTokens,omitempty"`
	// Maximum number of conversation turns before the query stops.
	MaxTurns *int `json:"maxTurns,omitempty"`
	// Maximum budget in USD for the query.
	MaxBudgetUSD *float64 `json:"maxBudgetUsd,omitempty"`
	// API-side task budget in tokens. (alpha)
	TaskBudget *OptionsTaskBudget `json:"taskBudget,omitempty"`
	// MCP (Model Context Protocol) server configurations.
	MCPServers map[string]McpServerConfig `json:"mcpServers,omitempty"`
	// Claude model to use.
	Model string `json:"model,omitempty"`
	// Output format configuration for structured responses.
	OutputFormat *OutputFormat `json:"outputFormat,omitempty"`
	// Path to the Claude Code executable.
	PathToClaudeCodeExecutable string `json:"pathToClaudeCodeExecutable,omitempty"`
	// Permission mode for the session.
	PermissionMode PermissionMode `json:"permissionMode,omitempty"`
	// Custom workflow instructions for plan mode.
	PlanModeInstructions string `json:"planModeInstructions,omitempty"`
	// Must be set to `true` when using `permissionMode: 'bypassPermissions'`.
	AllowDangerouslySkipPermissions *bool `json:"allowDangerouslySkipPermissions,omitempty"`
	// MCP tool name to use for permission prompts.
	PermissionPromptToolName string `json:"permissionPromptToolName,omitempty"`
	// Who answers permission prompts. (one of "host", "none")
	PermissionPrompts string `json:"permissionPrompts,omitempty"`
	// Load plugins for this session.
	Plugins []PluginConfig `json:"plugins,omitempty"`
	// How `plugins` reach the Claude Code process. - `'argv'` (default) - One `--plugin-dir <path>` flag per plugin. (one of "argv", "initialize")
	PluginDelivery string `json:"pluginDelivery,omitempty"`
	// Enable prompt suggestions.
	PromptSuggestions *bool `json:"promptSuggestions,omitempty"`
	// Enable periodic AI-generated progress summaries for running subagents.
	AgentProgressSummaries *bool `json:"agentProgressSummaries,omitempty"`
	// Session ID to resume.
	Resume string `json:"resume,omitempty"`
	// Use a specific session ID for the conversation instead of an auto-generated one.
	SessionID string `json:"sessionId,omitempty"`
	// When resuming, only resume messages up to and including the message with this UUID.
	ResumeSessionAt string `json:"resumeSessionAt,omitempty"`
	// With `resumeSessionAt`: declares the prompt UUID of the turn this truncating resume intends to discard.
	ResumeDropsTurn string `json:"resumeDropsTurn,omitempty"`
	// Sandbox settings for command execution isolation.
	Sandbox *SandboxSettings `json:"sandbox,omitempty"`
	// Additional settings to apply. (TS: string | Settings)
	Settings json.RawMessage `json:"settings,omitempty"`
	// Policy-tier settings supplied by the spawning parent process.
	ManagedSettings Settings `json:"managedSettings,omitempty"`
	// Control which filesystem settings to load. - `'user'` - Global user settings (`~/.claude/settings.json`) - `'project'` - Project settings (`.claude/settings.json`) - `'local'` - Local settings...
	SettingSources []SettingSource `json:"settingSources,omitempty"`
	// Skills to enable for the main session. (TS: string[] | 'all')
	Skills json.RawMessage `json:"skills,omitempty"`
	// Enable debug mode for the Claude Code process.
	Debug *bool `json:"debug,omitempty"`
	// Write debug logs to a specific file path.
	DebugFile string `json:"debugFile,omitempty"`
	// True when the host wants the CLI stderr stream (TS: the stderr callback).
	Stderr bool `json:"stderr,omitempty"`
	// Only use MCP servers passed via the `mcpServers` option (and servers declared by explicitly-passed agent definitions in `agents`), ignoring all other MCP configurations: project `.mcp.json`, user settings, plugins,...
	StrictMCPConfig *bool `json:"strictMcpConfig,omitempty"`
	// System prompt configuration. - `string` - Use a custom system prompt - `string[]` - Use a custom system prompt as an array of blocks; include `SYSTEM_PROMPT_DYNAMIC_BOUNDARY` as a standalone element to mark the split... (TS: string | string[] | { type: 'custom'; prompt: string | string[]; snapshot?: boolean; } | { type: 'preset'; preset: 'claude_code'; append?: string; exc)
	SystemPrompt json.RawMessage `json:"systemPrompt,omitempty"`
	// Custom title for a new session.
	Title string `json:"title,omitempty"`
}

// PermissionDecisionClassification is a string enum. Classification of this permission decision for telemetry. (TS: PermissionDecisionClassification)
type PermissionDecisionClassification string

const (
	PermissionDecisionClassificationUserTemporary PermissionDecisionClassification = "user_temporary"
	PermissionDecisionClassificationUserPermanent PermissionDecisionClassification = "user_permanent"
	PermissionDecisionClassificationUserReject    PermissionDecisionClassification = "user_reject"
)

// PermissionDeniedHookSpecificOutput (TS: PermissionDeniedHookSpecificOutput)
type PermissionDeniedHookSpecificOutput struct {
	// (always "PermissionDenied")
	HookEventName string `json:"hookEventName"`
	Retry         *bool  `json:"retry,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionDeniedHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PermissionDeniedHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionDeniedHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PermissionDeniedHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PermissionDenied"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRequestHookSpecificOutputDecisionAllow - variant of PermissionRequestHookSpecificOutputDecision
type PermissionRequestHookSpecificOutputDecisionAllow struct {
	// (always "allow")
	Behavior           string             `json:"behavior"`
	UpdatedInput       map[string]any     `json:"updatedInput,omitempty"`
	UpdatedPermissions []PermissionUpdate `json:"updatedPermissions,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRequestHookSpecificOutputDecisionAllow) UnmarshalJSON(b []byte) error {
	type plain PermissionRequestHookSpecificOutputDecisionAllow
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRequestHookSpecificOutputDecisionAllow) MarshalJSON() ([]byte, error) {
	type plain PermissionRequestHookSpecificOutputDecisionAllow
	if v.Behavior == "" {
		v.Behavior = "allow"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRequestHookSpecificOutputDecisionDeny - variant of PermissionRequestHookSpecificOutputDecision
type PermissionRequestHookSpecificOutputDecisionDeny struct {
	// (always "deny")
	Behavior  string `json:"behavior"`
	Message   string `json:"message,omitempty"`
	Interrupt *bool  `json:"interrupt,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRequestHookSpecificOutputDecisionDeny) UnmarshalJSON(b []byte) error {
	type plain PermissionRequestHookSpecificOutputDecisionDeny
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRequestHookSpecificOutputDecisionDeny) MarshalJSON() ([]byte, error) {
	type plain PermissionRequestHookSpecificOutputDecisionDeny
	if v.Behavior == "" {
		v.Behavior = "deny"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRequestHookSpecificOutput (TS: PermissionRequestHookSpecificOutput)
type PermissionRequestHookSpecificOutput struct {
	// (always "PermissionRequest")
	HookEventName string                                      `json:"hookEventName"`
	Decision      PermissionRequestHookSpecificOutputDecision `json:"decision"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRequestHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PermissionRequestHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRequestHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PermissionRequestHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PermissionRequest"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionResultAllow - variant of PermissionResult
type PermissionResultAllow struct {
	// (always "allow")
	Behavior               string                           `json:"behavior"`
	UpdatedInput           map[string]any                   `json:"updatedInput,omitempty"`
	UpdatedPermissions     []PermissionUpdate               `json:"updatedPermissions,omitempty"`
	ToolUseID              string                           `json:"toolUseID,omitempty"`
	DecisionClassification PermissionDecisionClassification `json:"decisionClassification,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionResultAllow) UnmarshalJSON(b []byte) error {
	type plain PermissionResultAllow
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionResultAllow) MarshalJSON() ([]byte, error) {
	type plain PermissionResultAllow
	if v.Behavior == "" {
		v.Behavior = "allow"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionResultDeny - variant of PermissionResult
type PermissionResultDeny struct {
	// (always "deny")
	Behavior               string                           `json:"behavior"`
	Message                string                           `json:"message"`
	Interrupt              *bool                            `json:"interrupt,omitempty"`
	ToolUseID              string                           `json:"toolUseID,omitempty"`
	DecisionClassification PermissionDecisionClassification `json:"decisionClassification,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionResultDeny) UnmarshalJSON(b []byte) error {
	type plain PermissionResultDeny
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionResultDeny) MarshalJSON() ([]byte, error) {
	type plain PermissionResultDeny
	if v.Behavior == "" {
		v.Behavior = "deny"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PolicySettingsOrigin is a string enum. Which policy sub-source supplied a `'managed'` value. (alpha) (TS: PolicySettingsOrigin)
type PolicySettingsOrigin string

const (
	PolicySettingsOriginHelper PolicySettingsOrigin = "helper"
	PolicySettingsOriginRemote PolicySettingsOrigin = "remote"
	PolicySettingsOriginPlist  PolicySettingsOrigin = "plist"
	PolicySettingsOriginHklm   PolicySettingsOrigin = "hklm"
	PolicySettingsOriginFile   PolicySettingsOrigin = "file"
	PolicySettingsOriginParent PolicySettingsOrigin = "parent"
	PolicySettingsOriginHkcu   PolicySettingsOrigin = "hkcu"
)

// PostModelSwitchHookSpecificOutput (TS: PostModelSwitchHookSpecificOutput)
type PostModelSwitchHookSpecificOutput struct {
	// (always "PostModelSwitch")
	HookEventName string `json:"hookEventName"`
	// Reaches the model with the next request the new model serves
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostModelSwitchHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PostModelSwitchHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostModelSwitchHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PostModelSwitchHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PostModelSwitch"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostToolBatchHookSpecificOutput (TS: PostToolBatchHookSpecificOutput)
type PostToolBatchHookSpecificOutput struct {
	// (always "PostToolBatch")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolBatchHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PostToolBatchHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolBatchHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PostToolBatchHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PostToolBatch"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostToolUseFailureHookSpecificOutput (TS: PostToolUseFailureHookSpecificOutput)
type PostToolUseFailureHookSpecificOutput struct {
	// (always "PostToolUseFailure")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolUseFailureHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PostToolUseFailureHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolUseFailureHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PostToolUseFailureHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PostToolUseFailure"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PostToolUseHookSpecificOutput (TS: PostToolUseHookSpecificOutput)
type PostToolUseHookSpecificOutput struct {
	// (always "PostToolUse")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Host-asserted context shown to the auto-mode permission classifier alongside this tool call's result.
	ClassifierContext string `json:"classifierContext,omitempty"`
	// Replaces the tool output before it is sent to the model
	UpdatedToolOutput json.RawMessage `json:"updatedToolOutput,omitempty"`
	// Replaces the output for MCP tools only.
	UpdatedMCPToolOutput json.RawMessage `json:"updatedMCPToolOutput,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PostToolUseHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PostToolUseHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PostToolUseHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PostToolUseHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PostToolUse"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PreModelSwitchHookSpecificOutput (TS: PreModelSwitchHookSpecificOutput)
type PreModelSwitchHookSpecificOutput struct {
	// (always "PreModelSwitch")
	HookEventName string `json:"hookEventName"`
	// Same contract as PreToolUse: allow proceeds (skipping the interactive cache-miss confirm), deny cancels the switch, ask asks the user to confirm (a headless session refuses instead) (one of "allow", "deny", "ask")
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PreModelSwitchHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PreModelSwitchHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PreModelSwitchHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PreModelSwitchHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PreModelSwitch"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PreToolUseHookSpecificOutput (TS: PreToolUseHookSpecificOutput)
type PreToolUseHookSpecificOutput struct {
	// (always "PreToolUse")
	HookEventName            string                 `json:"hookEventName"`
	PermissionDecision       HookPermissionDecision `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string                 `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             map[string]any         `json:"updatedInput,omitempty"`
	AdditionalContext        string                 `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PreToolUseHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain PreToolUseHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PreToolUseHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain PreToolUseHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "PreToolUse"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ResolvedSettingSource is a string enum. Source that contributed an effective setting value. (alpha) (TS: ResolvedSettingSource)
type ResolvedSettingSource string

const (
	ResolvedSettingSourceUser    ResolvedSettingSource = "user"
	ResolvedSettingSourceProject ResolvedSettingSource = "project"
	ResolvedSettingSourceLocal   ResolvedSettingSource = "local"
	ResolvedSettingSourceManaged ResolvedSettingSource = "managed"
	ResolvedSettingSourceFlag    ResolvedSettingSource = "flag"
)

// ProvenanceEntry - Per-key provenance entry. (alpha) (TS: ProvenanceEntry)
type ProvenanceEntry struct {
	Source ResolvedSettingSource `json:"source"`
	// Absolute path to the settings file, for filesystem-backed sources.
	Path string `json:"path,omitempty"`
	// Which policy sub-source supplied the value, when `source === 'managed'`.
	PolicyOrigin PolicySettingsOrigin `json:"policyOrigin,omitempty"`
}

// ResolveSettingsOptions - Options for . (alpha) (TS: ResolveSettingsOptions)
type ResolveSettingsOptions struct {
	// Directory to resolve project/local settings relative to.
	Cwd string `json:"cwd,omitempty"`
	// Which filesystem settings sources to load.
	SettingSources []SettingSource `json:"settingSources,omitempty"`
	// Restrictive policy-tier settings — equivalent to `Options.managedSettings` on `query()`.
	ManagedSettings Settings `json:"managedSettings,omitempty"`
	// Server-managed settings payload (the result of fetching `/api/claude_code/settings`).
	ServerManagedSettings Settings `json:"serverManagedSettings,omitempty"`
}

// ResolvedSettingsSource - inline object (field source of ResolvedSettings)
type ResolvedSettingsSource struct {
	Source       ResolvedSettingSource `json:"source"`
	Settings     Settings              `json:"settings"`
	Path         string                `json:"path,omitempty"`
	PolicyOrigin PolicySettingsOrigin  `json:"policyOrigin,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ResolvedSettingsSource) UnmarshalJSON(b []byte) error {
	type plain ResolvedSettingsSource
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ResolvedSettingsSource) MarshalJSON() ([]byte, error) {
	type plain ResolvedSettingsSource
	return marshalKeep(plain(v), v.Raw)
}

// ResolvedSettingsMachinePolicy - inline object (field machinePolicy of ResolvedSettings)
type ResolvedSettingsMachinePolicy struct {
	// `false`, or the kinds of source that may be there and did not load: `'profile'` (a macOS MDM profile), `'registry'` (the Windows HKLM policy value; under WSL also the Windows folder's managed-settings files while WSL... (TS: false | Array<'profile' | 'registry' | 'file'>)
	DidNotLoad json.RawMessage `json:"didNotLoad"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ResolvedSettingsMachinePolicy) UnmarshalJSON(b []byte) error {
	type plain ResolvedSettingsMachinePolicy
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ResolvedSettingsMachinePolicy) MarshalJSON() ([]byte, error) {
	type plain ResolvedSettingsMachinePolicy
	return marshalKeep(plain(v), v.Raw)
}

// ResolvedSettings - Result of . (alpha) (TS: ResolvedSettings)
type ResolvedSettings struct {
	// Merged settings after applying all enabled sources in precedence order.
	Effective Settings `json:"effective"`
	// For each top-level key in `effective`, which source supplied the value.
	Provenance map[string]ProvenanceEntry `json:"provenance"`
	// Per-source raw settings, low→high precedence.
	Sources []ResolvedSettingsSource `json:"sources"`
	// since 0.3.323 (not in 0.3.311)
	// What became of this machine's own policy sources, in the host's own read (not a Claude Code session's).
	MachinePolicy *ResolvedSettingsMachinePolicy `json:"machinePolicy,omitempty"`
}

// RewindFilesResult - Result of a rewindFiles operation. (TS: RewindFilesResult)
type RewindFilesResult struct {
	CanRewind    bool     `json:"canRewind"`
	Error        string   `json:"error,omitempty"`
	FilesChanged []string `json:"filesChanged,omitempty"`
	Insertions   *int     `json:"insertions,omitempty"`
	Deletions    *int     `json:"deletions,omitempty"`
	// Count of tracked files NOT restored or deleted because a symlink, hard link, or other non-regular file was detected at the tracked path, its parent directory no longer resolves to where it pointed when the checkpoint...
	SkippedLinks *int `json:"skippedLinks,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RewindFilesResult) UnmarshalJSON(b []byte) error {
	type plain RewindFilesResult
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RewindFilesResult) MarshalJSON() ([]byte, error) {
	type plain RewindFilesResult
	return marshalKeep(plain(v), v.Raw)
}

// APIRetryMessageNoResponse - inline object (field no_response of APIRetryMessage)
type APIRetryMessageNoResponse struct {
	WaitedMs    float64 `json:"waited_ms"`
	RetryWaitMs float64 `json:"retry_wait_ms"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *APIRetryMessageNoResponse) UnmarshalJSON(b []byte) error {
	type plain APIRetryMessageNoResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v APIRetryMessageNoResponse) MarshalJSON() ([]byte, error) {
	type plain APIRetryMessageNoResponse
	return marshalKeep(plain(v), v.Raw)
}

// APIRetryMessage - Emitted when an API request fails with a retryable error and will be retried after a delay. error_status is null for connection errors (e.g. timeouts) that had no HTTP response. (TS: SDKAPIRetryMessage)
type APIRetryMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "api_retry")
	Subtype      string  `json:"subtype"`
	Attempt      int     `json:"attempt"`
	MaxRetries   int     `json:"max_retries"`
	RetryDelayMs float64 `json:"retry_delay_ms"`
	// (nullable)
	ErrorStatus *int                  `json:"error_status"`
	Error       AssistantMessageError `json:"error"`
	// Present only when the API sent no response headers within the first-byte window (CLAUDE_STREAM_FIRST_BYTE_TIMEOUT_MS): waited_ms is how long the failed attempt waited for headers, retry_wait_ms how long the retry...
	NoResponse *APIRetryMessageNoResponse `json:"no_response,omitempty"`
	UUID       string                     `json:"uuid"`
	SessionID  string                     `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *APIRetryMessage) UnmarshalJSON(b []byte) error {
	type plain APIRetryMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v APIRetryMessage) MarshalJSON() ([]byte, error) {
	type plain APIRetryMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "api_retry"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ActiveGoalMessageValue - inline object (field value of ActiveGoalMessage)
type ActiveGoalMessageValue struct {
	Condition     string  `json:"condition"`
	Iterations    int     `json:"iterations"`
	SetAt         float64 `json:"set_at"`
	TokensAtStart float64 `json:"tokens_at_start"`
	LastReason    string  `json:"last_reason,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ActiveGoalMessageValue) UnmarshalJSON(b []byte) error {
	type plain ActiveGoalMessageValue
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ActiveGoalMessageValue) MarshalJSON() ([]byte, error) {
	type plain ActiveGoalMessageValue
	return marshalKeep(plain(v), v.Raw)
}

// ActiveGoalMessage - Emitted when the user's /goal Stop hook reports met (clears) or not-yet-met (bumps iterations + last_reason). (TS: SDKActiveGoalMessage)
type ActiveGoalMessage struct {
	// (always "active_goal")
	Type string `json:"type"`
	// (nullable)
	Value     *ActiveGoalMessageValue `json:"value"`
	UUID      string                  `json:"uuid"`
	SessionID string                  `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ActiveGoalMessage) UnmarshalJSON(b []byte) error {
	type plain ActiveGoalMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ActiveGoalMessage) MarshalJSON() ([]byte, error) {
	type plain ActiveGoalMessage
	if v.Type == "" {
		v.Type = "active_goal"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsageOverLimit - inline object (field over_limit of ContextUsage)
type ContextUsageOverLimit struct {
	TokensOver float64 `json:"tokens_over"`
	// (one of "hard_limit", "compaction_window")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsageOverLimit) UnmarshalJSON(b []byte) error {
	type plain ContextUsageOverLimit
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsageOverLimit) MarshalJSON() ([]byte, error) {
	type plain ContextUsageOverLimit
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsageCategory - One row of the /context usage-by-category breakdown. (TS: SDKContextUsageCategory)
type ContextUsageCategory struct {
	// Display name of the row as the CLI renders it, e.g.
	Name   string `json:"name"`
	Tokens int    `json:"tokens"`
	// What the row is: 'used' content occupies the window; 'free' is the remaining window; 'buffer' is the compaction reserve (autocompact or manual); 'deferred' rows are out-of-window tool schemas — listed for awareness,... (one of "used", "free", "buffer", "deferred")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsageCategory) UnmarshalJSON(b []byte) error {
	type plain ContextUsageCategory
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsageCategory) MarshalJSON() ([]byte, error) {
	type plain ContextUsageCategory
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsageMCPTool - inline object (field mcp_tool of ContextUsage)
type ContextUsageMCPTool struct {
	// Wire name, e.g.
	Name       string `json:"name"`
	ServerName string `json:"server_name"`
	Tokens     int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsageMCPTool) UnmarshalJSON(b []byte) error {
	type plain ContextUsageMCPTool
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsageMCPTool) MarshalJSON() ([]byte, error) {
	type plain ContextUsageMCPTool
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsageMemoryFile - inline object (field memory_file of ContextUsage)
type ContextUsageMemoryFile struct {
	Path string `json:"path"`
	// Display label of the memory-file source, e.g.
	Type   string `json:"type"`
	Tokens int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsageMemoryFile) UnmarshalJSON(b []byte) error {
	type plain ContextUsageMemoryFile
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsageMemoryFile) MarshalJSON() ([]byte, error) {
	type plain ContextUsageMemoryFile
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsageAgent - inline object (field agent of ContextUsage)
type ContextUsageAgent struct {
	AgentType string `json:"agent_type"`
	// Raw source identifier, e.g.
	Source string `json:"source"`
	Tokens int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsageAgent) UnmarshalJSON(b []byte) error {
	type plain ContextUsageAgent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsageAgent) MarshalJSON() ([]byte, error) {
	type plain ContextUsageAgent
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsageSkill - inline object (field skill of ContextUsage)
type ContextUsageSkill struct {
	Name string `json:"name"`
	// Raw source identifier, e.g.
	Source     string `json:"source"`
	PluginName string `json:"plugin_name,omitempty"`
	Tokens     int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsageSkill) UnmarshalJSON(b []byte) error {
	type plain ContextUsageSkill
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsageSkill) MarshalJSON() ([]byte, error) {
	type plain ContextUsageSkill
	return marshalKeep(plain(v), v.Raw)
}

// ContextUsage - Structured twin of the /context report — the data a client needs to render the context-usage card without parsing the markdown table. (TS: SDKContextUsage)
type ContextUsage struct {
	// Main-loop model the usage was computed for.
	Model string `json:"model"`
	// Estimated tokens in use, unclamped — may exceed raw_max_tokens when over limit.
	TotalTokens int `json:"total_tokens"`
	// The window usage is measured against: the resolved autocompact window — the model's believed limit, or a smaller compaction-policy window (a configured value, or e.g. the 200K boundary on 1M-window models).
	RawMaxTokens int `json:"raw_max_tokens"`
	// Rounded total_tokens / raw_max_tokens, 0-100+.
	Percentage float64 `json:"percentage"`
	// Present when total_tokens exceeds raw_max_tokens. kind says how the window was resolved, not whether the API will accept the next request: 'hard_limit' means the window is the model's believed limit (the API will...
	OverLimit   *ContextUsageOverLimit   `json:"over_limit,omitempty"`
	Categories  []ContextUsageCategory   `json:"categories"`
	MCPTools    []ContextUsageMCPTool    `json:"mcp_tools"`
	MemoryFiles []ContextUsageMemoryFile `json:"memory_files"`
	Agents      []ContextUsageAgent      `json:"agents"`
	// Omitted when no skills contribute tokens.
	Skills []ContextUsageSkill `json:"skills,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ContextUsage) UnmarshalJSON(b []byte) error {
	type plain ContextUsage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ContextUsage) MarshalJSON() ([]byte, error) {
	type plain ContextUsage
	return marshalKeep(plain(v), v.Raw)
}

// UsageReportSession - inline object (field session of UsageReport)
type UsageReportSession struct {
	TotalCostUSD       float64               `json:"total_cost_usd"`
	TotalAPIDurationMs float64               `json:"total_api_duration_ms"`
	TotalDurationMs    float64               `json:"total_duration_ms"`
	TotalLinesAdded    int                   `json:"total_lines_added"`
	TotalLinesRemoved  int                   `json:"total_lines_removed"`
	ModelUsage         map[string]ModelUsage `json:"model_usage"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReportSession) UnmarshalJSON(b []byte) error {
	type plain UsageReportSession
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReportSession) MarshalJSON() ([]byte, error) {
	type plain UsageReportSession
	return marshalKeep(plain(v), v.Raw)
}

// UsageReportRateLimitsLimitScopeModel - inline object (field model of UsageReportRateLimitsLimitScope)
type UsageReportRateLimitsLimitScopeModel struct {
	DisplayName string `json:"display_name"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReportRateLimitsLimitScopeModel) UnmarshalJSON(b []byte) error {
	type plain UsageReportRateLimitsLimitScopeModel
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReportRateLimitsLimitScopeModel) MarshalJSON() ([]byte, error) {
	type plain UsageReportRateLimitsLimitScopeModel
	return marshalKeep(plain(v), v.Raw)
}

// UsageReportRateLimitsLimitScope - inline object (field scope of UsageReportRateLimitsLimit)
type UsageReportRateLimitsLimitScope struct {
	// (optional, nullable)
	Model *UsageReportRateLimitsLimitScopeModel `json:"model,omitempty"`
	// (optional, nullable)
	Surface *UsageReportRateLimitsLimitScopeModel `json:"surface,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReportRateLimitsLimitScope) UnmarshalJSON(b []byte) error {
	type plain UsageReportRateLimitsLimitScope
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReportRateLimitsLimitScope) MarshalJSON() ([]byte, error) {
	type plain UsageReportRateLimitsLimitScope
	return marshalKeep(plain(v), v.Raw)
}

// UsageReportRateLimitsLimit - inline object (field limit of UsageReportRateLimits)
type UsageReportRateLimitsLimit struct {
	// The server's meter kind, e.g.
	Kind string `json:"kind"`
	// The server's row group, e.g.
	Group string `json:"group"`
	// Share of the window used, 0-100.
	Percent float64 `json:"percent"`
	// ISO 8601 timestamp when the window resets. (nullable)
	ResetsAt *string `json:"resets_at"`
	// What a scoped row is for, a model or a surface, with the server's display label. (optional, nullable)
	Scope *UsageReportRateLimitsLimitScope `json:"scope,omitempty"`
	// The server's reading of the row for a meter's colour, e.g.
	Severity string `json:"severity"`
	// The server's headline pick: the row a single-value indicator shows.
	IsActive bool `json:"is_active"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReportRateLimitsLimit) UnmarshalJSON(b []byte) error {
	type plain UsageReportRateLimitsLimit
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReportRateLimitsLimit) MarshalJSON() ([]byte, error) {
	type plain UsageReportRateLimitsLimit
	return marshalKeep(plain(v), v.Raw)
}

// UsageReportRateLimitsExtraUsage - inline object (field extra_usage of UsageReportRateLimits)
type UsageReportRateLimitsExtraUsage struct {
	IsEnabled bool `json:"is_enabled"`
	// (nullable)
	MonthlyLimit *float64 `json:"monthly_limit"`
	// (nullable)
	UsedCredits *float64 `json:"used_credits"`
	// (nullable)
	Utilization *float64 `json:"utilization"`
	// (optional, nullable)
	Currency *string `json:"currency,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReportRateLimitsExtraUsage) UnmarshalJSON(b []byte) error {
	type plain UsageReportRateLimitsExtraUsage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReportRateLimitsExtraUsage) MarshalJSON() ([]byte, error) {
	type plain UsageReportRateLimitsExtraUsage
	return marshalKeep(plain(v), v.Raw)
}

// UsageReportRateLimits - inline object (field rate_limits of UsageReport)
type UsageReportRateLimits struct {
	// The server's usage rows (the usage endpoint's limits[]), as sent: which meters apply, their scope, labels, severity and order are the server's, so a client renders them verbatim and a new meter needs no client release. (nullable)
	Limits []UsageReportRateLimitsLimit `json:"limits"`
	// Extra-usage (overage) spend for the billing period, when the plan has it. (optional, nullable)
	ExtraUsage *UsageReportRateLimitsExtraUsage `json:"extra_usage,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReportRateLimits) UnmarshalJSON(b []byte) error {
	type plain UsageReportRateLimits
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReportRateLimits) MarshalJSON() ([]byte, error) {
	type plain UsageReportRateLimits
	return marshalKeep(plain(v), v.Raw)
}

// UsageReport - Structured twin of a /usage result, carried beside its text: the session's totals, the plan's usage rows as the server sent them and the extra-usage spend, and nothing else from the usage body (the get_usage control... (TS: SDKUsageReport)
type UsageReport struct {
	// Cost and usage accumulated by the current session.
	Session UsageReportSession `json:"session"`
	// The plan's usage rows and extra-usage spend from the claude.ai usage endpoint; null when the CLI could not fetch them (no plan on this lane, or a token without the profile scope). (nullable)
	RateLimits *UsageReportRateLimits `json:"rate_limits"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UsageReport) UnmarshalJSON(b []byte) error {
	type plain UsageReport
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UsageReport) MarshalJSON() ([]byte, error) {
	type plain UsageReport
	return marshalKeep(plain(v), v.Raw)
}

// AssistantMessage - An assistant message. (TS: SDKAssistantMessage)
type AssistantMessage struct {
	// (always "assistant")
	Type string `json:"type"`
	// Shaped like an Anthropic Messages API Message object (role "assistant"): id, model, content blocks (text, thinking, tool_use, ...), stop_reason and usage.
	Message APIMessage `json:"message"`
	// (nullable)
	ParentToolUseID *string               `json:"parent_tool_use_id"`
	Error           AssistantMessageError `json:"error,omitempty"`
	UUID            string                `json:"uuid"`
	SessionID       string                `json:"session_id"`
	RequestID       string                `json:"request_id,omitempty"`
	// Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on an assistant message each time that send changes — the turn's FIRST top-level assistant message (which may carry only a...
	UserMessageUUID string `json:"user_message_uuid,omitempty"`
	// Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host merged into this one turn (several messages sent close together run as one...
	UserMessageUUIDs []string `json:"user_message_uuids,omitempty"`
	// Why this frame's turn continues one that a worker restart interrupted.
	ResumeReason string `json:"resume_reason,omitempty"`
	// This turn continued the preceding truncated assistant turn inside its trailing signed thinking block (max-output-tokens recovery). (TS literal `true`)
	ResumedFromIncompleteThinking bool `json:"resumed_from_incomplete_thinking,omitempty"`
	// Wire uuids of previously-delivered messages that this message replaces (refusal-fallback supersede).
	Supersedes []string `json:"supersedes,omitempty"`
	// True when this assistant message was truncated by an interrupt/abort before the stream completed: stop_reason was never received and the content may end mid-word. (TS literal `true`)
	Aborted bool `json:"aborted,omitempty"`
	// ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed.
	AgentID string `json:"agent_id,omitempty"`
	// Subagent type that produced this message.
	SubagentType string `json:"subagent_type,omitempty"`
	// Description of the subagent task that produced this message.
	TaskDescription string `json:"task_description,omitempty"`
	// ISO timestamp of when this content block finished on the originating process.
	Timestamp string `json:"timestamp,omitempty"`
	// Structured twin of the /context report, carried on the synthetic assistant message that delivers the markdown table.
	ContextUsage *ContextUsage `json:"context_usage,omitempty"`
	// Structured twin of the /usage report, carried on the synthetic assistant message that delivers its text: the session totals, the plan's usage rows and extra-usage spend, for remote clients that render a card from data.
	UsageReport *UsageReport `json:"usage_report,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *AssistantMessage) UnmarshalJSON(b []byte) error {
	type plain AssistantMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v AssistantMessage) MarshalJSON() ([]byte, error) {
	type plain AssistantMessage
	if v.Type == "" {
		v.Type = "assistant"
	}
	return marshalKeep(plain(v), v.Raw)
}

// AuthStatusMessage (TS: SDKAuthStatusMessage)
type AuthStatusMessage struct {
	// (always "auth_status")
	Type             string   `json:"type"`
	IsAuthenticating bool     `json:"isAuthenticating"`
	Output           []string `json:"output"`
	Error            string   `json:"error,omitempty"`
	UUID             string   `json:"uuid"`
	SessionID        string   `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *AuthStatusMessage) UnmarshalJSON(b []byte) error {
	type plain AuthStatusMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v AuthStatusMessage) MarshalJSON() ([]byte, error) {
	type plain AuthStatusMessage
	if v.Type == "" {
		v.Type = "auth_status"
	}
	return marshalKeep(plain(v), v.Raw)
}

// BackgroundTasksChangedMessageTask - inline object (field task of BackgroundTasksChangedMessage)
type BackgroundTasksChangedMessageTask struct {
	TaskID string `json:"task_id"`
	// Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
	RunID    string `json:"run_id,omitempty"`
	TaskType string `json:"task_type"`
	// Agent type of a local_agent task, such as "general-purpose" or a custom agent's name; "main-session" for a backgrounded main session.
	SubagentType string `json:"subagent_type,omitempty"`
	Description  string `json:"description"`
	// task_id of the subagent (local_agent) task whose agent launched this subagent, shell, monitor or workflow task.
	ParentTaskID string `json:"parent_task_id,omitempty"`
	// True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude them from activity indicators.
	Ambient *bool `json:"ambient,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BackgroundTasksChangedMessageTask) UnmarshalJSON(b []byte) error {
	type plain BackgroundTasksChangedMessageTask
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BackgroundTasksChangedMessageTask) MarshalJSON() ([]byte, error) {
	type plain BackgroundTasksChangedMessageTask
	return marshalKeep(plain(v), v.Raw)
}

// BackgroundTasksChangedMessage - The full set of live background tasks, emitted whenever membership changes (start, completion, kill, a foreground agent being backgrounded) or an entry's `ambient` flag or `parent_task_id` changes. (TS: SDKBackgroundTasksChangedMessage)
type BackgroundTasksChangedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "background_tasks_changed")
	Subtype string `json:"subtype"`
	// Every live background task after the change.
	Tasks     []BackgroundTasksChangedMessageTask `json:"tasks"`
	UUID      string                              `json:"uuid"`
	SessionID string                              `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *BackgroundTasksChangedMessage) UnmarshalJSON(b []byte) error {
	type plain BackgroundTasksChangedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v BackgroundTasksChangedMessage) MarshalJSON() ([]byte, error) {
	type plain BackgroundTasksChangedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "background_tasks_changed"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SlashCommand - Information about an available skill (invoked via /command syntax). (TS: SlashCommand)
type SlashCommand struct {
	// Skill name (without the leading slash)
	Name string `json:"name"`
	// Description of what the skill does
	Description string `json:"description"`
	// Hint for skill arguments (e.g., "<file>")
	ArgumentHint string `json:"argumentHint"`
	// Alternate names that resolve to this command (e.g., /cost and /stats both resolve to /usage)
	Aliases []string `json:"aliases,omitempty"`
	// True when the command is Claude Code's own; absent for a command defined by a user, project, plugin or MCP server.
	Builtin *bool `json:"builtin,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SlashCommand) UnmarshalJSON(b []byte) error {
	type plain SlashCommand
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SlashCommand) MarshalJSON() ([]byte, error) {
	type plain SlashCommand
	return marshalKeep(plain(v), v.Raw)
}

// CommandsChangedMessage - Fire-and-forget push of the full slash-command list after a mid-session change (e.g. skills discovered dynamically as the agent works in a subdirectory). (TS: SDKCommandsChangedMessage)
type CommandsChangedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "commands_changed")
	Subtype   string         `json:"subtype"`
	Commands  []SlashCommand `json:"commands"`
	UUID      string         `json:"uuid"`
	SessionID string         `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CommandsChangedMessage) UnmarshalJSON(b []byte) error {
	type plain CommandsChangedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CommandsChangedMessage) MarshalJSON() ([]byte, error) {
	type plain CommandsChangedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "commands_changed"
	}
	return marshalKeep(plain(v), v.Raw)
}

// CompactBoundaryMessageCompactMetadataPreservedSegment - inline object (field preserved_segment of CompactBoundaryMessageCompactMetadata)
type CompactBoundaryMessageCompactMetadataPreservedSegment struct {
	HeadUUID   string `json:"head_uuid"`
	AnchorUUID string `json:"anchor_uuid"`
	TailUUID   string `json:"tail_uuid"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CompactBoundaryMessageCompactMetadataPreservedSegment) UnmarshalJSON(b []byte) error {
	type plain CompactBoundaryMessageCompactMetadataPreservedSegment
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CompactBoundaryMessageCompactMetadataPreservedSegment) MarshalJSON() ([]byte, error) {
	type plain CompactBoundaryMessageCompactMetadataPreservedSegment
	return marshalKeep(plain(v), v.Raw)
}

// CompactBoundaryMessageCompactMetadataPreservedMessages - inline object (field preserved_messages of CompactBoundaryMessageCompactMetadata)
type CompactBoundaryMessageCompactMetadataPreservedMessages struct {
	AnchorUUID string   `json:"anchor_uuid"`
	UUIDs      []string `json:"uuids"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CompactBoundaryMessageCompactMetadataPreservedMessages) UnmarshalJSON(b []byte) error {
	type plain CompactBoundaryMessageCompactMetadataPreservedMessages
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CompactBoundaryMessageCompactMetadataPreservedMessages) MarshalJSON() ([]byte, error) {
	type plain CompactBoundaryMessageCompactMetadataPreservedMessages
	return marshalKeep(plain(v), v.Raw)
}

// CompactBoundaryMessageCompactMetadata - inline object (field compact_metadata of CompactBoundaryMessage)
type CompactBoundaryMessageCompactMetadata struct {
	// (one of "manual", "auto")
	Trigger    string   `json:"trigger"`
	PreTokens  int      `json:"pre_tokens"`
	PostTokens *int     `json:"post_tokens,omitempty"`
	DurationMs *float64 `json:"duration_ms,omitempty"`
	// Relink info for messagesToKeep.
	PreservedSegment *CompactBoundaryMessageCompactMetadataPreservedSegment `json:"preserved_segment,omitempty"`
	// Ordered messagesToKeep UUIDs.
	PreservedMessages *CompactBoundaryMessageCompactMetadataPreservedMessages `json:"preserved_messages,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CompactBoundaryMessageCompactMetadata) UnmarshalJSON(b []byte) error {
	type plain CompactBoundaryMessageCompactMetadata
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CompactBoundaryMessageCompactMetadata) MarshalJSON() ([]byte, error) {
	type plain CompactBoundaryMessageCompactMetadata
	return marshalKeep(plain(v), v.Raw)
}

// CompactBoundaryMessage (TS: SDKCompactBoundaryMessage)
type CompactBoundaryMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "compact_boundary")
	Subtype         string                                `json:"subtype"`
	CompactMetadata CompactBoundaryMessageCompactMetadata `json:"compact_metadata"`
	UUID            string                                `json:"uuid"`
	SessionID       string                                `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *CompactBoundaryMessage) UnmarshalJSON(b []byte) error {
	type plain CompactBoundaryMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v CompactBoundaryMessage) MarshalJSON() ([]byte, error) {
	type plain CompactBoundaryMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "compact_boundary"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ControlCancelRequest - Tells the other side that the sender no longer needs the answer to one of its own in-flight control_requests (for example a pending can_use_tool prompt after the turn was interrupted, or one that another client... (TS: SDKControlCancelRequest, not exported (wire type))
type ControlCancelRequest struct {
	// (always "control_cancel_request")
	Type string `json:"type"`
	// The request_id of the control_request being withdrawn.
	RequestID string `json:"request_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ControlCancelRequest) UnmarshalJSON(b []byte) error {
	type plain ControlCancelRequest
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ControlCancelRequest) MarshalJSON() ([]byte, error) {
	type plain ControlCancelRequest
	if v.Type == "" {
		v.Type = "control_cancel_request"
	}
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseCategory - inline object (field category of GetContextUsageResponse)
type GetContextUsageResponseCategory struct {
	// Which row this is. (one of "system_prompt", "system_tools", "mcp_tools", "mcp_instructions", "mcp_tools_deferred", "system_tools_deferred", "custom_agents", "memory_files", "skills", "messages", "autocompact_buffer", "compact_buffer", "free_space")
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	Tokens     int    `json:"tokens"`
	Color      string `json:"color"`
	IsDeferred *bool  `json:"isDeferred,omitempty"`
	// What the row is, the same classification the /context result's context_usage rows carry: 'used' content occupies the window; 'free' is the remaining window; 'buffer' is the compaction reserve; 'deferred' rows are... (one of "used", "free", "buffer", "deferred")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseCategory) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseCategory
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseCategory) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseCategory
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseGridRow - inline object (field gridRow of GetContextUsageResponse)
type GetContextUsageResponseGridRow struct {
	Color          string  `json:"color"`
	IsFilled       bool    `json:"isFilled"`
	CategoryName   string  `json:"categoryName"`
	Tokens         int     `json:"tokens"`
	Percentage     float64 `json:"percentage"`
	SquareFullness float64 `json:"squareFullness"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseGridRow) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseGridRow
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseGridRow) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseGridRow
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseMCPTool - inline object (field mcpTool of GetContextUsageResponse)
type GetContextUsageResponseMCPTool struct {
	Name       string `json:"name"`
	ServerName string `json:"serverName"`
	Tokens     int    `json:"tokens"`
	IsLoaded   *bool  `json:"isLoaded,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseMCPTool) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseMCPTool
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseMCPTool) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseMCPTool
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseMCPInstruction - inline object (field mcpInstruction of GetContextUsageResponse)
type GetContextUsageResponseMCPInstruction struct {
	// Server name, as mcp_status reports it and mcp_toggle takes it (not normalized, unlike mcpTools[].serverName).
	ServerName string `json:"serverName"`
	Tokens     int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseMCPInstruction) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseMCPInstruction
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseMCPInstruction) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseMCPInstruction
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseDeferredBuiltinTool - inline object (field deferredBuiltinTool of GetContextUsageResponse)
type GetContextUsageResponseDeferredBuiltinTool struct {
	Name     string `json:"name"`
	Tokens   int    `json:"tokens"`
	IsLoaded bool   `json:"isLoaded"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseDeferredBuiltinTool) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseDeferredBuiltinTool
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseDeferredBuiltinTool) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseDeferredBuiltinTool
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseSystemTool - inline object (field systemTool of GetContextUsageResponse)
type GetContextUsageResponseSystemTool struct {
	Name   string `json:"name"`
	Tokens int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseSystemTool) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseSystemTool
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseSystemTool) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseSystemTool
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseAgent - inline object (field agent of GetContextUsageResponse)
type GetContextUsageResponseAgent struct {
	AgentType string `json:"agentType"`
	Source    string `json:"source"`
	// Id of the providing plugin (its name and marketplace), when there is one.
	PluginID string `json:"pluginId,omitempty"`
	// Absolute path of the agent's file on the machine the CLI runs on, which over SSH or WSL is not the host's.
	Path   string `json:"path,omitempty"`
	Tokens int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseAgent) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseAgent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseAgent) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseAgent
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseSlashCommands - inline object (field slashCommands of GetContextUsageResponse)
type GetContextUsageResponseSlashCommands struct {
	TotalCommands    int `json:"totalCommands"`
	IncludedCommands int `json:"includedCommands"`
	Tokens           int `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseSlashCommands) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseSlashCommands
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseSlashCommands) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseSlashCommands
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseSkillsSkillFrontmatter - inline object (field skillFrontmatter of GetContextUsageResponseSkills)
type GetContextUsageResponseSkillsSkillFrontmatter struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	// Name of the providing plugin, when there is one.
	PluginName string `json:"pluginName,omitempty"`
	// Id of the providing plugin (its name and marketplace), when there is one.
	PluginID string `json:"pluginId,omitempty"`
	// Absolute path of the skill's file on the machine the CLI runs on, which over SSH or WSL is not the host's.
	Path   string `json:"path,omitempty"`
	Tokens int    `json:"tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseSkillsSkillFrontmatter) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseSkillsSkillFrontmatter
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseSkillsSkillFrontmatter) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseSkillsSkillFrontmatter
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseSkills - inline object (field skills of GetContextUsageResponse)
type GetContextUsageResponseSkills struct {
	TotalSkills      int                                             `json:"totalSkills"`
	IncludedSkills   int                                             `json:"includedSkills"`
	Tokens           int                                             `json:"tokens"`
	SkillFrontmatter []GetContextUsageResponseSkillsSkillFrontmatter `json:"skillFrontmatter"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseSkills) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseSkills
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseSkills) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseSkills
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseMessageBreakdownToolCallsByType - inline object (field toolCallsByType of GetContextUsageResponseMessageBreakdown)
type GetContextUsageResponseMessageBreakdownToolCallsByType struct {
	Name         string `json:"name"`
	CallTokens   int    `json:"callTokens"`
	ResultTokens int    `json:"resultTokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseMessageBreakdownToolCallsByType) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseMessageBreakdownToolCallsByType
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseMessageBreakdownToolCallsByType) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseMessageBreakdownToolCallsByType
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseMessageBreakdown - inline object (field messageBreakdown of GetContextUsageResponse)
type GetContextUsageResponseMessageBreakdown struct {
	ToolCallTokens          int                                                      `json:"toolCallTokens"`
	ToolResultTokens        int                                                      `json:"toolResultTokens"`
	AttachmentTokens        int                                                      `json:"attachmentTokens"`
	AssistantMessageTokens  int                                                      `json:"assistantMessageTokens"`
	UserMessageTokens       int                                                      `json:"userMessageTokens"`
	RedirectedContextTokens int                                                      `json:"redirectedContextTokens"`
	UnattributedTokens      int                                                      `json:"unattributedTokens"`
	ToolCallsByType         []GetContextUsageResponseMessageBreakdownToolCallsByType `json:"toolCallsByType"`
	AttachmentsByType       []GetContextUsageResponseSystemTool                      `json:"attachmentsByType"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseMessageBreakdown) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseMessageBreakdown
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseMessageBreakdown) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseMessageBreakdown
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponseAPIUsage - inline object (field apiUsage of GetContextUsageResponse)
type GetContextUsageResponseAPIUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponseAPIUsage) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponseAPIUsage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponseAPIUsage) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponseAPIUsage
	return marshalKeep(plain(v), v.Raw)
}

// GetContextUsageResponse - Breakdown of current context window usage by category (system prompt, tools, messages, etc.). (TS: SDKControlGetContextUsageResponse)
type GetContextUsageResponse struct {
	Categories   []GetContextUsageResponseCategory  `json:"categories"`
	TotalTokens  int                                `json:"totalTokens"`
	MaxTokens    int                                `json:"maxTokens"`
	RawMaxTokens int                                `json:"rawMaxTokens"`
	Percentage   float64                            `json:"percentage"`
	GridRows     [][]GetContextUsageResponseGridRow `json:"gridRows"`
	Model        string                             `json:"model"`
	MemoryFiles  []ContextUsageMemoryFile           `json:"memoryFiles"`
	MCPTools     []GetContextUsageResponseMCPTool   `json:"mcpTools"`
	// The 'mcp_instructions' row by server.
	MCPInstructions      []GetContextUsageResponseMCPInstruction      `json:"mcpInstructions,omitempty"`
	DeferredBuiltinTools []GetContextUsageResponseDeferredBuiltinTool `json:"deferredBuiltinTools,omitempty"`
	SystemTools          []GetContextUsageResponseSystemTool          `json:"systemTools,omitempty"`
	SystemPromptSections []GetContextUsageResponseSystemTool          `json:"systemPromptSections,omitempty"`
	Agents               []GetContextUsageResponseAgent               `json:"agents"`
	SlashCommands        *GetContextUsageResponseSlashCommands        `json:"slashCommands,omitempty"`
	Skills               *GetContextUsageResponseSkills               `json:"skills,omitempty"`
	AutoCompactThreshold *float64                                     `json:"autoCompactThreshold,omitempty"`
	IsAutoCompactEnabled bool                                         `json:"isAutoCompactEnabled"`
	MessageBreakdown     *GetContextUsageResponseMessageBreakdown     `json:"messageBreakdown,omitempty"`
	// (nullable)
	APIUsage *GetContextUsageResponseAPIUsage `json:"apiUsage"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetContextUsageResponse) UnmarshalJSON(b []byte) error {
	type plain GetContextUsageResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetContextUsageResponse) MarshalJSON() ([]byte, error) {
	type plain GetContextUsageResponse
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponseEvent - inline object (field event of GetHooksListingResponse)
type GetHooksListingResponseEvent struct {
	// Hook event name.
	Name string `json:"name"`
	// One-line summary, as the /hooks event list shows.
	Summary string `json:"summary"`
	// Whether hooks on this event can carry a matcher.
	SupportsMatcher bool `json:"supportsMatcher"`
	// Number of listed hooks.
	HookCount int `json:"hookCount"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponseEvent) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponseEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponseEvent) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponseEvent
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponseHookEditable - inline object (field editable of GetHooksListingResponseHook)
type GetHooksListingResponseHookEditable struct {
	Matcher string         `json:"matcher"`
	Config  map[string]any `json:"config"`
	// (TS literal `true`)
	HeadersRedacted bool `json:"headersRedacted,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponseHookEditable) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponseHookEditable
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponseHookEditable) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponseHookEditable
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponseHook - inline object (field hook of GetHooksListingResponse)
type GetHooksListingResponseHook struct {
	Event string `json:"event"`
	// Matcher with control characters revealed; '' when the entry has none.
	Matcher string `json:"matcher"`
	// Raw source name (userSettings, sessionHook, pluginHook, …).
	Source string `json:"source"`
	// User-facing source description.
	SourceLabel string `json:"sourceLabel"`
	PluginName  string `json:"pluginName,omitempty"`
	// Hook type (command, prompt, agent, http, mcp_tool, …).
	Type string `json:"type"`
	// List-row label: statusMessage when set, else the identity text; one line, control characters revealed.
	DisplayText string `json:"displayText"`
	// Identity text — the literal command/prompt/URL that runs, control characters revealed.
	CommandText string `json:"commandText"`
	// Label for commandText (Command, Prompt, URL, …).
	ContentLabel string `json:"contentLabel"`
	// If-condition, revealed, when set.
	Condition string `json:"condition,omitempty"`
	// Timeout in seconds.
	Timeout          *float64 `json:"timeout,omitempty"`
	StatusMessage    string   `json:"statusMessage,omitempty"`
	RunsOnce         *bool    `json:"runsOnce,omitempty"`
	RunsInBackground *bool    `json:"runsInBackground,omitempty"`
	// True when the session's mode or policy keeps this hook from running (the policy block and safeMode/bareMode say why); absent on rows that run. (TS literal `true`)
	Disabled bool `json:"disabled,omitempty"`
	// The entry as stored (raw matcher, '' when none, and raw hook object; no display escaping) for a host's edit form and as the target it names to `claude edit-hook`.
	Editable *GetHooksListingResponseHookEditable `json:"editable,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponseHook) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponseHook
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponseHook) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponseHook
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponseEventCatalog - inline object (field eventCatalog of GetHooksListingResponse)
type GetHooksListingResponseEventCatalog struct {
	Name            string `json:"name"`
	Summary         string `json:"summary"`
	SupportsMatcher bool   `json:"supportsMatcher"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponseEventCatalog) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponseEventCatalog
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponseEventCatalog) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponseEventCatalog
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponsePolicy - inline object (field policy of GetHooksListingResponse)
type GetHooksListingResponsePolicy struct {
	// Managed disableAllHooks — nothing runs at all.
	DisabledByPolicy bool `json:"disabledByPolicy"`
	// Managed allowManagedHooksOnly — non-managed hooks are blocked and managed hooks are intentionally not listed.
	ManagedOnly bool `json:"managedOnly"`
	// Managed strictPluginOnlyCustomization locks the hooks surface.
	PluginOnly bool `json:"pluginOnly"`
	// Effective disableAllHooks, whatever source set it.
	AllDisabled bool `json:"allDisabled"`
	// Hooks configured in managed settings (they run even under a non-managed disableAllHooks).
	PolicyHookCount int `json:"policyHookCount"`
	// Present when a managed settings source exists but could not be read: what the organization configured is unknown, so edit_hook refuses every edit (fail-closed) and a host locks its editing controls, as it does under... (TS literal `true`)
	PolicyUnreadable bool `json:"policyUnreadable,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponsePolicy) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponsePolicy
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponsePolicy) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponsePolicy
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponseSafeMode - inline object (field safeMode of GetHooksListingResponse)
type GetHooksListingResponseSafeMode struct {
	ManagedHooksStillApply bool `json:"managedHooksStillApply"`
	// How to leave safe mode, per its activation source.
	ExitHint string `json:"exitHint"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponseSafeMode) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponseSafeMode
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponseSafeMode) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponseSafeMode
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponseBareMode - inline object (field bareMode of GetHooksListingResponse)
type GetHooksListingResponseBareMode struct {
	// How to leave bare mode, per its activation source.
	ExitHint string `json:"exitHint"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponseBareMode) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponseBareMode
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponseBareMode) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponseBareMode
	return marshalKeep(plain(v), v.Raw)
}

// SettingsParseError - A settings file parse or validation error. (TS: SDKSettingsParseError)
type SettingsParseError struct {
	// Path to the settings file that failed to parse or validate.
	File string `json:"file,omitempty"`
	// Dot-notation path to the field with the error, or empty string for whole-file errors.
	Path string `json:"path"`
	// Human-readable error message.
	Message string `json:"message"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SettingsParseError) UnmarshalJSON(b []byte) error {
	type plain SettingsParseError
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SettingsParseError) MarshalJSON() ([]byte, error) {
	type plain SettingsParseError
	return marshalKeep(plain(v), v.Raw)
}

// GetHooksListingResponse - The hooks listing the CLI's /hooks menu renders, with display-ready strings. (TS: SDKControlGetHooksListingResponse, not exported (wire type))
type GetHooksListingResponse struct {
	// Events that have at least one listed hook, in the /hooks menu’s lifecycle order.
	Events []GetHooksListingResponseEvent `json:"events"`
	// One row per listed hook: events in lifecycle order, matchers in the menu’s priority order.
	Hooks []GetHooksListingResponseHook `json:"hooks"`
	// Every hook event in lifecycle order (name, the /hooks summary, whether its hooks take a matcher), for an add-hook form.
	EventCatalog []GetHooksListingResponseEventCatalog `json:"eventCatalog"`
	Policy       GetHooksListingResponsePolicy         `json:"policy"`
	// Present only when the session runs under --safe-mode.
	SafeMode *GetHooksListingResponseSafeMode `json:"safeMode,omitempty"`
	// Present only under --bare / CLAUDE_CODE_SIMPLE with the hooks surface gated off: settings-file, flag, policy, and plugin hooks never fire there; session hooks still run.
	BareMode *GetHooksListingResponseBareMode `json:"bareMode,omitempty"`
	// Settings files skipped by the merge — their hooks are neither listed nor running.
	Errors []SettingsParseError `json:"errors,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetHooksListingResponse) UnmarshalJSON(b []byte) error {
	type plain GetHooksListingResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetHooksListingResponse) MarshalJSON() ([]byte, error) {
	type plain GetHooksListingResponse
	return marshalKeep(plain(v), v.Raw)
}

// GetTaskOutputResponse - Success payload answering get_task_output. (TS: SDKControlGetTaskOutputResponse, not exported (wire type))
type GetTaskOutputResponse struct {
	// The end of the output, decoded as UTF-8; empty when the command has written nothing yet.
	Output string `json:"output"`
	// The size of the whole output in bytes.
	TotalBytes int `json:"total_bytes"`
	// Whether output holds only the end of a longer output (the last 8 KiB).
	Truncated bool `json:"truncated"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetTaskOutputResponse) UnmarshalJSON(b []byte) error {
	type plain GetTaskOutputResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetTaskOutputResponse) MarshalJSON() ([]byte, error) {
	type plain GetTaskOutputResponse
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseSession - inline object (field session of GetUsageResponse)
type GetUsageResponseSession struct {
	TotalCostUSD       float64               `json:"total_cost_usd"`
	TotalAPIDurationMs float64               `json:"total_api_duration_ms"`
	TotalDurationMs    float64               `json:"total_duration_ms"`
	TotalLinesAdded    int                   `json:"total_lines_added"`
	TotalLinesRemoved  int                   `json:"total_lines_removed"`
	ModelUsage         map[string]ModelUsage `json:"model_usage"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseSession) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseSession
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseSession) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseSession
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseRateLimitsFiveHour - inline object (field five_hour of GetUsageResponseRateLimits)
type GetUsageResponseRateLimitsFiveHour struct {
	// Percentage of the window used, 0-100. (nullable)
	Utilization *float64 `json:"utilization"`
	// ISO 8601 timestamp when the window resets. (nullable)
	ResetsAt *string `json:"resets_at"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseRateLimitsFiveHour) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseRateLimitsFiveHour
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseRateLimitsFiveHour) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseRateLimitsFiveHour
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseRateLimitsModelScoped - inline object (field model_scoped of GetUsageResponseRateLimits)
type GetUsageResponseRateLimitsModelScoped struct {
	// Server-supplied label for the model bucket (e.g.
	DisplayName string `json:"display_name"`
	// (nullable)
	Utilization *float64 `json:"utilization"`
	// (nullable)
	ResetsAt *string `json:"resets_at"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseRateLimitsModelScoped) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseRateLimitsModelScoped
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseRateLimitsModelScoped) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseRateLimitsModelScoped
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseRateLimits - inline object (field rate_limits of GetUsageResponse)
type GetUsageResponseRateLimits struct {
	// (optional, nullable)
	FiveHour *GetUsageResponseRateLimitsFiveHour `json:"five_hour,omitempty"`
	// (optional, nullable)
	SevenDay *GetUsageResponseRateLimitsFiveHour `json:"seven_day,omitempty"`
	// (optional, nullable)
	SevenDayOauthApps *GetUsageResponseRateLimitsFiveHour `json:"seven_day_oauth_apps,omitempty"`
	// (optional, nullable)
	SevenDayOpus *GetUsageResponseRateLimitsFiveHour `json:"seven_day_opus,omitempty"`
	// (optional, nullable)
	SevenDaySonnet *GetUsageResponseRateLimitsFiveHour `json:"seven_day_sonnet,omitempty"`
	// Per-model weekly windows from the server limits[] array, filtered by the overage-included-models allowlist.
	ModelScoped []GetUsageResponseRateLimitsModelScoped `json:"model_scoped,omitempty"`
	// (optional, nullable)
	ExtraUsage *UsageReportRateLimitsExtraUsage `json:"extra_usage,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseRateLimits) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseRateLimits
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseRateLimits) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseRateLimits
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseBehaviorsDayBehavior - inline object (field behavior of GetUsageResponseBehaviorsDay)
type GetUsageResponseBehaviorsDayBehavior struct {
	// (one of "cache_miss", "long_context", "subagent_heavy", "high_parallel", "cron", "max_effort", "workflow", "always_on_context", "large_tool_results")
	Key string `json:"key"`
	// Share of the weighted local usage attributed to this behavior, 0-100.
	Pct float64 `json:"pct"`
	// Requests in this window exhibiting the behavior.
	Count int `json:"count"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseBehaviorsDayBehavior) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseBehaviorsDayBehavior
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseBehaviorsDayBehavior) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseBehaviorsDayBehavior
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseBehaviorsDayRecommendation - inline object (field recommendation of GetUsageResponseBehaviorsDay)
type GetUsageResponseBehaviorsDayRecommendation struct {
	// The behavior key, or 'agent', 'skill', 'plugin' or 'mcp_server' for a named source.
	Key string `json:"key"`
	// Share of the weighted local usage this touches, 0-100.
	Pct float64 `json:"pct"`
	// What to do, verb first.
	Title string `json:"title"`
	// A few words: what the scan saw, or the command to run.
	Detail string `json:"detail,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseBehaviorsDayRecommendation) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseBehaviorsDayRecommendation
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseBehaviorsDayRecommendation) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseBehaviorsDayRecommendation
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseBehaviorsDayFix - inline object (field fix of GetUsageResponseBehaviorsDay)
type GetUsageResponseBehaviorsDayFix struct {
	// The recommendations it covers.
	Keys []string `json:"keys"`
	// User messages to send for the person, one each and in order, at priority 'later' so a running turn cannot reorder them: any /effort commands first, then one prompt for the rest, which goes with client_composed set.
	Messages []string `json:"messages"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseBehaviorsDayFix) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseBehaviorsDayFix
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseBehaviorsDayFix) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseBehaviorsDayFix
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseBehaviorsDayAgent - inline object (field agent of GetUsageResponseBehaviorsDay)
type GetUsageResponseBehaviorsDayAgent struct {
	Name string `json:"name"`
	// Share of the weighted local usage attributed to this item, 0-100.
	Pct float64 `json:"pct"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseBehaviorsDayAgent) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseBehaviorsDayAgent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseBehaviorsDayAgent) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseBehaviorsDayAgent
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseBehaviorsDay - inline object (field day of GetUsageResponseBehaviors)
type GetUsageResponseBehaviorsDay struct {
	// API requests found in local transcripts for this window.
	RequestCount int `json:"request_count"`
	// Distinct sessions observed in this window.
	SessionCount int `json:"session_count"`
	// Behavioral characteristics of local usage.
	Behaviors []GetUsageResponseBehaviorsDayBehavior `json:"behaviors"`
	// One recommended change per notable behavior or source, largest share first, worded for display.
	Recommendations []GetUsageResponseBehaviorsDayRecommendation `json:"recommendations,omitempty"`
	// What one "fix" action does.
	Fix        *GetUsageResponseBehaviorsDayFix    `json:"fix,omitempty"`
	Agents     []GetUsageResponseBehaviorsDayAgent `json:"agents"`
	Skills     []GetUsageResponseBehaviorsDayAgent `json:"skills"`
	Plugins    []GetUsageResponseBehaviorsDayAgent `json:"plugins"`
	MCPServers []GetUsageResponseBehaviorsDayAgent `json:"mcp_servers"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseBehaviorsDay) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseBehaviorsDay
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseBehaviorsDay) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseBehaviorsDay
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponseBehaviors - inline object (field behaviors of GetUsageResponse)
type GetUsageResponseBehaviors struct {
	// Last 24 hours.
	Day GetUsageResponseBehaviorsDay `json:"day"`
	// Last 7 days.
	Week GetUsageResponseBehaviorsDay `json:"week"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponseBehaviors) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponseBehaviors
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponseBehaviors) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponseBehaviors
	return marshalKeep(plain(v), v.Raw)
}

// GetUsageResponse - Structured /usage data: session cost/usage totals plus claude.ai plan rate-limit utilization. (TS: SDKControlGetUsageResponse)
type GetUsageResponse struct {
	// Cost and usage accumulated by the current session.
	Session GetUsageResponseSession `json:"session"`
	// Claude.ai subscription type ('pro', 'max', 'team', 'enterprise') or null for API key / 3P provider sessions. (nullable)
	SubscriptionType *string `json:"subscription_type"`
	// False when plan rate limits do not apply (API key, Bedrock, Vertex, or missing profile scope) — rate_limits will be null.
	RateLimitsAvailable bool `json:"rate_limits_available"`
	// Plan rate-limit utilization windows from the claude.ai usage endpoint, or null when unavailable. (nullable)
	RateLimits *GetUsageResponseRateLimits `json:"rate_limits"`
	// What's contributing to limits usage, from a scan of local transcripts on this machine (the same data the /usage dialog renders): behavioral characteristics plus per-skill/agent/plugin/MCP-server attribution. (nullable)
	Behaviors *GetUsageResponseBehaviors `json:"behaviors"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *GetUsageResponse) UnmarshalJSON(b []byte) error {
	type plain GetUsageResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v GetUsageResponse) MarshalJSON() ([]byte, error) {
	type plain GetUsageResponse
	return marshalKeep(plain(v), v.Raw)
}

// InitializeResponse - Response from session initialization with available commands, models, and account info. (TS: SDKControlInitializeResponse)
type InitializeResponse struct {
	Commands              []SlashCommand `json:"commands"`
	Agents                []AgentInfo    `json:"agents"`
	OutputStyle           string         `json:"output_style"`
	AvailableOutputStyles []string       `json:"available_output_styles"`
	Models                []ModelInfo    `json:"models"`
	// Information about the logged in user's account.
	Account AccountInfo `json:"account"`
	// Whether the `hooks` this initialize carried were registered: true on a session's first initialize, and on a repeated initialize from the process that owns the CLI's stdin (its set replaces the one registered...
	HooksApplied *bool `json:"hooks_applied,omitempty"`
	// Whether every plugin this initialize listed is loaded: true when each one is among the plugins the process loaded at launch (from `plugins` under --await-initialize, or from --plugin-dir), so a re-sent initialize...
	PluginsApplied *bool `json:"plugins_applied,omitempty"`
	// What became of each entry of this initialize's sdkMcpServerManifests, keyed by sdk server name, for the entries whose name is in sdkMcpServers.
	SdkMCPManifestsParked map[string]string `json:"sdk_mcp_manifests_parked,omitempty"`
	// Claude Code version of the process that runs this session's turns (the same value its system/init carries).
	ClaudeCodeVersion      string                 `json:"claude_code_version,omitempty"`
	FastModeState          FastModeState          `json:"fast_mode_state,omitempty"`
	FastModeDisabledReason FastModeDisabledReason `json:"fast_mode_disabled_reason,omitempty"`
	// since 0.3.317 (not in 0.3.311)
	// True on every initialize reply from a CLI that honors `onFailure` on hook matchers.
	HookOnFailureSupported *bool `json:"hook_on_failure_supported,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InitializeResponse) UnmarshalJSON(b []byte) error {
	type plain InitializeResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InitializeResponse) MarshalJSON() ([]byte, error) {
	type plain InitializeResponse
	return marshalKeep(plain(v), v.Raw)
}

// InterruptResponse - Result of an interrupt operation. (TS: SDKControlInterruptResponse)
type InterruptResponse struct {
	// Uuids of async user messages that survive this interrupt: commands still in the queue, plus any batch already dequeued for the imminent turn but not yet reachable by the abort.
	StillQueued []string `json:"still_queued"`
	// Present only when the request set cancel_queued:true — uuids of main-thread commands cancelled by this interrupt: every survivor that would otherwise have appeared under `still_queued`, including any uuid that was...
	Cancelled []string `json:"cancelled,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InterruptResponse) UnmarshalJSON(b []byte) error {
	type plain InterruptResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InterruptResponse) MarshalJSON() ([]byte, error) {
	type plain InterruptResponse
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRuleDescription - The CLI's plain-language reading of a rule (e.g. (TS: SDKPermissionRuleDescription)
type PermissionRuleDescription struct {
	Prefix   string `json:"prefix"`
	Emphasis string `json:"emphasis,omitempty"`
	Suffix   string `json:"suffix,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRuleDescription) UnmarshalJSON(b []byte) error {
	type plain PermissionRuleDescription
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRuleDescription) MarshalJSON() ([]byte, error) {
	type plain PermissionRuleDescription
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRuleEntry - One permission rule with its provenance and where it lives. (TS: SDKPermissionRuleEntry)
type PermissionRuleEntry struct {
	// (one of "allow", "deny", "ask")
	Behavior string `json:"behavior"`
	// Where the rule comes from. (one of "userSettings", "projectSettings", "localSettings", "flagSettings", "policySettings", "cliArg", "command", "session", "mcpMount", "toolsNarrowing", "mcpServerPolicy", "hostCredential")
	Source string `json:"source"`
	// The stored rule string VERBATIM, exactly as the session holds it.
	Rule string `json:"rule"`
	// Plain-language reading of the rule; absent where the terminal shows no subtitle either.
	Description *PermissionRuleDescription `json:"description,omitempty"`
	// Where the rule lives: 'persistent' (userSettings/projectSettings/localSettings — saved in a settings file), 'session' (cliArg/session — in memory only, for the rest of this session), or 'readonly'... (one of "persistent", "session", "readonly")
	Editability string `json:"editability"`
	// Present (true) when enterprise managed settings pin allowManagedPermissionRulesOnly and this rule, read from a non-policy settings file, is ignored by the session.
	NotInEffect *bool `json:"notInEffect,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRuleEntry) UnmarshalJSON(b []byte) error {
	type plain PermissionRuleEntry
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRuleEntry) MarshalJSON() ([]byte, error) {
	type plain PermissionRuleEntry
	return marshalKeep(plain(v), v.Raw)
}

// PermissionWorkspaceDirectory - One additional working directory in the permission scope. (TS: SDKPermissionWorkspaceDirectory)
type PermissionWorkspaceDirectory struct {
	Path string `json:"path"`
	// Where the directory grant came from: a settings source (e.g.
	Source string `json:"source"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionWorkspaceDirectory) UnmarshalJSON(b []byte) error {
	type plain PermissionWorkspaceDirectory
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionWorkspaceDirectory) MarshalJSON() ([]byte, error) {
	type plain PermissionWorkspaceDirectory
	return marshalKeep(plain(v), v.Raw)
}

// PermissionRulesState - The session's live permission rules state, as list_permission_rules reports it. (TS: SDKControlPermissionRulesState)
type PermissionRulesState struct {
	Rules                []PermissionRuleEntry          `json:"rules"`
	WorkspaceDirectories []PermissionWorkspaceDirectory `json:"workspaceDirectories"`
	// The session's original working directory.
	OriginalCwd string `json:"originalCwd"`
	// True when enterprise managed settings pin allowManagedPermissionRulesOnly: the session applies policy rules only, and rules from other settings files appear with notInEffect set.
	ManagedOnly bool `json:"managedOnly"`
	// Settings parse and validation errors, as get_settings reports them.
	Errors []SettingsParseError `json:"errors,omitempty"`
	// since 0.3.312-0.3.325 (not in 0.3.311)
	// True when this answer was cut because the connection stores its answers (a cloud or self-hosted session): `rules` holds only rows from settings sources and `mcpServerPolicy`, `workspaceDirectories` only folders from...
	Withheld *bool `json:"withheld,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionRulesState) UnmarshalJSON(b []byte) error {
	type plain PermissionRulesState
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionRulesState) MarshalJSON() ([]byte, error) {
	type plain PermissionRulesState
	return marshalKeep(plain(v), v.Raw)
}

// ListPermissionRulesResponse - Success payload of list_permission_rules. (TS: SDKControlListPermissionRulesResponse)
type ListPermissionRulesResponse struct {
	// The session's live permission rules state, as list_permission_rules reports it.
	State PermissionRulesState `json:"state"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ListPermissionRulesResponse) UnmarshalJSON(b []byte) error {
	type plain ListPermissionRulesResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ListPermissionRulesResponse) MarshalJSON() ([]byte, error) {
	type plain ListPermissionRulesResponse
	return marshalKeep(plain(v), v.Raw)
}

// McpReadResourceResponseContent - inline object (field content of McpReadResourceResponse)
type McpReadResourceResponseContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	// Base64, for a binary item.
	Blob string `json:"blob,omitempty"`
	// The content item's own `_meta` as the server sent it (MCP Apps puts the resource's `ui.csp` and `ui.permissions` here), minus keys under the CLI-reserved `com.anthropic/` prefix.
	XMeta map[string]any `json:"_meta,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpReadResourceResponseContent) UnmarshalJSON(b []byte) error {
	type plain McpReadResourceResponseContent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpReadResourceResponseContent) MarshalJSON() ([]byte, error) {
	type plain McpReadResourceResponseContent
	return marshalKeep(plain(v), v.Raw)
}

// McpReadResourceResponse - The server's resources/read result for an mcp_read_resource request. (TS: SDKControlMcpReadResourceResponse)
type McpReadResourceResponse struct {
	Contents []McpReadResourceResponseContent `json:"contents"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpReadResourceResponse) UnmarshalJSON(b []byte) error {
	type plain McpReadResourceResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpReadResourceResponse) MarshalJSON() ([]byte, error) {
	type plain McpReadResourceResponse
	return marshalKeep(plain(v), v.Raw)
}

// ReadFileResponse - File contents for the remote sidebar viewer. (TS: SDKControlReadFileResponse)
type ReadFileResponse struct {
	Contents  string `json:"contents"`
	AbsPath   string `json:"absPath"`
	Truncated *bool  `json:"truncated,omitempty"`
	// Set when the request asked for base64. (always "base64")
	Encoding string `json:"encoding,omitempty"`
	// since 0.3.317 (not in 0.3.311)
	// Present on every utf-8 answer.
	Lossy *bool `json:"lossy,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReadFileResponse) UnmarshalJSON(b []byte) error {
	type plain ReadFileResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReadFileResponse) MarshalJSON() ([]byte, error) {
	type plain ReadFileResponse
	return marshalKeep(plain(v), v.Raw)
}

// ReloadOutputStylesResponse - Refreshed output style names after reload, built-in and custom, in the order the initialize response lists them. (TS: SDKControlReloadOutputStylesResponse)
type ReloadOutputStylesResponse struct {
	AvailableOutputStyles []string `json:"available_output_styles"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadOutputStylesResponse) UnmarshalJSON(b []byte) error {
	type plain ReloadOutputStylesResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadOutputStylesResponse) MarshalJSON() ([]byte, error) {
	type plain ReloadOutputStylesResponse
	return marshalKeep(plain(v), v.Raw)
}

// ReloadPluginsResponsePlugin - inline object (field plugin of ReloadPluginsResponse)
type ReloadPluginsResponsePlugin struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	// The plugin's version as declared in its plugin.json manifest, emitted verbatim (plugin-author-controlled — validate before trusting).
	Version string `json:"version,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadPluginsResponsePlugin) UnmarshalJSON(b []byte) error {
	type plain ReloadPluginsResponsePlugin
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadPluginsResponsePlugin) MarshalJSON() ([]byte, error) {
	type plain ReloadPluginsResponsePlugin
	return marshalKeep(plain(v), v.Raw)
}

// ReloadPluginsResponseCacheImpact - inline object (field cache_impact of ReloadPluginsResponse)
type ReloadPluginsResponseCacheImpact struct {
	MCPServersAdded   []string `json:"mcp_servers_added"`
	MCPServersRemoved []string `json:"mcp_servers_removed"`
	// (one of "adds", "may-add", "removes", "may-remove"; nullable)
	LSPToolChange *string `json:"lsp_tool_change"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadPluginsResponseCacheImpact) UnmarshalJSON(b []byte) error {
	type plain ReloadPluginsResponseCacheImpact
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadPluginsResponseCacheImpact) MarshalJSON() ([]byte, error) {
	type plain ReloadPluginsResponseCacheImpact
	return marshalKeep(plain(v), v.Raw)
}

// ReloadPluginsResponse - Refreshed commands, agents, plugins, and MCP server status after reload. (TS: SDKControlReloadPluginsResponse)
type ReloadPluginsResponse struct {
	Commands   []SlashCommand                `json:"commands"`
	Agents     []AgentInfo                   `json:"agents"`
	Plugins    []ReloadPluginsResponsePlugin `json:"plugins"`
	MCPServers []McpServerStatus             `json:"mcpServers"`
	ErrorCount int                           `json:"error_count"`
	// Present only when the request asked to hold on cache impact and this CLI ran the check.
	Held *bool `json:"held,omitempty"`
	// What applying the held reload would change in the session's tool list: plugin MCP servers it would register or drop (scoped plugin:<plugin>:<server> names, plugin-authored — validate before showing) and whether it...
	CacheImpact *ReloadPluginsResponseCacheImpact `json:"cache_impact,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadPluginsResponse) UnmarshalJSON(b []byte) error {
	type plain ReloadPluginsResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadPluginsResponse) MarshalJSON() ([]byte, error) {
	type plain ReloadPluginsResponse
	return marshalKeep(plain(v), v.Raw)
}

// ReloadSkillsResponse - Refreshed skill commands after reload. (TS: SDKControlReloadSkillsResponse)
type ReloadSkillsResponse struct {
	Skills []SlashCommand `json:"skills"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ReloadSkillsResponse) UnmarshalJSON(b []byte) error {
	type plain ReloadSkillsResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ReloadSkillsResponse) MarshalJSON() ([]byte, error) {
	type plain ReloadSkillsResponse
	return marshalKeep(plain(v), v.Raw)
}

// RequestProgressMessage - Progress for a long-running client-originated control_request (currently only side_question), correlated by request_id. status 'started' means the worker accepted the request and launched the work; 'api_retry'... (TS: SDKControlRequestProgressMessage)
type RequestProgressMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "control_request_progress")
	Subtype string `json:"subtype"`
	// request_id of the in-flight control_request this progress belongs to.
	RequestID string `json:"request_id"`
	// (one of "started", "api_retry")
	Status       string   `json:"status"`
	Attempt      *int     `json:"attempt,omitempty"`
	MaxRetries   *int     `json:"max_retries,omitempty"`
	RetryDelayMs *float64 `json:"retry_delay_ms,omitempty"`
	// (optional, nullable)
	ErrorStatus *int   `json:"error_status,omitempty"`
	UUID        string `json:"uuid"`
	SessionID   string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RequestProgressMessage) UnmarshalJSON(b []byte) error {
	type plain RequestProgressMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RequestProgressMessage) MarshalJSON() ([]byte, error) {
	type plain RequestProgressMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "control_request_progress"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ControlResponse - Envelope for the single reply to a control_request, sent by whichever side received the request. (TS: SDKControlResponse)
type ControlResponse struct {
	// (always "control_response")
	Type     string                  `json:"type"`
	Response ControlResponseResponse `json:"response"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ControlResponse) UnmarshalJSON(b []byte) error {
	type plain ControlResponse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ControlResponse) MarshalJSON() ([]byte, error) {
	type plain ControlResponse
	if v.Type == "" {
		v.Type = "control_response"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ConversationResetMessage - Emitted by /clear, plan-mode exit, fresh-session, and onboarding flows. (TS: SDKConversationResetMessage)
type ConversationResetMessage struct {
	// (always "conversation_reset")
	Type              string `json:"type"`
	NewConversationID string `json:"new_conversation_id"`
	UUID              string `json:"uuid"`
	SessionID         string `json:"session_id"`
	// What discarded the conversation: 'clear' is the /clear command (or its /reset and /new aliases), 'plan_mode_exit' is leaving plan mode with the clear-context option, 'fresh_session' is a flow that starts a fresh... (one of "clear", "plan_mode_exit", "fresh_session", "onboarding")
	Trigger string `json:"trigger,omitempty"`
	// Only with trigger 'clear': the uuid of the user message whose /clear was executed (the client's own uuid for that message when it came from a Remote Control or stream-json client, otherwise the uuid the CLI assigned...
	UserMessageUUID string `json:"user_message_uuid,omitempty"`
	// When the reset happened, as an ISO 8601 string in UTC read from the clock of the process that performed it.
	Timestamp string `json:"timestamp,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ConversationResetMessage) UnmarshalJSON(b []byte) error {
	type plain ConversationResetMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ConversationResetMessage) MarshalJSON() ([]byte, error) {
	type plain ConversationResetMessage
	if v.Type == "" {
		v.Type = "conversation_reset"
	}
	return marshalKeep(plain(v), v.Raw)
}

// DeferredToolUse (TS: SDKDeferredToolUse)
type DeferredToolUse struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *DeferredToolUse) UnmarshalJSON(b []byte) error {
	type plain DeferredToolUse
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v DeferredToolUse) MarshalJSON() ([]byte, error) {
	type plain DeferredToolUse
	return marshalKeep(plain(v), v.Raw)
}

// ElicitationCompleteMessage - Emitted when an MCP server confirms that a URL-mode elicitation is complete. (TS: SDKElicitationCompleteMessage)
type ElicitationCompleteMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "elicitation_complete")
	Subtype       string `json:"subtype"`
	MCPServerName string `json:"mcp_server_name"`
	ElicitationID string `json:"elicitation_id"`
	UUID          string `json:"uuid"`
	SessionID     string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ElicitationCompleteMessage) UnmarshalJSON(b []byte) error {
	type plain ElicitationCompleteMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ElicitationCompleteMessage) MarshalJSON() ([]byte, error) {
	type plain ElicitationCompleteMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "elicitation_complete"
	}
	return marshalKeep(plain(v), v.Raw)
}

// FilesPersistedEventFile - inline object (field file of FilesPersistedEvent)
type FilesPersistedEventFile struct {
	Filename string `json:"filename"`
	FileID   string `json:"file_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *FilesPersistedEventFile) UnmarshalJSON(b []byte) error {
	type plain FilesPersistedEventFile
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v FilesPersistedEventFile) MarshalJSON() ([]byte, error) {
	type plain FilesPersistedEventFile
	return marshalKeep(plain(v), v.Raw)
}

// FilesPersistedEventFailed - inline object (field failed of FilesPersistedEvent)
type FilesPersistedEventFailed struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *FilesPersistedEventFailed) UnmarshalJSON(b []byte) error {
	type plain FilesPersistedEventFailed
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v FilesPersistedEventFailed) MarshalJSON() ([]byte, error) {
	type plain FilesPersistedEventFailed
	return marshalKeep(plain(v), v.Raw)
}

// FilesPersistedEvent (TS: SDKFilesPersistedEvent)
type FilesPersistedEvent struct {
	// (always "system")
	Type string `json:"type"`
	// (always "files_persisted")
	Subtype     string                      `json:"subtype"`
	Files       []FilesPersistedEventFile   `json:"files"`
	Failed      []FilesPersistedEventFailed `json:"failed"`
	ProcessedAt string                      `json:"processed_at"`
	UUID        string                      `json:"uuid"`
	SessionID   string                      `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *FilesPersistedEvent) UnmarshalJSON(b []byte) error {
	type plain FilesPersistedEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v FilesPersistedEvent) MarshalJSON() ([]byte, error) {
	type plain FilesPersistedEvent
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "files_persisted"
	}
	return marshalKeep(plain(v), v.Raw)
}

// HookProgressMessage (TS: SDKHookProgressMessage)
type HookProgressMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "hook_progress")
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Output    string `json:"output"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *HookProgressMessage) UnmarshalJSON(b []byte) error {
	type plain HookProgressMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v HookProgressMessage) MarshalJSON() ([]byte, error) {
	type plain HookProgressMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "hook_progress"
	}
	return marshalKeep(plain(v), v.Raw)
}

// HookResponseMessage (TS: SDKHookResponseMessage)
type HookResponseMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "hook_response")
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Output    string `json:"output"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	// (one of "success", "error", "cancelled")
	Outcome   string `json:"outcome"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *HookResponseMessage) UnmarshalJSON(b []byte) error {
	type plain HookResponseMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v HookResponseMessage) MarshalJSON() ([]byte, error) {
	type plain HookResponseMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "hook_response"
	}
	return marshalKeep(plain(v), v.Raw)
}

// HookStartedMessage (TS: SDKHookStartedMessage)
type HookStartedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "hook_started")
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *HookStartedMessage) UnmarshalJSON(b []byte) error {
	type plain HookStartedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v HookStartedMessage) MarshalJSON() ([]byte, error) {
	type plain HookStartedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "hook_started"
	}
	return marshalKeep(plain(v), v.Raw)
}

// InformationalMessage - Generic text banner emitted by the loop — non-error status lines, hook feedback (e.g. a UserPromptSubmit hook's block reason), slash-command output. (TS: SDKInformationalMessage)
type InformationalMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "informational")
	Subtype string `json:"subtype"`
	Content string `json:"content"`
	// Render level. (one of "info", "notice", "suggestion", "warning")
	Level string `json:"level"`
	// Dedupes progress messages for the same tool use.
	ToolUseID string `json:"tool_use_id,omitempty"`
	// When true, execution stops after this message (e.g. a Stop hook denied continuation).
	PreventContinuation *bool `json:"prevent_continuation,omitempty"`
	// Opaque feature tag on a line a host may treat specially; absent on ordinary lines.
	Tag       string `json:"tag,omitempty"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *InformationalMessage) UnmarshalJSON(b []byte) error {
	type plain InformationalMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v InformationalMessage) MarshalJSON() ([]byte, error) {
	type plain InformationalMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "informational"
	}
	return marshalKeep(plain(v), v.Raw)
}

// KeepAliveMessage - Liveness heartbeat. (TS: SDKKeepAliveMessage, not exported (wire type))
type KeepAliveMessage struct {
	// (always "keep_alive")
	Type string `json:"type"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *KeepAliveMessage) UnmarshalJSON(b []byte) error {
	type plain KeepAliveMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v KeepAliveMessage) MarshalJSON() ([]byte, error) {
	type plain KeepAliveMessage
	if v.Type == "" {
		v.Type = "keep_alive"
	}
	return marshalKeep(plain(v), v.Raw)
}

// LocalCommandOutputMessage - Output from a local slash command (e.g. /voice, /usage). (TS: SDKLocalCommandOutputMessage)
type LocalCommandOutputMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "local_command_output")
	Subtype   string `json:"subtype"`
	Content   string `json:"content"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *LocalCommandOutputMessage) UnmarshalJSON(b []byte) error {
	type plain LocalCommandOutputMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v LocalCommandOutputMessage) MarshalJSON() ([]byte, error) {
	type plain LocalCommandOutputMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "local_command_output"
	}
	return marshalKeep(plain(v), v.Raw)
}

// McpResourceLink (TS: SDKMcpResourceLink)
type McpResourceLink struct {
	URI         string         `json:"uri"`
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	MimeType    string         `json:"mimeType,omitempty"`
	Size        *int           `json:"size,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *McpResourceLink) UnmarshalJSON(b []byte) error {
	type plain McpResourceLink
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v McpResourceLink) MarshalJSON() ([]byte, error) {
	type plain McpResourceLink
	return marshalKeep(plain(v), v.Raw)
}

// MemoryRecallMessageMemory - inline object (field memory of MemoryRecallMessage)
type MemoryRecallMessageMemory struct {
	// Absolute path to the memory file, a synthesis sentinel of the form `<synthesis:DIR>` when mode is 'synthesize', or an https URL when scope is 'organization'.
	Path string `json:"path"`
	// (one of "personal", "team", "organization")
	Scope string `json:"scope"`
	// The surfaced memory body.
	Content string `json:"content,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MemoryRecallMessageMemory) UnmarshalJSON(b []byte) error {
	type plain MemoryRecallMessageMemory
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MemoryRecallMessageMemory) MarshalJSON() ([]byte, error) {
	type plain MemoryRecallMessageMemory
	return marshalKeep(plain(v), v.Raw)
}

// MemoryRecallMessage - Emitted when the memory recall supervisor surfaces relevant memories into the turn. (TS: SDKMemoryRecallMessage)
type MemoryRecallMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "memory_recall")
	Subtype string `json:"subtype"`
	// How memories were surfaced: 'select' returns full file bodies chosen by the parallel selector; 'synthesize' returns a Sonnet-authored paragraph distilled from many tiny memories. (one of "select", "synthesize")
	Mode      string                      `json:"mode"`
	Memories  []MemoryRecallMessageMemory `json:"memories"`
	UUID      string                      `json:"uuid"`
	SessionID string                      `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MemoryRecallMessage) UnmarshalJSON(b []byte) error {
	type plain MemoryRecallMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MemoryRecallMessage) MarshalJSON() ([]byte, error) {
	type plain MemoryRecallMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "memory_recall"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginHuman - variant of MessageOrigin
type MessageOriginHuman struct {
	// (always "human")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginHuman) UnmarshalJSON(b []byte) error {
	type plain MessageOriginHuman
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginHuman) MarshalJSON() ([]byte, error) {
	type plain MessageOriginHuman
	if v.Kind == "" {
		v.Kind = "human"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginChannel - variant of MessageOrigin
type MessageOriginChannel struct {
	// (always "channel")
	Kind   string `json:"kind"`
	Server string `json:"server"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginChannel) UnmarshalJSON(b []byte) error {
	type plain MessageOriginChannel
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginChannel) MarshalJSON() ([]byte, error) {
	type plain MessageOriginChannel
	if v.Kind == "" {
		v.Kind = "channel"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginPeer - variant of MessageOrigin
type MessageOriginPeer struct {
	// (always "peer")
	Kind string `json:"kind"`
	From string `json:"from"`
	// The SENDING session's permission class as declared by the host that injects this message on local stdin ('bypass' for sessions that run tools without asking, 'prompting' otherwise). (one of "bypass", "prompting")
	FromMode string `json:"fromMode,omitempty"`
	// Sender display name, normalized by the harness: Unicode control, format, surrogate, and line/paragraph-separator code points stripped (categories Cc/Cf/Cs/Zl/Zp — covers bidi controls, zero-width characters, and tag...
	Name string `json:"name,omitempty"`
	// The sender's host-openable session id (the envelope's `from-session` attribute — e.g. a desktop `local_<uuid>` or a cloud session `session_`/`ses_` id), set by the sender's host so a receiving UI can link this...
	FromSession string `json:"fromSession,omitempty"`
	// Task id of the in-process background subagent that sent this message, stamped by the harness from the sending loop (never from tool input).
	SenderTaskID string `json:"senderTaskId,omitempty"`
	// Decoded message body with the peer envelope stripped — byte-exact with what the model sees.
	Body string `json:"body,omitempty"`
	// Kernel-verified pid of the process that connected to this session's cross-session messaging socket, read from the connection (SO_PEERCRED / LOCAL_PEERPID) — never from the payload.
	VerifiedPeerPid *float64 `json:"verifiedPeerPid,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginPeer) UnmarshalJSON(b []byte) error {
	type plain MessageOriginPeer
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginPeer) MarshalJSON() ([]byte, error) {
	type plain MessageOriginPeer
	if v.Kind == "" {
		v.Kind = "peer"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginTaskNotification - variant of MessageOrigin
type MessageOriginTaskNotification struct {
	// (always "task-notification")
	Kind string `json:"kind"`
	// Present when the delivery is the fired stored prompt of a scheduled task/routine ('scheduled-trigger', stamped from server-asserted provenance, or declared by a local host for its own scheduled runs; the schedule... (one of "scheduled-trigger", "peer-send-message", "projects-relay", "session-inbox")
	Subkind string `json:"subkind,omitempty"`
	// On a 'scheduled-trigger' delivery, why it fired: a short lowercase token such as 'scheduled', 'manual', 'retry', 'catch_up' or 'api'.
	FireReason string `json:"fireReason,omitempty"`
	// On a background task's own notification: the run_id its task_* events carried, as of when the notification was queued.
	RunID string `json:"runId,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginTaskNotification) UnmarshalJSON(b []byte) error {
	type plain MessageOriginTaskNotification
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginTaskNotification) MarshalJSON() ([]byte, error) {
	type plain MessageOriginTaskNotification
	if v.Kind == "" {
		v.Kind = "task-notification"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginCoordinator - variant of MessageOrigin
type MessageOriginCoordinator struct {
	// (always "coordinator")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginCoordinator) UnmarshalJSON(b []byte) error {
	type plain MessageOriginCoordinator
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginCoordinator) MarshalJSON() ([]byte, error) {
	type plain MessageOriginCoordinator
	if v.Kind == "" {
		v.Kind = "coordinator"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginUnclassified - variant of MessageOrigin
type MessageOriginUnclassified struct {
	// (always "unclassified")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginUnclassified) UnmarshalJSON(b []byte) error {
	type plain MessageOriginUnclassified
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginUnclassified) MarshalJSON() ([]byte, error) {
	type plain MessageOriginUnclassified
	if v.Kind == "" {
		v.Kind = "unclassified"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginObserver - variant of MessageOrigin
type MessageOriginObserver struct {
	// (always "observer")
	Kind         string `json:"kind"`
	From         string `json:"from"`
	SenderTaskID string `json:"senderTaskId"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginObserver) UnmarshalJSON(b []byte) error {
	type plain MessageOriginObserver
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginObserver) MarshalJSON() ([]byte, error) {
	type plain MessageOriginObserver
	if v.Kind == "" {
		v.Kind = "observer"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginAutoContinuation - variant of MessageOrigin
type MessageOriginAutoContinuation struct {
	// (always "auto-continuation")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginAutoContinuation) UnmarshalJSON(b []byte) error {
	type plain MessageOriginAutoContinuation
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginAutoContinuation) MarshalJSON() ([]byte, error) {
	type plain MessageOriginAutoContinuation
	if v.Kind == "" {
		v.Kind = "auto-continuation"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MessageOriginObserverActivity - variant of MessageOrigin
type MessageOriginObserverActivity struct {
	// (always "observer-activity")
	Kind string `json:"kind"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MessageOriginObserverActivity) UnmarshalJSON(b []byte) error {
	type plain MessageOriginObserverActivity
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MessageOriginObserverActivity) MarshalJSON() ([]byte, error) {
	type plain MessageOriginObserverActivity
	if v.Kind == "" {
		v.Kind = "observer-activity"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserMessage - A user-role message. (TS: SDKUserMessage)
type UserMessage struct {
	// (always "user")
	Type string `json:"type"`
	// An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image, document, tool_result, ...).
	Message MessageParam `json:"message"`
	// (nullable)
	ParentToolUseID *string `json:"parent_tool_use_id"`
	IsSynthetic     *bool   `json:"isSynthetic,omitempty"`
	// Structured tool output — the tool's full Output object, not the string content sent to the model.
	ToolUseResult json.RawMessage `json:"tool_use_result,omitempty"`
	// (one of "now", "next", "later")
	Priority string         `json:"priority,omitempty"`
	Origin   *MessageOrigin `json:"origin,omitempty"`
	// When false, the message is appended to the transcript without triggering an assistant turn.
	ShouldQuery *bool `json:"shouldQuery,omitempty"`
	// ISO timestamp when the message was created on the originating process.
	Timestamp string `json:"timestamp,omitempty"`
	// The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and no slash-command dispatch. (TS literal `true`)
	ClientComposed bool   `json:"client_composed,omitempty"`
	UUID           string `json:"uuid,omitempty"`
	// Content the user pasted into the prompt rather than typed: each entry a string or an array of content blocks.
	PastedContent []json.RawMessage `json:"pasted_content,omitempty"`
	// Text the user pasted that is still in `message.content` where they put it: each entry one paste.
	InlinePastes []string `json:"inline_pastes,omitempty"`
	SessionID    string   `json:"session_id,omitempty"`
	// ID of the subagent that produced this message: the `task_id` of its task events, unchanged when the subagent is resumed.
	AgentID string `json:"agent_id,omitempty"`
	// Subagent type that produced this message.
	SubagentType string `json:"subagent_type,omitempty"`
	// Description of the subagent task that produced this message.
	TaskDescription string `json:"task_description,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserMessage) UnmarshalJSON(b []byte) error {
	type plain UserMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserMessage) MarshalJSON() ([]byte, error) {
	type plain UserMessage
	if v.Type == "" {
		v.Type = "user"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserMessageReplay (TS: SDKUserMessageReplay)
type UserMessageReplay struct {
	// (always "user")
	Type string `json:"type"`
	// An Anthropic Messages API user message: a MessageParam with role "user" whose content is a string or an array of content blocks (text, image, document, tool_result, ...).
	Message MessageParam `json:"message"`
	// (nullable)
	ParentToolUseID *string `json:"parent_tool_use_id"`
	IsSynthetic     *bool   `json:"isSynthetic,omitempty"`
	// Structured tool output — the tool's full Output object, not the string content sent to the model.
	ToolUseResult json.RawMessage `json:"tool_use_result,omitempty"`
	// (one of "now", "next", "later")
	Priority string         `json:"priority,omitempty"`
	Origin   *MessageOrigin `json:"origin,omitempty"`
	// When false, the message is appended to the transcript without triggering an assistant turn.
	ShouldQuery *bool `json:"shouldQuery,omitempty"`
	// ISO timestamp when the message was created on the originating process.
	Timestamp string `json:"timestamp,omitempty"`
	// The client composed this turn from content the user did not type; the CLI delivers its text as written, with no `@path` file-mention expansion and no slash-command dispatch. (TS literal `true`)
	ClientComposed bool   `json:"client_composed,omitempty"`
	UUID           string `json:"uuid"`
	SessionID      string `json:"session_id"`
	// (TS literal `true`)
	IsReplay        bool              `json:"isReplay"`
	FileAttachments []json.RawMessage `json:"file_attachments,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserMessageReplay) UnmarshalJSON(b []byte) error {
	type plain UserMessageReplay
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserMessageReplay) MarshalJSON() ([]byte, error) {
	type plain UserMessageReplay
	if v.Type == "" {
		v.Type = "user"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionDenial (TS: SDKPermissionDenial)
type PermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolUseID string         `json:"tool_use_id"`
	ToolInput map[string]any `json:"tool_input"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionDenial) UnmarshalJSON(b []byte) error {
	type plain PermissionDenial
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionDenial) MarshalJSON() ([]byte, error) {
	type plain PermissionDenial
	return marshalKeep(plain(v), v.Raw)
}

// TerminalReason is a string enum. Why the query loop terminated. (TS: TerminalReason)
type TerminalReason string

const (
	TerminalReasonBlockingLimit                  TerminalReason = "blocking_limit"
	TerminalReasonRapidRefillBreaker             TerminalReason = "rapid_refill_breaker"
	TerminalReasonPromptTooLong                  TerminalReason = "prompt_too_long"
	TerminalReasonImageError                     TerminalReason = "image_error"
	TerminalReasonModelError                     TerminalReason = "model_error"
	TerminalReasonApiError                       TerminalReason = "api_error"
	TerminalReasonMalformedToolUseExhausted      TerminalReason = "malformed_tool_use_exhausted"
	TerminalReasonAbortedStreaming               TerminalReason = "aborted_streaming"
	TerminalReasonAbortedTools                   TerminalReason = "aborted_tools"
	TerminalReasonStopHookPrevented              TerminalReason = "stop_hook_prevented"
	TerminalReasonHookStopped                    TerminalReason = "hook_stopped"
	TerminalReasonToolDeferred                   TerminalReason = "tool_deferred"
	TerminalReasonMaxTurns                       TerminalReason = "max_turns"
	TerminalReasonBackgroundRequested            TerminalReason = "background_requested"
	TerminalReasonCompleted                      TerminalReason = "completed"
	TerminalReasonBudgetExhausted                TerminalReason = "budget_exhausted"
	TerminalReasonStructuredOutputRetryExhausted TerminalReason = "structured_output_retry_exhausted"
	TerminalReasonToolDeferredUnavailable        TerminalReason = "tool_deferred_unavailable"
	TerminalReasonTurnSetupFailed                TerminalReason = "turn_setup_failed"
)

// ResultSuccess (TS: SDKResultSuccess)
type ResultSuccess struct {
	// (always "result")
	Type string `json:"type"`
	// (always "success")
	Subtype                    string   `json:"subtype"`
	DurationMs                 float64  `json:"duration_ms"`
	DurationAPIMs              float64  `json:"duration_api_ms"`
	TTFTMs                     *float64 `json:"ttft_ms,omitempty"`
	TTFTStreamMs               *float64 `json:"ttft_stream_ms,omitempty"`
	TimeToRequestMs            *float64 `json:"time_to_request_ms,omitempty"`
	UserMessageUUID            string   `json:"user_message_uuid,omitempty"`
	UserMessageUUIDs           []string `json:"user_message_uuids,omitempty"`
	ResumeReason               string   `json:"resume_reason,omitempty"`
	LocalCommand               string   `json:"local_command,omitempty"`
	RequestSentWallMs          *float64 `json:"request_sent_wall_ms,omitempty"`
	FirstContentFrameMs        *float64 `json:"first_content_frame_ms,omitempty"`
	FirstStreamPostMs          *float64 `json:"first_stream_post_ms,omitempty"`
	FirstStreamPostAckMs       *float64 `json:"first_stream_post_ack_ms,omitempty"`
	FirstStreamPostQueueWaitMs *float64 `json:"first_stream_post_queue_wait_ms,omitempty"`
	// (one of "durable_post", "ephemeral_post", "retry_backoff", "hold", "none")
	FirstStreamPostQueuedBehind string   `json:"first_stream_post_queued_behind,omitempty"`
	FirstStreamPostWallMs       *float64 `json:"first_stream_post_wall_ms,omitempty"`
	FirstTextPostMs             *float64 `json:"first_text_post_ms,omitempty"`
	FirstTextPostQueueWaitMs    *float64 `json:"first_text_post_queue_wait_ms,omitempty"`
	// (one of "durable_post", "ephemeral_post", "retry_backoff", "hold", "none")
	FirstTextPostQueuedBehind string   `json:"first_text_post_queued_behind,omitempty"`
	FirstTextPostWallMs       *float64 `json:"first_text_post_wall_ms,omitempty"`
	TimeToRequestFromSpawnMs  *float64 `json:"time_to_request_from_spawn_ms,omitempty"`
	WarmSpareClaimed          *bool    `json:"warm_spare_claimed,omitempty"`
	TimeOriginMs              *float64 `json:"time_origin_ms,omitempty"`
	IsError                   bool     `json:"is_error"`
	// (optional, nullable)
	APIErrorStatus *int   `json:"api_error_status,omitempty"`
	NumTurns       int    `json:"num_turns"`
	Result         string `json:"result"`
	// (nullable)
	StopReason *string `json:"stop_reason"`
	// Cumulative estimated cost in USD for this query() call, covering the same query-pipeline calls as modelUsage and sharing its lifecycle: cumulative across turns in streaming-input sessions — each result carries the...
	TotalCostUSD float64 `json:"total_cost_usd"`
	// MAIN AGENT LOOP ONLY — excludes Task subagent, sidechain, and auxiliary model calls, and is per-turn in streaming-input sessions.
	Usage Usage `json:"usage"`
	// Per-model totals for every model call made through the query pipeline during this query() call — main loop, Task subagents, sidechains, and internal calls such as compaction and Workflow agents.
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []PermissionDenial    `json:"permission_denials"`
	// User-initiated sends still waiting in the command queue when this result was produced.
	QueuedTurnCount  *int             `json:"queued_turn_count,omitempty"`
	StructuredOutput json.RawMessage  `json:"structured_output,omitempty"`
	DeferredToolUse  *DeferredToolUse `json:"deferred_tool_use,omitempty"`
	TerminalReason   TerminalReason   `json:"terminal_reason,omitempty"`
	// Delivery sequence of this result within the run: how many results the run numbered before this one, starting at 0, in the order the process writes them.
	ResultIndex            *int                   `json:"result_index,omitempty"`
	FastModeState          FastModeState          `json:"fast_mode_state,omitempty"`
	FastModeDisabledReason FastModeDisabledReason `json:"fast_mode_disabled_reason,omitempty"`
	Origin                 *MessageOrigin         `json:"origin,omitempty"`
	UUID                   string                 `json:"uuid"`
	SessionID              string                 `json:"session_id"`
	// since 0.3.318 (not in 0.3.311)
	// When the claude.ai credential this process's requests send ends for good, as an ISO 8601 instant, read as the result is written.
	CredentialExpiresAt string `json:"credential_expires_at,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ResultSuccess) UnmarshalJSON(b []byte) error {
	type plain ResultSuccess
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ResultSuccess) MarshalJSON() ([]byte, error) {
	type plain ResultSuccess
	if v.Type == "" {
		v.Type = "result"
	}
	if v.Subtype == "" {
		v.Subtype = "success"
	}
	return marshalKeep(plain(v), v.Raw)
}

// StartupFailureReason is a string enum. Why Claude Code refused to start, so a host can offer the fix instead of a retry. org_pin_api_key_conflict: managed settings pin a first-party or Cloud gateway sign-in, and an Anthropic API key or auth token is... (TS: SDKStartupFailureReason)
type StartupFailureReason string

const (
	StartupFailureReasonOrgPinApiKeyConflict              StartupFailureReason = "org_pin_api_key_conflict"
	StartupFailureReasonProviderNotAllowed                StartupFailureReason = "provider_not_allowed"
	StartupFailureReasonOrgVerifyFailed                   StartupFailureReason = "org_verify_failed"
	StartupFailureReasonOrgPinMismatch                    StartupFailureReason = "org_pin_mismatch"
	StartupFailureReasonManagedSettingsInvalid            StartupFailureReason = "managed_settings_invalid"
	StartupFailureReasonRemoteSettingsRequiredUnavailable StartupFailureReason = "remote_settings_required_unavailable"
	StartupFailureReasonGatewaySigninRequired             StartupFailureReason = "gateway_signin_required"
	StartupFailureReasonGatewayAccessDenied               StartupFailureReason = "gateway_access_denied"
	StartupFailureReasonProxyInvalid                      StartupFailureReason = "proxy_invalid"
	StartupFailureReasonTempDirUnusable                   StartupFailureReason = "temp_dir_unusable"
	StartupFailureReasonCwdUnavailable                    StartupFailureReason = "cwd_unavailable"
	StartupFailureReasonShellToolMissing                  StartupFailureReason = "shell_tool_missing"
	StartupFailureReasonSessionHeldByBackground           StartupFailureReason = "session_held_by_background"
	StartupFailureReasonWorktreeResumeRefused             StartupFailureReason = "worktree_resume_refused"
	StartupFailureReasonWorktreeUnverified                StartupFailureReason = "worktree_unverified"
	StartupFailureReasonCliVersionTooOld                  StartupFailureReason = "cli_version_too_old"
	StartupFailureReasonBypassRoot                        StartupFailureReason = "bypass_root"
	StartupFailureReasonOrgConfigRequiredUnavailable      StartupFailureReason = "org_config_required_unavailable"
	StartupFailureReasonOrgConfigRefused                  StartupFailureReason = "org_config_refused"
	StartupFailureReasonSandboxRequiredUnavailable        StartupFailureReason = "sandbox_required_unavailable"   // since 0.3.314 (not in 0.3.311)
	StartupFailureReasonInitializeNotFirst                StartupFailureReason = "initialize_not_first"           // since 0.3.317 (not in 0.3.311)
	StartupFailureReasonConfigFileInvalid                 StartupFailureReason = "config_file_invalid"            // since 0.3.318 (not in 0.3.311)
	StartupFailureReasonManagedSettingsNotDelivered       StartupFailureReason = "managed_settings_not_delivered" // since 0.3.312-0.3.325 (not in 0.3.311)
)

// ResultErrorInvalidConfigFile - inline object (field invalid_config_file of ResultError)
type ResultErrorInvalidConfigFile struct {
	// The file that is not valid JSON.
	Path string `json:"path"`
	// Whether this is Claude Code's global config file.
	IsGlobal bool `json:"is_global"`
	// Whether Claude Code knows the result of its search for a backup of the file.
	BackupSearched bool `json:"backup_searched"`
	// The backup that the message in errors names.
	BackupPath string `json:"backup_path,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ResultErrorInvalidConfigFile) UnmarshalJSON(b []byte) error {
	type plain ResultErrorInvalidConfigFile
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ResultErrorInvalidConfigFile) MarshalJSON() ([]byte, error) {
	type plain ResultErrorInvalidConfigFile
	return marshalKeep(plain(v), v.Raw)
}

// ResultError (TS: SDKResultError)
type ResultError struct {
	// (always "result")
	Type string `json:"type"`
	// (one of "error_during_execution", "error_max_turns", "error_max_budget_usd", "error_max_structured_output_retries")
	Subtype       string  `json:"subtype"`
	DurationMs    float64 `json:"duration_ms"`
	DurationAPIMs float64 `json:"duration_api_ms"`
	IsError       bool    `json:"is_error"`
	NumTurns      int     `json:"num_turns"`
	// (nullable)
	StopReason *string `json:"stop_reason"`
	// Cumulative estimated cost in USD for this query() call, covering the same query-pipeline calls as modelUsage and sharing its lifecycle: cumulative across turns in streaming-input sessions — each result carries the...
	TotalCostUSD float64 `json:"total_cost_usd"`
	// MAIN AGENT LOOP ONLY — excludes Task subagent, sidechain, and auxiliary model calls, and is per-turn in streaming-input sessions.
	Usage Usage `json:"usage"`
	// Per-model totals for every model call made through the query pipeline during this query() call — main loop, Task subagents, sidechains, and internal calls such as compaction and Workflow agents.
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []PermissionDenial    `json:"permission_denials"`
	// User-initiated sends still waiting in the command queue when this result was produced.
	QueuedTurnCount *int     `json:"queued_turn_count,omitempty"`
	Errors          []string `json:"errors"`
	// Set on the zeroed error_during_execution result a stream-json run writes before exiting on a known startup failure; errors carries the same text as stderr.
	StartupFailureReason StartupFailureReason `json:"startup_failure_reason,omitempty"`
	// Client uuid of the user message that triggered this turn (submitMessage options.uuid), echoed back so a consumer can link this error result to the send it answers — the same join key the success variant echoes,...
	UserMessageUUID string `json:"user_message_uuid,omitempty"`
	// Client uuids of every user message whose prompt this turn consumed, in consumption order — all members of a prompt batch the host merged into this one turn (several messages sent close together run as one turn whose...
	UserMessageUUIDs []string `json:"user_message_uuids,omitempty"`
	// Why this turn continued one that a worker restart interrupted.
	ResumeReason   string         `json:"resume_reason,omitempty"`
	TerminalReason TerminalReason `json:"terminal_reason,omitempty"`
	// Delivery sequence of this result within the run: how many results the run numbered before this one, starting at 0, in the order the process writes them.
	ResultIndex            *int                   `json:"result_index,omitempty"`
	FastModeState          FastModeState          `json:"fast_mode_state,omitempty"`
	FastModeDisabledReason FastModeDisabledReason `json:"fast_mode_disabled_reason,omitempty"`
	Origin                 *MessageOrigin         `json:"origin,omitempty"`
	UUID                   string                 `json:"uuid"`
	SessionID              string                 `json:"session_id"`
	// since 0.3.325 (not in 0.3.311)
	// Set when startup_failure_reason is config_file_invalid: the config file that did not parse and its backup, as separate fields.
	InvalidConfigFile *ResultErrorInvalidConfigFile `json:"invalid_config_file,omitempty"`
	// since 0.3.318 (not in 0.3.311)
	// When the claude.ai credential this process's requests send ends for good, as an ISO 8601 instant, read as the result is written.
	CredentialExpiresAt string `json:"credential_expires_at,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ResultError) UnmarshalJSON(b []byte) error {
	type plain ResultError
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ResultError) MarshalJSON() ([]byte, error) {
	type plain ResultError
	if v.Type == "" {
		v.Type = "result"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SystemInitMessageMCPServer - inline object (field mcp_server of SystemInitMessage)
type SystemInitMessageMCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	// Where the server definition came from — same values as McpServerStatus.source (sdk | plugin | a config scope).
	Source string `json:"source,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SystemInitMessageMCPServer) UnmarshalJSON(b []byte) error {
	type plain SystemInitMessageMCPServer
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SystemInitMessageMCPServer) MarshalJSON() ([]byte, error) {
	type plain SystemInitMessageMCPServer
	return marshalKeep(plain(v), v.Raw)
}

// SystemInitMessagePlugin - inline object (field plugin of SystemInitMessage)
type SystemInitMessagePlugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// The plugin's version as declared in its plugin.json manifest, emitted verbatim (plugin-author-controlled — validate before trusting).
	Version string `json:"version,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SystemInitMessagePlugin) UnmarshalJSON(b []byte) error {
	type plain SystemInitMessagePlugin
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SystemInitMessagePlugin) MarshalJSON() ([]byte, error) {
	type plain SystemInitMessagePlugin
	return marshalKeep(plain(v), v.Raw)
}

// SystemInitMessagePluginErrorManagedLock - inline object (field managed_lock of SystemInitMessagePluginError)
type SystemInitMessagePluginErrorManagedLock struct {
	PluginEnabled bool   `json:"plugin_enabled"`
	Name          string `json:"name"`
	Marketplace   string `json:"marketplace,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SystemInitMessagePluginErrorManagedLock) UnmarshalJSON(b []byte) error {
	type plain SystemInitMessagePluginErrorManagedLock
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SystemInitMessagePluginErrorManagedLock) MarshalJSON() ([]byte, error) {
	type plain SystemInitMessagePluginErrorManagedLock
	return marshalKeep(plain(v), v.Raw)
}

// SystemInitMessagePluginError - inline object (field plugin_error of SystemInitMessage)
type SystemInitMessagePluginError struct {
	Plugin  string `json:"plugin"`
	Type    string `json:"type"`
	Message string `json:"message"`
	// Present only when a --plugin-dir, SDK `plugins` or synced directory entry did not load at all: the path of that entry, resolved against the cwd (for a directory, the value its `plugins[]` row would have carried; for...
	Path string `json:"path,omitempty"`
	// since 0.3.325 (not in 0.3.311)
	// Present only when a managed-settings `enabledPlugins` entry dropped this --plugin-dir, SDK `plugins`, skills-directory or synced copy: the two halves of that entry, `name` and `marketplace`, and its value.
	ManagedLock *SystemInitMessagePluginErrorManagedLock `json:"managed_lock,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SystemInitMessagePluginError) UnmarshalJSON(b []byte) error {
	type plain SystemInitMessagePluginError
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SystemInitMessagePluginError) MarshalJSON() ([]byte, error) {
	type plain SystemInitMessagePluginError
	return marshalKeep(plain(v), v.Raw)
}

// SystemInitMessage - Session metadata the CLI emits at the start of each turn, normally ahead of every other message of that turn: session_id, model, working directory, tools, MCP servers, slash commands, permission mode, and the... (TS: SDKSystemMessage)
type SystemInitMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "init")
	Subtype string   `json:"subtype"`
	Agents  []string `json:"agents,omitempty"`
	// Where the credential used for API requests came from: 'ANTHROPIC_API_KEY' (environment variable), 'apiKeyHelper' (the configured helper command), '/login managed key' (an API key created and stored by /login with an...
	APIKeySource      ApiKeySource                 `json:"apiKeySource"`
	Betas             []string                     `json:"betas,omitempty"`
	ClaudeCodeVersion string                       `json:"claude_code_version"`
	Cwd               string                       `json:"cwd"`
	Tools             []string                     `json:"tools"`
	MCPServers        []SystemInitMessageMCPServer `json:"mcp_servers"`
	Model             string                       `json:"model"`
	// Permission mode for controlling how tool executions are handled.
	PermissionMode PermissionMode `json:"permissionMode"`
	SlashCommands  []string       `json:"slash_commands"`
	// Subset of slash_commands whose UX is bound to the local terminal (e.g. exit, statusline).
	TerminalSlashCommands []string                  `json:"terminal_slash_commands,omitempty"`
	OutputStyle           string                    `json:"output_style"`
	Skills                []string                  `json:"skills"`
	Plugins               []SystemInitMessagePlugin `json:"plugins"`
	// Plugin load-time errors.
	PluginErrors           []SystemInitMessagePluginError `json:"plugin_errors,omitempty"`
	FastModeState          FastModeState                  `json:"fast_mode_state,omitempty"`
	FastModeDisabledReason FastModeDisabledReason         `json:"fast_mode_disabled_reason,omitempty"`
	// The effort level the session will send on its next request — after env overrides, session state, org caps and model-support downgrades; the same value get_settings reports as applied.effort. null when no effort... (one of "low", "medium", "high", "xhigh", "max"; optional, nullable)
	Effort *string `json:"effort,omitempty"`
	// Whether the session's transcript is in focus view ('focus': the model is told the user sees only its final message per turn, so clients may collapse each turn to the prompt and the final response; 'default' otherwise). (one of "focus", "default")
	ViewMode string `json:"view_mode,omitempty"`
	// Protocol capabilities this CLI supports, so SDK consumers can feature-detect instead of version-sniffing.
	Capabilities []string `json:"capabilities,omitempty"`
	UUID         string   `json:"uuid"`
	SessionID    string   `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SystemInitMessage) UnmarshalJSON(b []byte) error {
	type plain SystemInitMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SystemInitMessage) MarshalJSON() ([]byte, error) {
	type plain SystemInitMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "init"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PartialAssistantMessage - An incremental streaming event for the assistant message being generated, emitted only when partial messages are requested (--include-partial-messages). (TS: SDKPartialAssistantMessage)
type PartialAssistantMessage struct {
	// (always "stream_event")
	Type string `json:"type"`
	// One Anthropic Messages API streaming event (message_start, content_block_start, content_block_delta, content_block_stop, message_delta, message_stop) as defined for the streaming Messages API.
	Event StreamEvent `json:"event"`
	// (nullable)
	ParentToolUseID *string  `json:"parent_tool_use_id"`
	UUID            string   `json:"uuid"`
	SessionID       string   `json:"session_id"`
	TTFTMs          *float64 `json:"ttft_ms,omitempty"`
	// Client uuid of the user message this turn is answering (submitMessage options.uuid), stamped on a non-ping stream event each time that send changes: the turn's FIRST non-ping stream event (normally the frame that...
	UserMessageUUID string `json:"user_message_uuid,omitempty"`
	// Client uuids of every user message whose prompt this turn has consumed so far, in consumption order — all members of a prompt batch the host merged into this one turn (several messages sent close together run as one...
	UserMessageUUIDs []string `json:"user_message_uuids,omitempty"`
	// Why this frame's turn continues one that a worker restart interrupted.
	ResumeReason string `json:"resume_reason,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PartialAssistantMessage) UnmarshalJSON(b []byte) error {
	type plain PartialAssistantMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PartialAssistantMessage) MarshalJSON() ([]byte, error) {
	type plain PartialAssistantMessage
	if v.Type == "" {
		v.Type = "stream_event"
	}
	return marshalKeep(plain(v), v.Raw)
}

// Status is a string enum. (TS: SDKStatus)
type Status string

const (
	StatusCompacting Status = "compacting"
	StatusRequesting Status = "requesting"
)

// StatusMessage (TS: SDKStatusMessage)
type StatusMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "status")
	Subtype string `json:"subtype"`
	// (nullable)
	Status         *Status        `json:"status"`
	PermissionMode PermissionMode `json:"permissionMode,omitempty"`
	// (one of "success", "failed")
	CompactResult string `json:"compact_result,omitempty"`
	CompactError  string `json:"compact_error,omitempty"`
	UUID          string `json:"uuid"`
	SessionID     string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *StatusMessage) UnmarshalJSON(b []byte) error {
	type plain StatusMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v StatusMessage) MarshalJSON() ([]byte, error) {
	type plain StatusMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "status"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ModelRefusalFallbackMessage - Emitted when the primary model ends the stream with stop_reason "refusal" and the turn is retried once on a fallback model (direction: "retry"). (TS: SDKModelRefusalFallbackMessage)
type ModelRefusalFallbackMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "model_refusal_fallback")
	Subtype string `json:"subtype"`
	// (always "refusal")
	Trigger string `json:"trigger"`
	// (one of "retry", "revert", "sticky")
	Direction string `json:"direction"`
	// 'session': the main thread fell back and the session model is swapped. (one of "session", "local")
	Scope         string `json:"scope,omitempty"`
	OriginalModel string `json:"original_model"`
	FallbackModel string `json:"fallback_model"`
	// (nullable)
	RequestID *string `json:"request_id"`
	// The refusal category ('cyber', 'bio', …): stop_details.category from the refused API response (client lane), or the fallback block's server-gated trigger.category (server lane). (optional, nullable)
	APIRefusalCategory *string `json:"api_refusal_category,omitempty"`
	// stop_details.explanation from the refused API response (client lane only — the server-lane trigger carries no explanation). (optional, nullable)
	APIRefusalExplanation *string `json:"api_refusal_explanation,omitempty"`
	// Wire uuids of the messages this fallback retracted — the refused partial as the consumer received it (one uuid per normalized SDK message; multi-block messages carry per-block derived uuids) plus any tombstoned...
	RetractedMessageUUIDs []string `json:"retracted_message_uuids,omitempty"`
	// UUID of the user message the refused request was for — the rewind target and composer prefill for edit-and-retry. (optional, nullable)
	RefusedUserMessageUUID *string `json:"refused_user_message_uuid,omitempty"`
	Content                string  `json:"content"`
	UUID                   string  `json:"uuid"`
	SessionID              string  `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ModelRefusalFallbackMessage) UnmarshalJSON(b []byte) error {
	type plain ModelRefusalFallbackMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ModelRefusalFallbackMessage) MarshalJSON() ([]byte, error) {
	type plain ModelRefusalFallbackMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "model_refusal_fallback"
	}
	if v.Trigger == "" {
		v.Trigger = "refusal"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ModelRefusalNoFallbackMessage - Emitted when the model ends the stream with stop_reason "refusal" and no retry runs: no fallback model is configured, or per-category routing declined the retry (the mapped fallback target is unresolvable, or... (TS: SDKModelRefusalNoFallbackMessage)
type ModelRefusalNoFallbackMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "model_refusal_no_fallback")
	Subtype       string `json:"subtype"`
	OriginalModel string `json:"original_model"`
	// (nullable)
	RequestID *string `json:"request_id"`
	// (optional, nullable)
	APIRefusalCategory *string `json:"api_refusal_category,omitempty"`
	// (optional, nullable)
	APIRefusalExplanation *string `json:"api_refusal_explanation,omitempty"`
	// (optional, nullable)
	RefusedUserMessageUUID *string `json:"refused_user_message_uuid,omitempty"`
	Content                string  `json:"content"`
	UUID                   string  `json:"uuid"`
	SessionID              string  `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ModelRefusalNoFallbackMessage) UnmarshalJSON(b []byte) error {
	type plain ModelRefusalNoFallbackMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ModelRefusalNoFallbackMessage) MarshalJSON() ([]byte, error) {
	type plain ModelRefusalNoFallbackMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "model_refusal_no_fallback"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PluginInstallMessage - Headless plugin installation progress (CLAUDE_CODE_SYNC_PLUGIN_INSTALL). started/completed bracket the whole install; installed/failed carry a per-marketplace name. (TS: SDKPluginInstallMessage)
type PluginInstallMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "plugin_install")
	Subtype string `json:"subtype"`
	// (one of "started", "installed", "failed", "completed")
	Status    string `json:"status"`
	Name      string `json:"name,omitempty"`
	Error     string `json:"error,omitempty"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PluginInstallMessage) UnmarshalJSON(b []byte) error {
	type plain PluginInstallMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PluginInstallMessage) MarshalJSON() ([]byte, error) {
	type plain PluginInstallMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "plugin_install"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ToolProgressMessageSubagentRetry - inline object (field subagent_retry of ToolProgressMessage)
type ToolProgressMessageSubagentRetry struct {
	AgentID      string  `json:"agent_id"`
	Attempt      int     `json:"attempt"`
	MaxRetries   int     `json:"max_retries"`
	RetryDelayMs float64 `json:"retry_delay_ms"`
	// (nullable)
	ErrorStatus   *int   `json:"error_status"`
	ErrorCategory string `json:"error_category"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolProgressMessageSubagentRetry) UnmarshalJSON(b []byte) error {
	type plain ToolProgressMessageSubagentRetry
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolProgressMessageSubagentRetry) MarshalJSON() ([]byte, error) {
	type plain ToolProgressMessageSubagentRetry
	return marshalKeep(plain(v), v.Raw)
}

// ToolProgressMessage (TS: SDKToolProgressMessage)
type ToolProgressMessage struct {
	// (always "tool_progress")
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	ToolName  string `json:"tool_name"`
	// (nullable)
	ParentToolUseID    *string                           `json:"parent_tool_use_id"`
	ElapsedTimeSeconds float64                           `json:"elapsed_time_seconds"`
	TaskID             string                            `json:"task_id,omitempty"`
	UUID               string                            `json:"uuid"`
	SessionID          string                            `json:"session_id"`
	Heartbeat          *bool                             `json:"heartbeat,omitempty"`
	SubagentType       string                            `json:"subagent_type,omitempty"`
	SubagentRetry      *ToolProgressMessageSubagentRetry `json:"subagent_retry,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolProgressMessage) UnmarshalJSON(b []byte) error {
	type plain ToolProgressMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolProgressMessage) MarshalJSON() ([]byte, error) {
	type plain ToolProgressMessage
	if v.Type == "" {
		v.Type = "tool_progress"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TaskNotificationMessageUsage - inline object (field usage of TaskNotificationMessage)
type TaskNotificationMessageUsage struct {
	TotalTokens int     `json:"total_tokens"`
	ToolUses    int     `json:"tool_uses"`
	DurationMs  float64 `json:"duration_ms"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskNotificationMessageUsage) UnmarshalJSON(b []byte) error {
	type plain TaskNotificationMessageUsage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskNotificationMessageUsage) MarshalJSON() ([]byte, error) {
	type plain TaskNotificationMessageUsage
	return marshalKeep(plain(v), v.Raw)
}

// TaskNotificationMessage (TS: SDKTaskNotificationMessage)
type TaskNotificationMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "task_notification")
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
	// Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
	RunID     string `json:"run_id,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	// (one of "completed", "failed", "stopped")
	Status string `json:"status"`
	// Machine-readable cause, set only when the task did not end through an ordinary completion, failure, or stop. (always "worker_restart")
	Reason     string                        `json:"reason,omitempty"`
	OutputFile string                        `json:"output_file"`
	Summary    string                        `json:"summary"`
	Usage      *TaskNotificationMessageUsage `json:"usage,omitempty"`
	// CLI-owned: for a backgrounded MCP task (task_type mcp_task) that completed, the `resource_link` content blocks of its final result — the files it returned by reference — collected from the raw result before the CLI...
	ResourceLinks  []McpResourceLink `json:"resource_links,omitempty"`
	SkipTranscript *bool             `json:"skip_transcript,omitempty"`
	// True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude them from activity indicators.
	Ambient   *bool  `json:"ambient,omitempty"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskNotificationMessage) UnmarshalJSON(b []byte) error {
	type plain TaskNotificationMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskNotificationMessage) MarshalJSON() ([]byte, error) {
	type plain TaskNotificationMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "task_notification"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TaskStartedMessage (TS: SDKTaskStartedMessage)
type TaskStartedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "task_started")
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
	// Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
	RunID       string `json:"run_id,omitempty"`
	ToolUseID   string `json:"tool_use_id,omitempty"`
	Description string `json:"description"`
	// Subagent type for Task tool subagents.
	SubagentType string `json:"subagent_type,omitempty"`
	// Whether the task was registered in the background (true) or in the foreground with the spawning tool call blocking on it (false).
	IsBackgrounded *bool `json:"is_backgrounded,omitempty"`
	// Nesting depth of a spawned subagent (local_agent) task: 1 for a top-level spawn, N+1 when spawned from inside a depth-N agent.
	SpawnDepth *int `json:"spawn_depth,omitempty"`
	// task_id of the subagent (local_agent) task whose agent launched this subagent, shell, monitor or workflow task.
	ParentTaskID string `json:"parent_task_id,omitempty"`
	TaskType     string `json:"task_type,omitempty"`
	// meta.name from the workflow script (e.g.
	WorkflowName string `json:"workflow_name,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
	// Ambient/housekeeping task.
	SkipTranscript *bool `json:"skip_transcript,omitempty"`
	// True for tasks that are not activity (every skip_transcript task, plus every live-update watcher, requested or auto-started); hosts should exclude them from activity indicators.
	Ambient   *bool  `json:"ambient,omitempty"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskStartedMessage) UnmarshalJSON(b []byte) error {
	type plain TaskStartedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskStartedMessage) MarshalJSON() ([]byte, error) {
	type plain TaskStartedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "task_started"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TaskUpdatedMessagePatch - inline object (field patch of TaskUpdatedMessage)
type TaskUpdatedMessagePatch struct {
	// (one of "pending", "running", "completed", "failed", "killed", "paused")
	Status         string   `json:"status,omitempty"`
	Description    string   `json:"description,omitempty"`
	EndTime        *float64 `json:"end_time,omitempty"`
	TotalPausedMs  *float64 `json:"total_paused_ms,omitempty"`
	Error          string   `json:"error,omitempty"`
	IsBackgrounded *bool    `json:"is_backgrounded,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskUpdatedMessagePatch) UnmarshalJSON(b []byte) error {
	type plain TaskUpdatedMessagePatch
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskUpdatedMessagePatch) MarshalJSON() ([]byte, error) {
	type plain TaskUpdatedMessagePatch
	return marshalKeep(plain(v), v.Raw)
}

// TaskUpdatedMessage (TS: SDKTaskUpdatedMessage)
type TaskUpdatedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "task_updated")
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
	// Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
	RunID string `json:"run_id,omitempty"`
	// Wire-safe subset of TaskState fields that changed.
	Patch     TaskUpdatedMessagePatch `json:"patch"`
	UUID      string                  `json:"uuid"`
	SessionID string                  `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskUpdatedMessage) UnmarshalJSON(b []byte) error {
	type plain TaskUpdatedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskUpdatedMessage) MarshalJSON() ([]byte, error) {
	type plain TaskUpdatedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "task_updated"
	}
	return marshalKeep(plain(v), v.Raw)
}

// TaskProgressMessage (TS: SDKTaskProgressMessage)
type TaskProgressMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "task_progress")
	Subtype string `json:"subtype"`
	TaskID  string `json:"task_id"`
	// Id of one run of the task: equal on every task_* event, background_tasks_changed entry and saved notification (its origin.runId) of that run.
	RunID       string `json:"run_id,omitempty"`
	ToolUseID   string `json:"tool_use_id,omitempty"`
	Description string `json:"description"`
	// Subagent type for Task tool subagents.
	SubagentType string                       `json:"subagent_type,omitempty"`
	Usage        TaskNotificationMessageUsage `json:"usage"`
	LastToolName string                       `json:"last_tool_name,omitempty"`
	// A one-line status for the task's row.
	Summary   string `json:"summary,omitempty"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *TaskProgressMessage) UnmarshalJSON(b []byte) error {
	type plain TaskProgressMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v TaskProgressMessage) MarshalJSON() ([]byte, error) {
	type plain TaskProgressMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "task_progress"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ThinkingTokensMessage - Live thinking-token estimate, digested from thinking_delta.estimated_tokens during the redacted-thinking phase (where the API otherwise streams only pings). estimated_tokens is the running total for the current... (TS: SDKThinkingTokensMessage)
type ThinkingTokensMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "thinking_tokens")
	Subtype              string  `json:"subtype"`
	EstimatedTokens      int     `json:"estimated_tokens"`
	EstimatedTokensDelta float64 `json:"estimated_tokens_delta"`
	// Client uuid of the user message that triggered this turn (submitMessage options.uuid), stamped on every thinking_tokens frame of a headless (stream-json / Agent SDK) turn so a consumer can attribute thinking progress...
	UserMessageUUID string `json:"user_message_uuid,omitempty"`
	UUID            string `json:"uuid"`
	SessionID       string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ThinkingTokensMessage) UnmarshalJSON(b []byte) error {
	type plain ThinkingTokensMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ThinkingTokensMessage) MarshalJSON() ([]byte, error) {
	type plain ThinkingTokensMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "thinking_tokens"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SessionStateChangedMessage - Mirrors notifySessionStateChanged. (TS: SDKSessionStateChangedMessage)
type SessionStateChangedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "session_state_changed")
	Subtype string `json:"subtype"`
	// (one of "idle", "running", "requires_action")
	State     string `json:"state"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionStateChangedMessage) UnmarshalJSON(b []byte) error {
	type plain SessionStateChangedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionStateChangedMessage) MarshalJSON() ([]byte, error) {
	type plain SessionStateChangedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "session_state_changed"
	}
	return marshalKeep(plain(v), v.Raw)
}

// WorkerShuttingDownMessage - Emitted by the bridge on opt-in graceful worker teardown (only when the teardown caller supplied a reason), before the heartbeat stops, so remote clients can show why the worker went away instead of waiting for... (TS: SDKWorkerShuttingDownMessage)
type WorkerShuttingDownMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "worker_shutting_down")
	Subtype string `json:"subtype"`
	// Short snake_case reason set by the host CLI (not user input), e.g.
	Reason    string `json:"reason"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *WorkerShuttingDownMessage) UnmarshalJSON(b []byte) error {
	type plain WorkerShuttingDownMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v WorkerShuttingDownMessage) MarshalJSON() ([]byte, error) {
	type plain WorkerShuttingDownMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "worker_shutting_down"
	}
	return marshalKeep(plain(v), v.Raw)
}

// NotificationMessage - Loop-side text notification. (TS: SDKNotificationMessage)
type NotificationMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "notification")
	Subtype string `json:"subtype"`
	Key     string `json:"key"`
	Text    string `json:"text"`
	// (one of "low", "medium", "high", "immediate")
	Priority  string   `json:"priority"`
	Color     string   `json:"color,omitempty"`
	TimeoutMs *float64 `json:"timeout_ms,omitempty"`
	UUID      string   `json:"uuid"`
	SessionID string   `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *NotificationMessage) UnmarshalJSON(b []byte) error {
	type plain NotificationMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v NotificationMessage) MarshalJSON() ([]byte, error) {
	type plain NotificationMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "notification"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ToolUseSummaryMessage (TS: SDKToolUseSummaryMessage)
type ToolUseSummaryMessage struct {
	// (always "tool_use_summary")
	Type                string   `json:"type"`
	Summary             string   `json:"summary"`
	PrecedingToolUseIDs []string `json:"preceding_tool_use_ids"`
	UUID                string   `json:"uuid"`
	SessionID           string   `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolUseSummaryMessage) UnmarshalJSON(b []byte) error {
	type plain ToolUseSummaryMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolUseSummaryMessage) MarshalJSON() ([]byte, error) {
	type plain ToolUseSummaryMessage
	if v.Type == "" {
		v.Type = "tool_use_summary"
	}
	return marshalKeep(plain(v), v.Raw)
}

// RateLimitInfo - Rate limit information for claude.ai subscription users. (TS: SDKRateLimitInfo)
type RateLimitInfo struct {
	// (one of "allowed", "allowed_warning", "rejected")
	Status   string   `json:"status"`
	ResetsAt *float64 `json:"resetsAt,omitempty"`
	// (one of "five_hour", "seven_day", "seven_day_opus", "seven_day_sonnet", "seven_day_overage_included", "overage")
	RateLimitType string   `json:"rateLimitType,omitempty"`
	Utilization   *float64 `json:"utilization,omitempty"`
	// (one of "allowed", "allowed_warning", "rejected")
	OverageStatus   string   `json:"overageStatus,omitempty"`
	OverageResetsAt *float64 `json:"overageResetsAt,omitempty"`
	// (one of "overage_not_provisioned", "org_level_disabled", "org_level_disabled_until", "out_of_credits", "seat_tier_level_disabled", "member_level_disabled", "seat_tier_zero_credit_limit", "group_zero_credit_limit", "member_zero_credit_limit", "org_service_level_disabled", "no_limits_configured", "fetch_error", "unknown")
	OverageDisabledReason string   `json:"overageDisabledReason,omitempty"`
	IsUsingOverage        *bool    `json:"isUsingOverage,omitempty"`
	OverageInUse          *bool    `json:"overageInUse,omitempty"`
	OverageEnabled        *bool    `json:"overageEnabled,omitempty"`
	SurpassedThreshold    *float64 `json:"surpassedThreshold,omitempty"`
	// Which spend limit blocked the request when it is not the member's own cap: 'group_pool' means a pooled group budget shared by the member's team is used up (the denial otherwise looks like the member's own monthly cap). (one of "service", "channel", "group_pool")
	LimitScope string `json:"limitScope,omitempty"`
	// (always "credits_required")
	ErrorCode                       string `json:"errorCode,omitempty"`
	CanUserPurchaseCredits          *bool  `json:"canUserPurchaseCredits,omitempty"`
	HasChargeableSavedPaymentMethod *bool  `json:"hasChargeableSavedPaymentMethod,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RateLimitInfo) UnmarshalJSON(b []byte) error {
	type plain RateLimitInfo
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RateLimitInfo) MarshalJSON() ([]byte, error) {
	type plain RateLimitInfo
	return marshalKeep(plain(v), v.Raw)
}

// RateLimitEvent - Rate limit event emitted when rate limit info changes. (TS: SDKRateLimitEvent)
type RateLimitEvent struct {
	// (always "rate_limit_event")
	Type string `json:"type"`
	// Rate limit information for claude.ai subscription users.
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
	UUID          string        `json:"uuid"`
	SessionID     string        `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *RateLimitEvent) UnmarshalJSON(b []byte) error {
	type plain RateLimitEvent
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v RateLimitEvent) MarshalJSON() ([]byte, error) {
	type plain RateLimitEvent
	if v.Type == "" {
		v.Type = "rate_limit_event"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PermissionDeniedMessage - Emitted when a tool call is auto-denied without an interactive permission prompt (e.g. auto-mode classifier, dontAsk mode, headless-agent auto-deny, or a deny rule). (TS: SDKPermissionDeniedMessage)
type PermissionDeniedMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "permission_denied")
	Subtype   string `json:"subtype"`
	ToolName  string `json:"tool_name"`
	ToolUseID string `json:"tool_use_id"`
	// Subagent ID when the denied tool call originated inside a subagent.
	AgentID string `json:"agent_id,omitempty"`
	// Discriminator from PermissionDecisionReason (e.g.
	DecisionReasonType string `json:"decision_reason_type,omitempty"`
	// Human-readable reason from the deciding component, when available.
	DecisionReason string `json:"decision_reason,omitempty"`
	// The rejection message returned to the model in the tool_result.
	Message   string `json:"message"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PermissionDeniedMessage) UnmarshalJSON(b []byte) error {
	type plain PermissionDeniedMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PermissionDeniedMessage) MarshalJSON() ([]byte, error) {
	type plain PermissionDeniedMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "permission_denied"
	}
	return marshalKeep(plain(v), v.Raw)
}

// PromptSuggestionMessage - Predicted next user prompt, emitted after each turn when promptSuggestions is enabled. (TS: SDKPromptSuggestionMessage)
type PromptSuggestionMessage struct {
	// (always "prompt_suggestion")
	Type       string `json:"type"`
	Suggestion string `json:"suggestion"`
	UUID       string `json:"uuid"`
	SessionID  string `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *PromptSuggestionMessage) UnmarshalJSON(b []byte) error {
	type plain PromptSuggestionMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v PromptSuggestionMessage) MarshalJSON() ([]byte, error) {
	type plain PromptSuggestionMessage
	if v.Type == "" {
		v.Type = "prompt_suggestion"
	}
	return marshalKeep(plain(v), v.Raw)
}

// MirrorErrorMessageKey - inline object (field key of MirrorErrorMessage)
type MirrorErrorMessageKey struct {
	ProjectKey string `json:"projectKey"`
	SessionID  string `json:"sessionId"`
	Subpath    string `json:"subpath,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MirrorErrorMessageKey) UnmarshalJSON(b []byte) error {
	type plain MirrorErrorMessageKey
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MirrorErrorMessageKey) MarshalJSON() ([]byte, error) {
	type plain MirrorErrorMessageKey
	return marshalKeep(plain(v), v.Raw)
}

// MirrorErrorMessage - Emitted when SessionStore.append() rejects or times out for a transcript-mirror batch after bounded retry (3 attempts with short backoff; timeouts are not retried). (TS: SDKMirrorErrorMessage)
type MirrorErrorMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "mirror_error")
	Subtype   string                `json:"subtype"`
	Error     string                `json:"error"`
	Key       MirrorErrorMessageKey `json:"key"`
	UUID      string                `json:"uuid"`
	SessionID string                `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *MirrorErrorMessage) UnmarshalJSON(b []byte) error {
	type plain MirrorErrorMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v MirrorErrorMessage) MarshalJSON() ([]byte, error) {
	type plain MirrorErrorMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "mirror_error"
	}
	return marshalKeep(plain(v), v.Raw)
}

// since 0.3.317 (not in 0.3.311)
// SessionWritesLostMessage - The CLI could not save lines of a session transcript, changes to ~/.claude.json, or both, because the disk was full or its quota was used up. (TS: SDKSessionWritesLostMessage)
type SessionWritesLostMessage struct {
	// (always "system")
	Type string `json:"type"`
	// (always "session_writes_lost")
	Subtype string `json:"subtype"`
	// Why the writes failed. (one of "ENOSPC", "EDQUOT")
	ErrorCode string `json:"error_code"`
	// What was not saved. (one of "transcript", "global_config")
	Lost []string `json:"lost"`
	// The sessions whose transcripts lost lines, in the order each first lost one.
	TranscriptSessionIDs []string `json:"transcript_session_ids,omitempty"`
	UUID                 string   `json:"uuid"`
	SessionID            string   `json:"session_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionWritesLostMessage) UnmarshalJSON(b []byte) error {
	type plain SessionWritesLostMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionWritesLostMessage) MarshalJSON() ([]byte, error) {
	type plain SessionWritesLostMessage
	if v.Type == "" {
		v.Type = "system"
	}
	if v.Subtype == "" {
		v.Subtype = "session_writes_lost"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SessionInfo - Session metadata returned by listSessions and getSessionInfo. (TS: SDKSessionInfo)
type SessionInfo struct {
	// Unique session identifier (UUID).
	SessionID string `json:"sessionId"`
	// Display title for the session: custom title, auto-generated summary, or first prompt.
	Summary string `json:"summary"`
	// Last modified time in integer milliseconds since epoch.
	LastModified int `json:"lastModified"`
	// File size in bytes.
	FileSize *int `json:"fileSize,omitempty"`
	// User-set session title via /rename.
	CustomTitle string `json:"customTitle,omitempty"`
	// First meaningful user prompt in the session.
	FirstPrompt string `json:"firstPrompt,omitempty"`
	// Git branch at the end of the session.
	GitBranch string `json:"gitBranch,omitempty"`
	// Working directory for the session.
	Cwd string `json:"cwd,omitempty"`
	// User-set session tag.
	Tag string `json:"tag,omitempty"`
	// Creation time in integer milliseconds since epoch, extracted from the first entry's timestamp.
	CreatedAt *int `json:"createdAt,omitempty"`
	// since 0.3.325 (not in 0.3.311)
	// Whether the session is archived: false if it was and has been unarchived, or a prompt was sent in it since.
	Archived *bool `json:"archived,omitempty"`
}

// SessionKey - Identifies a session transcript or subagent transcript in the store. (alpha) (TS: SessionKey)
type SessionKey struct {
	// Caller-defined scope.
	ProjectKey string `json:"projectKey"`
	SessionID  string `json:"sessionId"`
	// Undefined = main transcript.
	Subpath string `json:"subpath,omitempty"`
}

// SessionMessage - A message from a session transcript. (TS: SessionMessage)
type SessionMessage struct {
	// (one of "user", "assistant", "system")
	Type      string          `json:"type"`
	UUID      string          `json:"uuid"`
	SessionID string          `json:"session_id"`
	Message   json.RawMessage `json:"message"`
	// (nullable)
	ParentToolUseID *string `json:"parent_tool_use_id"`
	// agentId of the subagent that spawned this subagent, or null when this message belongs to a depth-1 subagent (spawned by the main loop) or to the main session itself. (nullable)
	ParentAgentID *string `json:"parent_agent_id"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionMessage) UnmarshalJSON(b []byte) error {
	type plain SessionMessage
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionMessage) MarshalJSON() ([]byte, error) {
	type plain SessionMessage
	return marshalKeep(plain(v), v.Raw)
}

// SessionMutationOptions - Options shared by session mutation functions (renameSession, tagSession, deleteSession, forkSession). (TS: SessionMutationOptions)
type SessionMutationOptions struct {
	// Project directory path (same semantics as `listSessions({ dir })`).
	Dir string `json:"dir,omitempty"`
}

// SessionStartHookSpecificOutput (TS: SessionStartHookSpecificOutput)
type SessionStartHookSpecificOutput struct {
	// (always "SessionStart")
	HookEventName      string   `json:"hookEventName"`
	AdditionalContext  string   `json:"additionalContext,omitempty"`
	InitialUserMessage string   `json:"initialUserMessage,omitempty"`
	SessionTitle       string   `json:"sessionTitle,omitempty"`
	WatchPaths         []string `json:"watchPaths,omitempty"`
	// Re-scan skill and command directories after SessionStart hooks complete, so skills installed by the hook are available in the same session
	ReloadSkills *bool `json:"reloadSkills,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionStartHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain SessionStartHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionStartHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain SessionStartHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "SessionStart"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SessionStoreEntry - One JSONL transcript line as observed by a adapter. (alpha) (TS: SessionStoreEntry)
type SessionStoreEntry struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SessionStoreEntry) UnmarshalJSON(b []byte) error {
	type plain SessionStoreEntry
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SessionStoreEntry) MarshalJSON() ([]byte, error) {
	type plain SessionStoreEntry
	return marshalKeep(plain(v), v.Raw)
}

// SessionSummaryEntry - Incrementally-maintained session summary. (alpha) (TS: SessionSummaryEntry)
type SessionSummaryEntry struct {
	SessionID string `json:"sessionId"`
	// Storage write time of the sidecar on the adapter.
	Mtime int `json:"mtime"`
	// Opaque SDK-owned state.
	Data map[string]any `json:"data"`
}

// SetupHookSpecificOutput (TS: SetupHookSpecificOutput)
type SetupHookSpecificOutput struct {
	// (always "Setup")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SetupHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain SetupHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SetupHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain SetupHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "Setup"
	}
	return marshalKeep(plain(v), v.Raw)
}

// StopHookSpecificOutput - Hook-specific output for the Stop event. additionalContext is non-error feedback delivered to the model; the conversation continues so the model can act on it. (TS: StopHookSpecificOutput)
type StopHookSpecificOutput struct {
	// (always "Stop")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *StopHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain StopHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v StopHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain StopHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "Stop"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SubagentStartHookSpecificOutput (TS: SubagentStartHookSpecificOutput)
type SubagentStartHookSpecificOutput struct {
	// (always "SubagentStart")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SubagentStartHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain SubagentStartHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SubagentStartHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain SubagentStartHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "SubagentStart"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SubagentStopHookSpecificOutput - Hook-specific output for the SubagentStop event. additionalContext is non-error feedback delivered to the subagent; the subagent continues so it can act on it. (TS: SubagentStopHookSpecificOutput)
type SubagentStopHookSpecificOutput struct {
	// (always "SubagentStop")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SubagentStopHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain SubagentStopHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SubagentStopHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain SubagentStopHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "SubagentStop"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserPromptSubmitHookSpecificOutput (TS: UserPromptSubmitHookSpecificOutput)
type UserPromptSubmitHookSpecificOutput struct {
	// (always "UserPromptSubmit")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	SessionTitle      string `json:"sessionTitle,omitempty"`
	// When decision is "block", omit the original prompt from the block message
	SuppressOriginalPrompt *bool `json:"suppressOriginalPrompt,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserPromptSubmitHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain UserPromptSubmitHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserPromptSubmitHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain UserPromptSubmitHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "UserPromptSubmit"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserPromptExpansionHookSpecificOutput (TS: UserPromptExpansionHookSpecificOutput)
type UserPromptExpansionHookSpecificOutput struct {
	// (always "UserPromptExpansion")
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
	// When decision is "block", omit the original prompt from the block message
	SuppressOriginalPrompt *bool `json:"suppressOriginalPrompt,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserPromptExpansionHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain UserPromptExpansionHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserPromptExpansionHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain UserPromptExpansionHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "UserPromptExpansion"
	}
	return marshalKeep(plain(v), v.Raw)
}

// WorktreeCreateHookSpecificOutput - Hook-specific output for the WorktreeCreate event. (TS: WorktreeCreateHookSpecificOutput)
type WorktreeCreateHookSpecificOutput struct {
	// (always "WorktreeCreate")
	HookEventName string `json:"hookEventName"`
	WorktreePath  string `json:"worktreePath"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *WorktreeCreateHookSpecificOutput) UnmarshalJSON(b []byte) error {
	type plain WorktreeCreateHookSpecificOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v WorktreeCreateHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type plain WorktreeCreateHookSpecificOutput
	if v.HookEventName == "" {
		v.HookEventName = "WorktreeCreate"
	}
	return marshalKeep(plain(v), v.Raw)
}

// SyncHookJSONOutput (TS: SyncHookJSONOutput)
type SyncHookJSONOutput struct {
	Continue       *bool  `json:"continue,omitempty"`
	SuppressOutput *bool  `json:"suppressOutput,omitempty"`
	StopReason     string `json:"stopReason,omitempty"`
	// (one of "approve", "block")
	Decision      string `json:"decision,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
	// A terminal escape sequence (e.g.
	TerminalSequence   string                                `json:"terminalSequence,omitempty"`
	Reason             string                                `json:"reason,omitempty"`
	HookSpecificOutput *SyncHookJSONOutputHookSpecificOutput `json:"hookSpecificOutput,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *SyncHookJSONOutput) UnmarshalJSON(b []byte) error {
	type plain SyncHookJSONOutput
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v SyncHookJSONOutput) MarshalJSON() ([]byte, error) {
	type plain SyncHookJSONOutput
	return marshalKeep(plain(v), v.Raw)
}

// ToolAnnotations - MCP tool annotations (ToolAnnotations from (hand-modelled stand-in for TS ToolAnnotations)
type ToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    *bool  `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool  `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *ToolAnnotations) UnmarshalJSON(b []byte) error {
	type plain ToolAnnotations
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v ToolAnnotations) MarshalJSON() ([]byte, error) {
	type plain ToolAnnotations
	return marshalKeep(plain(v), v.Raw)
}

// UserDialogRequest - A `request_user_dialog` control request from the CLI, asking the SDK consumer to render a blocking dialog and return the user's choice. (TS: UserDialogRequest)
type UserDialogRequest struct {
	// Identifier for the dialog the host should render.
	DialogKind string `json:"dialogKind"`
	// Dialog-specific data for the host renderer; shape is defined per dialogKind.
	Payload map[string]any `json:"payload"`
	// Present when the dialog is tied to a specific tool invocation.
	ToolUseID string `json:"toolUseID,omitempty"`
}

// UserDialogResultCompleted - variant of UserDialogResult
type UserDialogResultCompleted struct {
	// (always "completed")
	Behavior string          `json:"behavior"`
	Result   json.RawMessage `json:"result"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserDialogResultCompleted) UnmarshalJSON(b []byte) error {
	type plain UserDialogResultCompleted
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserDialogResultCompleted) MarshalJSON() ([]byte, error) {
	type plain UserDialogResultCompleted
	if v.Behavior == "" {
		v.Behavior = "completed"
	}
	return marshalKeep(plain(v), v.Raw)
}

// UserDialogResultCancelled - variant of UserDialogResult
type UserDialogResultCancelled struct {
	// (always "cancelled")
	Behavior string `json:"behavior"`
	// Raw is the JSON this value was decoded from; fields not declared above survive a decode/encode round trip.
	Raw json.RawMessage `json:"-"`
}

func (v *UserDialogResultCancelled) UnmarshalJSON(b []byte) error {
	type plain UserDialogResultCancelled
	return unmarshalKeep(b, (*plain)(v), &v.Raw)
}
func (v UserDialogResultCancelled) MarshalJSON() ([]byte, error) {
	type plain UserDialogResultCancelled
	if v.Behavior == "" {
		v.Behavior = "cancelled"
	}
	return marshalKeep(plain(v), v.Raw)
}

// ContentBlock is a discriminated union: One content block of an assistant or user message (Messages API content blocks and block params, merged into one union keyed by "type"). (hand-modelled stand-in for TS BetaContentBlock).
// Discriminator key(s): "type". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewContentBlock.
type ContentBlock struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *ContentBlock) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type"}, &u.Type)
}
func (u ContentBlock) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewContentBlock wraps a variant value (or any JSON-marshalable value) as a ContentBlock.
func NewContentBlock(v any) (ContentBlock, error) {
	var u ContentBlock
	err := wrapUnion(v, &u)
	return u, err
}

// AsTextBlock decodes u as TextBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsTextBlock() (*TextBlock, error) {
	return asVariant[TextBlock](u.Raw, u.Type == "text", "ContentBlock", "TextBlock")
}

// AsThinkingBlock decodes u as ThinkingBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsThinkingBlock() (*ThinkingBlock, error) {
	return asVariant[ThinkingBlock](u.Raw, u.Type == "thinking", "ContentBlock", "ThinkingBlock")
}

// AsRedactedThinkingBlock decodes u as RedactedThinkingBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsRedactedThinkingBlock() (*RedactedThinkingBlock, error) {
	return asVariant[RedactedThinkingBlock](u.Raw, u.Type == "redacted_thinking", "ContentBlock", "RedactedThinkingBlock")
}

// AsToolUseBlock decodes u as ToolUseBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsToolUseBlock() (*ToolUseBlock, error) {
	return asVariant[ToolUseBlock](u.Raw, u.Type == "tool_use", "ContentBlock", "ToolUseBlock")
}

// AsServerToolUseBlock decodes u as ServerToolUseBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsServerToolUseBlock() (*ServerToolUseBlock, error) {
	return asVariant[ServerToolUseBlock](u.Raw, u.Type == "server_tool_use", "ContentBlock", "ServerToolUseBlock")
}

// AsToolResultBlock decodes u as ToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsToolResultBlock() (*ToolResultBlock, error) {
	return asVariant[ToolResultBlock](u.Raw, u.Type == "tool_result", "ContentBlock", "ToolResultBlock")
}

// AsImageBlock decodes u as ImageBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsImageBlock() (*ImageBlock, error) {
	return asVariant[ImageBlock](u.Raw, u.Type == "image", "ContentBlock", "ImageBlock")
}

// AsDocumentBlock decodes u as DocumentBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsDocumentBlock() (*DocumentBlock, error) {
	return asVariant[DocumentBlock](u.Raw, u.Type == "document", "ContentBlock", "DocumentBlock")
}

// AsSearchResultBlock decodes u as SearchResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsSearchResultBlock() (*SearchResultBlock, error) {
	return asVariant[SearchResultBlock](u.Raw, u.Type == "search_result", "ContentBlock", "SearchResultBlock")
}

// AsMCPToolUseBlock decodes u as MCPToolUseBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsMCPToolUseBlock() (*MCPToolUseBlock, error) {
	return asVariant[MCPToolUseBlock](u.Raw, u.Type == "mcp_tool_use", "ContentBlock", "MCPToolUseBlock")
}

// AsMCPToolResultBlock decodes u as MCPToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsMCPToolResultBlock() (*MCPToolResultBlock, error) {
	return asVariant[MCPToolResultBlock](u.Raw, u.Type == "mcp_tool_result", "ContentBlock", "MCPToolResultBlock")
}

// AsContainerUploadBlock decodes u as ContainerUploadBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsContainerUploadBlock() (*ContainerUploadBlock, error) {
	return asVariant[ContainerUploadBlock](u.Raw, u.Type == "container_upload", "ContentBlock", "ContainerUploadBlock")
}

// AsWebSearchToolResultBlock decodes u as WebSearchToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsWebSearchToolResultBlock() (*WebSearchToolResultBlock, error) {
	return asVariant[WebSearchToolResultBlock](u.Raw, u.Type == "web_search_tool_result", "ContentBlock", "WebSearchToolResultBlock")
}

// AsWebFetchToolResultBlock decodes u as WebFetchToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsWebFetchToolResultBlock() (*WebFetchToolResultBlock, error) {
	return asVariant[WebFetchToolResultBlock](u.Raw, u.Type == "web_fetch_tool_result", "ContentBlock", "WebFetchToolResultBlock")
}

// AsCodeExecutionToolResultBlock decodes u as CodeExecutionToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsCodeExecutionToolResultBlock() (*CodeExecutionToolResultBlock, error) {
	return asVariant[CodeExecutionToolResultBlock](u.Raw, u.Type == "code_execution_tool_result", "ContentBlock", "CodeExecutionToolResultBlock")
}

// AsBashCodeExecutionToolResultBlock decodes u as BashCodeExecutionToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsBashCodeExecutionToolResultBlock() (*BashCodeExecutionToolResultBlock, error) {
	return asVariant[BashCodeExecutionToolResultBlock](u.Raw, u.Type == "bash_code_execution_tool_result", "ContentBlock", "BashCodeExecutionToolResultBlock")
}

// AsTextEditorCodeExecutionToolResultBlock decodes u as TextEditorCodeExecutionToolResultBlock; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlock) AsTextEditorCodeExecutionToolResultBlock() (*TextEditorCodeExecutionToolResultBlock, error) {
	return asVariant[TextEditorCodeExecutionToolResultBlock](u.Raw, u.Type == "text_editor_code_execution_tool_result", "ContentBlock", "TextEditorCodeExecutionToolResultBlock")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u ContentBlock) Decode() (any, error) {
	switch {
	case u.Type == "text":
		return u.AsTextBlock()
	case u.Type == "thinking":
		return u.AsThinkingBlock()
	case u.Type == "redacted_thinking":
		return u.AsRedactedThinkingBlock()
	case u.Type == "tool_use":
		return u.AsToolUseBlock()
	case u.Type == "server_tool_use":
		return u.AsServerToolUseBlock()
	case u.Type == "tool_result":
		return u.AsToolResultBlock()
	case u.Type == "image":
		return u.AsImageBlock()
	case u.Type == "document":
		return u.AsDocumentBlock()
	case u.Type == "search_result":
		return u.AsSearchResultBlock()
	case u.Type == "mcp_tool_use":
		return u.AsMCPToolUseBlock()
	case u.Type == "mcp_tool_result":
		return u.AsMCPToolResultBlock()
	case u.Type == "container_upload":
		return u.AsContainerUploadBlock()
	case u.Type == "web_search_tool_result":
		return u.AsWebSearchToolResultBlock()
	case u.Type == "web_fetch_tool_result":
		return u.AsWebFetchToolResultBlock()
	case u.Type == "code_execution_tool_result":
		return u.AsCodeExecutionToolResultBlock()
	case u.Type == "bash_code_execution_tool_result":
		return u.AsBashCodeExecutionToolResultBlock()
	case u.Type == "text_editor_code_execution_tool_result":
		return u.AsTextEditorCodeExecutionToolResultBlock()
	}
	return nil, unknownVariant("ContentBlock", u.Raw)
}

// ContentBlockDelta is a discriminated union: Delta of a content_block_delta event. (hand-modelled stand-in for TS ContentBlockDelta).
// Discriminator key(s): "type". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewContentBlockDelta.
type ContentBlockDelta struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *ContentBlockDelta) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type"}, &u.Type)
}
func (u ContentBlockDelta) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewContentBlockDelta wraps a variant value (or any JSON-marshalable value) as a ContentBlockDelta.
func NewContentBlockDelta(v any) (ContentBlockDelta, error) {
	var u ContentBlockDelta
	err := wrapUnion(v, &u)
	return u, err
}

// AsTextDelta decodes u as TextDelta; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlockDelta) AsTextDelta() (*TextDelta, error) {
	return asVariant[TextDelta](u.Raw, u.Type == "text_delta", "ContentBlockDelta", "TextDelta")
}

// AsInputJSONDelta decodes u as InputJSONDelta; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlockDelta) AsInputJSONDelta() (*InputJSONDelta, error) {
	return asVariant[InputJSONDelta](u.Raw, u.Type == "input_json_delta", "ContentBlockDelta", "InputJSONDelta")
}

// AsCitationsDelta decodes u as CitationsDelta; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlockDelta) AsCitationsDelta() (*CitationsDelta, error) {
	return asVariant[CitationsDelta](u.Raw, u.Type == "citations_delta", "ContentBlockDelta", "CitationsDelta")
}

// AsThinkingDelta decodes u as ThinkingDelta; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlockDelta) AsThinkingDelta() (*ThinkingDelta, error) {
	return asVariant[ThinkingDelta](u.Raw, u.Type == "thinking_delta", "ContentBlockDelta", "ThinkingDelta")
}

// AsSignatureDelta decodes u as SignatureDelta; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ContentBlockDelta) AsSignatureDelta() (*SignatureDelta, error) {
	return asVariant[SignatureDelta](u.Raw, u.Type == "signature_delta", "ContentBlockDelta", "SignatureDelta")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u ContentBlockDelta) Decode() (any, error) {
	switch {
	case u.Type == "text_delta":
		return u.AsTextDelta()
	case u.Type == "input_json_delta":
		return u.AsInputJSONDelta()
	case u.Type == "citations_delta":
		return u.AsCitationsDelta()
	case u.Type == "thinking_delta":
		return u.AsThinkingDelta()
	case u.Type == "signature_delta":
		return u.AsSignatureDelta()
	}
	return nil, unknownVariant("ContentBlockDelta", u.Raw)
}

// StreamEvent is a discriminated union: One Anthropic Messages API streaming event (BetaRawMessageStreamEvent), as carried on stream_event messages. (hand-modelled stand-in for TS BetaRawMessageStreamEvent).
// Discriminator key(s): "type". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewStreamEvent.
type StreamEvent struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *StreamEvent) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type"}, &u.Type)
}
func (u StreamEvent) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewStreamEvent wraps a variant value (or any JSON-marshalable value) as a StreamEvent.
func NewStreamEvent(v any) (StreamEvent, error) {
	var u StreamEvent
	err := wrapUnion(v, &u)
	return u, err
}

// AsMessageStartEvent decodes u as MessageStartEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StreamEvent) AsMessageStartEvent() (*MessageStartEvent, error) {
	return asVariant[MessageStartEvent](u.Raw, u.Type == "message_start", "StreamEvent", "MessageStartEvent")
}

// AsMessageDeltaEvent decodes u as MessageDeltaEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StreamEvent) AsMessageDeltaEvent() (*MessageDeltaEvent, error) {
	return asVariant[MessageDeltaEvent](u.Raw, u.Type == "message_delta", "StreamEvent", "MessageDeltaEvent")
}

// AsMessageStopEvent decodes u as MessageStopEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StreamEvent) AsMessageStopEvent() (*MessageStopEvent, error) {
	return asVariant[MessageStopEvent](u.Raw, u.Type == "message_stop", "StreamEvent", "MessageStopEvent")
}

// AsContentBlockStartEvent decodes u as ContentBlockStartEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StreamEvent) AsContentBlockStartEvent() (*ContentBlockStartEvent, error) {
	return asVariant[ContentBlockStartEvent](u.Raw, u.Type == "content_block_start", "StreamEvent", "ContentBlockStartEvent")
}

// AsContentBlockDeltaEvent decodes u as ContentBlockDeltaEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StreamEvent) AsContentBlockDeltaEvent() (*ContentBlockDeltaEvent, error) {
	return asVariant[ContentBlockDeltaEvent](u.Raw, u.Type == "content_block_delta", "StreamEvent", "ContentBlockDeltaEvent")
}

// AsContentBlockStopEvent decodes u as ContentBlockStopEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StreamEvent) AsContentBlockStopEvent() (*ContentBlockStopEvent, error) {
	return asVariant[ContentBlockStopEvent](u.Raw, u.Type == "content_block_stop", "StreamEvent", "ContentBlockStopEvent")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u StreamEvent) Decode() (any, error) {
	switch {
	case u.Type == "message_start":
		return u.AsMessageStartEvent()
	case u.Type == "message_delta":
		return u.AsMessageDeltaEvent()
	case u.Type == "message_stop":
		return u.AsMessageStopEvent()
	case u.Type == "content_block_start":
		return u.AsContentBlockStartEvent()
	case u.Type == "content_block_delta":
		return u.AsContentBlockDeltaEvent()
	case u.Type == "content_block_stop":
		return u.AsContentBlockStopEvent()
	}
	return nil, unknownVariant("StreamEvent", u.Raw)
}

// PermissionUpdate is a discriminated union (TS: PermissionUpdate).
// Discriminator key(s): "type". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewPermissionUpdate.
type PermissionUpdate struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *PermissionUpdate) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type"}, &u.Type)
}
func (u PermissionUpdate) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewPermissionUpdate wraps a variant value (or any JSON-marshalable value) as a PermissionUpdate.
func NewPermissionUpdate(v any) (PermissionUpdate, error) {
	var u PermissionUpdate
	err := wrapUnion(v, &u)
	return u, err
}

// AsPermissionUpdateAddRules decodes u as PermissionUpdateAddRules; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionUpdate) AsPermissionUpdateAddRules() (*PermissionUpdateAddRules, error) {
	return asVariant[PermissionUpdateAddRules](u.Raw, u.Type == "addRules", "PermissionUpdate", "PermissionUpdateAddRules")
}

// AsPermissionUpdateReplaceRules decodes u as PermissionUpdateReplaceRules; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionUpdate) AsPermissionUpdateReplaceRules() (*PermissionUpdateReplaceRules, error) {
	return asVariant[PermissionUpdateReplaceRules](u.Raw, u.Type == "replaceRules", "PermissionUpdate", "PermissionUpdateReplaceRules")
}

// AsPermissionUpdateRemoveRules decodes u as PermissionUpdateRemoveRules; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionUpdate) AsPermissionUpdateRemoveRules() (*PermissionUpdateRemoveRules, error) {
	return asVariant[PermissionUpdateRemoveRules](u.Raw, u.Type == "removeRules", "PermissionUpdate", "PermissionUpdateRemoveRules")
}

// AsPermissionUpdateSetMode decodes u as PermissionUpdateSetMode; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionUpdate) AsPermissionUpdateSetMode() (*PermissionUpdateSetMode, error) {
	return asVariant[PermissionUpdateSetMode](u.Raw, u.Type == "setMode", "PermissionUpdate", "PermissionUpdateSetMode")
}

// AsPermissionUpdateAddDirectories decodes u as PermissionUpdateAddDirectories; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionUpdate) AsPermissionUpdateAddDirectories() (*PermissionUpdateAddDirectories, error) {
	return asVariant[PermissionUpdateAddDirectories](u.Raw, u.Type == "addDirectories", "PermissionUpdate", "PermissionUpdateAddDirectories")
}

// AsPermissionUpdateRemoveDirectories decodes u as PermissionUpdateRemoveDirectories; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionUpdate) AsPermissionUpdateRemoveDirectories() (*PermissionUpdateRemoveDirectories, error) {
	return asVariant[PermissionUpdateRemoveDirectories](u.Raw, u.Type == "removeDirectories", "PermissionUpdate", "PermissionUpdateRemoveDirectories")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u PermissionUpdate) Decode() (any, error) {
	switch {
	case u.Type == "addRules":
		return u.AsPermissionUpdateAddRules()
	case u.Type == "replaceRules":
		return u.AsPermissionUpdateReplaceRules()
	case u.Type == "removeRules":
		return u.AsPermissionUpdateRemoveRules()
	case u.Type == "setMode":
		return u.AsPermissionUpdateSetMode()
	case u.Type == "addDirectories":
		return u.AsPermissionUpdateAddDirectories()
	case u.Type == "removeDirectories":
		return u.AsPermissionUpdateRemoveDirectories()
	}
	return nil, unknownVariant("PermissionUpdate", u.Raw)
}

// HookInput is a discriminated union (TS: HookInput).
// Discriminator key(s): "hook_event_name". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewHookInput.
type HookInput struct {
	// HookEventName is the "hook_event_name" discriminator ("" when absent).
	HookEventName string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *HookInput) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"hook_event_name"}, &u.HookEventName)
}
func (u HookInput) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewHookInput wraps a variant value (or any JSON-marshalable value) as a HookInput.
func NewHookInput(v any) (HookInput, error) { var u HookInput; err := wrapUnion(v, &u); return u, err }

// AsPreToolUseHookInput decodes u as PreToolUseHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPreToolUseHookInput() (*PreToolUseHookInput, error) {
	return asVariant[PreToolUseHookInput](u.Raw, u.HookEventName == "PreToolUse", "HookInput", "PreToolUseHookInput")
}

// AsPostToolUseHookInput decodes u as PostToolUseHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPostToolUseHookInput() (*PostToolUseHookInput, error) {
	return asVariant[PostToolUseHookInput](u.Raw, u.HookEventName == "PostToolUse", "HookInput", "PostToolUseHookInput")
}

// AsPostToolUseFailureHookInput decodes u as PostToolUseFailureHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPostToolUseFailureHookInput() (*PostToolUseFailureHookInput, error) {
	return asVariant[PostToolUseFailureHookInput](u.Raw, u.HookEventName == "PostToolUseFailure", "HookInput", "PostToolUseFailureHookInput")
}

// AsPostToolBatchHookInput decodes u as PostToolBatchHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPostToolBatchHookInput() (*PostToolBatchHookInput, error) {
	return asVariant[PostToolBatchHookInput](u.Raw, u.HookEventName == "PostToolBatch", "HookInput", "PostToolBatchHookInput")
}

// AsPermissionDeniedHookInput decodes u as PermissionDeniedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPermissionDeniedHookInput() (*PermissionDeniedHookInput, error) {
	return asVariant[PermissionDeniedHookInput](u.Raw, u.HookEventName == "PermissionDenied", "HookInput", "PermissionDeniedHookInput")
}

// AsNotificationHookInput decodes u as NotificationHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsNotificationHookInput() (*NotificationHookInput, error) {
	return asVariant[NotificationHookInput](u.Raw, u.HookEventName == "Notification", "HookInput", "NotificationHookInput")
}

// AsUserPromptSubmitHookInput decodes u as UserPromptSubmitHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsUserPromptSubmitHookInput() (*UserPromptSubmitHookInput, error) {
	return asVariant[UserPromptSubmitHookInput](u.Raw, u.HookEventName == "UserPromptSubmit", "HookInput", "UserPromptSubmitHookInput")
}

// AsUserPromptExpansionHookInput decodes u as UserPromptExpansionHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsUserPromptExpansionHookInput() (*UserPromptExpansionHookInput, error) {
	return asVariant[UserPromptExpansionHookInput](u.Raw, u.HookEventName == "UserPromptExpansion", "HookInput", "UserPromptExpansionHookInput")
}

// AsSessionStartHookInput decodes u as SessionStartHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsSessionStartHookInput() (*SessionStartHookInput, error) {
	return asVariant[SessionStartHookInput](u.Raw, u.HookEventName == "SessionStart", "HookInput", "SessionStartHookInput")
}

// AsSessionEndHookInput decodes u as SessionEndHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsSessionEndHookInput() (*SessionEndHookInput, error) {
	return asVariant[SessionEndHookInput](u.Raw, u.HookEventName == "SessionEnd", "HookInput", "SessionEndHookInput")
}

// AsStopHookInput decodes u as StopHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsStopHookInput() (*StopHookInput, error) {
	return asVariant[StopHookInput](u.Raw, u.HookEventName == "Stop", "HookInput", "StopHookInput")
}

// AsStopFailureHookInput decodes u as StopFailureHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsStopFailureHookInput() (*StopFailureHookInput, error) {
	return asVariant[StopFailureHookInput](u.Raw, u.HookEventName == "StopFailure", "HookInput", "StopFailureHookInput")
}

// AsSubagentStartHookInput decodes u as SubagentStartHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsSubagentStartHookInput() (*SubagentStartHookInput, error) {
	return asVariant[SubagentStartHookInput](u.Raw, u.HookEventName == "SubagentStart", "HookInput", "SubagentStartHookInput")
}

// AsSubagentStopHookInput decodes u as SubagentStopHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsSubagentStopHookInput() (*SubagentStopHookInput, error) {
	return asVariant[SubagentStopHookInput](u.Raw, u.HookEventName == "SubagentStop", "HookInput", "SubagentStopHookInput")
}

// AsPreCompactHookInput decodes u as PreCompactHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPreCompactHookInput() (*PreCompactHookInput, error) {
	return asVariant[PreCompactHookInput](u.Raw, u.HookEventName == "PreCompact", "HookInput", "PreCompactHookInput")
}

// AsPostCompactHookInput decodes u as PostCompactHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPostCompactHookInput() (*PostCompactHookInput, error) {
	return asVariant[PostCompactHookInput](u.Raw, u.HookEventName == "PostCompact", "HookInput", "PostCompactHookInput")
}

// AsPreModelSwitchHookInput decodes u as PreModelSwitchHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPreModelSwitchHookInput() (*PreModelSwitchHookInput, error) {
	return asVariant[PreModelSwitchHookInput](u.Raw, u.HookEventName == "PreModelSwitch", "HookInput", "PreModelSwitchHookInput")
}

// AsPostModelSwitchHookInput decodes u as PostModelSwitchHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPostModelSwitchHookInput() (*PostModelSwitchHookInput, error) {
	return asVariant[PostModelSwitchHookInput](u.Raw, u.HookEventName == "PostModelSwitch", "HookInput", "PostModelSwitchHookInput")
}

// AsPermissionRequestHookInput decodes u as PermissionRequestHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsPermissionRequestHookInput() (*PermissionRequestHookInput, error) {
	return asVariant[PermissionRequestHookInput](u.Raw, u.HookEventName == "PermissionRequest", "HookInput", "PermissionRequestHookInput")
}

// AsSetupHookInput decodes u as SetupHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsSetupHookInput() (*SetupHookInput, error) {
	return asVariant[SetupHookInput](u.Raw, u.HookEventName == "Setup", "HookInput", "SetupHookInput")
}

// AsTeammateIdleHookInput decodes u as TeammateIdleHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsTeammateIdleHookInput() (*TeammateIdleHookInput, error) {
	return asVariant[TeammateIdleHookInput](u.Raw, u.HookEventName == "TeammateIdle", "HookInput", "TeammateIdleHookInput")
}

// AsTaskCreatedHookInput decodes u as TaskCreatedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsTaskCreatedHookInput() (*TaskCreatedHookInput, error) {
	return asVariant[TaskCreatedHookInput](u.Raw, u.HookEventName == "TaskCreated", "HookInput", "TaskCreatedHookInput")
}

// AsTaskCompletedHookInput decodes u as TaskCompletedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsTaskCompletedHookInput() (*TaskCompletedHookInput, error) {
	return asVariant[TaskCompletedHookInput](u.Raw, u.HookEventName == "TaskCompleted", "HookInput", "TaskCompletedHookInput")
}

// AsElicitationHookInput decodes u as ElicitationHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsElicitationHookInput() (*ElicitationHookInput, error) {
	return asVariant[ElicitationHookInput](u.Raw, u.HookEventName == "Elicitation", "HookInput", "ElicitationHookInput")
}

// AsElicitationResultHookInput decodes u as ElicitationResultHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsElicitationResultHookInput() (*ElicitationResultHookInput, error) {
	return asVariant[ElicitationResultHookInput](u.Raw, u.HookEventName == "ElicitationResult", "HookInput", "ElicitationResultHookInput")
}

// AsConfigChangeHookInput decodes u as ConfigChangeHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsConfigChangeHookInput() (*ConfigChangeHookInput, error) {
	return asVariant[ConfigChangeHookInput](u.Raw, u.HookEventName == "ConfigChange", "HookInput", "ConfigChangeHookInput")
}

// AsInstructionsLoadedHookInput decodes u as InstructionsLoadedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsInstructionsLoadedHookInput() (*InstructionsLoadedHookInput, error) {
	return asVariant[InstructionsLoadedHookInput](u.Raw, u.HookEventName == "InstructionsLoaded", "HookInput", "InstructionsLoadedHookInput")
}

// AsWorktreeCreateHookInput decodes u as WorktreeCreateHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsWorktreeCreateHookInput() (*WorktreeCreateHookInput, error) {
	return asVariant[WorktreeCreateHookInput](u.Raw, u.HookEventName == "WorktreeCreate", "HookInput", "WorktreeCreateHookInput")
}

// AsWorktreeRemoveHookInput decodes u as WorktreeRemoveHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsWorktreeRemoveHookInput() (*WorktreeRemoveHookInput, error) {
	return asVariant[WorktreeRemoveHookInput](u.Raw, u.HookEventName == "WorktreeRemove", "HookInput", "WorktreeRemoveHookInput")
}

// AsCwdChangedHookInput decodes u as CwdChangedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsCwdChangedHookInput() (*CwdChangedHookInput, error) {
	return asVariant[CwdChangedHookInput](u.Raw, u.HookEventName == "CwdChanged", "HookInput", "CwdChangedHookInput")
}

// AsFileChangedHookInput decodes u as FileChangedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsFileChangedHookInput() (*FileChangedHookInput, error) {
	return asVariant[FileChangedHookInput](u.Raw, u.HookEventName == "FileChanged", "HookInput", "FileChangedHookInput")
}

// AsDirectoryAddedHookInput decodes u as DirectoryAddedHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsDirectoryAddedHookInput() (*DirectoryAddedHookInput, error) {
	return asVariant[DirectoryAddedHookInput](u.Raw, u.HookEventName == "DirectoryAdded", "HookInput", "DirectoryAddedHookInput")
}

// AsMessageDisplayHookInput decodes u as MessageDisplayHookInput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u HookInput) AsMessageDisplayHookInput() (*MessageDisplayHookInput, error) {
	return asVariant[MessageDisplayHookInput](u.Raw, u.HookEventName == "MessageDisplay", "HookInput", "MessageDisplayHookInput")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u HookInput) Decode() (any, error) {
	switch {
	case u.HookEventName == "PreToolUse":
		return u.AsPreToolUseHookInput()
	case u.HookEventName == "PostToolUse":
		return u.AsPostToolUseHookInput()
	case u.HookEventName == "PostToolUseFailure":
		return u.AsPostToolUseFailureHookInput()
	case u.HookEventName == "PostToolBatch":
		return u.AsPostToolBatchHookInput()
	case u.HookEventName == "PermissionDenied":
		return u.AsPermissionDeniedHookInput()
	case u.HookEventName == "Notification":
		return u.AsNotificationHookInput()
	case u.HookEventName == "UserPromptSubmit":
		return u.AsUserPromptSubmitHookInput()
	case u.HookEventName == "UserPromptExpansion":
		return u.AsUserPromptExpansionHookInput()
	case u.HookEventName == "SessionStart":
		return u.AsSessionStartHookInput()
	case u.HookEventName == "SessionEnd":
		return u.AsSessionEndHookInput()
	case u.HookEventName == "Stop":
		return u.AsStopHookInput()
	case u.HookEventName == "StopFailure":
		return u.AsStopFailureHookInput()
	case u.HookEventName == "SubagentStart":
		return u.AsSubagentStartHookInput()
	case u.HookEventName == "SubagentStop":
		return u.AsSubagentStopHookInput()
	case u.HookEventName == "PreCompact":
		return u.AsPreCompactHookInput()
	case u.HookEventName == "PostCompact":
		return u.AsPostCompactHookInput()
	case u.HookEventName == "PreModelSwitch":
		return u.AsPreModelSwitchHookInput()
	case u.HookEventName == "PostModelSwitch":
		return u.AsPostModelSwitchHookInput()
	case u.HookEventName == "PermissionRequest":
		return u.AsPermissionRequestHookInput()
	case u.HookEventName == "Setup":
		return u.AsSetupHookInput()
	case u.HookEventName == "TeammateIdle":
		return u.AsTeammateIdleHookInput()
	case u.HookEventName == "TaskCreated":
		return u.AsTaskCreatedHookInput()
	case u.HookEventName == "TaskCompleted":
		return u.AsTaskCompletedHookInput()
	case u.HookEventName == "Elicitation":
		return u.AsElicitationHookInput()
	case u.HookEventName == "ElicitationResult":
		return u.AsElicitationResultHookInput()
	case u.HookEventName == "ConfigChange":
		return u.AsConfigChangeHookInput()
	case u.HookEventName == "InstructionsLoaded":
		return u.AsInstructionsLoadedHookInput()
	case u.HookEventName == "WorktreeCreate":
		return u.AsWorktreeCreateHookInput()
	case u.HookEventName == "WorktreeRemove":
		return u.AsWorktreeRemoveHookInput()
	case u.HookEventName == "CwdChanged":
		return u.AsCwdChangedHookInput()
	case u.HookEventName == "FileChanged":
		return u.AsFileChangedHookInput()
	case u.HookEventName == "DirectoryAdded":
		return u.AsDirectoryAddedHookInput()
	case u.HookEventName == "MessageDisplay":
		return u.AsMessageDisplayHookInput()
	}
	return nil, unknownVariant("HookInput", u.Raw)
}

// McpServerConfig is a discriminated union: Union of MCP server configs a host can pass. (hand-modelled stand-in for TS McpServerConfig).
// Discriminator key(s): "type". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewMcpServerConfig.
type McpServerConfig struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *McpServerConfig) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type"}, &u.Type)
}
func (u McpServerConfig) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewMcpServerConfig wraps a variant value (or any JSON-marshalable value) as a McpServerConfig.
func NewMcpServerConfig(v any) (McpServerConfig, error) {
	var u McpServerConfig
	err := wrapUnion(v, &u)
	return u, err
}

// AsMcpStdioServerConfig decodes u as McpStdioServerConfig; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u McpServerConfig) AsMcpStdioServerConfig() (*McpStdioServerConfig, error) {
	return asVariant[McpStdioServerConfig](u.Raw, (u.Type == "stdio" || u.Type == ""), "McpServerConfig", "McpStdioServerConfig")
}

// AsMcpSSEServerConfig decodes u as McpSSEServerConfig; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u McpServerConfig) AsMcpSSEServerConfig() (*McpSSEServerConfig, error) {
	return asVariant[McpSSEServerConfig](u.Raw, u.Type == "sse", "McpServerConfig", "McpSSEServerConfig")
}

// AsMcpHttpServerConfig decodes u as McpHttpServerConfig; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u McpServerConfig) AsMcpHttpServerConfig() (*McpHttpServerConfig, error) {
	return asVariant[McpHttpServerConfig](u.Raw, u.Type == "http", "McpServerConfig", "McpHttpServerConfig")
}

// AsMcpSdkServerConfig decodes u as McpSdkServerConfig; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u McpServerConfig) AsMcpSdkServerConfig() (*McpSdkServerConfig, error) {
	return asVariant[McpSdkServerConfig](u.Raw, u.Type == "sdk", "McpServerConfig", "McpSdkServerConfig")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u McpServerConfig) Decode() (any, error) {
	switch {
	case (u.Type == "stdio" || u.Type == ""):
		return u.AsMcpStdioServerConfig()
	case u.Type == "sse":
		return u.AsMcpSSEServerConfig()
	case u.Type == "http":
		return u.AsMcpHttpServerConfig()
	case u.Type == "sdk":
		return u.AsMcpSdkServerConfig()
	}
	return nil, unknownVariant("McpServerConfig", u.Raw)
}

// ControlRequestInner is a discriminated union (TS: SDKControlRequestInner, not exported (wire type)).
// Discriminator key(s): "subtype". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewControlRequestInner.
type ControlRequestInner struct {
	// Subtype is the "subtype" discriminator ("" when absent).
	Subtype string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *ControlRequestInner) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"subtype"}, &u.Subtype)
}
func (u ControlRequestInner) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewControlRequestInner wraps a variant value (or any JSON-marshalable value) as a ControlRequestInner.
func NewControlRequestInner(v any) (ControlRequestInner, error) {
	var u ControlRequestInner
	err := wrapUnion(v, &u)
	return u, err
}

// AsInterruptRequest decodes u as InterruptRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsInterruptRequest() (*InterruptRequest, error) {
	return asVariant[InterruptRequest](u.Raw, u.Subtype == "interrupt", "ControlRequestInner", "InterruptRequest")
}

// AsCanUseToolRequest decodes u as CanUseToolRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsCanUseToolRequest() (*CanUseToolRequest, error) {
	return asVariant[CanUseToolRequest](u.Raw, u.Subtype == "can_use_tool", "ControlRequestInner", "CanUseToolRequest")
}

// AsInitializeRequest decodes u as InitializeRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsInitializeRequest() (*InitializeRequest, error) {
	return asVariant[InitializeRequest](u.Raw, u.Subtype == "initialize", "ControlRequestInner", "InitializeRequest")
}

// AsSetPermissionModeRequest decodes u as SetPermissionModeRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsSetPermissionModeRequest() (*SetPermissionModeRequest, error) {
	return asVariant[SetPermissionModeRequest](u.Raw, u.Subtype == "set_permission_mode", "ControlRequestInner", "SetPermissionModeRequest")
}

// AsSetModelRequest decodes u as SetModelRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsSetModelRequest() (*SetModelRequest, error) {
	return asVariant[SetModelRequest](u.Raw, u.Subtype == "set_model", "ControlRequestInner", "SetModelRequest")
}

// AsSetMaxThinkingTokensRequest decodes u as SetMaxThinkingTokensRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsSetMaxThinkingTokensRequest() (*SetMaxThinkingTokensRequest, error) {
	return asVariant[SetMaxThinkingTokensRequest](u.Raw, u.Subtype == "set_max_thinking_tokens", "ControlRequestInner", "SetMaxThinkingTokensRequest")
}

// AsRenameSessionRequest decodes u as RenameSessionRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsRenameSessionRequest() (*RenameSessionRequest, error) {
	return asVariant[RenameSessionRequest](u.Raw, u.Subtype == "rename_session", "ControlRequestInner", "RenameSessionRequest")
}

// AsSetColorRequest decodes u as SetColorRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsSetColorRequest() (*SetColorRequest, error) {
	return asVariant[SetColorRequest](u.Raw, u.Subtype == "set_color", "ControlRequestInner", "SetColorRequest")
}

// AsMcpStatusRequest decodes u as McpStatusRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpStatusRequest() (*McpStatusRequest, error) {
	return asVariant[McpStatusRequest](u.Raw, u.Subtype == "mcp_status", "ControlRequestInner", "McpStatusRequest")
}

// AsGetContextUsageRequest decodes u as GetContextUsageRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetContextUsageRequest() (*GetContextUsageRequest, error) {
	return asVariant[GetContextUsageRequest](u.Raw, u.Subtype == "get_context_usage", "ControlRequestInner", "GetContextUsageRequest")
}

// AsGetSessionCostRequest decodes u as GetSessionCostRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetSessionCostRequest() (*GetSessionCostRequest, error) {
	return asVariant[GetSessionCostRequest](u.Raw, u.Subtype == "get_session_cost", "ControlRequestInner", "GetSessionCostRequest")
}

// AsListModelsRequest decodes u as ListModelsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsListModelsRequest() (*ListModelsRequest, error) {
	return asVariant[ListModelsRequest](u.Raw, u.Subtype == "list_models", "ControlRequestInner", "ListModelsRequest")
}

// AsGetUsageRequest decodes u as GetUsageRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetUsageRequest() (*GetUsageRequest, error) {
	return asVariant[GetUsageRequest](u.Raw, u.Subtype == "get_usage", "ControlRequestInner", "GetUsageRequest")
}

// AsGetBinaryVersionRequest decodes u as GetBinaryVersionRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetBinaryVersionRequest() (*GetBinaryVersionRequest, error) {
	return asVariant[GetBinaryVersionRequest](u.Raw, u.Subtype == "get_binary_version", "ControlRequestInner", "GetBinaryVersionRequest")
}

// AsMcpCallRequest decodes u as McpCallRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpCallRequest() (*McpCallRequest, error) {
	return asVariant[McpCallRequest](u.Raw, u.Subtype == "mcp_call", "ControlRequestInner", "McpCallRequest")
}

// AsFileSuggestionsRequest decodes u as FileSuggestionsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsFileSuggestionsRequest() (*FileSuggestionsRequest, error) {
	return asVariant[FileSuggestionsRequest](u.Raw, u.Subtype == "file_suggestions", "ControlRequestInner", "FileSuggestionsRequest")
}

// AsHookCallbackRequest decodes u as HookCallbackRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsHookCallbackRequest() (*HookCallbackRequest, error) {
	return asVariant[HookCallbackRequest](u.Raw, u.Subtype == "hook_callback", "ControlRequestInner", "HookCallbackRequest")
}

// AsMcpMessageRequest decodes u as McpMessageRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpMessageRequest() (*McpMessageRequest, error) {
	return asVariant[McpMessageRequest](u.Raw, u.Subtype == "mcp_message", "ControlRequestInner", "McpMessageRequest")
}

// AsRewindFilesRequest decodes u as RewindFilesRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsRewindFilesRequest() (*RewindFilesRequest, error) {
	return asVariant[RewindFilesRequest](u.Raw, u.Subtype == "rewind_files", "ControlRequestInner", "RewindFilesRequest")
}

// AsCancelAsyncMessageRequest decodes u as CancelAsyncMessageRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsCancelAsyncMessageRequest() (*CancelAsyncMessageRequest, error) {
	return asVariant[CancelAsyncMessageRequest](u.Raw, u.Subtype == "cancel_async_message", "ControlRequestInner", "CancelAsyncMessageRequest")
}

// AsReadFileRequest decodes u as ReadFileRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsReadFileRequest() (*ReadFileRequest, error) {
	return asVariant[ReadFileRequest](u.Raw, u.Subtype == "read_file", "ControlRequestInner", "ReadFileRequest")
}

// AsSeedReadStateRequest decodes u as SeedReadStateRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsSeedReadStateRequest() (*SeedReadStateRequest, error) {
	return asVariant[SeedReadStateRequest](u.Raw, u.Subtype == "seed_read_state", "ControlRequestInner", "SeedReadStateRequest")
}

// AsMcpSetServersRequest decodes u as McpSetServersRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpSetServersRequest() (*McpSetServersRequest, error) {
	return asVariant[McpSetServersRequest](u.Raw, u.Subtype == "mcp_set_servers", "ControlRequestInner", "McpSetServersRequest")
}

// AsRegisterRepoRootRequest decodes u as RegisterRepoRootRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsRegisterRepoRootRequest() (*RegisterRepoRootRequest, error) {
	return asVariant[RegisterRepoRootRequest](u.Raw, u.Subtype == "register_repo_root", "ControlRequestInner", "RegisterRepoRootRequest")
}

// AsReloadPluginsRequest decodes u as ReloadPluginsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsReloadPluginsRequest() (*ReloadPluginsRequest, error) {
	return asVariant[ReloadPluginsRequest](u.Raw, u.Subtype == "reload_plugins", "ControlRequestInner", "ReloadPluginsRequest")
}

// AsReloadSkillsRequest decodes u as ReloadSkillsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsReloadSkillsRequest() (*ReloadSkillsRequest, error) {
	return asVariant[ReloadSkillsRequest](u.Raw, u.Subtype == "reload_skills", "ControlRequestInner", "ReloadSkillsRequest")
}

// AsReloadOutputStylesRequest decodes u as ReloadOutputStylesRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsReloadOutputStylesRequest() (*ReloadOutputStylesRequest, error) {
	return asVariant[ReloadOutputStylesRequest](u.Raw, u.Subtype == "reload_output_styles", "ControlRequestInner", "ReloadOutputStylesRequest")
}

// AsMcpReconnectRequest decodes u as McpReconnectRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpReconnectRequest() (*McpReconnectRequest, error) {
	return asVariant[McpReconnectRequest](u.Raw, u.Subtype == "mcp_reconnect", "ControlRequestInner", "McpReconnectRequest")
}

// AsMcpToggleRequest decodes u as McpToggleRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpToggleRequest() (*McpToggleRequest, error) {
	return asVariant[McpToggleRequest](u.Raw, u.Subtype == "mcp_toggle", "ControlRequestInner", "McpToggleRequest")
}

// AsStopTaskRequest decodes u as StopTaskRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsStopTaskRequest() (*StopTaskRequest, error) {
	return asVariant[StopTaskRequest](u.Raw, u.Subtype == "stop_task", "ControlRequestInner", "StopTaskRequest")
}

// AsBackgroundTasksRequest decodes u as BackgroundTasksRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsBackgroundTasksRequest() (*BackgroundTasksRequest, error) {
	return asVariant[BackgroundTasksRequest](u.Raw, u.Subtype == "background_tasks", "ControlRequestInner", "BackgroundTasksRequest")
}

// AsGetTaskOutputRequest decodes u as GetTaskOutputRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetTaskOutputRequest() (*GetTaskOutputRequest, error) {
	return asVariant[GetTaskOutputRequest](u.Raw, u.Subtype == "get_task_output", "ControlRequestInner", "GetTaskOutputRequest")
}

// AsApplyFlagSettingsRequest decodes u as ApplyFlagSettingsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsApplyFlagSettingsRequest() (*ApplyFlagSettingsRequest, error) {
	return asVariant[ApplyFlagSettingsRequest](u.Raw, u.Subtype == "apply_flag_settings", "ControlRequestInner", "ApplyFlagSettingsRequest")
}

// AsGetSettingsRequest decodes u as GetSettingsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetSettingsRequest() (*GetSettingsRequest, error) {
	return asVariant[GetSettingsRequest](u.Raw, u.Subtype == "get_settings", "ControlRequestInner", "GetSettingsRequest")
}

// AsGetHooksListingRequest decodes u as GetHooksListingRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsGetHooksListingRequest() (*GetHooksListingRequest, error) {
	return asVariant[GetHooksListingRequest](u.Raw, u.Subtype == "get_hooks_listing", "ControlRequestInner", "GetHooksListingRequest")
}

// AsUpdateSettingsRequest decodes u as UpdateSettingsRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsUpdateSettingsRequest() (*UpdateSettingsRequest, error) {
	return asVariant[UpdateSettingsRequest](u.Raw, u.Subtype == "update_settings", "ControlRequestInner", "UpdateSettingsRequest")
}

// AsElicitationControlRequest decodes u as ElicitationControlRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsElicitationControlRequest() (*ElicitationControlRequest, error) {
	return asVariant[ElicitationControlRequest](u.Raw, u.Subtype == "elicitation", "ControlRequestInner", "ElicitationControlRequest")
}

// AsUserDialogControlRequest decodes u as UserDialogControlRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsUserDialogControlRequest() (*UserDialogControlRequest, error) {
	return asVariant[UserDialogControlRequest](u.Raw, u.Subtype == "request_user_dialog", "ControlRequestInner", "UserDialogControlRequest")
}

// AsListPermissionRulesRequest decodes u as ListPermissionRulesRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsListPermissionRulesRequest() (*ListPermissionRulesRequest, error) {
	return asVariant[ListPermissionRulesRequest](u.Raw, u.Subtype == "list_permission_rules", "ControlRequestInner", "ListPermissionRulesRequest")
}

// AsMcpReadResourceRequest decodes u as McpReadResourceRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsMcpReadResourceRequest() (*McpReadResourceRequest, error) {
	return asVariant[McpReadResourceRequest](u.Raw, u.Subtype == "mcp_read_resource", "ControlRequestInner", "McpReadResourceRequest")
}

// since 0.3.312-0.3.325 (not in 0.3.311)
// AsSetHasSubscribersRequest decodes u as SetHasSubscribersRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlRequestInner) AsSetHasSubscribersRequest() (*SetHasSubscribersRequest, error) {
	return asVariant[SetHasSubscribersRequest](u.Raw, u.Subtype == "set_has_subscribers", "ControlRequestInner", "SetHasSubscribersRequest")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u ControlRequestInner) Decode() (any, error) {
	switch {
	case u.Subtype == "interrupt":
		return u.AsInterruptRequest()
	case u.Subtype == "can_use_tool":
		return u.AsCanUseToolRequest()
	case u.Subtype == "initialize":
		return u.AsInitializeRequest()
	case u.Subtype == "set_permission_mode":
		return u.AsSetPermissionModeRequest()
	case u.Subtype == "set_model":
		return u.AsSetModelRequest()
	case u.Subtype == "set_max_thinking_tokens":
		return u.AsSetMaxThinkingTokensRequest()
	case u.Subtype == "rename_session":
		return u.AsRenameSessionRequest()
	case u.Subtype == "set_color":
		return u.AsSetColorRequest()
	case u.Subtype == "mcp_status":
		return u.AsMcpStatusRequest()
	case u.Subtype == "get_context_usage":
		return u.AsGetContextUsageRequest()
	case u.Subtype == "get_session_cost":
		return u.AsGetSessionCostRequest()
	case u.Subtype == "list_models":
		return u.AsListModelsRequest()
	case u.Subtype == "get_usage":
		return u.AsGetUsageRequest()
	case u.Subtype == "get_binary_version":
		return u.AsGetBinaryVersionRequest()
	case u.Subtype == "mcp_call":
		return u.AsMcpCallRequest()
	case u.Subtype == "file_suggestions":
		return u.AsFileSuggestionsRequest()
	case u.Subtype == "hook_callback":
		return u.AsHookCallbackRequest()
	case u.Subtype == "mcp_message":
		return u.AsMcpMessageRequest()
	case u.Subtype == "rewind_files":
		return u.AsRewindFilesRequest()
	case u.Subtype == "cancel_async_message":
		return u.AsCancelAsyncMessageRequest()
	case u.Subtype == "read_file":
		return u.AsReadFileRequest()
	case u.Subtype == "seed_read_state":
		return u.AsSeedReadStateRequest()
	case u.Subtype == "mcp_set_servers":
		return u.AsMcpSetServersRequest()
	case u.Subtype == "register_repo_root":
		return u.AsRegisterRepoRootRequest()
	case u.Subtype == "reload_plugins":
		return u.AsReloadPluginsRequest()
	case u.Subtype == "reload_skills":
		return u.AsReloadSkillsRequest()
	case u.Subtype == "reload_output_styles":
		return u.AsReloadOutputStylesRequest()
	case u.Subtype == "mcp_reconnect":
		return u.AsMcpReconnectRequest()
	case u.Subtype == "mcp_toggle":
		return u.AsMcpToggleRequest()
	case u.Subtype == "stop_task":
		return u.AsStopTaskRequest()
	case u.Subtype == "background_tasks":
		return u.AsBackgroundTasksRequest()
	case u.Subtype == "get_task_output":
		return u.AsGetTaskOutputRequest()
	case u.Subtype == "apply_flag_settings":
		return u.AsApplyFlagSettingsRequest()
	case u.Subtype == "get_settings":
		return u.AsGetSettingsRequest()
	case u.Subtype == "get_hooks_listing":
		return u.AsGetHooksListingRequest()
	case u.Subtype == "update_settings":
		return u.AsUpdateSettingsRequest()
	case u.Subtype == "elicitation":
		return u.AsElicitationControlRequest()
	case u.Subtype == "request_user_dialog":
		return u.AsUserDialogControlRequest()
	case u.Subtype == "list_permission_rules":
		return u.AsListPermissionRulesRequest()
	case u.Subtype == "mcp_read_resource":
		return u.AsMcpReadResourceRequest()
	case u.Subtype == "set_has_subscribers":
		return u.AsSetHasSubscribersRequest()
	}
	return nil, unknownVariant("ControlRequestInner", u.Raw)
}

// ThinkingConfig is a discriminated union: Controls Claude's thinking/reasoning behavior. (TS: ThinkingConfig).
// Discriminator key(s): "type". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewThinkingConfig.
type ThinkingConfig struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *ThinkingConfig) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type"}, &u.Type)
}
func (u ThinkingConfig) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewThinkingConfig wraps a variant value (or any JSON-marshalable value) as a ThinkingConfig.
func NewThinkingConfig(v any) (ThinkingConfig, error) {
	var u ThinkingConfig
	err := wrapUnion(v, &u)
	return u, err
}

// AsThinkingAdaptive decodes u as ThinkingAdaptive; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ThinkingConfig) AsThinkingAdaptive() (*ThinkingAdaptive, error) {
	return asVariant[ThinkingAdaptive](u.Raw, u.Type == "adaptive", "ThinkingConfig", "ThinkingAdaptive")
}

// AsThinkingEnabled decodes u as ThinkingEnabled; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ThinkingConfig) AsThinkingEnabled() (*ThinkingEnabled, error) {
	return asVariant[ThinkingEnabled](u.Raw, u.Type == "enabled", "ThinkingConfig", "ThinkingEnabled")
}

// AsThinkingDisabled decodes u as ThinkingDisabled; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ThinkingConfig) AsThinkingDisabled() (*ThinkingDisabled, error) {
	return asVariant[ThinkingDisabled](u.Raw, u.Type == "disabled", "ThinkingConfig", "ThinkingDisabled")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u ThinkingConfig) Decode() (any, error) {
	switch {
	case u.Type == "adaptive":
		return u.AsThinkingAdaptive()
	case u.Type == "enabled":
		return u.AsThinkingEnabled()
	case u.Type == "disabled":
		return u.AsThinkingDisabled()
	}
	return nil, unknownVariant("ThinkingConfig", u.Raw)
}

// PermissionRequestHookSpecificOutputDecision is a discriminated union: inline union (field decision of PermissionRequestHookSpecificOutput).
// Discriminator key(s): "behavior". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewPermissionRequestHookSpecificOutputDecision.
type PermissionRequestHookSpecificOutputDecision struct {
	// Behavior is the "behavior" discriminator ("" when absent).
	Behavior string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *PermissionRequestHookSpecificOutputDecision) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"behavior"}, &u.Behavior)
}
func (u PermissionRequestHookSpecificOutputDecision) MarshalJSON() ([]byte, error) {
	return marshalUnion(u.Raw)
}

// NewPermissionRequestHookSpecificOutputDecision wraps a variant value (or any JSON-marshalable value) as a PermissionRequestHookSpecificOutputDecision.
func NewPermissionRequestHookSpecificOutputDecision(v any) (PermissionRequestHookSpecificOutputDecision, error) {
	var u PermissionRequestHookSpecificOutputDecision
	err := wrapUnion(v, &u)
	return u, err
}

// AsPermissionRequestHookSpecificOutputDecisionAllow decodes u as PermissionRequestHookSpecificOutputDecisionAllow; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionRequestHookSpecificOutputDecision) AsPermissionRequestHookSpecificOutputDecisionAllow() (*PermissionRequestHookSpecificOutputDecisionAllow, error) {
	return asVariant[PermissionRequestHookSpecificOutputDecisionAllow](u.Raw, u.Behavior == "allow", "PermissionRequestHookSpecificOutputDecision", "PermissionRequestHookSpecificOutputDecisionAllow")
}

// AsPermissionRequestHookSpecificOutputDecisionDeny decodes u as PermissionRequestHookSpecificOutputDecisionDeny; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionRequestHookSpecificOutputDecision) AsPermissionRequestHookSpecificOutputDecisionDeny() (*PermissionRequestHookSpecificOutputDecisionDeny, error) {
	return asVariant[PermissionRequestHookSpecificOutputDecisionDeny](u.Raw, u.Behavior == "deny", "PermissionRequestHookSpecificOutputDecision", "PermissionRequestHookSpecificOutputDecisionDeny")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u PermissionRequestHookSpecificOutputDecision) Decode() (any, error) {
	switch {
	case u.Behavior == "allow":
		return u.AsPermissionRequestHookSpecificOutputDecisionAllow()
	case u.Behavior == "deny":
		return u.AsPermissionRequestHookSpecificOutputDecisionDeny()
	}
	return nil, unknownVariant("PermissionRequestHookSpecificOutputDecision", u.Raw)
}

// PermissionResult is a discriminated union (TS: PermissionResult).
// Discriminator key(s): "behavior". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewPermissionResult.
type PermissionResult struct {
	// Behavior is the "behavior" discriminator ("" when absent).
	Behavior string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *PermissionResult) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"behavior"}, &u.Behavior)
}
func (u PermissionResult) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewPermissionResult wraps a variant value (or any JSON-marshalable value) as a PermissionResult.
func NewPermissionResult(v any) (PermissionResult, error) {
	var u PermissionResult
	err := wrapUnion(v, &u)
	return u, err
}

// AsPermissionResultAllow decodes u as PermissionResultAllow; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionResult) AsPermissionResultAllow() (*PermissionResultAllow, error) {
	return asVariant[PermissionResultAllow](u.Raw, u.Behavior == "allow", "PermissionResult", "PermissionResultAllow")
}

// AsPermissionResultDeny decodes u as PermissionResultDeny; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u PermissionResult) AsPermissionResultDeny() (*PermissionResultDeny, error) {
	return asVariant[PermissionResultDeny](u.Raw, u.Behavior == "deny", "PermissionResult", "PermissionResultDeny")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u PermissionResult) Decode() (any, error) {
	switch {
	case u.Behavior == "allow":
		return u.AsPermissionResultAllow()
	case u.Behavior == "deny":
		return u.AsPermissionResultDeny()
	}
	return nil, unknownVariant("PermissionResult", u.Raw)
}

// ControlResponseResponse is a discriminated union: inline union (field response of ControlResponse).
// Discriminator key(s): "subtype". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewControlResponseResponse.
type ControlResponseResponse struct {
	// Subtype is the "subtype" discriminator ("" when absent).
	Subtype string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *ControlResponseResponse) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"subtype"}, &u.Subtype)
}
func (u ControlResponseResponse) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewControlResponseResponse wraps a variant value (or any JSON-marshalable value) as a ControlResponseResponse.
func NewControlResponseResponse(v any) (ControlResponseResponse, error) {
	var u ControlResponseResponse
	err := wrapUnion(v, &u)
	return u, err
}

// AsControlSuccessResponse decodes u as ControlSuccessResponse; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlResponseResponse) AsControlSuccessResponse() (*ControlSuccessResponse, error) {
	return asVariant[ControlSuccessResponse](u.Raw, u.Subtype == "success", "ControlResponseResponse", "ControlSuccessResponse")
}

// AsControlErrorResponse decodes u as ControlErrorResponse; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ControlResponseResponse) AsControlErrorResponse() (*ControlErrorResponse, error) {
	return asVariant[ControlErrorResponse](u.Raw, u.Subtype == "error", "ControlResponseResponse", "ControlErrorResponse")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u ControlResponseResponse) Decode() (any, error) {
	switch {
	case u.Subtype == "success":
		return u.AsControlSuccessResponse()
	case u.Subtype == "error":
		return u.AsControlErrorResponse()
	}
	return nil, unknownVariant("ControlResponseResponse", u.Raw)
}

// MessageOrigin is a discriminated union: Provenance of a user-role message (peer session, team lead, channel). (TS: SDKMessageOrigin).
// Discriminator key(s): "kind". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewMessageOrigin.
type MessageOrigin struct {
	// Kind is the "kind" discriminator ("" when absent).
	Kind string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *MessageOrigin) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"kind"}, &u.Kind)
}
func (u MessageOrigin) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewMessageOrigin wraps a variant value (or any JSON-marshalable value) as a MessageOrigin.
func NewMessageOrigin(v any) (MessageOrigin, error) {
	var u MessageOrigin
	err := wrapUnion(v, &u)
	return u, err
}

// AsMessageOriginHuman decodes u as MessageOriginHuman; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginHuman() (*MessageOriginHuman, error) {
	return asVariant[MessageOriginHuman](u.Raw, u.Kind == "human", "MessageOrigin", "MessageOriginHuman")
}

// AsMessageOriginChannel decodes u as MessageOriginChannel; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginChannel() (*MessageOriginChannel, error) {
	return asVariant[MessageOriginChannel](u.Raw, u.Kind == "channel", "MessageOrigin", "MessageOriginChannel")
}

// AsMessageOriginPeer decodes u as MessageOriginPeer; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginPeer() (*MessageOriginPeer, error) {
	return asVariant[MessageOriginPeer](u.Raw, u.Kind == "peer", "MessageOrigin", "MessageOriginPeer")
}

// AsMessageOriginTaskNotification decodes u as MessageOriginTaskNotification; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginTaskNotification() (*MessageOriginTaskNotification, error) {
	return asVariant[MessageOriginTaskNotification](u.Raw, u.Kind == "task-notification", "MessageOrigin", "MessageOriginTaskNotification")
}

// AsMessageOriginCoordinator decodes u as MessageOriginCoordinator; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginCoordinator() (*MessageOriginCoordinator, error) {
	return asVariant[MessageOriginCoordinator](u.Raw, u.Kind == "coordinator", "MessageOrigin", "MessageOriginCoordinator")
}

// AsMessageOriginUnclassified decodes u as MessageOriginUnclassified; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginUnclassified() (*MessageOriginUnclassified, error) {
	return asVariant[MessageOriginUnclassified](u.Raw, u.Kind == "unclassified", "MessageOrigin", "MessageOriginUnclassified")
}

// AsMessageOriginObserver decodes u as MessageOriginObserver; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginObserver() (*MessageOriginObserver, error) {
	return asVariant[MessageOriginObserver](u.Raw, u.Kind == "observer", "MessageOrigin", "MessageOriginObserver")
}

// AsMessageOriginAutoContinuation decodes u as MessageOriginAutoContinuation; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginAutoContinuation() (*MessageOriginAutoContinuation, error) {
	return asVariant[MessageOriginAutoContinuation](u.Raw, u.Kind == "auto-continuation", "MessageOrigin", "MessageOriginAutoContinuation")
}

// AsMessageOriginObserverActivity decodes u as MessageOriginObserverActivity; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u MessageOrigin) AsMessageOriginObserverActivity() (*MessageOriginObserverActivity, error) {
	return asVariant[MessageOriginObserverActivity](u.Raw, u.Kind == "observer-activity", "MessageOrigin", "MessageOriginObserverActivity")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u MessageOrigin) Decode() (any, error) {
	switch {
	case u.Kind == "human":
		return u.AsMessageOriginHuman()
	case u.Kind == "channel":
		return u.AsMessageOriginChannel()
	case u.Kind == "peer":
		return u.AsMessageOriginPeer()
	case u.Kind == "task-notification":
		return u.AsMessageOriginTaskNotification()
	case u.Kind == "coordinator":
		return u.AsMessageOriginCoordinator()
	case u.Kind == "unclassified":
		return u.AsMessageOriginUnclassified()
	case u.Kind == "observer":
		return u.AsMessageOriginObserver()
	case u.Kind == "auto-continuation":
		return u.AsMessageOriginAutoContinuation()
	case u.Kind == "observer-activity":
		return u.AsMessageOriginObserverActivity()
	}
	return nil, unknownVariant("MessageOrigin", u.Raw)
}

// Message is a discriminated union: Every conversational and informational message the CLI emits on its output stream, discriminated by type (and subtype for system/result messages). (TS: SDKMessage).
// Discriminator key(s): "type", "subtype". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewMessage.
type Message struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Subtype is the "subtype" discriminator ("" when absent).
	Subtype string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *Message) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type", "subtype"}, &u.Type, &u.Subtype)
}
func (u Message) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewMessage wraps a variant value (or any JSON-marshalable value) as a Message.
func NewMessage(v any) (Message, error) { var u Message; err := wrapUnion(v, &u); return u, err }

// AsAssistantMessage decodes u as AssistantMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsAssistantMessage() (*AssistantMessage, error) {
	return asVariant[AssistantMessage](u.Raw, u.Type == "assistant" && u.Subtype == "", "Message", "AssistantMessage")
}

// AsUserMessage decodes u as UserMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsUserMessage() (*UserMessage, error) {
	return asVariant[UserMessage](u.Raw, u.Type == "user", "Message", "UserMessage")
}

// AsUserMessageReplay decodes u as UserMessageReplay; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsUserMessageReplay() (*UserMessageReplay, error) {
	return asVariant[UserMessageReplay](u.Raw, u.Type == "user" && peekBool(u.Raw, "isReplay"), "Message", "UserMessageReplay")
}

// AsResultSuccess decodes u as ResultSuccess; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsResultSuccess() (*ResultSuccess, error) {
	return asVariant[ResultSuccess](u.Raw, u.Type == "result" && u.Subtype == "success", "Message", "ResultSuccess")
}

// AsResultError decodes u as ResultError; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsResultError() (*ResultError, error) {
	return asVariant[ResultError](u.Raw, u.Type == "result" && (u.Subtype == "error_during_execution" || u.Subtype == "error_max_turns" || u.Subtype == "error_max_budget_usd" || u.Subtype == "error_max_structured_output_retries"), "Message", "ResultError")
}

// AsSystemInitMessage decodes u as SystemInitMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsSystemInitMessage() (*SystemInitMessage, error) {
	return asVariant[SystemInitMessage](u.Raw, u.Type == "system" && u.Subtype == "init", "Message", "SystemInitMessage")
}

// AsPartialAssistantMessage decodes u as PartialAssistantMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsPartialAssistantMessage() (*PartialAssistantMessage, error) {
	return asVariant[PartialAssistantMessage](u.Raw, u.Type == "stream_event" && u.Subtype == "", "Message", "PartialAssistantMessage")
}

// AsCompactBoundaryMessage decodes u as CompactBoundaryMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsCompactBoundaryMessage() (*CompactBoundaryMessage, error) {
	return asVariant[CompactBoundaryMessage](u.Raw, u.Type == "system" && u.Subtype == "compact_boundary", "Message", "CompactBoundaryMessage")
}

// AsStatusMessage decodes u as StatusMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsStatusMessage() (*StatusMessage, error) {
	return asVariant[StatusMessage](u.Raw, u.Type == "system" && u.Subtype == "status", "Message", "StatusMessage")
}

// AsAPIRetryMessage decodes u as APIRetryMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsAPIRetryMessage() (*APIRetryMessage, error) {
	return asVariant[APIRetryMessage](u.Raw, u.Type == "system" && u.Subtype == "api_retry", "Message", "APIRetryMessage")
}

// AsRequestProgressMessage decodes u as RequestProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsRequestProgressMessage() (*RequestProgressMessage, error) {
	return asVariant[RequestProgressMessage](u.Raw, u.Type == "system" && u.Subtype == "control_request_progress", "Message", "RequestProgressMessage")
}

// AsModelRefusalFallbackMessage decodes u as ModelRefusalFallbackMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsModelRefusalFallbackMessage() (*ModelRefusalFallbackMessage, error) {
	return asVariant[ModelRefusalFallbackMessage](u.Raw, u.Type == "system" && u.Subtype == "model_refusal_fallback", "Message", "ModelRefusalFallbackMessage")
}

// AsModelRefusalNoFallbackMessage decodes u as ModelRefusalNoFallbackMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsModelRefusalNoFallbackMessage() (*ModelRefusalNoFallbackMessage, error) {
	return asVariant[ModelRefusalNoFallbackMessage](u.Raw, u.Type == "system" && u.Subtype == "model_refusal_no_fallback", "Message", "ModelRefusalNoFallbackMessage")
}

// AsLocalCommandOutputMessage decodes u as LocalCommandOutputMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsLocalCommandOutputMessage() (*LocalCommandOutputMessage, error) {
	return asVariant[LocalCommandOutputMessage](u.Raw, u.Type == "system" && u.Subtype == "local_command_output", "Message", "LocalCommandOutputMessage")
}

// AsHookStartedMessage decodes u as HookStartedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsHookStartedMessage() (*HookStartedMessage, error) {
	return asVariant[HookStartedMessage](u.Raw, u.Type == "system" && u.Subtype == "hook_started", "Message", "HookStartedMessage")
}

// AsHookProgressMessage decodes u as HookProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsHookProgressMessage() (*HookProgressMessage, error) {
	return asVariant[HookProgressMessage](u.Raw, u.Type == "system" && u.Subtype == "hook_progress", "Message", "HookProgressMessage")
}

// AsHookResponseMessage decodes u as HookResponseMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsHookResponseMessage() (*HookResponseMessage, error) {
	return asVariant[HookResponseMessage](u.Raw, u.Type == "system" && u.Subtype == "hook_response", "Message", "HookResponseMessage")
}

// AsPluginInstallMessage decodes u as PluginInstallMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsPluginInstallMessage() (*PluginInstallMessage, error) {
	return asVariant[PluginInstallMessage](u.Raw, u.Type == "system" && u.Subtype == "plugin_install", "Message", "PluginInstallMessage")
}

// AsToolProgressMessage decodes u as ToolProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsToolProgressMessage() (*ToolProgressMessage, error) {
	return asVariant[ToolProgressMessage](u.Raw, u.Type == "tool_progress" && u.Subtype == "", "Message", "ToolProgressMessage")
}

// AsAuthStatusMessage decodes u as AuthStatusMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsAuthStatusMessage() (*AuthStatusMessage, error) {
	return asVariant[AuthStatusMessage](u.Raw, u.Type == "auth_status" && u.Subtype == "", "Message", "AuthStatusMessage")
}

// AsTaskNotificationMessage decodes u as TaskNotificationMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsTaskNotificationMessage() (*TaskNotificationMessage, error) {
	return asVariant[TaskNotificationMessage](u.Raw, u.Type == "system" && u.Subtype == "task_notification", "Message", "TaskNotificationMessage")
}

// AsTaskStartedMessage decodes u as TaskStartedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsTaskStartedMessage() (*TaskStartedMessage, error) {
	return asVariant[TaskStartedMessage](u.Raw, u.Type == "system" && u.Subtype == "task_started", "Message", "TaskStartedMessage")
}

// AsTaskUpdatedMessage decodes u as TaskUpdatedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsTaskUpdatedMessage() (*TaskUpdatedMessage, error) {
	return asVariant[TaskUpdatedMessage](u.Raw, u.Type == "system" && u.Subtype == "task_updated", "Message", "TaskUpdatedMessage")
}

// AsTaskProgressMessage decodes u as TaskProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsTaskProgressMessage() (*TaskProgressMessage, error) {
	return asVariant[TaskProgressMessage](u.Raw, u.Type == "system" && u.Subtype == "task_progress", "Message", "TaskProgressMessage")
}

// AsBackgroundTasksChangedMessage decodes u as BackgroundTasksChangedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsBackgroundTasksChangedMessage() (*BackgroundTasksChangedMessage, error) {
	return asVariant[BackgroundTasksChangedMessage](u.Raw, u.Type == "system" && u.Subtype == "background_tasks_changed", "Message", "BackgroundTasksChangedMessage")
}

// AsThinkingTokensMessage decodes u as ThinkingTokensMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsThinkingTokensMessage() (*ThinkingTokensMessage, error) {
	return asVariant[ThinkingTokensMessage](u.Raw, u.Type == "system" && u.Subtype == "thinking_tokens", "Message", "ThinkingTokensMessage")
}

// AsSessionStateChangedMessage decodes u as SessionStateChangedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsSessionStateChangedMessage() (*SessionStateChangedMessage, error) {
	return asVariant[SessionStateChangedMessage](u.Raw, u.Type == "system" && u.Subtype == "session_state_changed", "Message", "SessionStateChangedMessage")
}

// AsWorkerShuttingDownMessage decodes u as WorkerShuttingDownMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsWorkerShuttingDownMessage() (*WorkerShuttingDownMessage, error) {
	return asVariant[WorkerShuttingDownMessage](u.Raw, u.Type == "system" && u.Subtype == "worker_shutting_down", "Message", "WorkerShuttingDownMessage")
}

// AsCommandsChangedMessage decodes u as CommandsChangedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsCommandsChangedMessage() (*CommandsChangedMessage, error) {
	return asVariant[CommandsChangedMessage](u.Raw, u.Type == "system" && u.Subtype == "commands_changed", "Message", "CommandsChangedMessage")
}

// AsNotificationMessage decodes u as NotificationMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsNotificationMessage() (*NotificationMessage, error) {
	return asVariant[NotificationMessage](u.Raw, u.Type == "system" && u.Subtype == "notification", "Message", "NotificationMessage")
}

// AsFilesPersistedEvent decodes u as FilesPersistedEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsFilesPersistedEvent() (*FilesPersistedEvent, error) {
	return asVariant[FilesPersistedEvent](u.Raw, u.Type == "system" && u.Subtype == "files_persisted", "Message", "FilesPersistedEvent")
}

// AsToolUseSummaryMessage decodes u as ToolUseSummaryMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsToolUseSummaryMessage() (*ToolUseSummaryMessage, error) {
	return asVariant[ToolUseSummaryMessage](u.Raw, u.Type == "tool_use_summary" && u.Subtype == "", "Message", "ToolUseSummaryMessage")
}

// AsMemoryRecallMessage decodes u as MemoryRecallMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsMemoryRecallMessage() (*MemoryRecallMessage, error) {
	return asVariant[MemoryRecallMessage](u.Raw, u.Type == "system" && u.Subtype == "memory_recall", "Message", "MemoryRecallMessage")
}

// AsRateLimitEvent decodes u as RateLimitEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsRateLimitEvent() (*RateLimitEvent, error) {
	return asVariant[RateLimitEvent](u.Raw, u.Type == "rate_limit_event" && u.Subtype == "", "Message", "RateLimitEvent")
}

// AsElicitationCompleteMessage decodes u as ElicitationCompleteMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsElicitationCompleteMessage() (*ElicitationCompleteMessage, error) {
	return asVariant[ElicitationCompleteMessage](u.Raw, u.Type == "system" && u.Subtype == "elicitation_complete", "Message", "ElicitationCompleteMessage")
}

// AsPermissionDeniedMessage decodes u as PermissionDeniedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsPermissionDeniedMessage() (*PermissionDeniedMessage, error) {
	return asVariant[PermissionDeniedMessage](u.Raw, u.Type == "system" && u.Subtype == "permission_denied", "Message", "PermissionDeniedMessage")
}

// AsPromptSuggestionMessage decodes u as PromptSuggestionMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsPromptSuggestionMessage() (*PromptSuggestionMessage, error) {
	return asVariant[PromptSuggestionMessage](u.Raw, u.Type == "prompt_suggestion" && u.Subtype == "", "Message", "PromptSuggestionMessage")
}

// AsMirrorErrorMessage decodes u as MirrorErrorMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsMirrorErrorMessage() (*MirrorErrorMessage, error) {
	return asVariant[MirrorErrorMessage](u.Raw, u.Type == "system" && u.Subtype == "mirror_error", "Message", "MirrorErrorMessage")
}

// AsInformationalMessage decodes u as InformationalMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsInformationalMessage() (*InformationalMessage, error) {
	return asVariant[InformationalMessage](u.Raw, u.Type == "system" && u.Subtype == "informational", "Message", "InformationalMessage")
}

// AsConversationResetMessage decodes u as ConversationResetMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsConversationResetMessage() (*ConversationResetMessage, error) {
	return asVariant[ConversationResetMessage](u.Raw, u.Type == "conversation_reset" && u.Subtype == "", "Message", "ConversationResetMessage")
}

// since 0.3.317 (not in 0.3.311)
// AsSessionWritesLostMessage decodes u as SessionWritesLostMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u Message) AsSessionWritesLostMessage() (*SessionWritesLostMessage, error) {
	return asVariant[SessionWritesLostMessage](u.Raw, u.Type == "system" && u.Subtype == "session_writes_lost", "Message", "SessionWritesLostMessage")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u Message) Decode() (any, error) {
	switch {
	case u.Type == "user" && peekBool(u.Raw, "isReplay"):
		return u.AsUserMessageReplay()
	case u.Type == "assistant" && u.Subtype == "":
		return u.AsAssistantMessage()
	case u.Type == "user":
		return u.AsUserMessage()
	case u.Type == "result" && u.Subtype == "success":
		return u.AsResultSuccess()
	case u.Type == "result" && (u.Subtype == "error_during_execution" || u.Subtype == "error_max_turns" || u.Subtype == "error_max_budget_usd" || u.Subtype == "error_max_structured_output_retries"):
		return u.AsResultError()
	case u.Type == "system" && u.Subtype == "init":
		return u.AsSystemInitMessage()
	case u.Type == "stream_event" && u.Subtype == "":
		return u.AsPartialAssistantMessage()
	case u.Type == "system" && u.Subtype == "compact_boundary":
		return u.AsCompactBoundaryMessage()
	case u.Type == "system" && u.Subtype == "status":
		return u.AsStatusMessage()
	case u.Type == "system" && u.Subtype == "api_retry":
		return u.AsAPIRetryMessage()
	case u.Type == "system" && u.Subtype == "control_request_progress":
		return u.AsRequestProgressMessage()
	case u.Type == "system" && u.Subtype == "model_refusal_fallback":
		return u.AsModelRefusalFallbackMessage()
	case u.Type == "system" && u.Subtype == "model_refusal_no_fallback":
		return u.AsModelRefusalNoFallbackMessage()
	case u.Type == "system" && u.Subtype == "local_command_output":
		return u.AsLocalCommandOutputMessage()
	case u.Type == "system" && u.Subtype == "hook_started":
		return u.AsHookStartedMessage()
	case u.Type == "system" && u.Subtype == "hook_progress":
		return u.AsHookProgressMessage()
	case u.Type == "system" && u.Subtype == "hook_response":
		return u.AsHookResponseMessage()
	case u.Type == "system" && u.Subtype == "plugin_install":
		return u.AsPluginInstallMessage()
	case u.Type == "tool_progress" && u.Subtype == "":
		return u.AsToolProgressMessage()
	case u.Type == "auth_status" && u.Subtype == "":
		return u.AsAuthStatusMessage()
	case u.Type == "system" && u.Subtype == "task_notification":
		return u.AsTaskNotificationMessage()
	case u.Type == "system" && u.Subtype == "task_started":
		return u.AsTaskStartedMessage()
	case u.Type == "system" && u.Subtype == "task_updated":
		return u.AsTaskUpdatedMessage()
	case u.Type == "system" && u.Subtype == "task_progress":
		return u.AsTaskProgressMessage()
	case u.Type == "system" && u.Subtype == "background_tasks_changed":
		return u.AsBackgroundTasksChangedMessage()
	case u.Type == "system" && u.Subtype == "thinking_tokens":
		return u.AsThinkingTokensMessage()
	case u.Type == "system" && u.Subtype == "session_state_changed":
		return u.AsSessionStateChangedMessage()
	case u.Type == "system" && u.Subtype == "worker_shutting_down":
		return u.AsWorkerShuttingDownMessage()
	case u.Type == "system" && u.Subtype == "commands_changed":
		return u.AsCommandsChangedMessage()
	case u.Type == "system" && u.Subtype == "notification":
		return u.AsNotificationMessage()
	case u.Type == "system" && u.Subtype == "files_persisted":
		return u.AsFilesPersistedEvent()
	case u.Type == "tool_use_summary" && u.Subtype == "":
		return u.AsToolUseSummaryMessage()
	case u.Type == "system" && u.Subtype == "memory_recall":
		return u.AsMemoryRecallMessage()
	case u.Type == "rate_limit_event" && u.Subtype == "":
		return u.AsRateLimitEvent()
	case u.Type == "system" && u.Subtype == "elicitation_complete":
		return u.AsElicitationCompleteMessage()
	case u.Type == "system" && u.Subtype == "permission_denied":
		return u.AsPermissionDeniedMessage()
	case u.Type == "prompt_suggestion" && u.Subtype == "":
		return u.AsPromptSuggestionMessage()
	case u.Type == "system" && u.Subtype == "mirror_error":
		return u.AsMirrorErrorMessage()
	case u.Type == "system" && u.Subtype == "informational":
		return u.AsInformationalMessage()
	case u.Type == "conversation_reset" && u.Subtype == "":
		return u.AsConversationResetMessage()
	case u.Type == "system" && u.Subtype == "session_writes_lost":
		return u.AsSessionWritesLostMessage()
	}
	return nil, unknownVariant("Message", u.Raw)
}

// ResultMessage is a discriminated union: The outcome of a turn. (TS: SDKResultMessage).
// Discriminator key(s): "type", "subtype". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewResultMessage.
type ResultMessage struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Subtype is the "subtype" discriminator ("" when absent).
	Subtype string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *ResultMessage) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type", "subtype"}, &u.Type, &u.Subtype)
}
func (u ResultMessage) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewResultMessage wraps a variant value (or any JSON-marshalable value) as a ResultMessage.
func NewResultMessage(v any) (ResultMessage, error) {
	var u ResultMessage
	err := wrapUnion(v, &u)
	return u, err
}

// AsResultSuccess decodes u as ResultSuccess; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ResultMessage) AsResultSuccess() (*ResultSuccess, error) {
	return asVariant[ResultSuccess](u.Raw, u.Type == "result" && u.Subtype == "success", "ResultMessage", "ResultSuccess")
}

// AsResultError decodes u as ResultError; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u ResultMessage) AsResultError() (*ResultError, error) {
	return asVariant[ResultError](u.Raw, u.Type == "result" && (u.Subtype == "error_during_execution" || u.Subtype == "error_max_turns" || u.Subtype == "error_max_budget_usd" || u.Subtype == "error_max_structured_output_retries"), "ResultMessage", "ResultError")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u ResultMessage) Decode() (any, error) {
	switch {
	case u.Type == "result" && u.Subtype == "success":
		return u.AsResultSuccess()
	case u.Type == "result" && (u.Subtype == "error_during_execution" || u.Subtype == "error_max_turns" || u.Subtype == "error_max_budget_usd" || u.Subtype == "error_max_structured_output_retries"):
		return u.AsResultError()
	}
	return nil, unknownVariant("ResultMessage", u.Raw)
}

// StdoutMessage is a discriminated union: Everything the CLI writes to its output stream (stdout in stream-json mode): exactly one StdoutMessage per line, as a single JSON object. (TS: StdoutMessage, not exported (wire type)).
// Discriminator key(s): "type", "subtype". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewStdoutMessage.
type StdoutMessage struct {
	// Type is the "type" discriminator ("" when absent).
	Type string `json:"-"`
	// Subtype is the "subtype" discriminator ("" when absent).
	Subtype string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *StdoutMessage) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"type", "subtype"}, &u.Type, &u.Subtype)
}
func (u StdoutMessage) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewStdoutMessage wraps a variant value (or any JSON-marshalable value) as a StdoutMessage.
func NewStdoutMessage(v any) (StdoutMessage, error) {
	var u StdoutMessage
	err := wrapUnion(v, &u)
	return u, err
}

// AsAssistantMessage decodes u as AssistantMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsAssistantMessage() (*AssistantMessage, error) {
	return asVariant[AssistantMessage](u.Raw, u.Type == "assistant" && u.Subtype == "", "StdoutMessage", "AssistantMessage")
}

// AsUserMessage decodes u as UserMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsUserMessage() (*UserMessage, error) {
	return asVariant[UserMessage](u.Raw, u.Type == "user", "StdoutMessage", "UserMessage")
}

// AsUserMessageReplay decodes u as UserMessageReplay; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsUserMessageReplay() (*UserMessageReplay, error) {
	return asVariant[UserMessageReplay](u.Raw, u.Type == "user" && peekBool(u.Raw, "isReplay"), "StdoutMessage", "UserMessageReplay")
}

// AsResultSuccess decodes u as ResultSuccess; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsResultSuccess() (*ResultSuccess, error) {
	return asVariant[ResultSuccess](u.Raw, u.Type == "result" && u.Subtype == "success", "StdoutMessage", "ResultSuccess")
}

// AsResultError decodes u as ResultError; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsResultError() (*ResultError, error) {
	return asVariant[ResultError](u.Raw, u.Type == "result" && (u.Subtype == "error_during_execution" || u.Subtype == "error_max_turns" || u.Subtype == "error_max_budget_usd" || u.Subtype == "error_max_structured_output_retries"), "StdoutMessage", "ResultError")
}

// AsSystemInitMessage decodes u as SystemInitMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsSystemInitMessage() (*SystemInitMessage, error) {
	return asVariant[SystemInitMessage](u.Raw, u.Type == "system" && u.Subtype == "init", "StdoutMessage", "SystemInitMessage")
}

// AsPartialAssistantMessage decodes u as PartialAssistantMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsPartialAssistantMessage() (*PartialAssistantMessage, error) {
	return asVariant[PartialAssistantMessage](u.Raw, u.Type == "stream_event" && u.Subtype == "", "StdoutMessage", "PartialAssistantMessage")
}

// AsCompactBoundaryMessage decodes u as CompactBoundaryMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsCompactBoundaryMessage() (*CompactBoundaryMessage, error) {
	return asVariant[CompactBoundaryMessage](u.Raw, u.Type == "system" && u.Subtype == "compact_boundary", "StdoutMessage", "CompactBoundaryMessage")
}

// AsStatusMessage decodes u as StatusMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsStatusMessage() (*StatusMessage, error) {
	return asVariant[StatusMessage](u.Raw, u.Type == "system" && u.Subtype == "status", "StdoutMessage", "StatusMessage")
}

// AsAPIRetryMessage decodes u as APIRetryMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsAPIRetryMessage() (*APIRetryMessage, error) {
	return asVariant[APIRetryMessage](u.Raw, u.Type == "system" && u.Subtype == "api_retry", "StdoutMessage", "APIRetryMessage")
}

// AsRequestProgressMessage decodes u as RequestProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsRequestProgressMessage() (*RequestProgressMessage, error) {
	return asVariant[RequestProgressMessage](u.Raw, u.Type == "system" && u.Subtype == "control_request_progress", "StdoutMessage", "RequestProgressMessage")
}

// AsModelRefusalFallbackMessage decodes u as ModelRefusalFallbackMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsModelRefusalFallbackMessage() (*ModelRefusalFallbackMessage, error) {
	return asVariant[ModelRefusalFallbackMessage](u.Raw, u.Type == "system" && u.Subtype == "model_refusal_fallback", "StdoutMessage", "ModelRefusalFallbackMessage")
}

// AsModelRefusalNoFallbackMessage decodes u as ModelRefusalNoFallbackMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsModelRefusalNoFallbackMessage() (*ModelRefusalNoFallbackMessage, error) {
	return asVariant[ModelRefusalNoFallbackMessage](u.Raw, u.Type == "system" && u.Subtype == "model_refusal_no_fallback", "StdoutMessage", "ModelRefusalNoFallbackMessage")
}

// AsLocalCommandOutputMessage decodes u as LocalCommandOutputMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsLocalCommandOutputMessage() (*LocalCommandOutputMessage, error) {
	return asVariant[LocalCommandOutputMessage](u.Raw, u.Type == "system" && u.Subtype == "local_command_output", "StdoutMessage", "LocalCommandOutputMessage")
}

// AsHookStartedMessage decodes u as HookStartedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsHookStartedMessage() (*HookStartedMessage, error) {
	return asVariant[HookStartedMessage](u.Raw, u.Type == "system" && u.Subtype == "hook_started", "StdoutMessage", "HookStartedMessage")
}

// AsHookProgressMessage decodes u as HookProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsHookProgressMessage() (*HookProgressMessage, error) {
	return asVariant[HookProgressMessage](u.Raw, u.Type == "system" && u.Subtype == "hook_progress", "StdoutMessage", "HookProgressMessage")
}

// AsHookResponseMessage decodes u as HookResponseMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsHookResponseMessage() (*HookResponseMessage, error) {
	return asVariant[HookResponseMessage](u.Raw, u.Type == "system" && u.Subtype == "hook_response", "StdoutMessage", "HookResponseMessage")
}

// AsPluginInstallMessage decodes u as PluginInstallMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsPluginInstallMessage() (*PluginInstallMessage, error) {
	return asVariant[PluginInstallMessage](u.Raw, u.Type == "system" && u.Subtype == "plugin_install", "StdoutMessage", "PluginInstallMessage")
}

// AsToolProgressMessage decodes u as ToolProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsToolProgressMessage() (*ToolProgressMessage, error) {
	return asVariant[ToolProgressMessage](u.Raw, u.Type == "tool_progress" && u.Subtype == "", "StdoutMessage", "ToolProgressMessage")
}

// AsAuthStatusMessage decodes u as AuthStatusMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsAuthStatusMessage() (*AuthStatusMessage, error) {
	return asVariant[AuthStatusMessage](u.Raw, u.Type == "auth_status" && u.Subtype == "", "StdoutMessage", "AuthStatusMessage")
}

// AsTaskNotificationMessage decodes u as TaskNotificationMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsTaskNotificationMessage() (*TaskNotificationMessage, error) {
	return asVariant[TaskNotificationMessage](u.Raw, u.Type == "system" && u.Subtype == "task_notification", "StdoutMessage", "TaskNotificationMessage")
}

// AsTaskStartedMessage decodes u as TaskStartedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsTaskStartedMessage() (*TaskStartedMessage, error) {
	return asVariant[TaskStartedMessage](u.Raw, u.Type == "system" && u.Subtype == "task_started", "StdoutMessage", "TaskStartedMessage")
}

// AsTaskUpdatedMessage decodes u as TaskUpdatedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsTaskUpdatedMessage() (*TaskUpdatedMessage, error) {
	return asVariant[TaskUpdatedMessage](u.Raw, u.Type == "system" && u.Subtype == "task_updated", "StdoutMessage", "TaskUpdatedMessage")
}

// AsTaskProgressMessage decodes u as TaskProgressMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsTaskProgressMessage() (*TaskProgressMessage, error) {
	return asVariant[TaskProgressMessage](u.Raw, u.Type == "system" && u.Subtype == "task_progress", "StdoutMessage", "TaskProgressMessage")
}

// AsBackgroundTasksChangedMessage decodes u as BackgroundTasksChangedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsBackgroundTasksChangedMessage() (*BackgroundTasksChangedMessage, error) {
	return asVariant[BackgroundTasksChangedMessage](u.Raw, u.Type == "system" && u.Subtype == "background_tasks_changed", "StdoutMessage", "BackgroundTasksChangedMessage")
}

// AsThinkingTokensMessage decodes u as ThinkingTokensMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsThinkingTokensMessage() (*ThinkingTokensMessage, error) {
	return asVariant[ThinkingTokensMessage](u.Raw, u.Type == "system" && u.Subtype == "thinking_tokens", "StdoutMessage", "ThinkingTokensMessage")
}

// AsSessionStateChangedMessage decodes u as SessionStateChangedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsSessionStateChangedMessage() (*SessionStateChangedMessage, error) {
	return asVariant[SessionStateChangedMessage](u.Raw, u.Type == "system" && u.Subtype == "session_state_changed", "StdoutMessage", "SessionStateChangedMessage")
}

// AsWorkerShuttingDownMessage decodes u as WorkerShuttingDownMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsWorkerShuttingDownMessage() (*WorkerShuttingDownMessage, error) {
	return asVariant[WorkerShuttingDownMessage](u.Raw, u.Type == "system" && u.Subtype == "worker_shutting_down", "StdoutMessage", "WorkerShuttingDownMessage")
}

// AsCommandsChangedMessage decodes u as CommandsChangedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsCommandsChangedMessage() (*CommandsChangedMessage, error) {
	return asVariant[CommandsChangedMessage](u.Raw, u.Type == "system" && u.Subtype == "commands_changed", "StdoutMessage", "CommandsChangedMessage")
}

// AsNotificationMessage decodes u as NotificationMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsNotificationMessage() (*NotificationMessage, error) {
	return asVariant[NotificationMessage](u.Raw, u.Type == "system" && u.Subtype == "notification", "StdoutMessage", "NotificationMessage")
}

// AsFilesPersistedEvent decodes u as FilesPersistedEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsFilesPersistedEvent() (*FilesPersistedEvent, error) {
	return asVariant[FilesPersistedEvent](u.Raw, u.Type == "system" && u.Subtype == "files_persisted", "StdoutMessage", "FilesPersistedEvent")
}

// AsToolUseSummaryMessage decodes u as ToolUseSummaryMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsToolUseSummaryMessage() (*ToolUseSummaryMessage, error) {
	return asVariant[ToolUseSummaryMessage](u.Raw, u.Type == "tool_use_summary" && u.Subtype == "", "StdoutMessage", "ToolUseSummaryMessage")
}

// AsMemoryRecallMessage decodes u as MemoryRecallMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsMemoryRecallMessage() (*MemoryRecallMessage, error) {
	return asVariant[MemoryRecallMessage](u.Raw, u.Type == "system" && u.Subtype == "memory_recall", "StdoutMessage", "MemoryRecallMessage")
}

// AsRateLimitEvent decodes u as RateLimitEvent; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsRateLimitEvent() (*RateLimitEvent, error) {
	return asVariant[RateLimitEvent](u.Raw, u.Type == "rate_limit_event" && u.Subtype == "", "StdoutMessage", "RateLimitEvent")
}

// AsElicitationCompleteMessage decodes u as ElicitationCompleteMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsElicitationCompleteMessage() (*ElicitationCompleteMessage, error) {
	return asVariant[ElicitationCompleteMessage](u.Raw, u.Type == "system" && u.Subtype == "elicitation_complete", "StdoutMessage", "ElicitationCompleteMessage")
}

// AsPermissionDeniedMessage decodes u as PermissionDeniedMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsPermissionDeniedMessage() (*PermissionDeniedMessage, error) {
	return asVariant[PermissionDeniedMessage](u.Raw, u.Type == "system" && u.Subtype == "permission_denied", "StdoutMessage", "PermissionDeniedMessage")
}

// AsPromptSuggestionMessage decodes u as PromptSuggestionMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsPromptSuggestionMessage() (*PromptSuggestionMessage, error) {
	return asVariant[PromptSuggestionMessage](u.Raw, u.Type == "prompt_suggestion" && u.Subtype == "", "StdoutMessage", "PromptSuggestionMessage")
}

// AsMirrorErrorMessage decodes u as MirrorErrorMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsMirrorErrorMessage() (*MirrorErrorMessage, error) {
	return asVariant[MirrorErrorMessage](u.Raw, u.Type == "system" && u.Subtype == "mirror_error", "StdoutMessage", "MirrorErrorMessage")
}

// AsInformationalMessage decodes u as InformationalMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsInformationalMessage() (*InformationalMessage, error) {
	return asVariant[InformationalMessage](u.Raw, u.Type == "system" && u.Subtype == "informational", "StdoutMessage", "InformationalMessage")
}

// AsConversationResetMessage decodes u as ConversationResetMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsConversationResetMessage() (*ConversationResetMessage, error) {
	return asVariant[ConversationResetMessage](u.Raw, u.Type == "conversation_reset" && u.Subtype == "", "StdoutMessage", "ConversationResetMessage")
}

// AsActiveGoalMessage decodes u as ActiveGoalMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsActiveGoalMessage() (*ActiveGoalMessage, error) {
	return asVariant[ActiveGoalMessage](u.Raw, u.Type == "active_goal" && u.Subtype == "", "StdoutMessage", "ActiveGoalMessage")
}

// AsControlResponse decodes u as ControlResponse; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsControlResponse() (*ControlResponse, error) {
	return asVariant[ControlResponse](u.Raw, u.Type == "control_response" && u.Subtype == "", "StdoutMessage", "ControlResponse")
}

// AsControlRequest decodes u as ControlRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsControlRequest() (*ControlRequest, error) {
	return asVariant[ControlRequest](u.Raw, u.Type == "control_request" && u.Subtype == "", "StdoutMessage", "ControlRequest")
}

// AsControlCancelRequest decodes u as ControlCancelRequest; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsControlCancelRequest() (*ControlCancelRequest, error) {
	return asVariant[ControlCancelRequest](u.Raw, u.Type == "control_cancel_request" && u.Subtype == "", "StdoutMessage", "ControlCancelRequest")
}

// AsKeepAliveMessage decodes u as KeepAliveMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsKeepAliveMessage() (*KeepAliveMessage, error) {
	return asVariant[KeepAliveMessage](u.Raw, u.Type == "keep_alive" && u.Subtype == "", "StdoutMessage", "KeepAliveMessage")
}

// since 0.3.317 (not in 0.3.311)
// AsSessionWritesLostMessage decodes u as SessionWritesLostMessage; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u StdoutMessage) AsSessionWritesLostMessage() (*SessionWritesLostMessage, error) {
	return asVariant[SessionWritesLostMessage](u.Raw, u.Type == "system" && u.Subtype == "session_writes_lost", "StdoutMessage", "SessionWritesLostMessage")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u StdoutMessage) Decode() (any, error) {
	switch {
	case u.Type == "user" && peekBool(u.Raw, "isReplay"):
		return u.AsUserMessageReplay()
	case u.Type == "assistant" && u.Subtype == "":
		return u.AsAssistantMessage()
	case u.Type == "user":
		return u.AsUserMessage()
	case u.Type == "result" && u.Subtype == "success":
		return u.AsResultSuccess()
	case u.Type == "result" && (u.Subtype == "error_during_execution" || u.Subtype == "error_max_turns" || u.Subtype == "error_max_budget_usd" || u.Subtype == "error_max_structured_output_retries"):
		return u.AsResultError()
	case u.Type == "system" && u.Subtype == "init":
		return u.AsSystemInitMessage()
	case u.Type == "stream_event" && u.Subtype == "":
		return u.AsPartialAssistantMessage()
	case u.Type == "system" && u.Subtype == "compact_boundary":
		return u.AsCompactBoundaryMessage()
	case u.Type == "system" && u.Subtype == "status":
		return u.AsStatusMessage()
	case u.Type == "system" && u.Subtype == "api_retry":
		return u.AsAPIRetryMessage()
	case u.Type == "system" && u.Subtype == "control_request_progress":
		return u.AsRequestProgressMessage()
	case u.Type == "system" && u.Subtype == "model_refusal_fallback":
		return u.AsModelRefusalFallbackMessage()
	case u.Type == "system" && u.Subtype == "model_refusal_no_fallback":
		return u.AsModelRefusalNoFallbackMessage()
	case u.Type == "system" && u.Subtype == "local_command_output":
		return u.AsLocalCommandOutputMessage()
	case u.Type == "system" && u.Subtype == "hook_started":
		return u.AsHookStartedMessage()
	case u.Type == "system" && u.Subtype == "hook_progress":
		return u.AsHookProgressMessage()
	case u.Type == "system" && u.Subtype == "hook_response":
		return u.AsHookResponseMessage()
	case u.Type == "system" && u.Subtype == "plugin_install":
		return u.AsPluginInstallMessage()
	case u.Type == "tool_progress" && u.Subtype == "":
		return u.AsToolProgressMessage()
	case u.Type == "auth_status" && u.Subtype == "":
		return u.AsAuthStatusMessage()
	case u.Type == "system" && u.Subtype == "task_notification":
		return u.AsTaskNotificationMessage()
	case u.Type == "system" && u.Subtype == "task_started":
		return u.AsTaskStartedMessage()
	case u.Type == "system" && u.Subtype == "task_updated":
		return u.AsTaskUpdatedMessage()
	case u.Type == "system" && u.Subtype == "task_progress":
		return u.AsTaskProgressMessage()
	case u.Type == "system" && u.Subtype == "background_tasks_changed":
		return u.AsBackgroundTasksChangedMessage()
	case u.Type == "system" && u.Subtype == "thinking_tokens":
		return u.AsThinkingTokensMessage()
	case u.Type == "system" && u.Subtype == "session_state_changed":
		return u.AsSessionStateChangedMessage()
	case u.Type == "system" && u.Subtype == "worker_shutting_down":
		return u.AsWorkerShuttingDownMessage()
	case u.Type == "system" && u.Subtype == "commands_changed":
		return u.AsCommandsChangedMessage()
	case u.Type == "system" && u.Subtype == "notification":
		return u.AsNotificationMessage()
	case u.Type == "system" && u.Subtype == "files_persisted":
		return u.AsFilesPersistedEvent()
	case u.Type == "tool_use_summary" && u.Subtype == "":
		return u.AsToolUseSummaryMessage()
	case u.Type == "system" && u.Subtype == "memory_recall":
		return u.AsMemoryRecallMessage()
	case u.Type == "rate_limit_event" && u.Subtype == "":
		return u.AsRateLimitEvent()
	case u.Type == "system" && u.Subtype == "elicitation_complete":
		return u.AsElicitationCompleteMessage()
	case u.Type == "system" && u.Subtype == "permission_denied":
		return u.AsPermissionDeniedMessage()
	case u.Type == "prompt_suggestion" && u.Subtype == "":
		return u.AsPromptSuggestionMessage()
	case u.Type == "system" && u.Subtype == "mirror_error":
		return u.AsMirrorErrorMessage()
	case u.Type == "system" && u.Subtype == "informational":
		return u.AsInformationalMessage()
	case u.Type == "conversation_reset" && u.Subtype == "":
		return u.AsConversationResetMessage()
	case u.Type == "active_goal" && u.Subtype == "":
		return u.AsActiveGoalMessage()
	case u.Type == "control_response" && u.Subtype == "":
		return u.AsControlResponse()
	case u.Type == "control_request" && u.Subtype == "":
		return u.AsControlRequest()
	case u.Type == "control_cancel_request" && u.Subtype == "":
		return u.AsControlCancelRequest()
	case u.Type == "keep_alive" && u.Subtype == "":
		return u.AsKeepAliveMessage()
	case u.Type == "system" && u.Subtype == "session_writes_lost":
		return u.AsSessionWritesLostMessage()
	}
	return nil, unknownVariant("StdoutMessage", u.Raw)
}

// SyncHookJSONOutputHookSpecificOutput is a discriminated union: inline union (field hookSpecificOutput of SyncHookJSONOutput).
// Discriminator key(s): "hookEventName". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewSyncHookJSONOutputHookSpecificOutput.
type SyncHookJSONOutputHookSpecificOutput struct {
	// HookEventName is the "hookEventName" discriminator ("" when absent).
	HookEventName string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *SyncHookJSONOutputHookSpecificOutput) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"hookEventName"}, &u.HookEventName)
}
func (u SyncHookJSONOutputHookSpecificOutput) MarshalJSON() ([]byte, error) {
	return marshalUnion(u.Raw)
}

// NewSyncHookJSONOutputHookSpecificOutput wraps a variant value (or any JSON-marshalable value) as a SyncHookJSONOutputHookSpecificOutput.
func NewSyncHookJSONOutputHookSpecificOutput(v any) (SyncHookJSONOutputHookSpecificOutput, error) {
	var u SyncHookJSONOutputHookSpecificOutput
	err := wrapUnion(v, &u)
	return u, err
}

// AsPreToolUseHookSpecificOutput decodes u as PreToolUseHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPreToolUseHookSpecificOutput() (*PreToolUseHookSpecificOutput, error) {
	return asVariant[PreToolUseHookSpecificOutput](u.Raw, u.HookEventName == "PreToolUse", "SyncHookJSONOutputHookSpecificOutput", "PreToolUseHookSpecificOutput")
}

// AsUserPromptSubmitHookSpecificOutput decodes u as UserPromptSubmitHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsUserPromptSubmitHookSpecificOutput() (*UserPromptSubmitHookSpecificOutput, error) {
	return asVariant[UserPromptSubmitHookSpecificOutput](u.Raw, u.HookEventName == "UserPromptSubmit", "SyncHookJSONOutputHookSpecificOutput", "UserPromptSubmitHookSpecificOutput")
}

// AsUserPromptExpansionHookSpecificOutput decodes u as UserPromptExpansionHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsUserPromptExpansionHookSpecificOutput() (*UserPromptExpansionHookSpecificOutput, error) {
	return asVariant[UserPromptExpansionHookSpecificOutput](u.Raw, u.HookEventName == "UserPromptExpansion", "SyncHookJSONOutputHookSpecificOutput", "UserPromptExpansionHookSpecificOutput")
}

// AsSessionStartHookSpecificOutput decodes u as SessionStartHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsSessionStartHookSpecificOutput() (*SessionStartHookSpecificOutput, error) {
	return asVariant[SessionStartHookSpecificOutput](u.Raw, u.HookEventName == "SessionStart", "SyncHookJSONOutputHookSpecificOutput", "SessionStartHookSpecificOutput")
}

// AsSetupHookSpecificOutput decodes u as SetupHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsSetupHookSpecificOutput() (*SetupHookSpecificOutput, error) {
	return asVariant[SetupHookSpecificOutput](u.Raw, u.HookEventName == "Setup", "SyncHookJSONOutputHookSpecificOutput", "SetupHookSpecificOutput")
}

// AsPreModelSwitchHookSpecificOutput decodes u as PreModelSwitchHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPreModelSwitchHookSpecificOutput() (*PreModelSwitchHookSpecificOutput, error) {
	return asVariant[PreModelSwitchHookSpecificOutput](u.Raw, u.HookEventName == "PreModelSwitch", "SyncHookJSONOutputHookSpecificOutput", "PreModelSwitchHookSpecificOutput")
}

// AsPostModelSwitchHookSpecificOutput decodes u as PostModelSwitchHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPostModelSwitchHookSpecificOutput() (*PostModelSwitchHookSpecificOutput, error) {
	return asVariant[PostModelSwitchHookSpecificOutput](u.Raw, u.HookEventName == "PostModelSwitch", "SyncHookJSONOutputHookSpecificOutput", "PostModelSwitchHookSpecificOutput")
}

// AsSubagentStartHookSpecificOutput decodes u as SubagentStartHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsSubagentStartHookSpecificOutput() (*SubagentStartHookSpecificOutput, error) {
	return asVariant[SubagentStartHookSpecificOutput](u.Raw, u.HookEventName == "SubagentStart", "SyncHookJSONOutputHookSpecificOutput", "SubagentStartHookSpecificOutput")
}

// AsPostToolUseHookSpecificOutput decodes u as PostToolUseHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPostToolUseHookSpecificOutput() (*PostToolUseHookSpecificOutput, error) {
	return asVariant[PostToolUseHookSpecificOutput](u.Raw, u.HookEventName == "PostToolUse", "SyncHookJSONOutputHookSpecificOutput", "PostToolUseHookSpecificOutput")
}

// AsPostToolUseFailureHookSpecificOutput decodes u as PostToolUseFailureHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPostToolUseFailureHookSpecificOutput() (*PostToolUseFailureHookSpecificOutput, error) {
	return asVariant[PostToolUseFailureHookSpecificOutput](u.Raw, u.HookEventName == "PostToolUseFailure", "SyncHookJSONOutputHookSpecificOutput", "PostToolUseFailureHookSpecificOutput")
}

// AsPostToolBatchHookSpecificOutput decodes u as PostToolBatchHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPostToolBatchHookSpecificOutput() (*PostToolBatchHookSpecificOutput, error) {
	return asVariant[PostToolBatchHookSpecificOutput](u.Raw, u.HookEventName == "PostToolBatch", "SyncHookJSONOutputHookSpecificOutput", "PostToolBatchHookSpecificOutput")
}

// AsStopHookSpecificOutput decodes u as StopHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsStopHookSpecificOutput() (*StopHookSpecificOutput, error) {
	return asVariant[StopHookSpecificOutput](u.Raw, u.HookEventName == "Stop", "SyncHookJSONOutputHookSpecificOutput", "StopHookSpecificOutput")
}

// AsSubagentStopHookSpecificOutput decodes u as SubagentStopHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsSubagentStopHookSpecificOutput() (*SubagentStopHookSpecificOutput, error) {
	return asVariant[SubagentStopHookSpecificOutput](u.Raw, u.HookEventName == "SubagentStop", "SyncHookJSONOutputHookSpecificOutput", "SubagentStopHookSpecificOutput")
}

// AsPermissionDeniedHookSpecificOutput decodes u as PermissionDeniedHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPermissionDeniedHookSpecificOutput() (*PermissionDeniedHookSpecificOutput, error) {
	return asVariant[PermissionDeniedHookSpecificOutput](u.Raw, u.HookEventName == "PermissionDenied", "SyncHookJSONOutputHookSpecificOutput", "PermissionDeniedHookSpecificOutput")
}

// AsNotificationHookSpecificOutput decodes u as NotificationHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsNotificationHookSpecificOutput() (*NotificationHookSpecificOutput, error) {
	return asVariant[NotificationHookSpecificOutput](u.Raw, u.HookEventName == "Notification", "SyncHookJSONOutputHookSpecificOutput", "NotificationHookSpecificOutput")
}

// AsPermissionRequestHookSpecificOutput decodes u as PermissionRequestHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsPermissionRequestHookSpecificOutput() (*PermissionRequestHookSpecificOutput, error) {
	return asVariant[PermissionRequestHookSpecificOutput](u.Raw, u.HookEventName == "PermissionRequest", "SyncHookJSONOutputHookSpecificOutput", "PermissionRequestHookSpecificOutput")
}

// AsElicitationHookSpecificOutput decodes u as ElicitationHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsElicitationHookSpecificOutput() (*ElicitationHookSpecificOutput, error) {
	return asVariant[ElicitationHookSpecificOutput](u.Raw, u.HookEventName == "Elicitation", "SyncHookJSONOutputHookSpecificOutput", "ElicitationHookSpecificOutput")
}

// AsElicitationResultHookSpecificOutput decodes u as ElicitationResultHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsElicitationResultHookSpecificOutput() (*ElicitationResultHookSpecificOutput, error) {
	return asVariant[ElicitationResultHookSpecificOutput](u.Raw, u.HookEventName == "ElicitationResult", "SyncHookJSONOutputHookSpecificOutput", "ElicitationResultHookSpecificOutput")
}

// AsCwdChangedHookSpecificOutput decodes u as CwdChangedHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsCwdChangedHookSpecificOutput() (*CwdChangedHookSpecificOutput, error) {
	return asVariant[CwdChangedHookSpecificOutput](u.Raw, u.HookEventName == "CwdChanged", "SyncHookJSONOutputHookSpecificOutput", "CwdChangedHookSpecificOutput")
}

// AsFileChangedHookSpecificOutput decodes u as FileChangedHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsFileChangedHookSpecificOutput() (*FileChangedHookSpecificOutput, error) {
	return asVariant[FileChangedHookSpecificOutput](u.Raw, u.HookEventName == "FileChanged", "SyncHookJSONOutputHookSpecificOutput", "FileChangedHookSpecificOutput")
}

// AsWorktreeCreateHookSpecificOutput decodes u as WorktreeCreateHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsWorktreeCreateHookSpecificOutput() (*WorktreeCreateHookSpecificOutput, error) {
	return asVariant[WorktreeCreateHookSpecificOutput](u.Raw, u.HookEventName == "WorktreeCreate", "SyncHookJSONOutputHookSpecificOutput", "WorktreeCreateHookSpecificOutput")
}

// AsMessageDisplayHookSpecificOutput decodes u as MessageDisplayHookSpecificOutput; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u SyncHookJSONOutputHookSpecificOutput) AsMessageDisplayHookSpecificOutput() (*MessageDisplayHookSpecificOutput, error) {
	return asVariant[MessageDisplayHookSpecificOutput](u.Raw, u.HookEventName == "MessageDisplay", "SyncHookJSONOutputHookSpecificOutput", "MessageDisplayHookSpecificOutput")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u SyncHookJSONOutputHookSpecificOutput) Decode() (any, error) {
	switch {
	case u.HookEventName == "PreToolUse":
		return u.AsPreToolUseHookSpecificOutput()
	case u.HookEventName == "UserPromptSubmit":
		return u.AsUserPromptSubmitHookSpecificOutput()
	case u.HookEventName == "UserPromptExpansion":
		return u.AsUserPromptExpansionHookSpecificOutput()
	case u.HookEventName == "SessionStart":
		return u.AsSessionStartHookSpecificOutput()
	case u.HookEventName == "Setup":
		return u.AsSetupHookSpecificOutput()
	case u.HookEventName == "PreModelSwitch":
		return u.AsPreModelSwitchHookSpecificOutput()
	case u.HookEventName == "PostModelSwitch":
		return u.AsPostModelSwitchHookSpecificOutput()
	case u.HookEventName == "SubagentStart":
		return u.AsSubagentStartHookSpecificOutput()
	case u.HookEventName == "PostToolUse":
		return u.AsPostToolUseHookSpecificOutput()
	case u.HookEventName == "PostToolUseFailure":
		return u.AsPostToolUseFailureHookSpecificOutput()
	case u.HookEventName == "PostToolBatch":
		return u.AsPostToolBatchHookSpecificOutput()
	case u.HookEventName == "Stop":
		return u.AsStopHookSpecificOutput()
	case u.HookEventName == "SubagentStop":
		return u.AsSubagentStopHookSpecificOutput()
	case u.HookEventName == "PermissionDenied":
		return u.AsPermissionDeniedHookSpecificOutput()
	case u.HookEventName == "Notification":
		return u.AsNotificationHookSpecificOutput()
	case u.HookEventName == "PermissionRequest":
		return u.AsPermissionRequestHookSpecificOutput()
	case u.HookEventName == "Elicitation":
		return u.AsElicitationHookSpecificOutput()
	case u.HookEventName == "ElicitationResult":
		return u.AsElicitationResultHookSpecificOutput()
	case u.HookEventName == "CwdChanged":
		return u.AsCwdChangedHookSpecificOutput()
	case u.HookEventName == "FileChanged":
		return u.AsFileChangedHookSpecificOutput()
	case u.HookEventName == "WorktreeCreate":
		return u.AsWorktreeCreateHookSpecificOutput()
	case u.HookEventName == "MessageDisplay":
		return u.AsMessageDisplayHookSpecificOutput()
	}
	return nil, unknownVariant("SyncHookJSONOutputHookSpecificOutput", u.Raw)
}

// UserDialogResult is a discriminated union: The host's answer to a . (TS: UserDialogResult).
// Discriminator key(s): "behavior". Raw holds the complete JSON, so unknown variants and fields round-trip unchanged.
// Read with Decode or the As<Variant> accessors; build one for sending with NewUserDialogResult.
type UserDialogResult struct {
	// Behavior is the "behavior" discriminator ("" when absent).
	Behavior string `json:"-"`
	// Raw is the complete JSON of the value.
	Raw json.RawMessage `json:"-"`
}

func (u *UserDialogResult) UnmarshalJSON(b []byte) error {
	return unmarshalUnion(b, &u.Raw, []string{"behavior"}, &u.Behavior)
}
func (u UserDialogResult) MarshalJSON() ([]byte, error) { return marshalUnion(u.Raw) }

// NewUserDialogResult wraps a variant value (or any JSON-marshalable value) as a UserDialogResult.
func NewUserDialogResult(v any) (UserDialogResult, error) {
	var u UserDialogResult
	err := wrapUnion(v, &u)
	return u, err
}

// AsUserDialogResultCompleted decodes u as UserDialogResultCompleted; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u UserDialogResult) AsUserDialogResultCompleted() (*UserDialogResultCompleted, error) {
	return asVariant[UserDialogResultCompleted](u.Raw, u.Behavior == "completed", "UserDialogResult", "UserDialogResultCompleted")
}

// AsUserDialogResultCancelled decodes u as UserDialogResultCancelled; it returns an ErrVariantMismatch error when the discriminator does not match.
func (u UserDialogResult) AsUserDialogResultCancelled() (*UserDialogResultCancelled, error) {
	return asVariant[UserDialogResultCancelled](u.Raw, u.Behavior == "cancelled", "UserDialogResult", "UserDialogResultCancelled")
}

// Decode returns a pointer to the matching variant struct, or ErrUnknownVariant for a discriminator this package does not know (u.Raw still holds it).
func (u UserDialogResult) Decode() (any, error) {
	switch {
	case u.Behavior == "completed":
		return u.AsUserDialogResultCompleted()
	case u.Behavior == "cancelled":
		return u.AsUserDialogResultCancelled()
	}
	return nil, unknownVariant("UserDialogResult", u.Raw)
}
