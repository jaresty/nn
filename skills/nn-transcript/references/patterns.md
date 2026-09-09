---
name: patterns
applies_when: "When sweeping a pattern across many sessions — aggregate the cohort's deterministic signals, sample whole sessions, drive the navigate descent per sample, infer one Tier-2 dimension, synthesize a cross-session claim, and harvest it."
---

# nn-transcript / patterns — across sessions

Owning reference for `[sweep a pattern]`. Fetch before sweeping. Entered with the `cohort` (and
optionally a `proposed pattern`) in carried state. **Patterns = navigate applied across the
cohort** — it is the inverse of targeted navigation, statistical coverage over a corpus of runs,
closer in spirit to `nn shuf` than to `nn grep`. The visual grammar lives
in the core (`nn skills get nn-transcript`); **discovery** owns listing metadata.

## Command owners

Before measurement commands, load **discovery** for cohort metadata, **summaries** for usage/tool/timing
reductions, and **events** for payloads or time/error windows. Use the core's binding lazy dispatch.
Recorded intervals are not execution time or inferred retries; a synchronized gap is a locator,
not a causal explanation. Keep the whole-session sampling discipline below.

## Targeted matches select sessions; they do not prove patterns

Load `nn skills get nn-transcript --reference search` to locate attributable occurrences. That owner
covers literal/regex matching, inputs, payload scope, provenance, limits, and errors; a simple lookup
does not require this workflow. Here, a match is only a candidate-session locator. Add its whole
session to the sample and run the navigation descent before making a behavioral claim; never infer
recurrence by counting matching messages alone.

## The unit of sampling is the SESSION, not the message

Sample whole sessions, never messages within a session — a session is the coherent unit of
interpretation, and fragmenting it destroys the Tier-2 signal. When the corpus is large, sample
N whole sessions (spread across the time range, plus the cost outliers) and reason about all
tiers of each, rather than skimming a fragment of every session.

**Sampling is not retrieval coverage.** Selecting a session does not mean every event was inspected,
nor does it authorize an unbounded dump. Keep the session as the sampling unit, then retrieve the
assignment and work evidence needed for the declared question under the **interaction** envelope.
Complete every required page/segment before interpreting an event. Name inspected windows and leave
uninspected regions explicit; a bounded finding stays bounded. Whole-session claims require evidence
covering that claim across the session; partial inspection cannot establish absence elsewhere.
Cross-session behavioral claims still require the navigation descent and supporting evidence in
multiple sampled sessions. If the authorized evidence is insufficient, narrow the claim or ask for
more scope; do not equate selecting a session with understanding it.

## Steps

1. **Aggregate only deterministic fields the cohort actually carries (cheap, no inference).**
   From the swept `ls --json` (the cohort is the front door's one page; page further back with
   `ls --json --cursor <last-row.cursor>` with the same directory/filter for a wider window),
   use session id, schema, `agent_count`, and the bounded summaries described in **discovery**.
   `summary.cost` provides typed token counts and authority; `summary.topology` provides complete
   depth/width aggregates when its status allows them. Returned type frequencies are exact, but
   `types_truncated` and omission counts prohibit treating missing entries as absent.
   Null summaries are unavailable, not empty sessions. Do not rank unavailable costs as zero or
   partial totals as exact. Fetch `tree --json` only for actual edges, individual agents, subtree
   attribution, or uncapped type identities—not merely to recompute available summaries.

   Report the supported deterministic patterns first, with their field provenance.

For token totals and context-growth comparisons within selected agents, use
`nn transcript events <session> <agent-id> --summary usage --bucket-size 10` after loading **summaries**. Preserve missing-count authority and known-context denominators; do not equate usage records
with independently verified API calls or assume buckets are task phases.

For tool-volume comparisons within selected agents, use
`nn transcript events <session> <agent-id> --summary tools --limit 8 --group-by tool` after loading **summaries**; preserve unknown sizes, ambiguous joins, and explicit preview/result omissions.

For custom per-record usage or tool-event comparisons within selected agents, use
`nn transcript events <session> <agent-id> --select identity,message,usage,tools --json` after
loading **events**. Complete every snapshot-bound page and reconstruct oversized events before
aggregation. Count usage-bearing message events once, not extracted tool events. These are deterministic
measurements; repeated calls and large outputs are candidates for interpretation, not proof of waste.

2. **Select a small sample that earns Tier-2** — observed-token candidates, a spread across the
   time range (recent + older), and structural outliers established from complete `summary.topology`
   metrics (deep/wide). Fetch `tree --json` when the proposed distinction requires actual edges.
   Failure-based selection requires event evidence from `show`, not preview inference.
   **State the sample and why each session is in it.** If a `proposed pattern` was
   carried in from the front door, its named session ids are automatically in the sample. Keep it
   a handful of *whole* sessions.

3. **Interpret each sampled session across all tiers.** For each, drive the *navigate* descent
   (`tree` → enter the notable threads; see reference **navigate**) and infer the **one**
   requested Tier-2 dimension per thread (instruction-drift, context-re-derivation, groundedness,
   pivots, friction). Session sampling preserves context, but evidence must support the interpretation:
   say "in this inspected debrief thread the agent re-derived the session boundary from scratch"
   only after reading the relevant assignment and work, not merely because "some message looked odd."

4. **Surface only cross-session patterns.** A pattern is a claim that holds across **multiple**
   sampled sessions, e.g. "session-debrief subagents re-derive the daily-note boundary in the inspected
   debrief threads — a possible caching gap." Draw the cross-session summary as spatial ASCII when it helps (a small
   distribution chart, a ranked outlier list with cost-type bars) — same visual discipline as the
   core grammar.

5. **Propose harvesting patterns into notes.** Load `nn skills get nn-transcript --reference actions`
   for the capture proposal and approval contract. State the supported claim and proposed provenance
   links; apply the normal nn durability discipline. Finishing an investigation is not capture approval.

6. **Return to the core picker** (loop invariant — never terminate the branch on its own).

## Unknown schemas

Load `nn skills get nn-transcript --reference recovery` for native diagnosis and the four assertions
required before trusting an escape-hatch reconstruction. Recovery owns those checks for both single
sessions and cross-session samples; this workflow does not define a second validation contract.

## Worked example

Cohort of 8 sdk-cli sessions. The discovery summaries identify three with high
output-plus-cache-creation token counts and complete accounting. Fetching `tree --json` for those
three establishes debrief-chain topology;
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
propose capturing "session-debrief subagents re-derive the daily-note boundary in the three inspected outlier threads — a caching-gap candidate",
linked to the 3 sessions. **Return to the picker.**
