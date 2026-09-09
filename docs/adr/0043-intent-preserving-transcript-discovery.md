# ADR-0043: Intent-preserving transcript navigation and evidence-guided discovery

## Status

Accepted architectural direction; implementation pending except for the delivered foundations
listed below. Candidate signals, sampling strategies, thresholds, and command/schema designs remain
provisional. This record does not authorize automatic notebook writes, correction delivery, or runtime
control.

Date: 2026-09-09

Related: [ADR-0042: Transcript navigator](0042-transcript-navigator-duckdb-spine.md).
This ADR extends the interaction and discovery model; ADR-0042 continues to own canonical identity,
authenticated ownership, qualified parentage, lifecycle evidence, retained captures, and bounded transport.

## Context

The September 9 interaction trace exposed a gap between safe transcript queries and useful human
navigation. Open, Find, and Inspect sometimes reopened choosers despite an evident contextual target.
Sequential pattern filters charged conversational turns for empty results. Filter state and selection
were not consistently visible. A frozen unclosed-handoff desk could differ from newer room evidence.

Native readable bundle retrieval and evidence-based action suggestions have reduced some friction,
but neither alone establishes a consistent intent-preserving interaction model. These are observations
from a bounded interaction trace, not a representative usability study.

The metric discussion also distinguished observations from conclusions. Exact command recurrence,
explicit errors, and recognized edit counts can be mechanically computed. They do not establish
looping, drift, productivity, success, or a blocker. Read-only certification can correctly contain no
edits; implementation can alternate long investigation and verification phases. Shell side effects
may be unclassified. Message and tool-result records can represent the same invocation.

Sampling is a possible way to bound semantic inspection, not an accepted replacement for mechanical
signals. Fixed recent-event windows can be dominated by duplicates and setup instructions. Conversely,
error-only selection misses useful discoveries and subtle problems in apparently healthy work.

### Frame comparison

A query-first frame makes available filters the navigation model. An intent-preserving frame makes
the human's question and selected target the navigation model, compiling those into supported queries.
We choose the latter while retaining evidence custody. We reject edit volume as a proxy for quality
and repeated confirmations as a substitute for explicit scope and state.

## Decision

### 1. Separate evidence, interpretation, and authorization

- **CLI:** deterministic parsing, authenticated identity/ownership and joins, bounded selection,
  mechanically defined counts and recurrence, source event identities, retained snapshots, provenance,
  and transport. It does not run an LLM or assert semantic progress.
- **LLM-mediated Office:** interpret evidence, identify candidate patterns, recommend useful next
  actions, maintain conversational navigation state, and draft evidence-qualified proposals.
- **Human:** choose intent and scope, resolve genuine ambiguity, and approve concrete notebook or
  correction proposals. Approval to inspect evidence is not approval to write or send anything.

Mechanical observations can nominate an exchange for inference. An inferred pattern may nominate
further evidence for corroboration. Neither step converts a hypothesis into an established finding.
No composite productivity score, universal RAG threshold, or inferred runtime-liveness metric is adopted.

### 2. Preserve intent and explicit navigation state

Maintain selected conversation, room, and event; current surface; queue/filter; approved scope;
question/lens; recommended action and exact target; retained evidence identifiers; newer overlays;
and Back history. The state is conversational, not a claim of a persistent host UI or open windows.

Show a compact breadcrumb and selected target. Resolve commands using explicit operands first, then
an applicable selected target, then a unique applicable recommendation. If those do not resolve the
intent without changing scope, ask one focused clarification. Never infer a target from an empty filter
by silently returning to an unfiltered population.

- **Open** executes the selected target; **Open...** selects a different target.
- **Find** performs a bounded evidence-guided diagnostic within established scope rather than requiring
  the human to select a CLI filter. State the bounds; confirm a new or enlarged semantic scope.
- **Inspect** follows the contextual selected/recommended evidence item when unambiguous.
- **Back** restores the actual retained view, options, selection, question, and evidence without refresh.
- Echo numeric target selections before retrieval; preserve immediate conversational correction and
  Back. Navigation recovery does not undo external actions.

Room entry should give a bounded readable orientation without requiring a second 'orient me' turn.
This is not permission to impose an analytical lens or silently expand history. CLI mechanics stay
behind the Office surface unless requested or a failure requires a real human decision.

### 3. Recommend actions from evidence, not a fixed menu

Promote two or three concrete useful actions, with the target clear. Generic level menus remain
fallbacks; displaced controls stay under More, with Back and End always visible. An empty result should
lead to a justified next action or an honest statement that nothing compelling was found, not another
unranked filter menu. Do not invent findings to fill action slots.

'Attention' is broader than the open-handoff predicate. Disclose which population and signals a view
actually covers. Overlapping signal counts are not additive categories; unavailable evidence is not zero.

