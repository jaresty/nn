---
name: patterns
applies_when: "When sweeping a pattern across many sessions — aggregate the cohort's deterministic signals, sample whole sessions, drive the navigate descent per sample, infer one Tier-2 dimension, synthesize a cross-session claim, and harvest it."
---

# nn-transcript / patterns — across sessions

Owning reference for `[sweep a pattern]`. Fetch before sweeping. Entered with the `cohort` (and
optionally a `proposed pattern`) in carried state. **Patterns = navigate applied across the
cohort** — it is the inverse of targeted navigation, statistical coverage over a corpus of runs,
closer in spirit to `nn shuf` than to `nn grep`. The visual grammar and discovery contract live
in the core (`nn skills get nn-transcript`) — this reference does not restate them.

## Targeted matches select sessions; they do not prove patterns

Use `nn transcript search <query> <root> --json` to locate attributable occurrences without the
size ceiling and provenance loss of `nn grep`. A match is only a candidate-session locator. Add its
whole session to the sample and run the navigation descent before making a behavioral claim; never
infer recurrence by counting matching messages alone.

## The unit of sampling is the SESSION, not the message

Sample whole sessions, never messages within a session — a session is the coherent unit of
interpretation, and fragmenting it destroys the Tier-2 signal. When the corpus is large, sample
N whole sessions (spread across the time range, plus the cost outliers) and reason about all
tiers of each, rather than skimming a fragment of every session.

## Steps

1. **Aggregate only deterministic fields the cohort actually carries (cheap, no inference).**
   From the swept `ls --json` (the cohort is the front door's one page; page further back with
   `ls --json --cursor <last-row.cursor>` with the same directory/filter for a wider window),
   use session id, schema, `agent_count`,
   `total_cost`, and `tree_preview`. Do not claim typed cost, exact depth, or agent-type frequency
   from `ls`; those require `tree --json` for the selected sessions. Costs are token counts:
   treat `total_cost` as a provisional discovery signal, not economic cost or complete accounting.
   Fetch tree authority before comparing costs; do not rank unavailable child costs as zero or
   partial subtree costs as exact.

   Report the supported deterministic patterns first, with their field provenance.

2. **Select a small sample that earns Tier-2** — observed-token candidates, a spread across the
   time range (recent + older), and structural outliers established from `tree --json` (deep/wide).
   Failure-based selection requires event evidence from `show`, not preview inference.
   **State the sample and why each session is in it.** If a `proposed pattern` was
   carried in from the front door, its named session ids are automatically in the sample. Keep it
   a handful of *whole* sessions.

3. **Interpret each sampled session across all tiers.** For each, drive the *navigate* descent
   (`tree` → enter the notable threads; see reference **navigate**) and infer the **one**
   requested Tier-2 dimension per thread (instruction-drift, context-re-derivation, groundedness,
   pivots, friction). Because you sampled whole sessions, the interpretation is coherent — you can
   say "in this run the debrief agent re-derived the session boundary from scratch," not "some
   message looked odd."

4. **Surface only cross-session patterns.** A pattern is a claim that holds across **multiple**
   sampled sessions, e.g. "session-debrief subagents re-derive the daily-note boundary every run
   — a caching gap." Draw the cross-session summary as spatial ASCII when it helps (a small
   distribution chart, a ranked outlier list with cost-type bars) — same visual discipline as the
   core grammar.

5. **Harvest patterns into notes.** For each real pattern, `nn new` a note stating the pattern as
   a claim, linked to the sessions or design notes it bears on. Durability test: capture patterns
   that would inform future runs, not one-off observations.

6. **Return to the core picker** (loop invariant — never terminate the branch on its own).

## Escape-hatch join validation (four assertions)

When a sampled session has an `unknown` schema and you reconstruct its spawn relation with
DuckDB, an LLM-composed query fails silently-but-plausibly — a wrong join can resolve every row
and still be wrong. Before trusting the reconstructed relation, all four must pass:
- every non-root agent resolves to exactly one parent;
- no agent is its own ancestor (DAG, no cycles);
- each spawn timestamp is at or after its parent's start;
- the resolved spawn-edge count equals the count of spawn tool-calls.

Only emit the relation after all four pass. If any fails, the join is wrong — iterate. When a new
schema is cracked, capture the recipe as an `nn` note for reuse.

## Worked example

Cohort of 8 sdk-cli sessions. After fetching `tree --json` for each session, three have high
output-plus-cache-creation token counts with complete accounting and debrief-chain topology;
a `proposed pattern` `↻×3 re-derive daily-note boundary (b71c, a3f2, 9f2a)` was
carried in from the front door.

```
token counts (output + cache-creation; complete accounting):
  b71c  ████████  ◈ outlier
  a3f2  ██████    ◈ outlier
  9f2a  █████     ◈ outlier
  e10c  ▏          (recent, shallow)
  7d21  ▏          (recent, shallow)
sample = b71c, a3f2, 9f2a (the proposed ids) + e10c, 7d21 (contrast)
```

Per-session navigate descent confirms the debrief agent re-derives the daily-note boundary in all
3 outliers, and *not* in the 2 shallow contrasts. Pattern earned across multiple sessions →
harvest "session-debrief subagents re-derive the daily-note boundary every run — a caching gap",
linked to the 3 sessions. **Return to the picker.**
