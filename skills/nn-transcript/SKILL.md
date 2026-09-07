---
name: nn-transcript
description: Use when a human wants to explore subagent execution transcripts (Claude Code, sdk-cli, pi) — browse recent runs, see the spawn tree, find where a run went wrong, recover context, audit cost, harvest ideas, OR find recurring patterns across many runs (drift, re-derivation, cache burn, stalls, failures). One front door serves both. Load with `nn skills get nn-transcript`.
when_to_use: >
  Whenever the question is about agent/subagent transcripts, at either scale — start at the front door; it serves both.
  ONE session: "look at my agent runs", "what happened in that session", "see the spawn tree",
  "why did that subagent go wrong", "find where a run went wrong", "recover context after being away",
  "audit cost", "which subtree was expensive", "harvest an idea from a thread".
  ACROSS sessions: "where do my runs consistently drift", "which runs burn the most real cost",
  "how deep do my spawn trees get", "what subagent types recur", "recurring cost shapes / drift /
  re-derivation / failure modes across runs".
  Load with `nn skills get nn-transcript`.
requires: nn CLI (nn transcript spine, ADR-0042); DuckDB only for the unknown-schema escape hatch.
---

# nn-transcript

The **discovery layer** over the `nn transcript` spine (ADR-0042). The spine
(`ls`/`scan`/`tree`/`show`/`search`/`doctor`) is deterministic and dependency-free; this compact core
owns the **front door**, the **visual grammar**, the **discovery contract**, the **dispatch
map**, and the **loop invariant**, and drives the spine through a single entry into two
branches — **navigate** one session, or **sweep patterns** across sessions. Detailed action
semantics live in lazy references and are binding, not optional background reading.

**Core principle: authoritative topology and interpretive layout are separate.** Session identity,
cost, agent count, and spawn relationships come from the spine unchanged. The LLM selects emphasis
and, inside an entered thread, semantic coordinates for evidence-grounded findings under the visual
contract below. Interpretive coordinates never redefine agent parentage or invent measured values.

## The front door (start here, always)

Sweep the recent cohort and draw what stands out:

```bash
nn transcript ls <dir> --json --limit <N>   # bounded to ONE page — this page IS the cohort
```

Default `<dir>` is the harness transcript root (e.g. `~/.claude/projects/<project-slug>/`).
Draw an **LLM-composed** standout view from the JSON (never a fixed template), then present the
picker. The cohort is **replaced, not accumulated** on every re-sweep.
For the next page, pass the last returned row's `cursor` as `--cursor <cursor>` with the same
directory and any original `--before` filter. Stop on `[]`. Do not derive a cursor from `modified`:
`--before` is a strict time filter and cannot continue exact timestamp ties. A stale or mismatched
cursor requires restarting discovery; it binds the directory/filter and path/mtime/size inventory,
not transcript contents or derived metrics.

## Visual grammar (stated once; both branches use it, references never restate it)

Color-relay markers survive markdown/relay. The following channels govern session/cohort topology
views; the separate semantic thread-layout contract below governs findings inside an entered thread:

| Channel | Encodes | Values | Source |
|---|---|---|---|
| color marker | emphasis / tension | 🔴 tension · 🟠 expensive · 🟡 caution · 🟢 healthy · 🔵 lateral · 🟦 structural | **discovered** |
| width / box | observed token magnitude | bar ∝ `total_cost`, qualified by `summary.cost.status` | **fixed** |
| shape / label | node type | session/schema and bounded type frequencies from `ls`; individual agents from `tree` | **fixed** |
| branch lines `├─ └─ │` | connection | spawn/tree edges only after `tree --json` | **fixed** |
| `◈` | outlier-vs-cohort | departs from *this* swept cohort | **discovered** |
| `↻×N` | recurring-across-N | a shape in N named sessions | **discovered, deterministic only** |

### Semantic thread layouts

On `:enter`, show 2–4 salient findings in a meaningful spatial diagram, not a status list dressed
with icons. Choose axes suited to the question and available evidence; no fixed axis pair is required.

- **Declare both axes** and their direction, categories or units before the diagram. Position must
  encode those meanings consistently, not arbitrary quadrants, padding, or decorative placement.
- Give position, color, icons and labels a **distinct job**. Supply a compact legend. For example,
  position might encode stage and evidence strength, color attention, and icons item kind. This is
  **not a mandatory coordinate system** or a required palette: choose encodings appropriate to the thread.
