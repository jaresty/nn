# Transcript Office baseline A — implementation and evaluation boundary

## Implemented

ADR-0043's interaction-only baseline is implemented in the co-versioned skill. No new runtime UI,
activity metric, invocation classifier, or retrieval primitive was added.

Rule ownership:

| Owner | Responsibility |
| --- | --- |
| `references/interaction.md` | Explicit target/action resolution, inspection envelope, retained view records and Back |
| `references/rooms.md` | Neutral readable room entry; optional spatial lens |
| `references/review.md` | Evidence-guided Find and authoritative review population |
| `references/lenses.md` | Lens semantics; contextual operands versus genuine ambiguity |
| `references/actions.md` | Up to three useful recommendations and approval-only capture |
| `references/discovery.md` | Canonical row selection and Open/Open… |
| `references/navigate.md` | Authenticated office topology and domain dispatch |
| `SKILL.md` | Compact dispatch and shared evidence/visual grammar |

The rules agree directly. There is no override layer that leaves contradictory owners active.

## Automated checks

`TestTranscriptInteractionContract` checks embedded publication and dispatch from all owners.
`TestTranscriptInteractionSingleOwnerRules` protects nine explicit commitments: visible action wins
against stale selection, monotonic budget, navigation-state restoration, sample/authorization
separation, readable entry, up-to-three actions, contextual comparison operands, Find as intent, and
capture approval. The original nine commitments were mutation-tested. Updated restoration guards also
reject mandatory manual file persistence and promises of identical prose; static tests do not establish
native persistence or model determinism.

`TestTranscriptInteractionNativeBaseline` runs existing native commands against a deterministic fixture:
readable review -> source event ID -> complete exact-event payload. It checks that the failure remains
addressable and its evidence survives the transition. It is not an LLM navigation test.

Legacy discoverability tests now require fallback controls and up-to-three suggestions instead of
mandating fixed direct menus. Other pagination, ownership, cache, source-change, and native rendering
contracts remain under the existing suite.

## Conversational evaluation — still required

Static text tests cannot establish model compliance, semantic quality, or usability improvement.
The prior live interaction identified a file-lookup failure via existing review/exact-event commands;
that motivates this baseline but is not a post-change controlled comparison.

Before expanding the design, retain actual prompts, visible views/action bindings, tool calls, evidence
identities, and outputs for the following replay cases. Use the same fixed evidence for the old menu/
recent-tail condition and baseline A. Do not count authored expected responses as observed results.

| Case | Required observation |
| --- | --- |
| Selected A, displayed Inspect B | Bare Inspect retrieves B; explicit Inspect A retrieves A |
| Empty filter, prior room retained | No silent clearing or borrowed selection; explicit clear-and-inspect works |
| Sticky conversation selection | Open echoes and opens the selected row; Open… opens a chooser |
| Truncated failure | Authorized exact result exposes the cause without an extra filter menu |
| Follow-up within envelope | Retrieval proceeds without redundant confirmation and consumes budget |
| Envelope exhausted / another room needed | Stop and request renewed scope; no implicit authorization |
| Back after inspecting newer evidence | Restore navigation/evidence state, with no fresh retrieval or budget refund; no identical-prose promise |
| Compaction / cache expiry | Missing conversational state is disclosed; expired evidence is not silently refreshed; no manual persistence requirement |
| Capture suggestion/cancel | Proposal only; cancellation restores view; generic assent does not write a note |
| Read-only work / ordinary verification | No unsupported productivity or blocker claim |
| Recovered failure / useful discovery | Relevant next action with evidence and limits, not a compulsory correction |

Judge findings against inspected source evidence: useful means resolving a declared uncertainty and
recommending a relevant next action; a false alarm asserts more than that evidence supports. Record
unnecessary follow-up, source-output/context consumption, command and user-turn counts, and latency.
No identity, approval, scope, or Back regression is acceptable. Predeclare a measurable bottleneck
before trying any new selector or metric, then compare that increment against A rather than attributing
A's interaction benefits to added machinery.

## Resume

Install the current binary and load `nn skills get nn-transcript --reference interaction` before the
next Office interaction. Retain concise conversational state; do not create view files. Manual state
persistence proved costly in use and has been removed. The next justified native primitive is specified
in [the proposed Office state design](transcript-office-state-design.md), not yet implemented.
Automatic handoff refresh overlays and continuous watch remain separate increments; refresh is explicit.
