# ADR-0046: Standing approval for view-triggered attention

Status: Accepted; default pass-count expiry superseded by
[ADR-0047](0047-transcript-defaults-and-rejected-calls.md). Standing approval now uses per-check
bounds until opt-out/session end; explicitly human-imposed total limits still apply.

## Context

ADR-0045 distinguishes discovery from evaluation but prohibits automatic checks on view entry.
Requiring users to invoke every check makes attention an on-demand diagnostic rather than a way to
notice something they did not know to ask about. The user approved one-time opt-in for bounded checks
on Office entry and explicit refresh, with separate permission for investigation and intervention.

## Decision

Revise ADR-0045's entry-time prohibition: a session-scoped, explicit standing envelope can authorize
an initial check and subsequent checks on eligible Open or explicit Refresh actions. No timer, polling,
background worker, native navigation hook, or notebook configuration is introduced. The LLM follows
the skill and invokes the existing CLI; the native evaluator and policy are unchanged.

The envelope names canonical conversations, permitted surfaces and queue/filter selection rules,
per-pass candidates, metadata/task retrieval limits, output reservations, freshness, total passes and
cumulative allowance. Candidate IDs are resolved under that approved rule, not preselected by the user.
A suggested starting allowance is three passes, each using the existing at-most-three-room recipe;
this is a proposal, not automatic authorization. Smaller proposals are appropriate when context is
limited. Entry and refresh consume that allowance; they do not reset it.

Activation may check the current view once. Subsequent approved Open and explicit Refresh actions
trigger checks without another attention command or per-room approval. Restoration, repeated rendering,
transport pagination, and incidental tool completion do not trigger checks. A single navigation action
triggers at most one pass, even if it loads several workflow references. Reuse retained room results
when opening their evidence rather than implicitly evaluating them again.

Standing approval ends on opt-out, End, a new conversation, or lost authorization/consumption state.
Exhaustion pauses checks. An out-of-scope view can still open normally; disclose that attention was
not evaluated there and offer explicit extension. Do not repeatedly interrupt navigation to renew.
There is no budget refund on Back and no scope widening through new pages, changing filters, or a
refreshed lobby that contains new conversations.

Surface compact match cues with retained snapshot identity and a bounded coverage line. No matches,
unknown task scope, indeterminate results, errors, and unevaluated/skipped rooms remain distinct.
Preserve hallway membership/order and separately label attention freshness. Deeper inspection, extra
history, wider scope, capture, and intervention require their own approval unless separately covered.

## Explicit requests are authorization too

The user's recovery-check example exposed a separate confirmation loop: after “check if the signal
has recovered,” the assistant proposed that exact check and waited for “run it.” A clear imperative
for a known target authorizes its ordinary bounded read-only operation. Freshness alone does not
require another confirmation. The attention owner defines the one-room recovery-check recipe;
unknown targets, undefined bounds, additional scope, and explicit hard limits still require resolution.
This does not authorize mutations or enable standing checks. A current no-match is not proof that the
underlying work has recovered. Preserve the original result and disclose comparison limitations.

## Consequences and verification

Attention becomes proactive within a human-approved scope without continuous monitoring or control
of workers. Existing one-shot evaluation remains available when standing attention is off.

Published skill clauses and cross-owner consistency are regression-tested. These are instruction
publication checks, not runtime automation or evidence of LLM compliance. Controlled conversational
cases include enable → Open → Refresh, opt-out → Open, Back/re-render/pagination, exhaustion, new
conversation in refreshed lobby, unknown classification, and inspecting a retained matching result.
Their successful execution must not be inferred from passing static tests.
