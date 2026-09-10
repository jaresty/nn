# ADR-0051: Attention signals in observation

Status: Accepted

`transcript observe` retains its ROOT + two direct-child readable sample and adds an independently
scoped Attention signals section. No policy, metric, ownership or retained-evidence semantics change.

Default attention selection is ROOT followed by canonical agent IDs across the selected conversation,
including descendants, bounded to 20 total. `--attention-limit 1..20` reduces that bound;
repeatable `--attention-agent ID` overrides selection with 1–20 exact IDs. Reject duplicate/unknown IDs,
empty selectors and explicit limit combined with explicit IDs. Never drop an explicit target silently.
This is canonical sampling, not importance/recency ranking or exhaustive conversation coverage.

`--task` supplies the established classification for the whole attention cohort. Without it, report
not evaluated—task scope not established, zero evaluated and the whole population unevaluated; do not
infer implementation from labels, commands, or the current request. For mixed-task cohorts, select
classified IDs or use separate invocations. Non-implementation tasks preserve native inapplicable.
The compact observe owner documents these options without eagerly loading attention's full reference.

With classification, reuse native buildAttention and renderAttention. Preserve all native outcomes,
reasons, policy/digest/parameters, metrics, windows and snapshot-bound inspection commands. Summarize
match, no_match, indeterminate and inapplicable counts separately; no-match is not health, and unknown
is not no-match. Attention captures retain evidence in the normal private expiring cache, not notebook
truth. Initial candidate inventory and retained evaluation are separate acquisitions; report both
scopes, and never claim an atomic cross-stream snapshot.

Invalid selection/source inventory fails the command. A native evaluation batch failure is displayed
as an explicit attention error with no successful evaluation published, while readable stream output
remains usable. Do not retry, substitute IDs or classify the failed batch as no-match. Native attention
itself remains fail-closed. The combined observation retains its 200,000-byte publication bound.
No background monitoring, task inference, new detector engine or source-processing guarantee.

Verify selection beyond the readable sample, caps/explicit scope, absent/other task, native statuses and
retained evidence, error disclosure and unchanged readable streams. Counterfactual guards must reject
sample-conflation and invented task classification; publication checks are not conversational replay.
