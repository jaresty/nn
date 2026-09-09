---
name: observe
description: Bounded open-ended observation including ROOT, honest sampling and continuation.
applies_when: "Before a bare invocation, an open-ended what-is-happening request, or continuing observation."
---

# Observe recorded work

Load `nn skills get nn-transcript --reference interaction` first. Read-only observation is direct,
not a request for standing approval. Do not invent an investigative question to make it executable.

## Resolve scope, then sample

An explicit conversation or set remains that exact set. A named project is a project definition,
not the initial list of sessions. With neither an explicit nor retained scope, use the current harness transcript root
(e.g. `~/.pi/agent/sessions` for Pi or `~/.claude/projects` for Claude), state that source scope,
and sample the first conversation not known to be this observer.
This is a small first pass, not importance ranking. Load discovery before `ls` and navigate before tree.

Resolve `<transcript-root>` to that existing root, or the explicit project directory when supplied.
If the source root is unavailable or ambiguous, report it or ask; never silently scan cwd.
Native discovery is mtime-descending with path tie-breaking, not observed-work recency:

```bash
nn transcript ls <transcript-root> --conversation-kind conversation --limit 5 --json
```

Use each returned row's exact canonical path. `limit` bounds candidate output, not filesystem discovery
work. Known current-session identity is excluded from untargeted discovery; disclose that exclusion.
If identity is unavailable, say self-observation exclusion could not be established. Never guess it from
recency or name. Explicitly requested self-observation is allowed. If the candidate page contains only
excluded entries, obtain the next page through discovery's cursor contract or report that limitation.
Do not silently switch project, remove filters, or replace an explicit target with a recent one.

The candidate page is not the entire scope. For the initial brief pass, select one eligible conversation
from that page and disclose the selection. A request naming several conversations includes each named
conversation unless a concrete restriction is agreed; do not silently sample away an explicit operand.
For a large explicit set, disclose the cost and negotiate only the real bound, not permission to observe.

## Executable stream recipe

For each selected conversation, ROOT is a stream as well as a parent label.
Always include ROOT in the initial selected-conversation sample.

```bash
nn transcript tree <session> --parent ROOT --limit 2 --json
nn transcript events <session> ROOT --last 5 --format text --max-text-chars 1000
nn transcript events <session> <child-id> --last 5 --format text --max-text-chars 1000
```

Run the third command for each returned direct child using its exact ID, not a guessed label. Load
`nn skills get nn-transcript --reference events` before event retrieval. These are independent stream
windows, not a combined chronological capture. A roomless transcript still gets ROOT inspection.
An empty attributable ROOT window is a reported absence, not permission to manufacture ownership.

The tree page is canonical and bounded, not a most-important-child selector. Two direct children is
an initial budget, not one-child-per-conversation fairness or a native global selector. Descendants,
unreturned direct children, older work, and unselected conversations remain uninspected. Describe the
concrete sample and omissions briefly. If an omitted branch matters, deliberately select it from native
tree evidence and inspect it; no mandatory round-robin scheduler, exhaustive sweep, or hidden expansion.
These output limits do not bound source-prefix processing, memory, or elapsed time.

A clipped tail is orientation only. Do not claim to have read omitted payloads. For a concrete finding,
follow the exact event/window and complete transport according to events; assignment questions dispatch
to context, parent-return questions to handoffs. `review` is an optional child-only queue, never a
replacement for this ROOT-inclusive observation sample. Attention is an optional defined detector,
not the only way to notice a mismatch, untested assumption, repeated failure, or useful technique.

## Explain and continue

Summarize a few supported observations, their source/evidence level, material gaps, and a useful next
move. Recency is not importance, activity, success, productivity, or causality. Missing evidence stays
unknown. Do not manufacture a signal in a quiet sample or infer health from the absence of a detector hit.

Keep enough conversational state for continuity: the scope definition, current sample, inspected
windows, supported leads and gaps. No universal receipt layout, numerical finding quota, lead-state
machine, or serialized navigation database. Refresh may resample inside the same scope; disclose material
sample changes and do not mark departed or unrechecked leads resolved. Let a follow-up question switch
to investigate without forcing a visible mode ceremony or expanding the underlying observation scope.
Load `nn skills get nn-transcript --reference actions` when a durable learning is worth proposing.