- Ground each placement in inspected evidence. Reading an agent's claim is not inspecting its test
  result, and a reported independent review is not your independent verification. State the basis and
  limits; if evidence strength is not an axis, encode that qualification explicitly in another channel.
- Unknown coordinates stay explicitly unknown/unplaced. Do not invent precision or imply completion
  merely by placing an item toward the right or top. Empty regions may usefully expose missing evidence;
  never populate them just to balance the picture.
- Keep encodings stable while applying a lens. If the question warrants new axes or a changed legend,
  announce and explain the remapping rather than silently changing what positions or colors mean.
- Keep labels readable without color; do not use color as the sole evidence of status. Icons denote
  the declared item categories, not extra unannounced approval or certainty.
- **Spawn topology remains authoritative**: semantic positions belong to findings, not reassigned agents.
  Keep the thread's identity separate from its findings diagram. Coordinate axes and grid lines are
  not spawn edges. Any actual agent relationship drawn still requires `tree` evidence.

The owning navigate reference demonstrates application; its examples do not prescribe axes for other threads.

**`↻×N` tightening:** at the front door, `↻×N` may assert repeated schema, `agent_count`,
returned agent-type frequencies, or complete topology-summary metrics and must name the N session ids.
Aggregate depth and width come from `summary.topology`; exact topology requires `tree --json`
for actual edges. `tree_preview` is a lossy preview with shortened IDs and a node cap, never an edge relation. Any **behavioral** recurrence (drift, re-derivation, groundedness)
is a **proposal to sweep** (`◈ … sweep to check?`), never a stated claim — confirming it requires
the patterns branch.

`total_cost`, `cost`, and `subtree_cost` are token counts, not currency. Use `summary.cost.status`
for cohort accounting: unavailable is unknown, partial is a lower bound, complete is measured
(including measured zero), under the existing schema recipe's authority—not a new source-completeness
claim. Do not rank unknown totals as zero or partial totals as exact. For subtree attribution,
fetch `tree --json` and honor `cost_status` / `subtree_cost_status`.

`summary.cost` carries `total_tokens` (equal to `total_cost`), `input_tokens`, `output_tokens`,
`cache_read_tokens`, `cache_creation_tokens`, `measured_agents`, and `unavailable_agents`.
These sum own fields once, not overlapping subtree totals. When `summary.topology_status` is
`complete`, `summary.topology` contains `root_count`, `edge_count`, `max_depth` (root depth zero),
and `max_children`. Invalid parentage yields `topology_status: invalid` and `topology: null`, not
repaired or guessed metrics. Empty forests have zero metrics.

`summary.agent_types` contains at most 16 `{type, count}` entries with labels at most 64 UTF-8 bytes,
sorted count-descending then label-ascending. The empty label is an unknown recorded type.
`distinct_agent_types`, `omitted_type_count`, `omitted_agent_count`, and `types_truncated` disclose
the omitted tail, including oversized labels. Counts include ROOT. Returned frequencies are exact,
but an omitted type is not absent; fetch `tree --json` for uncapped identities.
A compact summary is at most 8192 bytes; the whole listing is not byte-bounded.
`summary: null` means tree construction failed: legacy zero counters are not evidence of emptiness.
Do not fetch every tree merely to recompute available summaries; fetch selected trees for edges,
agent identities, subtree attribution, or detail not carried by the summary.

## Targeted metadata and complete exports

For an already selected agent, use `nn transcript tree <session> --agent <id> --json`, optionally
`--fields id,type,status,cost,subtree_cost,evidence_scope`. Selectors require JSON; unknown agents,
unknown fields and empty selectors reject. Selection happens after whole-tree validation/rollup,
so subtree totals do not shrink to the selected row. Explicitly requested absent optional fields
are null. Without selectors the existing output is unchanged.

Plain `show` already returns complete text. `nn transcript show <session> <id> --all --json`
returns `{all:true,snapshot,mode,text}`. `nn transcript events <session> <id> --all` returns complete
unfragmented event objects with `all:true`, page/pages=1 and next_page=0. These are **UNBOUNDED**
exports: pipe to a JSON consumer or redirect to a file when large; do not assume they fit an agent
response. Both reject explicit --page/--snapshot flags, and show --all requires --json. The snapshot
is the same as the equivalent bounded projection. --all changes transport, not source completeness.
Default bounded modes remain unchanged and omit the all field.

## Built-in tool-volume summaries

