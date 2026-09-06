---
name: nn-transcript-navigate
description: Use when a human wants to navigate subagent execution transcripts (Claude Code, sdk-cli, pi) — see the spawn tree, find where a run went wrong, recover context, audit cost, or harvest ideas. Load with `nn skills get nn-transcript-navigate`.
when_to_use: When navigating agent/subagent transcripts to debug a run, recover context after being away, audit cost, or harvest durable ideas into notes. Load with `nn skills get nn-transcript-navigate`.
---

# nn-transcript-navigate

The **discovery layer** over the `nn transcript` spine (ADR-0042). The spine
(`scan`/`tree`/`show`/`doctor`) is deterministic and dependency-free; this skill drives it
to navigate a subagent execution landscape, discovers what is worth attending to per-thread,
harvests durable findings into notes, and reaches for DuckDB only when a schema is unknown.

Core principle: **geography is fixed and deterministic; the LLM discovers only appearance and
emphasis.** Node position (spawn hierarchy on the vertical axis, time on the horizontal, the
closed position set) comes from the spine and never moves. What this skill discovers is which
*appearance* signals to light and which *lens* to view through — never where a node sits.

## When to use

A human wants to answer one of the wedge/fast-follow jobs:
- **Debug** a bad or confusing run — reach the offending thread without reading raw JSONL.
- **Recover context** after being away — replay the causal skeleton (spawns, pivots, joins).
- **Audit** cost/effort — which subtree was expensive.
- **Harvest** — capture a durable idea from a thread into an `nn` note linked back to it.

## The descent stack

Three escalating resolutions of the same descent (from cheap/structural to expensive/inferred):

| Level | Command | Cost | What you see |
|-------|---------|------|--------------|
| front door | `nn transcript ls <dir>` | Tier 0/1, deterministic | recent sessions, most-recent-first, each with a mini subagent-tree |
| zoom out | `nn transcript tree <session>` | Tier 0/1, deterministic | agent spawn tree, lifespans, cost, subtree_cost, status |
| zoom in | `nn transcript show <session> <agent-id>` | Tier 0, deterministic | one agent's meaningful events (`--raw` for lossless full record) |
| `:enter` | *this skill runs inference* | Tier 2, per-thread | the salient discovered dimensions for that thread |

## Workflow

### 0. Front door — the recent-sessions menu (start here for interactive navigation)

This is the top of the descent: a browsable, time-ordered list of recent sessions, each
with a compact inline mini-tree of its subagent structure. Start here whenever the human
has not already named a specific session.

```bash
nn transcript ls <dir> [--limit N] [--before <RFC3339>]   # recent-first; mini-tree per row
nn transcript ls <dir> --json                              # structured rows for the picker
```

Each row is `date  schema  N agents  cost  session-id  mini-tree` where the mini-tree is a
compact one-line preview like `ROOT─a2bcf1─ae9fa7─…` (deep chain) or `ROOT─┬─A─B` (fan-out).

**Present this as an interactive menu.** Run `nn transcript ls <dir> --json`, then render the
rows with `AskUserQuestion`: one option per recent session (label = date + schema + agent
count + cost; description = the mini-tree preview), plus:
- **"Look further back"** — re-run `ls` with `--before <the oldest shown session's modified>`
  to page into older sessions, and present the next menu.
- the human's selection of a session → proceed to **step 2 (overview)** on that session's path.

**The front door is the default entry — run `ls` first, before any `tree`.** Do not jump
straight to `tree` on the current/most-recent session: presenting the menu is the point of the
navigator. The only case for skipping to step 2 is when the human *explicitly* names one
specific session to open (a path or an unambiguous id) — "the current session", "look", or a
bare directory are NOT that; for those, run `ls` and present the menu. When in doubt, `ls`
first. Default `<dir>` is the harness transcript root (e.g. `~/.claude/projects/<project-slug>/`
or, for pi, the dir of `$PI_SESSION_FILE`).

### 1. Classify (when needed)

```bash
nn transcript scan <dir>        # counts per schema: claude-code / sdk-cli / pi / unknown
```

