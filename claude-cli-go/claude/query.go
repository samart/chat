package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Message is one frame from the CLI's stdout that is meant for the host's
// consumer: system, assistant, user, stream_event, result and anything newer.
// Control frames and keep-alives are handled inside Query and never appear.
//
// Raw is the complete JSON object exactly as the CLI wrote it. Decode it with
// the wire package, or forward it unchanged (for example over gRPC), so
// fields added by newer CLIs are never lost.
type Message struct {
	Type            string
	Subtype         string
	SessionID       string
	UUID            string
	ParentToolUseID string // non-empty: the frame belongs to that subagent tool call
	Raw             json.RawMessage
}

type frameHeader struct {
	Type            string  `json:"type"`
	Subtype         string  `json:"subtype"`
	SessionID       string  `json:"session_id"`
	UUID            string  `json:"uuid"`
	ParentToolUseID *string `json:"parent_tool_use_id"`
	RequestID       string  `json:"request_id"`
	// control_request
	Request json.RawMessage `json:"request"`
	// control_response
	Response *wireControlResponseBody `json:"response"`
	// system/session_state_changed
	State       string `json:"state"`
	SDKHostOnly bool   `json:"sdk_host_only"`
	// result
	IsError bool            `json:"is_error"`
	Result  json.RawMessage `json:"result"`
	Errors  []string        `json:"errors"`
	// keep_alive
	StartupMark string `json:"startup_mark"`
	// system/commands_changed
	Commands json.RawMessage `json:"commands"`
}

// SDK timing constants (research/parity/transport.md §5).
const (
	promptHoldGuard   = 660 * time.Second // initialize guard while a blocking hook holds prompts
	defaultRunCeiling = 600 * time.Second // max wait for "idle" after a result
	closeWaitForExit  = 2 * time.Second   // Query.Close waits this long for the process
)

// Query is one live CLI process: the Go counterpart of the object returned by
// the SDK's query(). Create it with Start (streaming input, for a chat host)
// or Prompt (single prompt).
type Query struct {
	opts   Options
	tr     Transport
	router *controlRouter

	ctx    context.Context
	cancel context.CancelFunc

	queue *messageQueue
	out   chan Message

	initDone chan struct{}
	initResp json.RawMessage
	initErr  error
	hold     bool // prompts wait for a successful initialize (blocking hook)

	hooks map[string]HookFunc
	mcp   map[string]SdkMcpServer

	mu          sync.Mutex
	err         error
	commands    json.RawMessage // latest system/commands_changed
	startupMark string

	// Error-result substitution state (transport-0.3.311.md item 63).
	lastErrorText       string
	textBeforeModelLess string
	modelLessEnded      bool

	// Run tracking: decides when stdin may be closed (items 41-43).
	sessionState   string
	resultReceived bool
	runEnded       bool
	singleTurn     bool
	sentMessages   int
	inputRequested bool // EndInput called; close stdin when the run ends
	ceiling        *time.Timer

	closeOnce sync.Once
	closed    chan struct{}
	done      chan struct{} // reader finished
}

// Start launches the CLI in streaming-input mode: the process stays up across
// turns, user messages are written with Send, and stdin stays open until
// EndInput or Close. This is the mode a chat host uses.
//
// As in the SDK, the initialize control request is the first frame written
// and Start does not wait for its response; call Initialized to wait.
func Start(ctx context.Context, opts Options) (*Query, error) {
	return start(ctx, opts, nil, false)
}

// Prompt runs a single prompt, like the SDK's query() with a string prompt:
// the prompt is written after initialize, and stdin is closed when the run
// ends, so Messages closes after the final result.
func Prompt(ctx context.Context, prompt string, opts Options) (*Query, error) {
	q, err := start(ctx, opts, nil, true)
	if err != nil {
		return nil, err
	}
	if err := q.SendText(prompt); err != nil {
		q.Close()
		return nil, err
	}
	return q, nil
}

// StartWithTransport is Start with a caller-supplied Transport. Tests use it
// to replay recorded transcripts; a host can use it to run the CLI elsewhere
// (a container, ssh). The transport must not have been started.
func StartWithTransport(ctx context.Context, opts Options, tr Transport) (*Query, error) {
	return start(ctx, opts, tr, false)
}

