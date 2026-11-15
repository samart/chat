package claude

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Transport moves NDJSON lines between the host and one claude CLI process.
// It mirrors the SDK's ProcessTransport: one child process, stdin for
// host->CLI lines, stdout for CLI->host lines, stderr kept for diagnostics.
//
// Write and EndInput are safe for concurrent use. ReadLine is called only
// from the single reader goroutine owned by Query.
type Transport interface {
	// Start launches the process. It must be called exactly once.
	Start(ctx context.Context) error
	// Write sends one JSON value. The transport appends the trailing '\n'.
	// After EndInput the write is silently dropped, as in the SDK.
	Write(line []byte) error
	// ReadLine returns the next stdout line without its line terminator.
	// It returns io.EOF once stdout is closed.
	ReadLine() ([]byte, error)
	// EndInput closes stdin: "no more input". It never blocks, even if a
	// Write is stuck on a full pipe.
	EndInput() error
	// Close shuts the process down the way the SDK does: stdin EOF, then
	// SIGTERM if still alive after CloseGrace (2 s), then SIGKILL after
	// KillGrace (5 s). It returns once the process has exited. Idempotent.
	Close() error
	// Wait blocks until the process has exited and stderr has drained, and
	// returns the exit error (nil for exit code 0).
	Wait() error
	// Exited is closed when the process has exited.
	Exited() <-chan struct{}
	// StderrTail returns the redacted tail of stderr (at most 2048 bytes).
	StderrTail() string
	// PID returns the child process id, or 0 before Start.
	PID() int
}

// ProcessConfig is the fully resolved launch description built from Options.
type ProcessConfig struct {
	Path string   // executable: the native claude binary, or node/bun for a JS entrypoint
	Args []string // argv after Path
	Dir  string   // working directory
	Env  []string // complete environment (KEY=VALUE)

	// MaxLineBytes caps one stdout line. 0 means no limit, which is the SDK's
	// behaviour (tool output and images make lines large).
	MaxLineBytes int
	// OnStderr, when set, receives each raw stderr chunk (Options.Stderr).
	OnStderr func(string)
	// CloseGrace is the wait between stdin EOF and SIGTERM. Default 2 s (SDK).
	CloseGrace time.Duration
	// KillGrace is the wait between SIGTERM and SIGKILL. Default 5 s (SDK).
	KillGrace time.Duration
	// OnSpawned is called with the pid right after the process starts.
	OnSpawned func(pid int)
}

// SDK constants (sdk.mjs §4.4–4.6; see research/parity/transport.md).
const (
	defaultCloseGrace    = 2000 * time.Millisecond
	defaultKillGrace     = 5000 * time.Millisecond
	stderrDrainAfterExit = 200 * time.Millisecond
	stderrTailLimit      = 2048
	// stdoutDrainAfterExit bounds how long ReadLine may keep reading after the
	// process exited. It only matters if a grandchild inherited stdout.
	stdoutDrainAfterExit = 2 * time.Second
)

type processTransport struct {
	cfg ProcessConfig

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdoutR *os.File
	stdout  *bufio.Reader
	stderrR *os.File

	writeMu    sync.Mutex
	inputEnded chan struct{} // closed by EndInput
	inputOnce  sync.Once

	stderrMu   sync.Mutex
	stderrTail []byte
	stderrDone chan struct{}

	exited   chan struct{} // closed when cmd.Wait returned
	finished chan struct{} // closed when exit + stderr drain are complete
	waitErr  error

	closeOnce sync.Once
	closeDone chan struct{}

	abortMu sync.Mutex
	aborted bool
}

// NewProcessTransport returns a Transport that runs the CLI described by cfg.
func NewProcessTransport(cfg ProcessConfig) Transport {
	if cfg.CloseGrace <= 0 {
		cfg.CloseGrace = defaultCloseGrace
	}
	if cfg.KillGrace <= 0 {
		cfg.KillGrace = defaultKillGrace
	}
	return &processTransport{
		cfg:        cfg,
		inputEnded: make(chan struct{}),
		stderrDone: make(chan struct{}),
		exited:     make(chan struct{}),
		finished:   make(chan struct{}),
		closeDone:  make(chan struct{}),
	}
}

