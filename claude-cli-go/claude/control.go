package claude

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"sync"
)

// Wire envelopes for the control channel. Field order follows the SDK.
//
//	{"request_id":"…","type":"control_request","request":{"subtype":"…",…}}      host -> CLI
//	{"type":"control_response","response":{"subtype":"success","request_id":"…","response":{…}}}
//	{"type":"control_response","response":{"subtype":"error","request_id":"…","error":"…","error_code":"…"}}
//	{"type":"control_cancel_request","request_id":"…"}

// wireControlRequest is a host->CLI request. request_id comes before type,
// as the SDK writes it.
type wireControlRequest struct {
	RequestID string `json:"request_id"`
	Type      string `json:"type"` // "control_request"
	Request   any    `json:"request"`
}

// wireMCPControlRequest is an MCP-server-initiated request. Here type comes
// first and the id is a UUID, again as the SDK writes it.
type wireMCPControlRequest struct {
	Type      string `json:"type"` // "control_request"
	RequestID string `json:"request_id"`
	Request   any    `json:"request"`
}

type wireControlResponse struct {
	Type     string                  `json:"type"` // "control_response"
	Response wireControlResponseBody `json:"response"`
}

type wireControlResponseBody struct {
	Subtype   string          `json:"subtype"` // "success" | "error"
	RequestID string          `json:"request_id"`
	Response  json.RawMessage `json:"response,omitempty"`
	Error     string          `json:"error,omitempty"`
	ErrorCode string          `json:"error_code,omitempty"`
}

type wireControlCancel struct {
	Type      string `json:"type"` // "control_cancel_request"
	RequestID string `json:"request_id"`
}

// errNoReply is returned by an inbound handler to leave a request unanswered
// (the SDK's "leave unanswered" sentinel): no frame is written.
var errNoReply = errors.New("claude: no reply")

// marshalJSON encodes like JSON.stringify where encoding/json allows it:
// compact, and without escaping <, > and &. (encoding/json still escapes
// U+2028 and U+2029, which JSON.stringify does not; both are valid JSON.)
func marshalJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// newRequestID mirrors the SDK's Math.random().toString(36).substring(2, 15):
// up to 13 characters of [0-9a-z].
func newRequestID() string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 13)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b)
}

// newUUID returns a random UUIDv4 (used for MCP-originated requests).
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	const hex = "0123456789abcdef"
	out := make([]byte, 0, 36)
	for i, c := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out = append(out, '-')
		}
		out = append(out, hex[c>>4], hex[c&0x0f])
	}
	return string(out)
}

var errorCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// inboundHandler answers one CLI->host control request.
//   - (value, nil): success; a nil value omits the "response" key.
//   - (_, errNoReply): nothing is written.
//   - (_, err): {"subtype":"error","error":err.Error()}, plus error_code when
//     err was made with ErrorWithCode.
type inboundHandler func(ctx context.Context, requestID, subtype string, request json.RawMessage) (any, error)

type controlRouter struct {
	write func(v any) error

	mu        sync.Mutex
	pending   map[string]pendingReq
	inflight  map[string]context.CancelFunc
	closedErr error
}

type pendingReq struct {
	subtype string
	ch      chan wireControlResponseBody
}

func newControlRouter(write func(v any) error) *controlRouter {
	return &controlRouter{
		write:    write,
		pending:  map[string]pendingReq{},
		inflight: map[string]context.CancelFunc{},
	}
}

// request sends a host->CLI control request and waits for its response.
// payload must marshal to a JSON object whose first key is "subtype".
//
// Like the SDK there is no built-in timeout: bound the wait with ctx. When
// ctx ends after the frame was written, a control_cancel_request is sent.
func (r *controlRouter) request(ctx context.Context, subtype string, payload any) (json.RawMessage, error) {
	id, ch, err := r.begin(subtype, payload)
	if err != nil {
		return nil, err
	}
	return r.await(ctx, id, subtype, ch, true)
}

// begin registers a request and writes its frame. It returns as soon as the
// frame is on the wire, so the caller controls frame ordering.
func (r *controlRouter) begin(subtype string, payload any) (string, chan wireControlResponseBody, error) {
	id := newRequestID()
	ch := make(chan wireControlResponseBody, 1)

	r.mu.Lock()
	if r.closedErr != nil {
		err := r.closedErr
		r.mu.Unlock()
		return "", nil, err
	}
	r.pending[id] = pendingReq{subtype: subtype, ch: ch}
	r.mu.Unlock()

	if err := r.write(wireControlRequest{RequestID: id, Type: "control_request", Request: payload}); err != nil {
		r.drop(id)
		return "", nil, err
	}
	return id, ch, nil
}