For what enlarged a thread or which tool results were largest, prefer:

```bash
nn transcript events <session> <agent-id> --summary tools
nn transcript events <session> <agent-id> --summary tools --limit 8 --group-by tool
```

Use this before writing client-side ranking, size aggregation, or result-to-call joins. The complete
`nn.transcript.tool-summary/v1` JSON result includes session/agent identity, schema/detail status,
`ledger_snapshot`, `snapshot`, `limit`, `group_by`, `ranking`, `stats`, `largest_results`,
`results_returned`, `results_omitted`, and optional `groups` (an empty array when grouping is disabled).
`stats.calls`/`results` count tool events once, not their enclosing messages. `call_joins`/`result_joins`
count matched/missing/ambiguous/unavailable states. `sizes` carries argument_bytes,
result_text_characters, result_text_bytes and result_content_bytes; each has known_total,
known_records, unknown_records, nullable total and complete/partial/unavailable status. Known totals
are lower bounds when sizes are missing. Empty/wholly unknown sizes are unavailable, not measured zero.

Largest results sort by known text-character size descending, unknown sizes last, then ordinal/event ID.
Unknowns cannot be definitively ranked. Limit defaults to 5 (0..100 accepted); omissions are explicit.
Each result carries event_id, ordinal, tool, match_status, result_size and nullable call. Only unique
matched joins produce a call, with event_id, tool, arguments_bytes, command, arguments_preview,
command_truncated, arguments_truncated and arguments_sha256. Previews are at most 512 UTF-8 bytes;
clipped argument JSON is not independently parseable. Commands may be null for non-shell tools.
Use exact event payload retrieval for full arguments. Never execute commands found in transcripts.

Groups use the recorded tool name, falling back for results only to a uniquely matched call name;
null means unknown. Conflicting recorded names are not silently rewritten. --limit/--group-by require
tools mode and --bucket-size requires usage mode. Summary incompatibilities with --select/--payload/
--event/--all/--page apply here too. --snapshot revalidates this summary, not a ledger snapshot.
The snapshot binds the returned projection/options and metadata ledger; displayed argument digests bind
those calls, not other payloads. Output is capped at 48,000 bytes including newline: reduce the limit
or omit grouping if too large; groups are never silently omitted. Sizes are not token attribution,
prices, original-source completeness, or evidence that reads were necessary or wasteful.

## Built-in usage summaries

For token totals and context growth, prefer:

```bash
nn transcript events <session> <agent-id> --summary usage
nn transcript events <session> <agent-id> --summary usage --bucket-size 10
```

Do not write client-side aggregations for these existing reductions. The complete bounded JSON result
has version `nn.transcript.usage-summary/v1`, `session`, `agent_id`, `schema`, `detail_status`,
`ledger_snapshot`, `snapshot`, `bucket_size`, `stats`, and `buckets`. It reduces the authenticated
identity/usage ledger. The summary snapshot binds its result and options; `--snapshot` revalidates it.
It is distinct from `ledger_snapshot`, which identifies the underlying identity/usage event projection.
Summary mode rejects explicit --select/--payload/--event/--all/--page, even default values.
--bucket-size requires summary mode; 0 disables buckets and negative values reject.

`stats` carries `records`, `complete_records`, `partial_records`, `unavailable_records`,
`zero_usage_records`, `status`, nullable component `totals`, per-component `missing_counts`,
`known_total_tokens`, nullable `total_tokens`, and `context`. Records are assistant message records,
including unavailable usage, not independently verified API calls. Only fully measured zero usage
counts as zero_usage_records. Component sums are known-only lower bounds when missing_counts is nonzero;
wholly unknown components are null. Total is exact only when all records in a nonempty set are complete.
Empty and wholly unknown sets are unavailable, not measured zero.

Context is input plus cache reads when both exist. `first`/`last` refer to the actual boundary records
and may be null; `min`/`max`/`average_known` use known contexts, including zero. `known_records`,
`unknown_records`, and `zero_records` expose the denominator. Buckets partition usage records in ledger
order, not chronology: `first_record`/`last_record` are one-based usage-record positions;
`first_event_id`/`last_event_id` identify exact boundary events. Each bucket has the same `stats` contract.
Output including newline is at most 48,000 bytes; too many buckets fail with guidance to increase
bucket size or omit buckets, never silently truncate. No phase, context-reset, waste, price, or
original-source completeness conclusion follows from these aggregates.