func (t *processTransport) Start(ctx context.Context) error {
	if t.cmd != nil {
		return errors.New("claude: transport already started")
	}
	// Not exec.CommandContext: cancellation goes through Close so the child
	// gets stdin EOF and SIGTERM before any SIGKILL, as the SDK does.
	cmd := exec.Command(t.cfg.Path, t.cfg.Args...)
	cmd.Dir = t.cfg.Dir
	cmd.Env = t.cfg.Env
	setProcAttrs(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	// stdout and stderr use pipes we own. With cmd.StdoutPipe, cmd.Wait closes
	// the read side as soon as the process exits, which loses any output the
	// reader has not consumed yet (typically the final "result" frame).
	outR, outW, err := os.Pipe()
	if err != nil {
		return err
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return err
	}
	cmd.Stdout, cmd.Stderr = outW, errW

	err = cmd.Start()
	// The child has its own copies of the write ends; ours must be closed so
	// the readers see EOF when the child exits.
	outW.Close()
	errW.Close()
	if err != nil {
		outR.Close()
		errR.Close()
		return &CLINotFoundError{Path: t.cfg.Path, Err: err}
	}
	t.cmd, t.stdin = cmd, stdin
	t.stdoutR, t.stderrR = outR, errR
	t.stdout = bufio.NewReaderSize(outR, 256<<10)
	if t.cfg.OnSpawned != nil {
		t.cfg.OnSpawned(cmd.Process.Pid)
	}

	go t.pumpStderr()
	go t.reap()
	go func() {
		select {
		case <-ctx.Done():
			t.abortMu.Lock()
			t.aborted = true
			t.abortMu.Unlock()
			_ = t.Close()
		case <-t.finished:
		}
	}()
	return nil
}

// reap waits for the process, then gives stderr 200 ms to drain (the SDK's
// "sdk-exit-after-stderr-drained") so exit errors carry the full tail.
func (t *processTransport) reap() {
	t.waitErr = t.cmd.Wait()
	close(t.exited)
	select {
	case <-t.stderrDone:
	case <-time.After(stderrDrainAfterExit):
		t.stderrR.Close()
	}
	close(t.finished)
	// If something else still holds stdout open, unblock the reader.
	time.AfterFunc(stdoutDrainAfterExit, func() { t.stdoutR.Close() })
}

func (t *processTransport) pumpStderr() {
	defer close(t.stderrDone)
	buf := make([]byte, 32<<10)
	for {
		n, err := t.stderrR.Read(buf)
		if n > 0 {
			t.stderrMu.Lock()
			t.stderrTail = append(t.stderrTail, buf[:n]...)
			if len(t.stderrTail) > 2*stderrTailLimit {
				t.stderrTail = tailBytes(redactSecrets(t.stderrTail), stderrTailLimit)
			}
			t.stderrMu.Unlock()
			if t.cfg.OnStderr != nil {
				t.cfg.OnStderr(string(buf[:n]))
			}
		}
		if err != nil {
			return
		}
	}
}

func (t *processTransport) Write(line []byte) error {
	if t.stdin == nil {
		return errors.New("claude: ProcessTransport is not ready for writing")
	}
	if bytes.IndexByte(line, '\n') >= 0 {
		// encoding/json never emits raw newlines, so this is a caller bug.
		return errors.New("claude: NDJSON line contains a newline")
	}
	select {
	case <-t.inputEnded:
		return nil // SDK: writes after stdin ended are silently dropped
	default:
	}
	select {
	case <-t.exited:
		return t.exitError("Cannot write to terminated process")
	default:
	}
	buf := make([]byte, 0, len(line)+1)
	buf = append(append(buf, line...), '\n')

	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if _, err := t.stdin.Write(buf); err != nil {
		select {
		case <-t.inputEnded:
			return nil // stdin was closed while this write was in flight
		default:
		}
		return &ProcessError{Op: "write stdin", ExitCode: -1, Err: err, Stderr: t.StderrTail()}
	}
	return nil
}

func (t *processTransport) ReadLine() ([]byte, error) {
	var line []byte
	for {
		chunk, isPrefix, err := t.stdout.ReadLine()
		if err != nil {
			if len(line) > 0 && errors.Is(err, io.EOF) {
				return line, nil
			}
			if errors.Is(err, os.ErrClosed) {
				return nil, io.EOF
			}
			return nil, err
		}
		line = append(line, chunk...)
		if t.cfg.MaxLineBytes > 0 && len(line) > t.cfg.MaxLineBytes {
			for isPrefix { // drain the rest so framing stays aligned
				if _, isPrefix, err = t.stdout.ReadLine(); err != nil {
					break
				}
			}
			return nil, &LineTooLongError{Limit: t.cfg.MaxLineBytes}
		}
		if !isPrefix {
			return line, nil
		}
	}
}

// EndInput closes stdin without taking the write lock: closing the pipe
// unblocks a Write that is stuck because the child stopped reading.
func (t *processTransport) EndInput() error {
	var err error
	t.inputOnce.Do(func() {
		close(t.inputEnded)
		if t.stdin != nil {
			err = t.stdin.Close()
		}
	})
	return err
}

func (t *processTransport) Wait() error {
	if t.cmd == nil {
		return errors.New("claude: transport not started")
	}
	<-t.finished
	if t.waitErr == nil {
		return nil
	}
	return t.exitError("")
}

func (t *processTransport) Exited() <-chan struct{} { return t.exited }

func (t *processTransport) waitExit(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-t.exited:
		return true
	case <-timer.C:
		return false
	}
}

