# Transcript attention and review: performance observations

Implements [ADR-0044](adr/0044-bundled-transcript-attention-policies.md).
These are local measurements, not guarantees, calibrated thresholds, or evidence of LLM usability.

## Production changes

- All normal Pi capture callers stream the existing JSON/base64 representation to a private atomic
  cache file, avoiding a second full encoded copy. Byte-for-byte guards compare against `json.Marshal`.
- Attention captures the full root authority plus only selected authenticated sidechains. Existing
  review still captures its full population. Selected-room metrics, identity, and reload parity are tested.
- Neither optimization requires a production feature flag. The old serializer is only a test oracle.
- Fresh processing still reads whole selected source prefixes. A 100-message result window is not a
  bound on source reads, allocations, or elapsed time. No heap profile was performed.

## Method and retained samples

[Raw measurement JSON](transcript-attention-benchmark.json) contains no transcript payloads. It includes
pre-optimization, streaming-only, final-flow, and final interleaved-review samples and measurement-binary
identity. The baseline is clean commit `1452b7db875ca7f00360b66e78551f6f50367ced`.
Later search/documentation refinements do not change the measured capture implementation.

macOS 26.5.2 arm64; each sample launches a fresh CLI process via `/usr/bin/time -l`. Wall time includes
launch/output capture. OS caches were not flushed and host load was uncontrolled: these are not cold-disk
measurements. Three repetitions per flow; five alternating baseline/current pairs for final review.
Live Pi files were not frozen. Temporary driver/stdout/stderr files were retained under
`/tmp/nn-attention-bench/`; those paths are not permanent evidence storage.

Synthetic Pi and inline Claude inputs each contain 5,000 command-invocation records (~1 MiB); the SDK
fixture contains the same child data under authenticated Task/meta-sidecar layout. Real inputs are the
selected lsp-trace Pi room `8111b451-93bf-448` and Claude leancoffee ROOT. Pi's ~21 MiB root does not
represent its full sidechain inventory. `--task implementation` is benchmark input, not proof of the
real assignment's classification.

## Attention results

Final median seconds; inspection retrieves the first retained 20-record page.

| Input | Fresh | Replay | Inspect | Fresh bytes | Inspect bytes | Peak fresh RSS MiB |
|---|---:|---:|---:|---:|---:|---:|
| Synthetic Pi | 0.0643 | 0.0104 | 0.0096 | 1,806 | 5,023 | 39.2 |
| Synthetic inline Claude | 0.0532 | 0.0105 | 0.0094 | 1,735 | 5,203 | 38.0 |
| Synthetic SDK child | 0.0383 | 0.0104 | 0.0094 | 1,721 | 5,477 | 36.7 |
| Real Pi child | 0.6527 | 0.0103 | 0.0101 | 1,954 | 10,615 | 123.9 |
| Real Claude ROOT | 0.0496 | 0.0103 | 0.0095 | 1,824 | 6,960 | 31.0 |

Real Pi pre-optimization median was 1.7845 s, peak 1,702 MiB. Final samples show approximately 63% less
wall time and 93% less peak RSS. Streaming alone measured 1.6356 s and 661.5 MiB in a separate paired
run; selection accounts for the further reduction. Do not treat these separate sample sets as a
controlled causal decomposition. A pre-optimization synthetic Pi outlier of 0.7114 s remains retained.

Inspection originally returned ~48 KB for the real Pi window; native paging now returns 10,615 bytes
on page one while preserving full-window counts and snapshot/room binding.

## Existing review results

Final five alternating pairs on the unchanged normal Awaiting return command:

| Build | Median seconds | Peak RSS MiB |
|---|---:|---:|
| Baseline | 3.6758 | 2,244.1 |
| Candidate | 3.3640 | 907.3 |

This sample shows about 8% less wall time and 60% less peak RSS. Timing is variable: a prior three-pair
streaming-only sample measured 3.4034 versus 3.4842 s (no speedup), while the final three-run flow sample
measured 3.4171 versus 3.2051 s. The consistent benefit is memory; do not promise a review latency gain.
Tree-summary final baseline/candidate medians were 1.3100/1.2857 s for real Pi and 0.0257/0.0257 s for
real Claude. Review population and capture completeness were not reduced to obtain these figures.

## Reproduction

Build baseline from a clean archive and candidate from source; repeat and retain time/RSS/output bytes:

```bash
nn transcript attention "$SESSION" --agent "$AGENT" --task implementation --format json
nn transcript attention "$SESSION" --snapshot "$SNAPSHOT" --format json
nn transcript attention inspect "$SNAPSHOT" --agent "$AGENT" --format json
nn transcript tree "$SESSION" --summary --json
nn transcript review "$PI_SESSION" --queue awaiting-return --limit 20 --json
```

Replay/inspection use the fresh result's snapshot. Review is Pi-specific; attention supports both Claude
layouts and Pi. Do not compare differing source populations or interpret source growth as a regression.

## Verification boundary

Persistent guards cover existing-Datalog rule control, parameter binding, scope and sufficiency,
owned/deduplicated metrics, bounded windows/output, Pi/Claude adapters, retained replay, corruption and
expiry, notebook isolation, room order, and three published Office surfaces. Additional guards cover
stream serialization, scoped/full parity, and default selection without an enablement switch.
Fourteen isolated mutations each failed their named guard with other groups passing; each restoration
passed. Removing parameter-fact provisioning was rejected. Full, race, and vet checks gate publication.

Skill-text guards prove publication, not LLM compliance. Provisional ratio <0.05 and minimum 30 commands
are uncalibrated inspection hints. Shell invocations may edit files; recognized edit counts do not measure
all changes or successful work. No calibration study or controlled conversational comparison was done.
