# ADR-0042: Subagent-transcript navigator — pure-Go spine; DuckDB only for the unknown-schema escape hatch

**Status:** Proposed
**Date:** 2026-09-06
**Revised:** 2026-09-06 (shoshin review — moved DuckDB out of the deterministic spine)
**Revised:** 2026-09-06 (shoshin review — ALL spatial-ASCII views are LLM-composed by the skill from the spine's JSON, so each embeds interpretation; the spine does no visual rendering; not Mermaid/HTML, not flat text, not a fixed Go renderer)
**Revised:** 2026-09-06 (the canonical embedded skill and CLI spine are co-versioned; targeted transcript search belongs in the deterministic spine)

## Context

Agent harnesses (Claude Code, the Claude Agent SDK / `sdk-cli`, `pi`) record their
sessions as JSON-lines transcripts, including subagent spawns. The recurring pain is
"I cannot tell what my subagents are doing": the spawn hierarchy, where a run diverged,
where cost went, and what was worth keeping are all buried in raw JSONL.

An exploratory probe validated that DuckDB reads these transcripts directly and
reconstructs the spawn tree with lifespans and cost in a single query, correcting two
wrong assumptions along the way as query results came back. That probe established three
load-bearing facts:

1. **Three transcript schemas exist and share almost nothing at the field level.**
   - *interactive Claude Code*: one file per session, `uuid`/`parentUuid` record DAG,
     subagents inline as `Task` tool_use blocks.
   - *sdk-cli (Claude Agent SDK)*: session file plus `subagents/agent-*.jsonl` +
     `.meta.json`; the spawn edge is `meta.toolUseId → tool_use block id`, recursive.
   - *pi + `@tintinweb/pi-subagents`*: single file, compact `id`/`parentId` records; the
     spawn edge is a `custom{customType:"subagents:record"}.parentId → Agent toolCall id`;
     native dollar cost; note the stored format differs from the `--mode json` stream.
2. **LLM-generated extraction queries fail silently-but-plausibly.** A wrong join
   resolved 59/59 rows and looked correct until a cross-agent check showed 0/59. Any
   extraction path must ship verifiable assertions, not just example queries.
3. **DuckDB is the engine.** The tool's premise is DuckDB-over-transcripts; the
   dependency is a feature, not an implementation detail to hide.

Because the schemas are irreducibly different and new ones (every harness, every
subagent extension) will keep appearing, hardcoding one parser per schema is a losing
race. The design instead ships a deterministic spine plus a cookbook of recognized-schema
recipes, with an escape hatch for unknown schemas driven by the LLM at navigation time.

## Decision

Add a transcript-navigator command family: a **pure-Go spine** that parses the recognized
schemas with **zero external dependency**, runs **validation assertions automatically**,
and emits a **normalized spawn-DAG relation**. **DuckDB is not a spine dependency.** It is
required only by the **unknown-schema escape hatch**, where the LLM/skill layer composes an
ad-hoc query; there the spine shells out to the `duckdb` CLI (no embedded `go-duckdb`, no
cgo). Discovery-driven focus, lens choice, harvest, and unknown-schema recipe composition
live in that LLM/skill layer on top of the spine.

**Why the spine is pure-Go, not DuckDB-backed (shoshin review finding).** The recognized
schemas are fixed structures, and reconstructing their spawn DAG is a bounded recursive
traversal (`parent_id → id`, or `meta.toolUseId → tool_use id`, or
`custom.parentId → Agent toolCall id`) that pure Go performs directly over the JSON. The
exploratory probe used DuckDB because it was doing *escape-hatch work* — exploring schemas
it did not yet understand — which is exactly the case DuckDB is for. Binding DuckDB to the
deterministic spine imported a dependency the known-schema path does not need. Correctly
placed, most users never install DuckDB: `nn transcript tree` on a recognized schema just
works, and DuckDB is a `doctor`-checked optional prerequisite that only the escape hatch
requires.

The intended command family is:

```text
nn transcript scan [dir]            # discover transcript files, sniff schema, report counts
nn transcript tree <session>        # reconstruct the spawn DAG → normalized relation
nn transcript show <agent-id>       # events within one agent (zoom-in / :enter substrate)
nn transcript search <query> [dir]  # bounded schema-aware event matches with provenance
nn transcript doctor                # detect duckdb, report version and readiness
```

Exact command names and flags may be refined during implementation, but the following
boundaries are part of this decision.

### Keep nn pure-Go; DuckDB is optional and escape-hatch-only

`nn` remains a pure-Go, no-cgo binary. The recognized-schema recipes are pure-Go
traversals with no external dependency. When the escape hatch fires (unknown schema, LLM
composes a query), `nn transcript` invokes the `duckdb` CLI as a subprocess
(`duckdb -json -c "<sql>"`) rather than linking `go-duckdb` — embedding would introduce
cgo, a hundreds-of-megabyte binary, and harder cross-compilation that every `nn` user
would pay for a path most never take. Shelling out keeps the common path dependency-free
and confines DuckDB to where it is actually needed.

### Make the DuckDB dependency explicit, never silent

`nn transcript doctor` detects `duckdb` on PATH and reports its version and readiness.
Because DuckDB is escape-hatch-only, its absence does **not** block known-schema commands;
it blocks only an escape-hatch query. When the escape hatch needs `duckdb` and cannot find
it, the command fails with a structured error and platform-specific installation guidance
(e.g. `brew install duckdb`). Detection is automatic; installation is never automatic or
silent; outside a TTY, nn never prompts.

### Sniff schema, route to a recipe, keep an escape hatch

`nn transcript scan` classifies each transcript into one of the recognized schemas by
structural signature (inline-`Task` / `subagents/*.jsonl`+meta / pi event records) and
reports the classification. `nn transcript tree` runs the recipe bound to the sniffed
schema. When the sniffer matches no known schema, the spine reports the schema as unknown
and defers to the LLM/skill layer to compose a recipe — it does not guess. The three
probed schemas become the first three cookbook recipes, not three code paths.

### Emit one normalized spawn-DAG relation

Every recipe, whatever its input schema, produces the same normalized relation so the
overview and downstream lenses are schema-agnostic:

```text
agent(id, parent_id, type, started, ended, cost, subtree_cost, status, result)
```

`parent_id` is null only for the root. `subtree_cost` is computed after the tree by a
recursive rollup over the spawn edge (per-record cost is an input; subtree cost is
derived from the connection layer, so it cannot be produced per-record). This relation is
the descent-stack overview's data and the `:enter` substrate's index.

**Cost is carried as typed token components.** The relation carries
`input_tokens`, `output_tokens`, `cache_read_tokens`, and `cache_creation_tokens` alongside the
`cost` total, because the four classes have sharply different economics (cache-read is cheap,
output and cache-creation are expensive) and a flat sum cannot distinguish a cheap
cache-read-dominated thread from an expensive one. The `cost` total includes all four classes —
no token class is dropped. Skills rank sessions by *effective* (type-weighted) cost, not the
flat total. (Field names are merged across schemas: Claude Code
`cache_read_input_tokens`/`cache_creation_input_tokens`, pi `cacheRead`/`cacheWrite`.)

**The normalized relation is lossy by design.** It is the overview *projection*, not the
full record. Schema-native fields that do not fit the common shape — pi's native dollar
cost and `custom{customType:"subagents:record"}` records, Claude Code's `uuid` DAG, sdk-cli
`.meta.json` attribution — are deliberately dropped from `tree`'s output. `nn transcript
show <agent-id>` is the **lossless** path: it surfaces the raw schema-specific fields for
one agent. A reader must not treat the normalized columns as complete; `tree` answers
"what is the shape," `show` answers "what exactly happened here."

### Validate every extraction before emitting output

After any extraction — built-in recipe or escape-hatch query — the spine runs mandatory
assertions and refuses to emit output that fails:

- every non-root agent resolves to exactly one parent;
- no agent is its own ancestor (the graph is a DAG; no cycles);
- each spawn timestamp is at or after its parent's start;
- the resolved spawn-edge count matches the count of spawn tool-calls
  (`Agent`/`Task`) in the transcript.

These assertions are the guardrail against silently-plausible wrong joins. Enforcement is
automatic in the spine rather than advisory in the cookbook.

### Layer boundary: spine is deterministic, skill is discovery

The subcommand owns only the deterministic, schema-stable work: scan, recipe execution,
validation, and the normalized relation. The LLM/skill layer owns the discovery-driven
work: per-thread appearance-dimension focus on `:enter`, lens/emphasis selection, the
`nn`-note harvest bridge, and composing a recipe when the sniffer reports an unknown
schema. Node geography (spawn hierarchy on the Y bands, time on X, closed position set)
is deterministic and never moved by the skill layer; the skill may only light appearance
and choose emphasis.

### Provide bounded native hallway projections

`tree --summary --json` returns a bounded aggregate over the complete normalized relation: total
agents, ROOT direct children, non-ROOT edges, parentage-status counts, lifecycle-status counts, and
explicit unknown counts. `tree --parent <id> --limit N --json` returns an envelope of that manager's
canonical direct children with total/returned/omitted counts and an opaque continuation cursor.
Filtering occurs before limiting. The cursor binds a versioned digest of the normalized relation,
selected parent, and canonical child ordering; changed evidence or arguments reject stale rather than
mixing pages. Unknown parents and incompatible projection flags fail explicitly. Existing unflagged
JSON/text and exact-agent/description projections remain unchanged. These projections keep topology
selection, reduction, ordering, and snapshot custody in the pure-Go spine rather than client-side
`jq` reconstruction.

### Co-version the embedded skill and CLI contract

The canonical `nn-transcript` skill is embedded in and served by the same `nn` binary as the
transcript spine through `nn skills get nn-transcript`. The skill and commands therefore ship as
one versioned artifact. Do not add a separate transcript-capabilities command or a companion shell
preflight merely to negotiate between these two halves: an old binary would serve both an old skill
and old command set, so self-probing cannot teach that pair about a newer contract.

Instead, keep the coupling explicit and mechanically guarded. Integration tests enumerate every
transcript command, flag, and JSON field named by the embedded skill and fail when the serving CLI
does not provide it. When diagnosing a source-checkout versus installed-binary mismatch, report the
ordinary `nn` build/version identity; installation remains explicit and is never initiated by the
skill.

### Search transcripts through the spine, not general file grep

Add `nn transcript search <query> [dir]` for targeted lookup across supported transcript schemas.
The command owns deterministic matching and returns bounded results with session, agent, event,
timestamp, role, excerpt, and source-path provenance. It searches the same meaningful-event
projection used by `show` by default, with an explicit raw mode when schema-native payloads are
required. Matching is deterministic and does not assert drift, recurrence, failure, or other
behavioral interpretation.

The embedded skill owns query selection and interpretation of returned matches. It uses
`nn transcript search` rather than `nn grep`: general grep intentionally skips oversized files,
searches JSON syntax and attachment noise, and cannot reconstruct agent/session provenance. Search
does not replace `tree` topology, complete `show` reconstruction, or the whole-session sampling
required for behavioral patterns.

### Continue discovery without losing timestamp ties

`ls --before` remains a strict time filter, not a pagination cursor. Return `modified` with
RFC3339 nanosecond precision and sort discovery by descending mtime, then ascending path for ties.
Each JSON row adds an opaque `cursor`; pass the last row's cursor to
`nn transcript ls <same-dir> --json --limit N --cursor <cursor>` to continue. Repeat any original
`--before` filter unchanged. The JSON result remains an array; an empty array ends continuation.
Page size may change between calls.

The versioned cursor binds the normalized directory, before filter, ordered discovery inventory
(paths, full-precision mtimes, sizes), and the last returned position. Inventory/query changes
reject it as stale or mismatched; malformed or unsupported cursors fail explicitly. This guarantees
complete, nonduplicating traversal of an unchanged inventory, including exact timestamp ties.
It is not a snapshot of transcript contents, sidechain contents, or derived metrics. Restart
without a cursor to discover new/modified sessions. Paths are normalized lexically, not by inode.

### Bound complete thread retrieval to a snapshot

Text-mode `nn transcript show <session> <agent-id> [--raw]` remains byte-compatible and unbounded.
New agent workflows use JSON pagination:

```text
nn transcript show <session> <agent-id> --json [--raw]
                   [--page N] [--snapshot SHA256]
```

The compact JSON envelope is:

```json
{
  "snapshot": "<64 lowercase SHA-256 hex characters>",
  "page": 1,
  "pages": 3,
  "next_page": 2,
  "mode": "meaningful",
  "segments": [
    {"segment": 1, "segments": 8, "text": "<exact UTF-8 fragment>"}
  ]
}
```

The segment stream represents the complete bytes that the corresponding legacy text invocation
would emit, including agent/schema header, composed terminal metadata, and selected event detail.
Concatenating `text` in global one-based segment order reconstructs that output exactly. `--raw`
changes both the projection and `mode`; it never changes sidechain authentication or fallback
selection.

Page 1 defaults when `--page` is omitted and needs no snapshot. Every later page requires the
snapshot returned by page 1. A supplied snapshot on any page must match. The snapshot hashes a
versioned canonical encoding of the normalized session/agent/mode request and complete projected
output, so changed relevant evidence or mismatched arguments fail stale rather than mixing pages.

Every compact encoded page, including its trailing newline and JSON escaping, is at most 48,000
bytes. Packing measures actual encoded bytes. Segments split only at UTF-8 rune boundaries; invalid
UTF-8 in the projected output is rejected. Every page repeats snapshot, page count, and mode;
`next_page` is zero on the last page. Segment boundaries and page packing are deterministic. Even
empty output has one page and one explicit empty segment.

A consumer retrieves every page under one snapshot and verifies complete ordered segments before
making event-derived claims. This mirrors ADR-0035's lossless graph-body transport while retaining
`show`'s existing source-selection and security semantics.

#### Event modes

Pi event records include `message`, `user`, `assistant`, and native top-level `toolResult` records.
Classification and retrieval share this event-type vocabulary. Native tool results are raw event
evidence, not model usage measurements; tool-result-only sidechain evidence leaves usage unavailable.

For all Pi paths (ROOT, inline agent events, direct sidechains, and resolved sidechains), raw detail
emits each selected event's complete message payload in order, retaining usage and tool results.
It does not mean the outer JSONL wrapper or unrelated agents' records. Meaningful show and search
share one content policy: omit attachment records and tool-result roles (`toolResult`,
`tool_result`), as well as typed `tool_result` content blocks; retain text and tool-call previews.
Use raw mode to inspect or search tool-result errors and payloads. These corrections supersede the
old Pi early-return behavior that ignored raw mode and leaked text-form tool results in meaningful
show. Pagination reconstructs the corrected selected projection exactly.

#### Projection ownership and encoding boundary

Path authentication is necessary but not sufficient for Pi sidechain detail. Only event records
whose explicit `agentId` equals the requested agent are eligible, in both meaningful and raw modes.
Foreign, missing-owner, and non-event records must not be presented as that agent's events. If no
eligible events remain, use the existing terminal metadata or provisional unavailable fallback.
The main transcript's established empty-owner-to-ROOT convention remains unchanged; it does not
apply to an externally resolved sidechain.

Tool-input previews retain the existing 120-byte budget but end at a UTF-8 rune boundary before
adding the ellipsis. This corrects malformed previews for valid Unicode input in the shared renderer.
Text and paginated output use the same corrected projection; byte compatibility does not preserve
foreign-event attribution or invalid byte truncation as required behavior.

Lossless transport means exact reconstruction of that projection, not preservation or validation
of original source bytes. Existing JSON string decoding can replace malformed source UTF-8 with
U+FFFD; two sources producing the same projection under the same request can share a snapshot.
Raw modes that retain invalid bytes in the projection are rejected by JSON pagination. Adding
source-wide strict UTF-8 validation would be a separate ingestion-policy change, not part of this
transport contract. These guarantees apply consistently to text and raw projections; `--raw` does
not broaden event ownership.

### Semantic thread layouts

Thread-entry presentation separates immutable spawn topology from interpretive finding coordinates.
The co-versioned skill requires meaningful declared axes, a distinct role for each visual channel,
evidence-grounded placement and explicit uncertainty. Axis choices, palettes and icons are selected
for the thread/question, not fixed to a particular implementation-stage/evidence-strength example.
Changing a lens must not silently change encoding meanings. This changes presentation guidance only;
no CLI topology, identity, cost, schema or evidence authority is changed. Wording regression tests guard
the served instructions, not the semantic quality of an agent's rendered diagram.

### Deterministic tool-volume summaries

`events --summary tools [--limit N] [--group-by tool]` reuses authenticated ledger selection,
normalized sizes and unique joins. The independently versioned `nn.transcript.tool-summary/v1`
response includes counts, call/result join-status counts, per-size known sums and missing denominators,
largest results, matched call command/argument previews, and optional exact-name groups (null for unknown).
Only tool events contribute, never their enclosing message events. Result grouping uses its recorded
name, falling back only to a uniquely matched call's name. Ambiguous/missing/unavailable joins never
produce a guessed call. Inconsistent matched references reject.

The default limit is 5; 0 through 100 are accepted. Results sort by known text-character size descending,
unknown sizes last, then ledger ordinal and event ID. results_returned/results_omitted disclose the tail.
Known-only size totals are lower bounds when unknown records exist; wholly unknown and empty sets have
null totals, distinct from measured zero. Negative counts and sum overflow reject. Commands and argument
JSON previews are at most 512 UTF-8 bytes each, with truncation flags, full argument-byte metadata and
an argument JSON SHA-256 for displayed calls. Non-command tools retain argument previews and null commands.
Sizes are not tokens, prices, context attribution, or evidence of necessity/waste.

The response snapshot binds options, returned reductions and the identity/tools metadata ledger snapshot;
displayed call argument digests additionally bind their argument JSON. Other payloads are not covered.
--snapshot revalidates the summary. --limit/--group-by require tools mode; --bucket-size requires usage
mode. Existing summary incompatibilities with --select/--payload/--event/--all/--page remain. Output is
at most 48,000 bytes including newline; oversized responses fail with guidance to reduce the limit or
omit groups, never silently omit groups. Existing ledger and usage-summary outputs remain unchanged.

### Deterministic usage summaries

`events <session> <agent-id> --summary usage [--bucket-size N]` reduces the existing authenticated
identity/usage ledger, not another source parser. It emits one complete `nn.transcript.usage-summary/v1`
JSON object with `ledger_snapshot`, summary `snapshot`, schema/detail status, `bucket_size`, `stats`,
and `buckets`. --select/--payload/--event/--all/--page are incompatible, even if explicitly set to defaults.
--bucket-size requires summary mode; zero disables buckets, negative values reject. Optional --snapshot
pins the summary (not the underlying ledger snapshot) for fail-closed revalidation.

Records are assistant message records including missing usage, not independently verified API calls.
Derived tool events never contribute. Stats include records, complete/partial/unavailable record counts,
fully measured zero-usage records, status, nullable component sums, per-component missing_counts,
known_total_tokens, and nullable total_tokens. Component sums are known-only lower bounds when any
contributing component is missing; wholly unknown components stay null. Total is exact only for a
nonempty collection of complete records. Empty/wholly unknown collections are unavailable, not measured zero.
Negative counters and integer overflow reject rather than wrap.

Context is input plus cache-read counts when both exist. First/last refer to the actual boundary records
(and can be null); min/max/average_known use only known contexts, including zero. Known/unknown and
zero-context record counts disclose that denominator. Buckets partition usage records in ledger order,
with one-based first_record/last_record and first_event_id/last_event_id; the final bucket may be shorter.
No task-phase, reset, waste, price or source-completeness inference is introduced.

The summary digest binds its entire normalized result, bucket size and source ledger snapshot. Output
including newline is capped at 48,000 bytes. Too many buckets fail with guidance to increase bucket size
or omit buckets; none are silently omitted. Existing event/export modes and their snapshots are unchanged.

### Targeted tree projection and complete exports

`tree --agent <id> --fields <comma-separated top-level JSON fields> --json` filters only after
whole-tree validation/repair and cost rollup. It returns the usual array shape, preserves values,
rejects unknown agents/fields and empty selectors, and emits null for explicitly requested absent
optional fields. Either selector requires JSON. With neither selector, output is unchanged.
This avoids client-side row/field extraction; it does not promise cheaper tree construction.

`show --all --json` returns `{all:true,snapshot,mode,text}` with complete reconstructed text.
`events --all` returns the usual envelope with `all:true`, page/pages=1, next_page=0, and complete
unfragmented event objects. These are explicitly UNBOUNDED exports, not 48,000-byte pages.
Both reject explicitly supplied --page/--snapshot, even page 1; show --all requires --json.
All selected content is computed once, with the same snapshot as the corresponding bounded
projection. The all flag changes transport only, not selection, ownership, or source authority.
Default bounded envelopes omit the all field and remain unchanged. Large exports should be piped
to a JSON consumer or redirected to a file rather than assumed to fit an agent tool response.

### Normalized event ledger

`nn transcript events <session> <agent-id> --json` provides a versioned evidence ledger,
not a behavioral analysis. `--select identity,message,usage,tools,lifecycle` selects facets
(all by default; identity is mandatory and automatically included). `--payload` opts into
native message, tool-block, or lifecycle-data payloads. No payload or arguments appear by default.
`--event <event-id>` retrieves one exact event, retaining its ledger ordinal; unknown IDs fail.

Each event has `event_id`, `agent_id`, `ordinal`, `kind`, `timestamp`, `timestamp_source`, and
`source` (canonical absolute path, one-based decoded-record ordinal, optional native record ID).
Decoded-record ordinals exclude blank/malformed lines under the existing reader recipe; they are
not physical line numbers. Event IDs hash source path, decoded-record ordinal, owner and extraction
slot, not mutable payload or selected facets. They identify a position, not immutable content.
Ordering is source-path then decoded-record ordinal then extraction order, not cross-source chronology.
Message events precede their extracted tool events. Native duplicate IDs never collapse records.

Usage belongs only to assistant message events, never tool calls/results or lifecycle records.
Known components are nonnegative integer counts; absent/null components remain null, with
complete/partial/unavailable status and known total. Total and input-plus-cache context are null
when their required components are unknown. Reported totals do not override component accounting.
Text bytes/Unicode characters and serialized content/argument bytes are size measurements, not tokens.
Tool joins use exact call IDs within this agent's selected ledger, reporting matched/missing/ambiguous/
unavailable; missing IDs and duplicate IDs are never guessed into a one-to-one relation.
Lifecycle values remain producer observations, not task outcomes.

Pi message selection shares show's authenticated path and explicit sidechain-owner rules. The ledger
also retains all matching parent terminal records, not just the terminal fallback chosen by show.
SDK child paths are confined to the canonical subagents directory and exclude explicitly foreign owners.
Claude Code ROOT messages are available; inline Task execution is unavailable (parent result text is
not child execution). `detail_status` distinguishes available message evidence from unavailable detail;
terminal-only fallback is not an empty successful execution. Unknown schemas fail closed.

Pages carry `version`, `snapshot`, `page`, `pages`, `next_page`, normalized `select`, `payload`,
`schema`, `detail_status`, `event_filter`, and `events`. Each encoded page including newline is at most 48,000 bytes.
Ordinary entries are event objects. An oversized event becomes entries with `event_id`, `ordinal`,
`segment`, `segments`, and `text`: concatenate all ordered fragments and JSON-decode to recover that
one event. Never count fragments as events. Every later page requires the page-1 `--snapshot`.
The digest binds the normalized request and complete selected projection; changing selected evidence,
agent, facets or payload mode rejects continuation. It does not certify original source completeness,
unselected payload immutability, or raw byte custody. Existing show/tree output remains unchanged.

### Pi lifecycle and usage scope disclosure

Pi `tree --json` nodes add `evidence_scope` without changing legacy fields or text output.
Other schema recipes omit this Pi-specific object. Its fields describe existing projection sources:

- `status`: `last_terminal_record`, `background_spawn_record`, or `unavailable` for ROOT.
- `timestamps`: `last_terminal_record` for terminal children, `root_message_history` for ROOT,
  or `unavailable` for provisional children. Last means last in file order, not maximum timestamp.
  Scope identifies provenance, not availability or precision; missing legacy timestamps remain empty.
- `cost`: `retained_sidechain_history` only after authenticated owned event hydration,
  `root_message_history` for ROOT, or `unavailable` when child usage was not hydrated.
  These labels inherit existing recipe semantics and do not certify original-source completeness.
- `subtree_cost`: `subtree_aggregate`, combining each included node's own cost under its own scope.
- `terminal_record_count`: number of matching producer terminal records encountered, not an inferred
  attempt count; duplicate records remain counted. ROOT and provisional nodes normally have zero.

A resumed child's cumulative usage may span more history than its last terminal record's timestamps.
Do not calculate a rate or compare a run's usage using these incompatible windows. Producer
`completed` is a terminal status, not evidence that the requested task succeeded; task outcome requires
reading the complete thread and remains interpretation, never a status inferred by this recipe.
No per-run token allocation, lifecycle repair, or task-outcome inference is introduced. The serving
skill must disclose these distinctions; cohort summaries still aggregate own costs rather than
reclassifying them as latest-run usage.

### Bounded authoritative discovery summaries

Every `ls --json` row adds `summary`, computed from the same un-repaired agent slice already built
for that row. Existing fields, text output, JSON array shape, ordering, and cursor custody remain
unchanged. No second tree read is needed. Failed tree construction yields `summary: null`; legacy
zero counters and `(tree unavailable)` are not evidence of an empty or measured-zero session.

A non-null summary contains:

- `cost`: `total_tokens`, `input_tokens`, `output_tokens`, `cache_read_tokens`,
  `cache_creation_tokens`, `measured_agents`, `unavailable_agents`, and `status`. Sum each agent's
  **own** fields once, never subtree totals; `total_tokens` equals the legacy `total_cost`.
  `status` is `complete` when all agents in a nonempty slice have `cost_status: complete`,
  `unavailable` when none do, and `partial` otherwise. Missing/unrecognized own authority counts
  as unavailable. These statuses inherit the schema recipe's measurement semantics; they do not
  newly certify source completeness. Partial totals are lower bounds, not exact rankings.
- `topology_status`: `complete` for a rooted forest with nonempty unique IDs, resolved parents,
  and no cycles; otherwise `invalid`. This checks parentage only, not timestamps or source
  completeness. `topology` is null when invalid; otherwise it contains `root_count`, `edge_count`,
  `max_depth` (edges from a root, root depth zero), and `max_children` (direct children).
  An empty slice has zero for all four metrics. No silent repair is performed for summaries.
- `agent_types`: at most 16 `{type, count}` entries, sorted count-descending then label-ascending.
  Each retained type label is at most 64 UTF-8 bytes. Longer labels are omitted, not shortened;
  the empty label represents an unknown recorded type. Entries are exact frequencies across all
  agents, including ROOT. `distinct_agent_types`, `omitted_type_count`, `omitted_agent_count`,
  and `types_truncated` disclose the entire omitted tail, including oversized labels. Retain the
  first 16 eligible entries; do not mistake omission for absence.

The compact JSON encoding of a non-null summary is at most 8192 bytes, including worst-case string
escaping. This is a summary bound, not a bound on the whole row, cohort, or pretty-printed transport.
No agent IDs, result strings, or event bodies are included. Summaries support aggregate cohort
comparisons without full tree transport; exact edges, agent selection, subtree attribution, and
uncapped type identities still require `tree --json`. Behavioral claims still require events.

### Higher-level views get deterministic dimensions eagerly, inferred dimensions on demand

The zoom-out overview needs *aggregate* dimension analysis — signals lit across many agents
at once (which subtrees are expensive, where errors cluster, where work stalled). These
split by tier:

- **Tier 0/1 (deterministic): computed eagerly by the spine and available to the overview.**
  status, per-record cost, `subtree_cost`, tools, errors, recursion-depth, and the
  deterministic Tier-1 signals (friction, diverge/converge joins, handoff/blocking). The
  wedge jobs (debug, recover-context) depend only on these, so the overview serves the
  wedge with no inference. Which Tier-1 signals are core columns of the normalized relation
  versus computed on demand by a separate spine query is deferred to implementation.
- **Tier-2 (inferred: instruction-drift, groundedness, pivots, context-re-derivation):
  never computed eagerly across the overview.** Per the discovery principle, inference is
  paid per-thread on `:enter`. A Tier-2 signal is therefore not lit across the whole
  overview by default — doing so would smear plausible-but-false labels across the very map
  used to navigate. Tier-2 appears when a thread is entered, and additionally via an
  **explicit, opt-in sweep**: a deliberately expensive command that runs inference over all
  threads to light one Tier-2 dimension across the overview when the user asks for it (e.g.
  "show every thread that drifted"). The sweep is never the default and never implicit.

This keeps the overview cheap and honest: deterministic signals are trustworthy and always
present; inferred signals are opt-in and paid for explicitly.

### Visual navigation is LLM-composed, not a CLI render flag

The navigator's default surface is text, but text is the **default, not the ceiling**. Two
navigation views want *visual diagram* output:

- the **session picker** — a rendered diagram per recent session (spawn-tree shape), not a
  one-line ASCII mini-tree;
- the **`:enter` view** — a diagram of the thread's discovered dimensions with visual
  treatment: color encoding emphasis/tension, size encoding cost, shape encoding type, edges
  encoding connection.

The distinction is not *text vs. graphics* — it is **flat text (one-line strings, prose
lists) vs. spatial ASCII (multi-line 2D diagrams drawn in characters, with color markers).**
The defect in the shipped mini-tree was not that it was ASCII; it was that it was a *one-line
string* instead of a *drawn diagram*. The visual surface is **spatial ASCII**, not Mermaid or
HTML — this keeps the navigator relay-friendly and dependency-free, and it reuses the existing
color-relay vocabulary (colored-circle markers: tension / lateral / structural, chosen to
survive markdown and relay).

Critically, **all** spatial-ASCII rendering — the session picker, the tree overview, and the
`:enter` dimension diagram — is **LLM-composed by the skill layer from the spine's JSON**, not
drawn by a fixed Go renderer. The spine performs **no visual composition**: it emits the
`--json` normalized relation (agents, edges, cost, status) and, at most, a plain text fallback
for non-interactive use. It does not draw spatial diagrams.

The reason a fixed Go renderer is wrong — even for the deterministic skeleton — is that the
value of the visual view is **embedded interpretation**, and interpretation must vary per view:
which branch to emphasize, what to color (the thread that drifted, the expensive subtree),
what to annotate (a pivot, a stall), what to collapse (the boring parts). A hardcoded renderer
draws every session identically regardless of what is interesting about it; an LLM composing
the diagram from the JSON draws *this* session to surface *what matters here*. Most dimensions
worth seeing on `:enter` (instruction-drift, groundedness, pivots, purpose) are Tier-2 anyway —
they require interpretation the spine cannot do — so the composition layer is already the LLM;
extending it to the skeleton's layout keeps a single, consistent, interpretation-bearing
renderer rather than splitting a dumb Go tree from a smart skill diagram.

Therefore:

- The **spine** emits data only: the `--json` normalized relation, plus a plain text fallback.
  No box-drawing, no color, no layout decisions.
- The **skill layer** composes every spatial-ASCII view from that JSON: it lays out the tree
  with box-drawing branches, applies the color-relay markers (tension / lateral / structural),
  encodes cost as width/size and type as shape/label, and — on `:enter` — infers the Tier-2
  dimensions and draws them into the diagram. Every view is an *inference product* that can
  differ each time based on what the LLM judges salient, consistent with "navigation drives
  inference."

This preserves the pure-Go, no-cgo spine untouched and keeps DuckDB escape-hatch-only. The
visual layer is entirely additive and lives in the skill: no new dependency, no spine
rendering code, no rearchitecture.

### `:enter` is a navigable view, not an annotated node

The `:enter` view is a positioned navigation surface (substance + structure + direction), not a
flat list of dimensions (which is a map with no substance or direction). It is composed by the
skill from the spine's JSON + Tier-2 inference, governed by three rules:

- **Salience** picks the 2–3 most interesting dimensions:
  `score = (deviation + outlier + surprise) × lens_weight`, dropping any dimension with
  deviation=0 AND outlier=0 (the drop rule is what removes the boring list). Deterministic
  signals (outlier, cost) come from the spine; interpretive ones (drift, groundedness, surprise)
  from the LLM.
- **Encoding** draws each pick in its natural geometry with a single layout backbone plus
  non-colliding marks: layout-consuming encodings (containment, X/Y, adjacency) are mutually
  exclusive as the backbone; mark-only encodings (color, width, fill, arrows) attach to distinct
  channels. Backbone = highest-salience layout dimension, else the deterministic spawn tree.
- **Pivots** make every labeled feature an enterable move that names its relation to the current
  subject (explains, caused-by, contains, contrasts, then). A pivot must clear the drop rule
  (never pivot to a boring dimension). `contains` drills to a child node; every other relation
  reframes the same node around the new subject dimension, keeping a breadcrumb.

This keeps the spine deterministic (data only) and locates the navigation grammar in the skill,
consistent with the LLM-composed spatial-ASCII decision above.

## Consequences

- `nn` stays pure-Go and small; DuckDB becomes a named, doctor-checked prerequisite.
- New harnesses are onboarded by adding an SQL recipe and a sniff signature, or handled
  ad hoc by the skill layer's escape hatch — no new Go code per schema.
- The normalized relation decouples the overview/lenses from transcript formats.
- Validation makes LLM-composed extraction safe to trust, at the cost of requiring every
  recipe to satisfy the assertion suite.
- Live supervision (tailing an in-flight transcript) is explicitly out of scope for this
  decision; it is a different, incremental-ingest data story.

## Additional acceptance criteria

- No separate `nn transcript capabilities` command or transcript preflight shell script is added
  for communication between the embedded skill and its serving CLI.
- `nn transcript search` is bounded, deterministic, schema-aware, and emits session/agent/event
  provenance in JSON.
- Search defaults to meaningful events and requires an explicit raw option for schema-native noise.
- The embedded skill uses transcript search instead of `nn grep` for transcript content.
- A co-versioning integration test fails if the embedded skill names a transcript command, flag, or
  JSON field absent from the serving CLI.

## Open burden

Placing this in `nn` (rather than a standalone tool) is justified only if the **harvest
bridge is load-bearing** — i.e. navigating a transcript routinely produces durable `nn`
notes linked back to threads. If harvest proves decorative in practice, the in-nn
placement should be revisited: the transcript navigator would then be a separate tool that
merely *targets* nn for capture, not a subcommand of it. This burden is carried, not
discharged, by this ADR.
