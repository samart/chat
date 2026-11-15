# CLI Change Verification: Triage and Adoption Workflow

Status: design draft (2026-11-15)
Scope: what happens *after* the two signal workstreams produce output.

- (a) static **protocol surface manifest** diff between CLI / Agent SDK releases
- (b) **runtime probes**: zero-cost `initialize` handshake capture, plus rare paid scenario transcripts normalized into shape signatures

This document covers triage (classification), the official VS Code extension as a second signal, release gating and the adoption report, an LLM-assisted proposal step, and runtime defenses inside the plugin.

---

## 0. Design principles

1. **Consumer-driven severity.** A change is only *Breaking* if **our plugin consumes the thing that changed**. The same removal can be Breaking for us and Ignorable for someone else. So triage needs a third input besides the two diffs: our own **consumption manifest** (Section 1.2).
2. **Deterministic first, LLM second, human last.** Rules classify everything they can. The LLM only drafts *adoption proposals* and explanations. It never downgrades a rule-assigned Breaking. A human approves anything that changes the supported range or the UI.
3. **Detect features, don't compare versions.** At runtime the plugin asks the CLI what it can do (`initialize` response, probe-and-fallback). Version ranges are a coarse gate for support and messaging, not the feature switch.
4. **Everything unknown is visible.** Unknown items in the diff fail closed (block until classified). Unknown messages at runtime fail open (render a generic fallback, emit telemetry).
5. **Stability changes how loud a change is, not whether it gets triaged.** Stable, Observed and Internal items all get classified. Stability sets the default severity and who is notified.

---

## 1. Inputs and the normalized change record

### 1.1 Change record (output of workstreams a + b, input to triage)

Every diff line becomes one record:

```jsonc
{
  "id": "chg-2.1.330-ctrl-get_context_usage",
  "from": "2.1.325", "to": "2.1.330",
  "surface": "control_subtype",          // see surface list below
  "direction": "host->cli",              // host->cli | cli->host | both | n/a
  "key": "get_context_usage",
  "change": "added",                     // added | removed | renamed | type_changed |
                                         // field_added | field_removed | field_required |
                                         // enum_value_added | enum_value_removed | default_changed
  "path": null,                          // for field-level changes, e.g. "system/init.apiKeySource"
  "stability": "Observed",               // Stable | Observed | Internal
  "evidence": [
    {"source": "sdk-dts", "ref": "sdk.d.ts#SDKControlGetContextUsageRequest"},
    {"source": "changelog", "ref": "CHANGELOG.md#2.1.330", "quote": "Added /context usage to IDE"},
    {"source": "probe", "ref": "probe/initialize@2.1.330.json"},
    {"source": "vsix", "ref": "vsix/2.1.330: getContextUsage -> get_context_usage (send)"}
  ]
}
```

Surfaces: `message_type`, `system_subtype`, `content_block_type`, `stream_event_type`, `control_subtype`, `control_response_field`, `sdk_option`, `sdk_method`, `cli_flag`, `settings_key`, `slash_command`, `tool_name`, `hook_event`, `permission_mode`, `model_alias`, `init_field` (fields in `system/init` or the `initialize` response), `shape_signature` (from paid scenario transcripts).

### 1.2 Consumption manifest (ours)

A generated file, `protocol-consumption.json`, lists every protocol item the plugin **sends**, **reads**, or **renders**, plus how hard it depends on each one:

```jsonc
{
  "control_subtype:interrupt":      {"use": "send",   "dependency": "required", "fallback": null},
  "control_subtype:set_model":      {"use": "send",   "dependency": "feature",  "fallback": "restart-with --model"},
  "control_subtype:mcp_status":     {"use": "send",   "dependency": "feature",  "fallback": "hide MCP panel"},
  "control_subtype:can_use_tool":   {"use": "handle", "dependency": "required", "fallback": null},
  "system_subtype:init":            {"use": "read",   "dependency": "required", "fields": ["session_id","model","tools","slash_commands","permissionMode"]},
  "content_block_type:thinking":    {"use": "render", "dependency": "feature"},
  "cli_flag:--output-format":       {"use": "pass",   "dependency": "required"},
  "cli_flag:--permission-prompt-tool": {"use": "pass", "dependency": "required"}
}
```

