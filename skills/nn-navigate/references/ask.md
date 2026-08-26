---
name: ask
applies_when: "When Ask is available, promoted, launched, cancelled, reopened, or hands Graph evidence to Canvas or Document."
---

# Reference: Ask

### `◇ Ask…` — bounded human consultation with a retained frame

Ask suspends the positioned walk for one explicit human decision and composes an existing `nn ask` surface; it is a skill-level action, never an `nn navigate` subcommand. Its invariant is: **suspend one bounded decision, ingest the native human artifact as evidence, then resume the same complete navigation frame**.

The decision-oriented Ask submenu under `<short-id> · Quick actions › All actions › Ask` has exactly:

1. `◇ Review or group this graph material`
   **Graph:** select and annotate stored notes, relationships, and groups.
2. `◇ Explain or critique these concepts`
   **Canvas:** edit an explanatory within-note or cross-note structure.
3. `◇ Annotate the reasoning`
   **Document:** comment on prose, evidence, or a proposed route.

Route by the decision the human must make, not by a raw surface name. A concrete MCP/skill action may be promoted only when the integration is available, can be populated, and exposes deterministic read-back; never enumerate a generic MCP submenu or invent a new `nn` MCP client.

Before launching Ask, snapshot the complete conversation-scoped frame and menu UI state, not just the focus ID:

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

Retain active direction, link-type, status, representation, depth, route, and impact traversal context inside `filters` or the applicable complete frames. Ask follows this lifecycle exactly:

```text
Positioned → AskPrepared → AwaitingHuman → ResultAvailable → Positioned
```

- **Positioned → AskPrepared:** state one decision the human can change, include only bounded visited evidence, select the cheapest faithful surface, and retain the invoking menu plus ordered menu stack.
- **AskPrepared → AwaitingHuman:** launch the existing synchronous `nn ask` surface, or perform the concrete ADR-0022 MCP/skill prepare/launch/read-back handshake.
- **AwaitingHuman → ResultAvailable:** read the thin envelope and its native artifact; do not flatten document, graph, canvas, or external artifacts into a cross-surface schema.
- **ResultAvailable → Positioned:** report the human finding as evidence, restore the byte-for-byte-equivalent frame values, and reopen the invoking Ask submenu. If Ask was a promoted top-level shortcut, return to Quick actions instead. Cancellation follows this same return path and menu rule with no state mutation.

Ask does not change focus; does not push Back or clear Forward; does not modify bookmarks, filters, traversal context, or visited evidence automatically; does not create or edit notes or links; and does not turn a selected note into an implicit Recenter. It must not persist navigation/menu UI state in notebook files, frontmatter, the index, or Git. A result can justify a later Recenter, Peek, mutation proposal, or Arrive only as a separately proposed action under that action's normal contract.

Surface preparation and interpretation rules:

- **Document — Annotate the reasoning:** generate a temporary navigation brief from bounded source evidence and launch Document Ask against that brief. Comments and edits are human evidence; the temporary brief is not notebook content.
- **Graph — Review or group this graph material:** use the retained focus and an explicit bounded node allowlist. The graph-native result has exactly `groups`, `overall_comment`, and `handoff` at top level; note and edge comments live on members inside groups. The strict decoder accepts exactly `null`, `canvas`, or `document` and strictly rejects obsolete `handoffs` and `explain_on_canvas`, unknown top-level keys, and every other handoff value. The graph answer may contain multiple saved, optionally named groups with group comments and lightweight feedback classifications. Modifier-click adds/removes notes or relationships; lasso, when available, identifies members rather than durable geometry. A lasso directly catches every node whose rendered circle intersects its rectangle. It directly catches an edge exactly when the rectangle contains the rendered path midpoint or intersects the visible relationship label; a relationship segment merely crossing elsewhere is not directly selected. Each group records node and edge membership. Mark direct lasso hits `selection: explicit`. An unchosen connecting edge between explicitly lassoed nodes is `selection: implicit` unless its midpoint or label was directly caught; a directly caught edge pulls any endpoint outside the rectangle into the group as an implicit node. This distinguishes explicitly selected edges from edges included implicitly, keeps relationships interpretable, and preserves direct-versus-derived intent. Classifications such as `relevant`, `evidence`, `tension`, `belong-together`, `needs-attention`, and `unclear` are feedback labels, never notebook link types.
- **Canvas — Explain or critique these concepts:** seed sections, claims, evidence, assumptions, and implications within a note, or selected notes/stored edges/named Graph groups across notes. Clearly distinguish copied stored-edge references from explanatory arrows, containment, ordering, proximity, and grouping. Every seeded explanatory diagram has exactly one unobtrusive diagram-level explanatory note with this text: **Illustrative layout and inferred relationships — not literal notebook structure. Source IDs identify evidence; only edges explicitly marked STORED are notebook edges.** Mark copied notebook edges `STORED`, and do not repeat `NON_STORED` on each node or relation; canvas geometry is evidence supplied by the human, not notebook meaning.

#### Reopen a retained Graph Ask

The conversational shorthand is exactly `reopen graph ask`. When a prior Graph Ask is retained, rerun `nn ask --surface graph` with the same retained focus, exact bounded node scope, and same goal and instructions. Use the existing `--focus`, `--nodes`, and `--instructions` inputs to create a new clean surface. The previous result and groups are retained only as agent evidence and must not be silently injected into the new request or surface.