func start(ctx context.Context, opts Options, tr Transport, singleTurn bool) (*Query, error) {
	if len(opts.SupportedDialogKinds) > 0 && opts.OnUserDialog == nil {
		return nil, errors.New("supportedDialogKinds requires onUserDialog")
	}
	if tr == nil {
		argv, err := buildArgs(&opts)
		if err != nil {
			return nil, err
		}
		path, args, err := resolveCommand(&opts, argv)
		if err != nil {
			return nil, err
		}
		tr = NewProcessTransport(ProcessConfig{
			Path: path, Args: args, Dir: opts.Cwd, Env: buildEnv(&opts),
			MaxLineBytes: opts.MaxLineBytes, OnStderr: opts.Stderr, OnSpawned: opts.OnProcessSpawned,
		})
	}

	qctx, cancel := context.WithCancel(context.Background())
	q := &Query{
		opts:       opts,
		tr:         tr,
		ctx:        qctx,
		cancel:     cancel,
		queue:      newMessageQueue(),
		out:        make(chan Message),
		initDone:   make(chan struct{}),
		hooks:      map[string]HookFunc{},
		mcp:        map[string]SdkMcpServer{},
		singleTurn: singleTurn,
		closed:     make(chan struct{}),
		done:       make(chan struct{}),
	}
	q.router = newControlRouter(q.writeJSON)
	for _, s := range opts.SdkMcpServers {
		if s.Name == "" || s.Handle == nil {
			cancel()
			return nil, errors.New("claude: SdkMcpServer needs Name and Handle")
		}
		q.mcp[s.Name] = s
	}
	initReq, err := q.buildInitializeRequest(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := tr.Start(ctx); err != nil {
		cancel()
		return nil, err
	}
	go q.readLoop()
	go q.pump()

	// The initialize frame is written here, synchronously, so it is always
	// the first frame on stdin. Its response is awaited in the background.
	id, ch, err := q.router.begin("initialize", initReq)
	if err != nil {
		q.Close()
		return nil, err
	}
	go q.awaitInitialize(id, ch)
	return q, nil
}

func (q *Query) awaitInitialize(id string, ch chan wireControlResponseBody) {
	ctx := q.ctx
	if q.opts.InitializeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(q.ctx, q.opts.InitializeTimeout)
		defer cancel()
	} else if q.hold {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(q.ctx, promptHoldGuard)
		defer cancel()
	}
	resp, err := q.router.await(ctx, id, "initialize", ch, false)
	if err != nil {
		select {
		case <-q.closed:
			// Closed by the host while initialize was pending: not a failure.
			q.initErr = ErrClosed
			close(q.initDone)
			return
		default:
		}
		var abort *AbortError
		if errors.As(err, &abort) && q.ctx.Err() == nil {
			// Our own deadline fired, not a Close.
			d := q.opts.InitializeTimeout
			if d == 0 {
				d = promptHoldGuard
			}
			err = &InitializeTimeoutError{Timeout: strconv.FormatInt(d.Milliseconds(), 10) + "ms", Stderr: q.tr.StderrTail()}
		}
		q.initErr = err
		close(q.initDone)
		q.fail(err)
		return
	}
	if q.hold {
		// A hook asked to block on failure. Only CLIs that report
		// hook_on_failure_supported honour that (2.1.311 does not), so fail
		// loudly rather than run with a hook that silently will not block.
		var caps struct {
			Supported *bool `json:"hook_on_failure_supported"`
		}
		_ = json.Unmarshal(resp, &caps)
		if caps.Supported == nil || !*caps.Supported {
			err := errors.New("claude: this CLI does not support hook onFailure (initialize did not report hook_on_failure_supported); remove HookMatcher.OnFailure or use a newer CLI")
			q.initErr = err
			close(q.initDone)
			q.fail(err)
			return
		}
	}
	q.initResp = resp
	close(q.initDone)

	// Requests that were waiting when a previous host detached come back in
	// the initialize response and are handled as if they had just arrived.
	var pending struct {
		Permissions []wireInbound `json:"pending_permission_requests"`
		Dialogs     []wireInbound `json:"pending_user_dialog_requests"`
	}
	if json.Unmarshal(resp, &pending) == nil {
		for _, p := range pending.Permissions {
			if subtypeOf(p.Request) == "can_use_tool" {
				q.router.handleInbound(q.ctx, p.RequestID, "can_use_tool", p.Request, q.handleControlRequest)
			}
		}
		for _, p := range pending.Dialogs {
			if subtypeOf(p.Request) == "request_user_dialog" {
				q.router.handleInbound(q.ctx, p.RequestID, "request_user_dialog", p.Request, q.handleControlRequest)
			}
		}
	}
}

