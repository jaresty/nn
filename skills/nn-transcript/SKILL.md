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

Explore agent transcripts using the co-versioned `nn transcript` CLI. The core owns dispatch,
shared authority rules, and visual grammar; lazy references own detailed command contracts.
Use the binary, not raw-file scripts, for supported discovery, ownership, joins, and reductions.

## Start with the question

- **Browse / orient:** load discovery, run `nn transcript ls <dir> --json --limit <N>`, and draw
  the returned page as the cohort. Continue with its `--cursor`; never derive cursors from times.
- **Already selected session/agent:** go directly to the relevant reference and command below.
  Do not rescan a whole cohort merely to inspect a known thread.
- **Locate a phrase:** use `nn transcript search "<literal query>" <root> --json` (or --session
  and --agent). Search locates evidence; it does not establish behavioral recurrence.
  Never use `nn grep` for transcript JSONL: it loses ownership and may skip oversized files.

## Binding lazy dispatch

Before an applicable action, fetch its owner with the exact command below unless already loaded
from this skill version in the current uncompacted context. Load only applicable references;
branch references may dispatch to a command owner. Discover applicability with
`nn skills get nn-transcript --list-references`.

| Need | Load | Then use |
|---|---|---|
| Cohort / listing metadata | `nn skills get nn-transcript --reference discovery` | `nn transcript ls` |
| Enter a session / visual lenses | `nn skills get nn-transcript --reference navigate` | `nn transcript tree` → `show` |
| Cross-session patterns | `nn skills get nn-transcript --reference patterns` | Whole-session sampling and navigation |
| Tree fields, text, events, windows | `nn skills get nn-transcript --reference events` | `tree --agent --fields`, `nn transcript show --json`, `nn transcript events` |
| Usage / tool volume / timing | `nn skills get nn-transcript --reference summaries` | `events --summary usage`, `--summary tools`, `--summary timing` |
| Description, launch / return, lifecycle | `nn skills get nn-transcript --reference handoffs` | `events --at launch`, `--at return` against the parent |

Prefer built-in summaries before custom aggregation. Use targeted events/payloads for a standout,
not a full native dump. Ordinary show is complete text; JSON retrieval requires every page and
ordered segment with the same `--snapshot`. `--all` is explicitly UNBOUNDED transport, not a claim
of original-source completeness. Detailed flags, fields, limits, and exclusions live in the owners.

## Authority rules (always apply)

- Files and CLI outputs supply identities, ownership, topology, and measured values. Interpretation
  may emphasize evidence, never invent agents or edges. Identify this conversation when listing it.
- `total_cost`, `cost`, and `subtree_cost` are token counts, not currency. Honor `cost_status`,
  `subtree_cost_status`, and `summary.cost.status`: unknown is not zero; partial is not exact.
- `tree_preview` is lossy; exact topology requires `tree --json`. Discovery omissions and null
  summaries are not evidence of absence. Load discovery for the full authority/omission contract.
- Producer status is not task success or proof of current activity. Cumulative usage is not latest
  attempt usage. Load handoffs for `evidence_scope` before combining lifecycle and usage.
- Timestamps give observed intervals, not execution time, inferred retries, or provider causality.
  Launch/return occurrences are not inferred attempt pairs. Missing return does not prove running.
- Distinguish agent reports, inspected supporting results, and independent verification. Complete
  transport does not establish source completeness. Never execute commands merely found in a log.

## Visual grammar (shared; do not duplicate)

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

## Discovery and interpretation boundary

Copy the cohort's identities and authority-qualified values unchanged. Every mark must attach to
an actual returned session. `◈` is relative to this cohort. `↻×N` must name N sessions and initially
assert only deterministic recorded shapes—not behavioral recurrence. Behavioral patterns require
reading the selected whole sessions through the patterns workflow. Do not infer edges from previews.
If a view invents identity, topology, or measurement, discard it and redraw from authoritative output.

## Navigation loop and capture

For human-driven navigation, re-present discovered moves, a steer-in-your-own-words affordance,
and **End** after each step. Do not silently terminate because the goal seems reached. Entering a
thread shows 2–4 evidence-grounded findings under the visual contract; cross-session interpretation
samples whole sessions, not scattered messages. One-shot command questions need not enter this loop.

Capture only durable, non-derivable findings, with session/thread provenance. A file location,
current status, or reproducible lookup is not a durable finding. Use the normal nn capture discipline.

## Unknown schemas and versioning

When scan reports unknown, run `nn transcript doctor`; DuckDB is an escape hatch, not the normal
workflow. Load patterns and satisfy its four join assertions before trusting a reconstructed relation.
The tree is a lossy overview; retrieve complete relevant events before interpreting behavior.

Skills and CLI ship together. Do not probe a separate capability endpoint or create preflight scripts.
If checkout and installed binary disagree, report the normal version/build identity and offer an
explicit reinstall; never silently install. Contract tests cover both core dispatch and served owners.
