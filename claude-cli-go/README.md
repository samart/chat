# claude-cli-go

A Go client for the `claude` CLI's stream-json protocol, written to behave on the wire like Anthropic's TypeScript Agent SDK. Anthropic publishes the Agent SDK for TypeScript and Python only; this package is the equivalent layer for a Go host.

- **Target:** Claude Code CLI **2.1.311** (Agent SDK **0.3.311**).
- **Status:** starter package. The protocol client is complete and tested against the real 2.1.311 binary. The gRPC server that sits on top of it is specified in [../proto/claudechat/v1/chat.proto](../proto/claudechat/v1/chat.proto) and not implemented yet.
- **Module path:** `example.com/claudecli` is a placeholder. Change it in `go.mod` and the imports before publishing.

## Packages

| Package | What it is |
| --- | --- |
| `claude` | The client: spawns the CLI, speaks NDJSON, runs the control channel, answers approvals, hooks and in-process MCP requests. Hand-written. |
| `wire` | Every message, control request and option type from the SDK's `sdk.d.ts`, as Go structs that round-trip JSON without loss. Generated; see [../research/parity/typesgen/](../research/parity/typesgen/). |
| `cmd/chatdemo` | A 200-line terminal chat showing the whole loop. |

## Using it

```go
q, err := claude.Start(ctx, claude.Options{
    PathToClaudeCodeExecutable: "/path/to/claude",      // required: no binary is bundled
    Cwd:                        workspace,
    Resume:                     sessionID,              // "" for a new chat
    IncludePartialMessages:     true,                   // stream_event token deltas
    EnableFileCheckpointing:    true,                   // needed for RewindFiles
    SettingSources:             []string{"user", "project", "local"},
    SystemPrompt:               claude.PresetSystemPrompt(""), // Claude Code's own prompt
    InitializeTimeout:          60 * time.Second,
    CanUseTool: func(ctx context.Context, req claude.PermissionRequest) (*claude.PermissionResult, error) {
        // Forward to the UI and wait. ctx is cancelled if the CLI withdraws the request.
        return claude.Allow(req.Input), nil
    },
})
defer q.Close()

init, _ := q.Initialized(ctx)          // commands, models, account, capabilities (JSON)
q.SendText("Fix the failing test")

for m := range q.Messages() {          // system, stream_event, assistant, user, result, …
    v, err := m.Decode()               // typed value from package wire; m.Raw is always complete
    _ = v; _ = err
}
if err := q.Err(); err != nil { /* *ProcessError, *ErrorResultError, … */ }
```

During a session: `q.Interrupt`, `q.SetPermissionMode`, `q.SetModel`, `q.McpServerStatus`, `q.RewindFiles`, `q.StopTask`, `q.GetTaskOutput`, `q.GetContextUsage`, and `q.Request(ctx, subtype, fields…)` for anything else. Control requests have no built-in timeout, as in the SDK: pass a context with a deadline.

Three rules a host must follow:

1. **Check `is_error` on `result`, not `subtype`.** API failures arrive as `subtype: "success"` with `is_error: true`.
2. **Never drop a frame you do not recognise.** `Message.Raw` is the CLI's exact bytes; forward it.
3. **`SystemPrompt: nil` means an empty system prompt**, exactly as in the SDK. Use `PresetSystemPrompt` to get Claude Code's behaviour.

## Tests

```sh
go test ./...                    # unit, fake-CLI, and replay of the recorded 2.1.311 transcripts
go test -race ./...
```

The replay test feeds every recording under `../verification/runs/v2.1.311/mock/` through the client and checks that each frame is delivered once, in order and byte-identical, and that every CLI request gets its reply.

Two tests drive the **real binary** against the local fake Messages API. They contact nothing outside this machine and use a throwaway config directory:

```sh
(cd ../verification && node mock-api.mjs --port 8791 --log /tmp/mock.ndjson) &
CLAUDE_INTEGRATION_BIN=../reference/cli-bins/2.1.311/x/package/claude \
CLAUDE_INTEGRATION_MOCK_URL=http://127.0.0.1:8791 \
  go test ./claude -run TestIntegration -v
```

## Where to read next

- [PARITY.md](PARITY.md): what matches the SDK, what deliberately differs, and what is not implemented.
- [../research/parity/transport-0.3.311.md](../research/parity/transport-0.3.311.md): the 64-item checklist of SDK 0.3.311 behaviour this client follows.
- [../research/parity/api-surface.md](../research/parity/api-surface.md): every `Query` method, option, message and control type.
- [../spec/](../spec/): the product spec and the 2.1.311 surface map.