type wireInbound struct {
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
}

func subtypeOf(request json.RawMessage) string {
	var s struct {
		Subtype string `json:"subtype"`
	}
	_ = json.Unmarshal(request, &s)
	return s.Subtype
}

// writeJSON marshals v and writes it as one NDJSON line.
func (q *Query) writeJSON(v any) error {
	b, err := marshalJSON(v)
	if err != nil {
		return err
	}
	return q.tr.Write(b)
}

// ---- reading -----------------------------------------------------------

func (q *Query) readLoop() {
	defer close(q.done)
	var readErr error
	for {
		line, err := q.tr.ReadLine()
		if err != nil {
			var tooLong *LineTooLongError
			if errors.As(err, &tooLong) {
				continue // skip the line; framing is still aligned
			}
			if !errors.Is(err, io.EOF) {
				readErr = err
			}
			break
		}
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var h frameHeader
		if json.Unmarshal(line, &h) != nil {
			continue // SDK: non-JSON stdout lines are skipped, never fatal
		}
		q.route(h, line)
	}

	err := readErr
	if err == nil {
		// stdout closed: collect the exit status.
		if werr := q.tr.Wait(); werr != nil {
			err = werr
		}
	}
	q.mu.Lock()
	if err != nil {
		text := q.lastErrorText
		if q.modelLessEnded {
			text = q.textBeforeModelLess
		}
		var abort *AbortError
		if text != "" && !errors.As(err, &abort) {
			err = &ErrorResultError{Text: text, Exit: err}
		}
	}
	select {
	case <-q.closed:
		err = nil // closed by the host: the iterator ends normally
	default:
	}
	if q.err == nil {
		q.err = err
	}
	finalErr := q.err
	q.mu.Unlock()

	q.router.close(finalErr)
	q.queue.close()
}

func (q *Query) route(h frameHeader, line []byte) {
	switch h.Type {
	case "control_response":
		if h.Response != nil {
			q.router.deliver(*h.Response)
		}
		return
	case "control_request":
		q.router.handleInbound(q.ctx, h.RequestID, subtypeOf(h.Request), h.Request, q.handleControlRequest)
		return
	case "control_cancel_request":
		q.router.cancelInbound(h.RequestID)
		return
	case "keep_alive":
		if h.StartupMark != "" && len(h.StartupMark) <= 64 {
			q.mu.Lock()
			q.startupMark = h.StartupMark
			q.mu.Unlock()
		}
		return
	case "transcript_mirror":
		return
	}

	raw := append(json.RawMessage(nil), line...)
	switch {
	case h.Type == "system" && h.Subtype == "session_state_changed":
		q.onSessionState(h.State)
		if h.SDKHostOnly {
			return
		}
	case h.Type == "result":
		q.onResult(h, line)
	default:
		q.onOtherFrame(h)
	}

	m := Message{Type: h.Type, Subtype: h.Subtype, SessionID: h.SessionID, UUID: h.UUID, Raw: raw}
	if h.ParentToolUseID != nil {
		m.ParentToolUseID = *h.ParentToolUseID
	}
	q.queue.push(m)
}

func (q *Query) pump() {
	defer close(q.out)
	for {
		m, ok := q.queue.pop()
		if !ok {
			return
		}
		select {
		case q.out <- m:
		case <-q.closed:
			return
		}
	}
}

// Messages returns the stream of frames. The channel is closed when the CLI's
// stdout ends or the query is closed; call Err afterwards.
//
// Frames are buffered without limit between the reader and this channel, as
// in the SDK, so a slow consumer never stalls approvals or control replies.
func (q *Query) Messages() <-chan Message { return q.out }

// Err reports why Messages ended: nil after a clean exit or Close, otherwise
// a *ProcessError, *ErrorResultError, *AbortError or *InitializeTimeoutError.
func (q *Query) Err() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.err
}

func (q *Query) fail(err error) {
	q.mu.Lock()
	if q.err == nil {
		q.err = err
	}
	q.mu.Unlock()
	go q.tr.Close()
}

// ---- run tracking (when to close stdin) --------------------------------

func (q *Query) hasBidirectionalNeeds() bool {
	o := &q.opts
	return len(o.SdkMcpServers) > 0 || len(o.Hooks) > 0 || o.CanUseTool != nil ||
		o.OnElicitation != nil || o.OnUserDialog != nil
}

