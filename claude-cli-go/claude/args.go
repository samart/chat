package claude

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// buildArgs turns Options into the CLI argument list, in the SDK's order and
// encoding (research/parity/transport.md §1.2):
//
//   - argv always starts: --output-format stream-json --verbose --input-format stream-json
//   - every valued built-in flag is ONE element, "--flag=value"
//   - --mcp-config and extra args are two elements, "--key" "value", unless
//     the value starts with "-" and is longer than one character
func buildArgs(o *Options) ([]string, error) {
	argv := []string{"--output-format", "stream-json", "--verbose", "--input-format", "stream-json"}
	flag := func(name, value string) { argv = append(argv, name+"="+value) }

	// 1. thinking (Thinking wins over MaxThinkingTokens)
	th := o.Thinking
	if th == nil && o.MaxThinkingTokens != nil {
		if *o.MaxThinkingTokens == 0 {
			th = &ThinkingConfig{Type: "disabled"}
		} else {
			th = &ThinkingConfig{Type: "enabled", BudgetTokens: o.MaxThinkingTokens}
		}
	}
	if th != nil {
		switch th.Type {
		case "enabled":
			if th.BudgetTokens == nil {
				argv = append(argv, "--thinking=adaptive")
			} else {
				flag("--max-thinking-tokens", strconv.Itoa(*th.BudgetTokens))
			}
		case "disabled":
			argv = append(argv, "--thinking=disabled")
		case "adaptive":
			argv = append(argv, "--thinking=adaptive")
		default:
			return nil, fmt.Errorf("claude: unknown thinking type %q", th.Type)
		}
		if th.Type != "disabled" && th.Display != "" {
			flag("--thinking-display", th.Display)
		}
	}
	if o.Effort != "" {
		flag("--effort", o.Effort)
	}
	if o.MaxTurns != 0 {
		flag("--max-turns", strconv.Itoa(o.MaxTurns))
	}
	if o.MaxBudgetUSD != nil {
		flag("--max-budget-usd", jsNumber(*o.MaxBudgetUSD))
	}
	if o.TaskBudgetTotal != 0 {
		flag("--task-budget", strconv.Itoa(o.TaskBudgetTotal))
	}
	if o.Model != "" {
		flag("--model", o.Model)
	}
	if o.Agent != "" {
		flag("--agent", o.Agent)
	}
	if len(o.Betas) > 0 {
		flag("--betas", strings.Join(o.Betas, ","))
	}
	if len(o.JSONSchema) > 0 {
		flag("--json-schema", string(o.JSONSchema))
	}
	if o.DebugFile != "" {
		flag("--debug-file", o.DebugFile)
	} else if o.Debug {
		argv = append(argv, "--debug")
	}
	// 11. permission prompt tool
	if o.CanUseTool != nil {
		if o.PermissionPromptToolName != "" {
			return nil, errors.New("canUseTool callback cannot be used with permissionPromptToolName. Please use one or the other.")
		}
		argv = append(argv, "--permission-prompt-tool=stdio")
	} else if o.PermissionPromptToolName != "" {
		flag("--permission-prompt-tool", o.PermissionPromptToolName)
	}
	if o.PermissionPrompts != "" {
		flag("--permission-prompts", o.PermissionPrompts)
	}
	if o.Continue {
		argv = append(argv, "--continue")
	}
	if o.Resume != "" {
		flag("--resume", o.Resume)
	}
	// 16. allowed tools, with skills folded in
	allowed := append([]string(nil), o.AllowedTools...)
	if o.Skills != nil {
		var add []string
		if o.Skills.All {
			add = []string{"Skill"}
		} else {
			for _, n := range o.Skills.Names {
				if err := validateSkillName(n); err != nil {
					return nil, err
				}
				add = append(add, "Skill("+n+")")
			}
		}
		have := map[string]bool{}
		for _, a := range allowed {
			have[a] = true
		}
		for _, a := range add {
			if !have[a] {
				allowed = append(allowed, a)
				have[a] = true
			}
		}
	}
	if len(allowed) > 0 {
		flag("--allowedTools", strings.Join(allowed, ","))
	}
	if len(o.DisallowedTools) > 0 {
		flag("--disallowedTools", strings.Join(o.DisallowedTools, ","))
	}
	if o.Tools != nil {
		if o.Tools.Preset {
			argv = append(argv, "--tools=default")
		} else {
			flag("--tools", strings.Join(o.Tools.List, ",")) // [] -> "--tools="
		}
	}
	// 19. external MCP servers only; SDK servers never appear here
	if len(o.McpServers) > 0 {
		cfg, err := mcpConfigJSON(o.McpServers)
		if err != nil {
			return nil, err
		}
		argv = append(argv, twoOrOne("--mcp-config", cfg)...)
	}
	if o.SettingSources != nil {
		flag("--setting-sources", strings.Join(o.SettingSources, ",")) // [] -> "--setting-sources="
	}
	if o.StrictMcpConfig {
		argv = append(argv, "--strict-mcp-config")
	}
	if o.PermissionMode != "" {
		flag("--permission-mode", o.PermissionMode)
	}
	if o.AllowDangerouslySkipPermissions {
		argv = append(argv, "--allow-dangerously-skip-permissions")
	}
	if o.FallbackModel != "" {
		if o.FallbackModel == o.Model {
			return nil, errors.New("Fallback model cannot be the same as the main model. Please specify a different model for fallbackModel option.")
		}
		flag("--fallback-model", o.FallbackModel)
	}
	if o.IncludeHookEvents {
		argv = append(argv, "--include-hook-events")
	}
	if o.IncludePartialMessages {
		argv = append(argv, "--include-partial-messages")
	}
	if o.ProjectConfigRoot != nil {
		flag("--project-config-root", *o.ProjectConfigRoot)
	}
	for _, d := range o.AdditionalDirectories {
		flag("--add-dir", d)
	}
	for _, p := range o.Plugins {
		if p.SkipMcpDiscovery {
			flag("--plugin-dir-no-mcp", p.Path)
		} else {
			flag("--plugin-dir", p.Path)
		}
	}
	if o.ForkSession {
		argv = append(argv, "--fork-session")
	}
	if o.ResumeSessionAt != "" {
		flag("--resume-session-at", o.ResumeSessionAt)
	}
	if o.ResumeDropsTurn != nil {
		flag("--resume-drops-turn", strconv.FormatBool(*o.ResumeDropsTurn))
	}
	if o.SessionID != "" {
		flag("--session-id", o.SessionID)
	}
	if o.NoSessionPersistence {
		argv = append(argv, "--no-session-persistence")
	}
	// 37. extra args, then settings last (unless the caller already gave it)
	extras := append([]ExtraArg(nil), o.ExtraArgs...)
	if o.Settings != "" {
		replaced := false
		for i := range extras {
			if extras[i].Key == "settings" {
				v := o.Settings
				extras[i].Value, replaced = &v, true
			}
		}
		if !replaced {
			extras = append(extras, Arg("settings", o.Settings))
		}
	}
	for _, e := range extras {
		if e.Value == nil {
			argv = append(argv, "--"+e.Key)
		} else {
			argv = append(argv, twoOrOne("--"+e.Key, *e.Value)...)
		}
	}
	return argv, nil
}