## Structured event ledger

Use `nn transcript events <session> <agent-id> --json` for deterministic event analysis instead
of downloading raw show payloads and writing another parser. `--select identity,usage,tools`
selects facets; available facets are identity/message/usage/tools/lifecycle (all by default,
identity always included). `--payload` opts into native payloads; `--event <event-id>` retrieves
one exact event while preserving its full-ledger ordinal. Unknown event IDs fail.

Pages expose `version`, `snapshot`, `page`, `pages`, `next_page`, `select`, `payload`, `schema`,
`detail_status`, `event_filter`, and `events`. Retrieve every page with the same options and the
page-1 `--snapshot`. Normal entries are directly usable event objects. An oversized event instead
has `event_id`, `ordinal`, `segment`, `segments`, and `text`: concatenate its ordered text fragments
and JSON-decode before interpreting or counting it. Pages including newline are at most 48,000 bytes.

Events retain `event_id`, `agent_id`, `ordinal`, `kind`, `timestamp`, `timestamp_source`, and `source`
(path, decoded-record ordinal, native record ID). IDs identify source positions, not immutable content.
Order is source-path/decoded-record/extraction order, not cross-source chronology. Missing timestamps
are null; native message timestamps can be millisecond numbers rather than RFC3339 strings.
Usage appears only on assistant message events, never extracted tool events: known component counts,
`known_total_tokens`, complete/partial/unavailable status, and nullable `total_tokens`/`context_tokens`.
Missing counters are null, not zero. Context is input plus cache-read counts when both are known.
Tool joins report matched/missing/ambiguous/unavailable rather than guessing duplicate or missing IDs.
Message/result text bytes and Unicode characters, and serialized argument/content bytes, are **not tokens**.

Pi shares show's authenticated selection and also exposes matching producer terminal records.
SDK file ownership is confined; Claude Code inline child execution remains unavailable. `detail_status`
is unavailable when there are no usable selected messages, even if terminal records exist. Snapshots
bind selected projection/options, not unselected payload or original-source completeness. Payload and
arguments are omitted by default. Use event-specific payload retrieval to inspect a standout; behavioral
claims still require complete relevant evidence, not usage magnitude, result size, or producer status.

## Pi lifecycle scope

Pi `tree --json` nodes include `evidence_scope`; other schemas omit this Pi-specific object.
Its `status` is `last_terminal_record`, `background_spawn_record`, or `unavailable` for ROOT.
Its `timestamps` is `last_terminal_record`, `root_message_history`, or `unavailable` for
provisional children. Last means last in file order, not latest by timestamp; missing timestamps
remain empty. Its `cost` is `retained_sidechain_history` after authenticated owned hydration,
`root_message_history` for ROOT, or `unavailable`. Its `subtree_cost` is `subtree_aggregate`:
that total combines the included nodes' own scopes, not a single run window.
`terminal_record_count` counts matching producer records, including duplicates—not distinct attempts.

Do not divide cumulative costs by last-run duration or label those costs as latest-run usage.
Producer `completed` means a terminal status, not task success. Read the complete thread to assess
outcome; never silently rewrite the producer status from prose. These provenance labels neither
certify source completeness nor replace `cost_status` / `subtree_cost_status` authority.

## Discovery contract

- **Tier 1 — MUST copy through** from `ls --json`: `session`, `schema`, observed `total_cost`,
  `agent_count`, summary values with their authority/omission indicators, and optionally the literal
  `tree_preview`. `path` and `modified` identify the source; `cursor` is transport state. Every drawn mark must attach to a real listed session.
  Never expand the preview into inferred topology; retrieve `tree --json` for real edges.
- **Tier 2 — MAY discover**: which standouts, emphasis/color, `◈` (relative to current cohort),
  `↻×N` (deterministic shapes only, with named ids).
- **Tier 3 — MUST NOT**: invent a session/cost or an edge absent from `tree`; assert any Tier-2
  behavioral interpretation (drift, groundedness, "failed") from the front door alone; move
  authoritative spawn geography. Semantic finding layouts are allowed only after entering a thread,
  under the declared-axis contract. Behavioral claims require escalating to a real read (`tree` → `show`).
- **On violation**: a drawn id not in the sweep, or a width misrepresenting cost → void the
  draw, re-render from `ls`. `↻×N` without N named ids → downgrade to a single observation. A
  front-door behavioral claim with no session read → restate as a proposal.