How it is built: (1) a typed registry in the plugin's protocol layer. Every send, handler and renderer is registered by key, so the manifest falls out of the code. (2) A CI lint that fails if a string-literal `subtype:"..."` or `--flag` appears outside that registry. Without this manifest you cannot tell "removed" apart from "removed and we break".

---

## 2. Classification rubric

### 2.1 Classes

| Class | Definition | Gate effect | Owner action |
|---|---|---|---|
| **B: Breaking** | We consume the item and the change makes current plugin code fail: removed or renamed, type changed, a field we read removed, a new *required* field in a message we must answer, a flag we pass removed, an enum value we send removed, or a semantics or default change that changes a required flow. | **Blocks** raising `maxTested` and the bundled/recommended version. | Fix plus contract test before upgrade. |
| **R: Renders badly** | Compatible on the wire, but the UI would degrade: a new `system` subtype, a new content-block type, a new tool name, a new enum value in something we display (permission mode, stop reason), or a new `stream_event` delta type. | Does not block if the tolerant reader covers it (verified by fixture). **Blocks** if the fallback is missing, would throw, or would show raw JSON to users. | Add a specific renderer, or confirm the generic fallback is acceptable. |
| **A: Adoption opportunity** | A new capability worth exposing in the chat UI, such as a picker, card, command, panel or setting. | Never blocks. | Goes to the adoption backlog with a proposal and a decision. |
| **I: Ignorable/Internal** | Not consumed and not user-facing for us: internal subtypes, TUI-only flags, settings keys we don't surface, telemetry fields. | Never blocks. | Auto-closed and logged. Reopened if a second signal fires (Section 3). |
| **U: Unclassified** | No rule matched with confidence. | **Blocks** until a human classifies it. | Human triage. The decision is recorded as a new rule. |

One item can carry **two classes**. The usual pair is **R+A**: a new system subtype both renders badly today and is worth a first-class renderer. The gate uses the most severe class. The backlog uses all of them.

### 2.2 Decision procedure (rules engine, evaluated in order)

```
1. if consumed(item) and change in {removed, renamed, type_changed, field_removed(on a read field),
                                   enum_value_removed(on a sent value), field_required(on a handled msg)}
       -> B   (severity: Stable=P0, Observed=P0, Internal=P1 "we were depending on Internal; also file debt")
2. if consumed(item) and change == default_changed and dependency == required
       -> B (or U if the rule cannot tell; behaviour changes need a scenario probe)
3. if change == added and surface in {system_subtype, content_block_type, tool_name,
                                      stream_event_type, permission_mode, hook_event(rendered)}
       -> R, plus A if (stability != Internal) or vsix_adopted(item)
4. if change == added and surface in {control_subtype, sdk_option, sdk_method, cli_flag, slash_command, model_alias}
       -> A if (stability == Stable) or vsix_adopted(item) or changelog_mentions_user_facing(item)
       -> I otherwise (kept on a watchlist and re-evaluated when the VSIX adopts it)
5. if change in {field_added} and consumed(parent)  -> I  (tolerant reader); A if field is display-worthy (heuristic: name matches /cost|usage|tokens|title|summary|progress/)
6. if not consumed(item) and change in {removed, renamed}  -> I (but if stability == Stable, note in report: "public API removal")
7. if stability == Internal and not vsix_adopted(item) -> I
8. else -> U
```

`vsix_adopted(item)` comes from Section 3. It is the main way an Internal or Observed item gets promoted into A.

### 2.3 Worked examples (realistic protocol items)

