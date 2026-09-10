# ADR-0060: Apply worker recency before acquisition

Status: Accepted; native and mutation guards verified (see ../transcript-initial-acquisition-verification.md)

Initial Pi observation uses source-change metadata to select worker histories
before opening them. A recent owned-source mtime or attributed parent-event
timestamp selects a worker. A stable old source with known old parent evidence
is outside the window. Never use the parent file's mtime to mark all workers
recent. This is a source-change clock, not proof of event-time recency or activity.

Unknown worker recency receives an independent canonical-order allowance of 20
history inspections. The rest are explicitly deferred, not silently omitted or
classified healthy. This allowance does not limit known-recent evaluations and
is independent of the displayed-detail limit. Explicit worker inspection remains
available. ROOT discovery is still required.

Retain metadata for skipped workers so unchanged Refresh does not acquire them;
changed sources or relevant metadata remain eligible on explicit Refresh.
Readable sample entries outside the initial window or deferred by its allowance
do not open worker histories and instead explain their exclusion.

This supersedes the earlier requirement to scan every history to establish event
timestamps. Preserve population accounting and native metrics for acquired
workers. Verify old-source exclusion, parent-mtime independence, recent-parent
selection, unknown-budget accounting and unchanged Refresh.
