# Parity with the Agent SDK (0.3.311)

The reference is [../research/parity/transport-0.3.311.md](../research/parity/transport-0.3.311.md), section (c): 64 numbered rules read out of SDK 0.3.311's `sdk.mjs`. Item numbers below refer to that list. "Verified" means a test in this module checks it.

## Matches the SDK

| Area | Items | Where | Verified by |
| --- | --- | --- | --- |
| Command resolution: native binary vs JS entrypoint | 1 | `resolveCommand` | `TestResolveCommand` |
| argv prefix, `--flag=value` encoding, push order, empty-list forms | 3–5, 9–12 | `buildArgs` | `TestArgsChecklist`, `TestJSNumber` |
| Pre-spawn errors | 6 | `buildArgs` | `TestArgsErrors` |
| Extra args and `--settings` last | 7 | `buildArgs` | `TestArgsChecklist` |
| No implicit `--await-initialize` on resume | 11 | `buildArgs` | `TestArgsChecklist` |
| Environment additions and removals | 13, 15–20 | `buildEnv` | `TestEnv` |
| Run-end ceiling from `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS`, 0 = none | 21 | `runCeiling` | — |
| Compact JSON, key order, no `<>&` escaping | 23 | `marshalJSON`, `orderedObject` | `TestTurnWithApprovalHookAndMcp` |
| Writes after stdin EOF dropped; write to dead process errors | 24 | `processTransport.Write` | `TestWriteAfterEndInputIsDropped` |
| Line reader: no limit, blank and non-JSON lines skipped | 25 | `readLoop` | `TestTurnWithApprovalHookAndMcp` |
| stderr tail of 2048 bytes, redacted; 200 ms drain after exit | 26, 27 | `pumpStderr`, `reap` | `TestExitErrorText` |
| initialize is the first frame; envelope has `request_id` first | 28, 30 | `start`, `wireControlRequest` | fake-CLI and replay tests |
| initialize key order; hook ids `hook_0…`; SDK MCP servers announced there | 31–35 | `buildInitializeRequest` | `TestTurnWithApprovalHookAndMcp` |
| In-process MCP manifest capture, 250 ms deadline | 29 | `captureMcpManifests` | — |
| Default system prompt `[""]`; preset omits it | 34 | `buildInitializeRequest` | `TestUnknownAndDefaultReplies` |
| `pending_permission_requests` replayed from the initialize response | 37 | `awaitInitialize` | — |
| String prompt shape; prompts do not wait for the initialize response | 38, 39 | `SendText` | `TestTurnWithApprovalHookAndMcp` |
| When stdin closes (result, `idle`, ceiling) | 40–43 | `onResult`, `onSessionState`, `EndInput` | `TestPromptClosesStdinAfterResult` |
| Swallowed frames; everything else delivered unchanged | 44–47 | `route` | replay test (all 2.1.311 recordings) |
| Request ids, no control timeout, cancel frame | 48, 49 | `controlRouter` | — |
| Reply envelopes; unknown subtype error text, no `error_code` | 50, 51 | `handleInbound`, `handleControlRequest` | `TestUnknownAndDefaultReplies` |
| A cancelled handler's late result is still written | 51 | `handleInbound` | `TestCancelStillSendsLateResult` |
| `can_use_tool` reply `{…result, toolUseID}`; nil = no reply | 52 | `handleCanUseTool` | fake-CLI and integration tests |
| `hook_callback`, `mcp_message` (request vs notification ack) | 53, 54 | `handleControlRequest`, `handleMcpMessage` | `TestTurnWithApprovalHookAndMcp` |
| `elicitation` default decline; `request_user_dialog` no reply | 55, 56 | `handleControlRequest` | `TestUnknownAndDefaultReplies` |
| Shutdown: stdin EOF, SIGTERM at 2 s, SIGKILL 5 s later | 59 | `processTransport.Close` | `TestCloseWhileWriteBlocked` |
| Close cancels handlers and fails pending requests | 61 | `Query.Close` | `TestClosedQueryFailsRequests` |
| Exit error text and error-result substitution | 62, 63 | `exitError`, `onResult`, `onOtherFrame` | `TestExitErrorText`, `TestInterruptAndErrorResultOnExit` |

