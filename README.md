# Chat interface specs: Claude CLI chat host

Research, specification and starter code for a chat system built on the `claude` CLI:

- a **Go server** hosts one `claude` CLI process per conversation and speaks the CLI's stream-json protocol;
- a separate **chat client** talks to the Go server over **gRPC**;
- the target is **Claude Code CLI 2.1.311** (Agent SDK 0.3.311 behaviour).

Anthropic publishes the Agent SDK for TypeScript and Python only, so the Go client here reproduces what the TypeScript SDK does on the wire.

## Start here

| What | Where |
| --- | --- |
| Technical specification | [spec/claude-chat-spec.md](spec/claude-chat-spec.md); living copy at https://claude.ai/artifact/PVcDrqBmQeh6Kf2kRNtsDV |
| 2.1.311 protocol surface map (flags, messages, control requests) | [spec/surface-2.1.311.md](spec/surface-2.1.311.md) |
| gRPC contract between chat client and Go server | [proto/claudechat/v1/chat.proto](proto/claudechat/v1/chat.proto) |
| Go protocol client | [claude-cli-go/](claude-cli-go/) (`README.md`, `PARITY.md`) |
| What the real 2.1.311 binary emits (36 recorded scenarios) | [verification/scenarios-2.1.311.md](verification/scenarios-2.1.311.md) |
| HTML design overview | https://claude.ai/artifact/8Q2v6EskY12BW9WiCaqSVf (source: [design/claude-chat-design.html](design/claude-chat-design.html)) |

The Markdown spec in this folder and the living document are kept in step; if they differ, the file in this folder is newer. To republish the HTML overview, pass its artifact URL, or a second page is created.

## Layout

```
spec/                     The specification, the 2.1.311 surface map, diagram images
proto/claudechat/v1/      chat.proto: the gRPC contract (not yet compiled by a protobuf compiler)
claude-cli-go/            Go module example.com/claudecli
  claude/                 protocol client (hand-written): transport, control channel, approvals, hooks, MCP
  wire/                   every SDK message / control / option type (generated from sdk.d.ts 0.3.311)
  cmd/chatdemo/           terminal chat example
research/
  extension-2.1.311.md    what the official VS Code extension 2.1.311 does, claim by claim
  parity/transport-0.3.311.md   64-item checklist of how SDK 0.3.311 drives the CLI (the Go client follows it)
  parity/transport.md     the same for 0.3.325 (background)
  parity/api-surface.md   every Query method, option, message and control type
  parity/typesgen/        generator for claude-cli-go/wire/types.go
verification/             CLI change-verification pipeline and recordings
  compare.sh              run the free suite against one binary (never re-run into an existing runs/ folder)
  probe.mjs               zero-cost control-only probe
  run-scenario.mjs        scenario runner (mock by default; refuses to overwrite a recording without --force)
  mock-api.mjs            fake Anthropic Messages API (deterministic, local)
  shape.mjs               shape inference, sequence signatures, diffs
  scenarios-2.1.311.md    recorded behaviour of 2.1.311, with the verified Agent map fields
  static.md / runtime.md / triage.md    pipeline design
  runs/v2.1.311/, runs/v2.1.325/        recordings
  scenario-docgen/        generator for scenarios-2.1.311.md; orig/ has the harness scripts before the new scenarios
tools/                    manifest extractors (SDK types, binary strings, --help, settings schema, changelog)
reference/                third-party material, about 1.6 GB, not for git
  cli-bins/               darwin-arm64 CLI binaries 2.1.311 and 2.1.325 (integrity- and SHA-256-verified)
  sdk-pkgs/               Agent SDK packages 0.3.278 / 0.3.300 / 0.3.311 / 0.3.325
  vsix-2.1.311/           official extension 2.1.311 (darwin-arm64), unpacked
  vsix-2.1.325/           official extension 2.1.325 (linux-x64), unpacked
  changelogs/             CLI and SDK changelog snapshots
```

## Running things

Go client tests (unit, fake CLI, and a replay of all 2.1.311 recordings):

```sh
cd claude-cli-go && go test -race ./...
```

Go client against the real 2.1.311 binary and the local fake API (no network, throwaway config directory):

```sh
(cd verification && node mock-api.mjs --port 8791 --log /tmp/mock.ndjson) &
cd claude-cli-go
CLAUDE_INTEGRATION_BIN=../reference/cli-bins/2.1.311/x/package/claude \
CLAUDE_INTEGRATION_MOCK_URL=http://127.0.0.1:8791 go test ./claude -run TestIntegration -v
```

Checking a new CLI version against the baseline:

```sh
cd verification
node mock-api.mjs --port 8787 --log runs/mock.ndjson &
./compare.sh <new-version> <path-to-new-binary>          # writes runs/v<new-version>/
node shape.mjs diff runs/v2.1.311/mock/shapes.json runs/v<new-version>/mock/shapes.json
python3 -I ../tools/diff-manifest.py manifests/sdk-0.3.311.json manifests/sdk-<new>.json
```

Every run is hermetic: a throwaway `CLAUDE_CONFIG_DIR`, a dummy API key pointed at the local mock, inherited `CLAUDE*` / `ANTHROPIC*` variables removed. Nothing touches `~/.claude`.

## Open decisions

- Whether driving a user's Claude subscription from a third-party product is within Anthropic's usage terms. Not settled; confirm before shipping.
- `chat.proto` has been reviewed by hand but not compiled; run `buf lint` / `protoc` before generating code.
- The Go client sends `CLAUDE_CODE_ENTRYPOINT=sdk-go` and `CLAUDE_AGENT_SDK_VERSION=0.3.311`. Both are settings; see `claude-cli-go/PARITY.md`.
