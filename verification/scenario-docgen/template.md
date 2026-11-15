# Scenario recordings on CLI 2.1.311 (mock API)

Real `claude` 2.1.311 output for features that had no runtime evidence: subagents and the `task_*` messages, background tasks, `stop_task`, permission prompts from subagents, interrupting a pending approval, `AskUserQuestion`, `/compact`, queued user messages, prompt-type slash commands and `get_task_output`.

Everything here was produced by `reference/cli-bins/2.1.311/x/package/claude` (`--version` prints `2.1.311 (Claude Code)`) driven over stream-json by `run-scenario.mjs`, with the model replaced by `mock-api.mjs`. No real API call was made: the dummy key and throwaway `CLAUDE_CONFIG_DIR` were kept, the mock logged 152 `POST /v1/messages` requests (recording and determinism passes together), all with user agent `claude-cli/2.1.311 (external, sdk-cli)`, and the debug logs name no Anthropic host.

- Recordings: `runs/v2.1.311/mock/<scenario>/` (`transcript.ndjson`, `seq.txt`, `debug-1.log`, `meta.json`). 22 new directories; the 14 earlier ones are untouched.
- Mock request logs: `runs/v2.1.311/mock-requests-scenarios2.ndjson` and `mock-bodies-scenarios2.ndjson` (21 scenarios, each run twice: the recording, then a determinism pass written to a scratch directory), plus `…-scenarios2b.ndjson` for `subagent_failed`. The bodies file holds the `messages` array of every request, which is how the "what the model receives" excerpts below were obtained.
- Shapes: `runs/v2.1.311/mock/shapes-extended.json` is inferred from all 36 transcripts. `shapes.json` (the earlier 14) was left as it was. `node shape.mjs diff shapes.json shapes-extended.json` lists every key and field that is new; that list is reproduced in "Index of new keys and fields".
- Determinism: every new scenario was run twice and produced an identical `seq.txt`. The 14 earlier scenarios were re-run with the modified harness into a scratch directory and their sequences match the existing recordings.

