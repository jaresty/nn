# ADR-0043: Intent-preserving transcript navigation and evidence-guided discovery

## Status

Accepted architectural direction. The baseline-A skill contract and native capability/contract tests
are implemented; controlled conversational evaluation and conditional follow-on primitives remain pending.
See [baseline A implementation and evaluation boundary](../transcript-interaction-baseline.md). Candidate signals, sampling strategies, thresholds, and command/schema designs remain
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

Show a compact breadcrumb and selected target. Bind every promoted action to a verb, exact target,
selection options, and evidence identity. Resolve explicit operands first, then the uniquely displayed
matching action, then an applicable selected target. A background selection must not override the
visible action's promise. If these do not resolve intent without changing scope, ask one focused
clarification. Never silently clear an empty filter to borrow a target from the underlying desk.

Back restores structured navigation/evidence state, not identical LLM prose. A snapshot alone cannot
restore an interpretation; rerendering by an LLM is not deterministic. Do not require the LLM to create
private temporary JSON or preserve rendered responses. Until native navigation storage exists, retain
concise conversational state and disclose loss across compaction. Never silently refresh lost evidence.

A small native navigation session is the chosen next persistence direction: store selection, filters,
picker mappings, history, action targets, and evidence references, not LLM prose or conclusions. Keep
consumed inspection budget in a separate monotonic session ledger; Back/Forward cannot refund it.
Evidence captures remain authoritative and separately retained. This facility is designed, not shipped;
see [native navigation-session design](../transcript-office-state-design.md) for scope and open choices.

- **Open** executes the selected target; **Open...** selects a different target.
- **Find** performs a bounded evidence-guided diagnostic within established scope rather than requiring
  the human to select a CLI filter. State the bounds; confirm a new or enlarged semantic scope.
- **Inspect** follows the contextual selected/recommended evidence item when unambiguous.
- **Back** restores retained navigation options, selection, question, and evidence without refresh;
  it does not guarantee identical prose or recomputed conclusions.
- Echo numeric target selections before retrieval; preserve immediate conversational correction and
  Back. Navigation recovery does not undo external actions.

Room entry should give a bounded readable orientation without requiring a second 'orient me' turn.
This is not permission to impose an analytical lens or silently expand history. CLI mechanics stay
behind the Office surface unless requested or a failure requires a real human decision.

### Awaiting return as the default Office population

The default Pi Office view uses `review --queue awaiting-return --order observed-recent`:
authenticated launches with zero recorded parent returns, regardless of terminal count. Show every
row of each bounded page with eligible total, current range, and Next; a three-action recommendation
budget must not become a three-room listing cap. Badge terminal evidence without inferring liveness.
The current Pi producer-terminal projection also counts those records as returns; do not invent a
terminal-only population. Repeated launches with any return remain ambiguous, not paired attempts.
All rooms and hierarchy remain under More → All rooms. Legacy CLI review defaults remain compatible;
the Office explicitly selects its queue. Non-Pi offices retain the disclosed hierarchy fallback.

### Human hallway ordering

Explicit Pi hallway browsing requests `tree --parent ID --order observed-recent`; legacy CLI canonical ordering stays
unchanged. Rank all direct children before pagination: optional `--selected ID` pinned first, then
greatest valid owned assistant/tool-result work timestamp descending, unknown last, ID tie-break.
Launch/return/lifecycle timestamps and file modification times do not stand in for observed work.
An explicit selected room must be a direct child; it is a presentation pin, not a reparenting operation.

Capture and persist the ordered child projection once. Bind parent, order, selected ID, limit, source
input, and strictness into retained continuation. Cached continuation does not reread live sources;
missing/expired/corrupt cache fails explicitly. Ordinary canonical unpinned pagination is unchanged.
Initial recent/pinned support is Pi-only; other schemas keep clearly labeled canonical browsing until
an equally authoritative work-recency adapter exists. A recent page is not an activity or importance
ranking. Label it 'Recent rooms'; retain its rendered/conversational state on Back without refresh.

### 3. Recommend actions from evidence, not a fixed menu

Promote up to three concrete useful actions, with the target clear. Generic level menus remain
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

### Match initial evidence to the question

Activity-only questions use recent-work bundles. Single-room assignment-alignment questions use
`context` first; multi-room alignment uses opt-in `review --last N --include-assignment`. The option
includes native payloads and all retained independent launch records for the selected room page,
using the same captured inputs and handoff joins as context. Missing/ambiguous assignments remain
qualified; no governing occurrence or steering is inferred. Default review output is unchanged.
Assignment records have separate counts from recent events and share bounded, cached transport.

Plan assignment retrieval as part of the initial inspection envelope when alignment is the question.
It must not consume an arbitrary per-room follow-up merely because the first operation omitted necessary
context. Initial and follow-up output still count against the declared total budget; enlarge it explicitly
when required. Do not include potentially large assignments in every activity-only scan by default.

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