// await waits for the response to a request started with begin.
func (r *controlRouter) await(ctx context.Context, id, subtype string, ch chan wireControlResponseBody, sendCancel bool) (json.RawMessage, error) {
	select {
	case body, ok := <-ch:
		if !ok {
			return nil, r.err()
		}
		if body.Subtype == "error" {
			ce := &ControlError{Subtype: subtype, Message: body.Error}
			if errorCodePattern.MatchString(body.ErrorCode) {
				ce.ErrorCode = body.ErrorCode
			}
			return nil, ce
		}
		return body.Response, nil
	case <-ctx.Done():
		r.drop(id)
		if sendCancel {
			_ = r.write(wireControlCancel{Type: "control_cancel_request", RequestID: id})
		}
		return nil, &AbortError{Reason: "Control request aborted: " + ctx.Err().Error()}
	}
}

func (r *controlRouter) drop(id string) {
	r.mu.Lock()
	delete(r.pending, id)
	r.mu.Unlock()
}

func (r *controlRouter) err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closedErr != nil {
		return r.closedErr
	}
	return ErrClosed
}

// deliver routes a control_response to the waiting request. Responses with no
// waiter are dropped (the SDK parks them in a bounded map and never reads them
// on this path).
func (r *controlRouter) deliver(body wireControlResponseBody) {
	r.mu.Lock()
	p, ok := r.pending[body.RequestID]
	if ok {
		delete(r.pending, body.RequestID)
	}
	r.mu.Unlock()
	if ok {
		p.ch <- body
	}
}

// handleInbound runs h for a CLI->host request on its own goroutine, so a
// slow approval never blocks the stdout reader.
//
// SDK parity: a request id already in flight is skipped; a request arriving
// after close is skipped; a response computed after close is not written; a
// control_cancel_request cancels the handler's context, but the handler's
// eventual result or error is still written.
func (r *controlRouter) handleInbound(parent context.Context, requestID, subtype string, request json.RawMessage, h inboundHandler) {
	ctx, cancel := context.WithCancel(parent)
	r.mu.Lock()
	if _, dup := r.inflight[requestID]; dup || r.closedErr != nil {
		r.mu.Unlock()
		cancel()
		return
	}
	r.inflight[requestID] = cancel
	r.mu.Unlock()

	go func() {
		defer func() {
			r.mu.Lock()
			delete(r.inflight, requestID)
			r.mu.Unlock()
			cancel()
		}()
		resp, err := h(ctx, requestID, subtype, request)
		if errors.Is(err, errNoReply) {
			return
		}
		r.mu.Lock()
		closed := r.closedErr != nil
		r.mu.Unlock()
		if closed {
			return
		}
		body := wireControlResponseBody{RequestID: requestID}
		if err != nil {
			body.Subtype, body.Error = "error", err.Error()
			var coded *codedError
			if errors.As(err, &coded) {
				body.ErrorCode = coded.code
			}
		} else {
			body.Subtype = "success"
			if resp != nil {
				raw, merr := marshalJSON(resp)
				if merr != nil {
					body.Subtype, body.Error = "error", merr.Error()
				} else {
					body.Response = raw
				}
			}
		}
		_ = r.write(wireControlResponse{Type: "control_response", Response: body})
	}()
}

// cancelInbound handles a control_cancel_request from the CLI.
func (r *controlRouter) cancelInbound(requestID string) {
	r.mu.Lock()
	cancel, ok := r.inflight[requestID]
	r.mu.Unlock()
	if ok {
		cancel()
	}
}

// close fails every pending request and cancels every inbound handler.
func (r *controlRouter) close(err error) {
	if err == nil {
		err = ErrClosed
	}
	r.mu.Lock()
	if r.closedErr != nil {
		r.mu.Unlock()
		return
	}
	r.closedErr = err
	pending := r.pending
	r.pending = map[string]pendingReq{}
	inflight := r.inflight
	r.inflight = map[string]context.CancelFunc{}
	r.mu.Unlock()
	for _, p := range pending {
		close(p.ch)
	}
	for _, c := range inflight {
		c()
	}
}
