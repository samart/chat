package claude

import (
	"errors"
	"fmt"
	"strings"
)

// ErrClosed is returned by Query methods after Close. The SDK's text for a
// pending control request in this state is "Query closed before response received".
var ErrClosed = errors.New("claude: Query closed before response received")

// CLINotFoundError means the executable could not be started.
type CLINotFoundError struct {
	Path string
	Err  error
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("Claude Code native binary not found or failed to launch at %s: %v. Set Options.PathToClaudeCodeExecutable to a valid path.", e.Path, e.Err)
}
func (e *CLINotFoundError) Unwrap() error { return e.Err }

// ProcessError is an I/O failure or an unexpected exit of the CLI process.
// Its text follows the SDK: "<message>. stderr: <redacted tail>".
type ProcessError struct {
	Message  string // e.g. "Claude Code process exited with code 1"
	Op       string // set for I/O failures, e.g. "write stdin"
	ExitCode int    // -1 when unknown or killed by a signal
	Err      error
	Stderr   string // redacted tail, at most 2048 bytes
}

func (e *ProcessError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = fmt.Sprintf("claude: %s: %v", e.Op, e.Err)
	}
	if tail := strings.TrimSpace(e.Stderr); tail != "" {
		msg += ". stderr: " + tail
	}
	return msg
}
func (e *ProcessError) Unwrap() error { return e.Err }

// ErrorResultError replaces the exit error when the CLI exited non-zero after
// emitting an error result: "Claude Code returned an error result: <text>".
type ErrorResultError struct {
	Text string
	Exit error
}

func (e *ErrorResultError) Error() string { return "Claude Code returned an error result: " + e.Text }
func (e *ErrorResultError) Unwrap() error { return e.Exit }

// ControlError is a control_response with subtype "error".
type ControlError struct {
	Subtype   string // the request subtype that failed
	Message   string
	ErrorCode string // wire: error_code, e.g. "unsupported_request"
}

func (e *ControlError) Error() string { return e.Message }

// IsUnsupported reports whether the CLI rejected the request because it does
// not know the subtype. Use it for feature detection: try an undocumented
// subtype once and remember the answer.
func (e *ControlError) IsUnsupported() bool {
	return e.ErrorCode == "unsupported_request" ||
		strings.Contains(e.Message, "Unsupported control request subtype")
}

// IsUnsupported is a convenience wrapper around ControlError.IsUnsupported.
func IsUnsupported(err error) bool {
	var ce *ControlError
	return errors.As(err, &ce) && ce.IsUnsupported()
}

// codedError lets a host handler attach an error_code to its error reply.
type codedError struct {
	msg  string
	code string
}

func (e *codedError) Error() string { return e.msg }

// ErrorWithCode returns an error that is sent to the CLI with an error_code.
func ErrorWithCode(message, code string) error { return &codedError{msg: message, code: code} }

// InitializeTimeoutError means the CLI did not answer the initialize request
// within Options.InitializeTimeout. The SDK's startup() tags the same failure
// errorClass "initialize_timeout"; query() itself has no such timeout.
type InitializeTimeoutError struct {
	Timeout string
	Stderr  string
}

func (e *InitializeTimeoutError) Error() string {
	msg := "Subprocess initialization did not complete within " + e.Timeout
	if tail := strings.TrimSpace(e.Stderr); tail != "" {
		msg += ". stderr: " + tail
	}
	return msg
}

// AbortError is returned when a request or the whole query was cancelled.
type AbortError struct{ Reason string }

func (e *AbortError) Error() string {
	if e.Reason == "" {
		return "Operation aborted"
	}
	return e.Reason
}
