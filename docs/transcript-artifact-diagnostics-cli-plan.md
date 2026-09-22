# Transcript artifact diagnostics: increment-2 CLI contract (approved direction)

Status: implementation handoff, not shipped. ADR 0068 remains Proposed. Increment 1's pure recognizer is accepted; explicit artifact reading is excluded.

## Interface

Add opt-in `nn transcript events ... --diagnostics` for JSON ordinary event pages and `--all`. Support existing `--select`, `--payload`, exact `--event`, time/error/last filters, and search/context windows without changing their selection or order. Reject `--format text`, `--summary`, `--at`, `--include-assignment`, and `--include-errors` when combined with diagnostics, before acquisition. Do not silently drop diagnostics on a special route. Existing outputs, raw payloads, exports, and snapshots without the flag must remain unchanged.

Each selected complete event gets a `diagnostics` object with the accepted `nn.transcript.artifact-diagnostics/v1` schema (`version`, `event_id`, `status`, non-null `findings`, `inspection_complete`, non-null `issues`). This is recorded-signal detection, not source completeness or artifact availability. No artifact path may be opened, statted, normalized, or followed. Do not change `--payload`'s outward meaning: the `payload` field appears only if explicitly requested, even though diagnostics inspect it internally. The page header's `payload` remains the caller's requested boolean.

## Source-qualified seam and hazard

`projectLedger` can retain payload for internal inspection. `newTranscriptEventsCmdUsing` routes JSON output through ordinary, query, and window builders. `buildQueriedLedgerPage` copies source events then removes unselected facets but does **not** remove hidden payload. `buildWindowLedgerPage` copies events and removes payload when not requested. Consequently, simply passing `payload || diagnostics` to projection and attaching diagnostics before the builders leaks payload in the query route. This is a source observation, not a runtime call-edge claim.

Diagnose the **final selected event set**, not unrelated events, after query/window/exact-event selection; retain the full evidence projection needed by existing matching and receipts. Guarantee final output stripping on **every** route, including query copies, regardless of facet selection. Never mutate input records or an unflagged projection. Avoid a second independent selection implementation: supply an opt-in post-selection decoration/finalization hook or an equivalent explicit seam in the existing builders, and verify selection equivalence. Diagnosing before selection is permissible only if the output, bounds, and hidden-evidence snapshot still meet the same contract; do not claim an inspected corpus from the selected result alone.

## Snapshot and transport

Keep the existing snapshot algorithm byte-for-byte for unflagged calls. For diagnostic calls, domain-separate the hash and bind the caller's flag/options, recognizer version, selected diagnostic projection, and the retained payload evidence used to produce it even when hidden from JSON. For query/window routes, preserve their whole-evidence `ledger_snapshot` receipts and bind the relevant diagnostic evidence without weakening existing full-history change detection. Changed hidden evidence that yields the same diagnostic must still invalidate a pinned continuation. Changed diagnostic options, recognizer version, or query/window selection also reject stale/mismatched continuations. A later page requires the page-1 snapshot. Do not put internal evidence digests or hidden payload in the JSON output.

Decorate complete events **before** existing encoding/fragmentation. Fragments are only transport; reconstruct all ordered segments before interpreting an event. Keep the 48,000-byte serialized page bound, including diagnostics. `--all` remains explicitly unbounded and returns complete decorated event objects with no paging change. No new acquisition path or external reads.

## Test-first gates

1. Assertion-specific RED: construct tool-result payload bearing a recognized envelope, call JSON events with `--diagnostics --last 1` and without `--payload`; assert detection and **absence** of the `payload` field. Repeat with a search/context window. Both routes must select the same IDs/order as their unflagged counterparts. Establish a compiling failure at the assertion, not a missing-symbol failure.
2. Assert `--payload --diagnostics` returns the same native payload bytes/value as `--payload` without diagnostics; unflagged output has no `diagnostics` and remains unchanged. Verify `--all` complete event export and exact `--event` selection.
3. Obtain page 1 with a diagnostic whose hidden payload has irrelevant bytes; change only those bytes while preserving event ID and diagnostic result. A continuation with the old snapshot must fail. Test mismatch when toggling diagnostics/options and when query/window evidence changes, including unselected evidence covered by existing receipt semantics.
4. Force oversized decorated events and reconstruct across all segments/pages: reconstructed diagnostic equals the unfragmented export, every encoded page is at most 48,000 bytes, and a fragment alone is not an event/diagnosis.
5. Assert `not_detected`, `uninspected`, missing-path detection, marker-only detection, and no file I/O with an unavailable/path-trap artifact. Preserve Pi sidechain ownership and existing raw/export tests.
6. Reject each incompatible route with an explicit error. Keep `--all` incompatible with its existing paging flags.

Run focused CLI and recognizer tests in the inner loop; then `go test ./cmd/nn/cmd`, `go vet ./cmd/nn/cmd`, `gofmt` checks and `git diff --check`. Do not claim a filesystem spy if only static review was performed.

## Ownership and stop condition

One implementation writer; parent reviews attributable RED/GREEN transcript, actual diff, focused tests, and compatibility checks before acceptance. Writer derives its own Bar phase, re-establishes the exact READY managed LSP session for minimal source/API confirmation, and does not repeat broad discovery. If an existing builder cannot provide a selection-preserving finalization seam or binding hidden evidence requires changing ordinary snapshots, stop and report that counterexample rather than relaxing the contract. Do not edit ADR status or implement an artifact reader in this increment.