func (q *Query) onResult(h frameHeader, line []byte) {
	var extra struct {
		NumTurns *int `json:"num_turns"`
	}
	_ = json.Unmarshal(line, &extra)
	var resultText string
	_ = json.Unmarshal(h.Result, &resultText)

	q.mu.Lock()
	// Error-result bookkeeping.
	text := ""
	if h.IsError {
		if h.Subtype == "success" {
			text = resultText
		} else {
			var parts []string
			for _, e := range h.Errors {
				if e = strings.TrimSpace(e); e != "" {
					parts = append(parts, e)
				}
			}
			text = strings.Join(parts, "; ")
		}
	}
	q.lastErrorText = text
	q.modelLessEnded = h.Subtype == "success" && !h.IsError && extra.NumTurns != nil && *extra.NumTurns == 0 &&
		len(h.Result) > 0 && resultText == ""
	if !q.modelLessEnded {
		q.textBeforeModelLess = q.lastErrorText
	}

	// Run tracking.
	q.resultReceived = true
	end := q.sessionState == "" || q.sessionState == "idle" || !q.hasBidirectionalNeeds()
	if !end {
		q.armCeilingLocked()
	}
	q.mu.Unlock()
	if end {
		q.endRun()
	}
}

// onOtherFrame handles every delivered frame that is neither a result nor a
// session_state_changed.
func (q *Query) onOtherFrame(h frameHeader) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if h.Type == "system" && h.Subtype == "commands_changed" && len(h.Commands) > 0 {
		q.commands = append(json.RawMessage(nil), h.Commands...)
	}

	// Frames that leave the error-result state untouched.
	exempt := h.Type == "active_goal" || h.Type == "autocompact_state" ||
		(h.Type == "system" && (h.Subtype == "post_turn_summary" || h.Subtype == "task_summary" || h.Subtype == "session_metadata"))
	if !exempt {
		q.lastErrorText = ""
		switch {
		case h.Type == "system" && h.Subtype == "init":
			q.modelLessEnded = false
		case h.Type != "system" || h.Subtype == "compact_boundary":
			q.textBeforeModelLess = ""
			q.modelLessEnded = false
		}
	}

	// A new top-level assistant frame means the run is active again.
	if (h.Type == "assistant" || h.Type == "stream_event") && h.ParentToolUseID == nil {
		if !(q.singleTurn && q.runEnded) {
			q.runEnded = false
			q.clearCeilingLocked()
		}
	}
}

func (q *Query) onSessionState(state string) {
	q.mu.Lock()
	q.sessionState = state
	if state == "idle" {
		end := q.resultReceived
		q.mu.Unlock()
		if end {
			q.endRun()
		}
		return
	}
	if !(q.singleTurn && q.runEnded) {
		q.runEnded = false
		if state == "requires_action" {
			q.clearCeilingLocked()
		} else if q.resultReceived {
			q.armCeilingLocked()
		}
	}
	q.mu.Unlock()
}

func (q *Query) armCeilingLocked() {
	q.clearCeilingLocked()
	if d := q.runCeiling(); d > 0 {
		q.ceiling = time.AfterFunc(d, q.endRun)
	}
}

func (q *Query) clearCeilingLocked() {
	if q.ceiling != nil {
		q.ceiling.Stop()
		q.ceiling = nil
	}
}

func (q *Query) endRun() {
	q.mu.Lock()
	q.clearCeilingLocked()
	q.runEnded = true
	closeInput := q.singleTurn || q.inputRequested
	q.mu.Unlock()
	if closeInput {
		_ = q.tr.EndInput()
	}
}

// runCeiling is how long to wait for "idle" after a result before closing
// stdin anyway: CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS from the child env if it
// is a finite number >= 0, otherwise 600 s. Zero means no ceiling.
func (q *Query) runCeiling() time.Duration {
	v := ""
	if q.opts.Env != nil {
		v = q.opts.Env["CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS"]
	} else {
		v = os.Getenv("CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS")
	}
	if ms, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && ms >= 0 && !math.IsInf(ms, 0) {
		if ms > 2147483647 {
			ms = 2147483647
		}
		return time.Duration(ms) * time.Millisecond
	}
	return defaultRunCeiling
}

// ---- writing -----------------------------------------------------------

// waitForHold blocks user writes while a blocking hook requires a successful
// initialize first (SDK "prompt hold").
func (q *Query) waitForHold() error {
	if !q.hold {
		return nil
	}
	select {
	case <-q.initDone:
		return q.initErr
	case <-q.closed:
		return ErrClosed
	}
}