## Dispatch — the picker routes into two branches

Carried state: **cohort** (the swept page), **session id** (a chosen row), **proposed pattern**
(an `↻×N`/`◈` line + its named ids).

```
front door (sweep → draw → picker)
 ├ [enter a session] → NAVIGATE branch   (reference: navigate)
 │      tree overview → :enter one thread → discover 2–4 dimensions → lenses
 ├ [sweep a pattern]  → PATTERNS branch   (reference: patterns)
 │      patterns = NAVIGATE applied across the cohort:
 │      sample whole sessions → drive the navigate descent per sample →
 │      infer ONE Tier-2 dimension → synthesize the cross-session claim → harvest
 ├ [look further back] → re-sweep `ls --json --cursor <last-row.cursor>` → new cohort → picker
 └ [End] → stop; summarize where it landed
```

## Binding lazy-reference rule

Before executing any applicable branch action, MUST fetch every owning reference, unless that
exact reference from this exact skill version has already been fetched in the current
uncompacted context:

```bash
nn skills get nn-transcript --reference <name>
```

Discover the stable reference inventory and applicability when needed:

```bash
nn skills get nn-transcript --list-references
```

- `[enter a session]` → fetch reference **navigate** before descending.
- `[sweep a pattern]` → fetch reference **patterns** before sweeping.

The grammar and contract above are shared — references point back here and never restate them.

## Loop invariant (never silently converges)

- The **picker is re-presented after every step**: after the front-door draw, after a `:enter`
  thread step, after a patterns synthesis, after a `look further back` re-sweep, after a harvest.
- **End is the only exit.** The skill never stops because it judges the goal "reached."
- Every picker contains: the **discovered moves** + a **steer-with-your-own-words** affordance +
  the explicit **End** option.

## Harvest (the capture bridge)

Whenever navigation or a sweep surfaces a durable, non-derivable finding, capture it with
provenance back to the thread/session:

```bash
nn new --quick --title "<finding restated as a claim>"
```

Apply the durability test: capture only what would change behavior in a future session with no
memory of this one.

## Targeted transcript search

When the human asks where a phrase, command, error, or decision appears, use the schema-aware
spine rather than general file grep:

```bash
nn transcript search "<literal query>" <transcript-root> --json --limit <N>
nn transcript search "<literal query>" --session <session-path> [--agent <agent-id>] --json
```

Search is deterministic and returns bounded session/agent/event provenance. It does not establish
behavioral recurrence: use returned matches to select whole sessions, then enter or sweep them.
Use `--raw` only when the human explicitly needs schema-native payloads. Never use `nn grep` for
transcript JSONL: it skips oversized files and cannot preserve transcript ownership provenance.

## Co-versioned command contract

This skill is served by the same `nn` binary as the transcript spine. Do not create a companion
preflight script or probe a separate capability endpoint. Every command, flag, and JSON field named
by this skill must be covered by the repository's skill/CLI conformance test. When a source checkout
and installed binary disagree, report the ordinary `nn` version/build identity and offer an
explicit reinstall only when the checkout is known; never reinstall automatically.

## Escape hatch (unknown schema) & the lossy relation

When `scan` reports `unknown`, the spine has no recipe — compose one with DuckDB (`nn transcript
doctor` first), and **validate the join with the four assertions** before trusting output (see
reference **patterns** for the assertions; they apply to any reconstructed relation). The `tree`
relation is the overview *projection* (lossy by design); use snapshot-bound paginated
`nn transcript show <session> <agent-id> --json` (`--raw`), pass `--snapshot` on every later page,
and reconstruct every segment for exactly what happened.

## Success criteria (a reviewer can check a transcript against these)

- Navigation always begins at the front door (`ls --json`, bounded to one page = the cohort).
- Every drawn mark attaches to a real swept session; no invented ids/costs/edges.
- Front-door `↻×N` asserts only deterministic shapes; behavioral recurrence appears only as a
  proposal until the patterns branch reads the named sessions.
- Every non-End step is followed by a picker containing discovered-moves + steer + End; the only
  picker not followed by another is the one whose End was selected.
- Grammar + contract appear exactly once (here); branch references never restate them.
- Every harvested finding is a durable claim with provenance.

## Write to disk

This skill installs at `skills/nn-transcript/SKILL.md` (project-scoped) with its references under
`skills/nn-transcript/references/`. It replaces `nn-transcript-navigate` and
`nn-transcript-patterns`.
