# ADR-0071: Persist validated transcript search projections in the rebuildable index

**Status:** Accepted
**Date:** 2026-10-21

## Context

`nn transcript search` currently opens every selected transcript and JSON-decodes every record for every query. Default discovery spans Claude, Codex, and Pi roots, so query-independent decoding dominates runtime even when `--limit` is small. Context windows then re-open matched files through the ledger reader.

Transcript files must remain authoritative. The SQLite database at `~/.config/nn/index.db` is a derived cache and may be deleted or rebuilt at any time.

## Decision

Add transcript-specific tables to the existing SQLite index. Cache one validated search projection per searchable transcript event, including its canonical source path, parsed-record ordinal, session and event provenance, timestamp, role, meaningful content, a pre-lowercased meaningful-search value, and message digest. Raw message payloads are not persisted.

A file record stores canonical path, byte size, and nanosecond modification time. New or observably changed files are scanned and validated completely. Their file row and event rows are replaced in one transaction only after the complete scan succeeds. Unchanged files perform no transcript-record JSON decoding.

Before each search, cached paths whose authoritative files no longer exist are removed. Queries consider only the files selected by the current command, so cached rows from an unselected but still-existing root cannot affect results.

The existing streaming search remains the semantic oracle and fallback test surface. Ordinary meaningful-content substring searches execute inside SQLite against only the command's selected-path set and stop at `limit + 1`. Meaningful-content regex searches scan validated cached projections to preserve Go regexp semantics. `--raw` searches use the authoritative streaming scanner because persisting raw message JSON would create a multi-gigabyte duplicate corpus. All paths preserve the same agent, time, ordering, limit, truncation, excerpt, and provenance semantics.

`-C` context remains source-backed. A cached match carries its parsed-record ordinal and SHA-256 message digest; the existing context reader must find the same attributable source record before rendering context. A source mutation between search and context therefore fails rather than attributing cached text to changed source bytes.

Cache schema versions are local to the transcript-search tables. An incompatible transcript schema is dropped and rebuilt. SQLite corruption or an unsuccessful source refresh fails the command without publishing cache-derived matches for the failed operation.

## Consequences

- The first search pays approximately the existing full-scan cost plus transactional inserts.
- Warm searches avoid JSON decoding unchanged transcript files and scan compact SQLite projections instead.
- Metadata identity assumes normal filesystem behavior: content writes update size or nanosecond mtime. Explicit timestamp restoration after an equal-size in-place rewrite is outside automatic invalidation guarantees.
- Meaningful text and its normalized search value are duplicated in `index.db`; raw JSON is not.
- `--raw` searches retain their prior full-scan cost.
- The cache remains disposable and does not become notebook or transcript truth.

## Verification

Persistent tests cover cold population, warm reuse without record decoding, a same-snapshot substring parity matrix against the uncached oracle, changed-file invalidation, deleted-file pruning, transactional rejection of malformed changed input, and source-backed context mutation detection.