The JSON examples are copied from the transcripts by a script. Ids are shortened (`6b471cf6…` is a uuid, `<run>` is the scenario directory, `<tmp>/…` is the CLI's task-output directory), strings longer than 230 characters are cut and marked `…[+N chars]`, and `usage`, `modelUsage` and `model_usage_since_last_result` are left out of `result` examples because their shape did not change. Field names and all other values are exact.

## Harness changes

`mock-api.mjs`
- New scripts for every scenario below. A subagent's requests are dispatched like any other: its first user message is the `prompt` from the Agent tool input, so the parent script puts a second marker there (`SCENARIO:subagent_child`, `…_slow`, `…_stuck`, `…_perm`, `…_outer`, `…_error`).
- A turn can carry `delay` (ms before the response starts), in addition to the existing `slow`. This is what keeps a child "running" long enough for the host to act.
- `__AGENT__` as a tool name resolves to `Agent` or `Task`, whichever the request's tool list offers. On 2.1.311 the API tool is `Agent`.
- If the first user message has no marker, the `x-mock-scenario` request header selects the script. Only the two slash-command scenarios use it, through `ANTHROPIC_CUSTOM_HEADERS`, because a prompt-type command replaces the user text.
- `--bodies <file>` dumps request bodies (without `tools` and `system`). The request log gained `first_user_head`, `last_user_head` and `via_header`.

`run-scenario.mjs`
- Scenario options: `init` (extra `initialize` fields), `on(m, api)` (a per-message hook that can send control requests, write user messages, or suppress the default reply), `quietEnd` (close stdin only after a `result` followed by N ms of silence, for work that continues after a result), `stampUuid` (give host user messages a `uuid`).
- Records `harness:harness_stdin_end` when a `quietEnd` scenario closes stdin.
- Refuses to write into a directory that already holds a `transcript.ndjson` unless `--force` is given. Before this, re-running a scenario whose cwd happened to be empty (for example `text`) would have truncated the existing recording.
- The hermetic environment handling is unchanged.

`compare.sh`
- The 22 working scenarios are in the scenario list. `MOCK` can be overridden from the environment, and `seq.txt` is written for each scenario.

## Results at a glance

| # | Scenario | Worked | What was captured |
|---|---|---|---|
| 1 | `subagent` | yes | `task_started`, `task_progress`, `task_updated`, `task_notification`; child messages with `parent_tool_use_id` and `agent_id` |
| 1a | `subagent_forward_text` | yes | the same with `--forward-subagent-text`: child text blocks appear |
| 1b | `subagent_nested` | yes | `spawn_depth: 2`, `parent_task_id`; the inner agent's messages are not forwarded |
| 1c | `subagent_failed` | yes | `patch.status: "failed"`, `patch.error`, `task_notification.status: "failed"` |
| 2 | `subagent_background` | yes | `background_tasks_changed`, the async tool_result, the notification-triggered turn and its `result.origin` |
| 2a | `subagent_parallel` | yes | two background agents, `tasks` with two entries, two notification turns |
| 2b | `subagent_to_background` | yes | host `background_tasks` request, `task_updated` with `patch.is_backgrounded` |
| 3 | `subagent_stop` | yes | `stop_task` on a foreground agent: `killed` / `stopped` |
| 3a | `subagent_stop_bg` | yes | `stop_task` on a background agent, a repeated stop, and a stop of an unknown id |
| 3b | `subagent_interrupt` | yes | `interrupt` while a foreground agent runs |
| 3c | `subagent_bg_interrupt`, `subagent_bg_interrupt_affordance` | yes | `interrupt` while a background agent runs, without and with `perTaskStopAffordance` |
| 4 | `subagent_permission` | yes | `can_use_tool` from a subagent carries `agent_id` |
| 5 | `interrupt_pending_approval` | yes | the CLI sends `control_cancel_request` |
| 6 | `ask_user_question` | yes | `can_use_tool` for AskUserQuestion, `updatedInput.answers`, the tool_result |
| 7 | `compact` | yes | `status: "compacting"`, `compact_result`, `compact_boundary`, the result with `local_command` |
| 8 | `queued_message`, `queued_interrupt`, `queued_cancel` | yes | `command_lifecycle`, `user_message_uuid(s)`, `still_queued`, `cancelled` |
| 9 | `slash_prompt_command`, `slash_prompt_command_replay` | yes | the expansion is not in the stream; with `--replay-user-messages` only a command marker is |
| 10 | `task_output` | yes | `get_task_output` during and after a background shell command, and its error |
| – | `agentProgressSummaries` probe | no data | see "Not covered" |

## How to read the sequences

Each sequence is built from the transcript with the same message keys as `shape.mjs seq` (`dir:type/subtype`, `control_response/<request subtype>/<success|error>`, `user|assistant/<block types>`). Three things are added for readability, and the unannotated signature is in `<scenario>/seq.txt`:

- consecutive `stream_event` keys are collapsed to `in:stream_event/*+`;
- a `hook_callback` request and the harness's empty reply are one line;
- a note in parentheses gives normalized values: agents are lettered `A`, `B` in order of appearance (the letter stands for the `task_id`, which equals the `agent_id`), host user messages with a uuid are `M1`, `M2`.

A line with `agent A` is a message that carries `parent_tool_use_id` and `agent_id`. Lines without it are main-thread messages (`parent_tool_use_id: null`).

---

## 1. `subagent`: a foreground Agent call

The main agent calls `Agent` with `run_in_background: false`. The child makes one Bash call and reports.

{{seq subagent}}

Observations:

- The tool is called `Agent` in the API request and in the `tool_use` block. `system/init.tools` still lists it as `Task`.
- No `background_tasks_changed` is emitted for a foreground agent.
- No `stream_event` is emitted for the child. Across all 22 recordings, 0 of 519 stream events have a non-null `parent_tool_use_id`, with `--include-partial-messages` on.
- The child's text is not forwarded while it runs in the foreground: its first reply was a text block plus a tool_use block, and only the tool_use block arrived. Its final report arrives only inside the Agent tool_result and in `task_notification.summary`. See 1a.
- `task_progress` arrived once, for the child's tool call. Its position relative to the child's `assistant` message is not fixed (before it here, after it in the background runs).
- `task_updated` then `task_notification` arrive before the Agent `tool_result`.

`system/task_started`:

{{ex task_started_fg}}

The child's prompt, echoed as a `user` message (only when the agent starts in the foreground):

{{ex child_user_prompt}}

`system/task_progress`:

{{ex task_progress}}

The `PreToolUse` hook callback for the child's tool call. `input.agent_id` and `input.agent_type` are present only for calls made inside a subagent; the callback for the main thread's own `Agent` call has neither:

{{ex hook_child}}

The child's `assistant` message and the child's tool result. Compared with a main-thread message, `parent_tool_use_id` is non-null and there are three extra top-level fields: `agent_id`, `subagent_type` and `task_description`. A forwarded child tool_result has no `tool_use_result`:

{{ex child_assistant}}

{{ex child_tool_result}}

`system/task_updated` and `system/task_notification`:

{{ex task_updated_completed}}

{{ex task_notification_completed}}

The Agent tool_result on the main thread. `tool_use_result` is the structured form:

{{ex agent_tool_result_fg}}

The `result`. `subagent_stats` was already present in earlier recordings but always zero:

{{ex result_fg}}

### 1a. `subagent_forward_text`: the same with `--forward-subagent-text`

{{seq subagent_forward_text}}

With the flag, the child's text blocks are forwarded as separate `assistant` messages with the same attribution fields:

{{ex child_assistant_text}}

### 1b. `subagent_nested`: an agent that spawns an agent

{{seq subagent_nested}}

- The inner agent's `task_started` has `spawn_depth: 2` and `parent_task_id` set to the outer agent's `task_id`. `parent_task_id` is absent on a depth-1 agent.
- **None of the inner agent's own messages are forwarded**: no prompt echo, no `assistant`, no tool_result. The stream shows its `task_*` messages and the hook callback for its tool call (with the inner `agent_id`), nothing else.
- The outer agent's `Agent` tool_use and the matching tool_result are forwarded as the outer agent's messages (`agent A`). The inner task's `tool_use_id` is the id of that forwarded tool_use block.
- When an agent calls `Agent`, its `task_progress` has `last_tool_name: "Agent"` and `description` set to the inner call's description.

{{ex task_started_nested}}

{{ex task_progress_agent}}

### 1c. `subagent_failed`: the child's model request fails

The mock answers the child's request with HTTP 400.

{{seq subagent_failed}}

{{ex task_updated_failed}}

{{ex task_notification_failed}}

There is no `task_progress` and no `usage` on the notification. The Agent tool_result is `is_error: true` with the same text, and the result has `subagent_stats.failed: 1`.

---

## 2. `subagent_background`: Agent with `run_in_background: true`

The child's two replies are delayed (1.5 s and 0.5 s) so that the main turn finishes first.

{{seq subagent_background}}

Observations:

- `background_tasks_changed` (with the new task) arrives before `task_started`, and the Agent tool_result follows immediately with an "Async agent launched" text. The main turn then continues and ends with a normal `result` while the agent is still running.
- The child's prompt is **not** echoed as a `user` message for an agent that starts in the background.
- The child's **text blocks are forwarded** for a background agent, without `--forward-subagent-text` (the `in:assistant/text (agent A)` lines). This differs from the foreground case.
- On completion: `task_updated` → `task_notification` → `background_tasks_changed` with `tasks: []`.
- The notification then starts a new main-thread turn by itself. That turn begins with a second `system/init`, has no `user` message in the stream, and ends with a `result` that has `origin` and the next `result_index`.

`system/background_tasks_changed`, at start and at the end:

{{ex bg_changed}}

{{ex bg_changed_empty}}

`system/task_started` (`is_backgrounded: true`):

{{ex task_started_bg}}

The Agent tool_result for a background launch:

{{ex agent_tool_result_bg}}

The `result` of the notification-triggered turn. `origin.runId` equals the task's `run_id`:

{{ex result_notification}}

What the model receives as the user message of that turn (from the mock's body dump; `---` separates content blocks):

{{body subagent_background 2}}

### 2a. `subagent_parallel`: two background agents from one assistant message

{{seq subagent_parallel}}

`background_tasks_changed` is emitted on every membership change with the full set. With two agents running:

{{ex bg_changed_two}}

Each completion produced its own notification turn and its own `result` (`result_index` 1 and 2).

### 2b. `subagent_to_background`: the host moves a foreground agent to the background

While a foreground agent runs, the host sends `background_tasks` with the Agent call's `tool_use_id`.

{{seq subagent_to_background}}

{{ex bg_req}}

{{ex task_updated_bg}}

{{ex bg_resp}}

The Agent call returns at once with the same "Async agent launched" tool_result as in scenario 2, the turn continues, and from then on the agent behaves like a background agent: its text is forwarded, and its completion triggers a notification turn. No second `task_started` is sent.

---

## 3. `subagent_stop`: `stop_task` while a subagent runs

The child's second model request hangs in the mock (15 s). The host sends `stop_task` 2.5 s after `task_started`.

{{seq subagent_stop}}

{{ex stop_req}}

{{ex task_updated_killed}}

{{ex task_notification_stopped}}

{{ex stop_resp}}

- The two statuses differ by design: `task_updated.patch.status` is `"killed"`, `task_notification.status` is `"stopped"`.
- `task_updated` and `task_notification` arrive before the `control_response` to `stop_task`.
- For a stopped task the notification's `summary` is the task description and there is no `usage`.
- The response payload is empty.
- The foreground Agent call ends with an error tool_result and the main turn continues normally (`result/success`, `subagent_stats.killed.user: 1`):

{{ex stop_tool_result}}

### 3a. `subagent_stop_bg`: stopping a background agent

`initialize` declares `perTaskStopAffordance: true`. The host stops the agent after the main turn has ended, then sends `stop_task` again for the same task, then for an id that does not exist.

{{seq subagent_stop_bg}}

- Same `killed` / `stopped` pair, followed by `background_tasks_changed` with `tasks: []`, all before the `control_response`.
- After the stop, a forwarded child `user` message with the text `[Request interrupted by user]` arrives. A stopped foreground agent does not produce one.

{{ex stop_bg_child_user}}

- The stop starts a notification turn, like a completion does. The model's notification text says `<status>killed</status>` and `Agent "…" was stopped by user`:

{{body subagent_stop_bg 2}}

- **`stop_task` always answers `success` with an empty payload**: for a running task, for a task already stopped, and for `task_id: "no-such-task"`. The response cannot be used to tell whether anything was stopped; `task_updated` / `task_notification` can.

### 3b. `subagent_interrupt`: `interrupt` while a foreground agent runs

{{seq subagent_interrupt}}

The interrupt kills the foreground agent (`killed` / `stopped`, `subagent_stats.killed.user: 1`) and aborts the turn: `result/error_during_execution` with `terminal_reason: "aborted_tools"`. Here the `control_response` to `interrupt` comes before the task messages.

### 3c. `interrupt` while a background agent runs

The main turn is streaming text when the host interrupts; a background agent is mid-run.

Without `perTaskStopAffordance` (`subagent_bg_interrupt`), the interrupt also kills the background agent:

{{seq subagent_bg_interrupt}}

`subagent_stats.killed` is `{"parent":0,"user":0,"system":1}` in this result. No notification turn follows.

With `perTaskStopAffordance: true` in `initialize` (`subagent_bg_interrupt_affordance`), the interrupt only aborts the turn and the agent keeps running until the host sends `stop_task`:

{{seq subagent_bg_interrupt_affordance}}

---

## 4. `subagent_permission`: a subagent's tool needs approval

The child calls `Bash` with `touch child-marker`.

{{seq subagent_permission}}

**The `can_use_tool` request carries `agent_id`.** It is the only field that ties the prompt to the subagent: there is no `parent_tool_use_id`, `subagent_type` or task description on the request. `agent_id` equals the `task_id` of the `task_started` message, whose `tool_use_id` is the Agent call.

{{ex can_use_tool_child}}

The reply format is unchanged (`{"behavior":"allow","updatedInput":…}`). The child's tool_result records the decision source as `user_temporary`:

{{ex child_tool_result_perm}}

---

## 5. `interrupt_pending_approval`: interrupt instead of answering

The main agent calls `Bash` with `touch pending-marker`. The harness leaves the `can_use_tool` unanswered and sends `interrupt` 400 ms later.

{{seq interrupt_pending_approval}}

**Yes, the CLI sends `control_cancel_request` for the pending `can_use_tool`.** Order, as recorded (ms since start: 150, 551, 554, 555, 558, 559, 561):

1. in `control_request/can_use_tool` (request_id `d7445a77…`)
2. out `control_request/interrupt`
3. in `control_cancel_request` with the same `request_id`
4. in `control_response` for the interrupt
5. in `user` tool_result (rejected), then `user` text `[Request interrupted by user for tool use]`
6. in `result/error_during_execution`

{{ex pend_can}}

{{ex pend_cancel}}

{{ex pend_int_resp}}

{{ex pend_tool_result}}

{{ex pend_result}}

The host never answered the `can_use_tool`, and nothing else was needed: the CLI did not wait for an answer. The denied call is listed in `permission_denials`, and `terminal_reason` is `"aborted_tools"` (the earlier `interrupt` recording, with no tool pending, has `"aborted_streaming"`).

---

## 6. `ask_user_question`

{{seq ask_user_question}}

The `can_use_tool` request. Compared with a Bash request it has `requires_user_interaction: true` and no `description`, `permission_suggestions` or `blocked_path`:

{{ex ask_can}}

The harness answers with `allow` and puts the answers in `updatedInput.answers`, keyed by the question text:

{{ex ask_resp}}

The tool_result. `message.content[0].content` is the string the model receives (confirmed in the mock's body dump), and `tool_use_result` is the structured form:

{{ex ask_tool_result}}

---

## 7. `compact`: the user message `/compact`

Three host messages: a normal turn, `/compact`, then a normal turn.

{{seq compact}}

{{ex compact_status1}}

{{ex compact_status2}}

{{ex compact_boundary}}

Two inbound `user` messages with string content follow. The first is the synthetic summary message that replaces the history; the second is the command's output:

{{ex compact_user_synth}}

{{ex compact_user_stdout}}

The `result` for the command:

{{ex compact_result}}

- This `result/success` has `local_command: "compact"`, `num_turns: 0`, `result: ""`, and **no `terminal_reason` and no `api_error_status`**. Both fields were present on every earlier result, so the shape diff reports them as required → optional.
- The summarisation is one streaming `/v1/messages` request whose last user message starts with `CRITICAL: Respond with TEXT ONLY. Do NOT call any tools.` None of its stream events or its assistant message reach the host.
- `system/init` is emitted again before `compact_boundary`, and once more at the start of the next turn.
- These `user` messages appear without `--replay-user-messages`.

---

## 8. Queued user messages

All three scenarios give host user messages a `uuid`. The first turn streams slowly; the second user message is written after the third delta.

### `queued_message`

{{seq queued_message}}

- The queue is reported through a message type of its own, **`command_lifecycle`**, with `state` `queued` → `started` → `completed` (or `cancelled`). It has no type declaration in `sdk.d.ts` 0.3.311 (the name occurs once, inside a comment); `system/init.capabilities` advertises `msg_lifecycle_v1`. Every uuid-stamped user message gets these frames, including the first one.
- There is no `queued_command` message in the stream.
- The queued message is not folded into the running turn. It runs as its own turn after the first `result`, with its own `system/init` and `result`.
- **`result.queued_turn_count` was 0 on both results**, including the first one, which was produced while M2 was still queued. The 0.3.311 type comment describes it as the number of sends still waiting. Do not use it to decide whether another turn follows; use `command_lifecycle`.
- With uuids, turns are attributable: `system/init` and `system/status` carry `user_message_uuids`; `stream_event/message_start`, `assistant` and `result` carry `user_message_uuid` and `user_message_uuids`. The other stream events do not.

The host's second message and its lifecycle frames:

{{ex q_user2}}

{{ex q_life_queued}}

{{ex q_life_started}}

{{ex q_life_completed}}

{{ex q_status}}

{{ex q_result}}

### `queued_interrupt`: plain `interrupt` with a message queued

{{seq queued_interrupt}}

{{ex qi_resp}}

The queued message survives, is listed in `still_queued`, and runs as the next turn. The interrupted turn's own message gets `state: "cancelled"` after its result:

{{ex qi_life_cancelled}}

### `queued_cancel`: `interrupt` with `cancel_queued: true`

{{seq queued_cancel}}

{{ex qc_req}}

{{ex qc_life}}

{{ex qc_resp}}

The `cancelled` lifecycle frame for the queued message arrives before the `control_response`. No second turn runs; the only result is the aborted one:

{{ex qc_result}}

---

## 9. `slash_prompt_command`: `/init`

{{seq slash_prompt_command}}

**The expansion does not appear in the stream.** The host sees an ordinary turn. The mock's request log shows what was sent to the model: the first user message is a `<command-message>init</command-message>\n<command-name>/init</command-name>` block followed by the full `/init` prompt text ("Please analyze this codebase and create a CLAUDE.md file…").

With `--replay-user-messages` (`slash_prompt_command_replay`) one extra `user` message appears, holding the command marker only, not the expanded prompt:

{{seq slash_prompt_command_replay}}

{{ex slash_replay}}

---

## 10. `task_output`: `get_task_output` for a background shell command

The main agent runs `echo bg-line-1; sleep 2; echo bg-line-2` with `run_in_background: true`. The host calls `get_task_output` 0.5 s after `task_started`, again after `task_notification`, and once with an unknown id.

{{seq task_output}}

A background shell command is a task of `task_type: "local_bash"`. Its `task_started` has no `subagent_type`, `spawn_depth` or `prompt`, and it produces no `task_progress`:

{{ex bash_bg_changed}}

{{ex bash_task_started}}

{{ex bash_tool_result}}

{{ex out_req}}

While running, after the task ended, and for an unknown id:

{{ex out_resp1}}

{{ex out_resp2}}

{{ex out_resp3}}

After the command exits, the output ends with a `[exited with code 0]` line that the command did not print.

{{ex bash_task_notification}}

Completion also starts a notification turn (`result.origin.kind: "task-notification"`).

---

## Not covered

- **`task_progress.summary`** (`initialize.agentProgressSummaries: true`). Probed with a subagent that stayed alive for 16.5 s: no summary request reached the mock and no `summary` field appeared. The trigger is probably a longer timer. The probe is the `subagent_progress_summary` scenario in `run-scenario.mjs`; it is not in `compare.sh` and no recording was kept. A proper test needs a child that runs for a minute or more and a mock script for the summary request, whose wording is not yet known.
- `task_updated.patch.status` values `pending`, `running`, `paused`, and `patch.description` / `patch.total_paused_ms`. Only `completed`, `killed`, `failed`, `end_time`, `error` and `is_backgrounded` were seen.
- `ambient`, `skip_transcript`, `workflow_name`, `resource_links`, `reason` on task messages, and `parent_task_id` inside `background_tasks_changed` (the nested agent ran in the foreground, so it never entered that list).
- `tool_progress` and `keep_alive` did not appear in any recording.
- A subagent tool call that is denied, a `can_use_tool` from a background subagent, and `control_cancel_request` for a subagent's pending prompt.
- `/compact` triggered automatically (`trigger: "auto"`).

## Index of new keys and fields

From `node shape.mjs diff runs/v2.1.311/mock/shapes.json runs/v2.1.311/mock/shapes-extended.json`.

New message keys:

| Key | Section |
|---|---|
| `in:system/task_started`, `task_progress`, `task_updated`, `task_notification` | 1 |
| `in:system/background_tasks_changed` | 2 |
| `out:control_request/stop_task`, `in:control_response/stop_task/success` | 3 |
| `out:control_request/background_tasks`, `in:control_response/background_tasks/success` | 2b |
| `in:control_cancel_request` | 5 |
| `in:system/compact_boundary`, `in:user/string` | 7 |
| `in:command_lifecycle` | 8 |
| `out:control_request/get_task_output`, `in:control_response/get_task_output/success`, `…/error` | 10 |
| `harness:harness_stdin_end` | harness marker, not a CLI message |

New fields on keys that were already recorded:

| Message | Field | Section |
|---|---|---|
| `assistant`, `user` (forwarded from a subagent) | `agent_id`, `subagent_type`, `task_description`; `parent_tool_use_id` non-null | 1 |
| `control_request/can_use_tool` | `agent_id`; `tool_name: "AskUserQuestion"` with `requires_user_interaction` | 4, 6 |
| `control_request/hook_callback` | `input.agent_id`, `input.agent_type` | 1 |
| `control_response/interrupt/success` | `response.cancelled` | 8 |
| `result/success` | `origin`, `local_command`, `user_message_uuid`, `user_message_uuids`, `request_sent_wall_ms`, `first_request_input_tokens` | 2, 7, 8 |
| `result/error_during_execution` | `terminal_reason: "aborted_tools"`, `stop_reason: "tool_use"`, `user_message_uuid(s)` | 5, 8 |
| `system/status` | `status: "compacting"`, `compact_result`, `user_message_uuids` | 7, 8 |
| `system/init` | `user_message_uuids` | 8 |
| `stream_event/message_start`, `assistant` | `user_message_uuid`, `user_message_uuids` | 8 |
| `out:control_request/initialize` | `perTaskStopAffordance` | 3 |
| `out:control_request/interrupt` | `cancel_queued` | 8 |
| `out:user` | `uuid` | 8 |

Fields that stopped being always present (reported as BREAK by the diff; all are consequences of new cases, not removals):

| Message | Field | Why |
|---|---|---|
| `result/success` | `terminal_reason`, `api_error_status` | absent on the `/compact` result |
| `user/tool_result` | `tool_use_result` | absent on tool results forwarded from a subagent |
| `user/tool_result` | `tool_result_meta[].permission_decision` | absent when the entry has `non_execution_kind: "user-rejected"` |

---

## Agent map: verified message fields on 2.1.311

Counts over the 22 new recordings: 16 `task_started` (15 `local_agent`, 1 `local_bash`), 14 `task_progress`, 17 `task_updated`, 16 `task_notification`, 16 `background_tasks_changed`. All five are `type: "system"` and all carry `uuid` and `session_id` (the main session's id, also for nested agents).

### `task_started`

| Field | Type | Present | Observed |
|---|---|---|---|
| `task_id` | string | always | `local_agent`: `a` + 16 hex (`a66645f303da6f90e`), equal to the agent's `agent_id`. `local_bash`: `b` + 8 characters (`bs47pyb1e`) |
| `run_id` | string | always | `0mwi7r73f-76524bb7`; the same on every `task_*` message and `background_tasks_changed` entry of the run, and equal to `result.origin.runId` |
| `tool_use_id` | string | always | id of the `Agent` / `Bash` tool_use block that started the task |
| `description` | string | always | the tool input's `description` |
| `task_type` | string | always | `"local_agent"`, `"local_bash"` |
| `is_backgrounded` | boolean | always | `false` for a foreground Agent call, `true` for `run_in_background: true` |
| `subagent_type` | string | `local_agent` only | `"general-purpose"`, `"Explore"` |
| `spawn_depth` | integer | `local_agent` only | `1`, and `2` for an agent spawned by an agent |
| `parent_task_id` | string | only when spawned by another agent | the spawning agent's `task_id` |
| `prompt` | string | `local_agent` only | the full Agent prompt |

Not emitted for a foreground Bash call. Emitted once per task; moving a task to the background does not emit a second one.

### `task_progress`

| Field | Type | Present | Observed |
|---|---|---|---|
| `task_id`, `run_id`, `tool_use_id` | string | always | as on `task_started` |
| `description` | string | always | **not the task description**: a description of the current activity, `"Running <Bash description>"` for a Bash call, the inner call's description for an `Agent` call |
| `subagent_type` | string | always | as on `task_started` |
| `last_tool_name` | string | always | `"Bash"`, `"Agent"` |
| `usage` | object | always | `{total_tokens, tool_uses, duration_ms}`, cumulative for the task |
| `summary` | string | never seen | see "Not covered" |

Emitted once per tool call made by a `local_agent` task. Never emitted for `local_bash`, and not emitted at all when the agent makes no tool call (`subagent_failed`).

### `task_updated`

| Field | Type | Present | Observed |
|---|---|---|---|
| `task_id`, `run_id` | string | always | no `tool_use_id` on this message |
| `patch.status` | string | on a terminal change | `"completed"`, `"killed"` (stop_task or interrupt), `"failed"` |
| `patch.end_time` | integer | with `patch.status` | epoch milliseconds |
| `patch.error` | string | with `status: "failed"` | the error text |
| `patch.is_backgrounded` | boolean | when a foreground task is moved to the background | `true`, alone in the patch |

### `task_notification`

| Field | Type | Present | Observed |
|---|---|---|---|
| `task_id`, `run_id`, `tool_use_id` | string | always | as on `task_started` |
| `status` | string | always | `"completed"`, `"stopped"`, `"failed"`. A task whose `task_updated` says `killed` is `stopped` here |
| `summary` | string | always | completed agent: its final text. Stopped: the task description. Failed: the error text. `local_bash`: `Background command "<description>" completed (exit code 0)` |
| `output_file` | string | always | `<tmp>/<session id>/tasks/<task_id>.output` |
| `usage` | object | completed `local_agent` only | `{total_tokens, tool_uses, duration_ms}`; absent for stopped, failed and `local_bash` |

Emitted for foreground tasks too, just before the Agent tool_result. Always directly after the `task_updated` that carries the terminal status.

### `background_tasks_changed`

| Field | Type | Present | Observed |
|---|---|---|---|
| `tasks` | array | always | the full live set after the change; `[]` when the last one ends |
| `tasks[].task_id`, `tasks[].run_id` | string | always | as on `task_started` |
| `tasks[].task_type` | string | always | `"local_agent"`, `"local_bash"` |
| `tasks[].description` | string | always | the task description |
| `tasks[].subagent_type` | string | `local_agent` only | |

Entries have no `tool_use_id`, `status` or `is_backgrounded`. Emitted only for background work: never for an agent that stays in the foreground.

### Ordering, as recorded

| Event | Order |
|---|---|
| Foreground agent start | `assistant` (Agent tool_use) → `task_started` → child prompt as `user` |
| Foreground agent end | `task_updated` → `task_notification` → Agent `tool_result` on the main thread |
| Background agent start | `background_tasks_changed` → `task_started` → Agent `tool_result` ("Async agent launched") |
| Background agent end | `task_updated` → `task_notification` → `background_tasks_changed` → new main turn (`system/init` … `result` with `origin`) |
| Host `background_tasks` | `background_tasks_changed` → `task_updated {is_backgrounded:true}` → `control_response` → Agent `tool_result` |
| Host `stop_task` | `task_updated {status:"killed"}` → `task_notification {status:"stopped"}` → (`background_tasks_changed`) → `control_response` |
| Host `interrupt`, foreground agent | `control_response` → `task_updated` → `task_notification` → `tool_result` → `result/error_during_execution` |
| Host `interrupt`, background agent, no `perTaskStopAffordance` | `task_updated` → `task_notification` → `background_tasks_changed` → `control_response` → aborted turn's `result` |
| Host `interrupt`, background agent, `perTaskStopAffordance: true` | the agent is untouched; only the turn is aborted |

The SDK type comments say the order between `background_tasks_changed` and the `task_*` messages is unspecified. Treat the rows above as what 2.1.311 did, and keep the map keyed by `task_id`.

### How `parent_tool_use_id` and `agent_id` appear

| Where | `parent_tool_use_id` | `agent_id` | Notes |
|---|---|---|---|
| Main-thread `assistant` / `user` | `null` | absent | |
| `assistant` / `user` forwarded from a depth-1 agent | the `Agent` tool_use id on the main thread (= `task_started.tool_use_id`) | the agent's id (= `task_id`) | also `subagent_type` and `task_description`; `session_id` is the main session |
| Messages of a depth-2 agent | – | – | **not forwarded at all**; only its `task_*` messages and hook callbacks exist |
| `stream_event` | always `null` | absent | no partial messages for subagents |
| `task_*`, `background_tasks_changed` | absent | absent | use `task_id` (same value as `agent_id`) and `tool_use_id` |
| `control_request/can_use_tool` from a subagent | absent | present | the only link to the agent |
| `hook_callback` input from a subagent | absent | present, with `agent_type` | absent for main-thread tool calls |
| Agent `tool_result` on the main thread | `null` | absent | `tool_use_result.agentId` holds the id |

Which child messages are forwarded:

| Child message | Foreground agent | Foreground + `--forward-subagent-text` | Background agent |
|---|---|---|---|
| prompt echo (`user` text) | yes | yes | no |
| `assistant` tool_use blocks | yes | yes | yes |
| `user` tool_result | yes | yes | yes |
| `assistant` text blocks | no | yes | yes |
| `[Request interrupted by user]` (`user` text) after a stop | no | not tested | yes |

One `assistant` message per content block, as on the main thread.

To attach a forwarded message to a node: `agent_id` → the task with that `task_id`. To attach a task to its place in the transcript: `task_started.tool_use_id` → the `Agent` tool_use block, which for a nested agent is a block in a message forwarded from its parent (`parent_task_id`).
