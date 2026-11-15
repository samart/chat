package claude

import "regexp"

// redactSecrets masks credentials in stderr before it is kept or shown.
//
// The SDK redacts the stderr tail it attaches to exit errors (bearer tokens,
// Anthropic keys, AWS keys, GitHub tokens, JWTs and more). This is a smaller
// set of the same idea, not a byte-exact port: treat the output as "safer to
// display", not as proof that no secret remains.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{8,}`),
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/\-]{12,}=*`),
	regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\b`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password|authorization)["']?\s*[:=]\s*["']?)[^\s"',;]{8,}`),
}

func redactSecrets(b []byte) []byte {
	out := b
	for i, re := range secretPatterns {
		switch i {
		case 1, 6:
			out = re.ReplaceAll(out, []byte("${1}[REDACTED]"))
		default:
			out = re.ReplaceAll(out, []byte("[REDACTED]"))
		}
	}
	return out
}