`ls` already classifies each session; use `scan` only for a quick schema census of a dir.
If a session's schema is `unknown`, jump to **Escape hatch** below. Otherwise proceed — the
spine has a recipe for it.

### 2. Overview (zoom out)

```bash
nn transcript tree <session>            # text descent-stack lifespan tree
nn transcript tree <session> --json     # the normalized relation for programmatic use
```

Read the tree for the deterministic (Tier 0/1) signals — these are trustworthy and always
present, and they serve the debug + recover jobs directly:
- **cost / subtree_cost** — which branch spent the tokens (answers *audit*).
- **status** — active vs completed.
- **hierarchy + lifespans** — who spawned what, when (answers *recover context*).

Present the tree to the human with colored relay markers when relaying to a person (node type,
cost hot-spots). Do **not** infer Tier-2 signals (drift, groundedness, pivots) across the whole
tree here — that is deferred to `:enter` or the opt-in sweep.

### 3. Enter a thread (`:enter`) — discover per-thread dimensions

When the human focuses on one agent, this is where inference is paid — scoped to that one
thread, never globally.

```bash
nn transcript show <session> <agent-id>   # the thread's raw events
```

Then answer one question: **"what is worth attending to in THIS thread?"** Read the thread's
events and propose **2–4** salient dimensions. Draw from this starting palette, or name a novel
one the thread makes salient:
- **instruction-drift** — did it do what its spawn prompt asked?
- **context-re-derivation** — did it waste turns rediscovering already-known context?
- **groundedness** — are its claims backed by tool results, or asserted?
- **pivots** — where did it change direction?
- **friction** — retries, denials, backtracks.

Classify each proposed dimension into the grammar and **respect the hard boundary**:
- **appearance** (state on the node) and **emphasis** (which lens) — discovery MAY propose these.
- **position** (geography) and **connection** (edge types) — discovery may NEVER touch these;
  they belong to the spine. If a thread seems to need a new position dimension, that is a
  request to change the base geography — surface it explicitly to the human, never apply it
  silently.

Present the discovered dimensions as appearance annotations on the entered thread. The
navigator gets smarter as you go deeper — the opposite of a fixed dashboard.

### 4. Harvest (the capture bridge)

Whenever navigation surfaces a durable, non-derivable finding, capture it into `nn` linked back
to the thread. This is the reason the navigator lives in `nn`.

```bash
nn new --quick --title "<finding restated as a claim>"
```

Then link it to a note representing the run/thread if one exists, or record the session/agent id
in the note body so the provenance is preserved. Apply the durability test first: capture only
findings that would change behavior in a future session with no memory of this one.

## Composing the spatial-ASCII views (LLM-drawn, interpretation-bearing)

All visual views — the picker, the tree overview, and the `:enter` dimension diagram — are
**drawn by you (the LLM) from the spine's JSON**, never by a fixed renderer. The spine emits
`nn transcript ls --json` and `nn transcript tree --json`; you lay out the ASCII. Because *you*
compose it, the diagram embeds interpretation and can differ each time: emphasize the expensive
branch, color the thread that drifted, annotate a pivot, collapse the boring parts. Draw *this*
session to surface *what matters here* — not a uniform template.

**Color-relay markers** (survive markdown/relay): 🔴 tension (drift, error, contradiction) ·
🔵 lateral (provenance, process) · 🟦 structural (spawn/derivation) · 🟢 healthy/complete ·
🟡 caution (friction, stall) · 🟠 expensive. Encode: **color** = emphasis/tension, **width or
box size** = cost magnitude, **shape/label** = agent type, **branch lines** (`├─ └─ │`) =
connection.

### Picker view (from `ls --json`)

Draw each recent session as a small **multi-line** ASCII tree (not a one-line string), sized
and marked by what stands out. Example — three sessions, the middle one flagged expensive:

```
① 2026-08-09  sdk-cli  31 agents  🟠 $10.3k
   ROOT
   └─🟦 debrief-chain ×29  (deep recursive nest)

② 2026-08-10  sdk-cli  2 agents  🟠 $278k
   ROOT ═══════════════  (heavy main thread)
   └─🔵 capture: browser-demo

③ 2026-08-12  claude-code  1 agent  🟢 $162k
   ROOT  (no subagents)
```

