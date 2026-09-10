# Compatible raw-record adapter results

The adapter code was discarded after evaluation; this findings record is retained. Production readers remain unchanged.

## Contract

Compared complete rawRecord values (including byte-identical message/data fields)
and record admission with encoding/json. Scalars and raw payloads are copied
before parser or input-buffer reuse. Standard fallback handles known duplicates,
case-folded/escaped keys, Unicode escapes, invalid UTF-8, incompatible field types,
parser errors, and records larger than 256 KiB. Strict JSON validation remains on
the fast path; scanner framing and ordinals are not changed.

The same frozen corpus used by the earlier experiment was tested: 6658 lines,
25232738 bytes, SHA-256
`d4ceff5a7574c1b569c6d3cd29c1244232b2659a47ed1830e86804d2eaf5ea70`.
All records matched, with 6652 fast-path admissions and six standard fallbacks.
The targeted table exercised six fast-path and sixteen fallback cases. Retained
value ownership and race checks passed. A 30-second differential fuzz run reported
54731 executions and no failure. This finite run is not proof of equivalence over
all possible inputs.

## Full record decoding benchmark

Darwin/arm64, Apple M5 Max, same frozen bytes and three 1-second benchmark runs:

| Decoder | ns/pass | Bytes allocated/pass | Allocations/pass |
|---|---|---|---|
| encoding/json | 72173609, 71623556, 71480669 | ~27714612 | 81428 |
| Compatible adapter | 68636932, 68559029, 68806315 | ~25153426 | ~61505 |

This is roughly a 4% time reduction, 9% fewer allocated bytes and 24% fewer
allocations. Unlike the earlier field-only trial, it retains complete raw
payloads. It is not an end-to-end observation benchmark.

## Decision

Do not switch production readers on this evidence. The modest decoding gain
comes with an additional parser, strict-validation pass, raw-span traversal and
fallback logic. The much larger field-only result suggests investigating repeated
downstream message decoding separately; it does not justify claiming a comparable
full-decoder or CLI speedup.

Commands used during the trial (the temporary adapter tests have since been removed):

- `go test ./cmd/nn/cmd -run '^TestTranscriptJSON' -v`
- `go test ./cmd/nn/cmd -run '^$' -bench '^BenchmarkTranscriptRecordDecoder$' -benchtime=1s -count=3`
- `go test ./cmd/nn/cmd -run '^$' -fuzz '^FuzzTranscriptJSONCompatibility$' -fuzztime=30s -parallel=2`

Logs: `/tmp/nn-json-adapter-{tests,benchmark,fuzz,race}.log`.
