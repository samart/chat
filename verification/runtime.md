# CLI change verification: runtime half

The goal: each time a new `claude` CLI release comes out, run it the way the editor plugin does (stream-json over stdio), record what it actually sends, and flag differences from the previous release that the chat UI must handle or might want to adopt.

Everything below was run on this machine against **Claude Code 2.1.278** (`/opt/homebrew/bin/claude`, a Mach-O arm64 Bun build). **No paid model call was made.** The zero-cost probe made no `/v1/messages` call (confirmed in `--debug-file`). Every scenario that sent a user turn went to a local fake Messages API, using a dummy API key that would get a 401 from the real API if a request ever got out.

## Files (all in this directory)

| File | Purpose |
|---|---|
| `probe.mjs` | Zero-model-call probe. Sends only `control_request`s and never a user message (`send()` throws on `type:"user"`). Records both directions. |
| `fake-mcp.mjs` | A 30-line stdio MCP server with one tool (`echo`). Gives `mcp_status` a connected server, and gives scenarios a tool with no side effects. |
| `mock-api.mjs` | A fake Anthropic Messages API: canned SSE streams, error codes and rate-limit headers, plus a request log. |
| `run-scenario.mjs` | Scenario driver that behaves like the plugin: it answers `can_use_tool`, `hook_callback` and interrupts. `--target mock` is free. `--target real --allow-spend` is the paid path. |
| `shape.mjs` | `infer` (observed shapes), `diff` (BREAK/ADOPT/INFO), `seq` (sequence signature), `catalog` / `catdiff` (value catalogs). |
| `runs/2.1.278/{isolated,user-nomcp,hermetic,oauth-baseurl}/` | Real probe transcripts, `results.json` and `debug.log`. |
| `runs/mock/<scenario>/`, `runs/mock2/<scenario>/` | Two independent full mock-suite runs: `transcript.ndjson`, `seq.txt`, `debug-*.log`, `meta.json`. |
| `runs/mock/shapes.json`, `runs/2.1.278/*.shapes.json`, `*.catalog.json` | Golden artifacts. |

Note: `runs/2.1.278/{isolated,user-nomcp,oauth-baseurl}/transcript.ndjson` hold the real `initialize` response, which includes the account email. `shape.mjs` reduces emails to a `format: email` tag, so the goldens contain no PII. Do not commit the raw transcripts.

---

## 1. Zero-cost probing (no user prompt)

### How

```sh
node probe.mjs --out runs/<ver>/isolated --cwd <NEW empty dir> --mode isolated \
  --extra-subtypes list_models,get_plan,get_hooks_listing,list_permission_rules,get_session_cost,get_usage,background_tasks,get_workspace_diff
```

Spawned command line (from `argv.json`):

```
claude -p --input-format stream-json --output-format stream-json --verbose
  --permission-prompt-tool stdio --include-partial-messages --model haiku
  --max-turns 1 --max-budget-usd 0.01 --no-session-persistence
  --mcp-config '{"mcpServers":{"probe_sdk":{"type":"sdk","name":"probe_sdk"},
                 "probe_stdio":{"type":"stdio","command":"node","args":[".../fake-mcp.mjs"]},
                 "probe_broken":{"type":"stdio","command":"/nonexistent/mcp-server-binary"}}}'
  --strict-mcp-config --debug-file <out>/debug.log --setting-sources=
```

The probe sends these, in order: `initialize` (with one `SessionStart` hook callback and `sdkMcpServers:["probe_sdk"]`), `mcp_status`, `get_settings`, `get_context_usage`, `set_permission_mode` (plan, then a bogus mode, then default), `set_model`, `set_max_thinking_tokens`, `interrupt` (while idle), `rewind_files` (no checkpoint), an unknown subtype, `initialize` a second time, and the extra read-only subtypes. It then closes stdin. The CLI exits with code 0 in about 5 s.

Modes:

- `isolated` uses `--setting-sources=` and `--strict-mcp-config`. This is the baseline for diffs.
- `user-nomcp` loads user settings and plugins but no ambient MCP servers. Use it for the catalog only.
- `--hermetic <mock-url>` scrubs inherited `CLAUDE*`/`ANTHROPIC*` variables, points `CLAUDE_CONFIG_DIR` at a throwaway directory and uses a dummy API key against the mock. Use it to run a new or untrusted binary without touching `~/.claude`.

### What it yields (real output, 2.1.278)

**The `initialize` response** is the main payoff. Its top-level keys:
`commands, agents, output_style, available_output_styles, user_output_styles_dir, models, unavailable_models, account, pid, current_permission_mode, hooks_applied, analytics_disabled, remote_control_auto_enable, remote_control_auto_connect_default, remote_control_available, remote_control_auto_on_by_default, ide_rc_auto_enable_gate, fast_mode_state, fast_mode_disabled_reason, session_state`

```json
"models": [{"value":"default","resolvedModel":"claude-opus-5[1m]","displayName":"Default (recommended)",
            "description":"Opus 5 with 1M context · Best for everyday, complex tasks","supportsEffort":true,
            "supportedEffortLevels":["low","medium","high","xhigh","max"],"supportsAdaptiveThinking":true,
            "supportsFastMode":true,"supportsAutoMode":true}, ... ,
           {"value":"haiku","resolvedModel":"claude-haiku-4-5-20251001","displayName":"Haiku","description":"Haiku 4.5 · Fastest for quick answers"}],
"unavailable_models": [{"value":"cc-update-required-1","displayName":"Opus 5.5 (disabled)",
            "description":"Update to 2.1.280+ to use Opus 5.5","disabled":true, ...}],
"account": {"email":"<redacted>","organization":"<redacted>","subscriptionType":"Claude Max","apiProvider":"firstParty"},
"available_output_styles": ["default","Proactive","Concise","Explanatory","Learning"],
"fast_mode_state":"off","fast_mode_disabled_reason":"sdk_opt_in_required","session_state":"idle"
```

- `commands`: 58 in isolated mode, 81 with user plugins. Each is `{name, description, argumentHint, builtin, aliases?}`. `builtin` is **absent** on plugin and skill commands; the shape diff flagged this.
- `agents`: 7, each `{name, description}`.
- In `--hermetic` (API-key) mode, `account` is `{tokenSource:"none", apiKeySource:"ANTHROPIC_API_KEY", apiProvider:"firstParty"}`, there are 42 commands, and the model list is different. Compare runs only within the same auth mode.

**Other responses:**

```json
mcp_status → {"mcpServers":[
  {"name":"probe_stdio","status":"connected","serverInfo":{"name":"fake-mcp","version":"0.0.1"},
   "config":{"type":"stdio",...},"scope":"dynamic","source":"dynamic","tools":[{"name":"echo","annotations":{}}]},
  {"name":"probe_broken","status":"failed","error":"ENOENT: no such file or directory, posix_spawn 'stdio'",...},
  {"name":"probe_sdk","status":"connected","serverInfo":{"name":"probe_sdk","version":"0.0.1"},"scope":"dynamic","source":"sdk","tools":[{"name":"host_ping","annotations":{}}]}]}
get_settings → {"effective":{},"sources":[],"applied":{"model":"claude-haiku-4-5-20251001","effort":null,"advisor":null,"ultracode":false}}
get_context_usage → {categories[{name,tokens,color,kind,isDeferred?}], totalTokens:27118, maxTokens:200000, percentage:14,
                     gridRows[[...]], model, memoryFiles, mcpTools, agents, slashCommands{...}, skills{...},
                     autoCompactThreshold:167000, isAutoCompactEnabled, messageBreakdown{...}, apiUsage:null}
set_permission_mode(plan) → {"mode":"plan"}   plus an unsolicited {"type":"system","subtype":"status","status":null,"permissionMode":"plan",...}
set_permission_mode(__bogus__) → error "Cannot set permission mode: must be one of acceptEdits, auto, bypassPermissions, default, dontAsk, plan"
set_model → success, no payload, plus system/status
interrupt (idle) → {"still_queued":[]}
rewind_files (no checkpoint) → {"canRewind":false,"error":"No file checkpoint found for this message."}
unknown subtype → error "Unsupported control request subtype: __probe_unknown_subtype__"
initialize (2nd time) → success again, same payload
list_models → models + the disabled update-required entry
get_hooks_listing → {events:[],hooks:[],eventCatalog:[33 events: PreToolUse … MessageDisplay, each {name,summary,supportsMatcher}]}
get_usage → {session{...}, subscription_type:"max", rate_limits_available, rate_limits{five_hour{utilization,resets_at,...}, seven_day{...}, ~30 codename keys}, behaviors}
get_plan → {"exists":false}; get_workspace_diff → {"diff":null}; background_tasks → {} (+ system/background_tasks_changed)
```

