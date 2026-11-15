package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// replayTransport feeds recorded CLI output to the client and captures what
// the client writes. Recordings come from verification/run-scenario.mjs:
// one JSON record per line, {"dir":"in"|"out"|"harness","msg":{…}}, where
// "in" is CLI stdout.
type replayTransport struct {
	lines   chan []byte
	mu      sync.Mutex
	written []string
	exited  chan struct{}
	once    sync.Once
}

func newReplay(inbound [][]byte) *replayTransport {
	r := &replayTransport{lines: make(chan []byte, len(inbound)), exited: make(chan struct{})}
	for _, l := range inbound {
		r.lines <- l
	}
	return r
}
func (r *replayTransport) Start(context.Context) error { return nil }
func (r *replayTransport) Write(line []byte) error {
	r.mu.Lock()
	r.written = append(r.written, string(line))
	r.mu.Unlock()
	return nil
}
func (r *replayTransport) ReadLine() ([]byte, error) {
	select {
	case l := <-r.lines:
		return l, nil
	case <-r.exited:
		return nil, io.EOF
	}
}
func (r *replayTransport) EndInput() error         { return nil }
func (r *replayTransport) Close() error            { r.once.Do(func() { close(r.exited) }); return nil }
func (r *replayTransport) Wait() error             { <-r.exited; return nil }
func (r *replayTransport) Exited() <-chan struct{} { return r.exited }
func (r *replayTransport) StderrTail() string      { return "" }
func (r *replayTransport) PID() int                { return 0 }

func recordingsDir(version string) string {
	return filepath.Join("..", "..", "verification", "runs", "v"+version, "mock")
}

// TestReplayRecordedTranscripts runs every recorded scenario of the target
// CLI version through the client and checks the contract a host relies on:
// every non-control frame is delivered once, in order and byte-identical;
// every CLI->host control request is answered (or deliberately left
// unanswered); nothing crashes on frames this client has no code for.
func TestReplayRecordedTranscripts(t *testing.T) {
	dir := recordingsDir(TargetCLIVersion)
	scenarios, _ := filepath.Glob(filepath.Join(dir, "*", "transcript.ndjson"))
	if len(scenarios) == 0 {
		t.Skipf("no recordings under %s", dir)
	}
	for _, path := range scenarios {
		name := filepath.Base(filepath.Dir(path))
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			var inbound [][]byte
			var wantDeliver []string
			wantReply := map[string]string{} // request_id -> subtype
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 1<<20), 256<<20)
			for sc.Scan() {
				var rec struct {
					Dir string          `json:"dir"`
					Msg json.RawMessage `json:"msg"`
				}
				if json.Unmarshal(sc.Bytes(), &rec) != nil || rec.Dir != "in" || len(rec.Msg) == 0 {
					continue
				}
				var h frameHeader
				if json.Unmarshal(rec.Msg, &h) != nil {
					continue
				}
				inbound = append(inbound, rec.Msg)
				switch h.Type {
				case "control_response", "control_cancel_request", "keep_alive", "transcript_mirror":
				case "control_request":
					wantReply[h.RequestID] = subtypeOf(h.Request)
				default:
					if h.Type == "system" && h.Subtype == "session_state_changed" && h.SDKHostOnly {
						continue
					}
					wantDeliver = append(wantDeliver, string(rec.Msg))
				}
			}
			if len(inbound) == 0 {
				t.Skip("empty recording")
			}

			tr := newReplay(inbound)
			opts := Options{
				CanUseTool: func(_ context.Context, req PermissionRequest) (*PermissionResult, error) {
					return Allow(req.Input), nil
				},
				SdkMcpServers: []SdkMcpServer{{Name: "probe_sdk", Handle: func(_ context.Context, m json.RawMessage) (json.RawMessage, error) {
					var p struct {
						ID json.RawMessage `json:"id"`
					}
					_ = json.Unmarshal(m, &p)
					if len(p.ID) == 0 {
						return nil, nil
					}
					return json.RawMessage(`{"jsonrpc":"2.0","id":` + string(p.ID) + `,"result":{}}`), nil
				}}},
			}
			q, err := StartWithTransport(context.Background(), opts, tr)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			deadline := time.After(20 * time.Second)
			for len(got) < len(wantDeliver) {
				select {
				case m, ok := <-q.Messages():
					if !ok {
						t.Fatalf("stream ended after %d of %d frames: %v", len(got), len(wantDeliver), q.Err())
					}
					got = append(got, string(m.Raw))
				case <-deadline:
					t.Fatalf("timed out after %d of %d frames", len(got), len(wantDeliver))
				}
			}
			for i := range wantDeliver {
				if got[i] != wantDeliver[i] {
					t.Fatalf("frame %d differs:\n got  %.200s\n want %.200s", i, got[i], wantDeliver[i])
				}
			}
			time.Sleep(100 * time.Millisecond) // let reply goroutines finish
			q.Close()
			if err := q.Err(); err != nil && !errors.Is(err, ErrClosed) {
				t.Fatalf("unexpected error: %v", err)
			}

			tr.mu.Lock()
			written := strings.Join(tr.written, "\n")
			tr.mu.Unlock()
			for id, sub := range wantReply {
				replied := strings.Contains(written, `"request_id":"`+id+`"`)
				switch sub {
				case "request_user_dialog", "remote_tool_call", "remote_plumbing_call", "remote_tools_probe", "remote_tools_reannounce":
					if replied {
						t.Errorf("%s (%s) must be left unanswered", id, sub)
					}
				default:
					if !replied {
						t.Errorf("no reply written for %s (%s)", id, sub)
					}
				}
			}
			if !strings.HasPrefix(tr.written[0], `{"request_id":"`) || !strings.Contains(tr.written[0], `"request":{"subtype":"initialize"`) {
				t.Errorf("first frame must be initialize: %.120s", tr.written[0])
			}
		})
	}
}
