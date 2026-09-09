---
name: discovery
applies_when: "Before listing or interpreting discovery cohorts, cursors, cost/topology summaries, and type-frequency omissions."
---

# nn-transcript / discovery

Load `nn skills get nn-transcript --reference interaction` for selected-target Open/Open… behavior,
visible breadcrumbs, action binding, and exact retained-view Back. Preserve a visibly selected row;
Open executes it without re-listing, while Open… chooses a different row. No selection or ambiguous
target means one focused clarification. Echo numeric selections before retrieval, without another prompt.

## Optional browsing

Bare invocation dispatches to `nn skills get nn-transcript --reference observe` for bounded observation
before choices. Explicit browsing may present a conversation picker with exact returned labels,
project/schema and session ID as secondary text, and More conversations… for native cursor pagination.
An empty page does not prove the inventory empty. Stop on dismissal. Selecting a conversation opens
it directly; explicit questions bypass browsing. Ask only for genuine ambiguity or concrete restrictions.

Optional attention dispatches to `nn skills get nn-transcript --reference attention`. View entry never
prompts for standing approval. Preserve the named cohort and filters rather than substituting a
background room. For a question, discovery is a means of locating evidence, not a required first stop.

## Find an agent by launch name

Use `nn transcript ls <root> --json` to select the parent session, then
`nn transcript tree <session> --description "<exact launch name>" --json`. This filters authenticated
launch metadata after complete tree validation and rollup, returns every exact case-sensitive match in
canonical tree order, and returns `[]` when there are none; descriptions are not unique identities.
It composes with `--fields` and rejects combination with `--agent`. Keep the selected agent ID for
subsequent `show`, summary, or handoff commands. If the likely parent session is already known, go
directly to its tree; do not rescan unrelated sessions. This is launch-metadata lookup:
do not use `nn transcript search`, because
content search can match assignments, tool results, or the current conversation rather than the
authoritative launch name. Load **handoffs** before interpreting the description or retrieving
launch/return records.

## Native discovery

### Target-first routing

When the human supplies a **target hint** naming a project, workspace, or office, resolve its Pi
session-directory scope and list that targeted scope first. Do not replace it with a generic recent
cohort merely because another project is current. If zero scopes match, report that; if multiple
scopes are plausible, present the ambiguous candidates rather than guessing. Generic recent-session
discovery is the fallback only when no target was supplied or the human explicitly requests recency.

Retain the complete selected `ls` row as conversational state, especially `path`, `session`, `schema`,
and cursor/scope provenance. The exact `path` is the downstream command argument. Pass it
byte-for-byte to `tree`, `events`, and `show`; never rebuild it from `session`, cwd, current project,
or a guessed directory slug.

Sweep the selected cohort and draw what stands out:

```bash
nn transcript ls <dir> --json --conversation-kind conversation --limit <N>   # bounded candidate page, not the entire scope
```

Default `<dir>` is the harness transcript root (e.g. `~/.claude/projects/<project-slug>/`).
Use native `--conversation-kind conversation` for a conversation lobby or
`--conversation-kind sidechain` for a sidechain-only cohort; do not pipe through `jq` merely to remove
the other kind. The filter is applied before pagination, and its value is bound into the cursor
snapshot. Draw an **LLM-composed** standout view from the JSON (never a fixed template), then present
the picker. Each picker option label must exactly equal the corresponding displayed lobby `label`;
put provenance and exact session ID in secondary text rather than replacing the label with generic
`Enter …` wording. Every row carries the original first-line ROOT user `opening_label` plus a readable primary
`label` selected from the latest non-acknowledgement ROOT user message. Acknowledgement-only messages
such as `yes`, `ok`, `continue`, and `let's do it` are skipped. Exact `label_provenance` is `recent`
for an unmodified later message, `opening` when the opening remains selected, `interpreted` for a
bounded shortening, or `untitled` when no usable user message exists (`recorded` remains reserved for
future authenticated metadata). Display the label as primary identity and the exact session ID as
secondary identity; never present interpreted text as recorded metadata. Keep the lobby compact: show
at most three standout conversations plus the explicit omitted count, then contextual actions or fallback shortcuts
`What stands out?`, `Scan…`, `Open conversation…`, and `More…`; keep `Back` and `End` visible.

`conversation_kind` classifies each retained row as `conversation` or `sidechain`; Pi agent execution
directories remain visible but are explicitly marked `sidechain`, while known nested `subagents`
transcript files remain excluded and are reached through the selected conversation's tree.
`owner_session` is null unless cross-session ownership is authenticated; never fill it from recency,
path resemblance, or a label match. `open_window_status` is `unavailable`
because retained transcripts do not establish which Pi windows are open. A host-authenticated source
is required before that value may change.

Labels are mutable transcript-derived presentation: retain the complete selected row across Back and
other non-refresh navigation. Reacquire labels only on explicit discovery refresh. The displayed sample is
**replaced, not accumulated** on refresh; its scope definition is retained.
For the next page, pass the last returned row's `cursor` as `--cursor <cursor>` with the same
directory and any original `--before` filter. Stop on `[]`. Do not derive a cursor from `modified`:
`--before` is a strict time filter and cannot continue exact timestamp ties. A stale or mismatched
cursor requires restarting discovery; it binds the directory/filter and path/mtime/size inventory,
not transcript contents or derived metrics.

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

## Discovery contract

- **Tier 1 — MUST copy through** from `ls --json`: `session`, `schema`, observed `total_cost`,
  `agent_count`, summary values with their authority/omission indicators, and optionally the literal
  `tree_preview`. `path` and `modified` identify the source; `cursor` is transport state. Every drawn mark must attach to a real listed session.
  Never expand the preview into inferred topology; retrieve `tree --json` for real edges.
- **Tier 2 — MAY discover**: which standouts, emphasis/color, `◈` (relative to current cohort),
  `↻×N` (deterministic shapes only, with named ids).
- **Tier 3 — MUST NOT**: invent a session/cost or an edge absent from `tree`; assert any Tier-2
  behavioral interpretation (drift, groundedness, "failed") from the front door alone; move
  authoritative spawn geography. Optional semantic finding layouts use the declared-axis contract. Behavioral claims require escalating to a real read (`tree` → `show`).
- **On violation**: a drawn id not in the sweep, or a width misrepresenting cost → void the
  draw, re-render from `ls`. `↻×N` without N named ids → downgrade to a single observation. A
  front-door behavioral claim with no session read → restate as a proposal.
