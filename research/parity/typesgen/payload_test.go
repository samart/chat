package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResponsePayloads(t *testing.T) {
	files, _ := filepath.Glob(os.Getenv("CAPTURES") + "/*/*/transcript.ndjson")
	more, _ := filepath.Glob(os.Getenv("CAPTURES") + "/*/transcript.ndjson")
	files = append(files, more...)
	newer := map[string]func() any{
		"initialize":            func() any { return new(InitializeResponse) },
		"get_context_usage":     func() any { return new(GetContextUsageResponse) },
		"get_usage":             func() any { return new(GetUsageResponse) },
		"mcp_status":            func() any { return new(McpStatusResponse) },
		"list_permission_rules": func() any { return new(ListPermissionRulesResponse) },
		"get_hooks_listing":     func() any { return new(GetHooksListingResponse) },
		"rewind_files":          func() any { return new(RewindFilesResult) },
		"interrupt":             func() any { return new(InterruptResponse) },
		"background_tasks":      func() any { return new(BackgroundTasksResponse) },
		"can_use_tool":          func() any { return new(PermissionResult) },
		"hook_callback":         func() any { return new(HookJSONOutput) },
		"mcp_message":           func() any { return new(McpMessageResponse) },
	}
	got := map[string]int{}
	for _, f := range files {
		subs := map[string]string{}
		fh, _ := os.Open(f)
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
		for sc.Scan() {
			var env struct {
				Msg struct {
					Type      string          `json:"type"`
					RequestID string          `json:"request_id"`
					Request   json.RawMessage `json:"request"`
					Response  struct {
						Subtype   string          `json:"subtype"`
						RequestID string          `json:"request_id"`
						Response  json.RawMessage `json:"response"`
					} `json:"response"`
				} `json:"msg"`
			}
			if json.Unmarshal(sc.Bytes(), &env) != nil {
				continue
			}
			if env.Msg.Type == "control_request" {
				var r struct{ Subtype string `json:"subtype"` }
				_ = json.Unmarshal(env.Msg.Request, &r)
				subs[env.Msg.RequestID] = r.Subtype
			}
			if env.Msg.Type == "control_response" && env.Msg.Response.Subtype == "success" && len(env.Msg.Response.Response) > 2 {
				sub := subs[env.Msg.Response.RequestID]
				mk, ok := newer[sub]
				if !ok {
					continue
				}
				v := mk()
				if err := json.Unmarshal(env.Msg.Response.Response, v); err != nil {
					t.Fatalf("%s: %v", sub, err)
				}
				b, _ := json.Marshal(v)
				semEq(t, env.Msg.Response.Response, b)
				got[sub]++
			}
		}
		fh.Close()
	}
	t.Logf("typed payloads round-tripped: %v", got)
}
