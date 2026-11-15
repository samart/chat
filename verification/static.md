# Static half of a CLI change-verification pipeline

Goal: on every Claude Code CLI or Agent SDK release, build a machine-diffable **protocol surface manifest** without running the model, diff it against the previous release, and classify each change so the chat-panel owner can decide what to adopt.

Everything below was run for real on 2026-11-15 against SDK 0.3.278 / 0.3.300 / 0.3.325, CLI 2.1.278 (local) and 2.1.325 (VSIX). No prompt was sent to any model.

Scratchpad root (abbreviated below as `$S`):
`/Users/user/Development/ai/chat-interface-specs`

| What | Absolute path |
|---|---|
| Extractors and diff tool | `$S/tools/` (`extract-sdk.mjs`, `extract-bundle.py`, `extract-cli-help.py`, `extract-settings-schema.py`, `extract-changelog.py`, `diff-manifest.py`) |
| Manifests | `$S/verification/manifests/*.json` |
| Diffs | `$S/verification/diff-sdk-278-300.md`, `$S/verification/diff-sdk-300-325.md`, `$S/verification/diff-bin-278-325.md` (plus a `.json` file for each) |
| Changelog slices and cross-checks | `$S/verification/changelog-sdk-300-325.json`, `$S/verification/changelog-cli-300-325.json` |
| Packed SDK tarballs, untrusted and never executed | `$S/reference/sdk-pkgs/` |

---

## 1. Key facts discovered

* **Version lockstep.** `@anthropic-ai/claude-agent-sdk@0.3.N` ships `"claudeCodeVersion": "2.1.N"` in its package.json, and the patch numbers match exactly. One SDK release therefore describes exactly one CLI release. dist-tags: SDK `latest=next=0.3.325`; CLI (`@anthropic-ai/claude-code`) `latest=next=2.1.325`, `stable=2.1.311`. The CLI package has 563 versions.
* **The SDK ships one rolled-up declaration file.** `sdk.d.ts` is about 600 KB and 10.8k lines, built with api-extractor. It also ships `sdk-tools.d.ts` (180 KB of tool input/output types), `bridge.d.ts`, `browser-sdk.d.ts` and `core.d.ts`. Exports map: `.`, `./core`, `./bridge`, `./browser`, `./extract`, `./sdk-tools`.
* `sdk.d.ts` declares, by name, the following types:
  * `SDKMessage`: 40 members in 0.3.325, which expand to 44 `type:subtype` keys.
  * `StdoutMessage`: the full CLI-to-host stream-json line union, which adds control_request, control_response, control_cancel_request, keep_alive and active_goal.
  * `SDKControlRequestInner`: 41 control subtypes, with a matching `SDKControl*Response` type for each.
  * `Options`: 69 fields.
  * `Query`: 30 public methods.
  * `Settings`: 182 keys, identical to the settings JSON schema's top-level properties.
  * `HookEvent` / `HOOK_EVENTS`: 33 events.
  * `SDKStartupFailureReason`, `PermissionMode`, `BUILTIN_TOOL_NAMES`, and others.