Capture remains available on every surface through 'capture that' and More. Promote a specific capture
only for a useful supported candidate. Search existing notes and propose creation/update with source
session/room/event identities and evidence limits. Require explicit approval of the concrete proposal
before writing notes or links. Restore the view on cancellation or completion.

Correction is not a default prominent action when no concern is established. Retrieve sufficient
assignment and work evidence before drafting. Recorded returns are not proof of successful completion,
and no correction is delivered without separate authorization and an authenticated delivery mechanism.

### 4. Inspect coherent bounded evidence

The next retrieval capability should select an invocation, its authentically matched result when
available, and explicitly bounded surrounding activity. Preserve native event IDs, ordinals, source
provenance, and canonical ordering; exchange presentation must not rewrite the underlying ledger or
invent joins. Missing/ambiguous joins remain explicit, including results absent at capture time.

Count each identified invocation once for activity summaries. Do not count enclosing messages and
extracted tool records as independent operations. This is not authorization to merge unrelated calls
with the same name or arguments. Exact duplicate identification must use authenticated source identity;
ambiguous invocation ownership/identity is disclosed rather than guessed.

Bound both selection work and output, measure their costs separately, and retain transport integrity.
Text remains lossy; exact evidence must be available when a claim depends on clipped content. No client
Python, ad hoc parsing, or manual page loop should be necessary for ordinary readable inspection.

### 5. Pilot signals and semantic follow-up together

Candidate hints for the first pilot:

- Recognized edit/non-edit/unclassified tool-call counts, with classification coverage.
- Exact repeated invocations under a documented tool-and-argument equality rule.
- Explicit failures and repeated failure signatures under a documented extraction/equality rule.

Report counts and denominators, not just ratios. Zero edits is 'no recognized edits', not an infinite
score. Unknown shell effects must not be silently classified as non-edit. Assignment and phase context
are necessary before interpreting edit activity. Counts from overlapping windows are not independent
corroboration.

The LLM inspects nominated exchanges and, if useful within scope, follows up with assignment context,
an exact result, or a bounded adjacent exchange. A small exploratory sample is a candidate way to avoid
signal-selection bias; its size, selection strategy, reproducibility, and efficacy require evaluation.
No fixed three-room/five-event policy or specific ranking algorithm is adopted here.

Coverage must distinguish eligible population, selected/retrieved evidence, actual inspected content,
truncated or uninspected content, omitted history/rooms, and unknown/unavailable evidence. Complete
transport is not complete semantic inspection or original-source completeness.

### 6. Overlay freshness without rewriting history

When entering a room from an unclosed-handoff view, acquire a bounded newer handoff observation while
preserving the original desk snapshot. Label both evidence identities and disclose relevant changes:
'on desk snapshot: unclosed; newer handoff evidence: returned'. Failed or unavailable refresh remains
unknown and must not invalidate the retained view. A return does not establish task success.

This observation is sequential, not a globally simultaneous source snapshot. It must not silently
replace the selection or previous findings. Back returns to the original view. Benchmark the check
before calling it lightweight; define its failure and latency behavior before enabling it by default.

'What changed?' requires identified baseline/current captures. Continuous watching, notification,
refresh scheduling, and runtime steering are separate future decisions, not implied by this ADR.

## Alternatives considered

| Alternative | Benefit | Why not the selected direction |
| --- | --- | --- |
| Richer deterministic pattern dashboard first | Cheap, explainable counts | Makes filters the navigation model and risks presenting signals as diagnoses |
| Pure recent-tail semantic sampling | Simple and open-ended | Duplicate/setup-heavy tails and recency bias can waste context; needs corroboration |
| Unbounded whole-transcript inference | Broad potential context | Unacceptable cost/latency and weak accounting of actual inspection |
| Edit-ratio or composite productivity ranking | Easy prioritization | Task-dependent, vulnerable to classification errors, and rewards inappropriate editing |
| More confirmations at each transition | Explicit individual actions | Repeated ceremony does not fix hidden selection, scope, or stale-view semantics |
| Replace frozen evidence on each navigation step | Appears current | Breaks reproducible Back and obscures changed evidence |

## Consequences

Benefits: fewer translation/selection turns; clearer scope and recovery; targeted use of context;
reproducible evidence; useful findings beyond error counting; capture integrated without automatic writes.

Costs: a shared navigation-state contract, new bounded exchange selection, conservative classification,
comparison/overlay semantics, and evaluation of semantic recommendations. A skill instruction alone
cannot guarantee correct LLM behavior; replay testing must assess actual interactions.

