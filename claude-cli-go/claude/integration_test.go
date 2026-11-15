package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegrationRealCLI drives a real claude binary against the local fake
// Messages API in verification/mock-api.mjs. It never contacts Anthropic and
// never touches ~/.claude. It is skipped unless both variables are set:
//
//	CLAUDE_INTEGRATION_BIN=<path to claude>  CLAUDE_INTEGRATION_MOCK_URL=http://127.0.0.1:8787
func TestIntegrationRealCLI(t *testing.T) {
	bin, mock := os.Getenv("CLAUDE_INTEGRATION_BIN"), os.Getenv("CLAUDE_INTEGRATION_MOCK_URL")
	if bin == "" || mock == "" {
		t.Skip("set CLAUDE_INTEGRATION_BIN and CLAUDE_INTEGRATION_MOCK_URL to run")
	}
	if !strings.HasPrefix(mock, "http://127.0.0.1:") && !strings.HasPrefix(mock, "http://localhost:") {
		t.Fatal("the mock URL must be local")
	}
	dir := t.TempDir()
	cwd := filepath.Join(dir, "cwd")
	_ = os.MkdirAll(cwd, 0o755)
	env := map[string]string{
		"PATH":                    os.Getenv("PATH"),
		"HOME":                    dir,
		"CLAUDE_CONFIG_DIR":       filepath.Join(dir, "config"),
		"ANTHROPIC_API_KEY":       "sk-ant-mock-not-a-real-key",
		"ANTHROPIC_BASE_URL":      mock,
		"DISABLE_AUTOUPDATER":     "1",
		"DISABLE_TELEMETRY":       "1",
		"DISABLE_ERROR_REPORTING": "1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
	}
	var asked []PermissionRequest
	hookCalls := 0
	opts := Options{
		PathToClaudeCodeExecutable: bin,
		Cwd:                        cwd,
		Env:                        env,
		Model:                      "haiku",
		MaxTurns:                   4,
		SettingSources:             []string{},
		IncludePartialMessages:     true,
		SystemPrompt:               PresetSystemPrompt(""),
		InitializeTimeout:          60 * time.Second,
		ExtraArgs:                  []ExtraArg{Flag("no-session-persistence")},
		CanUseTool: func(_ context.Context, req PermissionRequest) (*PermissionResult, error) {
			asked = append(asked, req)
			return Allow(req.Input), nil
		},
		Hooks: []HookEventConfig{{Event: "PreToolUse", Matchers: []HookMatcher{{Hooks: []HookFunc{
			func(context.Context, json.RawMessage, string) (any, error) { hookCalls++; return nil, nil },
		}}}}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// The mock picks its script from the first user message of a session, so
	// each scenario gets its own process.
	session := func(prompt string, during func(q *Query)) []Message {
		q, err := Start(ctx, opts)
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		init, err := q.Initialized(ctx)
		if err != nil {
			t.Fatalf("initialize: %v\nstderr: %s", err, q.StderrTail())
		}
		var ir struct {
			Commands []struct{ Name string } `json:"commands"`
			Models   []json.RawMessage       `json:"models"`
		}
		if json.Unmarshal(init, &ir) != nil || len(ir.Commands) == 0 || len(ir.Models) == 0 {
			t.Fatalf("initialize response lacks commands/models: %.300s", init)
		}
		if err := q.SendText(prompt); err != nil {
			t.Fatal(err)
		}
		var got []Message
		for {
			select {
			case m, ok := <-q.Messages():
				if !ok {
					t.Fatalf("stream ended: %v\nstderr: %s", q.Err(), q.StderrTail())
				}
				got = append(got, m)
				if m.Type == "result" {
					if during != nil {
						during(q)
					}
					return got
				}
			case <-ctx.Done():
				t.Fatalf("timeout; got %s\nstderr: %s", types(got), q.StderrTail())
			}
		}
	}

	// Session 1: plain text, then control requests on the live session.
	got := session("SCENARIO:text hi", func(q *Query) {
		if err := q.SetPermissionMode(ctx, "acceptEdits"); err != nil {
			t.Errorf("set_permission_mode: %v", err)
		}
		if err := q.SetPermissionMode(ctx, "default"); err != nil {
			t.Errorf("set_permission_mode: %v", err)
		}
		if err := q.SetPermissionMode(ctx, "no-such-mode"); err == nil {
			t.Error("an invalid permission mode must be rejected")
		} else {
			t.Logf("invalid mode error: %.160s", err)
		}
		if st, err := q.McpServerStatus(ctx); err != nil || len(st) == 0 {
			t.Errorf("mcp_status: %v %s", err, st)
		}
		if _, err := q.Request(ctx, "definitely_not_a_subtype"); !IsUnsupported(err) {
			t.Errorf("want unsupported-subtype error, got %v", err)
		}
		if cmds, err := q.SupportedCommands(ctx); err != nil || !strings.Contains(string(cmds), `"compact"`) {
			t.Errorf("supported commands: %v %.100s", err, cmds)
		}
	})
	seq := types(got)
	t.Logf("text session: %s", compact(seq))
	if got[0].Type != "system" || got[0].Subtype != "init" || got[0].SessionID == "" ||
		!strings.Contains(seq, "stream_event") || !strings.Contains(seq, "assistant") || got[len(got)-1].Subtype != "success" {
		t.Fatalf("unexpected sequence: %s", seq)
	}

	// Session 2: a tool that needs approval, plus a host PreToolUse hook.
	got = session("SCENARIO:tool_allow", nil)
	t.Logf("tool session: %s", compact(types(got)))
	if len(asked) != 1 || asked[0].ToolName != "Bash" || asked[0].ToolUseID == "" {
		t.Fatalf("can_use_tool not received as expected: %+v", asked)
	}
	if hookCalls == 0 {
		t.Fatal("PreToolUse hook was not called")
	}
	if _, err := os.Stat(filepath.Join(cwd, "allowed-marker")); err != nil {
		t.Fatalf("the approved tool did not run: %v", err)
	}
	if got[len(got)-1].Subtype != "success" {
		t.Fatalf("tool session ended with %s", got[len(got)-1].Subtype)
	}
	t.Logf("approval request: tool=%s display_name=%q suggestions=%d bytes", asked[0].ToolName, asked[0].DisplayName, len(asked[0].PermissionSuggestions))

	// Session 3: a denied tool. The denial text must reach the model.
	opts.CanUseTool = func(_ context.Context, req PermissionRequest) (*PermissionResult, error) {
		return Deny("not allowed by the test"), nil
	}
	got = session("SCENARIO:tool_deny", nil)
	t.Logf("deny session: %s", compact(types(got)))
	if _, err := os.Stat(filepath.Join(cwd, "denied-marker")); err == nil {
		t.Fatal("a denied tool ran")
	}
	denied := false
	for _, m := range got {
		if m.Type == "user" && strings.Contains(string(m.Raw), "not allowed by the test") {
			denied = true
		}
	}
	if !denied {
		t.Fatal("the deny message did not come back as a tool_result")
	}
}

func compact(seq string) string {
	out, prev, n := []string{}, "", 0
	flush := func() {
		if prev == "" {
			return
		}
		if n > 1 {
			out = append(out, prev+"×"+string(rune('0'+min(n, 9))))
		} else {
			out = append(out, prev)
		}
	}
	for _, s := range strings.Fields(seq) {
		if s == prev {
			n++
			continue
		}
		flush()
		prev, n = s, 1
	}
	flush()
	return strings.Join(out, " ")
}
