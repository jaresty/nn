# ADR-0015: Session Debrief as Friction Log and Protocol Discovery Engine

## Status

Proposed

## Context

The current session debrief (`nn-session-debrief`) captures what notes were created or updated during a session. It does not capture *where the agent had to course-correct* — misapplied tools, repeated mistakes, or systemic gaps that a protocol could prevent.

Four related ideas were proposed:

1. **Friction log in debrief**: Session summary should collect moments of correction and surface them as candidates for protocol creation.
2. **Observations as an intermediate type**: Friction candidates should be written as `observation` notes (an existing nn type), not draft protocols — the debrief agent cannot validate whether a friction moment warrants a protocol, so it captures the raw fact without overclaiming.
3. **`nn review` as the human promotion gate**: A dedicated review pass surfaces unreviewed observations, lets the user promote them to protocols, discard them, or link them as evidence to existing protocols.
4. **Error-handling behavior as a virtual protocol**: Error-handling behavior is injected as a `virtual-nn-error-handling` entry in the session-start hook alongside `nn-capture-discipline`. The error hook is trimmed to a short pointer rather than eliminated — its event-binding is kept, its inline behavior replaced with a reference to the virtual protocol.

## Decision

### 1. Debrief adds a Friction Review pass

`nn-session-debrief` scans the transcript after a session ends (running as an agent, no user present) for course-corrections: tool retries, user corrections, unexpected failures, repeated mistakes. For each friction moment it asks: *was there a systemic gap that recurred or could recur?* If yes, it writes an `observation` note tagged `friction-candidate`.

### 2. Friction candidates are `observation` notes tagged `friction-candidate`

The debrief agent writes `nn new --type observation --tags friction-candidate` notes with:
- A description of the friction moment
- The transcript context where it occurred

`observation` is an existing first-class nn note type. Observations are not behavioral constraints — they are raw, unvalidated notices. Writing one does not imply it should become a protocol.

Reviewed observations (promoted, discarded, or linked) are updated with `--tags reviewed` so they do not resurface in future `nn review` passes.

### 3. `nn review` promotes observations to protocols

`nn review` queries `nn list --type observation --tag friction-candidate --json` (excluding `reviewed`-tagged notes) to surface unreviewed friction observations. For each, the reviewer chooses:

- **Promote**: create a new `nn new --type protocol` note derived from the observation, set `applies_when` and `status: draft`, then link the observation as evidence (`nn link <obs-id> <protocol-id> --type supports`). Update the observation with `--tags reviewed`.
- **Discard**: `nn update <id> --tags reviewed` — no further action.
- **Link**: link to an existing protocol as supporting evidence (`nn link <obs-id> <protocol-id> --type supports`), then tag as reviewed.

Promotion creates a new protocol note rather than mutating the observation's type, preserving the observation as a traceable source.

`nn review` also performs a conflict-detection pass over any newly promoted draft protocols before they reach `permanent`. The conflict-detection algorithm is out of scope for this ADR and will be addressed separately.

### 4. Debrief trigger

`nn-session-debrief` is triggered via the existing `Stop` hook (session-end event). No new hook is added.

### 5. Error-handling behavior moves to a virtual protocol; the error hook is eliminated

The `nn-capture-discipline` protocol already demonstrates the virtual protocol pattern: behavior injected at session start via `nn show --global` with a `virtual-<id>` identifier, no real note required. Error-handling behavior follows the same pattern — a `virtual-nn-error-handling` entry is added to the session-start hook's injected output. This eliminates the separate error hook: one hook, one `nn show --global` call, all protocols loaded together.

Virtual protocols live in the hook configuration text. To update error-handling behavior, edit the injected block in the session-start hook — no hook reconfiguration or new note required.

The existing error hook is not eliminated — it is trimmed. Its event-binding is retained so it still fires on error events, but its body is replaced with a short reminder to consult `virtual-nn-error-handling` (already loaded at session start). This keeps the trigger mechanism intact while moving the behavioral content to one place.

## Open Questions

- **Conflict-detection algorithm**: What constitutes a conflict between two protocols, and how should `nn review` detect it? Deferred to a follow-on ADR or implementation spike.
- **`nn review` current scope**: This ADR adds friction-candidate surfacing to `nn review`. The existing responsibilities of `nn review` should be documented to confirm these additions are extensions, not replacements.

## Consequences

- No new CLI types are needed; `observation` already exists.
- `--tag` filtering (`nn list --tag friction-candidate`) is the query mechanism; no `--field` flag is required.
- Session debriefs become slightly longer (friction pass) but produce lower-stakes, auto-created notes rather than premature constraints.
- The two-trigger design (debrief writes observations, `nn review` promotes to protocols) means protocol creation is never blocked by user availability, and no protocol is enforced before human review.
- Promotion creates a new protocol note linked to the source observation, preserving audit trail without mutating note types.
- `nn review` gains a new responsibility: surfacing `friction-candidate` observations alongside its existing draft-promotion flow.
- Error hook is trimmed to a pointer; its event-binding is retained but its body is replaced with a reminder to consult the virtual protocol loaded at session start.
