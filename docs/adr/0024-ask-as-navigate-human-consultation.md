# ADR-0024: `nn ask` as a human-consultation action inside `nn-navigate`

## Status

Accepted — revision in progress

## Date

2026-08-25

## Context

ADR-0020 established `nn ask` as a routed, one-shot human-feedback primitive: an
agent prepares a question and context, a human responds through a surface suited
to the shape of the answer, and the agent reads a thin result envelope and decides
what follows. ADR-0021 added a bounded graph surface for relational feedback.
ADR-0022 documented MCP/skill surfaces as an LLM-driven handshake. ADR-0023
established navigation actions as agent-level compositions with explicit look/move
semantics rather than new CLI verbs.

`nn-navigate` does not currently treat human consultation as an available move.
When a positioned walk reaches an interpretation, grouping, or structural judgment
that would benefit from human input, the agent must leave the navigation workflow,
construct an ad hoc `nn ask` invocation, and later reconstruct the retained focus,
goal, filters, history, bookmarks, and menu state. This makes Ask difficult to
discover and risks accidental focus or history changes when the result returns.

The surfaces have distinct decision value:

- **document** is best for annotating prose, evidence, routes, and proposed
  interpretations;
- **graph** is best for pointing at, selecting, grouping, and commenting on stored
  nodes and relationships;
- **canvas** is best for explaining within-note and cross-note structures that are
  not faithfully represented by notebook nodes and stored edges;
- **MCP/skill** surfaces remain concrete extension points when the agent can both
  populate and read back the surface.

The graph surface should not become a second implementation of the full
`nn-navigate` command grammar. Ordinary clicking already supports review. Its
special value is graph-native pointing: multi-selecting notes and edges, lassoing
sets, saving named groups, and commenting on selections that are awkward to
identify in prose.

## Decision

### 1. Add `Ask…` as a focus-retaining `nn-navigate` action class

`nn-navigate` gains a human-consultation action class represented as `◇ Ask…`.
It is a skill-level action that composes existing `nn ask` surfaces; it is not a
new `nn navigate` CLI command.

Ask has the following invariant:

> An Ask action suspends the positioned walk, delegates one bounded decision to a
> human, ingests the returned artifact, and resumes the same retained navigation
> frame.

Ask does not:

- change focus;
- push Back or clear Forward;
- modify bookmarks or active traversal filters;
- create or edit notes or links automatically;
- turn a human-selected destination into an implicit Recenter;
- persist navigation UI state in the notebook.

A result may justify a later Recenter, Peek, link proposal, note update, or Arrive,
but that action is proposed and executed separately under its normal contract.

The human-facing effect legend adds:

```text
◇ human consultation; retained focus and history
```

`Ask…` remains available under `All navigation actions…` and may be promoted into
the adaptive Quick actions only when human input could materially change the next
decision. It is not promoted when the next move is already unambiguous or the
expected decision improvement is smaller than the interruption cost.

### 2. Route by the decision the human must make, not by raw surface name

The Ask submenu uses decision-oriented actions:

```text
◇ Review or group this graph material
  Graph: select and annotate stored notes, relationships, and groups.

◇ Explain or critique these concepts
  Canvas: edit an explanatory within-note or cross-note structure.

◇ Annotate the reasoning
  Document: comment on prose, evidence, or a proposed route.
```

A concrete MCP/skill action may be promoted when a suitable integration is
available and exposes read-back. There is no generic enumerated MCP submenu and no
new `nn` MCP client, preserving ADR-0022.

### 3. Preserve the complete navigation frame across Ask

Before launching a surface, the agent retains the complete conversation-scoped
navigation frame:

```yaml
focus: note-id
goal: string
filters: {}
back: []
forward: []
bookmarks: {}
visited_evidence: []
menu_stack: []
```

The Ask lifecycle is:

```text
Positioned → AskPrepared → AwaitingHuman → ResultAvailable → Positioned
```

- **Positioned → AskPrepared:** create one explicit human decision, bounded
  evidence, and a selected surface.
- **AskPrepared → AwaitingHuman:** launch the existing synchronous `nn ask`
  surface or the ADR-0022 MCP/skill handshake.
- **AwaitingHuman → ResultAvailable:** receive the thin envelope or external
  read-back artifact.
