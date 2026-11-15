package claude

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeOptions(t *testing.T, scenario string) (Options, string) {
	t.Helper()
	rec := filepath.Join(t.TempDir(), "stdin.ndjson")
	return Options{
		PathToClaudeCodeExecutable: os.Args[0],
		Env:                        map[string]string{"CLAUDE_FAKE_SCENARIO": scenario, "CLAUDE_FAKE_RECORD": rec, "PATH": os.Getenv("PATH")},
	}, rec
}

func readRecord(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func collect(t *testing.T, q *Query, until func(Message) bool) []Message {
	t.Helper()
	var got []Message
	timeout := time.After(10 * time.Second)
	for {
		select {
		case m, ok := <-q.Messages():
			if !ok {
				return got
			}
			got = append(got, m)
			if until != nil && until(m) {
				return got
			}
		case <-timeout:
			t.Fatalf("timed out; got %d messages; stderr: %s", len(got), q.StderrTail())
		}
	}
}

func types(ms []Message) string {
	var s []string
	for _, m := range ms {
		if m.Subtype != "" {
			s = append(s, m.Type+"/"+m.Subtype)
		} else {
			s = append(s, m.Type)
		}
	}
	return strings.Join(s, " ")
}

func TestTurnWithApprovalHookAndMcp(t *testing.T) {
	opts, rec := fakeOptions(t, "basic")
	var seen PermissionRequest
	opts.CanUseTool = func(ctx context.Context, req PermissionRequest) (*PermissionResult, error) {
		seen = req
		return Allow(req.Input), nil
	}
	opts.Hooks = []HookEventConfig{
		{Event: "PreToolUse", Matchers: []HookMatcher{
			{Matcher: ptr("Edit|Write"), Hooks: []HookFunc{func(context.Context, json.RawMessage, string) (any, error) { return nil, nil }}},
			{Matcher: ptr("Bash"), Timeout: ptr(5), Hooks: []HookFunc{func(_ context.Context, in json.RawMessage, id string) (any, error) {
				return map[string]any{"continue": true, "sawToolUse": id}, nil
			}}},
		}},
	}
	opts.SdkMcpServers = []SdkMcpServer{{Name: "ide", Handle: func(_ context.Context, msg json.RawMessage) (json.RawMessage, error) {
		if strings.Contains(string(msg), `"tools/list"`) {
			return json.RawMessage(`{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}`), nil
		}
		return nil, nil
	}}}
	opts.SystemPrompt = PresetSystemPrompt("Be brief.")

	q, err := Start(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	init, err := q.Initialized(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cmds, _ := q.SupportedCommands(context.Background()); !strings.Contains(string(cmds), `"compact"`) || !strings.Contains(string(init), `"account"`) {
		t.Fatalf("initialize response not exposed: %s", init)
	}
	if err := q.SendText("hi <there> & \"you\""); err != nil {
		t.Fatal(err)
	}
	got := collect(t, q, func(m Message) bool { return m.Type == "result" })

	// control_*, keep_alive, non-JSON, blank lines and sdk_host_only state
	// frames are swallowed; an unknown frame type is delivered.
	want := "system/init stream_event assistant user brand_new_frame_type result/success"
	if types(got) != want {
		t.Fatalf("frames:\n got  %s\n want %s", types(got), want)
	}
	if !strings.Contains(string(got[1].Raw), `"<b>&hi</b>"`) {
		t.Errorf("Raw must be the CLI's bytes: %s", got[1].Raw)
	}
	if seen.ToolName != "Bash" || seen.AgentID != "agent-7" || seen.RequestID != "p1" || !strings.Contains(string(seen.Raw), `"future_field":true`) {
		t.Errorf("permission request not decoded/preserved: %+v", seen)
	}
	if err := q.SetModel(context.Background(), "opus"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.GetContextUsage(context.Background()); !IsUnsupported(err) {
		t.Fatalf("want unsupported error, got %v", err)
	}
	q.Close()

	lines := readRecord(t, rec)
	wantInit := `{"request_id":"` // request_id first, then type, then request with subtype first
	if !strings.HasPrefix(lines[0], wantInit) || !strings.Contains(lines[0], `","type":"control_request","request":{"subtype":"initialize","hooks":{"PreToolUse":[{"matcher":"Edit|Write","hookCallbackIds":["hook_0"]},{"matcher":"Bash","hookCallbackIds":["hook_1"],"timeout":5}]},"sdkMcpServers":["ide"],"appendSystemPrompt":"Be brief."}}`) {
		t.Errorf("initialize frame:\n%s", lines[0])
	}
	wantUser := `{"type":"user","session_id":"","message":{"role":"user","content":[{"type":"text","text":"hi <there> & \"you\""}]},"parent_tool_use_id":null}`
	if lines[1] != wantUser {
		t.Errorf("user frame:\n got  %s\n want %s", lines[1], wantUser)
	}
	all := strings.Join(lines, "\n")
	for _, w := range []string{
		`{"type":"control_response","response":{"subtype":"success","request_id":"p1","response":{"behavior":"allow","updatedInput":{"command":"touch x"},"toolUseID":"toolu_1"}}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"h1","response":{"continue":true,"sawToolUse":"toolu_1"}}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"m1","response":{"mcp_response":{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}}}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"m2","response":{"mcp_response":{"jsonrpc":"2.0","result":{},"id":0}}}}`,
		`"request":{"subtype":"set_model","model":"opus"}}`,
	} {
		if !strings.Contains(all, w) {
			t.Errorf("missing frame on stdin:\n%s\n--- all ---\n%s", w, all)
		}
	}
}

func TestUnknownAndDefaultReplies(t *testing.T) {
	opts, rec := fakeOptions(t, "unknown")
	q, err := Start(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SendText("go"); err != nil {
		t.Fatal(err)
	}
	collect(t, q, func(m Message) bool { return m.Type == "assistant" })
	time.Sleep(200 * time.Millisecond)
	q.Close()
	all := strings.Join(readRecord(t, rec), "\n")
	if w := `{"type":"control_response","response":{"subtype":"error","request_id":"u1","error":"Unsupported control request subtype: mystery_request"}}`; !strings.Contains(all, w) {
		t.Errorf("unknown subtype reply missing:\n%s", all)
	}
	if strings.Contains(all, `"request_id":"u2"`) {
		t.Errorf("remote_tools_probe must get no reply:\n%s", all)
	}
	if w := `{"type":"control_response","response":{"subtype":"success","request_id":"u3","response":{"action":"decline"}}}`; !strings.Contains(all, w) {
		t.Errorf("elicitation default reply missing:\n%s", all)
	}
	// Default system prompt is an empty prompt, as in the SDK.
	if !strings.Contains(all, `"request":{"subtype":"initialize","systemPrompt":[""]}`) {
		t.Errorf("default initialize payload:\n%s", strings.SplitN(all, "\n", 2)[0])
	}
}

func TestCancelStillSendsLateResult(t *testing.T) {
	opts, rec := fakeOptions(t, "cancel")
	cancelled := make(chan struct{})
	opts.CanUseTool = func(ctx context.Context, req PermissionRequest) (*PermissionResult, error) {
		<-ctx.Done() // the CLI's control_cancel_request cancels this context
		close(cancelled)
		return Deny("cancelled by CLI"), nil
	}
	q, err := Start(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	_ = q.SendText("go")
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("handler context was not cancelled by control_cancel_request")
	}
	time.Sleep(200 * time.Millisecond)
	q.Close()
	all := strings.Join(readRecord(t, rec), "\n")
	if w := `"request_id":"p1","response":{"behavior":"deny","message":"cancelled by CLI","toolUseID":"toolu_1"}`; !strings.Contains(all, w) {
		t.Errorf("SDK parity: a late result after cancel is still written:\n%s", all)
	}
}

func TestInterruptAndErrorResultOnExit(t *testing.T) {
	opts, _ := fakeOptions(t, "exit3")
	q, err := Start(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Interrupt(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	got := collect(t, q, func(m Message) bool { return m.Type == "result" })
	if types(got) != "result/error_during_execution" {
		t.Fatalf("got %s", types(got))
	}
	_ = q.EndInput()
	rest := collect(t, q, nil) // until the channel closes
	if types(rest) != "result/success" {
		t.Fatalf("got %s", types(rest))
	}
	var er *ErrorResultError
	if err := q.Wait(); !errors.As(err, &er) || err.Error() != "Claude Code returned an error result: API Error: 529 overloaded" {
		t.Fatalf("want error-result substitution, got %v", err)
	}
	var pe *ProcessError
	if !errors.As(q.Err(), &pe) || pe.ExitCode != 3 || !strings.Contains(pe.Stderr, "something broke") {
		t.Fatalf("underlying exit error: %+v", pe)
	}
}

func TestPromptClosesStdinAfterResult(t *testing.T) {
	opts, _ := fakeOptions(t, "basic")
	opts.CanUseTool = func(_ context.Context, req PermissionRequest) (*PermissionResult, error) {
		return Allow(req.Input), nil
	}
	q, err := Prompt(context.Background(), "one shot", opts)
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, q, nil) // channel closes because stdin is closed after the result
	if !strings.HasSuffix(types(got), "result/success") {
		t.Fatalf("got %s", types(got))
	}
	if err := q.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestClosedQueryFailsRequests(t *testing.T) {
	opts, _ := fakeOptions(t, "basic")
	q, err := Start(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	q.Close()
	if err := q.SetModel(context.Background(), "x"); !errors.Is(err, ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
	if err := q.SendText("x"); !errors.Is(err, ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
}
