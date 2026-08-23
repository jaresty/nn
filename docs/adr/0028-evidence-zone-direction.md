# ADR 0028: Evidence zones follow dependency meaning from both endpoints

## Status

Accepted

## Context

A navigation trial followed a claim's outgoing `grounded-by` edge to an evidence observation. The claim's zoned Orient placed that evidence in BOTTOM even though the legend assigned outgoing `grounded-by` to TOP. At the evidence focus, four outgoing `supports` edges were absent.

The notes had reciprocal edges: claim `grounded-by` evidence and evidence `supports` claim. `grounded-by` was already mapped correctly, but incoming `supports` overwrote its TOP placement with BOTTOM. Outgoing `supports` mapped to no zone.

## Decision

Evidence zones follow semantic dependency:

```text
claim ─grounded-by→ evidence
grounded-by OUT → TOP
grounded-by IN  → BOTTOM

evidence ─supports→ claim
supports OUT → BOTTOM
supports IN  → TOP
```

TOP contains what the focus answers to or depends on. BOTTOM contains what builds on or is corroborated by the focus. Reciprocal `grounded-by` and `supports` edges therefore agree from either focus instead of overwriting each other with conflicting zones.

All non-evidence zone mappings remain unchanged.

## Consequences

- Claim-focused Orient shows evidential bases in TOP.
- Evidence-focused Orient shows supported claims in BOTTOM.
- Reciprocal evidence edges produce one consistent zone regardless of assignment order.
- Typed navigation guidance must choose direction explicitly: `grounded-by` walks claim to evidence; `supports` walks evidence to claim.