// twoOrOne is the SDK's zE helper: normally "--key", "value"; one element
// "--key=value" when the value starts with "-" and is longer than one char.
func twoOrOne(name, value string) []string {
	if len(value) > 1 && strings.HasPrefix(value, "-") {
		return []string{name + "=" + value}
	}
	return []string{name, value}
}

// mcpConfigJSON renders {"mcpServers":{name:config,…}} in the given order.
func mcpConfigJSON(servers []McpServerConfig) (string, error) {
	var b strings.Builder
	b.WriteString(`{"mcpServers":{`)
	for i, s := range servers {
		if s.Name == "" || len(s.Config) == 0 {
			return "", errors.New("claude: McpServerConfig needs Name and Config")
		}
		if i > 0 {
			b.WriteByte(',')
		}
		name, err := marshalJSON(s.Name)
		if err != nil {
			return "", err
		}
		b.Write(name)
		b.WriteByte(':')
		b.Write(s.Config)
	}
	b.WriteString("}}")
	return b.String(), nil
}

// validateSkillName applies the SDK's checks: no parentheses, commas,
// control characters or "*", and no leading "/".
func validateSkillName(n string) error {
	if n == "" || strings.HasPrefix(n, "/") || strings.ContainsAny(n, "(),*") {
		return fmt.Errorf("claude: invalid skill name %q", n)
	}
	for _, r := range n {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("claude: invalid skill name %q", n)
		}
	}
	return nil
}