Then offer entry via `AskUserQuestion`. The layout is your call — collapse repetitive chains
(`×29`), widen expensive nodes, mark what a human should notice first.

### Tree overview (from `tree --json`)

Draw the spawn DAG with box-drawing branches, marking each node by type and cost. Emphasize the
costly subtree; collapse uninteresting repetition. Example:

```
🟦 ROOT  [main]  $448k ████████████████
└─🔵 abc616  [nn-capture]  $1.7k ▏
```

### `:enter` dimension diagram (from `show` + your Tier-2 inference)

This is the interpretation-heavy view. After reading the thread, infer 2–4 salient dimensions,
tag each by grammar, and draw the entered node with its dimensions arranged spatially and
visually treated. Position/connection come from the spine (fixed); appearance/emphasis are your
discovery. Example for an entered thread:

```
        ┌─────────────────────────────┐
   🟦   │  a7665a  ·  nn-capture       │  ← type (shape/label)
  spawn │  purpose: browser-demo shots │
   ↑    └─────────────────────────────┘
        cost  ██▏ $24.5k                  ← appearance: width = cost
        🟢 groundedness: high (real URLs, node snippet)
        🟢 instruction-drift: none (did the screenshot task)
        🟡 friction: one interrupted tool call
```

Never move node position or invent connection types when drawing — only appearance and
emphasis are yours (see the discovery boundary above). If a thread needs a new *position*
dimension, surface it to the human, do not draw it silently.

### 5. Lenses (emphasis)

Offer the human a lens that lights a coherent subset of appearance signals for their job:
- **debug lens** — errors, friction, drift, pivots.
- **audit lens** — subtree_cost, tools, re-derivation.
- **harvest lens** — notes-touched, groundedness, pivots.
- **recover lens** — pivots, joins, lifespan ordering.

A lens changes only *what is emphasized*, never node position.

## Opt-in Tier-2 sweep

To light an inferred dimension across the *whole* overview (e.g. "show every thread that
drifted"), run inference over all threads deliberately — this is expensive and never the
default. Walk the agents from `nn transcript tree --json`, `nn transcript show` each, infer the
one requested Tier-2 dimension per thread, and annotate the overview. Tell the human this costs
one inference pass per agent before starting. Never trigger a sweep implicitly.

## Escape hatch (unknown schema)

When `scan` reports a schema as `unknown`, the spine has no recipe. Compose one with DuckDB:

1. Confirm DuckDB is available: `nn transcript doctor`. If missing, tell the human the exact
   install command (e.g. `brew install duckdb`) and stop.
2. Inspect the shape: `duckdb -c "DESCRIBE SELECT * FROM read_json_auto('<file>', maximum_object_size=20000000);"`
3. Find the spawn edge by iterating queries (as the design probe did): identify the parent→child
   join key, verify it resolves **across** agent boundaries, and — critically — **validate**
   before trusting output. An LLM-composed query fails silently-but-plausibly: a wrong join can
   resolve every row and still be wrong. Run these assertions on your reconstructed relation:
   - every non-root agent resolves to exactly one parent;
   - no agent is its own ancestor (DAG, no cycles);
   - each spawn timestamp is at or after its parent's start;
   - the resolved spawn-edge count equals the count of spawn tool-calls.
   Only emit the relation after all four pass. If any fails, the join is wrong — iterate.
4. When a new schema is cracked, capture the recipe as an `nn` note so a future session (or a
   future spine recipe) can reuse it.

## The normalized relation is lossy by design

`nn transcript tree` emits the overview *projection*:
`agent(id, parent_id, type, started, ended, cost, subtree_cost, status, result)`. It
deliberately drops schema-native fields. When the human needs exactly what happened in one
agent, use `nn transcript show <session> <agent-id>` — the lossless path.

## Success criteria

- Never render Tier-2 signals across the whole overview except via an explicit sweep.
- Never move node position or invent connection types during discovery — appearance/emphasis only.
- Every harvested finding is a durable claim captured with provenance back to its thread.
- Every escape-hatch query is validated by the four assertions before its output is trusted.