**CLI→host requests with no prompt.** As soon as it starts, the CLI makes three `mcp_message` calls to the host-side SDK server: `initialize` (clientInfo version `2.1.278`, protocolVersion `2025-11-25`), then `notifications/initialized`, then `tools/list`. The `SessionStart` hook callback did **not** fire.

**What you cannot get without a prompt:** `system/init` (its tools, MCP servers, slash_commands, skills, plugins and capabilities), any `stream_event`, `assistant`, `result` or `can_use_tool`, and hook callbacks. `system/init` is emitted only after the first user message. No `keep_alive` or `control_cancel_request` appeared in about 5 s of idle time or in any scenario.

### Probe gotchas found

1. **`get_context_usage` is not network-free.** It made 18–36 calls to `/v1/messages/count_tokens`. They are not generation calls, but use `--no-context-usage` if you want zero API traffic.
2. **Server-driven catalogs change from run to run.** Two isolated runs a few minutes apart returned different model lists (`claude-fable-5-2[1m]` in one, `claude-fable-5-1[1m]` in the other). `unavailable_models` was missing in the first run and present in the second. Treat models and `rate_limits` codenames as data, not schema. Require a catalog change to repeat in two consecutive probes before reporting it.
3. **Inherited environment leaks in.** Running from inside a Claude Code session passes about 16 `CLAUDE*` variables to the child (for example `CLAUDE_CODE_ENTRYPOINT`, `CLAUDECODE` and `CLAUDE_CODE_CHILD_SESSION`). The harness must scrub them (`run-scenario.mjs` and `probe.mjs --hermetic` do).
4. **The error text lists valid values.** The `set_permission_mode` error string names every valid mode, so it is a cheap way to read that enum for each version. Note that `manual` appears in `--help` but is not accepted here.
5. **You can test whether a subtype exists**, because "Unsupported control request subtype" differs from any other error. `strings claude.exe` turns up about 100 subtype literals, among them `list_models`, `get_plan`, `side_question`, `generate_session_title`, `update_settings`, `end_session` and `plugin_install`. **Probe only allow-listed read-only subtypes.** `side_question` and `generate_session_title` call the model. `update_settings`, `apply_flag_settings` and `plugin_install` change user state.
6. **`commands[].builtin` is optional.** It is missing on plugin and skill commands.

---

## 2. Scenario fixtures for the paid part (and a free mock equivalent)

All scenarios share these settings:

- `--model haiku --max-turns 4 --max-budget-usd 0.05 --setting-sources= --strict-mcp-config --permission-prompt-tool stdio --include-partial-messages`
- A fresh empty cwd.
- A scrubbed environment.
- A `PreToolUse` hook callback registered in `initialize`, so that `hook_callback` is exercised too.

For real runs, also use `--tools <only what the scenario needs>` and a short `--system-prompt`. The default first request is about 27k tokens of system prompt and tool definitions, per `get_context_usage`. The protocol shapes do not depend on the system prompt, so this cuts the dominant cost. Set the budget from the first measured `result.total_cost_usd` instead of guessing.