End to end against the real 2.1.311 binary (`TestIntegrationRealCLI`): initialize, streaming, an approved tool with a host `PreToolUse` hook, a denied tool, `set_permission_mode`, `mcp_status`, and unsupported-subtype detection.

## Deliberate differences

| Difference | Why |
| --- | --- |
| `CLAUDE_CODE_ENTRYPOINT` defaults to `sdk-go`, not `sdk-ts` (item 14) | Honest identification. The CLI treats any `sdk-` prefix as an SDK client, and `TestIntegrationEntrypointParity` shows identical initialize responses, `system/init` frames, commands and tools for both values on 2.1.311. Set `Options.Entrypoint` to change it. |
| `CLAUDE_AGENT_SDK_VERSION` is sent as `0.3.311` (item 15) | The SDK sends its own version; the CLI's manifest lists tested wrapper versions. This client claims the SDK version whose behaviour it copies. Set `Options.SDKVersion` to change it, and decide deliberately. |
| `Options.InitializeTimeout` exists (item 36) | The SDK's `query()` has none. A server needs one; 0 keeps SDK behaviour. |
| `Close` returns after at most 2 s and `Transport.Close` blocks until exit | Same timings as the SDK; Go callers usually want a blocking variant. |
| `HookMatcher.OnFailure` is accepted, and fails fast if the CLI does not report `hook_on_failure_supported` | `onFailure` arrived after 0.3.311. CLI 2.1.311 ignores it, so silently accepting it would be unsafe. |
| U+2028 and U+2029 are escaped in outgoing JSON (item 23) | `encoding/json` always escapes them. Both forms are valid JSON and decode identically. |
| Key order of `PermissionResult.Raw` replies is `behavior`, then sorted, then `toolUseID` | The SDK keeps the callback's key order; a `Raw` result arrives as an unordered object. Use the typed fields for exact order. |
| `ExtraArgs` are emitted in slice order (item 7) | JS puts integer-like keys first; no real flag is integer-named. |
| Lines are split on `\n` only (item 25) | The SDK's reader also splits on lone `\r`, U+2028 and U+2029, which would break valid JSON lines. This is the tolerant superset. |
| Messages are buffered without limit between the reader and `Messages()` | Same as the SDK's queue, stated here because it matters to a server: a consumer that never reads grows memory. |

## Not implemented

| SDK feature | Items | Note |
| --- | --- | --- |
| `sandbox` option merged into `--settings` | 8 | Pass a complete settings JSON in `Options.Settings` instead. |
| `managedSettings`, `plugins` via initialize, `workload`, `channels`, session store / `--session-mirror` | 5, 35 | Use `Options.ExtraArgs` / `InitializeExtra` if needed. |
| Auto `--debug-file` when `DEBUG_CLAUDE_AGENT_SDK` is set in the parent | 5 (row 10b) | Set `Options.DebugFile`. |
| `oauth_token_refresh`, `host_auth_token_refresh` callbacks | 57, 58 | Answered as unsupported. The CLI manages its own login. |
| `toolAliases`, `codeRunningMcpTools`, `webSearchIsolationExemptMcpServers` initialize fields | 31 | Add through `InitializeExtra`. |
| Killing children when the parent process exits | 60 | The server should call `Close` on shutdown. |
| Unmatched `control_response` cache | 46 | Unmatched responses are dropped; nothing reads the SDK's cache on this path. |
| `startup()` / `prewarm()` | — | Only the `query()` path is ported. |
| Secret redaction identical to the SDK's | 26 | `redactSecrets` covers common token shapes; treat it as best effort. |

## When the target CLI changes

1. Run the verification suite for the new version (`../verification/compare.sh`) and read the adoption report.
2. Re-run the transport comparison against the new `sdk.mjs` and update the checklist.
3. Regenerate `wire` from the new `sdk.d.ts` (`../research/parity/typesgen/build.sh`).
4. Change `TargetCLIVersion` / `TargetSDKVersion`, record new transcripts, and run `go test ./...`: the replay test picks up `../verification/runs/v<TargetCLIVersion>/mock/` automatically.

Known items for 2.1.325: unknown-subtype replies gain `error_code: "unsupported_request"`; `onFailure` and the prompt hold become real; `managedSettings` can travel in initialize; `--await-initialize` is implied on resume with the bundled binary.
