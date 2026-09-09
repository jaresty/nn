# ADR-0045: Preserve discovery intent in Transcript Office attention

## Status

Accepted for implementation. Conversational effectiveness remains to be verified.

## Context

In the live lobby walkthrough following ADR-0044, a request for attention signals fell back to a
previously selected office's room. The human approved that proposal and the native evaluation was
correct, but the possible discovery job—finding where to look among the visible conversations—was
replaced with checking a nominated target. Two no-match results did not establish a useful destination.

The existing selected-target fallback allowed this interpretation. The problem is interaction scope,
not a need for another rule evaluator or automatic whole-notebook policy activation.

## Decision

Implement the consolidated contract through the co-versioned nn-transcript skill. Shared interaction
owns target resolution and authorization; attention owns the bounded discovery/evaluation recipe;
discovery, navigate, review, and rooms delegate rather than redefining it.

- An explicit operand wins target resolution, not authorization. A uniquely bound action comes next.
- Bare attention at a lobby means discovery within its explicitly identified displayed conversations;
  at a hallway it means discovery within that office and active queue/filter; at a room it means that
  room. A background room selection cannot override broader visible discovery intent.
- One concrete proposal binds candidate scope, metadata selection, attempted-room cap, classification
  retrieval, output reservations, freshness, follow-ups, and stops. Acceptance executes that bounded
  operation without per-room reconfirmation. It does not authorize implementation, capture, or control.
- Count attempted candidates, including unknowns and errors. Do not silently replace them, widen an
  empty filter, extend pagination, or refund budget on Back.
- Task scope cannot be inferred from a label. Preserve missing classification versus established
  inapplicability, native indeterminate/no-match outcomes, errors, and unevaluated scope separately.
- Keep retained evidence identity and hallway membership/order unchanged. Fresh evaluation is separate
  from a hallway refresh. No-match is not health and does not automatically widen discovery.

The small default is at most three attempted rooms: one per named lobby conversation (at most three),
first three rooms on a retained hallway page, or the selected room. The owning skill specifies metadata
selection, optional bounded assignment context, native policy calls, and cumulative reservations.
These are declared defaults for a proposal, never standing permission. Native processing may still read
whole files; output reservations are not source-read or runtime limits.

## Consequences and verification

No new runtime navigation state, automatic scanner, task classifier, policy, or production toggle is
introduced. Existing attention command semantics and Datalog rules are unchanged. Native commands
continue to own identities, ownership, transport, and retained results; the LLM coordinates intent.

Publication guards verify served owner text and dispatch, not model compliance. An addressable replay
matrix covers stale selection, explicit out-of-scope operands, empty filters, unknown task scope,
consumed attempts, missing sources, and Back after budget exhaustion. Conceptual examples are not
executed conversational replays; report that limitation until actual replays are conducted.