| Item (change) | Stability | Consumed by us? | Class | Reasoning / action |
|---|---|---|---|---|
| New `system` subtype `compact_boundary` (with `compact_metadata.{trigger, pre_tokens}`) | Stable | No (new) | **R + A** | Today it shows as an "unknown system event" or is silently dropped, and the transcript looks like it lost history. Adopt: a divider such as "Conversation compacted (auto, 154k tokens to summary)". The rewind/fork logic must treat it as a history boundary. The official extension does exactly that: `if(e.type==="system"&&e.subtype==="compact_boundary")this.#l.clear()`. |
| New control subtype `get_context_usage` (host to CLI) | Observed | No | **A** | No wire risk: we never send it. Adopt as a context meter in the composer footer that opens a breakdown popover. High priority because the VSIX exposes `getContextUsage -> get_context_usage`. Feature-detect with probe-and-fallback (Section 5.2). |
| New CLI flag `--effort <low\|medium\|high>` | Stable (in `--help`, CHANGELOG) | No | **A** | Adopt as an effort picker next to the model picker. Check whether there is also a runtime control subtype (for example `set_max_thinking_tokens`, or a future `set_effort`). If it exists only as a flag, switching effort costs a process restart with `--resume`, and the proposal must say so. |
| Removed field `system/init.apiKeySource` | Observed | Yes (read for the account badge) | **B** (P0 if read without default, else R) | If the code is `init.apiKeySource.toUpperCase()`, it throws: Breaking. If the code is `init.apiKeySource ?? "unknown"`, the badge shows "unknown": Renders badly. The lint should push all `init` reads through the tolerant accessor so this becomes R. |
| Renamed control subtype `mcp_status` to `mcp_server_status` | Observed | Yes (MCP panel) | **B** | The CLI returns an error response for the unknown subtype, and the MCP panel is empty. The fix is dual-send: try new, fall back to old, cached per session. Contract test against both versions. |
| New tool name `Monitor` appears in `system/init.tools` and `tool_use.name` | Stable | Generic tool card only | **R** (+A if the official UI has a bespoke card) | The generic card shows raw `input` JSON. Acceptable as R if it is collapsed and readable. Adopt a custom card if the input has a natural UI (a command plus a "watching..." status). |
| `can_use_tool` request gains optional `blocked_path` | Observed | Yes (handler) | **I** (+A) | Optional field, so the tolerant reader ignores it. Adopt: show the path in the permission dialog. |
| `can_use_tool` response now **requires** `toolUseID` echo | Observed | Yes | **B** | Our responses get rejected and permissions hang. P0. |
| New `permission_mode` enum value `auto` | Stable | Yes (mode picker displays current mode) | **R + A** | The picker would show an empty label or crash on lookup. Add the label, and offer it in the picker if `initialize` advertises it. |
| New slash command `/ultrareview` | Observed | No (commands come dynamically from `init.slash_commands`) | **I** for code, maybe **A** | The command palette already lists it dynamically. Adopt only if it deserves bespoke UI. Here it does: the VSIX maps `launchUltrareview -> ultrareview_launch`, so it is a launch flow, not just text. |
| New settings key `voice.enabled` | Internal | No | **I** | Not surfaced. Auto-close. |
| New control subtypes `ui_attach`, `ui_render`, `ui_press` | Internal | No | **I** (watchlist) | TUI-mirroring internals. The VSIX references them, but as host plumbing, not a UI feature. Keep on the watchlist only. |
| Flag `--permission-prompt-tool` removed | Stable | Yes (we pass it) | **B** (P0) | The CLI exits with "unknown option", so no session starts. The highest-severity class there is. |
| Default model alias changes (`default` now points elsewhere) | Stable | Yes (we display the model) | **U** then **R** | The wire is unchanged but the UX shifts. Needs a human decision and a probe to confirm the label we show. |
| New `stream_event` delta `citations_delta` | Stable (API) | Partially (we render `text_delta`) | **R** | The partial-message renderer must ignore it without dropping text, so a fixture is required. **A** if we want citation chips. |

### 2.4 Severity and routing

- **P0** (Breaking on a required dependency): page the owner, and add the version to the `blocked` list automatically.
- **P1** (Breaking on a feature dependency with a working fallback, or R with no fallback): blocks `maxTested`, but the bundled version can stay pinned.
- **P2** (R with an acceptable fallback): ticket in the next release.
- **P3** (A): adoption backlog, prioritized by the adoption score (Section 4.4).

---

## 3. The official VS Code extension as a second signal

### 3.1 Why

The official extension shows what Anthropic treats as **user-facing in an IDE chat panel**. That is the same product shape as this plugin. Three signals come out of it:

- **Adoption:** "the official UI started sending `X` in version N". This is a strong A signal, and it can promote an Observed or Internal item into A.
- **Abandonment:** "the official UI stopped sending `X`". This warns that `X` may be removed soon. If we consume `X`, raise a pre-emptive P2 to migrate.
- **Handling:** "the official UI now handles `system/X` or `content_block/Y`". This confirms the item is a real rendering concern (R).