| Scenario | Real prompt (Haiku) | Host behaviour | Sequence signature observed (mock, 2.1.278) |
|---|---|---|---|
| `text` | "Reply with exactly: hello" | none | `user → system/init → system/status → stream_event/message_start → content_block_start/text → text_delta+ → assistant/text → content_block_stop → message_delta → message_stop → result/success` |
| `thinking` | "Think briefly, then reply…" (`--effort low`) | none | adds `content_block_start/thinking`, `thinking_delta`, `signature_delta`, `assistant/thinking`, **`system/thinking_tokens`** |
| `tool_allow` | "Run `touch allowed-marker` with Bash" | `can_use_tool` → `{behavior:"allow",updatedInput:input}` | `… content_block_start/tool_use → input_json_delta+ → assistant/tool_use → control_request/hook_callback → (stream stop) → control_request/can_use_tool → … → user/tool_result → system/status → second turn → result/success` |
| `tool_deny` | "Run `touch denied-marker`…" | `can_use_tool` → `{behavior:"deny",message}` | the tool_result is `{"is_error":true,"content":"Denied by verification harness"}`; the result has `permission_denials:[{tool_name,tool_use_id,tool_input}]` and is still `subtype:"success"` |
| `file_edit` | "Create hello.txt with Write, then Edit it" | allow ×2 | two `can_use_tool` (Write, Edit), each with `permission_suggestions:[{type:"setMode",mode:"acceptEdits",destination:"session"}]`; `tool_use_result` carries `{type:"create",filePath,content}` and `{filePath,oldString,newString,…}` |
| `plan_mode` | "Plan creating notes.txt, present via ExitPlanMode" (`--permission-mode plan`) | allow ExitPlanMode, then `set_permission_mode default` | `can_use_tool(ExitPlanMode)` → `system/status{permissionMode}` → `user/tool_result` → `control_response/set_permission_mode` → final turn |
| `mcp_tool` | "Call echo on probe_stdio" (with `fake-mcp.mjs`) | allow | `can_use_tool` has `mcp_server:{name,source}` and `display_name:"Echo"`; the suggestion is `addRules … destination:"localSettings"` |
| `interrupt` | "Count slowly from 1 to 200" | send `interrupt` after the 3rd delta | `text_delta+ → out:interrupt → control_response/interrupt → assistant/text (partial) → user/text "[Request interrupted by user]" → result/error_during_execution` (`terminal_reason:"aborted_streaming"`, `errors:["[ede_diagnostic] …"]`) |
| `resume` | turn 1: "Remember 'tangerine'"; run 2 with `--resume <id>`: "What word?" | 2 processes, `--session-id` then `--resume` | both runs are `init → status → assistant → result`. The mock log confirms that run 2's request carried 3 prior messages. |
| `max_turns` | "Run echo 1, 2, 3 as separate Bash calls" (`--max-turns 2`) | allow | `result/error_max_turns`, `terminal_reason:"max_turns"`, `is_error:true` |
| `rate_limit_warning` | mock only | none | **no `rate_limit_event`** under API-key auth (see below) |
| `error_429` / `529` / `500` | mock only | none (`CLAUDE_CODE_MAX_RETRIES=1`) | `system/api_retry{attempt,max_retries,retry_delay_ms,error_status,error:"rate_limit"\|"overloaded"\|"server_error"}` → synthetic `assistant` with `is_api_error_message:true`, `error:"rate_limit"`, `model:"<synthetic>"` → **`result.subtype:"success"` with `is_error:true`**, `terminal_reason:"api_error"`, `api_error_status:429` |
| `error_400` | mock only ("prompt is too long") | none | `result` `is_error:true`, `terminal_reason:"prompt_too_long"`, `api_error_status:400` |

Behaviours the UI needs to know, all observed in these runs:

