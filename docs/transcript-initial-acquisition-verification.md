# Initial observation acquisition verification

## Parent-only roster and shared parent capture (ADR-0059)

The roster uses parent topology rather than hydrating all worker usage. The parent
capture is shared with wide observation. Native sample parity and zero worker-body
reads during roster discovery are guarded. An existing sample test caught map-order
selection; explicit canonical ID sorting restored the original child selection.

Three paired runs on one frozen 515-agent fixture:

| Version | Wall seconds | Raw records decoded |
|---|---|---:|
| Before parent-only roster | 7.64, 6.77, 6.58 | 108100 |
| Parent-only roster | 5.87, 5.40, 6.23 | 64084 |

Median: 6.77 → 5.87 seconds. Coverage matched in each run. This fixture copied
worker bytes without preserving original mtimes; it was not used to evaluate the
subsequent source-mtime selection policy.

## Pre-acquisition worker recency (ADR-0060)

A separate frozen fixture preserved worker mtimes as well as bytes. Both versions
used that identical source set. Parent mtime is not a selection input.

| Version | Wall seconds | Raw records decoded | Worker streams opened |
|---|---|---:|---:|
| Parent-only roster | 5.18, 5.23, 5.32 | 64264 | 364 |
| Pre-acquisition recency | 3.32, 2.17, 2.35 | 20122 | 1 |

Median: 5.23 → 2.35 seconds. All six runs reported population 516, evaluated 1,
outside_window 515, unavailable/error/deferred 0. The candidate skipped 363
old authenticated worker sources using metadata; unknown-history allowance used 0.
These are finite local measurements, not a universal speed or memory guarantee.
The parent source still needs discovery reads; not all repeated decoding is gone.

The selection clock changed deliberately: recent source mtime or attributed parent
timestamps select a worker, even if its stored work events have old timestamps.
This is not evidence of current activity. Unknown recency permits 20 canonical-order
worker history inspections independently of displayed detail; additional unknowns
remain explicitly deferred. Excluded readable children show an explanation without
opening their histories. Non-Pi native readable behavior is unchanged.

## Guards and verification

- Old worker histories are not decoded; a freshly modified parent file alone does
  not make the worker recent. The readable sample also does not open the old file.
- Recent worker metadata and recent attributed parent evidence independently select
  a worker while preserving its native historical metric window.
- 26 unknown workers with display limit 1 produce 20 history inspections and six
  deferred agents. Unchanged Refresh retains that count with zero decoding.
- A changed formerly skipped worker is acquired on explicit Refresh.
- Full `go test ./...`, targeted observe race tests, `go vet ./...`, and diff hygiene
  passed after restoring the compact publication wording and budget.
- Four isolated mutations failed attributably and were restored individually:
  old-source exclusion, unknown-history allowance, recent-parent selection, and
  readable-history skipping. Each restored guard passed.

Logs: `/tmp/nn-roster-{tests,full,race,vet}.log`,
`/tmp/nn-recency-{tests,full-2,race,vet,mutations}.log`, and the paired
`/tmp/nn-{roster,recency}-{baseline,candidate}-{0,1,2}{,-time}.txt` files.
Frozen fixture creation is opt-in via `TestFreezeObservationBenchmark`; it is not
an ordinary observation dependency. Sources are captured sequentially, not atomically.
No fresh conversational replay, installation or push was performed in this tranche.
