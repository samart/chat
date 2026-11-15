package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestIntegrationEntrypointParity checks that identifying as "sdk-go" rather
// than the TypeScript SDK's "sdk-ts" does not change what the CLI offers: the
// initialize response and the system/init frame must have the same shape.
// Same opt-in variables as TestIntegrationRealCLI.
func TestIntegrationEntrypointParity(t *testing.T) {
	bin, mock := os.Getenv("CLAUDE_INTEGRATION_BIN"), os.Getenv("CLAUDE_INTEGRATION_MOCK_URL")
	if bin == "" || mock == "" {
		t.Skip("set CLAUDE_INTEGRATION_BIN and CLAUDE_INTEGRATION_MOCK_URL to run")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	probe := func(entrypoint string) (initKeys, sysKeys, commands, tools string) {
		dir := t.TempDir()
		cwd := filepath.Join(dir, "cwd")
		_ = os.MkdirAll(cwd, 0o755)
		q, err := Start(ctx, Options{
			PathToClaudeCodeExecutable: bin, Cwd: cwd, Entrypoint: entrypoint,
			Env: map[string]string{"PATH": os.Getenv("PATH"), "HOME": dir, "CLAUDE_CONFIG_DIR": filepath.Join(dir, "config"),
				"ANTHROPIC_API_KEY": "sk-ant-mock-not-a-real-key", "ANTHROPIC_BASE_URL": mock,
				"DISABLE_AUTOUPDATER": "1", "DISABLE_TELEMETRY": "1", "DISABLE_ERROR_REPORTING": "1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"},
			Model: "haiku", SettingSources: []string{}, SystemPrompt: PresetSystemPrompt(""),
			ExtraArgs: []ExtraArg{Flag("no-session-persistence")}, InitializeTimeout: 60 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer q.Close()
		init, err := q.Initialized(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var im map[string]json.RawMessage
		_ = json.Unmarshal(init, &im)
		var cmds []struct{ Name string }
		_ = json.Unmarshal(im["commands"], &cmds)
		var names []string
		for _, c := range cmds {
			names = append(names, c.Name)
		}
		sort.Strings(names)
		_ = q.SendText("SCENARIO:text hi")
		for m := range q.Messages() {
			if m.Type == "system" && m.Subtype == "init" {
				var sm map[string]json.RawMessage
				_ = json.Unmarshal(m.Raw, &sm)
				sysKeys = keys(sm)
				tools = string(sm["tools"])
			}
			if m.Type == "result" {
				break
			}
		}
		return keys(im), sysKeys, strings.Join(names, ","), tools
	}

	ai, as, ac, at := probe("sdk-ts")
	bi, bs, bc, bt := probe("sdk-go")
	for _, d := range []struct{ what, a, b string }{
		{"initialize response keys", ai, bi}, {"system/init keys", as, bs}, {"slash commands", ac, bc}, {"tools", at, bt},
	} {
		if d.a != d.b {
			t.Errorf("%s differ between sdk-ts and sdk-go:\n sdk-ts: %s\n sdk-go: %s", d.what, d.a, d.b)
		}
	}
	t.Logf("identical: %d initialize keys, %d system/init keys, %d commands", strings.Count(ai, ",")+1, strings.Count(as, ",")+1, strings.Count(ac, ",")+1)
}

func keys(m map[string]json.RawMessage) string {
	var k []string
	for name := range m {
		k = append(k, name)
	}
	sort.Strings(k)
	return strings.Join(k, ",")
}
