# ADR 0068: Surface upstream transcript elision and recorded artifact references

## Status

Accepted — bounded recognition, opt-in JSON `events --diagnostics`, and the
separate explicit artifact reader are implemented with the admission and
provenance constraints below.

## Context and evidence

On 2026-09-22, an inspected retained MCP result reported `omitted: true` and a
`fullResultPath`. Reading that exact local artifact recovered the projected
`showAgent` definition. Complete transcript export had preserved the retained
wrapper, not the omitted body. A separate reported incident involved the literal
`[MCP text output truncated: … 198.7 KiB]` marker; that incident's original payload
was not independently inspected. These are distinct upstream loss signals, not
nn's own display clipping or pagination.

The inspected `projectLedger` retains raw message/block payloads. `ledgerText`
decodes strings and content blocks, including nested `tool_result` content, but
does not recursively decode JSON embedded within arbitrary text. Neither fact
establishes repository-wide absence of other helpers.

The inspected Pi sidechain path is deliberately specialized:
`piBackgroundLocators` extracts background-agent paths, `showAgent` checks the
requested agent ID, and `readOwnedPiSidechain` validates the path before selecting
owned records. `validatePiSidechainPath` resolves symlinks, requires
`tasks/<agentID>.output`, and requires a `pi-subagents-*` ancestor. This is layout
validation, not a pinned trusted-root guarantee. It is not the admission contract
for arbitrary MCP result files.

## Decision

### 1. Report upstream loss without changing retained evidence

Add an opt-in JSON event diagnostic projection (`events --diagnostics`, implemented).
It reports recognized upstream truncation/elision separately from display clipping
and transport segmentation. Ordinary selection, filtering, raw payloads, exports,
and default output remain unchanged. Diagnostics never imply that absent bytes
were recovered or that a transcript is source-complete.

A diagnostic carries its source event ID, recognizer/version, retained field or
text-block location, kind (`upstream_elision` or `upstream_truncation`), and any
recorded artifact reference. No recognized signal means `not_detected`, not
`complete`. Malformed or over-budget recognition is explicitly `uninspected`.
Recorded byte counts remain producer-reported counts, not measured loss.

### 2. Recognize narrow envelopes, not arbitrary embedded JSON

Initial recognizers cover the retained MCP adapter elision shape and the explicit
MCP text truncation marker. An elision recognizer requires a tool-result context,
`omitted: true`, and the adapter-shaped fields, rather than treating any occurrence
of `fullResultPath` as authority. It may decode one JSON text envelope and the
known `ok/data` wrapper used by mcpScript. Apply fixed byte, nesting, and match
limits before decoding; disclose when a limit prevents inspection.

Do not recursively search arbitrary JSON strings, user/assistant prose, or all
fields for paths. This bounded envelope recognition is not the deferred generic
nested-payload extractor. A recognized marker without a recorded path still
produces a warning, never a guessed location.

### 3. Preserve artifact references as untrusted recorded data

A reference includes the verbatim recorded path and its exact event/field
provenance. Merely listing diagnostics performs no artifact stat, directory scan,
network request, or file read. Do not treat a Pi `fullOutputPath` as an MCP
`fullResultPath`, or change sidechain ownership/resolution behavior.

References are scoped to a retained event snapshot, not globally stable positional
event IDs. Raw paths must be safely quoted/escaped in readable output; never render
a transcript-provided path as an executable shell command.

### 4. Make artifact reading a separate explicit operation

Implemented interface: `transcript artifact read <session> <agent-id> --event
<id> --snapshot <hash> --finding <one-based-index> --allow-root <absolute-root>`.
The snapshot must come from the default-facet, no-payload `events --diagnostics
--event <id>` projection, not a filtered or payload projection. The selected
finding must carry a recorded artifact path; callers never supply a path as
authority. Reading is never triggered by listing, search, export, or diagnostic
enrichment. No editor, browser, shell execution, network fetch, or recursive
artifact-following occurs.

Only absolute local paths lexically beneath the explicitly approved root are
eligible. Reject parent traversal, unsupported URI schemes, non-regular files,
and symlink traversal in the root, intermediate components, or leaf. Darwin/Linux
use descriptor-relative no-follow traversal/opening so a path swap cannot escape
the approved descriptor chain. A filename prefix alone is not an authorization
boundary. Other platforms fail closed; a symlinked root spelling must be replaced
with an explicitly approved symlink-free spelling.

Return at most 16,384 acquired bytes as base64 plus event/snapshot/reference
provenance, recorded path, measured byte count, digest of acquired bytes, and
explicit `complete`, `partial`, or `unstable` coverage. Do not splice
artifact content into the original event or overwrite the retained payload. A
successful read is a new acquisition of current artifact bytes: without a recorded
digest it is not proof of identity with historical output. Oversized reads disclose
partial coverage and never label a prefix digest as a whole-file digest.

Distinguish missing/expired (absence cannot determine which), denied, changed
snapshot, unsupported file type, read failure, and partial acquisition. These
outcomes do not turn the original tool result into a tool failure. Artifacts may
contain sensitive source; no automatic notebook capture or network publication.

## Compatibility and implementation boundary

Use a separate diagnostic model rather than mutating raw ledger payloads. Existing
ordinary output and event IDs remain unchanged. Opt-in diagnostic paging snapshots
must bind recognizer version/options and selected retained evidence; artifact
contents must not influence the transcript snapshot. Artifact acquisition has its
own provenance. Existing output limits remain enforced after enrichment.

Known source seams are `transcript_events_projection.go` (event projection),
`transcript_event_context.go` (searchable result content), and transcript event
renderers. This ADR does not assert server-reported calls among those seams.
The artifact reader must be separate from the Pi sidechain resolver.

## Alternatives and consequences

- Raw export plus external JSON tools remains supported, but leaves recovery cues
  easy to miss. Opt-in diagnostics add guidance without replacing that escape hatch.
- Automatic artifact opening is rejected: recorded paths are untrusted, potentially
  stale, and may reference sensitive files.
- Reusing the Pi resolver is rejected: its layout/agent ownership contract answers a
  different question and would reject ordinary MCP artifacts or weaken sidechains.
- General recursive extraction is deferred: it would require a separate contract
  for depth, selection, duplicate content, ambiguity, and provenance.
- Recognition is necessarily partial and versioned. Unrecognized adapters remain
  raw retained evidence, not implicitly complete results.

## Delivery and verification

Recognition, JSON diagnostics and the separate reader are implemented. Focused
security tests cover snapshot-before-open, root/path denial, symlink components,
FIFO, bounded bytes, acquired-prefix digest, detectable change and a swap adversary;
reader tests and the owning-package suite passed on Darwin, with Linux/Windows
cross-compilation. Tests do not prove every concurrent race absent or historical
artifact identity. Diagnostic listing still has no artifact reader; no dynamic
artifact-open spy has been run for the listing path. The delegated reader writer
missed its child-local Bar sequencing gate; parent review retained its code as
bounded evidence, corrected typed unsupported-platform behavior under a new phase,
and independently verified the integrated implementation.

See [regression-test plan](../transcript-artifact-diagnostics-test-plan.md).
