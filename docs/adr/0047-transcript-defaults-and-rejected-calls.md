# ADR-0047: Transcript picker, sustained attention, and rejected calls

Status: Accepted

## Decisions

Bare transcript invocation opens the conversation picker after one bounded discovery page. Explicit
operands and clear actions still execute directly; the default picker is not a confirmation menu.
Picker labels retain the displayed conversation labels, with exact identity in secondary text and a
More conversations action for explicit pagination.

Supersede ADR-0046's default pass-count expiry: standing attention remains enabled until opt-out or
session end (or lost authorization state). Every eligible user Open/explicit Refresh receives its
approved per-check allowance, not an arbitrary remaining-pass count. Cumulative usage stays visible
and monotonic. Actual context/resource constraints pause acquisition; Back, restoration, pagination,
and incidental tool completion do not run checks or refund consumption. Explicit scope restrictions
and any human-imposed total ceiling remain binding. No timer, background scanner, or global consent.

Metric v2 separately counts validation-rejected calls, excluding them from the command/edit ratio.
Initial support is deliberately narrow: Pi `bash` with an argument object missing `command`, paired
with one later same-source `toolResult`, exact call ID/name, `isError: true`, the Pi validation-error
format naming the missing command, and matching Received arguments. Both records must be inside the
selected owned work window. Call/result ambiguity, unknown formats, other tools/schemas, and ordinary
execution errors remain on the existing conservative path. No adjacency-based pairing or inference
from assistant prose. Results still consume window slots and never create extra commands.

The installed Pi validator inspected during this change formats pre-execution failures as
`Validation failed for tool "NAME":` followed by errors and `Received arguments:`; the agent loop
validates before `tool.execute`. This is a producer-format adapter, not cryptographic attestation or
proof against fabricated transcript files. The quoted user event was not recovered through the supplied
room context; it is not claimed as a verified reproduction.

Expose rejected counts and the linked result in retained evidence. Fresh pages declare metric version
2; older retained pages with no metric_version are v1 and are replayed without reinterpretation. Policy
thresholds, Datalog evaluation, and the last-100-owned-message window are unchanged.

## Verification boundary

Native tests cover positive rejection, ambiguous/wrong/source-mismatched/ordinary results, window
edges, rejected-only zero denominator, and retained output. Documentation tests protect served picker
routing and per-check standing approval. Static instructions do not prove conversational compliance.
