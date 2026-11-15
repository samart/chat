package claude

import (
	"context"
	"encoding/json"
)

// Request sends any host->CLI control request and returns the "response"
// object. Use it for subtypes this client has no method for. fields are the
// request's fields after "subtype", in order.
//
// There is no built-in timeout (the SDK has none either): pass a ctx with a
// deadline. If the CLI does not know the subtype the error satisfies
// IsUnsupported.
func (q *Query) Request(ctx context.Context, subtype string, fields ...InitField) (json.RawMessage, error) {
	payload := append(orderedObject{{"subtype", subtype}}, fields...)
	// interrupt and cancel_async_message wait behind a blocking-hook hold,
	// like user prompts.
	if subtype == "interrupt" || subtype == "cancel_async_message" {
		if err := q.waitForHold(); err != nil {
			return nil, err
		}
	}
	return q.router.request(ctx, subtype, payload)
}

// F builds one request field for Request.
func F(key string, value any) InitField { return InitField{Key: key, Value: value} }

// Interrupt stops the current turn (SDK interrupt()). With cancelQueued the
// CLI also drops user messages that were queued behind it. The response may
// carry still_queued and cancelled message uuids.
func (q *Query) Interrupt(ctx context.Context, cancelQueued bool) (json.RawMessage, error) {
	if cancelQueued {
		return q.Request(ctx, "interrupt", F("cancel_queued", true))
	}
	return q.Request(ctx, "interrupt")
}

// SetPermissionMode switches the permission mode (SDK setPermissionMode()).
func (q *Query) SetPermissionMode(ctx context.Context, mode string) error {
	_, err := q.Request(ctx, "set_permission_mode", F("mode", mode))
	return err
}

// SetModel switches the model for later turns (SDK setModel()). An empty
// string resets to the default.
func (q *Query) SetModel(ctx context.Context, model string) error {
	var v any
	if model != "" {
		v = model
	}
	_, err := q.Request(ctx, "set_model", F("model", v))
	return err
}

// SetMaxThinkingTokens sets the thinking budget (SDK setMaxThinkingTokens()).
// A nil tokens value clears the limit.
func (q *Query) SetMaxThinkingTokens(ctx context.Context, tokens *int, thinkingDisplay string) error {
	var t any = json.RawMessage("null")
	if tokens != nil {
		t = *tokens
	}
	_, err := q.Request(ctx, "set_max_thinking_tokens", F("max_thinking_tokens", t), F("thinking_display", strOrNil(thinkingDisplay)))
	return err
}

// McpServerStatus lists MCP servers with their status and tools (SDK mcpServerStatus()).
func (q *Query) McpServerStatus(ctx context.Context) (json.RawMessage, error) {
	return q.Request(ctx, "mcp_status")
}

// ReconnectMcpServer reconnects one MCP server (SDK reconnectMcpServer()).
func (q *Query) ReconnectMcpServer(ctx context.Context, serverName string) error {
	_, err := q.Request(ctx, "mcp_reconnect", F("serverName", serverName))
	return err
}

// ToggleMcpServer enables or disables one MCP server (SDK toggleMcpServer()).
func (q *Query) ToggleMcpServer(ctx context.Context, serverName string, enabled bool) error {
	_, err := q.Request(ctx, "mcp_toggle", F("serverName", serverName), F("enabled", enabled))
	return err
}

// SetMcpServers replaces the dynamically configured MCP servers (SDK
// setMcpServers()). servers is the {"name": config} object; in-process
// servers appear as {"type":"sdk","name":"…"}.
func (q *Query) SetMcpServers(ctx context.Context, servers json.RawMessage) (json.RawMessage, error) {
	return q.Request(ctx, "mcp_set_servers", F("servers", servers))
}

// RewindFiles restores files to their state at a user message (SDK
// rewindFiles()). It needs Options.EnableFileCheckpointing. Run it with
// dryRun first to list what would change.
func (q *Query) RewindFiles(ctx context.Context, userMessageID string, dryRun bool) (json.RawMessage, error) {
	var d any
	if dryRun {
		d = true
	}
	return q.Request(ctx, "rewind_files", F("user_message_id", userMessageID), F("dry_run", d))
}