Observation from the local extracted VSIX (`vsix/extension`, `package.json` version **2.1.325**): it ships `resources/native-binary/claude`. The official extension **bundles an exact CLI build** whose version tracks the extension version, so a VSIX version maps 1:1 onto a CLI version. That makes "adopted in VSIX N" directly comparable to "appeared in CLI N". For comparison, the locally installed CLI here is 2.1.278, so the user-facing CLI can lag the bundled one.

### 3.2 Acquisition

- **Open VSX:** `GET https://open-vsx.org/api/anthropic/claude-code/<version>` returns metadata including `files.download`. Platform-specific builds use `.../anthropic/claude-code/<targetPlatform>/<version>`, for example `darwin-arm64` or `linux-x64`. Pick one platform consistently. `extension.js` and `webview/index.js` should be platform-independent, but verify that with a hash compare across two platforms once.
- **VS Code Marketplace:** `GET https://marketplace.visualstudio.com/_apis/public/gallery/publishers/anthropic/vsextensions/claude-code/<version>/vspackage?targetPlatform=<platform>`. The response may be gzip-encoded. To list versions, use the gallery `extensionquery` POST with `IncludeVersions`.
- Cadence: a **daily** poll of the latest version on both registries, plus backfill of any versions you missed (versions can ship several per week).
- Hygiene: the VSIX is untrusted data. Download it into a fresh empty directory and unzip it without executing anything. The extractor scripts live elsewhere and get paths as arguments. Never run the bundled binary for analysis (probes use the npm CLI). Store only the extracted **string inventory** (below), not redistributed code. This is interoperability signal, not code reuse.

### 3.3 Extraction (minified `extension.js` + `webview/index.js`)

Extraction is regex/string-based, deliberately simple, and robust to minifier renames because it anchors on string literals, which survive minification.

| Inventory | Pattern(s) | Meaning |
|---|---|---|
| **Control subtypes sent** | `request\(\{subtype:"([a-z_]+)"` (primary), `subtype:"([a-z_]+)"` (broad) | Host-to-CLI requests the extension issues |
| **Method to subtype map** | `async ([A-Za-z_]+)\([^)]*\)\{[^}]{0,160}?subtype:"([a-z_]+)"` | Gives a human-meaningful name, e.g. `getContextUsage -> get_context_usage`, `mcpServerStatus -> mcp_status`, `launchUltrareview -> ultrareview_launch` |
| **Subtypes handled** | `subtype===?"([a-z_]+)"`, `case"([a-z_]+)"` | CLI-to-host messages the extension branches on (e.g. `compact_boundary`) |
| **Message / content types handled** | `type===?"([a-z_]+)"`, `case"([a-z_]+)"` in `webview/index.js` | Content blocks and message kinds the webview renders (e.g. `server_tool_use`, `redacted_thinking`, `tool_reference`, `search_result`) |
| **Webview <-> host messages** | `type:"([a-z_]+_(request\|response))"`, `case"([a-z_]+_request)"` | Extension-internal RPC (e.g. `list_sessions_request`, `get_session_request`). It shows which UI features exist even when they are not protocol items. |
| **SDK options used** | keys inside the object passed to the query/spawn call, e.g. `(maxThinkingTokens\|permissionMode\|allowedTools\|...):` matched against the SDK option list from the manifest | Which `Options` the official UI sets |
| **CLI flags passed** | `"--([a-z-]+)"` intersected with the `--help` flag list | Which flags it spawns with |
| **Tool names with bespoke rendering** | tool names from the manifest, intersected with literals in `webview/index.js` | Which tools get custom cards |
| **Settings surfaced** | `contributes.configuration.properties` in `package.json`, and the bundled `claude-code-settings.schema.json` | User-visible IDE settings |

Calibration from VSIX 2.1.325: 76 distinct `subtype:"..."` literals, 55 `request({subtype:...})` call sites, and about 40 recoverable method-to-subtype pairs. The broad pattern catches noise (`success`, `error`, `status` are response or other subtypes), so **intersect every inventory with the manifest's known surface** and report the leftovers separately as "unknown literals". Those leftovers are often the earliest hint of an unreleased protocol item.

### 3.4 VSIX diff and joint signal

Diff inventory N-1 to N, producing `vsix_adopted`, `vsix_dropped`, `vsix_newly_handled`. Then join with the CLI diff:

| CLI manifest says | VSIX says | Interpretation | Effect on triage |
|---|---|---|---|
| added in N | sends/handles in N | Shipped as a user-facing IDE feature | A, priority boost (+2 in the adoption score). Promote stability Internal to Observed. |
| added in N | absent | Plumbing, experiment, or headless/SDK-only | Follow the base rule (often I or watchlist) |
| present for a while | **newly** adopted in N | Anthropic decided it is ready for the IDE | Re-open a previously I item as A ("official UI adopted X in 2.1.N") |
| present | **dropped** in N | Likely deprecation | If we consume it: pre-emptive P2 migration ticket. If Stable: watch for removal. |
| removed in N | still sent by VSIX N-1 | Hard removal | Confirms B urgency if we consume it |
| absent from manifest | literal appears | Unreleased or undocumented surface | "Unknown literals" section in the report; no action |

Store the history as `vsix-adoption-ledger.jsonl` (`{item, firstSeenVsix, lastSeenVsix, role: send|handle|render}`). It answers "when did the official UI adopt X?" in one query.

---

## 4. Release gating

### 4.1 Version policy

Declared in the plugin's `package.json` (or a `cli-compat.json` shipped with it):

```jsonc
"claudeCli": {
  "min": "2.1.250",          // floor: oldest version providing every *required* capability
  "recommended": "2.1.325",  // what the plugin bundles or installs by default (exact pin)
  "maxTested": "2.1.330",    // newest version that passed the full gate
  "blocked": ["2.1.301"],    // known-bad (P0 found, or upstream regression)
  "policy": "soft-ceiling"   // above maxTested: run, show non-modal banner, telemetry tagged
}
```

- **Floor (`min`):** computed, not hand-picked. It is the oldest version whose manifest contains every `dependency: required` entry in the consumption manifest. Below the floor the plugin refuses to start a session and offers "Update Claude Code".
- **Recommended:** an exact pin. If the plugin bundles a binary (as the official one does), this is the bundled build. Otherwise it is what the installer fetches. It moves only after the gate passes, and at most weekly unless a security fix lands.
- **Ceiling (`maxTested`):** soft by default. Users who run a newer CLI keep working, because the runtime defenses in Section 5 are designed for this. They see a dismissible banner, and unknown-message telemetry is tagged `aboveCeiling=true`. A **hard** ceiling is only used for a version in `blocked`.
- **Blocked list:** updated out-of-band through remote config (Section 5.5), so a bad CLI release can be blocked without shipping a plugin release.
- Floor raises are deliberate: they happen when adopting a feature that becomes *required*, and stay within N-2 minor versions or about 60 days, whichever is older, to avoid stranding enterprise users on pinned CLIs.

### 4.2 Compatibility matrix

Generated and published with each plugin release. Rows are CLI versions and columns are plugin versions. Each cell is a status plus capability flags:

| CLI \ Plugin | 0.9.x | 0.10.x | 0.11.x (current) |
|---|---|---|---|
| 2.1.250 | OK | OK | OK (floor) |
| 2.1.301 | BLOCKED (P0: init crash) | BLOCKED | BLOCKED |
| 2.1.325 | R: compact_boundary unstyled | OK | OK (recommended) |
| 2.1.330 | R | R: Monitor generic card | OK (maxTested) |
| 2.1.331 (latest) | ? | ? | **PENDING**: nightly found U x1 |

A second view lists **capability by CLI version** (from probes plus the manifest): `set_model`, `get_context_usage`, `--effort`, `compact_boundary`, and so on, each marked present, absent or unknown. That view drives the feature-detection fixtures.

### 4.3 CI cadence

| Job | Trigger | Cost | What it does |
|---|---|---|---|
| **Nightly dist-tag watch** | cron 02:00 UTC | free | `npm view @anthropic-ai/claude-code dist-tags --json` (`latest`, `next`/`beta` if present) and the SDK package. If either moved, fire the pipeline. |
| **Static manifest diff** | new version | free | Workstream (a), from `maxTested` to `new`. |
| **Initialize probe** | new version | free | Workstream (b) zero-cost path: spawn, send `initialize`, capture, kill. Diff `commands`, `models`, `output_styles`, `agents`, and the response shape. |
| **Triage + report** | after the two jobs above | free (+LLM step) | Rules engine, then VSIX join, then report, then PR or issue. |
| **Contract tests** | every plugin PR + new CLI | free | Replay recorded fixtures (per CLI version) through the plugin's protocol layer. Assert no throw, no raw-JSON render, and the expected UI model. |
| **VSIX watch** | daily | free | Section 3. |
| **Paid scenario probes** | only when triage finds **B or U on a cli->host message shape**, any `default_changed`, or a minor version bump; also weekly on `recommended` | $ (budget-capped per month) | Workstream (b) full transcripts, then shape signatures, then diff against the last signatures. |
| **Next-tag soak** | weekly | free | Same as nightly but against the `next` tag, for early warning. |