// SendText writes a plain-text user turn in the exact shape the SDK uses for
// a string prompt.
func (q *Query) SendText(text string) error {
	type textBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type inner struct {
		Role    string      `json:"role"`
		Content []textBlock `json:"content"`
	}
	return q.send(struct {
		Type            string  `json:"type"`
		SessionID       string  `json:"session_id"`
		Message         inner   `json:"message"`
		ParentToolUseID *string `json:"parent_tool_use_id"`
	}{"user", "", inner{"user", []textBlock{{"text", text}}}, nil})
}

// Send writes one user message verbatim. msg must marshal to an SDKUserMessage
// object: {"type":"user","message":{"role":"user","content":…},
// "parent_tool_use_id":null,"session_id":""}. Like the SDK's streaming input,
// nothing is added: set "uuid" yourself if you need to refer to the message
// later (rewind, cancel_async_message).
func (q *Query) Send(msg any) error { return q.send(msg) }

func (q *Query) send(msg any) error {
	if err := q.waitForHold(); err != nil {
		return err
	}
	select {
	case <-q.closed:
		return ErrClosed
	default:
	}
	// SDK streaming input: each new message starts a new run.
	q.mu.Lock()
	if !q.singleTurn || q.sentMessages == 0 {
		q.resultReceived = false
		q.runEnded = false
		q.clearCeilingLocked()
	}
	q.sentMessages++
	q.mu.Unlock()
	return q.writeJSON(msg)
}

// EndInput signals that no more user messages will be sent. As in the SDK,
// stdin is closed immediately unless the query has callbacks (approvals,
// hooks, in-process MCP servers) and at least one message was sent; then it
// is closed when the current run ends, so those callbacks keep working.
func (q *Query) EndInput() error {
	q.mu.Lock()
	deferClose := q.sentMessages > 0 && q.hasBidirectionalNeeds() && !q.runEnded
	q.inputRequested = true
	q.mu.Unlock()
	if deferClose {
		return nil
	}
	return q.tr.EndInput()
}

// ---- lifecycle ---------------------------------------------------------

// Initialized waits for the initialize response and returns it. The object
// carries commands, models, agents, account, output styles and capability
// flags; decode it with the wire package.
func (q *Query) Initialized(ctx context.Context) (json.RawMessage, error) {
	select {
	case <-q.initDone:
		return q.initResp, q.initErr
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-q.closed:
		return nil, ErrClosed
	}
}

// PID returns the CLI's process id.
func (q *Query) PID() int { return q.tr.PID() }

// StderrTail returns the redacted tail of the CLI's stderr.
func (q *Query) StderrTail() string { return q.tr.StderrTail() }

// Done is closed when the CLI's output has ended.
func (q *Query) Done() <-chan struct{} { return q.done }

// Close stops the query, like the SDK's close(): pending control requests
// fail with ErrClosed, in-flight handlers are cancelled, stdin is closed and
// the process is terminated (SIGTERM after 2 s, SIGKILL 5 s later). Close
// waits at most 2 s for the process to exit; termination continues in the
// background. It is safe to call more than once.
func (q *Query) Close() error {
	q.closeOnce.Do(func() {
		close(q.closed)
		q.cancel()
		q.router.close(ErrClosed)
		go q.tr.Close()
		select {
		case <-q.tr.Exited():
		case <-time.After(closeWaitForExit):
		}
		q.queue.close()
	})
	return nil
}

// Wait blocks until the CLI process has exited and returns Err.
func (q *Query) Wait() error {
	<-q.done
	return q.Err()
}

// ---- unbounded message queue -------------------------------------------

type messageQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []Message
	closed bool
}

func newMessageQueue() *messageQueue {
	mq := &messageQueue{}
	mq.cond = sync.NewCond(&mq.mu)
	return mq
}

func (mq *messageQueue) push(m Message) {
	mq.mu.Lock()
	if !mq.closed {
		mq.items = append(mq.items, m)
	}
	mq.mu.Unlock()
	mq.cond.Signal()
}

// pop returns the next message. After close it still drains what is queued.
func (mq *messageQueue) pop() (Message, bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	for len(mq.items) == 0 && !mq.closed {
		mq.cond.Wait()
	}
	if len(mq.items) == 0 {
		return Message{}, false
	}
	m := mq.items[0]
	mq.items[0] = Message{}
	mq.items = mq.items[1:]
	return m, true
}

func (mq *messageQueue) close() {
	mq.mu.Lock()
	mq.closed = true
	mq.mu.Unlock()
	mq.cond.Broadcast()
}
