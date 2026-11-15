package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// The test binary doubles as a scripted fake claude CLI: when
// CLAUDE_FAKE_SCENARIO is set, TestMain runs the fake instead of the tests.
func TestMain(m *testing.M) {
	if sc := os.Getenv("CLAUDE_FAKE_SCENARIO"); sc != "" {
		runFakeCLI(sc)
		return
	}
	os.Exit(m.Run())
}

func runFakeCLI(scenario string) {
	out := bufio.NewWriter(os.Stdout)
	emit := func(s string) { fmt.Fprintln(out, s); out.Flush() }
	var rec *os.File
	if p := os.Getenv("CLAUDE_FAKE_RECORD"); p != "" {
		rec, _ = os.Create(p)
		defer rec.Close()
	}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 64<<20)
	for in.Scan() {
		line := in.Text()
		if rec != nil {
			fmt.Fprintln(rec, line)
		}
		var f struct {
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
			Request   struct {
				Subtype string `json:"subtype"`
			} `json:"request"`
		}
		_ = json.Unmarshal([]byte(line), &f)
		switch {
		case f.Type == "control_request" && f.Request.Subtype == "initialize":
			emit("not json at all") // must be skipped
			emit("")
			emit(`{"type":"keep_alive","startup_mark":"ready"}`)
			emit(`{"type":"control_response","response":{"subtype":"success","request_id":"` + f.RequestID + `","response":{"commands":[{"name":"compact","description":"d","argumentHint":""}],"models":[{"value":"default"}],"account":{"email":"x"},"pid":1}}}`)
		case f.Type == "control_request" && f.Request.Subtype == "interrupt":
			emit(`{"type":"control_response","response":{"subtype":"success","request_id":"` + f.RequestID + `","response":{"still_queued":[]}}}`)
			emit(`{"type":"result","subtype":"error_during_execution","is_error":true,"errors":[" aborted "],"session_id":"s1","uuid":"r2"}`)
		case f.Type == "control_request" && f.Request.Subtype == "set_model":
			emit(`{"type":"control_response","response":{"subtype":"success","request_id":"` + f.RequestID + `"}}`)
		case f.Type == "control_request":
			emit(`{"type":"control_response","response":{"subtype":"error","request_id":"` + f.RequestID + `","error":"Unsupported control request subtype: ` + f.Request.Subtype + `","error_code":"unsupported_request"}}`)
		case f.Type == "user":
			emit(`{"type":"system","subtype":"init","session_id":"s1","uuid":"i1","model":"m","tools":["Bash"]}`)
			emit(`{"type":"system","subtype":"session_state_changed","state":"running","sdk_host_only":true,"session_id":"s1"}`)
			emit(`{"type":"stream_event","session_id":"s1","parent_tool_use_id":null,"event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"<b>&hi</b>"}}}`)
			emit(`{"type":"assistant","session_id":"s1","uuid":"a1","parent_tool_use_id":null,"message":{"id":"msg1","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"touch x"}}]}}`)
			switch scenario {
			case "unknown":
				emit(`{"type":"control_request","request_id":"u1","request":{"subtype":"mystery_request","x":1}}`)
				emit(`{"type":"control_request","request_id":"u2","request":{"subtype":"remote_tools_probe"}}`)
				emit(`{"type":"control_request","request_id":"u3","request":{"subtype":"elicitation","mcp_server_name":"m","message":"?"}}`)
			case "cancel":
				emit(`{"type":"control_request","request_id":"p1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"touch x"},"tool_use_id":"toolu_1"}}`)
				time.Sleep(50 * time.Millisecond)
				emit(`{"type":"control_cancel_request","request_id":"p1"}`)
			default:
				emit(`{"type":"control_request","request_id":"p1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"touch x"},"tool_use_id":"toolu_1","agent_id":"agent-7","permission_suggestions":[{"type":"addRules"}],"display_name":"Bash","future_field":true}}`)
				emit(`{"type":"control_request","request_id":"h1","request":{"subtype":"hook_callback","callback_id":"hook_1","input":{"hook_event_name":"PreToolUse"},"tool_use_id":"toolu_1"}}`)
				emit(`{"type":"control_request","request_id":"m1","request":{"subtype":"mcp_message","server_name":"ide","message":{"jsonrpc":"2.0","id":7,"method":"tools/list"}}}`)
				emit(`{"type":"control_request","request_id":"m2","request":{"subtype":"mcp_message","server_name":"ide","message":{"jsonrpc":"2.0","method":"notifications/initialized"}}}`)
			}
		case f.Type == "control_response":
			if scenario == "basic" && f.RequestID == "" {
				var r struct {
					Response struct {
						RequestID string `json:"request_id"`
					} `json:"response"`
				}
				_ = json.Unmarshal([]byte(line), &r)
				if r.Response.RequestID == "p1" {
					emit(`{"type":"user","session_id":"s1","uuid":"u1","parent_tool_use_id":null,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}}`)
					emit(`{"type":"brand_new_frame_type","payload":{"x":1}}`)
					emit(`{"type":"system","subtype":"session_state_changed","state":"idle","sdk_host_only":true,"session_id":"s1"}`)
					emit(`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s1","uuid":"r1","total_cost_usd":0.01}`)
				}
			}
		}
	}
	if scenario == "exit3" {
		emit(`{"type":"result","subtype":"success","is_error":true,"result":"API Error: 529 overloaded","session_id":"s1","uuid":"r9"}`)
		fmt.Fprintln(os.Stderr, "fatal: something broke")
		os.Exit(3)
	}
}