### 4.4 What blocks an upgrade

These block raising `maxTested` or `recommended`:

1. Any **B** (P0 or P1) without a merged fix and a contract test.
2. Any **U** (an unclassified item is a block, not a pass).
3. Any **R** whose fallback fixture fails: the renderer throws, shows raw JSON outside dev mode, or drops sibling content.
4. A shape-signature regression in a paid scenario on a `required` flow (permission prompt, interrupt, resume, tool result rendering).
5. Upstream signals: the version is in `blocked`, or the CHANGELOG for the version mentions a regression in a flow we depend on (LLM-flagged, human-confirmed).

These **do not** block: A items (never), I items, and R items with passing fallbacks.

**Automation:** if nothing blocks, CI opens a PR that bumps `maxTested` with the adoption report as the PR body, and auto-merges after the green contract tests if there are no R items. Bumping `recommended` always needs a human approval.

### 4.5 The Adoption Report

The report is generated per candidate version as `reports/cli-<from>..<to>.md` plus a sidecar `.json`. Sections:

1. **Header:** from and to versions, date, gate verdict (PASS / BLOCKED / NEEDS-TRIAGE), and counts by class (B / R / A / I / U).
2. **Blocking items (B, U, failing R):** each with evidence, the consumption-manifest entry hit, a fix sketch, and an owner.
3. **Renders-badly items:** current fallback screenshot or snapshot ref, whether the fixture passes, and the proposed renderer.
4. **Adoption opportunities**, sorted by adoption score. Each has a proposal, UI spec section, official-UI signal and effort estimate.
5. **Official UI delta (VSIX N-1 to N):** adopted, dropped, newly handled, and unknown literals.
6. **Runtime telemetry crossover:** unknown message fingerprints seen in the wild for this version (Section 5.4). This confirms or contradicts the static diff.
7. **Ignored/internal (collapsed):** one line each with the rule that ignored it, so a reviewer can contest it.
8. **Compatibility matrix delta**, and the proposed `claudeCli` policy change.
9. **Decisions required:** checkboxes for a human (approve maxTested bump, approve or defer each A item).

Adoption score: `(stability: Stable 3 / Observed 2 / Internal 0) + (vsix_adopted ? 2 : 0) + (changelog user-facing ? 1 : 0) + (in-the-wild telemetry hits ? 1 : 0) + (fits existing spec section ? 1 : 0) - (needs restart or flag-only ? 1 : 0)`. Use it only to sort the backlog, never to gate.

**Example entry (section 4):**

```markdown
### A-03  Context usage meter  (`control_subtype:get_context_usage`)   score 6  [R?: no]  [stability: Observed]

**What changed:** New host->cli control request `get_context_usage` (added 2.1.330). Response:
`{ total_tokens, max_tokens, categories: [{name, tokens}], ... }` (shape from probe/initialize@2.1.330
+ scenario sig `ctx-usage-v1`).

**Official UI signal:** Adopted by VSIX 2.1.325 as `getContextUsage -> get_context_usage` (send); rendered in
webview footer. First seen VSIX 2.1.325, still present 2.1.330.

**CHANGELOG:** "2.1.330: IDE: show context window usage" (verbatim, CHANGELOG.md L1423)

**Proposal (draft by LLM, edited by @owner):**
- UI spec §4.2 "Composer footer": add a compact ring meter (percentage); click opens popover with category
  breakdown (§7.1 "Popover pattern"). Warn color at ≥80%.
- Poll on `result` message and after `compact_boundary`; no timer polling.
- Feature detection: probe once per session; on `error` response ("unsupported subtype") hide meter and cache
  `capabilities.contextUsage=false` for this CLI version.
- Fallback for older CLIs: none (meter hidden). Does **not** raise `min`.

**Effort:** S (1-2d). **Risk:** Observed (not Stable SDK). Keep behind capability flag + remote kill switch.

**Evidence:** chg-2.1.330-ctrl-get_context_usage · vsix-ledger#get_context_usage · probe/2.1.330/init.json

**Decision:** [ ] Adopt now  [ ] Defer to <version>  [ ] Reject (reason: ____)
```