Separate the initial sample from the authorized inspection envelope. The envelope names the exact
room set, initial window, permitted follow-up kinds/counts, cumulative context/output budget, freshness
policy, and stop conditions. Approval of an envelope permits its bounded read-only follow-up without
repeated confirmations. Approval of only an initial sample does not authorize unspecified expansion.
When no envelope exists, state a concrete bound and obtain agreement through the selected action.

The LLM inspects nominated exchanges and follows up with assignment context, an exact result, or a
bounded adjacent exchange only within that envelope. Stop when there is enough evidence for a useful
next action, the budget is exhausted, or the needed evidence is unavailable. Ask before enlarging the
room set, follow-up budget, or other authorization boundary. A small exploratory sample is a candidate way to avoid
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

Re-inspect exact integration points before implementation. No shipped command name, flag set, cache format,
or schema is added by this ADR; the linked native-state design describes proposed operations.

### Pending work and ordered continuation

The interaction-only baseline is a real delivery, not a prerequisite for an already-chosen metrics
pipeline. Evaluate each increment before committing to the next. Native summaries and exchange
retrieval are conditional on a demonstrated limitation of existing commands.

1. **Reconcile state and interaction contracts.** Inventory conflicting fixed-menu/metadata-first rules
   across the core and references. Define selection, scope, recommendation targeting, breadcrumbs,
   and navigation-state Back transitions. Preserve the compact-core/lazy-reference contract.
   Acceptance: replay Open/Find/Inspect/Back without unnecessary choosers or silent filter clearing.
   Next: freeze replay cases and evaluation criteria before adding evidence machinery.
2. **Deliver baseline A using existing commands.** Combine sticky selection, action bindings, readable
   recent evidence, exact-event inspection, assignment context when authorized, and retained-view Back.
   Exercise a stale selected event A with a visible Inspect B action, empty-filter Inspect, sample versus
   follow-up authorization, capture cancellation, compaction loss, and a failure hidden by truncation.
   Acceptance: explicit operands win; Inspect follows the visible action; no silent scope expansion;
   no automatic notebook write; no silent refresh on Back; no scripts or manual view files for ordinary navigation. A useful finding
   cites enough retained evidence for its conclusion and recommends a relevant next action. A false
   alarm asserts a concern unsupported by inspected evidence; unnecessary follow-up consumes retrieval
   without resolving a declared uncertainty. Record those judgments, not just counts.
3. **Evaluate A before choosing increment B.** Use a fixed retained test set and declared comparable
   room/context budgets. Compare the prior recent-tail/menu baseline against A, then compare any new
   selection primitive B against A, so interaction gains cannot be misattributed to new metrics. Include
   read-only work, productive edit/test cycles, repeated/recovered failures, shell ambiguity, missing
   evidence, and useful discoveries. Record supported findings, false alarms, context, latency, and user
   turns. No identity/authorization/Back regression is acceptable. Declare a target bottleneck and a
   measurable success criterion before B; report trade-offs and unfavorable results rather than assuming
   more plausible findings means improvement. If A suffices, stop without adding B.
4. **Add only a demonstrated missing primitive.** If A cannot retrieve the required coherent evidence
   within budget, design bounded invocation/result plus surrounding retrieval. If candidate selection
   is the measured bottleneck, trial activity hints with defined invocation identity, unknown classifier
   cases, equality, windows, and denominators. These are alternatives, not mandatory sequential features.
   Acceptance includes no duplicate count inflation, no guessed joins/shell effects, source identities,
   snapshot integrity, and stable bounded transport. Evaluate B against A before extending.
5. **Test freshness overlays separately.** They are not a prerequisite for discovery evaluation. Define
   old/new identities, unavailable/error and latency behavior before enabling entry-time checks. A return
   during navigation must not imply success or rewrite Back. Until this increment passes, make refresh
   explicit and retain the older view as historical. Continuous watching remains out of scope.

For executable changes, use assertion-specific regression tests and isolated mutations, then normal/race
suites, vet, diff hygiene, installation, and realistic live/retained replay measurements. Documentation
checks validate dispatch and consistency, not semantic recommendation quality.

### Open choices / non-decisions

- Invocation classifier boundaries, shell-command handling, and exact recurrence/signature equality.
- Window size/overlap, exploration sampling strategy, and follow-up/context budgets.
- Native API/schema names, cache reuse, and processing limits for activity/exchange projections.
- How assignment classification is evidenced and how updates affect interpretation.
- Pilot dataset, quantitative improvement targets, and reviewer agreement; qualitative acceptance
  criteria above are fixed, but no validated metric thresholds exist.
- Persistent cross-conversation UI state, watch scheduling, and runtime control remain outside scope.

### Immediate next action

Mandatory manual view persistence has been removed from baseline A. Use conversational navigation
state with explicit restoration limits now. Validate and implement the linked minimal native-session
design next: manual state-management friction is an observed limitation, unlike unproven metric needs.
Retain the one-room inspection loop and conversational replay cases; contract tests alone do not
establish model compliance or benefit. No metric dashboard is implied.
