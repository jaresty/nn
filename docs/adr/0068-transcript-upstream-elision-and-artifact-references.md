# ADR 0068: Surface upstream transcript elision and recorded artifact references

## Status

Proposed — recognition and opt-in JSON `events --diagnostics` are implemented;
the explicit artifact reader and final security/interface acceptance remain proposed.

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

Proposed interface: `transcript artifact read` identifies the session, agent,
exact event, diagnostic snapshot, and reference, and requires a caller-supplied
absolute `--allow-root`. Reading is never triggered by listing, search, export,
or diagnostic enrichment. No editor, browser, shell execution, network fetch,
or recursive artifact-following occurs.

Only absolute local paths resolving beneath the explicitly approved root are
eligible. Reject parent traversal, unsupported URI schemes, non-regular files,
and symlink traversal for the initial implementation. Use descriptor-relative,
no-follow traversal/opening so a path swap cannot escape the approved root between
validation and reading. A filename prefix alone is not an authorization boundary.
Fail closed where this admission contract cannot be implemented.

Return bounded bytes plus event/snapshot/reference provenance, recorded path,
measured byte count, digest of acquired bytes, and explicit coverage. Do not splice
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

Opt-in diagnostic recognition and JSON event reporting are implemented and covered
by focused and owning-package tests. The reader remains unimplemented: require its
admission and provenance tests before implementation. Keep this ADR Proposed until
the reader's interface and safety contract are accepted. Diagnostic listing has no
artifact reader; static review supports no artifact I/O, but no dynamic artifact-open
spy has been run for the listing path.

See [regression-test plan](../transcript-artifact-diagnostics-test-plan.md).
