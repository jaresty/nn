---
name: nn-transcript-patterns
description: Use when looking for recurring patterns ACROSS many subagent transcripts — where runs consistently drift, re-derive context, burn cache, stall, or fail — rather than navigating one session. Load with `nn skills get nn-transcript-patterns`.
when_to_use: When the question is about patterns across many sessions (recurring cost shapes, drift, re-derivation, failure modes), not navigation within one session. Load with `nn skills get nn-transcript-patterns`.
---

# nn-transcript-patterns

The **cross-session** companion to `nn-transcript-navigate`. Where the navigator descends into
*one* session (pick → tree → enter), this skill sweeps *many* sessions to surface recurring
shapes: which patterns repeat, which sessions are outliers, where cost/drift/stalls cluster.

It is the inverse of targeted navigation — statistical coverage over a corpus of runs, closer
in spirit to `nn shuf` than to `nn grep`.

## When to use

The question spans sessions, not one run:
- "Across my recent runs, where do subagents consistently drift from their prompt?"
- "Which runs burn the most *real* cost (output + cache creation, not cache reads)?"
- "How deep do my spawn trees usually get? Which are outliers?"
- "What subagent types recur, and which ones tend to stall or fail?"

If the human wants to navigate *one* session, use `nn-transcript-navigate` instead.

## The unit of sampling is the SESSION, not the message

Sample whole sessions, never messages within a session — a session is the coherent unit of
interpretation, and fragmenting it destroys the Tier-2 signal. When the corpus is large,
sample N sessions (spread across the time range, plus the cost outliers) and reason about all
tiers of each sampled session, rather than skimming a fragment of every session.

## Workflow

### 1. Sweep the corpus (deterministic, cheap)

```bash
nn transcript ls <dir> --json                 # all sessions with typed cost + tree shape
nn transcript ls <dir> --json --before <ts>   # page further back for a wider window
```

Aggregate the deterministic (Tier 0/1) signals across all rows — no inference yet:
- **cost distribution** — and split by token type. `cache_read_tokens` are cheap; `output_tokens`
  and `cache_creation_tokens` are the expensive real cost. Rank sessions by *effective* cost
  (weight output/cache-creation heavily, cache-read lightly), not by the flat total — a session
  that looks huge may be mostly cache reads.
- **depth / fan-out** — recursion depth and agent count per session; flag outliers.
- **agent-type frequency** — which subagent types recur across sessions.
- **schema mix** — claude-code vs sdk-cli vs pi.

Report the deterministic patterns first: the distribution, the outliers, the recurring types.

### 2. Select a sample of sessions to interpret

From the swept corpus, pick a sample that earns Tier-2 inference:
- the cost outliers (top effective-cost sessions),
- a spread across the time range (recent + older),
- any structural outliers (unusually deep, unusually wide, many failures).

State the sample and why each session is in it. Keep the sample small enough to reason about
every tier of each — a handful of whole sessions, not a fragment of every session.

### 3. Interpret each sampled session across all tiers

For each sampled session, drive `nn-transcript-navigate`'s descent (tree → enter the notable
threads) and infer the Tier-2 dimensions per thread: instruction-drift, context-re-derivation,
groundedness, pivots, friction. Because you sampled whole sessions, the interpretation is
coherent — you can say "in this run the debrief agent re-derived the session boundary from
scratch," not just "some message looked odd."

### 4. Surface the recurring patterns

Synthesize across the sample. A pattern is a claim that holds across *multiple* sampled
sessions, e.g.:
- "session-debrief subagents re-derive the daily-note boundary every run — a caching gap."
- "the deepest recursive nests are all debrief chains; other work stays shallow."
- "cache-creation dominates cost on long sessions; cache-read dominates on resumed ones."

Draw the cross-session summary as spatial ASCII when it helps (a small distribution chart, a
ranked outlier list with cost-type bars) — LLM-composed, same discipline as the navigator's
visual layer.

### 5. Harvest patterns into notes

A cross-session pattern is exactly the kind of durable, non-derivable finding worth capturing.
For each real pattern, `nn new` a note stating the pattern as a claim, and link it to the
sessions or the design notes it bears on. Apply the durability test: capture patterns that
would inform future runs, not one-off observations.

## Success criteria

- Sample whole sessions, never message fragments — the session is the unit of interpretation.
- Rank by effective (type-weighted) cost, never the flat total that cache-reads inflate.
- Every reported pattern holds across multiple sampled sessions, not one.
- Recurring patterns are harvested as durable notes with provenance.
