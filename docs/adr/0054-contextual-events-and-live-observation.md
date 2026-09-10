# ADR-0054: Contextual events and recent-change observation

Status: Accepted

## Events

Accept an exact agent either positionally or through --agent, never both. Normalize
before acquisition and snapshot binding. --include-assignment adds authenticated
launch evidence in an independent section, without consuming event limits or filtering
assignments by event predicates. Reuse capture, handoff joins, event projection and
retained page transport. The opt-in bundle has an explicit versioned receipt; bare
events and context keep their contracts. Missing/unsupported assignment evidence is
reported, not inferred. Pi sources share one sequential complete-record-prefix capture.
Other adapters retain selected projections and disclose unavailable assignment support.
Text has a separate 8000-character assignment budget and a combined 200000-byte ceiling;
JSON preserves lossless 48000-byte page/segment transport. Replay never reopens sources.

## Observe

Normal observation considers the whole conversation, not a first-20 population.
Initial eligibility is evidence changed within --recent (default 5m; revised by ADR-0056), plus unknown
recency. Per-agent evidence fingerprints include owned records, complete authenticated
launch records/invocations, relevant lifecycle metadata, policy and task overrides.
--refresh names a retained observation and compares fingerprints; new/changed agents
are evaluated and unchanged results remain explicitly not re-evaluated. --snapshot
replays retained readable output without acquisition. Missing/expired state fails
with explicit fresh-baseline guidance rather than silently resetting the boundary.
No implicit latest-state pointer, scheduler or monitoring database.

Evaluation is not capped at 20 agents. --attention-limit becomes a detail-display
limit; canonical readable ROOT/two-child sampling remains independent. Exact-agent
selection remains an explicit advanced restriction, never the normal recipe. Native
attention captures remain bounded batches; observation coverage accounts for all
agents and errors. A failed request publishes no misleading partial coverage.
Output and retention hard limits remain explicit errors, not silent sampling.

All policies are measured; task applicability is separate. Unknown timestamps are
conservative eligibility, not proof of recent work. Fingerprint equality means no
observed relevant change, never healthy/resolved. Source acquisition is not bounded
by the output limits and multi-file acquisition is sequential, not atomic.