This is **not browser Back** and **not draft resumption**: it starts a fresh synchronous Graph Ask session rather than reopening a page, restoring `draft.json`, or continuing the old selection UI. On terminal return, terminal completion or cancellation restores the exact navigation frame and invoking menu captured for the rerun, with no focus, history, bookmark, filter, traversal, visited-evidence, or notebook mutation. If there is no retained prior Graph Ask, report the action unavailable and invent nothing—do not reconstruct a focus, scope, goal, instructions, result, or groups. This shorthand is agent-owned conversation behavior; it introduces no new CLI flag, Graph result schema, server route, or Graph frontend behavior.

Graph Ask offers three mutually exclusive terminal buttons: **Send** → `handoff: null`; **Send to Canvas** → `handoff: canvas`; **Send to Document** → `handoff: document`. There is no persistent toggle or fallback state. Clicking any one submits exactly once and closes the Graph Ask surface. The graph-selection artifact remains authoritative, with exactly `groups`, `overall_comment`, and `handoff` at top level. Graph emits a canvas-seed artifact only for `canvas`, labeled `NON_STORED`. Document has no separate seed artifact because the later temporary brief supplies the needed context without artifact proliferation. The Graph server and the `nn ask` CLI MUST NOT launch Canvas or Document directly.

When `canvas` intent is returned, `nn-navigate` owns this separate agent-mediated handoff while keeping the walk suspended:

1. Read the thin Graph result envelope, graph-selection artifact, and canvas-seed artifact. Preserve the Graph artifact unchanged.
2. `nn-navigate` MUST read the selected note bodies and stored relationships for every selected or grouped note ID; group names, comments, classifications, and explicit/implicit membership are context, not substitutes for source bodies.
3. From those bodies, derive Mermaid capable of representing in-node structure—sections, claims, evidence, assumptions, and implications—not merely note IDs and stored graph topology. Copied stored-edge references remain visibly distinct from explanatory structure. Use real newlines inside Mermaid Markdown-string labels rather than literal `<br>` so Canvas receives editable multiline text and can compute enclosing geometry.
4. In the Mermaid text, use the single Canvas disclosure specified above, retain source IDs as provenance, and mark only copied notebook edges `STORED`. Do not stamp every derived node or relationship with a repeated warning; the diagram-level note owns that boundary.
5. Invoke `nn ask --surface canvas --mermaid` with that derived Mermaid and a bounded explanation/critique decision. This is the only Canvas launch in the handoff.
6. After the synchronous Ask completes, read the returned Canvas artifact as native human evidence, then restore the exact snapshotted frame, including focus, goal, filters and traversal context, Back, Forward, bookmarks, visited evidence, current menu, and ordered menu stack. Reopen the invoking Ask submenu, or Quick actions when Ask was promoted.

When `document` intent is returned, `nn-navigate` owns a separate source-grounded handoff:

1. Read the selected note bodies and stored edges for every selected or grouped source ID; Graph comments, classifications, and membership never substitute for notebook evidence.
2. Write a temporary Markdown brief with explicit sections for purpose, groups, source IDs and bounded excerpts, stored edge evidence, agent interpretation, uncertainties, and explicit review questions. Include a clear **NON_STORED/generated disclosure** stating that the brief, grouping, excerpts, interpretation, and proposed route are generated review material rather than notebook content or new stored relationships.
3. Invoke `nn ask --surface document --document <temporary-brief.md>` with the bounded review purpose. Graph itself does not invoke Document.
4. Read the thin Document result envelope and native decision artifact, interpret the Plannotator result as human evidence, and preserve the Graph artifact unchanged.
5. Remove the temporary brief when no longer needed, then restore the exact snapshotted frame, including focus, goal, filters and traversal context, Back, Forward, bookmarks, visited evidence, current menu, and ordered menu stack. Reopen the invoking Ask submenu, or Quick actions when Ask was promoted.

A selected answer-composition handoff does not change focus, mutate history, launch an implicit Recenter, or store arrows/groups as notebook links. Cancellation at either Ask boundary restores the same exact frame without launching or mutating anything further. On the hosted Canvas page, Cancel stops autosave, requests server cancellation, and attempts to close the browser window; if browser security blocks `window.close()`, a stable terminal cancelled page replaces the live session and must not surface a shutdown-racing draft error.

Ask promotion has an interruption cost. Promote one concrete `◇ Ask — <decision>` shortcut only when human input could materially change the next decision, name what answer would change, and use bounded evidence to justify the interruption. Generic uncertainty is insufficient. Regardless of whether Ask was promoted, completion and Cancellation restore the retained frame; reopen the invoking Ask submenu for submenu invocation and Quick actions for a promoted invocation.

Every picker and submenu has at most four rows. Esc or a declined chooser in a submenu returns to the parent menu by popping only the UI menu stack, without mutating focus, graph history, notes, links, the goal, filters, traversal context, or notebook content. **Esc means parent menu; conversational `Back` means previous graph frame.** Never render an explicit `Back` row. **Esc at Quick actions closes only the picker; focus and graph history remain retained.** When relaying that dismissal, say the picker closed and name the retained focus plus the `navigate` resume affordance; do not imply Arrive or navigation-state loss. During cold Teleport, before any positioned focus exists, use the bounded landing chooser described below; after landing, Orient and use this adaptive picker.
