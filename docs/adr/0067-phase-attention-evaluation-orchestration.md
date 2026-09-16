# ADR 0067: Separate attention evaluation orchestration into phases

## Status

Accepted

## Context

`buildAttentionPolicies` validates requests, resolves and classifies transcript
paths, acquires provider-specific records, constructs agent trees, evaluates
attention signals, assembles response models, and retains snapshots. The function
is therefore an articulation point with dependencies across transcript discovery,
capture, evaluation, presentation, and persistence.

Moving the whole workflow into `internal/attention` would invert the existing
boundary: that package owns policy evaluation, while transcript schemas, captures,
trees, and retained command output remain command-domain concerns.

## Decision

Keep the existing attention entry points and command-owned data types, but divide
`buildAttentionPolicies` into three explicit phases:

- validate the selected agents, task overrides, and bundled policies;
- prepare a canonical transcript session, provider capture, agent index, and
  handoff context;
- evaluate selected rooms and assemble retained evidence.

`buildAttentionPolicies` coordinates these phases and persists their completed
result. Validation and error ordering remain unchanged, as do room ordering,
provider-specific acquisition, output schemas, and snapshot behavior.

## Consequences

- The orchestration function no longer directly coordinates every transcript and
  attention dependency.
- Each phase has one reason to change and can be tested independently later.
- Provider-specific transcript acquisition stays outside `internal/attention`.
- Existing callers and compatibility helpers require no changes.
- This is a structural refactor and does not alter CLI behavior or output.