- **An API error does not produce `subtype:"error_*"`.** It arrives as `result.subtype:"success"` with `is_error:true`. The UI must branch on `is_error` and `terminal_reason`, not on `subtype`.
- **The complete `assistant` message and the `PreToolUse` `hook_callback` arrive before the stream's `content_block_stop` and `message_stop`**, because tool execution starts while the stream is still finishing. Do not assume "stream finished, then tool".
- **`system/status` messages are asynchronous.** They can arrive after the next control request is sent. When diffing sequences, treat `system/status` and `system/background_tasks_changed` as floating.
- **`Bash echo …` is auto-allowed and never reaches `can_use_tool`.** The permission scenarios must use a command that writes, such as `touch`. The first version of this harness used `echo`, and its deny scenario silently ran the command.
- `result` keys: `api_error_status, duration_api_ms, duration_ms, first_content_frame_ms?, is_error, modelUsage, num_turns, permission_denials, queued_turn_count, result, result_index, session_id, stop_reason, subagent_stats, subtype, terminal_reason, time_to_request_ms?, total_cost_usd, ttft_ms?, ttft_stream_ms?, usage, uuid, fast_mode_*`.
- `system/init` keys: `agents, analytics_disabled, apiKeySource, capabilities, claude_code_version, cwd, fast_mode_*, mcp_servers, memory_paths, messaging_socket_path, model, output_style, permissionMode, plugins, product_feedback_disabled, session_id, skills, slash_commands, terminal_slash_commands, tools, uuid`.

### Prompts that need the real API (describe only, not run)

Run these with `--target real --allow-spend`, rarely (for example once per minor release), with prompts as in the table:

- `text`, `thinking`, `tool_allow`, `tool_deny`, `file_edit`, `plan_mode`, `interrupt` and `resume` confirm that the mock shapes match reality.
- A real rate limit cannot be forced. Mock it.

### Recording format

Each transcript line looks like `{"ms":<since start>,"run":"1|2","dir":"in|out|harness","msg":<parsed>}`, with `raw` kept for unparseable lines. Both directions are recorded, so the host's own requests and responses become part of the golden.

---

## 3. Normalization, shapes and diffs

### Three golden artifacts per CLI version

1. **Sequence signature** (`shape.mjs seq`) for each scenario. Every message becomes a *key*, and runs of identical keys collapse to `key+` because the number of deltas is nondeterministic. A key is built as follows:
   - `dir:type[/subtype]`
   - `stream_event/<event.type>[/<delta.type|content_block.type>]`
   - `assistant|user/<sorted content block types>`
   - `control_request/<subtype>`
   - `control_response/<subtype of the matching request, correlated through request_id>/<success|error>`

   Diff with a normal line diff, treating floating system messages as an unordered set.
2. **Observed shape** (`shape.mjs infer`), a JSON-Schema-like tree for each message key. It is built by folding every observed instance:
   - The type set records unions such as `["integer","null"]`.
   - A field is optional if it was present in fewer instances than its parent object.
   - Literal values are kept **only** for discriminator keys (`type`, `subtype`, `status`, `mode`, `terminal_reason`, `behavior`, `stop_reason`, `hook_event_name`, `tool_name`, `error`, …), so they become `enum` lists.
   - Every other string becomes a format tag (`uuid`, `datetime`, `email`, `anthropic-id`, `path`) or plain `string`. Ids, timestamps, paths, user text and PII therefore never reach the golden.
   - Data-keyed objects are collapsed to `additionalProperties`. These are configured paths such as `rate_limits`, `modelUsage` and `mcpServers`, plus a heuristic for keys that look like model ids or paths.
   - Tool inputs and outputs (`can_use_tool.request.input`, `tool_input`, `tool_use_result`, `content[].input`) are recorded as `opaque` because they are per-tool schemas. The static half covers them through `sdk-tools.d.ts` or the zod schemas embedded in the binary.
3. **Catalog** (`shape.mjs catalog`): sorted value lists taken from the probe. These cover `initialize` keys, builtin and other commands, agents, output styles, models with their resolved ids, unavailable models, model capability keys, permission modes (from the error text), hook events, and accepted and rejected subtypes.

### Diff classification (`shape.mjs diff old.json new.json`)

- **BREAK** (the command exits 1): a field was removed, a type is no longer seen, or a field went from required to optional. These may crash or mis-render the plugin.
- **ADOPT**: a new message key, field, type or enum value. These are candidate UI features.
- **INFO**: a key was not observed (a coverage gap, or possibly a removal), or a field went from optional to always present.

Real output, isolated vs user-settings probe on the same version, showing the classifier on real data:

