# ADR-0061: Render unknown attention applicability as a hypothetical

Status: Accepted; native and mutation guards verified (see ../transcript-hypothetical-attention-verification.md)

Normal observation does not require task classification. Therefore an
implementation-scoped signal with measured evidence commonly has unknown
applicability and the formal outcome `needs_context`. Keep that outcome and task
authority unchanged, but make the independently measured condition useful.

Copy each policy's declared task scope into its signal receipt. In text output,
show scope beside signal identity. When applicability is unknown, mechanically
render one conditional sentence from scope and condition:

- match: if this is a `<scope>` task, the signal would trigger;
- no_match: if this is a `<scope>` task, the signal would not trigger;
- indeterminate: if this is a `<scope>` task, available evidence is insufficient;
- error: no hypothetical; preserve the error.

This sentence does not classify the worker, infer task phase, or replace the
formal outcome. Established applicable/not-applicable results remain direct and
do not receive hypothetical wording. Old retained receipts without per-signal
scope remain readable and do not invent it.
