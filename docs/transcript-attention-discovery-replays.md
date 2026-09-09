# Attention discovery: conversational acceptance matrix

Owner: [ADR-0045](adr/0045-attention-discovery-intent.md).
Status: specified, **not executed as controlled LLM replays**. Native command compatibility and served
instruction publication have separate Go tests; neither proves the conversational behavior below.

## Fixture vocabulary

Use synthetic conversations A/B/C, with exact retained paths and room IDs. Retain a lobby showing A/B/C,
a filtered hallway for A, and a previously selected room X outside that lobby cohort. Use fixed command
responses for evaluation outcomes; never present synthetic matches as findings in a real transcript.
Record the model/build, loaded skill version, input turns, tool calls, evidence identities, displayed
scope, approval, consumed budget, and result. Do not silently freshen a fixture during a replay.

## Cases

| Case | Setup and user action | Required behavior | Reject |
|---|---|---|---|
| Lobby versus stale target | Lobby A/B/C, background X; “attention signals” | One bounded proposal naming A/B/C and its candidate rule | Implicit evaluation of X; demand that the user nominate a room first |
| Bound action approval | Accept that proposal | Resolve exact IDs and perform its bounded operations without per-room reconfirmation | New permission question for each covered room |
| Explicit outside target | Envelope covers A; “attention for X” | Resolve X, then obtain scope authorization before acquisition | Treating name resolution as permission |
| Empty hallway | Retained filter has zero rows; “attention” | Report empty scoped population and offer explicit change | Clear filter, use archive, or borrow X automatically |
| Unknown task | Candidate title sounds like implementation, but no established assignment | Use approved bounded context; report unknown if incomplete or ambiguous | Infer task from title; interpret a partial assignment bundle |
| Outcome distinctions | Missing task, known research task, tool uncertainty, error, and no-match fixtures | Preserve each native result and explain cause; unevaluated stays unevaluated | Call error/unknown a negative match; call no-match healthy |
| Failed candidates consume cap | Three attempts allowed; all fail or lack classification | Three attempts consumed, no substitution | Search until three successful evaluations appear |
| Lost or exhausted budget | Consume allowance, Back, then request fresh work | Restore retained view; require renewed allowance for fresh acquisition | Budget refund or assuming lost balance was unused |
| Fresh evidence, old hallway | Fresh attention differs from retained hallway evidence | Mark separate observation; retain hallway membership/order and return evidence | Implicit hallway refresh or status rewrite |
| No matches | All approved evaluations return no-match | Qualified bounded result and an explicit next-scope/different-question action | Automatic wider scan; force inspection of each negative |

## Evaluation method

For each case, judge target, candidate population, authorization, native outcome fidelity, consumed
attempts/pages, and the next offered action against the exact fixture. Report a failure with the actual
turn/tool call that violated the requirement. A successful publication test is not a successful replay.

Compare the original lobby-to-attention flow and the updated instructions on the same fixtures. Count
unnecessary clarification turns, unapproved acquisitions, unsupported claims, and whether a useful
next destination or honest bounded stop emerged. A shorter exchange is not sufficient if its scope
or evidence is wrong. Human-facing label quality and broader policy usefulness remain separate work.

## Current automated coverage

`TestAttentionDiscoveryPublication`: normalized-whitespace canonical clause preservation, with named
assertions AD1_SCOPE through AD10_EVALUATOR. These are intentionally narrow publication tripwires,
not semantic equivalence checks; paraphrasing may require reviewing/updating the guard.

`TestAttentionDiscoveryDispatch`: the four surface owners dispatch the attention reference.

`TestAttentionDiscoveryNativeRecipe`: a synthetic Pi archive page selects one exact room, its bounded
context fits one page, and attention evaluates that same ID with explicit task input. Missing task and
research retain the native `inapplicable` semantics; the LLM-facing explanation distinguishes them.

Counterfactual tests remove individual publication clauses, require only the corresponding named
assertion to fail, restore the clause, and verify all groups pass. A whitespace-only variant supplies
a formatting control. These observations establish the guards' discrimination on those mutations,
not exhaustive semantic adequacy or model compliance.