* **The published d.ts is filtered.** `@internal` members are stripped and leave runs of blank lines. The count grows each release: 33 blank-line holes in 0.3.278, 55 in 0.3.300, 78 in 0.3.325. The runtime `sdk.mjs` still sends **28 control subtypes that are not in the d.ts**, for example `side_question`, `generate_session_title`, `get_status`, `export_conversation`, `claude_authenticate`, `mcp_authenticate`, `remote_control`, `get_plan`, `set_cwd` and `ultrareview_launch`. They sit behind non-public Query methods such as `askSideQuestion` and `generateSessionTitle`, and the official extension calls most of them (section 5).
* **`manifest.json` in the SDK package** is a CLI build manifest. It holds the version, the commit, a per-platform binary checksum, and `sdkCompat: {harnessSchema: 1, testedWrapperVersions: [0.3.291 … 0.3.324]}`. That list is a free compatibility signal: it says which SDK wrapper versions this CLI was tested against. A future change to `harnessSchema` should raise an alarm.
* **`@anthropic-ai/claude-code` no longer contains a readable `cli.js`.** It ships `bin/claude.exe` (a native Bun-compiled binary), `install.cjs`, `cli-wrapper.cjs` and `sdk-tools.d.ts`. Its `sdk-tools.d.ts` is byte-identical to the SDK's file for the same version (verified for 2.1.278). The binaries are pulled through `@anthropic-ai/claude-code-<platform>` optional dependencies.
* **The Bun-compiled binary embeds the full JS bundle as plaintext.** That holds on every platform, including the linux binary that will not run on this Mac. Commander option definitions, `.hideHelp()`, slash-command objects, and all `subtype:"…"` literals can be string-mined with mmap and regex in about 5 seconds for 365 MB. The mined CLI flags match `claude --help` exactly: 63 of 63 top-level flags, with only the built-in `--help` and `--version` missing.
* **CHANGELOGs.** `anthropics/claude-code/CHANGELOG.md` is about 1.25 MB and has one `## 2.1.N` section per release. The 2.1.300→2.1.325 range covers 21 releases and 1554 bullets. `anthropics/claude-agent-sdk-typescript/CHANGELOG.md` is much more protocol-focused: 0.3.300→0.3.325 covers 25 releases and 114 bullets, and 0.3.325 alone names `archiveSession()`, `refuseIfBusy`, `managed_lock` on `plugin_errors`, and others.

## 2. Sources, ranked

| Rank | Source | How to get it statically | Gives | Stability label |
|---|---|---|---|---|
| 1 | **SDK `sdk.d.ts`** | `npm pack @anthropic-ai/claude-agent-sdk@X --ignore-scripts`, untar, parse with the TypeScript compiler API (parse only, no type-check) | All wire message `type/subtype` keys with field shapes, control request subtypes and response shapes, Options, Query methods, exported functions, string enums (HookEvent, PermissionMode, startup failure reasons…), the Settings interface | **Stable** |
| 2 | **SDK `sdk-tools.d.ts`** (identical copy in the claude-code package) | same tarball | Built-in tool input/output schemas, which matter for rendering tool cards | Stable |
| 3 | **SDK CHANGELOG.md** (GitHub) | `gh api repos/anthropics/claude-agent-sdk-typescript/contents/CHANGELOG.md -H 'Accept: application/vnd.github.raw'` | Human explanation per version; cross-check for the diff | Stable (prose) |
| 4 | **Settings JSON schema** | `claude-code-settings.schema.json` in the VSIX (`$id` = schemastore `claude-code-settings.json`) | Nested setting paths with enums, descriptions and deprecations (280 dotted paths vs 182 top-level keys) | Stable |
| 5 | **`claude --help` recursion** (needs a runnable binary for your OS) | `claude --help`, `claude <cmd> --help`, two levels deep, under 3 s | Public CLI flags with choices and defaults; subcommand tree (54 commands) | Stable |
| 6 | **String-mining the binary bundle** (any platform) | mmap + regex over `claude` or `claude.exe` | Public **and hidden** flags (65 `.hideHelp()` flags in 2.1.325), slash commands (102), every subtype literal (205), control subtypes the CLI sends, internal env vars | Observed / Internal |
| 7 | **String-mining `extension.js` (VSIX) and `sdk.mjs`** | same miner | Which hidden flags and non-public control subtypes the official extension actually uses | **Observed** |
| 8 | **CLI CHANGELOG.md** | `gh api repos/anthropics/claude-code/contents/CHANGELOG.md -H 'Accept: application/vnd.github.raw'` | Broad user-facing notes; too noisy to drive the diff (736 of 1554 bullets look protocol-ish) | Explanation only |

## 3. Manifest format (JSON)

There is one file per source per version. A release manifest is the union of the per-source files plus a `classification` join. Sections are keyed maps, so a diff is a pure key/field comparison:

```jsonc
{
  "manifestVersion": 1,
  "source": { "kind": "npm|cli-help|bundle|settings-schema", "package": "@anthropic-ai/claude-agent-sdk",
              "version": "0.3.325", "claudeCodeVersion": "2.1.325", "strippedMemberHoles": 78 },
  // CLI->host stream-json lines, keyed "type" or "type:subtype"
  "stdoutWire":   { "system:session_writes_lost": { "typeName": "SDKSessionWritesLostMessage", "exported": true,
                      "fields": { "error_code": {"optional": false, "type": "'ENOSPC' | 'EDQUOT'"},
                                  "lost": {"optional": false, "type": "('transcript' | 'global_config')[]"}, ... },
                      "shapeHash": "a2882e610be8" } },
  "sdkMessages":  { ... same shape, SDKMessage union only ... },
  "controlRequests":  { "set_has_subscribers": { "typeName": "SDKControlSetHasSubscribersRequest", "exported": false,
                          "fields": { "has_subscribers": {"optional": false, "type": "boolean | null"}, ... } } },
  "controlResponses": { "SDKControlInitializeResponse": { "fields": {...}, "shapeHash": "..." } },
  "options":      { "verbatimPrompts": {"optional": true, "type": "boolean"} },
  "queryMethods": { "reloadPlugins": {"optional": false, "type": "(options?: {...}): Promise<...>"} },
  "settings":     { "autoMode": {"optional": true, "type": "{ allow?: string[]; ... }"} },
  "exports":      { "archiveSession": {"kind": "Function", "shapeHash": "...", "signature": "function archiveSession(...)"} },
  "stringEnums":  { "SDKStartupFailureReason": ["bypass_root", "config_file_invalid", ...], "HookEvent": [...] },
  "tools":        { "GrepOutput": {"shapeHash": "...", "fields": {...}} },
  // from cli-help / bundle mining
  "cliFlags":       { "--output-format": {"type": "<format>", "choices": ["json","stream-json","text"], "descHash": "..."} },
  "cliHiddenFlags": { "--await-initialize": {"type": "boolean", "descHash": "..."} },
  "cliCommands":    { "plugin marketplace add": {"type": "add [options] <source>", "aliases": []} },
  "slashCommands":  { "/compact": {"type": "local", "aliases": [], "descHash": "..."} },
  "observedWire":   { "subtypeLiterals": [...], "controlRequestsSent": [...], "typeLiterals": [...] },
  "binaryStrings":  { "envVars": [...] }
}
```

Design choices:

* `fields` values are **normalized type text**. The TS printer runs with comments removed and whitespace collapsed. `shapeHash` is sha256 over the sorted fields, so a doc-comment-only edit does not register as a change. Descriptions are kept only as `descHash`, so wording changes show up as `neutral` and can be filtered out.
* Union members are discriminated by their literal `type` and `subtype` values. Aliases are followed, including the namespace-qualified `coreTypes.SDKMessage`. Intersections (`BaseHookInput & {...}`) are flattened.
* `exported:false` means the type is reachable on the wire but not importable. It is still Stable, because it sits in the public d.ts union.

## 4. Extraction commands

