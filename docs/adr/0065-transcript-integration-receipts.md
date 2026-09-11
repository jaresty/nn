# ADR-0065: Transcript integration receipts

**Status:** Accepted

## Context

Retained subagent transcripts contain attributable task and execution evidence, but later notebook search cannot directly bridge a project topic or integrated result back to the transcript that produced it. Raw child summaries must not become accepted knowledge without parent inspection and adjudication.

## Decision

Add `nn transcript receipt <session>` to create one expiring draft observation after substantive delegated work is accepted, partially accepted, or rejected. The command resolves explicit paths or discovered session IDs through the shared transcript resolver, records the stable provider/session identity, assignment, parent disposition, adopted and rejected claims, resulting artifacts, verification, and a reconstruction command. It does not copy raw transcript bodies.

Required flags are `--assignment` and `--disposition accepted|partially-adopted|rejected`. Repeatable `--adopted`, `--rejected`, `--result`, and `--verification` fields are optional. `--parent-session` records the parent when known. `--expires-in` defaults to `336h` (14 days). Optional `--link-to`, `--link-type`, and `--annotation` triples create links in the same note write.

The receipt is tagged `subagent-handoff` and `transcript-receipt`, remains draft, and is committed through one backend write. Session IDs are stored by default; machine-local resolved paths are used only to validate and classify the transcript. A virtual protocol should trigger receipts only after parent-side transcript inspection and consequential adjudication, not after every subagent return.

## Consequences

Receipts make transcript evidence discoverable through ordinary notebook search and graph links while preserving the distinction between producer output and parent adoption. They add short-lived Git-backed notes, so expired unreferenced receipts require periodic cleanup and sensitive raw prompts must remain outside the note.
