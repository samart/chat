//go:build !windows

package claude

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Regression: Close must terminate a child that never reads stdin, even while
// a Write is blocked on the full pipe.
func TestCloseWhileWriteBlocked(t *testing.T) {
	tr := NewProcessTransport(ProcessConfig{Path: "/bin/sleep", Args: []string{"30"}, Env: os.Environ(),
		CloseGrace: 200 * time.Millisecond, KillGrace: 200 * time.Millisecond})
	if err := tr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	big := []byte(`"` + strings.Repeat("x", 1<<20) + `"`)
	go func() { _ = tr.Write(big) }() // fills the pipe and blocks
	time.Sleep(150 * time.Millisecond)
	done := make(chan struct{})
	go func() { _ = tr.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close hung while a Write was blocked")
	}
	if err := tr.Wait(); err == nil || !strings.Contains(err.Error(), "terminated by signal SIGTERM") {
		t.Fatalf("want SIGTERM exit error, got %v", err)
	}
}

// Regression: output written just before exit must not be lost when the
// process is reaped while a slow consumer is still reading.
func TestNoOutputLossOnExit(t *testing.T) {
	for i := 0; i < 20; i++ {
		tr := NewProcessTransport(ProcessConfig{Path: "/bin/sh", Env: os.Environ(), CloseGrace: 3 * time.Second,
			Args: []string{"-c", `cat >/dev/null; i=0; while [ $i -lt 2000 ]; do echo "{\"n\":$i}"; i=$((i+1)); done`}})
		if err := tr.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		go func() { _ = tr.Close() }()
		n := 0
		for {
			if n%50 == 0 {
				time.Sleep(time.Millisecond) // a slow consumer
			}
			if _, err := tr.ReadLine(); err != nil {
				break
			}
			n++
		}
		if n != 2000 {
			t.Fatalf("run %d: read %d of 2000 lines", i, n)
		}
	}
}

func TestExitErrorText(t *testing.T) {
	tr := NewProcessTransport(ProcessConfig{Path: "/bin/sh", Env: os.Environ(),
		Args: []string{"-c", `echo "boom sk-ant-api03-SECRETSECRETSECRET" >&2; exit 3`}})
	if err := tr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := tr.ReadLine(); err != nil {
			break
		}
	}
	err := tr.Wait()
	want := "Claude Code process exited with code 3. stderr: boom [REDACTED]"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v\nwant %s", err, want)
	}
}

func TestWriteAfterEndInputIsDropped(t *testing.T) {
	tr := NewProcessTransport(ProcessConfig{Path: "/bin/cat", Env: os.Environ()})
	if err := tr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := tr.Write([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if line, err := tr.ReadLine(); err != nil || string(line) != `{"a":1}` {
		t.Fatalf("echo: %q %v", line, err)
	}
	_ = tr.EndInput()
	if err := tr.Write([]byte(`{"b":2}`)); err != nil {
		t.Fatalf("write after EndInput should be dropped silently, got %v", err)
	}
	if err := tr.Wait(); err != nil {
		t.Fatal(err)
	}
}