- **ResultAvailable → Positioned:** interpret the native artifact, report the
  human finding, preserve the prior frame, and reopen the appropriate navigation
  menu. Cancellation follows the same return path with no state mutation.

The result is evidence supplied by the human, not notebook truth. Derived canvas
arrows, spatial grouping, and proposed graph relationships are labeled non-stored
until a separate mutation workflow applies them.

### 4. Define the graph Ask answer as annotated selections

Graph Ask remains a bounded hosted surface under ADR-0021. Its primary answer unit
becomes an annotated selection:

```yaml
selection:
  name: optional string
  classification: optional string
  nodes: []
  edges: []
  comment: string
```

The surface supports:

- explicit Inspect, Select, and Lasso affordances rather than overloading plain click;
- modifier-click as an expert shortcut to add or remove active-selection items;
- lasso selection over graph material;
- multiple saved, optionally named groups;
- editing and deleting saved groups;
- reselecting a saved group so its node and edge membership can be revised;
- group-level comments in addition to existing node, edge, and overall comments;
- optional lightweight classifications such as `relevant`, `evidence`, `tension`,
  `belong-together`, `needs-attention`, and `unclear`.

The answer-building lifecycle is `Select → Organize → Explain → Review`. Selection
is transient working state; saved groups are editable session-local answer objects.
Clearing a selection does not delete a saved group, and deleting a group never
mutates notes or links.

This lifecycle serves six durable jobs independent of the current controls: **bound the evidence**
under discussion; **express structure** rather than mere membership;
**correct the interpretation** before committing it; move from stored topology to
**spatial explanation** when useful; **preserve epistemic boundaries** between
notebook truth and derived meaning; and **return a coherent answer** another agent
can interpret without replaying the interaction. The surface must afford
**correction as strongly as accumulation**: editing, deleting, and reselecting are
first-class actions rather than exceptional recovery paths.

These classifications are feedback labels, not notebook link types.

Lasso uses screen-space rectangle geometry without turning that geometry into
notebook meaning. A node is directly selected when its rendered circle intersects
the rectangle. An edge is directly selected exactly when its rendered path
midpoint is contained by the rectangle or its visible relationship label
intersects the rectangle; a relationship segment merely crossing elsewhere is
not a direct selection.

The grouped artifact records those direct lasso hits as `selection: explicit` and
records derived context as `selection: implicit`. An unchosen connecting edge
between explicitly lassoed nodes is implicit unless its midpoint or label was
directly caught. Conversely, a directly caught edge includes either
endpoint outside the rectangle as an implicit node, so every selected
relationship remains interpretable. Layout geometry itself is not persisted: a
lasso identifies members, not a durable spatial region.

A graph result may contain multiple groups:

```yaml
groups:
  - id: group-1
    name: Core argument
    classification: belong-together
    nodes:
      - id: note-a
        selection: explicit
    edges:
      - source: note-a
        target: note-b
        type: supports
        selection: implicit
    comment: These should be reviewed as one argument.
overall_comment: optional string
handoff: null # or canvas or document
```

The result has exactly `groups`, `overall_comment`, and `handoff` at top level.
The strict decoder accepts exactly `null`, `canvas`, or `document`; it strictly rejects obsolete `handoffs` and `explain_on_canvas`, unknown top-level keys, and every other handoff value. Nothing in this schema mutates notes or relationships automatically.

#### Reopening a retained Graph Ask

The conversational shorthand is exactly `reopen graph ask`. When a prior Graph Ask is retained, the agent must rerun `nn ask --surface graph` with the same retained focus, exact bounded node scope, and same goal and instructions. It supplies those values through the existing `--focus`, `--nodes`, and `--instructions` inputs and opens a new clean surface. The previous result and groups are retained only as agent evidence and must not be silently injected into the new request or surface.

This is **not browser Back** and **not draft resumption**. It creates a fresh synchronous Graph Ask session; it does not navigate browser history, restore the prior `draft.json`, or continue the old selection UI. On terminal return, terminal completion or cancellation restores the exact navigation frame and invoking menu captured for the rerun without mutation. If there is no retained prior Graph Ask, report the action unavailable and invent nothing: the agent must not reconstruct a focus, scope, goal, instructions, result, or groups. This is conversation-level agent behavior and adds no CLI flag, Graph result schema, server route, or Graph frontend behavior.

### 5. Use explicit handoff intent for source-grounded review and explanation

