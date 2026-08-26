# ADR-0039: Represent process-history succession with `follows`

**Status:** Accepted
**Date:** 2026-08-26

## Context

The canonical relationship vocabulary distinguishes conceptual structure, evidence, tension, provenance, governance, and task dependency. A notebook-wide legacy-link audit found a final coherent class that none of those relationships represents: a later workflow or inquiry step retaining an earlier completed step or observation as process context.

Forcing these links into `extends`, `grounded-by`, `source-of`, or `requires` would incorrectly assert conceptual addition, evidential dependence, derivation, or blocking. Deleting them would discard useful discovery history. Multiple narrower types such as `reorients-after`, `latest-context`, and `lateral-continuation` would overfit presentation wording rather than capture their shared semantics.

## Decision

Add one canonical link type, `follows`:

> The source is a later workflow or inquiry step that proceeds after the target, without implying derivation, evidential dependence, conceptual extension, governance, contradiction, or task dependency.

Stored direction is:

```text
later step → earlier context [follows]
```

`follows` is a lateral relationship and occupies the RIGHT graph zone in either direction relative to focus. It uses the lateral edge-color family.

New writes must provide the same non-empty annotation required by every canonical link type. Legacy untyped edges remain readable and are not automatically inferred as `follows`.

## Consequences

- Process history remains queryable without contaminating evidence or conceptual paths.
- `follows` does not imply that the target caused, justified, authorized, or blocked the source.
- Typed traversal can include or exclude process succession explicitly.
- Graph legends and user-facing references must distinguish `follows` from provenance and task relationships while grouping it in the same lateral zone.
- The nine audited process-history edges motivating this decision may be migrated exactly; unrelated untyped edges require independent review.

## Alternatives rejected

### Reuse `extends`

Rejected because temporal continuation does not add structure or scope to the target claim.

### Reuse `grounded-by`

Rejected because several source notes explicitly identify the target as prior context rather than their evidential basis.

### Reuse `source-of`

Rejected because preceding a workflow step does not establish derivation or authorship.

### Delete the relationships

Rejected because the succession context is useful and consistently annotated.

### Add several process types

Rejected because all audited cases share the same safe semantic floor: later workflow context without stronger entailments.
