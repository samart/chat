package wire

import (
	"encoding/json"
	"reflect"
	"testing"
)

func semEq(t *testing.T, a, b []byte) {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x, y) {
		t.Fatalf("not equal:\n in: %s\nout: %s", a, b)
	}
}

var samples = []string{
	`{"type":"system","subtype":"init","apiKeySource":"none","claude_code_version":"2.1.311","cwd":"/x","tools":["Bash"],"mcp_servers":[{"name":"a","status":"connected","future":1}],"model":"m","permissionMode":"default","slash_commands":[],"output_style":"default","skills":[],"plugins":[],"uuid":"u","session_id":"s","brand_new_field":{"deep":[1,2]}}`,
	`{"type":"assistant","message":{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"hi","citations":null},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}},{"type":"future_block","x":1}],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":2,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"iterations":[]}},"parent_tool_use_id":null,"uuid":"u","session_id":"s"}`,
	`{"type":"result","subtype":"success","duration_ms":1.5,"duration_api_ms":1,"is_error":false,"api_error_status":null,"num_turns":1,"result":"","stop_reason":null,"total_cost_usd":0.01,"usage":{"input_tokens":1,"output_tokens":2,"cache_creation_input_tokens":0,"cache_read_input_tokens":0},"modelUsage":{},"permission_denials":[],"uuid":"u","session_id":"s"}`,
	`{"type":"result","subtype":"error_max_turns","duration_ms":1,"duration_api_ms":1,"is_error":true,"num_turns":"oops","stop_reason":null,"total_cost_usd":0,"usage":{"input_tokens":1,"output_tokens":2,"cache_creation_input_tokens":0,"cache_read_input_tokens":0},"modelUsage":{},"permission_denials":[],"errors":[],"uuid":"u","session_id":"s"}`,
	`{"type":"control_request","request_id":"r1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"rm -rf /"},"permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"rm:*"}],"behavior":"allow","destination":"session"}],"tool_use_id":"t1","hidden_internal":true}}`,
	`{"type":"control_response","response":{"subtype":"success","request_id":"r2","response":{"commands":[],"agents":[],"output_style":"default","available_output_styles":[],"models":[],"account":{}},"pending_permission_requests":[]}}`,
	`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"h"}},"parent_tool_use_id":null,"uuid":"u","session_id":"s"}`,
	`{"type":"keep_alive"}`,
	`{"type":"system","subtype":"session_writes_lost","error_code":"ENOSPC","lost":["transcript"],"uuid":"u","session_id":"s"}`,
	`{"type":"totally_new","x":1}`,
}

func TestRoundTrip(t *testing.T) {
	for _, s := range samples {
		var m StdoutMessage
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		out, _ := json.Marshal(m)
		semEq(t, []byte(s), out)
		v, err := m.Decode()
		if m.Type == "totally_new" {
			if err == nil {
				t.Fatal("expected unknown")
			}
			continue
		}
		if err != nil {
			t.Fatalf("decode %s: %v", s, err)
		}
		// re-encode the typed variant: must still be lossless
		out2, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		semEq(t, []byte(s), out2)
		t.Logf("%T ok", v)
	}
}

func TestTyped(t *testing.T) {
	var m StdoutMessage
	_ = json.Unmarshal([]byte(samples[1]), &m)
	a, err := m.AsAssistantMessage()
	if err != nil {
		t.Fatal(err)
	}
	tb, err := a.Message.Content[0].AsTextBlock()
	if err != nil || tb.Text != "hi" {
		t.Fatal(err)
	}
	tu, _ := a.Message.Content[1].AsToolUseBlock()
	if tu.Name != "Bash" {
		t.Fatal(tu)
	}
	var cr StdoutMessage
	_ = json.Unmarshal([]byte(samples[4]), &cr)
	req, _ := cr.AsControlRequest()
	cut, err := req.Request.AsCanUseToolRequest()
	if err != nil || cut.ToolName != "Bash" {
		t.Fatal(err)
	}
	pu, err := cut.PermissionSuggestions[0].Decode()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%T %+v", pu, pu)
	var r StdoutMessage
	_ = json.Unmarshal([]byte(samples[3]), &r)
	re, err := r.AsResultError()
	if err != nil || re.Subtype != "error_max_turns" || re.NumTurns != 0 {
		t.Fatal(err, re)
	}
	// build a response
	allow, _ := NewPermissionResult(PermissionResultAllow{UpdatedInput: map[string]any{"command": "ls"}})
	payload, _ := json.Marshal(allow)
	t.Logf("permission result: %s", payload)
	resp := ControlResponse{Response: mustInner(t, ControlSuccessResponse{RequestID: "r1", Response: map[string]any{"behavior": "allow"}})}
	b, _ := json.Marshal(resp)
	t.Logf("control response: %s", b)
	if string(b) == "" {
		t.Fatal()
	}
}

func mustInner(t *testing.T, v any) ControlResponseResponse {
	u, err := NewControlResponseResponse(v)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
