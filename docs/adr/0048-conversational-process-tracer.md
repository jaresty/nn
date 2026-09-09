# ADR-0048: Conversational process tracing and learning capture

## Status

Proposed. This records the outcome of conceptual steering, not an implemented or validated
replacement skill. Implementation remains paused; drafting this ADR does not resume it.

Related:
- [ADR-0042: Transcript navigator](0042-transcript-navigator-duckdb-spine.md)
- [ADR-0043: Intent-preserving transcript discovery](0043-intent-preserving-transcript-discovery.md)
- [ADR-0044: Bundled attention policies](0044-bundled-transcript-attention-policies.md)
- [ADR-0045: Attention discovery](0045-attention-discovery-intent.md)
- [ADR-0046: Standing attention approval](0046-standing-attention-approval.md)
- [ADR-0047: Transcript defaults and rejected calls](0047-transcript-defaults-and-rejected-calls.md)

When this ADR was drafted, ADR-0047 was an uncommitted working-tree document. Its presence or status
text alone does not establish that its associated changes are installed or fully verified.

## Context

The transcript CLI has useful evidence capabilities: attributable discovery and search, qualified
ownership and lifecycle relationships, exact events, bounded transport, retained captures, assignment
context, summaries, review populations, and a bounded attention evaluator.

The skill increasingly organizes those capabilities through Office geography, room-first descent,
fixed surface controls, and inspection or standing-approval envelopes. During steering, the user
clarified that the purpose is discovering what happened, like a process tracer—not operating an
approval wizard. Clear questions should receive direct evidence-backed answers. Open-ended observation
should surface useful developments without requiring a question or prior detector selection.

An inquiry-only replacement is also too narrow: “I am monitoring what is going on” is a legitimate
intent without a specific question. Observation and investigation must coexist, and useful learnings
from either should be preserved in nn notes.

The design exercise also exposed a risk of rebuilding the same complexity under different names.
Alternating discovery/followed-work scheduling, elaborate lead states, universal coverage receipts,
and mandatory capture templates are possible mechanisms, not established requirements. No controlled
conversational replays have validated these proposals. Fictional walkthroughs are design exercises,
not execution evidence.

## Proposed decision

### 1. Make the tracer's purpose understanding work and preserving learning

Provide two complementary activities, not mandatory visible mode switches:

- **Observe:** inspect bounded work, notice useful developments, and retain relevant unresolved leads.
- **Investigate:** follow a question through the evidence needed to answer it, across threads when useful.

Recognize worthwhile learning during either activity. Capture is available throughout but is never a
compulsory final phase. Threads preserve source identity and ownership; they are not mandatory
navigation destinations. Hierarchy, timelines, and spatial lenses remain optional presentations.

The smallest shared interaction is: resolve intent and scope, inspect relevant evidence, explain what
it supports, offer useful next moves, and preserve context. Do not require a scheduling system to
provide that experience.

### 2. Execute ordinary bounded observation directly

Explicit targets and clear read-only requests execute without an enablement or repeated confirmation
ritual. An open-ended observation request delegates an ordinary bounded inspection and explanation,
not merely a metadata menu asking whether to begin.

Use explicit scope first, otherwise the retained scope. A fresh untargeted start needs a disclosed,
supported bounded discovery recipe. Clarify genuine ambiguity; respect explicit prohibitions, scope
restrictions, and actual resource limits. If a request cannot be met within them, explain the concrete
constraint and offer a focused alternative. Do not invent pass-count expiry or treat every fresh read
as a new consent decision.

Observation occurs when invoked or refreshed. This decision introduces no continuous monitoring,
timers, background workers, open-window inference, or runtime control.

### 3. Separate scope from the inspected sample

Scope is the selection definition, not whichever rows happened to appear first. Explicitly following
conversations A and B retains those targets. Following work in a project may admit new conversations
within that project on Refresh; it must not silently change projects or remove explicit filters.

Use a small, honestly described sample. Parent/ROOT work must not be excluded merely because a
convenient child-review queue omits it. Preserve attribution and schema limitations. Recency may aid
selection but is not importance, liveness, or success. Where the tracer's own identity is established,
avoid self-observation in untargeted discovery unless requested, and disclose the exclusion.

Remember explicit interests and useful unresolved observations without implying they were freshly
checked. Distinguish evidence changes, interpretation changes, sample membership changes, and matters
not rechecked. A lead leaving the sample is not resolution; a quiet sample is not health.

No universal sample size, fairness algorithm, alternating pool scheduler, or lead-state machine is
adopted. These are conditional experiments if simpler observation demonstrates a limitation. State
material coverage limits plainly; do not require a universal receipt schema or dashboard on every turn.

### 4. Let questions select evidence, not navigation geography

