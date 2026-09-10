# ADR-0053: Combined event and failure tails

Status: Accepted

## Decision

Add `events --last N --include-errors M --format text` for one selected agent.
Acquire and project its ledger once, then independently select the last N ordinary
ledger events and last M explicit failure events. Both limits are 1–200.
Use the existing error predicate and canonical ordering, not text matching.
Inclusive time bounds apply to both selections. Preserve IDs and joins; identify
overlap explicitly while rendering each independently useful section.

Bind both views and their options into one composite snapshot identifier. This is
an evidence identifier, not retained replay or an atomic multi-source acquisition.
Buffer both sections before publishing, with a combined 200,000-byte ceiling and
existing per-event character limits. Empty failure tails remain explicit; older
failures do not establish unresolved problems or current health.

This initial shortcut is readable text only. Reject errors-only, search/context,
exact-event, summary, handoff and JSON/pagination combinations rather than guessing
which subset the second tail should search. Existing commands remain unchanged.
The events skill owns the combined Refresh recipe; no new command or parser.