// GetContextUsage returns token usage by category (SDK getContextUsage()).
// The CLI makes token-counting API calls to answer: call it on demand only.
func (q *Query) GetContextUsage(ctx context.Context) (json.RawMessage, error) {
	return q.Request(ctx, "get_context_usage")
}

// StopTask stops one subagent or background task (SDK stopTask()).
func (q *Query) StopTask(ctx context.Context, taskID string) error {
	_, err := q.Request(ctx, "stop_task", F("task_id", taskID))
	return err
}

// BackgroundTask moves a running foreground task (an Agent or Bash tool call)
// to the background, so the main turn can continue. toolUseID is the id of
// the tool_use block that started it. The CLI answers after emitting
// system/background_tasks_changed and a task_updated with is_backgrounded.
func (q *Query) BackgroundTask(ctx context.Context, toolUseID string) (json.RawMessage, error) {
	return q.Request(ctx, "background_tasks", F("tool_use_id", toolUseID))
}

// GetTaskOutput returns a background task's output (SDK getTaskOutput()):
// {output, totalBytes, truncated}.
func (q *Query) GetTaskOutput(ctx context.Context, taskID string) (json.RawMessage, error) {
	return q.Request(ctx, "get_task_output", F("task_id", taskID))
}

// CancelAsyncMessage cancels a queued user message by its uuid (SDK
// cancelAsyncMessage()).
func (q *Query) CancelAsyncMessage(ctx context.Context, messageUUID string) (json.RawMessage, error) {
	return q.Request(ctx, "cancel_async_message", F("message_uuid", messageUUID))
}

// GetSettings returns the merged effective settings (SDK getSettings()).
func (q *Query) GetSettings(ctx context.Context) (json.RawMessage, error) {
	return q.Request(ctx, "get_settings")
}

// RenameSession sets the session title (SDK renameSession()).
func (q *Query) RenameSession(ctx context.Context, title, sessionID string) error {
	_, err := q.Request(ctx, "rename_session", F("title", title), F("source", "host"), F("session_id", strOrNil(sessionID)))
	return err
}

// GenerateSessionTitle asks the CLI to generate a title from a description.
// Not declared in the SDK's public types: check IsUnsupported.
func (q *Query) GenerateSessionTitle(ctx context.Context, description string, persist *bool) (json.RawMessage, error) {
	return q.Request(ctx, "generate_session_title", F("description", description), F("persist", boolPtrOrNil(persist)))
}

// ListPermissionRules returns the saved allow/deny/ask rules.
func (q *Query) ListPermissionRules(ctx context.Context) (json.RawMessage, error) {
	return q.Request(ctx, "list_permission_rules")
}

// ReloadPlugins reloads plugins; the CLI then emits system/commands_changed.
func (q *Query) ReloadPlugins(ctx context.Context) (json.RawMessage, error) {
	return q.Request(ctx, "reload_plugins")
}

// ReloadSkills reloads skills.
func (q *Query) ReloadSkills(ctx context.Context) (json.RawMessage, error) {
	return q.Request(ctx, "reload_skills")
}

// SupportedCommands returns the slash commands: the latest
// system/commands_changed list if one arrived, otherwise the list from the
// initialize response (SDK supportedCommands()).
func (q *Query) SupportedCommands(ctx context.Context) (json.RawMessage, error) {
	init, err := q.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	q.mu.Lock()
	latest := q.commands
	q.mu.Unlock()
	if len(latest) > 0 {
		return latest, nil
	}
	return initField(init, "commands"), nil
}

// SupportedModels returns the model list from the initialize response.
func (q *Query) SupportedModels(ctx context.Context) (json.RawMessage, error) {
	init, err := q.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	return initField(init, "models"), nil
}

// SupportedAgents returns the agent list from the initialize response.
func (q *Query) SupportedAgents(ctx context.Context) (json.RawMessage, error) {
	init, err := q.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	return initField(init, "agents"), nil
}

// AccountInfo returns the account object from the initialize response.
func (q *Query) AccountInfo(ctx context.Context) (json.RawMessage, error) {
	init, err := q.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	return initField(init, "account"), nil
}

func initField(init json.RawMessage, key string) json.RawMessage {
	var m map[string]json.RawMessage
	if json.Unmarshal(init, &m) != nil {
		return nil
	}
	return m[key]
}
