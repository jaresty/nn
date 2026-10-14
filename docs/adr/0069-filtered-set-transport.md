# ADR 0069: Scope-preserving filtered-set transport

**Status:** Accepted
**Date:** 2026-10-14

## Context

Weekly review can identify a semantic cleanup set such as “expired daily notes,” but a later workflow can accidentally broaden that scope to all expired notes, recover the intended subset with external JSON filtering, and exceed the agent transport limit. Existing `nn list` filters already compose, while its envelope is search-only and its result transport has no stable continuation. `nn graph bodies` already establishes a lossless, snapshot-bound, sub-50KB paging contract for graph traversals, but not arbitrary selected IDs.

## Decision

1. Embedded workflows must preserve every selected predicate in inventory and mutation commands. They must display the resolved query, preview cardinality, and partition large sets before promising a full-set semantic review.
2. The global CLI protocol carries only the cross-cutting scope-preservation invariant; specialized cleanup procedure remains in skills.
3. `nn list --json --envelope` applies to any filtered query. Its metadata distinguishes returned results from all matching results.
4. `nn list --count` emits only the number of notes matching all active filters and cannot be combined with output-shaping or continuation flags.
5. Filtered list pagination is opt-in, snapshot-bound, deterministic, and encoded below the existing 48KB safe transport ceiling. Later pages require the snapshot returned by page 1; a changed notebook or query rejects continuation.
6. Lossless paginated body transport accepts an explicit arbitrary ID set in addition to graph traversal selection. Selection modes are mutually exclusive, and every selected stored body can be reconstructed exactly from ordered segments.
7. Existing invocations retain their current output when none of the new flags or selection modes is requested.

### Filtered-list transport contract

Pagination is explicitly selected with:

```text
nn list <filters> --json --envelope [--fields FIELDS] [--rich] [--full]
        --page N [--snapshot SHA256]
```

Page 1 requires no snapshot. Every later page requires the snapshot returned by page 1, and a supplied snapshot must match on any page. Each response is one JSON object:

```json
{
  "snapshot": "<sha256>",
  "page": 1,
  "pages": 3,
  "next_page": 2,
  "result_count": 120,
  "total_matching": 340,
  "query": "<present for search results>",
  "results": []
}
```

`page` is one-based and `next_page` is zero on the final page. `result_count` describes the current page; `total_matching` describes the complete filtered set. Pages preserve the command's existing deterministic result order, and concatenating `results` from pages 1 through `pages` reconstructs the exact complete serialized projection without duplicates or omissions.

The snapshot hashes a versioned encoding of the normalized selection/output request and the complete serialized result projection. A changed filter, search, sort, field projection, output shape, or relevant notebook content therefore rejects continuation rather than mixing result sets. Every response, including its final newline, is at most 48,000 bytes. A single serialized result too large to fit rejects with guidance to narrow fields or disable `--full`; records are never silently truncated by pagination.

Pagination requires `--json --envelope`. It is incompatible with `--limit`, `--count`, and `--show-first`; `--snapshot` without `--page`, pages below one, missing later-page snapshots, stale snapshots, and out-of-range pages fail explicitly.

Arbitrary-set body retrieval is selected with `nn graph bodies --ids ID1,ID2,...`. IDs are deduplicated and ordered deterministically; unknown IDs fail rather than being omitted. `--ids` is mutually exclusive with `--focus` and traversal flags. The existing graph-body envelope, 48,000-byte ceiling, snapshot continuation, UTF-8-safe segmentation, and exact reconstruction contract remain unchanged.

## Consequences

- Agents can preserve semantic scope without `jq`-based recovery from broad supersets.
- Large result sets gain honest counts and stable continuation rather than relying on harness spill files.
- Snapshot and page-packing concepts are shared with the established graph-body contract where practical.
- New modes add validation surface, but legacy scripts remain compatible.