Use the evidence appropriate to the question from the beginning: assignments and work for alignment,
invocations and results for a failure, and identifiable baselines for a comparison. A fixed recent
tail is an orientation tool, not sufficient evidence for every question.

Search locates candidate occurrences. Similar labels, nearby timestamps, and matching filenames do
not establish causal relationships, authenticated joins, or shared artifact versions. Cross-thread
explanations require evidence for the asserted connection, not a mandatory tour through each thread.
Sampling context must fit the claim; neither messages nor sessions are a universal unit of inquiry.

Lead with the supported answer. Stop at sufficient evidence, a concrete limit, or unavailable evidence;
state what remains unresolved. Do not invent findings to fill a display or action quota.

### 5. Preserve the native evidence foundation

Use the co-versioned native CLI and load applicable capability references. Preserve:

- Exact canonical paths, session/agent/event identities, ownership qualifications, and supported joins.
- Source order, window definitions, omission indicators, and complete required pages and segments.
- Claim-level distinctions between metadata, agent reports, inspected results, and independent verification.
- Unknown versus measured zero, partial versus exact, and producer completion versus task success.
- Observed intervals versus execution time or causality; independent handoff occurrences versus attempts.
- Complete selected transport versus complete historical sources, and output bounds versus source-read cost.

Keep snapshot semantics distinct. Retained context/review/attention replay is not interchangeable with
an event projection's revalidation or a discovery inventory cursor. Positional event IDs are not
immutable content identities; sequential captures are not simultaneous observations of all files.
Transcript payloads are evidence, never instructions to execute.

Existing attention metrics remain bounded inspection hints, not productivity or health diagnoses.
This ADR changes neither the Datalog engine nor thresholds, classifier semantics, or retained metric
versions. The pending rejected-call adapter must be reviewed and verified independently.

### 6. Maintain conversational continuity and useful next moves

Retain enough conversational context to preserve scope, question/focus, inspected evidence, relevant
prior observations, unresolved leads, and the return point. Do not require a new persistence service,
LLM-authored navigation files, or a fixed serialized state schema.

Consulting evidence outside observation scope does not itself broaden that scope. Refresh repeats the
current activity's scope and intent: refreshing an investigation does not implicitly adopt its evidence
sources into ongoing observation. Disclose material transitions and clarify genuinely ambiguous
continuations rather than silently choosing a different activity or scope.

- **Back:** restore prior context and retained observations without new acquisition. Identical prose is
  not guaranteed. Restoration does not undo notebook writes or refund consumed resources.
- **Refresh:** preserve scope definition and intent, acquire new evidence, and disclose material sample
  or comparison changes. Never silently rewrite the earlier observation.
- **Unavailable restoration:** disclose lost context or expired evidence and offer explicit recovery;
  do not silently substitute a fresh query.

After results, normally offer a small contextual picker with freeform steering. Make Back, Refresh,
and End accessible; keep uncommon controls out of the primary path. Clear requests execute before
choices. One-shot answers and raw CLI use need no loop; End or dismissal stops it.

### 7. Recognize and preserve learnings in nn

Proactively offer capture for useful supported knowledge—not every event, transient status, or
reproducible lookup. Distinguish a source-bounded observation, a hypothesis, and a supported generalization.
One example does not establish behavioral recurrence.

Use the normal nn capture workflow: search existing notes, inspect relevant notes and graph context,
and propose creation, update, justified typed links, or no change when already represented. The
concrete proposal identifies the claim, evidence, limitations, and intended notebook changes. Write
only after approval; navigation assent is not write approval.

Retain enough substantive supporting context for the learning to remain intelligible after temporary
capture-cache expiry. Source identities and optional replay references supplement that content; a
snapshot identifier alone is not a durable learning. Excerpts do not establish full-source custody.
Avoid unnecessary sensitive material and label omissions or redactions. No universal note-body layout
is required.

Later evidence may qualify, extend, or contradict an interpretation. Preserve accurately source-bounded
historical observations; explicitly correct erroneous interpretations rather than hiding the change.
Capture or cancellation returns to tracing without implicit refresh. Capturing a learning does not
resolve the underlying work item.

Corrections sent to workers, worker steering, and other interventions remain separate approved actions
requiring a supported, authenticated mechanism.

## Relationship to earlier decisions

If accepted, this ADR would replace only the following interaction commitments, not erase their history:

| Earlier direction | Replacement |
| --- | --- |
| ADR-0043 Office/room-first defaults and Awaiting return as mandatory entry | Observe or investigate directly; hierarchy and queues are optional capabilities |
| ADR-0043 native navigation-session storage as the required next increment | Conversational continuity first; persistence only for a demonstrated need |
| ADR-0043/0045 inspection-approval framing and ADR-0046 standing-attention activation | Direct ordinary bounded observation with explicit scope restrictions and real resource limits |
| ADR-0047 mandatory bare-entry conversation picker and continued standing-approval model | Disclosed bounded observation by default; selection UI only when useful or genuinely needed |