func (t *processTransport) Close() error {
	t.closeOnce.Do(func() {
		defer close(t.closeDone)
		if t.cmd == nil || t.cmd.Process == nil {
			return
		}
		_ = t.EndInput()
		if t.waitExit(t.cfg.CloseGrace) {
			<-t.finished
			return
		}
		// Windows has no SIGTERM: the SDK goes straight to SIGKILL after
		// CloseGrace + KillGrace.
		if runtime.GOOS != "windows" {
			_ = t.cmd.Process.Signal(syscall.SIGTERM)
		}
		if t.waitExit(t.cfg.KillGrace) {
			<-t.finished
			return
		}
		_ = t.cmd.Process.Kill()
		<-t.finished
	})
	<-t.closeDone
	return nil
}

// exitError builds the SDK's exit error text:
//
//	Claude Code process exited with code N. stderr: <tail>
//	Claude Code process terminated by signal SIG. stderr: <tail>
func (t *processTransport) exitError(fallback string) error {
	t.abortMu.Lock()
	aborted := t.aborted
	t.abortMu.Unlock()
	if aborted {
		return &AbortError{Reason: "Claude Code process aborted by user"}
	}
	msg := fallback
	code := -1
	var ee *exec.ExitError
	select {
	case <-t.exited:
		if errors.As(t.waitErr, &ee) {
			code = ee.ExitCode()
			if sig, ok := exitSignal(ee); ok {
				msg = "Claude Code process terminated by signal " + sig
			} else {
				msg = fmt.Sprintf("Claude Code process exited with code %d", code)
			}
		} else if t.waitErr != nil && msg == "" {
			msg = "Claude Code process error: " + t.waitErr.Error()
		}
	default:
	}
	if msg == "" {
		msg = "Claude Code process error"
	}
	return &ProcessError{Message: msg, ExitCode: code, Err: t.waitErr, Stderr: t.StderrTail()}
}

func (t *processTransport) StderrTail() string {
	t.stderrMu.Lock()
	defer t.stderrMu.Unlock()
	return string(bytes.TrimSpace(tailBytes(redactSecrets(t.stderrTail), stderrTailLimit)))
}

func (t *processTransport) PID() int {
	if t.cmd == nil || t.cmd.Process == nil {
		return 0
	}
	return t.cmd.Process.Pid
}

func tailBytes(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	b = b[len(b)-n:]
	for len(b) > 0 && b[0]&0xC0 == 0x80 { // do not start mid-rune
		b = b[1:]
	}
	return append([]byte(nil), b...)
}

// LineTooLongError reports a stdout line above ProcessConfig.MaxLineBytes.
// The line is skipped; the session continues.
type LineTooLongError struct{ Limit int }

func (e *LineTooLongError) Error() string {
	return fmt.Sprintf("claude: stdout line exceeds %d bytes", e.Limit)
}
