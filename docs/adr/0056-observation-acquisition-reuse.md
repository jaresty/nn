# ADR-0056: Conservative observation acquisition reuse

Status: Accepted (implementation and verification in progress)

## Decision

Use five minutes as the default initial observation window. Preserve explicit
window overrides and retained Refresh windows. Do not change the native signal's
last-100-owned-work-message metric into a five-minute metric.

An unchanged Refresh may reuse retained results and readable samples without
transcript decoding only when source identity, size, modification time, change
time, authenticated locator resolutions, policy definitions, and options agree.
Capture metadata before and after each source read. Unstable acquisition and
unsupported metadata disable the shortcut. Old snapshots without this metadata
use normal acquisition. Metadata continuity is a conservative optimization, not
cryptographic verification of content; say so in the output.

The parent transcript is shared for canonical agents, lifecycle and full launch
joins. Acquire worker histories individually rather than retaining a raw capture
of every worker simultaneously. Attention retains its existing immutable metric
and evidence projections; explicitly describe those as sequential projections,
not as a shared raw-source capture. Preserve canonical ordinals, ownership,
assignment authority, native signal input windows and omission accounting.

Initial discovery still reads historical sources where necessary to establish
recency and unknown evidence. A shorter window alone does not bound source reads.
Do not silently use file mtime as proof that all contained event timestamps are
old. Incremental byte-offset parsing and initial tail-only discovery are deferred
until their continuity and authority contracts have executable guards.

## Verification

Compare initial and unchanged Refresh separately. Record elapsed time, peak
memory and the distinction between decoding retained state and decoding
transcripts. Persistent guards and isolated mutations must reject stale reuse,
changed metrics, altered retained results, incomplete coverage and a one-hour
default. No claim of complete optimization or broad usability follows from a
single real-recording benchmark.