Risks: signal bias, false alarms, missing shell edits, stale evidence, expensive whole-conversation
projection hidden by small output, and overconfident summaries of clipped text. Mitigate through explicit
unknowns, native identities, bounded follow-up, retained snapshots, and matched-baseline evaluation.

## Resumption snapshot

### Delivered foundations (historical commit anchors)

- `3692914`: immutable complete-record-prefix captures and cached transport-page replay.
- `38e0502`: bounded native readable review/context bundles with automatic page consumption.
- `924c6e9`: suggested-action and approval-only capture skill contracts.

These anchors document delivered work, not proof that every new behavior in this ADR already exists.
The supplied interaction trace shows remaining selection/menu friction after those changes.

### Relevant implementation areas

- `cmd/nn/cmd/transcript_review.go`: queue selection and existing deterministic filters.
- `cmd/nn/cmd/transcript_context.go`: assignment/recent bundles.
- `cmd/nn/cmd/transcript_capture.go`: retained inputs and page replay.
- `cmd/nn/cmd/transcript_bundle_text.go`: readable bounded bundle rendering.
- `cmd/nn/cmd/transcript_events*.go`: event identities, ownership, joins, selection, and transport.
- `skills/nn-transcript/SKILL.md` and `references/`: interaction/state/action contracts.
- Transcript command/contract tests and performance documentation under `docs/`.

Re-inspect exact integration points before implementation. No new command name, flag set, cache format,
or schema is specified by this ADR.

### Pending work and ordered continuation

1. **Reconcile state and interaction contracts.** Inventory conflicting fixed-menu/metadata-first rules
   across the core and references. Define selection, scope, recommendation targeting, breadcrumbs,
   and exact Back transitions. Preserve the compact-core/lazy-reference contract.
   Acceptance: replay Open/Find/Inspect/Back without unnecessary choosers or silent filter clearing.
   Next: use this state model to bind the single-room pilot.
2. **Implement one-room activity evidence.** Specify invocation identity, classification and unknowns,
   equality rules, window bounds, and count denominators before adding a native summary.
   Acceptance: duplicates do not inflate counts; shell ambiguity and missing joins remain unknown;
   read-only work is not scored poorly for lacking edits. Then nominate exact source evidence.
3. **Implement bounded exchange retrieval.** Retrieve a selected invocation/result plus limited
   surrounding activity from retained evidence. Design limits and option/snapshot binding explicitly.
   Acceptance: stable continuation after append/deletion; missing/ambiguous/out-of-window joins remain
   qualified; no partial publication after corrupt/missing pages; readable output requires no script.
   Next: expose direct inspection from a finding.
4. **Close the semantic loop.** Combine activity hints, bounded exchange inspection, and optional
   approved exploratory sampling. Offer a concrete recommendation, not a compulsory filter choice.
   Acceptance: distinguish observation/hypothesis/finding, report inspection limits, and allow 'nothing
   compelling'. Consult assignment context before drift/low-edit judgments. Then test capture proposals
   and cancellation while preserving navigation state.
5. **Add and benchmark handoff overlays.** Define retained-versus-new evidence and unavailable/error
   behavior before enabling entry-time checks. Acceptance: a return during navigation is shown without
   implying success or rewriting Back. Then evaluate the end-to-end pilot.
6. **Evaluate against recent-tail inspection.** Run both approaches over the same retained conversations
   with declared comparable room/context budgets, including clean read-only work, productive edit/test
   cycles, unchanged retries, recovered failures, shell edits, missing evidence, and useful discoveries.
   Measure useful supported findings, false alarms, context consumed, processing/replay latency, and
   user turns. Record truncation, omissions, and reviewer judgment basis. Report unfavorable results.
   Expand to multi-room prioritization only if the pilot justifies the added complexity; otherwise
   revise selection/context before adopting thresholds or a dashboard.

For executable changes, use assertion-specific regression tests and isolated mutations, then normal/race
suites, vet, diff hygiene, installation, and realistic live/retained replay measurements. Documentation
checks validate dispatch and consistency, not semantic recommendation quality.

### Open choices / non-decisions

- Invocation classifier boundaries, shell-command handling, and exact recurrence/signature equality.
- Window size/overlap, exploration sampling strategy, and follow-up/context budgets.
- Native API/schema names, cache reuse, and processing limits for activity/exchange projections.
- How assignment classification is evidenced and how updates affect interpretation.
- Metrics for useful semantic findings and evaluation acceptance criteria; no validated thresholds yet.
- Persistent cross-conversation UI state, watch scheduling, and runtime control remain outside scope.

### Immediate next action

Start with step 1 and freeze concrete conversational replay cases before implementing step 2.
The first delivery is one room -> activity summary -> coherent exchange -> qualified semantic finding
-> direct Inspect, with exact Back and approval-only capture. Do not begin with a dashboard.
