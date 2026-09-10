# Hypothetical attention interpretation verification

ADR-0061 changes presentation and receipts, not attention policy evaluation.
Each new signal receipt copies the policy's declared task scope. Text rendering
places that scope beside signal identity. Unknown applicability receives a
mechanical sentence derived only from scope and condition:

- match: the signal would trigger if this were a scope-matching task;
- no_match: it would not trigger;
- indeterminate: evidence would be insufficient.

Applicable and not-applicable results receive no hypothetical. Error conditions
receive no hypothetical. Retained old receipts with no copied scope remain
readable and do not invent one. Coverage pages preserve scope and hypothetical
without requiring the policy header.

Guards cover the condition table, collapsed signal output, retained coverage,
direct applicable output, and old scope-less receipts. Full tests, targeted race
tests, `go vet ./...`, and `git diff --check` passed. Three isolated mutations
failed attributably and were restored: omitted scope, reversed no-match wording,
and hypothetical leakage into established applicability.

No task is inferred from launch text, edit counts, lifecycle, or task excerpts.
The formal `needs_context` outcome remains unchanged. LLM guidance requires
relaying the hypothetical without claiming implementation, recovery, phase change,
or success.

Logs: `/tmp/nn-hypothetical-{targeted,contract,full,race,vet,mutations}.log`.
