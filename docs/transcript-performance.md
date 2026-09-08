# Transcript performance

## Search streaming optimization (2026-09-07)

Baseline: `602d1be1cc0028c2362387a5fcf06623bcde9f65`.
Machine: Apple M5 Max, darwin/arm64. Measurements are serial, cache-primed
local runs, not cold-cache guarantees. Other active sessions can mutate the real
corpus; identical returned bytes do not establish an immutable corpus.

### Observations

Directory discovery found 9,703 transcript files in 0.21 s. The materialized
search phase took 29.14 s and allocated 18.36 GB cumulatively in a one-iteration
Go CPU/allocation profile. Discovery was not the dominant cost.

Representative installed baseline vs candidate CLI runs, output redirected to
files, with `/usr/bin/time -l`:

| Directory-wide search, limit 5 | Before | After | Output |
| --- | ---: | ---: | --- |
| Raw `one-queue reference-backed` | 27.39 s | 17.19 s | byte-identical |
| Meaningful `Known-good:` | 24.18 s | 4.22 s | byte-identical |
| Runtime-generated absent query, raw | 25.86 s | 17.54 s | both zero matches, byte-identical |

Raw phrase peak RSS decreased from 439 MB to 139 MB; meaningful search from
437 MB to 90 MB. An earlier supposedly absent query was present in the agent's
own transcript and is excluded from the no-match measurement above.

Adjacent-command baseline (two runs, unchanged): recent `ls --limit 6` took
4.48–5.17 s; targeted tree 0.55–0.58 s; selected-child usage summary 0.15–0.16 s.
The single-session ROOT query took 0.14–0.16 s but returned no matches. These
commands were not optimized in this change. Root/child searches only search
records in the selected file; this change does not add external-sidechain search.

### Fixed, repeatable benchmark

```sh
go test ./cmd/nn/cmd -run '^$' -bench '^BenchmarkTranscriptSearchBounded$' -benchtime=3x -benchmem
go test ./cmd/nn/cmd -run '^TestTranscriptSearch' -count=1
```

The generated 16 MiB fixture has 128 large matching records and a session header.
With limit 1, the second match establishes truncation early:

| Mode | Before | After | Allocated before → after |
| --- | ---: | ---: | ---: |
| Meaningful | 111.6 ms | 4.16 ms | 89.75 MB → 1.30 MB |
| Raw | 73.2 ms | 3.00 ms | 89.73 MB → 1.30 MB |

This is deliberately an early-truncation workload, not a universal speedup.
The allocation guard uses a generous 8 MiB ceiling and no wall-clock assertion.

### Implementation and compatibility

Search retains only matches plus the current decoded record, rather than every
record in a file. Raw non-matches avoid a second message decode. After a limit+1
match confirms truncation, matching/decoding stops where possible, but scanning
continues through every file to preserve read/open and oversized-line errors.
When a file's session header is not yet known, decoding continues until its first
valid nonempty session ID is found, including headers after the match limit.
A shared initial scanner buffer reduces per-file allocation.

No persistent cache, concurrency, CLI/schema changes, or ownership expansion.
The common `readRecords` reader and tree/event projections are unchanged.
Search still performs case-insensitive matching with the same Unicode lowercase
semantics, records parsed-record fallback ordinals, skips malformed records, and
preserves file order, complete excerpts, filters, exact truncation, and partial
results on errors. It does not stop reading later files merely because the output
limit has been reached.

A frozen materialized reference algorithm tests successful-result parity across
modes, limits, filters, Unicode, late/multiple/missing headers, malformed records,
and ID-less events. Error parity includes scanner failures in the matching file,
later files, and a later missing file. Three isolated overlays distinguish changed
session provenance, swallowed scanner errors, and removed post-truncation work
suppression; each fails only its designated new guard, with restoration passing.

### Remaining work

Raw no-match searches still scan and decode the full corpus. Listing has its own
multi-second baseline. Further optimization needs separate profiling and guards;
these measurements do not justify skipping error checks, changing search scope,
or introducing an unauthenticated/stale cache.