```bash
S=/Users/user/Development/ai/chat-interface-specs
# 0. untrusted packages: fresh dir, never run their scripts
mkdir -p $S/sdk-pkgs && cd $S/sdk-pkgs
npm view @anthropic-ai/claude-agent-sdk versions --json        # version list
npm view @anthropic-ai/claude-agent-sdk time --json            # release timestamps (0.3.325 = 2026-11-13T20:05Z)
for v in 0.3.278 0.3.300 0.3.325; do npm pack @anthropic-ai/claude-agent-sdk@$v --ignore-scripts; mkdir v$v; tar -xzf anthropic-ai-claude-agent-sdk-$v.tgz -C v$v; done
# TypeScript (trusted) lives in the tools dir, not next to the data
mkdir -p $S/tools/ts && (cd $S/tools/ts && npm init -y && npm install typescript@5.6.3 --ignore-scripts)

# 1. SDK d.ts -> manifest
node $S/tools/extract-sdk.mjs $S/reference/sdk-pkgs/v0.3.325/package > $S/verification/manifests/sdk-0.3.325.json
# 2. CLI help (runnable binary only)
python3 -I $S/tools/extract-cli-help.py /opt/homebrew/bin/claude > $S/verification/manifests/cli-2.1.278.json
# 3. Binary / bundle mining (works on a linux binary from macOS)
python3 -I $S/tools/extract-bundle.py $S/reference/vsix-2.1.325/extension/resources/native-binary/claude bin-linux --version 2.1.325 > $S/verification/manifests/bin-2.1.325.json
python3 -I $S/tools/extract-bundle.py $S/reference/vsix-2.1.325/extension/extension.js vsix-ext --version 2.1.325 > $S/verification/manifests/ext-2.1.325.json
# 4. Settings schema (nested paths)
python3 -I $S/tools/extract-settings-schema.py $S/reference/vsix-2.1.325/extension/claude-code-settings.schema.json 2.1.325 > $S/verification/manifests/settings-schema-2.1.325.json
# 5. Diff (markdown or --json)
python3 -I $S/tools/diff-manifest.py $S/verification/manifests/sdk-0.3.300.json $S/verification/manifests/sdk-0.3.325.json [--json]
# 6. Changelog slice + "silent change" cross-check
gh api repos/anthropics/claude-agent-sdk-typescript/contents/CHANGELOG.md -H 'Accept: application/vnd.github.raw' > SDK-CHANGELOG.md
python3 -I $S/tools/extract-changelog.py SDK-CHANGELOG.md 0.3.300 0.3.325 --diff diff-sdk-300-325.json
```

Runtimes: d.ts extraction takes about 1 s per version, `--help` recursion about 3 s, and mining a 365 MB binary about 5 s.

To get the binary for a new CLI version without installing it: `npm pack @anthropic-ai/claude-code-linux-x64@2.1.N --ignore-scripts`, or the platform you run. Mining works on any platform's binary. Verify it against `manifest.json` → `platforms.<plat>.checksum` from the matching SDK version.

## 5. Diff and classification scheme

`diff-manifest.py` emits records of the form `{section, key, change, impact, stability, detail}`.

* **change** has three values:
  * `added` / `removed`: the key appears or disappears.
  * `changed-shape`: the `shapeHash` differs. `detail` then lists per-field `+field`, `-field` and `~field`, with a divergence window showing where the normalized type text first differs.
* **impact**, judged from the host's point of view:
  * `additive`: new keys, new optional fields, or a new enum value in a CLI-produced message.
  * `breaking`: a key or field removed, an enum value removed, an optional field that became required, or a new required field in a **host-produced** section (`controlRequests`, `options`, `settings`).
  * `neutral`: type text changed but it is unclear which way, or only a description hash changed.
  * Internal-stability changes are always `neutral`.
* **stability**:
  * **Stable**: present in `sdk.d.ts` / `sdk-tools.d.ts`, the settings schema, or `--help` output.
  * **Observed**: absent from those, but referenced by the official extension's `extension.js` or by the SDK runtime `sdk.mjs`. This covers non-public Query methods and the hidden flags the extension passes.
  * **Internal**: only found in the CLI binary. That includes hidden flags no client uses, subtype literals nobody else references, and env vars.
  * Assignment is a set join, for example `controlRequestsSent(ext) − keys(controlRequests(d.ts))` gives the Observed set.

### Stability join results for 2.1.325

* **Observed control subtypes** are sent by `sdk.mjs` and `extension.js` but are not in the d.ts. There are 28 of them, and the extension's call-site counts come from the method→subtype map:
  * `askSideQuestion→side_question` ×2
  * `generateSessionTitle→generate_session_title` ×2
  * `exportConversation→export_conversation` ×3
  * `getStatus→get_status` ×2
  * `getChromeDialog→get_chrome_dialog` ×6
  * `getMemoryDialog→get_memory_dialog` ×5
  * `getSkillsDialog` ×4, `getSandboxDialog` ×4
  * `claudeAuthenticate` / `claudeOAuthCallback` / `claudeOAuthWaitForCompletion` ×1 each
  * `mcpAuthenticate` / `mcpClearAuth` / `mcpSubmitOAuthCallbackUrl` ×1 each
  * `messageRated` ×2, `submitFeedback` ×2, `submitSurveyFeedback` ×1
  * `enableRemoteControl→remote_control` ×2
  * `getPlan` ×1
  * Present in the SDK but with 0 extension call sites: `claimSession`, `enableChannel`, `setCwd`, `setPromptSuggestionsPaused`, `setChromeBrowserHints`, `launchUltrareview`.
