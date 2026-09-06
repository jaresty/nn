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

**Core principle (the `:enter` boundary, applied everywhere): geography is fixed and
deterministic; the LLM discovers only appearance and emphasis.** Session identity, cost, agent
count, and tree shape come from the spine unchanged. What the LLM discovers is which standouts
to light, how to emphasize them, and which lens to view through — never the numbers or positions.

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

Color-relay markers survive markdown/relay. Reuse the `:enter` grammar, plus two cross-session marks:

| Channel | Encodes | Values | Source |
|---|---|---|---|
| color marker | emphasis / tension | 🔴 tension · 🟠 expensive · 🟡 caution · 🟢 healthy · 🔵 lateral · 🟦 structural | **discovered** |
| width / box | observed token magnitude | provisional bar ∝ `total_cost`; exact comparison requires authority from `tree` | **fixed** |
| shape / label | node type | session/schema from `ls`; agent/type-name from `tree` | **fixed** |
| branch lines `├─ └─ │` | connection | spawn/tree edges only after `tree --json` | **fixed** |
| `◈` | outlier-vs-cohort | departs from *this* swept cohort | **discovered** |
| `↻×N` | recurring-across-N | a shape in N named sessions | **discovered, deterministic only** |

**`↻×N` tightening:** at the front door, `↻×N` may assert only repeated schema or `agent_count`
values and must name the N session ids. Agent-type frequency and depth require `tree --json` for
each named session; exact topology requires that same relation. `tree_preview` is a lossy preview with shortened IDs and a
node cap, never an edge relation. Any **behavioral** recurrence (drift, re-derivation, groundedness)
is a **proposal to sweep** (`◈ … sweep to check?`), never a stated claim — confirming it requires
the patterns branch.

`total_cost`, `cost`, and `subtree_cost` are token counts, not currency. `ls` provides only observed
`total_cost`, without completeness authority. Label it provisional; do not call a session cheaper
or fully accounted from that number alone. For cost comparisons, fetch `tree --json` and honor
`cost_status` / `subtree_cost_status`: unavailable is unknown, partial is a lower bound, complete
is measured (including measured zero).

## Discovery contract

- **Tier 1 — MUST copy through** from `ls --json`: `session`, `schema`, observed `total_cost`,
  `agent_count`, and optionally the literal `tree_preview`. `path` and `modified` identify the
  source; `cursor` is transport state. Every drawn mark must attach to a real listed session.
  Never expand the preview into inferred topology; retrieve `tree --json` for real edges.
- **Tier 2 — MAY discover**: which standouts, emphasis/color, `◈` (relative to current cohort),
  `↻×N` (deterministic shapes only, with named ids).
- **Tier 3 — MUST NOT**: invent a session/cost or an edge absent from `tree`; assert any Tier-2
  behavioral interpretation (drift, groundedness, "failed") from the front door alone; move
  geography. Behavioral claims require escalating to a real read (`tree` → `show`).
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
