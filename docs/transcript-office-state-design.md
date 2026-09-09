# Native Transcript Office navigation state — proposed design

Status: design only; no commands or flags below are shipped. Extends ADR-0043.

## Problem and boundary

Requiring the LLM to serialize rendered views to temporary JSON creates avoidable tool calls, context,
and failure modes. Existing evidence snapshots do not retain navigation state, and LLM rerendering is
not deterministic. The immediate skill correction removes mandatory file persistence and retains
conversational state with explicit loss/expiry disclosure.

The native facility should restore **the same structured navigation and evidence state**, not the same
prose, semantic finding, or model-generated recommendation. It must not store LLM conclusions as
canonical facts or turn cached navigation into fresh source authority.

## Minimal data model

- **Office session:** opaque random ID, monotonic revision, current history position, scoped envelope
  and monotonic usage ledger. Private per-user cache, atomic updates, bounded retention and explicit
  deletion. It is UI state, not notebook content and not a claim about runtime worker sessions.
- **Immutable view:** opaque ID, prior-view ID, projection kind, native query options, canonical selected
  session/room/event identities, ordered picker mapping, evidence/capture references, structured native
  projection result, and separately typed action-target bindings. No rendered LLM prose, reasoning,
  conclusions, or inferred progress. A question/lens is user navigation intent, never evidence.
- **History:** ordered view IDs with current position. A new successful navigation after Back drops the
  forward branch; no budget or approval is restored. Missing history endpoints fail explicitly.
- **Budget ledger:** authorization/envelope version, limits, reserved/consumed amounts, unique request IDs.
  It is independent of view history. Navigation does not create or expand authorization.

Native projections populate state: do not expose a workflow that makes the LLM reconstruct all row,
filter, picker, and capture fields in a giant `save --view <JSON>` argument. User/model-supplied selectors
are validated against the retained projection. An action target can be stored as a typed navigation
binding; its model-supplied rationale is not part of canonical navigation state.

## Proposed operations (names subject to implementation design)

1. **Start/status:** create or inspect an Office session; return its opaque ID, revision, current view,
   envelope usage, and availability. No implicit global 'current office'. Every later operation requires
   the session ID, preventing two conversations from sharing accidental history.
2. **Navigate/select:** opt an existing projection into this session, supplying its normal selectors and
   expected session revision. The CLI records options, native structured output, picker mappings, and
   evidence references directly. Selection can change without reacquiring evidence. New projection
   publication becomes current only after complete success; an error leaves the previous view usable.
3. **Back/forward:** move the history pointer and return the retained structured view with the current
   session ledger. Do not reread live transcripts or recompute a projection. If evidence is unavailable,
   return retained state as historical with unavailable evidence inspection; never silently refresh.
4. **Refresh:** explicit new acquisition under existing identity/scope constraints; publish a new view,
   preserving the former view. Source changes never rewrite a prior view. A failure preserves selection.
5. **Authorize/reserve/settle:** an explicit envelope is recorded; the tool can enforce numeric and scope
   limits, not authenticate whether a human actually consented. The assistant still owns the approval
   boundary. Reserve conservatively before retrieval, reject requests outside scope/budget, and use
   idempotency keys so retries cannot double-charge. Back never releases consumption. Failed requests
   and abandoned reservations require a specified conservative accounting rule before implementation.
6. **Close:** remove the navigation session without deleting independently authoritative evidence or
   mutating notes. Expiry/deletion is explicit on reuse; no hidden recreation.

Illustrative surface: `office start/status/back/forward/close` and an opt-in office identifier/revision
on supported projections. Do not add a token to every existing response or change legacy command JSON.
Actual syntax must be chosen after inspecting command integration and cache ownership.

## Rendering and restoration

The CLI can emit deterministic text for its structured native model. The LLM may explain it, but cannot
promise verbatim historical explanations. Back restores selection/filter/order/evidence references and
budget visibility. If an earlier interpretation is absent from conversational context, do not reconstruct
it as though it were the prior finding. New analysis is labeled new and subject to the remaining envelope.

Preserve context-bound action targets only when explicitly stored and still applicable. Do not invent a
recommendation merely because one was previously displayed. A discarded/expired binding is unavailable,
not an excuse to redirect a short command to another room.

## Integrity and concurrency

Opaque IDs do not grant evidence authority. Validate projection references, selector membership, scope,
and ownership. Use compare-and-swap on the session revision; stale writers fail without corrupting
history or spending budget twice. View publication and navigation-pointer changes are atomic. Retention
must bound disk size/history length and disclose pruning; do not silently substitute a newer view when
an old one is pruned. Native evidence cache lifetime is independent of navigation metadata lifetime.

An authorization record is not a notebook-write or runtime-control grant. This facility never creates
notes, sends corrections, steers workers, infers liveness, or authenticates task success.

## First implementation slice and tests

Start with an explicit Office session, native review/context projection attachment, selection, status,
Back/Forward, revision checks, and monotonic ledger. Defer generic document storage, cross-session
persistent UI, monitoring, metrics, and automatic handoff overlays.

Acceptance cases:

- A native projection creates a view without LLM-authored JSON or temporary-file tool calls.
- Back/Forward restores exact structured state after live append/deletion, without source rereads.
- Budget remains spent across Back, Forward, branch replacement, and retry; unknown usage fails closed.
- Two Office sessions remain isolated; stale revisions cannot overwrite a newer selection.
- Failed navigation keeps the prior state; corrupt/missing/expired artifacts fail or disclose retained
  metadata-only availability without a silent refresh.
- Legacy commands remain unchanged; no LLM conclusions enter canonical state.
- Compare operations, bytes, and latency against manual persistence. Test semantic rendering separately
  from deterministic native-state equality.

## Open implementation choices

Cache schema/version and retention caps; exact command/flag syntax; which native projection shapes can
be retained without reparsing; atomic ledger/publication ordering; failed-operation charging and retry
recovery; evidence reference validation; and storage of optional action bindings. Resolve these before
implementation, rather than shipping a generic JSON store and recreating the original burden.
