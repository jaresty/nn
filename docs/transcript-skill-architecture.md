# Transcript skill documentation ownership

This is a maintainer map, not an additional runtime prerequisite or a new CLI architecture.
The runtime entry point is `nn skills get nn-transcript`; applicability is available through
`nn skills get nn-transcript --list-references`.

## Four logical layers

| Layer | Job | Owner |
|---|---|---|
| Entry | Route the user's intent without forcing an unrelated workflow | `skills/nn-transcript/SKILL.md` |
| Shared contracts | Evidence authority and shared visual grammar | Core |
| Shared contracts | Direct bounded reads, target resolution, scope/sample separation, retained state, Back/Refresh | `interaction` |
| Shared contracts | Suggested actions and approval-only capture | `actions` |
| Workflows | Default bounded observation including ROOT; sampling and continuity | `observe` |
| Workflows | Question-shaped evidence selection across streams | `investigate` |
| Capabilities + optional view | Discover and interpret a conversation candidate page | `discovery` |
| Optional views | Browse authenticated hierarchy or render a semantic layout | `navigate` |
| Workflows | Enter a readable room, expand evidence, return to source view | `rooms` |
| Workflows | Review queues, evidence-guided Find, correction proposals | `review` |
| Workflows | Select and apply spatial/question lenses | `lenses` |
| Workflows | Sample sessions, investigate, and support a cross-session claim | `patterns` |
| Workflow + capability | Bounded attention discovery, evaluation recipe, policy interpretation | `attention` |
| Capabilities | Literal/regex search, input scope, provenance, limits and errors | `search` |
| Capabilities | Targeted projections, text/events, payloads and transport | `events` |
| Capabilities | Assignment and recent-work evidence | `context` |
| Capabilities | Native usage/tool/timing reductions | `summaries` |
| Capabilities | Recorded launch/return and lifecycle meanings | `handoffs` |
| Capabilities | Unknown-schema diagnosis and reconstructed-join validation | `recovery` |

These are responsibility categories, not directory boundaries. A reference may combine a workflow
and a capability when they serve the same immediate job; attention is intentionally not split merely
to make the table uniform. The core's shared rules remain compact and binding; reference summaries
and examples do not create alternative owners or exceptions.

## ADR-0048 interaction replacement

Observe and investigate are the main activities; hierarchy and queues are optional presentations.
Ordinary bounded reads execute before contextual choices. Standing-approval/Office defaults are retired,
not layered underneath the new core. Capture remains separately approved through actions. Native
retrieval, ownership, evidence and replay contracts retain their existing owners.

The initial observation recipe uses existing commands: bounded conversation discovery, ROOT plus up to
two direct children per selected conversation, then bounded event tails. It does not imply exhaustive
coverage, a global mixed-stream ranker, bounded source processing or a fairness scheduler. Native fixture
execution establishes command compatibility, not conversational usability.

## Search and recovery extractions

A person finding a string needs search mechanics, not session sampling and behavioral synthesis.
Core therefore routes text lookup directly to `search`; `patterns` loads it only to locate candidates.
The search owner points back to patterns only when the task actually concerns recurrence.

Unknown-schema recovery is useful within one session as well as a cohort. Core and patterns dispatch
`recovery`; only recovery defines the four join-validation assertions. Native supported schemas stay
on the native CLI path. Reconstruction does not invent missing authority.

The patterns workflow remains responsible for selecting sessions, choosing evidence relevant to the
question, investigating a dimension, synthesizing a bounded cross-session claim, and proposing capture.
It distinguishes the session sampling unit from the evidence actually retrieved. It does not make
partial evidence sufficient for a whole-session claim or establish absence outside inspected windows.
Capture permission remains owned by actions, not by completing the patterns workflow.

## Maintenance rules

- Route by intent, not one-file-per-command. A direct lookup must not require an Office tour.
- Give each detailed rule one normative home. Elsewhere, summarize its purpose and link to its owner.
- A workflow selects appropriate evidence; a capability defines what retrieval returns and omits.
- Use CLI help for the exhaustive option inventory. Skills explain choices, interactions, authority,
  coverage, and failure handling rather than duplicating every help line.
- Follow a move through core dispatch, applicability metadata, importing workflows, and served-reference
  tests. Preserve useful pointers at previous entry points; do not leave a second normative copy.
- Keep native command behavior unchanged during documentation moves. Retain source identity,
  unknown/error distinctions, complete transport requirements, and authorization boundaries.
- Maintain the core's existing 200-line / 14,000-byte bound instead of relaxing it to fit new details.

## Verification boundary

`TestTranscriptDocumentationOwners` exercises embedded CLI-served owners, direct core/listing routes,
selected contract terms, removal of moved rules from patterns, and the sampling clarification.
`TestTranscriptSkillLazyDispatch` retains the compact-core bound and existing reference checks.
Native transcript tests separately protect command semantics. `TestTranscriptTracerNativeRecipe`
executes the initial ROOT/direct-child recipe on Pi, SDK-layout and inline Claude fixtures.
`TestTranscriptTracerPublication` checks served commitments; controlled conversational replay must be
reported separately, with exact executed inputs, outputs, notebook effects and limitations.

The content checks are deliberately narrow publication tripwires. They neither prove arbitrary
paraphrases equivalent nor demonstrate that an LLM follows the workflow. Review the normative text
when changing wording, and evaluate conversational effectiveness separately. Do not describe these
checks as a comprehensive semantic documentation audit.