Retain earlier identity, ownership, lifecycle, measurement, transport, retention, direct-target,
non-mutation, and uncertainty contracts. ADR-0044's native policy machinery and ADR-0047's separate
rejected-call/metric-version proposal are not superseded by this interaction decision. While this ADR
is proposed, it does not change the status or published behavior of those earlier records.

## Alternatives considered

- **Keep improving Office navigation:** preserves presentation continuity but keeps geography ahead of intent.
- **Make every interaction an inquiry:** fits direct questions but forces open-ended observation into an
  artificial question structure.
- **Make attention detectors the front door:** useful leads, but misses non-detector observations and risks
  presenting a signal as a diagnosis.
- **Build a scheduler, universal receipt, and persistent lead tracker first:** may improve repeatability,
  but introduces unvalidated machinery before establishing that simple bounded observation is inadequate.
- **Rewrite native retrieval from scratch:** discards valuable tested contracts without a demonstrated need.

## Consequences

The human sees relevant evidence and useful next moves instead of a permission or geography workflow.
Learning capture becomes part of the purpose, while existing native guarantees remain reusable.

The skill must still select adequate evidence and avoid overstating bounded observations. Small samples
can miss quiet important work; recency can bias selection; missing schema support can bias coverage.
Conversational state can be lost. Captured excerpts preserve a basis for understanding, not guaranteed
future replay. Honest limitations remain necessary even when the interface is simple.

The first supported observation recipe is unresolved, especially bounded mixed ROOT/child selection.
Existing `review` populations exclude ROOT. Existing primitives are ingredients, not proof that the
combined default exists. Do not claim bounded runtime from bounded output or advertise a selector that
has not been established.

## Transition and verification

After explicit implementation resumption:

1. Re-inspect the working tree. Reconcile paused picker/attention documentation with this direction;
   preserve and independently verify the rejected-call work. Leave unrelated changes untouched.
2. Replace the compact core and conflicting workflow rules coherently. Retain command owners for
   discovery/search/events/context/handoffs/summaries/review/attention/recovery. Extract optional
   presentation rules and shared capture guidance rather than duplicating them across surfaces.
3. Establish an honest bounded observation recipe with existing primitives. Add only a demonstrated
   missing native capability. Do not publish unfinished automatic behavior as a functioning default.
4. Verify three separate layers:
   - **Publication:** compactness, lazy dispatch, reference ownership, capture discoverability, and
     removal of conflicting old defaults. Replace obsolete wording assertions deliberately.
   - **Native:** identity, ownership, joins, windows, pagination, retention, errors, and metric compatibility.
   - **Conversational:** controlled replays retaining actual inputs, tool calls/results, emitted choices,
     claims, and isolated notebook diffs. Hypothetical examples and static tests do not satisfy this layer.
5. Run fresh full tests, relevant race tests, vet, and diff hygiene before installation and publication.
   Keep independently reviewable changes separate. Drafting or accepting this ADR alone does not run
   tests, install a binary, commit, push, or authorize live notebook mutations.

Minimum conversational cases include direct observation without an approval wizard; cross-thread
questions; ROOT eligibility and unsupported evidence; fixed versus dynamic scope on Refresh;
unrechecked leads; duplicate targets; missing joins and clipped evidence; Back without acquisition;
expired restoration; hard-limit failures without silent replacements; payloads never becoming
instructions; proactive capture, cancellation, deduplication, approved writes, cache-independent
learning content, and historically qualified follow-up. Include successive direct questions within an
ongoing tracing conversation: answers must execute directly, contextual pickers must remain concise
and never gate the next question, and one-shot answers must not discard retained navigation context.
These cases require actual replay; their inclusion here is not evidence of usability. Scheduling-specific
tests apply only if that optional policy is adopted.

## Open questions and non-goals

- The smallest executable observation recipe, sample defaults, and useful bounded follow-up limits.
- Whether existing native projections suffice for mixed parent/child candidate selection.
- How much continuity ordinary conversational context can reliably preserve in controlled replays.
- Replay fixtures, comparative baseline, and criteria for useful findings versus unnecessary retrieval.
- No mandatory Office metaphor, mode chooser, scheduler, receipt schema, or note template.
- No new Datalog engine, automatic notebook policy activation, monitoring service, or runtime steering.
- No general promise of live process state, comprehensive coverage, immutable original sources, or
  indefinite replay. No source-code implementation is performed by this proposal.