Graph Ask exposes three mutually exclusive terminal buttons: **Send** → `handoff: null`; **Send to Canvas** → `handoff: canvas`; **Send to Document** → `handoff: document`. There is no persistent toggle or fallback state. Clicking any one submits exactly once and closes the Graph Ask surface. Graph emits only the one terminal intent and appropriate seed metadata. It never launches either destination surface.

#### Document handoff

Document is selected when the decision concerns prose, evidence, a route, or a
proposed interpretation. The Graph result carries `document` intent in `handoff`.
Document has no separate seed artifact. A seed would duplicate the
Graph result and the source-grounded brief that the agent must build, so omitting it
avoids unnecessary artifact proliferation.

After Graph Ask completes, `nn-navigate` must:

1. read the selected note bodies and stored edges for every selected or grouped
   source ID; Graph comments and group membership are context, not source evidence;
2. write a temporary Markdown brief with explicit sections for purpose, groups,
   source IDs and bounded excerpts, stored edge evidence, agent interpretation,
   uncertainties, and explicit review questions;
3. include a clear **NON_STORED/generated disclosure** that the brief, excerpts,
   grouping, interpretation, and proposed route are generated review material—not
   notebook content or newly stored relationships;
4. invoke `nn ask --surface document --document <temporary-brief.md>` with a bounded
   review purpose;
5. read the thin Document envelope and native decision artifact, interpret the
   Plannotator result as human evidence, and preserve the Graph artifact; and
6. remove the temporary brief when no longer needed and restore the exact
   snapshotted frame, including focus, goal, filters and traversal context, Back,
   Forward, bookmarks, visited evidence, current menu, and ordered menu stack.

Cancellation restores that frame and produces no mutation. Graph and its server do
not invoke Document; the agent owns this source-reading and review boundary.

#### Canvas handoff

Canvas is selected when the decision concerns grouping, argument structure,
sequence, boundaries, or another explanatory relation that the stored note graph
cannot express faithfully.

A canvas explanation may include:

- sections, claims, evidence, assumptions, and implications within one note;
- selected nodes and stored edges across several notes;
- named groups returned by Graph Ask;
- proposed or explanatory arrows clearly distinguished from stored edges.

When `canvas` is selected, submitting Graph Ask first returns the canonical native
`graph-selection` artifact and a `canvas-seed` artifact. Canvas-seed is preserved
only for Canvas intent: no canvas seed is emitted for `null` or `document`.
The deterministic seed preserves grouped IDs, explicit/implicit
membership, and comments needed for the handoff and records `storage: NON_STORED`;
the visual diagram carries the single diagram-level disclosure defined below
instead of repeating that marker on each element.

The Graph server and the `nn ask` CLI MUST NOT launch Canvas or Document directly.
They return Graph-native evidence, one intent, and a canvas-seed artifact only for `canvas`.
Direct launch would wrongly move source interpretation into the CLI and would
prevent the agent from enriching note-level topology with body-level structure.

`nn-navigate` owns the subsequent agent-mediated answer-composition handoff, not a
direct mechanical graph conversion and not a focus change:

```text
Graph result → nn-navigate reads graph-selection + canvas-seed
             → nn-navigate reads every selected note body and stored relationship
             → nn-navigate derives explanatory Mermaid with in-node structure
             → nn ask --surface canvas --mermaid <diagram>
             → nn-navigate reads the native Canvas artifact
             → nn-navigate restores the exact snapshotted navigation/menu frame
```

The Mermaid must be capable of exposing sections, claims, evidence, assumptions,
and implications inside notes rather than reproducing only note IDs and stored
edges. Use real newlines inside Mermaid Markdown-string labels rather than literal
`<br>` so Canvas creates editable multiline text with correctly computed enclosing
geometry. Each explanatory diagram includes exactly one unobtrusive diagram-level
explanatory note:

> Illustrative layout and inferred relationships — not literal notebook structure. Source IDs identify evidence; only edges explicitly marked STORED are notebook edges.

Source note IDs retain visible provenance and copied notebook edges are explicitly
marked `STORED`. Do not repeat `NON_STORED` on each derived node or relation: the
single note explains that arrows, containment, ordering, proximity, grouping, and
layout are inferred unless an edge is marked `STORED`.

