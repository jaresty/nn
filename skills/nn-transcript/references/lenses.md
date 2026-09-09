---
name: lenses
applies_when: "When a human asks to scan, arrange, compare, or interrogate transcript evidence through a preset, inferred, blank, or user-defined lens at office or room scope."
---

# nn-transcript / lenses — shared projection language

Named lenses are presets, not a closed vocabulary. Accept a named preset, direct axes, a question,
a comparison, a visual metaphor, or an unspecified request such as “show another useful view.” Translate
it into an explicit projection before rendering; let the human revise any dimension conversationally.

## Projection contract

State:

- scope: office, team, room, or selected events;
- question;
- bounded authenticated population;
- X, Y, grouping, mark, and filters;
- whether each dimension is authoritative or interpreted;
- retrieved, uninspected, omitted, and unknown counts;
- evidence snapshot or query boundary.

Every spatial channel has one declared meaning. Unknown coordinates remain unknown or unplaced. An
interpreted dimension may organize inspected evidence, but cannot create identity, topology, measured
values, causal attribution, completion, or current activity.

If part of a requested lens is unsupported, preserve the useful remainder and name the rejected
inference. For example, recorded interruptions and timing gaps may be shown while provider-retry
causality remains unavailable.

## Shared intent and transition grammar

**Incomplete operations open choices.** Bare `scan`, `change lens`, `another view`, and `compare`
requests transition to a context-appropriate chooser; they never authorize the LLM to supply missing
axes, lens, or comparison operands. **Explicit operands execute directly**: for example, `scan by
cost`, `group by manager`, or `compare report with result` applies the named operation.

**Attention questions** such as “what stands out?” or “what needs attention?” delegate selection of a
cheap metadata-safe lens and may render immediately. **Suggestions require approval**: `suggest a
scan` proposes one lens and waits rather than rendering it. **Delegated choice** such as “choose for
me” or “orient me” authorizes selecting and rendering a supported lens. Navigation requests such as
“show background workers” select only the population; they do not silently choose a lens.

A comparison request without **comparison operands** opens a chooser for the comparison set. `Back`
and `Refresh` are navigation/state operations, not implicit new lens requests.

## Level-aware scan activation

At a conversation lobby, office hallway, or nested team, **attention-oriented language** such as
“what needs attention?”, “anything interesting?”, or “what stands out?” must **automatically render**
a cheap **metadata-only** scan at the **current level**.
Do not require the human to know the term Office Scan. Use only already retrieved authoritative
identity, topology, parentage, lifecycle, measured-cost, and missing-value fields; do not infer drift,
failure, groundedness, waste, or current activity.

Make the operation visible even when it was not automatically activated: lobby = **Scan conversations**;
office or team = **Scan this level**; room or selected-event view = **Scan this level** or **Change lens**.
The same open projection language applies at office, team, room, and selected-event scope.

## Office Scan

An **Office Scan** applies a lens to a bounded authenticated population from `tree --json`.
The default population is the current hallway's direct children; recursive scope is explicit. Every
scan must state its **scope**, **question**, evidence boundary, mappings, filters, and counts for
**eligible**, **inspected**, **uninspected**, **omitted**, and **unknown** members. Include a compact
**readable legend**: labels must remain understandable without icons, and no claim may depend on
**color alone**. State topology-depth counts. Never rank an uninspected room negatively merely because
its tail was not retrieved.

Office-level examples include:

- reported claims × strongest evidence;
- room × recorded interruption kind;
- manager × observed handoff evidence;
- work-product kind × evidence strength;
- recorded timing gap × tool-interval observation.

These are suggestions, not an allow-list. A blank lens may expose available dimensions and ask the
human for a question, axes, grouping, filter, comparison set, or metaphor.

## Room drill-down and Back

Selecting a room from a scan **preserves the question**, focus, and valid filters while adapting the
population from rooms to events and aggregate marks to exact provenance. State the office placement
as bounded prior context. Room evidence may overturn that impression.

**Back** restores the **same Office Scan**, arrangement, population, and retained snapshot rather than
the generic hallway. Refresh retains the lens definition and human selection but reacquires evidence.

## Evidence vocabulary

Keep these distinct for every consequential claim:

- metadata only;
- agent report;
- inspected retained result;
- independent verification.

Attach qualifications to the claim they constrain. Producer completion is not task success;
background launch is not current activity; missing return does not prove running. Timestamps locate
observed intervals, not causes or execution time.

## Conversational operations

Accept operations such as: flip axes; group by manager; emphasize unknowns; remove cost; compare
reports with results; focus an event class; define another lens; suggest a challenging view; reset to
a blank lens; refresh; or drill into a selected room. Before redrawing, announce changed mappings.

The LLM owns lens translation and interpretation. CLI outputs own retrieval, identity, topology,
order, measured values, and snapshots.
