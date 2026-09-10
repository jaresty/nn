# ADR-0057: Incremental worker evidence with verified prefixes

Status: Accepted; implementation in progress

Worker observation retains a bounded work-message suffix, the latest two owned
user messages, an oversized-candidate witness, and historical summaries required
for native availability, recency, fingerprints and omission counts. Canonical
ordinals count exactly the records accepted by the existing raw-record decoder.
Incomplete final lines are not consumed.

A retained worker checkpoint is immutable, private and explicitly referenced by
observation state. It is not a hidden latest-state index. On Refresh, validate
scope and file identity and stream-hash the previously consumed prefix. Only a
matching prefix permits append-only decoding. Replacement, truncation, mismatch,
unsupported identity or unavailable checkpoints take a fresh streaming scan.

Prefix verification still reads historical bytes; it avoids historical JSON
record decoding, not all historical I/O. Initial scans remain necessary to
establish canonical ordinals and historical recency. Parent discovery and
readable tree/tail acquisition are not converted to sparse worker checkpoints.

Preserve the native detector window, authenticated ownership, complete parent
assignment joins, availability and unknown-recency summaries, and exact earlier
record counts. Bounded caching may be declined when a checkpoint exceeds its
storage budget; this must not suppress the agent or change the measured result.

Verify native equivalence, appended and partial records, foreign ownership,
rewrite/replacement/truncation fallback, and retained-cache loss. Benchmark
initial, unchanged and changed Refresh separately before release.