To the human, **Send to Canvas** records one coordinated handoff intent, but its two
Ask sessions and the agent-owned enrichment boundary remain explicit. `nn-navigate`
invokes the existing Mermaid pathway, waits synchronously, reads the returned
native Canvas artifact rather than only a screenshot, and then restores the exact
frame values captured before Graph Ask: focus, goal, filters and traversal context,
Back, Forward, bookmarks, visited evidence, current menu, and ordered menu stack.
Cancellation restores that same frame without mutation. The handoff produces
editable native Canvas content rather than a raster-image-only result and never
turns generated structure into notebook links.

Mermaid conversion remains authoritative for element geometry, IDs, bindings,
files, and editable text. For a new seed, Canvas mounts an empty editor, waits for
the imperative Excalidraw API and drawing fonts, then parses Mermaid and passes
the unchanged `convertToExcalidrawElements` result to the public scene API. It
adds every returned helper file before `updateScene`, performs no manual glyph
measurement, padding, resizing, or container mutation, and fits the complete
seed exactly once. A restored draft remains authoritative initial data: Canvas
does not rerun Mermaid conversion or refit it.

Canvas cancellation is terminal for the hosted browser lifecycle. It stops pending
and in-flight autosave, requests server cancellation, and attempts `window.close()`.
When browser security blocks closing a tab that was not opened by script, the page
renders a stable cancelled state with editing actions disabled. A shutdown-racing
draft request must never replace that state with a draft error.

### 6. Keep native artifacts and agent interpretation

This ADR does not introduce a cross-surface semantic result schema. Each surface
continues to write its native artifact under ADR-0020's thin envelope. The
`nn-navigate` skill owns preparation, interpretation, frame restoration, and any
subsequent proposed action.

Graph's group model is native to Graph Ask; it does not force document or canvas
results into the same shape.

## Consequences

- Human consultation becomes discoverable without adding a new CLI navigation
  surface.
- Ask is explicitly neither a look nor a move: it delegates a decision while
  preserving the positioned frame.
- Graph Ask gains a distinctive purpose—pointing at and grouping stored graph
  material—rather than duplicating conversational navigation.
- Canvas can explain within-note and cross-note structures while preserving the
  distinction between stored graph facts and derived explanation.
- Document remains the cheapest general-purpose surface for detailed review.
- The agent must retain navigation and menu state while blocked on human input.
- Ask promotion introduces interruption cost, so the skill must justify promoted
  consultation with a concrete decision that human input could change.

## Implementation order

1. Update `nn-navigate` with the `◇ Ask…` action class, lifecycle, frame-retention
   rules, promotion criteria, and return behavior.
2. Add document-backed `Annotate the reasoning` using a source-grounded temporary
   Markdown brief and Plannotator interpretation.
3. Add graph-backed `Review or group this graph material` using the retained
   focus and a bounded node allowlist.
4. Extend Graph Ask with multi-select, lasso, named groups, group comments, and a
   graph-native result artifact.
5. Add canvas-backed `Explain or critique these concepts`, including
   within-note/cross-note seed conventions.
6. Add editable saved-group lifecycle: edit, delete, reselect, revise membership,
   and save changes.
7. Add the agent-mediated Graph → LLM → Mermaid → Canvas handoff, preserving the
   Graph artifact and navigation frame while applying the single diagram-level
   epistemic disclosure and marking only copied notebook edges `STORED`.
8. Add a Review stage that summarizes groups, relationships, Canvas artifacts, and
   the no-notebook-mutation boundary before submission.
9. Permit concrete MCP/skill actions only when capability and read-back are known.

## Alternatives considered

**Expose the full `nn-navigate` action grammar inside Graph Ask:** Deferred. It
would duplicate focus/history semantics in a second UI and obscure Graph Ask's
stronger native role as a pointing and grouping surface.

**Make Graph Ask primarily choose the agent's next move:** Rejected. Selecting a
next destination is one possible downstream use, but the graph surface provides
broader value by identifying and annotating sets of stored material.

**Expose only raw surface names (`Graph`, `Canvas`, `Document`):** Rejected. The
human decision should lead; the surface is an implementation choice determined by
the answer shape.

**Standardize one result schema across every Ask surface:** Rejected, preserving
ADR-0020. Native artifacts retain information better, and the agent already owns
interpretation.

**Automatically apply human-selected links or groups:** Rejected. Human feedback
is evidence; mutation remains a separate explicit operation.