```
BREAK in:control_response/initialize/success.response.response.commands[].builtin  required -> optional
ADOPT in:control_response/get_settings/success.response.response.effective.enabledPlugins  new field (object)
ADOPT in:control_response/get_settings/success.response.response.effective.permissions  new field (object)
...
INFO  in:control_response/get_usage/success  message key not observed (scenario coverage or removal)
```

**Determinism check (real):** the full 15-scenario mock suite was run twice into `runs/mock` and `runs/mock2`. `shape.mjs diff` printed nothing, and every `seq.txt` was identical. On the mock, a diff is therefore attributable to the CLI version alone.

### Release pipeline

```
new binary B_new, previous B_old (both kept on disk)
1. probe B_new --mode isolated (x2, require stable)        -> catalog + shapes   (free; no generation calls)
2. mock suite with B_new (all scenarios)                    -> seq/*.txt + shapes (free, deterministic)
3. diff vs B_old goldens: shape diff, seq diff, catdiff     -> report: BREAK / ADOPT / INFO
4. if BREAK, or a new message key appears in a scenario that the mock cannot fake
   (e.g. new rate_limit/system subtype): run the real suite --target real --allow-spend (cheap, capped)
5. human triages ADOPT items; accepted -> new goldens committed under goldens/<version>/
```

To make step 2 useful as soon as a release appears, run the mock suite against **both** the old and new binaries on the same day. That separates CLI changes from changes in the mock or the harness.

---

## 4. Fake Messages API: verdict

**Verdict: feasible and worth doing. It was prototyped here and it works. It cannot replace the paid suite completely.**

What was proven:

- `ANTHROPIC_BASE_URL=http://127.0.0.1:8787` with `ANTHROPIC_API_KEY=sk-ant-mock-not-a-real-key` and `CLAUDE_CONFIG_DIR=<tmp>` sent **all** model traffic (`POST /v1/messages?beta=true`, `stream:true`, 27 tools, model `claude-haiku-4-5-20251001`) to the mock. The debug log lists only `http://127.0.0.1:8787`. The CLI also sends a `HEAD /api/hello` connectivity check.
- **All 15 scenarios ran to completion in under 1 s each and were deterministic.** That covers real tool execution (Bash, Write, Edit, ExitPlanMode, MCP tools), permission round-trips, hook callbacks, interrupts, `--resume` (history was replayed to the mock), max-turns, retries and API errors.
- With **OAuth** auth plus `ANTHROPIC_BASE_URL`, `count_tokens` traffic also went to the mock (18 requests). This means subscription-mode runs can probably be mocked too. That was checked without any prompt; no generation was sent under OAuth.

The traps found while prototyping:

- `--bare` restricts tools to Bash, Edit and Read (even with `--tools`), so it does not fit here. Isolate with `CLAUDE_CONFIG_DIR`, the dummy key and `--setting-sources=` instead.
- The mock must produce correct SSE: `message_start`, `content_block_start/delta/stop`, `message_delta` and `message_stop`, with `input_json_delta` for tools and `signature_delta` for thinking. The mock decides how far a conversation has progressed by counting assistant turns in the request, and picks the script from a `SCENARIO:<name>` marker in the first user message.
- `total_cost_usd` is computed locally from the mock's `usage`, so the value is fictitious but nonzero.

Limits, which are the reason the paid suite still exists:

1. The mock can only emit what the mock author already knows about. A new **server-side** content block or delta type (for example a new server tool or citation block) is not covered until the mock learns it. That gap is exactly what the rare paid run is for.
2. **`rate_limit_event` was not emitted** under API-key auth, even when the mock sent `anthropic-ratelimit-unified-status: allowed_warning` and the related headers that appear in the binary's strings. It most likely needs OAuth or subscription auth. Next step: run `rate_limit_warning` with OAuth plus the mock base URL, after the zero-cost routing check above. That check needs explicit user approval, because a misroute would use subscription quota.
3. Model behaviour such as whether Haiku actually calls the tool is not tested. Only the CLI's protocol is.
4. Side queries (title generation, prompt suggestions) get a generic `ok`. If a release adds a new side query whose output appears in the stream, the mock must be taught it.

