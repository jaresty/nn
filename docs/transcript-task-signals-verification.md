# Task-oriented attention and combined event tails: verification

## Delivered contracts

ADRs 0052 and 0053 cover automatic per-agent/per-signal measurement with independent
applicability, and independently bounded readable event/failure tails acquired together.
Metric v2 and historical Policy.Evaluate semantics remain unchanged. Version-1
attention capture replay does not reopen sources or reevaluate policy.

## Executed checks

- Full `go test ./... -count=1`: passed.
- Targeted race tests for attention, signals, observe and combined tails: passed.
- `go vet ./...` and `git diff --check`: passed.
- Published bare-observe and known-task recipes executed on Pi, SDK-layout and
  Claude fixtures. These are native/publication tests, not conversational replay.
- Test-only second policy: independent windows and per-signal error isolation.
- Mixed overrides, absent task, task-context attribution, UTF-8 clipping,
  ambiguous launch joins and legacy replay after source deletion: passed.
- Combined-tail Craft guards: ten isolated mutations each failed only their named
  assertion; baseline/restoration passed. Removal of the publication ceiling was
  rejected by the publication guard. Guards persist in
  `cmd/nn/cmd/transcript_events_combined_test.go`.

Session logs: `/tmp/nn-both-{full,race,vet}.log` and
`/tmp/nn-combined-craft-results.log`. The mutation runner is scratch orchestration,
not a runtime feature. An earlier combined verification command timed out at 300
seconds; the subsequent separate race run and final combined verification passed.

## Read-only real-recording checks

For exact worker `bd339eab-fdf6-460` in the supplied lsp-trace recording:

- Combined event read returned ten recent events and ten explicit failures under
  shared snapshot `c6bc44f36b1202d7a021dde85dc96b8d737b9ecbf39acdedae22f5ab2e9a91fc`.
- Observe without task override performed one signal evaluation, selecting one
  agent and disclosing 505 omitted agents. Condition was `no_match`; applicability
  remained unknown and outcome was `needs_context`. Authenticated launch context
  was explicitly partial (411 text bytes omitted).
- Recorded outputs: `/tmp/nn-combined-live.txt`, `/tmp/nn-signals-live.txt`.

These are source-bounded observations, not statements of present worker health,
resolution, productivity or success. No lsp-trace files or agents were modified.

## Limits

No fresh conversational agent replay was performed for this revision; that remains
separate from native tests and requires delegation authorization. Mutation sensitivity
is not exhaustive semantic or usability validation. Combined-tail identity guards
check preserved IDs and nonmutation of input joins, not every cross-source join case.
Output limits do not bound source-read time or memory. No install/push claim follows
from the test results alone.
