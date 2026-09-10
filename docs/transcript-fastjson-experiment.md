# fastjson field-extraction experiment

Baseline: `fb5e16a`. Candidate: `github.com/valyala/fastjson v1.6.10`, test-only.
No production parser or installed binary has been switched.

## Frozen input and procedure

A complete-line prefix of the supplied Pi parent recording was copied once to
`/tmp/nn-fastjson-frozen.jsonl`: 25232738 bytes, 6658 lines, SHA-256
`d4ceff5a7574c1b569c6d3cd29c1244232b2659a47ed1830e86804d2eaf5ea70`.
The source copy is not claimed to be an atomic live-conversation snapshot.
Worker sidechains are not part of this corpus.

The experiment extracts record type, agent ID, top-level timestamp and normalized
message role. The standard path uses the existing raw-record decode and role
extraction; the candidate parses once with a reused fastjson.Parser and copies
returned strings before parser reuse. It does NOT preserve complete raw payloads
or measure complete observation. Avoided raw-payload copying is part of the
measured difference; this is not a full decoder replacement benchmark.

Run with:

```sh
NN_FASTJSON_CORPUS=/path/to/frozen.jsonl go test ./cmd/nn/cmd -run '^TestFastjson' -v
NN_FASTJSON_CORPUS=/path/to/frozen.jsonl go test ./cmd/nn/cmd -run '^$' -bench '^BenchmarkTranscriptFieldParser$' -benchtime=1s -count=3
```

## Observed results

Darwin/arm64, Apple M5 Max, Go module toolchain 1.26.1:

| Procedure | ns/pass, three runs | Approximate bytes allocated/pass |
|---|---|---:|
| Existing field extraction | 136745938, 135251208, 134729198 | 58755340 |
| fastjson field extraction | 5503221, 5506357, 5398519 | 282055 |

All four extracted fields and success/failure admission matched over this frozen
corpus. The retained-string reuse guard passed. These are bounded observations,
not proof of general compatibility or a 25x end-to-end CLI speedup.

The semantic survey found:

- Duplicate `agentId`: encoding/json selects the later string, fastjson.Get selects
  the earlier value in the tested case.
- `AGENTID`: the current struct decoder matches case-insensitively; direct fastjson
  lookup does not.
- Numeric `agentId`: the current decoder rejects the record; GetStringBytes returns
  nil without rejecting the record. Adopting that behavior could change ordinals.

The survey records differences rather than asserting that they are compatible.
Before production adoption, an adapter must preserve admission, field matching,
duplicate handling, retained-value ownership, raw-payload requirements, canonical
ordinals and malformed/incomplete-line behavior. Benchmark that adapter and the
full observation pipeline on frozen sources before deciding to keep it.

Logs: `/tmp/nn-fastjson-differential.log`, `/tmp/nn-fastjson-benchmark.log`.