Recommendation: the mock suite runs on **every** release. The paid suite runs on minor releases, or when the mock diff or the static half reports a new stream or content type.

---

## 5. Tolerant-reader rules for the plugin at runtime

1. **Never crash on unknown input.** Unknown `type`, unknown `system.subtype`, unknown `stream_event.event.type`, delta type or content block type: log once per (type, subtype, CLI version) with a sample, then ignore. Keep a small "unknown messages" debug panel so drift is visible in dogfooding.
2. **Answer every `control_request` from the CLI.** For an unknown subtype, reply `{"subtype":"error","request_id","error":"unsupported: <subtype>"}`. A request left unanswered can block a tool call or turn. `can_use_tool` must always be answered; on an internal failure, default to deny with a message.
3. Correlate `control_response` by `request_id`, and put a timeout on every host request. Treat `"Unsupported control request subtype: X"` as a feature-detection signal (disable that UI affordance) rather than an error.
4. **Decide success from `result.is_error` and `terminal_reason`, never from `subtype` alone.** API errors arrive as `subtype:"success"` with `is_error:true`. Show `api_error_status` and `result` text for those.
5. Treat every field as optional unless the plugin cannot work without it. Real example: `commands[].builtin` is missing on plugin commands, and `first_content_frame_ms` and `ttft_ms` on `result` are sometimes missing. Use defaults instead of assertions.
6. Do not assume ordering between `stream_event`s, the complete `assistant` message, `hook_callback`/`can_use_tool` and `system/status`. Key UI state by `uuid`, `message.id` and `tool_use_id`, and make updates idempotent. Partial text from `stream_event` must be replaced by the final `assistant` message.
7. Treat enums as open sets (`terminal_reason`, `permissionMode`, `api_retry.error`, `permission_suggestions[].type`). Unknown values get a generic rendering. Fill model, permission-mode and output-style pickers from `initialize` and `list_models`, not from constants, and respect `disabled:true` / `unavailable_models`.
8. Ignore `keep_alive` and honour `control_cancel_request` by dismissing the matching pending permission prompt. Neither was observed in these runs, so add a mock scenario for them once their triggers are known.
9. Pass through what you do not understand: when allowing a tool, return `updatedInput` equal to the original `input` unless the user edited it, and do not drop unknown keys.
10. Record the CLI version (`system/init.claude_code_version` and the `initialize` clientInfo) with every logged unknown, so field reports map to releases.

---

## Open items / next steps

- **Run the probe and mock suite against the newer 2.1.325 binary.** Its darwin-arm64 package was downloaded to `reference/cli-bins/2.1.325/anthropic-ai-claude-agent-sdk-darwin-arm64-0.3.325.tgz`. The checksum is in `sdk-pkgs/v0.3.325/package/manifest.json`. Extracting and executing it was **blocked by the permission classifier** in this session, so a real cross-version diff still needs the user's approval. With approval, the commands are:
  ```sh
  node probe.mjs --bin <2.1.325>/package/claude --hermetic http://127.0.0.1:8787 --out runs/2.1.325/hermetic --cwd <new empty>
  for s in …; do node run-scenario.mjs --bin <2.1.325>/package/claude --target mock --mock-url http://127.0.0.1:8787 --scenario $s --out runs-2.1.325/mock/$s; done
  node shape.mjs diff runs/mock/shapes.json runs-2.1.325/mock/shapes.json
  ```
  The 2.1.278 `initialize` already says "Update to 2.1.280+ to use Opus 5.5", so new model metadata is expected.
- Add mock scenarios for subagents (the `Agent` tool, `parent_tool_use_id`), `AskUserQuestion`, elicitation (`elicitation` control request), compaction (`compact_boundary`), background tasks, and `--prompt-suggestions` (`prompt_suggestion` message). All of these subtypes appear in the binary's strings.
- Run `claude --help` and diff its output on every release (baseline saved in `runs/2.1.278/help.txt`). It is free and catches new flags (for example `--permission-prompts`, `--forward-subagent-text`, `--include-hook-events`).
