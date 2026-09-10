# Attention condition-check verification

ADR-0062 adds optional structured checks to the bundled low-edit-ratio condition
receipt. It does not change rule evaluation, thresholds, applicability, outcome
precedence, or task authority.

Verified cases:

- 0 edits / 15 commands: command minimum fails; ratio threshold passes; no match.
- 5 edits / 40 commands: command minimum passes; ratio threshold fails; no match.
- 0 edits / 40 commands: both checks pass; match.
- zero denominator or unavailable/unknown evidence: indeterminate; unevaluated
  comparisons are omitted and the existing reason remains authoritative.
- parsed policy variants receive no bundled diagnostics, preventing an explanation
  from contradicting a changed Datalog operator.

Text output renders actual/operator/threshold/pass state and cites the first failed
check in an unknown-applicability hypothetical. New receipts expose checks in JSON.
Old receipts without checks remain readable. Coverage pages reuse the retained
hypothetical. LLM guidance requires relaying failed checks without inferring task
classification or phase transitions.

Full tests, targeted race tests, `go vet ./...`, and `git diff --check` passed.
Three isolated mutations failed attributably and were restored: forced command
check success, forced ratio check success, and omitted check rendering.

Logs: `/tmp/nn-condition-checks-{targeted,contract,full,race,vet,mutations}.log`.
