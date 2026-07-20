# ADR-0018: nn tee command

**Status**: Proposed  
**Date**: 2026-07-20

## Context

`nn read` surfaces related notes when reading a file; `nn grep` annotates code matches with related notes. Both require a path argument — neither handles arbitrary piped content.

Assumption: when a user pipes external content through a pipeline (e.g., `curl <url> | nn tee | jq .`, or `ps aux | nn tee`), they want related notes surfaced without interrupting the pipeline. This assumption rests on the constraint that stdout must remain byte-for-byte identical to stdin — any note output on stdout would corrupt the pipeline.

Assumption: stderr is the correct output stream for related notes because it is conventionally used for annotations and diagnostics that should not affect downstream pipeline consumers.

## Decision

Add `nn tee` as a cobra subcommand that:

1. Reads stdin in full (or up to a content limit), writes it byte-for-byte to stdout.
2. Runs a BM25 search against the piped content using the same internal search machinery as `nn list --search`.
3. Prints matching related notes to stderr in the same format as `nn read`'s `## Related notes` footer.
4. Exits 0 when stdin is processed without error, regardless of whether any notes are found.

Content is truncated to a configurable limit (default: 4096 bytes) before BM25 search to avoid index overload on large inputs. The truncation limit applies to the search query only — stdout receives the full content.

The command reuses `internal/index` BM25 search, not a separate implementation.

## Consequences

- `nn tee` is pipeline-transparent: downstream commands receive stdin unchanged.
- Related notes surface on stderr — visible in terminal but not captured by `>` redirection unless `2>&1` is used.
- Large piped inputs (log files, bulk curl output) are searched on a truncated window; the truncation is noted in the stderr header.
- No note is created or modified; `nn tee` is read-only.

## Falsifiable Gates (must pass before implementation)

Each gate names a literal string that must appear in a tool-executed result before the governed implementation phase proceeds.

### Gate 0 — cobra command registered
**Governed behavior**: `nn tee` is recognized as a valid subcommand.  
**Assertion artifact**: `go test ./cmd/nn/... -run TestTeeCommandRegistered`  
**FAIL literal**: `FAIL` with `TestTeeCommandRegistered`  
**PASS literal**: `ok`  
**Gate condition**: the literal string `ok` appears in a `go test` result above the Phase 1 implementation action.

### Gate 1 — stdin passthrough
**Governed behavior**: stdout contains exactly the bytes read from stdin.  
**Assertion artifact**: `go test ./cmd/nn/... -run TestTeePassthrough`  
**FAIL literal**: `FAIL` with `TestTeePassthrough`  
**PASS literal**: `ok`  
**Gate condition**: the literal string `stdout contains exactly the bytes` appears as a comment in the test, and `ok` appears in the run result above Phase 2 implementation.

### Gate 2 — stderr note output
**Governed behavior**: related notes are printed to stderr, not stdout.  
**Assertion artifact**: `go test ./cmd/nn/... -run TestTeeRelatedToStderr`  
**FAIL literal**: `FAIL` with `TestTeeRelatedToStderr`  
**PASS literal**: `ok`  
**Gate condition**: the literal string `Related notes:` is asserted in the test to appear on stderr, and `ok` appears in the run result above Phase 3 implementation.

### Gate 3 — pipeline exit code
**Governed behavior**: exits 0 when stdin processes without error.  
**Assertion artifact**: `go test ./cmd/nn/... -run TestTeeExitCode`  
**FAIL literal**: `FAIL` with `TestTeeExitCode`  
**PASS literal**: `ok`  
**Gate condition**: `ok` appears in the run result above any integration test.

## Implementation Phases

Each phase is scoped to exactly one governed symbol:

1. **`newTeeCmd`** — register cobra command, wire `RunE` to a stub that returns nil. Gate 0 must pass.
2. **`runTee` (passthrough)** — implement stdin→stdout copy using `io.Copy`. Gate 1 must pass.
3. **`runTee` (BM25 + stderr)** — call `index.Search(content[:limit])`, write results to stderr. Gate 2 must pass.
4. **`runTee` (exit code)** — verify `os.Exit` path on stdin error only. Gate 3 must pass.
