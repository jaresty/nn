# Observation acquisition optimization — native verification

This is a bounded implementation/Craft verification, not conversational replay or
broader usability validation. Changes remain uninstalled and unpushed. This document records staged measurements, not one controlled comparison.

## Delivered

- Initial default: five minutes; explicit and retained windows remain supported.
- Native detector work window remains unchanged (100 owned work messages for the
  current policy), including canonical IDs, joins and earlier-record accounting.
- Parent evidence is shared across signal evaluations; worker histories are
  acquired separately rather than retained together in one raw-source capture.
- Fingerprints stream their components into a digest instead of assembling one
  large JSON byte slice.
- Unchanged Refresh checks canonical scope, authenticated locators, source file
  identity/size/mtime/change-time, options and policies before retaining results.
- Changed parent transcripts need not invalidate unchanged worker evidence:
  per-worker metadata keys include complete launch and lifecycle evidence.
- Capture metadata is checked before and after the read; uncertainty disables
  reuse. Unsupported platforms and older snapshots take normal acquisition.
- Readable replay uses a retained byte boundary, not headings found in source
  text. Repeated unchanged Refresh avoids transcript decoding when its metadata
  checks pass. This is not cryptographic verification of source content.

## Live read-only measurement

Source: `/Users/schwa/.pi/agent/sessions/--Users-schwa-dev-lsp-trace--/2026-09-04T14-39-04-442Z_01a06cdb-e0fa-77b0-a090-f2e24727c340.jsonl`

| Operation | Population | Raw-record decodes | Elapsed | Peak resident bytes |
|---|---:|---:|---:|---:|
| Initial, default five minutes | 510 | 106624 | 5.33 s | 166985728 |
| Immediately following unchanged Refresh | 510 | 0 | 0.01 s | 30408704 |

Initial coverage: two evaluations, 508 outside-window agents, no deferred agents.
Refresh: two retained unchanged results, the same 508 outside-window agents.
These counts are coverage, not health, success, causality or productivity.

An earlier implementation used 1186234368 peak resident bytes on a 508-agent
recording. The recording and recency default changed between runs; this is not a
controlled benchmark or a causal timing comparison. Single-run elapsed numbers
are rounded by `/usr/bin/time`. No byte-read measurement was collected.

The process-local counter covers `readRecords` and `parseCapturedRecords`, counts
repeated raw-record decoding, and excludes retained state JSON, schema probes and
message projection passes. Replay retains the original measurement; it does not
claim that the acquisition happened again.

Logs: `/tmp/nn-opt-final-{live,refresh}.txt` and corresponding `-time.txt` files.

## Persistent guards and isolated mutations

`cmd/nn/cmd/transcript_observe_optimization_test.go` runs in the ordinary Go suite.
Every mutation below failed its named assertion; its individual restoration then
passed that guard. No mutation was accepted on compilation failure alone.

| Mutation | Observed failing assertion |
|---|---|
| Change a retained command count | `OPT_RETAINED_RESULT FAIL` |
| Ignore source identity/size/times | `OPT_METADATA FAIL` |
| Block required fresh acquisition | `OPT_FALLBACK FAIL` |
| Drop the boundary work record | `OPT_NATIVE_WINDOW FAIL` |
| Restore the one-hour default | `OPT_DEFAULT FAIL` |
| Retain only one agent | `OPT_COVERAGE FAIL` |
| Locate readable content by an untrusted heading | `OPT_READABLE_BOUNDARY FAIL` |
| Decode the transcript during unchanged reuse | `OPT_ZERO_DECODE FAIL` |

The readable-boundary guard refines retained-result preservation; it is not an
additional independent request property. Source guards also cover authority,
canonical scope, policy changes, append, truncation, replacement and same-size
rewrite with restored mtime. Platform-dependent reuse checks skip where identity
and change-time support are unavailable; normal acquisition remains supported.

Harness/logs: `/tmp/nn-opt-mutations.py`, `/tmp/nn-opt-mutations.log`, and individual
mutation/restoration logs. This is counterfactual sensitivity, not retrospective
test-first evidence for earlier implementation.

Verification passed:

- `go test ./... -count=1` (`/tmp/nn-opt-full-2.log`)
- Relevant observation, capture, assignment, alias and virtual-protocol race
  tests (`/tmp/nn-opt-race.log`)
- `go vet ./...` (`/tmp/nn-opt-vet.log`)
- `git diff --check`

## Additional assignment-bundle release checks

`transcript_events_assignment_edges_test.go` now covers multi-page payloads,
48,000-byte transport limits, readable assignment clipping without losing the
selected section, replay of every page after source deletion, and rejection of
cross-selection snapshot reuse. JSON and readable defaults select different
facets and therefore correctly retain different bindings.

Injected page-read failures, snapshot/page mismatches, and missing segments all
fail before the renderer writes any output. A 300,000-character selected-event
fixture also verifies the 200,000-byte readable ceiling without partial evidence
publication (Cobra may emit usage text in the test helper). These cases passed
both normal and race-enabled runs. No production change was needed for them.

## Remaining work

Initial discovery still scans historical sources to establish recency, and the
readable tree/tails still have independent acquisitions. ADR-0057 now implements
bounded worker checkpoints and verified-prefix incremental decoding. Prefix
verification still reads historical bytes. Checkpoint format v2 explicitly
persists record ordinals; an integration guard caught their loss in the initial
format. Native-equivalence, partial-append, ownership, checkpoint binding and
rewrite/truncation/replacement tests pass.

The final pre-fastjson run measured 7.00 seconds and 168280064 resident bytes;
initial acquisition remains slower than the earlier tranche on the changing
recording. A stamp pointer originally retained its entire worker checkpoint;
copying the stamp removed that retention. The line reader also now avoids
bytes-to-string-to-bytes copies, with a compatibility guard for whitespace,
malformed lines, ordinals and final-line behavior. Full tests and vet passed
again in `/tmp/nn-pre-fastjson-{tests,vet}.log`; targeted race checks passed in
`/tmp/nn-byte-reader-race.log`. Further parser experiments must use frozen input. No background
monitor, mandatory navigation database, or hidden latest-state pointer was added.

The earlier assignment-inclusive-event and wide-observation release has separate
verification obligations; this optimization pass does not claim to discharge
all of them. No fresh conversational replay was performed.
