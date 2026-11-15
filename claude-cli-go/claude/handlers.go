package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// orderedObject marshals as a JSON object with keys in the given order, so
// frames match the SDK's key order. Fields with a nil Value are omitted
// (JSON.stringify drops undefined).
type orderedObject []InitField

func (o orderedObject) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	first := true
	for _, f := range o {
		if f.Value == nil {
			continue
		}
		v, err := marshalJSON(f.Value)
		if err != nil {
			return nil, err
		}
		k, err := marshalJSON(f.Key)
		if err != nil {
			return nil, err
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func rawOrNil(r json.RawMessage) any {
	if len(r) == 0 {
		return nil
	}
	return r
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolPtrOrNil(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}

// buildInitializeRequest assembles the initialize control request in the
// SDK's key order (research/parity/transport.md §5.1). It also registers hook
// callbacks under the ids hook_0, hook_1, … and decides whether prompts must
// wait for the response (a hook with onFailure other than "continue").
func (q *Query) buildInitializeRequest(ctx context.Context) (orderedObject, error) {
	o := &q.opts
	req := orderedObject{{"subtype", "initialize"}}

	// hooks: {Event: [{matcher, hookCallbackIds, timeout, onFailure}]}
	if len(o.Hooks) > 0 {
		var events orderedObject
		n := 0
		for _, ev := range o.Hooks {
			if len(ev.Matchers) == 0 {
				continue
			}
			var matchers []orderedObject
			for _, m := range ev.Matchers {
				ids := make([]string, 0, len(m.Hooks))
				for _, fn := range m.Hooks {
					if fn == nil {
						return nil, errors.New("claude: nil hook callback")
					}
					id := fmt.Sprintf("hook_%d", n)
					n++
					q.hooks[id] = fn
					ids = append(ids, id)
				}
				mo := orderedObject{}
				if m.Matcher != nil {
					mo = append(mo, InitField{"matcher", *m.Matcher})
				}
				mo = append(mo, InitField{"hookCallbackIds", ids})
				if m.Timeout != nil {
					mo = append(mo, InitField{"timeout", *m.Timeout})
				}
				if m.OnFailure != "" {
					mo = append(mo, InitField{"onFailure", m.OnFailure})
					if m.OnFailure != "continue" {
						q.hold = true
					}
				}
				matchers = append(matchers, mo)
			}
			events = append(events, InitField{ev.Event, matchers})
		}
		req = append(req, InitField{"hooks", events})
	}

	// In-process MCP servers are announced here, never in --mcp-config.
	if len(o.SdkMcpServers) > 0 {
		names := make([]string, 0, len(o.SdkMcpServers))
		var cfgs orderedObject
		for _, s := range o.SdkMcpServers {
			names = append(names, s.Name)
			if len(s.Settings) > 0 && string(s.Settings) != "{}" {
				cfgs = append(cfgs, InitField{s.Name, s.Settings})
			}
		}
		req = append(req, InitField{"sdkMcpServers", names})
		if len(cfgs) > 0 {
			req = append(req, InitField{"sdkMcpServerConfigs", cfgs})
		}
		if manifests := captureMcpManifests(ctx, o.SdkMcpServers, firstNonEmpty(o.SDKVersion, TargetSDKVersion)); len(manifests) > 0 {
			req = append(req, InitField{"sdkMcpServerManifests", manifests})
		}
	}

	req = append(req, InitField{"jsonSchema", rawOrNil(o.JSONSchema)})

	// systemPrompt: nil option -> [""]; preset -> omitted + appendSystemPrompt.
	var systemPrompt, appendPrompt, excludeDynamic, snapshot any
	switch sp := o.SystemPrompt; {
	case sp == nil:
		systemPrompt = []string{""}
	case sp.Preset:
		appendPrompt = strOrNil(sp.Append)
		excludeDynamic = boolPtrOrNil(sp.ExcludeDynamicSections)
		snapshot = boolPtrOrNil(sp.Snapshot)
	default:
		blocks := sp.Custom
		if blocks == nil {
			blocks = []string{""}
		}
		systemPrompt = blocks
		snapshot = boolPtrOrNil(sp.Snapshot)
	}
	var skills any
	if o.Skills != nil && !o.Skills.All {
		skills = o.Skills.Names
		if o.Skills.Names == nil {
			skills = []string{}
		}
	}
	var dialogKinds any
	if len(o.SupportedDialogKinds) > 0 {
		dialogKinds = o.SupportedDialogKinds
	}
	req = append(req,
		InitField{"systemPrompt", systemPrompt},
		InitField{"appendSystemPrompt", appendPrompt},
		InitField{"planModeInstructions", strOrNil(o.PlanModeInstructions)},
		InitField{"systemPromptSnapshot", snapshot},
		InitField{"appendSubagentSystemPrompt", strOrNil(o.AppendSubagentSystemPrompt)},
		InitField{"excludeDynamicSections", excludeDynamic},
		InitField{"agents", rawOrNil(o.Agents)},
		InitField{"title", strOrNil(o.Title)},
		InitField{"skills", skills},
		InitField{"promptSuggestions", boolPtrOrNil(o.PromptSuggestions)},
		InitField{"agentProgressSummaries", boolPtrOrNil(o.AgentProgressSummaries)},
		InitField{"forwardSubagentText", boolPtrOrNil(o.ForwardSubagentText)},
		InitField{"supportedDialogKinds", dialogKinds},
		InitField{"perTaskStopAffordance", boolPtrOrNil(o.PerTaskStopAffordance)},
		InitField{"rapidFollowupPreempt", boolPtrOrNil(o.RapidFollowupPreempt)},
		InitField{"forwardNestedSubagentToolCalls", boolPtrOrNil(o.ForwardNestedSubagentToolCalls)},
		InitField{"workspaceTrust", rawOrNil(o.WorkspaceTrust)},
	)
	for _, extra := range o.InitializeExtra {
		replaced := false
		for i := range req {
			if req[i].Key == extra.Key {
				req[i].Value, replaced = extra.Value, true
			}
		}
		if !replaced {
			req = append(req, extra)
		}
	}
	return req, nil
}

// handleControlRequest answers one CLI->host control request. The behaviour
// for each subtype, including the cases that send no reply, follows the SDK's
// processControlRequest (research/parity/transport.md §7).
func (q *Query) handleControlRequest(ctx context.Context, requestID, subtype string, request json.RawMessage) (any, error) {
	switch subtype {
	case "can_use_tool":
		return q.handleCanUseTool(ctx, requestID, request)

	case "hook_callback":
		var r struct {
			CallbackID string          `json:"callback_id"`
			Input      json.RawMessage `json:"input"`
			ToolUseID  string          `json:"tool_use_id"`
		}
		if err := json.Unmarshal(request, &r); err != nil {
			return nil, err
		}
		fn := q.hooks[r.CallbackID]
		if fn == nil {
			return nil, fmt.Errorf("No hook callback found for ID: %s", r.CallbackID)
		}
		return fn(ctx, r.Input, r.ToolUseID)

	case "mcp_message":
		return q.handleMcpMessage(ctx, request)

	case "elicitation":
		if q.opts.OnElicitation == nil {
			return orderedObject{{"action", "decline"}}, nil
		}
		res, err := q.opts.OnElicitation(ctx, requestID, request)
		if err == nil && res == nil {
			return nil, errNoReply
		}
		return res, err

	case "request_user_dialog":
		if q.opts.OnUserDialog == nil {
			return nil, errNoReply // a capable client, or the CLI's own deadline, settles it
		}
		res, err := q.opts.OnUserDialog(ctx, requestID, request)
		if err == nil && res == nil {
			return nil, errNoReply
		}
		return res, err

	case "remote_tool_call", "remote_plumbing_call", "remote_tools_probe", "remote_tools_reannounce":
		return nil, errNoReply // meant for another machine serving this session's tools

	default:
		// SDK 0.3.311 sends this text with no error_code. (0.3.325 adds
		// error_code "unsupported_request"; add it when targeting that CLI.)
		return nil, errors.New("Unsupported control request subtype: " + subtype)
	}
}

func (q *Query) handleCanUseTool(ctx context.Context, requestID string, request json.RawMessage) (any, error) {
	if q.opts.CanUseTool == nil {
		return nil, errors.New("canUseTool callback is not provided.")
	}
	var req PermissionRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return nil, err
	}
	req.RequestID = requestID
	req.Raw = request

	res, err := q.opts.CanUseTool(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, errNoReply
	}
	// The SDK replies {...callbackResult, toolUseID}: the callback's keys
	// first, then toolUseID from the request.
	var toolUseID any
	if req.ToolUseID != "" {
		toolUseID = req.ToolUseID
	}
	if len(res.Raw) > 0 {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(res.Raw, &fields); err != nil {
			return nil, fmt.Errorf("claude: PermissionResult.Raw is not a JSON object: %w", err)
		}
		delete(fields, "toolUseID")
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// "behavior" first, as every SDK caller writes it.
		out := orderedObject{}
		if v, ok := fields["behavior"]; ok {
			out = append(out, InitField{"behavior", v})
		}
		for _, k := range keys {
			if k != "behavior" {
				out = append(out, InitField{k, fields[k]})
			}
		}
		return append(out, InitField{"toolUseID", toolUseID}), nil
	}
	switch res.Behavior {
	case "allow":
		return orderedObject{
			{"behavior", "allow"},
			{"updatedInput", rawOrNil(res.UpdatedInput)},
			{"updatedPermissions", rawOrNil(res.UpdatedPermissions)},
			{"toolUseID", toolUseID},
		}, nil
	case "deny":
		out := orderedObject{{"behavior", "deny"}, {"message", res.Message}}
		if res.Interrupt {
			out = append(out, InitField{"interrupt", true})
		}
		return append(out, InitField{"toolUseID", toolUseID}), nil
	default:
		return nil, fmt.Errorf("claude: PermissionResult.Behavior must be \"allow\" or \"deny\", got %q", res.Behavior)
	}
}

// mcpAck is the SDK's reply to an MCP notification.
var mcpAck = json.RawMessage(`{"mcp_response":{"jsonrpc":"2.0","result":{},"id":0}}`)

func (q *Query) handleMcpMessage(ctx context.Context, request json.RawMessage) (any, error) {
	var r struct {
		ServerName string          `json:"server_name"`
		Message    json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(request, &r); err != nil {
		return nil, err
	}
	srv, ok := q.mcp[r.ServerName]
	if !ok {
		return nil, fmt.Errorf("SDK MCP server not found: %s", r.ServerName)
	}
	var probe struct {
		Method *string         `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(r.Message, &probe)
	isRequest := probe.Method != nil && len(probe.ID) > 0 && string(probe.ID) != "null"

	resp, err := srv.Handle(ctx, r.Message)
	if !isRequest {
		return mcpAck, nil // notifications are acknowledged whatever the handler did
	}
	if err != nil {
		// A handler failure becomes a JSON-RPC error so the CLI gets an
		// answer for this id.
		msg, _ := marshalJSON(err.Error())
		resp = json.RawMessage(`{"jsonrpc":"2.0","id":` + string(probe.ID) + `,"error":{"code":-32603,"message":` + string(msg) + `}}`)
	}
	if len(resp) == 0 {
		return nil, errors.New("No message handler registered")
	}
	return orderedObject{{"mcp_response", resp}}, nil
}

// SendMcpMessage sends a message that an in-process MCP server initiates
// (for example a notifications/tools/list_changed notification) to the CLI.
// The SDK writes these with type first and a UUID request id, and never
// waits for the reply.
func (q *Query) SendMcpMessage(serverName string, message json.RawMessage) error {
	if _, ok := q.mcp[serverName]; !ok {
		return fmt.Errorf("SDK MCP server not found: %s", serverName)
	}
	return q.writeJSON(wireMCPControlRequest{
		Type:      "control_request",
		RequestID: newUUID(),
		Request: orderedObject{
			{"subtype", "mcp_message"},
			{"server_name", serverName},
			{"message", message},
		},
	})
}
