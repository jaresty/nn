# ADR 0066: Normalize transcript relations before discovery projection

## Status

Accepted

## Context

`nn transcript ls` exposes provider-neutral discovery fields such as
`conversation_kind` and `owner_session`, but currently classifies sidechains from
filesystem layout alone. Pi records cross-session parentage in a session header's
`parentSession` field. SubagentWorkflow child sessions are stored beside their
parent rather than under a `pi-agent-*` directory, so the layout heuristic
misclassifies those children as root conversations and leaves `owner_session`
empty.

Embedding Pi's `parentSession` field directly in listing and filtering logic
would fix the immediate symptom while coupling generic discovery behavior to one
provider schema.

## Decision

Transcript providers normalize their schema-specific relationship evidence into
a shared internal relation model before discovery projects it:

- relation kind: root or sidechain;
- owner session identity when authenticated;
- parentage authority: recorded, heuristic, or unavailable.

Recorded parent metadata takes precedence over filesystem heuristics. A recorded
parent is authenticated only when its referenced transcript exists and its
recorded session identity can be read. Generic discovery derives
`conversation_kind` and `owner_session` from the normalized relation and uses the
same relation for filtering and cursor validation.

Provider-specific extraction remains behind the transcript schema boundary. The
existing `pi-agent-*` directory convention remains a Pi compatibility fallback;
it can classify a sidechain but cannot fabricate an owner.

## Consequences

- Flat child transcripts with valid recorded parentage appear as sidechains with
  an owner session.
- Missing or malformed parent references do not create authenticated ownership.
- Existing Pi agent execution directories remain discoverable as unowned
  sidechains.
- Other providers can add recorded cross-session parentage without changing
  transcript listing or filtering.
- Discovery may read bounded session-header metadata while building inventory.
