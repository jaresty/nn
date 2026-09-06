# ADR-0042: Subagent-transcript navigator — pure-Go spine; DuckDB only for the unknown-schema escape hatch

**Status:** Proposed
**Date:** 2026-09-06
**Revised:** 2026-09-06 (shoshin review — moved DuckDB out of the deterministic spine)
**Revised:** 2026-09-06 (shoshin review — ALL spatial-ASCII views are LLM-composed by the skill from the spine's JSON, so each embeds interpretation; the spine does no visual rendering; not Mermaid/HTML, not flat text, not a fixed Go renderer)

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

## Consequences

- `nn` stays pure-Go and small; DuckDB becomes a named, doctor-checked prerequisite.
- New harnesses are onboarded by adding an SQL recipe and a sniff signature, or handled
  ad hoc by the skill layer's escape hatch — no new Go code per schema.
- The normalized relation decouples the overview/lenses from transcript formats.
- Validation makes LLM-composed extraction safe to trust, at the cost of requiring every
  recipe to satisfy the assertion suite.
- Live supervision (tailing an in-flight transcript) is explicitly out of scope for this
  decision; it is a different, incremental-ingest data story.

## Open burden

Placing this in `nn` (rather than a standalone tool) is justified only if the **harvest
bridge is load-bearing** — i.e. navigating a transcript routinely produces durable `nn`
notes linked back to threads. If harvest proves decorative in practice, the in-nn
placement should be revisited: the transcript navigator would then be a separate tool that
merely *targets* nn for capture, not a subcommand of it. This burden is carried, not
discharged, by this ADR.
