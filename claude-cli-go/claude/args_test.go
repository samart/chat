package claude

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func mustArgs(t *testing.T, o Options) []string {
	t.Helper()
	a, err := buildArgs(&o)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

var base = []string{"--output-format", "stream-json", "--verbose", "--input-format", "stream-json"}

// The numbers refer to the Go implementation checklist in
// research/parity/transport.md.
func TestArgsChecklist(t *testing.T) {
	allow := func(context.Context, PermissionRequest) (*PermissionResult, error) { return nil, nil }
	cases := []struct {
		name string
		o    Options
		want []string // elements after the 5 base ones
	}{
		{"1 base only", Options{}, nil},
		{"3 order: model, resume, allowedTools", Options{AllowedTools: []string{"Read", "Bash"}, Resume: "abc", Model: "opus"},
			[]string{"--model=opus", "--resume=abc", "--allowedTools=Read,Bash"}},
		{"4 empty settingSources", Options{SettingSources: []string{}}, []string{"--setting-sources="}},
		{"4 settingSources", Options{SettingSources: []string{"user", "project", "local"}}, []string{"--setting-sources=user,project,local"}},
		{"5 empty tools", Options{Tools: &ToolsOption{List: []string{}}}, []string{"--tools="}},
		{"5 preset tools", Options{Tools: &ToolsOption{Preset: true}}, []string{"--tools=default"}},
		{"6 canUseTool", Options{CanUseTool: allow}, []string{"--permission-prompt-tool=stdio"}},
		{"7 mcp config is two elements", Options{McpServers: []McpServerConfig{{Name: "b", Config: json.RawMessage(`{"type":"stdio","command":"x"}`)}, {Name: "a", Config: json.RawMessage(`{"type":"http","url":"u"}`)}}},
			[]string{"--mcp-config", `{"mcpServers":{"b":{"type":"stdio","command":"x"},"a":{"type":"http","url":"u"}}}`}},
		{"8 extra args", Options{ExtraArgs: []ExtraArg{Flag("replay-user-messages"), Arg("foo", "bar"), Arg("neg", "-5"), Arg("dash", "-")}, Settings: `{"a":1}`},
			[]string{"--replay-user-messages", "--foo", "bar", "--neg=-5", "--dash", "-", "--settings", `{"a":1}`}},
		{"10 maxThinkingTokens 0", Options{MaxThinkingTokens: ptr(0)}, []string{"--thinking=disabled"}},
		{"10 maxThinkingTokens N", Options{MaxThinkingTokens: ptr(8000)}, []string{"--max-thinking-tokens=8000"}},
		{"10 thinking enabled no budget", Options{Thinking: &ThinkingConfig{Type: "enabled"}}, []string{"--thinking=adaptive"}},
		{"10 thinking wins, display", Options{Thinking: &ThinkingConfig{Type: "adaptive", Display: "summarized"}, MaxThinkingTokens: ptr(1)},
			[]string{"--thinking=adaptive", "--thinking-display=summarized"}},
		{"11 budget 0 emitted, turns 0 not", Options{MaxBudgetUSD: ptr(0.0), MaxTurns: 0}, []string{"--max-budget-usd=0"}},
		{"12 skills list", Options{AllowedTools: []string{"Read", "Skill(x)"}, Skills: &SkillsOption{Names: []string{"x", "y"}}},
			[]string{"--allowedTools=Read,Skill(x),Skill(y)"}},
		{"12 skills all", Options{Skills: &SkillsOption{All: true}}, []string{"--allowedTools=Skill"}},
		{"13 json schema", Options{JSONSchema: json.RawMessage(`{"type":"object"}`)}, []string{`--json-schema={"type":"object"}`}},
		{"session flags", Options{Resume: "s1", ForkSession: true, ResumeSessionAt: "u1", IncludePartialMessages: true, PermissionMode: "plan", AdditionalDirectories: []string{"/a", "/b"}},
			[]string{"--resume=s1", "--permission-mode=plan", "--include-partial-messages", "--add-dir=/a", "--add-dir=/b", "--fork-session", "--resume-session-at=u1"}},
	}
	for _, c := range cases {
		got := mustArgs(t, c.o)
		if !reflect.DeepEqual(got[:5], base) {
			t.Errorf("%s: base = %v", c.name, got[:5])
		}
		if rest := got[5:]; !reflect.DeepEqual(rest, c.want) && !(len(rest) == 0 && len(c.want) == 0) {
			t.Errorf("%s:\n got  %q\n want %q", c.name, rest, c.want)
		}
	}
}

func TestArgsErrors(t *testing.T) {
	allow := func(context.Context, PermissionRequest) (*PermissionResult, error) { return nil, nil }
	if _, err := buildArgs(&Options{CanUseTool: allow, PermissionPromptToolName: "x"}); err == nil {
		t.Error("canUseTool + permissionPromptToolName must fail")
	}
	if _, err := buildArgs(&Options{Model: "a", FallbackModel: "a"}); err == nil {
		t.Error("fallback == model must fail")
	}
}

func TestJSNumber(t *testing.T) {
	for in, want := range map[float64]string{0.5: "0.5", 2: "2", 1e-7: "1e-7", 0.000001: "0.000001", 1e21: "1e+21", 123456789012: "123456789012", 1.5e-10: "1.5e-10"} {
		if got := jsNumber(in); got != want {
			t.Errorf("jsNumber(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveCommand(t *testing.T) {
	p, a, _ := resolveCommand(&Options{PathToClaudeCodeExecutable: "/x/claude", ExecutableArgs: []string{"--w"}}, []string{"A"})
	if p != "/x/claude" || !reflect.DeepEqual(a, []string{"--w", "A"}) {
		t.Errorf("native: %s %v", p, a)
	}
	p, a, _ = resolveCommand(&Options{PathToClaudeCodeExecutable: "/x/cli.js"}, []string{"A"})
	if p != "node" || !reflect.DeepEqual(a, []string{"/x/cli.js", "A"}) {
		t.Errorf("js: %s %v", p, a)
	}
	if _, _, err := resolveCommand(&Options{}, nil); err == nil {
		t.Error("missing path must fail")
	}
}

func envMap(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		m[k] = v
	}
	return m
}

func TestEnv(t *testing.T) {
	e := envMap(buildEnv(&Options{Env: map[string]string{"PATH": "/bin", "NODE_OPTIONS": "--x", "DEBUG": "*", "CLAUDE_CODE_ENTRYPOINT": ""}, EnableFileCheckpointing: true, AskUserQuestionPreviewFormat: "markdown"}))
	want := map[string]string{
		"PATH": "/bin", "CLAUDE_CODE_ENTRYPOINT": DefaultEntrypoint, "CLAUDE_AGENT_SDK_VERSION": TargetSDKVersion,
		"CLAUDE_CODE_SDK_READS_SESSION_STATE": "1", "CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING": "true",
		"CLAUDE_CODE_QUESTION_PREVIEW_FORMAT": "markdown",
	}
	if !reflect.DeepEqual(e, want) {
		t.Errorf("env:\n got  %v\n want %v", e, want)
	}
	e = envMap(buildEnv(&Options{Env: map[string]string{"CLAUDE_CODE_ENTRYPOINT": "my-editor", "claude_code_sdk_reads_session_state": "0", "DEBUG_CLAUDE_AGENT_SDK": "true", "CLAUDE_AGENT_SDK_VERSION": "9"}}))
	if e["CLAUDE_CODE_ENTRYPOINT"] != "my-editor" || e["CLAUDE_AGENT_SDK_VERSION"] != "9" || e["DEBUG"] != "1" {
		t.Errorf("explicit values not kept: %v", e)
	}
	if _, dup := e["CLAUDE_CODE_SDK_READS_SESSION_STATE"]; dup {
		t.Error("a case variant of CLAUDE_CODE_SDK_READS_SESSION_STATE exists, so it must not be added")
	}
}