### 4.6 LLM-assisted proposal step

This is *not* a classifier of record. It drafts and annotates, and a human approves.

**Run:** in CI, after the rules engine. Use the pinned `claude -p --output-format json` with a restricted tool set (read-only, no network). The repo's own CLI subscription or key provides access. Inputs are written to files and referenced, not inlined from untrusted sources without delimiting.

**Inputs:**
- change records (A, R, U only; B goes to humans directly)
- CHANGELOG slice from `from` to `to`, treated as data and delimited
- VSIX delta and adoption ledger
- the **UI spec section index** (headings plus a one-paragraph summary of each section of the plugin's UI spec)
- the consumption manifest
- the **decision ledger** (past adopt/defer/reject decisions with reasons), so it does not re-propose rejected items

**Output (strict JSON schema, validated):**
```jsonc
{ "item_id": "chg-...", "proposed_class_note": "agree|suggest-R|suggest-U",  // advisory only
  "title": "...", "user_value": "...", "spec_sections": ["§4.2", "§7.1"],
  "ui_sketch": "...", "feature_detection": "...", "fallback": "...",
  "effort": "S|M|L", "risks": ["..."],
  "evidence_ids": ["..."],          // MUST reference provided evidence; uncited claims are dropped
  "changelog_quote": "..." }        // MUST be a verbatim substring of the provided CHANGELOG slice
```

**Guardrails:**
- Validation drops any proposal that cites an evidence ID that doesn't exist or a non-verbatim CHANGELOG quote.
- The LLM cannot change a gate decision. `suggest-U` on an item the rules called I or A puts it in human triage, which only makes the gate *more* conservative.
- It also does **CHANGELOG cross-reading**. It flags CHANGELOG lines that have no matching manifest change ("behavioral change with no surface diff", e.g. "permission prompts now batch"). Those become U items that need a scenario probe. This catches behavior changes the static diff cannot see.
- Humans approve through PR checkboxes. The decisions are appended to `decision-ledger.jsonl` and become future rules. For example, "reject: `voice_*`" auto-classifies future voice items as I.

---

## 5. Runtime defenses in the plugin

### 5.1 Tolerant reader

- **Parse at the boundary into a discriminated union with an `Unknown` arm.** Each NDJSON line becomes `Known<T>` or `Unknown{raw, type, subtype}`. Nothing throws past the parser. Schema validation (zod or similar) runs in `passthrough` mode: required fields are checked only for items marked `required` in the consumption manifest, and unknown fields are kept.
- **Read through accessors with defaults** (`init.get("apiKeySource", "unknown")`). A lint rule bans direct deep property access on protocol objects.
- **Rendering fallbacks by surface:**
  - unknown `system` subtype: a collapsed, muted "System event: `<subtype>`" row (raw JSON shown in dev mode only). Never dropped silently, because a dropped `compact_boundary`-like event corrupts the user's mental model of the history.
  - unknown content block: an "Unsupported content (`<type>`)" chip inside the message, with siblings rendered normally.
  - unknown tool name: the generic tool card (name, collapsed input, result text). Good enough to be R rather than B.
  - unknown `stream_event` delta: ignored, and the final `assistant` message reconciles the content.
  - unknown enum value in displayed state (permission mode, stop reason): show the raw value title-cased, with no crash on lookup.
- **Unknown inbound `control_request` (CLI to host):** reply immediately with `control_response {subtype:"error", error:"unsupported by host"}`. Never leave it unanswered, because the CLI may block waiting.
- **Outbound requests:** every `control_request` has a timeout (e.g. 10s; interrupt 3s), and a timeout maps to "capability unknown", not a crash.

### 5.2 Feature detection, not version checks

- **From `initialize` / `system/init`:** treat the response as the capability source of truth: `commands` (slash commands), `models`, `output_styles`, `agents`, `account`, available `tools`, `permissionMode`, MCP servers. The UI renders pickers from these lists, so a new model, command or output style needs no code change. That is why many "new slash command" items are I.
- **Probe-and-cache for control subtypes:** on first use, send the request. If the response is `error` with an unsupported-subtype message, set `capabilities[subtype]=false` and cache it per `(cliVersion, sessionId)`. Use lazy probes for optional features (the context meter) and an eager probe at session start only for features that shape the initial UI (e.g. `set_model` availability decides whether the model picker is live or "requires restart").
- **Flags:** parse `claude --help` once per CLI binary hash and cache which flags exist. The plugin never passes an optional flag (e.g. `--effort`) that the binary does not advertise.
- **Capability registry API:** `caps.has("contextUsage")`. The UI hides or disables features from it. Version comparisons are allowed in exactly two places: the floor check at startup, and a small `knownBugs` table (`{range, workaround}`) for behavior that cannot be detected.
- **Dual-path for renames:** when triage finds a rename (B), the fix is "try new, fall back to old" behind the registry. That keeps the floor where it is.

### 5.3 Startup guard

- Resolve the CLI binary and run `--version`, then compare against `claudeCli`:
  - below `min`: block with an update CTA
  - in `blocked`: block with an explanation and a pin suggestion
  - above `maxTested`: a non-modal "newer than tested" banner
  - otherwise: silent
- Record `(cliVersion, binaryHash, pluginVersion)` in the session context for telemetry and bug reports.

### 5.4 Unknown-message telemetry (drift in the wild)

- **Event:** `protocol.unknown` with a privacy-safe fingerprint and **no content**:
  `{pluginVersion, cliVersion, aboveCeiling, direction, surface, type, subtype, blockType|toolName|enumValue, keysetHash, sampleCount}`.
  `keysetHash` is a hash of the sorted top-level key names, which detects shape drift without values. Tool names and subtypes are protocol identifiers, not user data. MCP tool names (`mcp__server__tool`) are reduced to `mcp__*`.
- **Also emit:** `protocol.control_unsupported` (a probe got an unsupported error), `protocol.timeout` (by subtype), and `protocol.schema_violation` on a `required` item. The last one is the earliest Breaking signal from the field.
- **Volume control:** dedupe per session and fingerprint, rate-limit, opt-in under the plugin's existing telemetry setting, and respect `DISABLE_TELEMETRY`-style env vars.
- **Feedback loop:** a daily aggregation pulls the top new fingerprints by `cliVersion`. A fingerprint that does not exist in the manifest for that version becomes a **U** change record with `source: field-telemetry`. Typical causes are behavior gated server-side or by feature flags, or a CLI version you never diffed. This makes field telemetry the third signal after the static diff and the VSIX.

### 5.5 Remote kill switches

A small signed JSON fetched at startup (cached, with a short timeout and fallback to the last known value):
`{ blockedCliVersions: [...], disableFeatures: [{feature:"contextUsage", cliRange:">=2.1.331 <2.1.333"}], banner: {...} }`.
It lets you block a bad CLI or turn off an adopted Observed feature without a plugin release. That matters because Observed and Internal items can change without notice.

---

## 6. End-to-end flow (summary)

```
npm dist-tag moved ─┐
VSIX published ─────┼─► manifest diff (a) + initialize probe (b) + VSIX inventory diff
field telemetry ────┘                │
                                     ▼
                  normalized change records + consumption manifest
                                     │
                         rules engine (B/R/A/I/U) ◄── decision ledger
                                     │
                       VSIX join (adopted/dropped) ── adoption score
                                     │
              [B/U on cli->host shape or default_changed] ─► paid scenario probe ─► re-classify
                                     │
                  LLM proposals (A/R/U only, schema-validated, evidence-cited)
                                     │
                    adoption report (.md + .json) + compat matrix delta
                                     │
             gate: PASS ─► auto PR bump maxTested ─► human approves recommended / A items
                   BLOCKED ─► issue with owners, version optionally added to `blocked`
```

## 7. Open questions

- Is it acceptable to bundle an exact CLI build, as the official extension does? Bundling makes `recommended` a hard pin and the soft ceiling matters less. Not bundling makes runtime defenses the primary safety net.
- Scenario-probe budget: monthly cap and which flows are "required". Proposal: permission prompt, interrupt, resume, tool-result render, compaction.
- Should Internal items the plugin consumes (e.g. depending on an Observed-but-undocumented `ui_*` subtype) be forbidden by lint, or allowed with an explicit `risk: internal` annotation and a kill switch?
- Where are VSIX inventories and probe captures retained, and for how long? Retain inventories (string lists) indefinitely. Discard downloaded VSIX binaries after extraction.
