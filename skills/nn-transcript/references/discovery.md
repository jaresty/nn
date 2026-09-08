---
name: discovery
applies_when: "Before listing or interpreting discovery cohorts, cursors, cost/topology summaries, and type-frequency omissions."
---

# nn-transcript / discovery

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
  authoritative spawn geography. Semantic finding layouts are allowed only after entering a thread,
  under the declared-axis contract. Behavioral claims require escalating to a real read (`tree` → `show`).
- **On violation**: a drawn id not in the sweep, or a width misrepresenting cost → void the
  draw, re-render from `ls`. `↻×N` without N named ids → downgrade to a single observation. A
  front-door behavioral claim with no session read → restate as a proposal.