// jsNumber formats a float the way JavaScript's Number.prototype.toString
// does, so flags such as --max-budget-usd match the SDK byte for byte:
// plain decimals for 1e-6 <= |x| < 1e21, otherwise an exponent without a
// zero-padded exponent ("1e-7", "1e+21").
func jsNumber(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	s := strconv.FormatFloat(f, 'e', -1, 64) // e.g. 1e-07, 1.5e+21
	mant, exp, _ := strings.Cut(s, "e")
	sign := exp[:1]
	digits := strings.TrimLeft(exp[1:], "0")
	if digits == "" {
		digits = "0"
	}
	return mant + "e" + sign + digits
}

// resolveCommand applies the SDK's native-vs-JS rule (transport.md §2).
func resolveCommand(o *Options, argv []string) (path string, args []string, err error) {
	p := o.PathToClaudeCodeExecutable
	if p == "" {
		return "", nil, errors.New("claude: Options.PathToClaudeCodeExecutable is required (this client bundles no CLI binary)")
	}
	js := false
	for _, ext := range []string{".js", ".mjs", ".tsx", ".ts", ".jsx"} {
		if strings.HasSuffix(p, ext) {
			js = true
		}
	}
	if !js {
		return p, append(append([]string(nil), o.ExecutableArgs...), argv...), nil
	}
	exe := o.Executable
	if exe == "" {
		exe = "node"
	}
	args = append(append([]string(nil), o.ExecutableArgs...), p)
	return exe, append(args, argv...), nil
}

// buildEnv produces the child environment (transport.md §3).
func buildEnv(o *Options) []string {
	env := map[string]string{}
	inherited := o.Env == nil
	if inherited {
		for _, kv := range os.Environ() {
			if k, v, ok := strings.Cut(kv, "="); ok && k != "" {
				env[k] = v
			}
		}
	} else {
		for k, v := range o.Env {
			env[k] = v
		}
	}
	hasCI := func(name string) bool {
		for k := range env {
			if strings.EqualFold(k, name) {
				return true
			}
		}
		return false
	}
	delCI := func(name string) {
		for k := range env {
			if strings.EqualFold(k, name) {
				delete(env, k)
			}
		}
	}

	if env["CLAUDE_CODE_ENTRYPOINT"] == "" {
		env["CLAUDE_CODE_ENTRYPOINT"] = firstNonEmpty(o.Entrypoint, DefaultEntrypoint)
	}
	// The SDK overwrites this in its own process env before copying it, so an
	// inherited value never survives; a caller-supplied env keeps its value.
	if inherited || env["CLAUDE_AGENT_SDK_VERSION"] == "" {
		env["CLAUDE_AGENT_SDK_VERSION"] = firstNonEmpty(o.SDKVersion, TargetSDKVersion)
	}
	if o.EnableFileCheckpointing {
		env["CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING"] = "true"
	}
	// Tells the CLI to emit system/session_state_changed frames.
	if !hasCI("CLAUDE_CODE_SDK_READS_SESSION_STATE") {
		env["CLAUDE_CODE_SDK_READS_SESSION_STATE"] = "1"
	}
	if o.AskUserQuestionPreviewFormat != "" {
		env["CLAUDE_CODE_QUESTION_PREVIEW_FORMAT"] = o.AskUserQuestionPreviewFormat
	}
	if inherited {
		// The SDK scrubs these from an inherited environment unless the
		// matching option enables them; this client has no such options.
		delCI("CLAUDE_CODE_QUESTION_EXTENDED")
		delCI("CLAUDE_CODE_QUESTION_OPTIONAL_DESCRIPTIONS")
		delCI("CLAUDE_CODE_AGENT_PROGRESS_SUMMARIES")
	}
	delete(env, "NODE_OPTIONS")
	if envTruthy(env["DEBUG_CLAUDE_AGENT_SDK"]) {
		env["DEBUG"] = "1"
	} else {
		delete(env, "DEBUG")
	}

	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