* **Internal control subtypes** appear in the binary only: `get_workspace_diff`, `host_auth_token_refresh`, `register_device_hooks`, `upload_device_hook_template` and `remote_control_work_secret`.
* **Observed hidden CLI flags** are referenced by extension.js/SDK: `--await-initialize`, `--managed-settings-in-initialize`, `--session-mirror`, `--resume-session-at`, `--resume-drops-turn`, `--permission-prompt-tool`, `--max-turns`, `--max-thinking-tokens`, `--thinking`, `--thinking-display`, `--sdk-url`, `--channels`, `--plugin-dir-no-mcp`, `--project-config-root`, `--managed-settings`, `--rc`, `--from-link`, `--gateway-json` and `--resolve-only`. The remaining 46 hidden flags are Internal.
* **Wire envelope types missing from d.ts**: `control_closed` (new since 2.1.278), `update_environment_variables` and `transcript_mirror`.

## 6. Real diffs found

### SDK 0.3.278 → 0.3.300 (`$S/verification/diff-sdk-278-300.md`)
```
## sdkMessages
- changed-shape `system:init` [additive]: +plugin_errors; +view_mode
- changed-shape `system:task_started` [additive]: +parent_task_id; +run_id
- changed-shape `user` [additive]: +agent_id; +client_composed; +inline_pastes
- changed-shape `conversation_reset` [additive]: +timestamp; +trigger; +user_message_uuid
- changed-shape `result:success` [additive]: +first_text_post_queue_wait_ms; +first_text_post_queued_behind
## controlRequests
- added `get_task_output` (SDKControlGetTaskOutputRequest: subtype, task_id)
- added `mcp_read_resource` (SDKControlMcpReadResourceRequest: serverName, subtype, uri)
- changed-shape `elicitation`: +property_order
- changed-shape `initialize`: +sdkMcpServerManifestsOrigin
## controlResponses
- changed-shape `SDKControlInitializeResponse`: +claude_code_version; +sdk_mcp_manifests_parked
## options
- added `verbatimPrompts`: `boolean`
## queryMethods
- added `readMcpResource`: `(serverName: string, uri: string): Promise<SDKControlMcpReadResourceResponse>;`
## settings
- added allowedProviders, deniedModels, availableModelsMatch, idleCompaction, maxProseWidth, allowClaudeInChromeWithManagedMcp
- changed-shape `attribution`: type: `…` -> `boolean | …`            (now also accepts a boolean)
## tools
- added OfferChromeSetupInput/Output; AgentInput +effort; FileReadInput +allow_large; WebFetchInput +offset
## exports
- added prewarm, SpareProcess, ClaimOptions
## stringEnums
- SDKStartupFailureReason +['org_config_refused','org_config_required_unavailable','provider_not_allowed']
```

