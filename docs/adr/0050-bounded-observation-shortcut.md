# ADR-0050: Bounded observation shortcut

Status: Accepted

Provide `nn transcript observe <session>` as a readable, one-shot composition of the existing
canonical ROOT direct-child selector and readable event tails. Include ROOT and up to two canonical
direct children, five events per stream, 1,000 readable characters per event. Preserve native
ownership, IDs, detail availability and independent snapshots. Disclose omitted direct children and
other uninspected branches/history; canonical order is not recency or importance. This is not an atomic
cross-stream snapshot, monitor, retained capture, or source-read performance guarantee.

Reuse native command implementations, not shell execution or another transcript parser. Buffer the
bounded result and publish only after all selected reads succeed. No discovery or scope substitution
inside this command. Advanced selection and complete evidence retrieval remain separate commands.

The observe reference owns this compact initial-read contract without requiring interaction,
navigate or events references first. Load those owners only for their specialized follow-up actions.
Keep explicit scope, self-exclusion, Back/Refresh, evidence qualification and capture approval rules.
Verify the served command template on Pi, SDK and Claude fixtures; publication checks establish
instruction presence, not model compliance. Existing event/search work is retained unchanged.
