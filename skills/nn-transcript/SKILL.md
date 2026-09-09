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

## Transcript Office — default entry experience

The default entry experience is the **Transcript Office**: an LLM-mediated navigation and analysis
surface, not an interactive CLI, terminal picker, or persistent TUI. Hide CLI mechanics unless requested.

- **Browse / orient:** when the human names an explicit project, workspace, or office, resolve that
  target before generic recent-session discovery. Otherwise load discovery and run
  `nn transcript ls <dir> --json --conversation-kind conversation --limit <N>`. Identify this conversation.
  Show the readable `label` and exact session ID; retain `opening_label`, `label_provenance`
  (`recent`, `opening`, `interpreted`, `untitled`; `recorded` requires authentication),
  `conversation_kind`, `owner_session`, and `open_window_status`. Discovery owns their full semantics;
  never present interpreted text as recorded metadata or infer open windows from transcripts.
  Retain the selected row's exact `path`; never reconstruct it from session ID, cwd, or project.
  Continue with its `--cursor`, not a derived time. Load navigate, then review: Pi defaults to **Awaiting return**.
  Hierarchy stays under More → All rooms; nested managers open sub-offices.
- **Already selected session/agent:** use its owner directly.
  Bare attention uses **attention**'s surface scope, not a background room.
- **Find an agent by launch name or description:** use `nn transcript ls` to select the parent
  session, then use `nn transcript tree <session> --description "<name>" --json`.
  This is metadata discovery: do not use `nn transcript search`, which searches event content and
  can match prompts, results, or the current conversation instead of the authoritative tree label.
- **Locate text inside transcripts:** load **search** for literal/regex lookup, multiple inputs,
  payload scope, and limits. Search locates evidence; it does not establish behavioral recurrence.

## Binding lazy dispatch

Before an applicable action, fetch its owner with the exact command below unless already loaded
from this skill version in the current uncompacted context. Load only applicable references;
branch references may dispatch to a command owner. Discover applicability with
`nn skills get nn-transcript --list-references`.

| Need | Load | Then use |
|---|---|---|
| Cohort / listing metadata | `nn skills get nn-transcript --reference discovery` | `nn transcript ls` |
| Enter an office / hallway | `nn skills get nn-transcript --reference navigate` | Awaiting return; tree for All rooms |
| Unclosed Work Desk / review patterns / correction drafts | `nn skills get nn-transcript --reference review` | `nn transcript review` (deterministic Pi evidence) |
| Open / Refresh; attention signals | `nn skills get nn-transcript --reference attention` | Standing opt-in checks; `nn transcript attention` |
| Assignment versus recent work | `nn skills get nn-transcript --reference context` | `nn transcript context` (recorded launches + bounded tail) |
| Enter one room | `nn skills get nn-transcript --reference rooms` | Bounded `events --last 5` Situation Board |
| Scan or rearrange any view | `nn skills get nn-transcript --reference lenses` | Preset, blank, inferred, or user-defined lens |
| Find transcript text | `nn skills get nn-transcript --reference search` | `nn transcript search` (literal/regex; files/directories) |
| Cross-session patterns | `nn skills get nn-transcript --reference patterns` | Session sampling and evidence-bounded investigation |
| Unknown schema | `nn skills get nn-transcript --reference recovery` | `nn transcript doctor`; validated escape hatch |
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
- Producer status is not task success and is not proof of current activity. Cumulative usage is not latest
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

For a requested or delegated spatial lens, show supported findings in a meaningful spatial diagram,
not a status list dressed with icons. Plain room entry is neutral and readable under **rooms**.

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
assert only deterministic recorded shapes—not behavioral recurrence. Bounded candidate patterns require inspected evidence and explicit limits;
whole-session or cross-session behavioral conclusions require the **patterns** workflow. Do not infer edges from previews.
If a view invents identity, topology, or measurement, discard it and redraw from authoritative output.

## Navigation loop and capture

For human-driven navigation, **every navigable surface** uses a visible action rail with up to three high-value actions exposed **directly**, followed by **More…** for **uncommon** controls; **Back**
and **End** remain visible outside the overflow. An ordinary action requires at most one intermediate chooser. `More…` may expose context-appropriate **Scan this level…**, **Change lens…**, hierarchy,
refresh, and advanced operations that are not already direct shortcuts. An **ellipsis** means the
operation needs more input and opens a chooser; a label without one executes immediately. Keep
**entity picker labels** exact and separate from view controls.

Use these level-specific defaults when no stronger evidence-based suggestion is available:

- **Conversation lobby** — `What stands out?`, `Scan…`, `Open conversation…`, `More…`.
- **Office or team** — `Attention`, `Scan…`, `Open room…`, `More…`.
- **Room** — `Orient me`, `Choose lens…`, `Inspect event…`, `More…`.
- **Selected event** — `Explain`, `Compare…`, `Inspect payload`, `More…`.

Before navigating, load `nn skills get nn-transcript --reference interaction` for targeting, view state,
and inspection authorization. For suggestions/capture load `nn skills get nn-transcript --reference actions`;
**Capture…** is always available under More or via “capture that”, but writes require proposal approval.

Compact default surfaces show **at most three standout** entities and an **explicit omitted count**;
full retained population counts remain visible. Also preserve a steer-in-your-own-words affordance.
Do not bury scans as optional documentation or silently terminate because the goal seems reached.
Office views
may apply higher-level scans over a bounded authenticated population. Entering a room switches from
the office metaphor to a rearrangeable Situation Board; named lenses are presets and user-defined
questions, axes, groups, filters, comparisons, and metaphors are first-class. Drill-down preserves the
question; Back restores the same Office Scan or hallway. A refresh preserves human selection and lens
but reacquires mutable evidence. One-shot command questions need not enter this loop.

Capture only durable, non-derivable findings, with session/thread provenance. A file location,
current status, or reproducible lookup is not a durable finding. Use the normal nn capture discipline.

## Unknown schemas and versioning

When scan reports unknown, load **recovery** for diagnosis and reconstructed-join validation.
DuckDB is an escape hatch, not the normal workflow; patterns is not a prerequisite.
The tree is a lossy overview; retrieve complete relevant events before interpreting behavior.

Skills and CLI ship together. Do not probe a separate capability endpoint or create preflight scripts.
If checkout and installed binary disagree, report the normal version/build identity and offer an
explicit reinstall; never silently install. Contract tests cover both core dispatch and served owners.