### SDK 0.3.300 → 0.3.325 (`$S/verification/diff-sdk-300-325.md`)
```
## sdkMessages
- added `system:session_writes_lost` (SDKSessionWritesLostMessage: error_code, lost, session_id, transcript_session_ids, ...)
- changed-shape `result:error_*` [additive]: +credential_expires_at; +invalid_config_file
- changed-shape `system:init`: ~plugin_errors   (entries gained `managed_lock?: {plugin_enabled; name; marketplace?}`)
## controlRequests
- added `set_has_subscribers` (has_subscribers: boolean | null)    <- not mentioned in either changelog
- changed-shape `mcp_call`: +check_permissions; +result_not_for_model
- changed-shape `mcp_status`: +liveReply
- changed-shape `reload_plugins`: +add_plugins; +refuse_if_busy
## controlResponses
- SDKControlInitializeResponse +hook_on_failure_supported; SDKControlReadFileResponse +lossy; GetContextUsage +mcpInstructions
## queryMethods
- changed-shape `reloadPlugins`: `{ holdOnCacheImpact?: boolean; …` -> `{ holdOnCacheImpact?: boolean; addPlugins?: SdkPluginConfig[]; refuseIfBusy?: boolean; …`
## settings
- added autoMode, attributionProject, attributionProjectPolicy, disableReplyCheck, skipAutoPermissionPrompt, sshHostCommands, useAutoModeDuringPlan, disableWindowsBashCommandSplit
- changed-shape `permissions`: +blockWritesOutsideWorkingDirectories, blockReadsExemptPaths, workingDirectoriesAsGiven, ...
- changed-shape `sandbox`: +allowNestedUserNamespaces
## tools
- GrepOutput/GlobOutput +timedOut; BashOutput +backgroundWakesOwner; CronDeleteOutput +endedLoopWakeup; ArtifactInput +page_token
## exports
- added archiveSession, unarchiveSession, matchDeniedModels, BUILTIN_TOOL_NAMES, LEGACY_TOOL_NAME_ALIASES, SDKSessionWritesLostMessage
## stringEnums
- SDKStartupFailureReason +['config_file_invalid','initialize_not_first','managed_settings_not_delivered','sandbox_required_unavailable']
```
The SDK CHANGELOG for 0.3.325 confirms several of these independently: `archiveSession()`, `refuseIfBusy`, `managed_lock` on `plugin_errors`, `invalid_config_file`, and the `ProposeSkillsInput` change. The changelog cross-check (`extract-changelog.py --diff`) marked 39 changes as explained by the SDK changelog and 25 by the CLI changelog. 38 changes are **silent in both changelogs**, for example:

* `controlRequests:set_has_subscribers` (added)
* `mcp_status.liveReply`
* `SDKControlInitializeResponse.hook_on_failure_supported`
* `SDKControlGetUsageResponse.behaviors`
* `BashOutput.backgroundWakesOwner`
* the settings keys `attributionProject`, `disableReplyCheck`, `sshHostCommands` and `disableWindowsBashCommandSplit`

Those are exactly the items a human should look at. The matching is a fuzzy substring test, so some items marked silent are really just worded differently in the changelog.

### CLI binary 2.1.278 (darwin) → 2.1.325 (linux), mined (`$S/verification/diff-bin-278-325.md`)
```
## cliHiddenFlags (Internal unless the extension uses them)
- added --await-claim, --managed-settings-in-initialize, --host-auth-is-org-sign-in, --client-data-url,
        --session-began-as-copy, --allow-other-models, --include-unsupported-models, --web, --from-link, ...
## cliFlags
- added --desktop, --trust-workspace, --trusted-folder, --result-format, --session-tunnel, --profile, ...
- removed --registry
## slashCommands
- added /project, /restart, /use-case;  removed /plugin-types, /update (only /upgrade remains as an object)
## observedWire
- controlRequestsSent +[claim_session, export_conversation, get_chrome_browsers, get_chrome_dialog, get_status,
                        get_task_output, mcp_read_resource, set_prompt_suggestions_paused, submit_survey_feedback, ...]
                      -[oauth_token_refresh]
- subtypeLiterals +[session_writes_lost, set_has_subscribers, mcp_tasks_changed, per_turn_effort_changed,
                    permission_check_status, instruction_size_warning, ui_* (a whole new ui_render/ui_press/ui_prompt_* family), ...]
- typeLiterals +[control_closed]
```

## 7. Pitfalls

1. **The public d.ts is not the whole protocol.** `@internal` members and subtypes are stripped, leaving blank-line holes; count them as a drift signal. The SDK runtime and the official extension use 28 or more control subtypes that never appear in the d.ts. Treat those as Observed and pin them with runtime probes (the dynamic half) before relying on them.
2. **The `exported:false` types are still wire.** `SDKControl*Request` types are mostly non-exported `declare type`s. Don't filter them out by `export`.
3. **Namespaces and aliases.** `StdoutMessage` references `coreTypes.SDKMessage`, a namespace re-export. Resolve the rightmost identifier, or `stdoutWire` silently comes out empty.
4. **The direction of a change matters for "breaking".** A new required field in a CLI→host message is additive. In a host→CLI control request or Option, it is breaking. The first version of the diff wrongly flagged `Query.readMcpResource` as breaking.
5. **Normalize or drown.** Raw d.ts diffs are dominated by doc-comment churn, since descriptions are long and constantly reworded. Strip comments with the TS printer, and keep descriptions only as hashes.
6. **Long nested types** (`settings.permissions`, `marketplaces`) produce `~field` noise. Show a divergence window, or recurse into nested object literals. Nested settings paths are better taken from the JSON schema extractor (280 dotted paths).
7. **Binary mining is heuristic.**
   * Minified variable names change every build (`new vs("--x"…)`), so match on `new <1-4 char ident>("--`.
   * Descriptions use `"`, `'` or backticks.
   * Slash commands use getter forms (`get description(){return"…"}`), and the first miner version falsely reported `/resume` and `/teleport` as removed because of it.
   * Some subtypes are only dispatched via `case"x":` (`set_color` and `get_binary_version` were missed), so use the d.ts as the authoritative list and mining only to find extras.
   * Flags are mined flat, without their subcommand.
   * The env-var regex catches concatenation junk.
8. **Compare like with like.** The darwin binary (217 MB) and the linux binary (365 MB) are both mineable, but compare the same platform across versions where possible. In the 278 vs 325 comparison here the platforms differ, and that may inflate the literal counts.
9. **`--help` needs a runnable binary.** It only covers public flags, and its wrapping is fixed at 80 columns (COLUMNS is ignored), so the parser must join continuation lines. Command descriptions can embed `Examples:` blocks (`claude mcp --help`), and those must not be parsed as sections. Set `DISABLE_AUTOUPDATER=1` and `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`.
10. **Changelog matching is fuzzy.** CLI changelog volume is huge, about 75 bullets per release. Use it to explain diff entries, not to drive them. SDK changelog entries use camelCase (`refuseIfBusy`) while the wire uses snake_case (`refuse_if_busy`), so normalize case and underscores before matching.
11. **Untrusted packages.** Always `npm pack --ignore-scripts` into a fresh directory, never `npm install` them, and parse with `python3 -I` or a node script that lives outside the package directory. TypeScript is installed separately in `$S/tools/ts`.
12. **Skipped versions.** npm has gaps (for example 0.3.297, 0.3.313 and 0.3.320 are missing from the SDK list), and `stable` lags `latest` (2.1.311 vs 2.1.325). Diff against the last version you actually shipped, not N-1.

## 8. Suggested pipeline (static half)

1. Trigger on a new `npm view @anthropic-ai/claude-agent-sdk dist-tags.latest` or a new CLI tag.
2. Pack the SDK@N and the `claude-code-<your-platform>`@N binary.
3. Run `extract-sdk` (Stable), `extract-cli-help` on the new binary (Stable flags/commands), and `extract-bundle` on the binary (hidden flags, slash commands, literals) and on the latest VSIX `extension.js` if available (Observed).
4. Run `extract-settings-schema` on the VSIX schema, or fetch it from schemastore.
5. Run `diff-manifest` against the stored manifest of the CLI version currently shipped in the plugin.
6. Run `extract-changelog --diff` on both changelogs to tag each change explained or silent.
7. Write a report grouped by section × stability × impact. Fail CI on `breaking` items in Stable sections the plugin consumes, for example `stdoutWire` keys it renders or `controlRequests` it sends. Additive Stable items become "adoption candidates". Observed and Internal items are FYI only.
8. Check `manifest.json.sdkCompat.testedWrapperVersions` and `harnessSchema`. A `harnessSchema` bump means the plugin needs the dynamic half (real stream-json probes) before adopting the new binary.
