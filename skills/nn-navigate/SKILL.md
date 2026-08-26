---
name: nn-navigate
description: "Use when a human is iteratively navigating the nn graph or asks to teleport, orient, recenter, peek, scan, ask for human consultation, arrive, use Back/Forward history, or manage navigation bookmarks. Load the compact core with `nn skills get nn-navigate`, then fetch each applicable owning reference."
when_to_use: "When graph exploration is human-driven or iterative and must retain a positioned focus, including teleport, orient, recenter, peek, scan, Ask consultation, arrive, history, bookmarks, and navigation resumption."
---

# nn-navigate

Binding dispatcher for iterative, human-driven navigation of the nn note graph. This compact core owns activation, reference dispatch, the retained state model, focus mutation, Orient, the presentation gate, and compaction safety. Detailed action semantics remain binding in lazy references; they are not optional background reading.

## Preflight and activation

Activate this skill when a human is co-navigating a positioned graph walk, exploration proceeds iteratively across turns, or the request uses `teleport`, `orient`, `recenter`, `peek`, `scan`, `arrive`, `navigate`, Ask, Back/Forward, history, or bookmarks as navigation actions.

Before human-driven iterative navigation, if the session inventory has not run, run:

```bash
nn skills list
```

Then load this core:

```bash
nn skills get nn-navigate
```

Use `nn-guide` only for command syntax and command semantics. `nn-navigate` is the sole owner of the human navigation workflow, presentation contract, and conversation-scoped navigation state. None of the named navigation verbs creates a new `nn navigate`, `nn teleport`, `nn orient`, `nn recenter`, `nn peek`, `nn scan`, or `nn arrive` CLI surface; the agent composes existing commands.

## Binding lazy-reference rule

Before executing any applicable action, MUST fetch every owning reference, unless that exact reference from this exact skill version has already been fetched in the current uncompacted context:

```bash
nn skills get nn-navigate --reference <name>
```

Discover the stable reference inventory and applicability when needed:

```bash
nn skills get nn-navigate --list-references
```

Do not guess a reference path, read a packaged path directly, or assume the compact core contains the omitted contract. Do not concatenate every reference preemptively. Fetch the smallest owning set before the action, and fetch all owners when one action crosses boundaries. A summary, prior version, dispatch paragraph, broad `nn-guide` entry, or virtual protocol is not a substitute for the owning reference.

| Action or seam | Owning reference(s) that MUST be fetched before acting |
|---|---|
| Any human-facing Focus + Map + Moves view, chooser, return, relay color/legend, semantic direction label, or presentation validation | `presentation` |
| Ask preparation, launch, cancellation, retained Graph Ask reopening, Graph result interpretation, or Graph→Canvas/Document handoff | `ask`; also `state`; fetch `presentation` before reopening a picker |
| Enter, Orient, Recenter, Peek, Teleport, Arrive, or conversational `navigate` resume | `movement`; also `presentation`; fetch `state` when a retained frame/history is read or changed |
| Local territory, Global landscape, typed destination discovery, typed impact, or typed path witnesses | `scan-and-routes`; also `presentation`; fetch `movement` before a resulting Recenter/Arrive |
| Show verbatim, Explain in depth, Analogize, Find an analog, Find gaps, Visualize, or Quiz | `lenses`; also `presentation`; fetch `state` if recovery or retained-frame validity is at issue |
| Teleport/Visit/Recenter/Go to history mutation, Back, Forward, Bookmark, Where am I?, Ask frame preservation, unknown-state recovery, or compaction | `state`; fetch `movement` and `presentation` before restoring a positioned view |

Reference ownership is action ownership. Execute no specialized detail from memory when its owner has not been fetched. If an action becomes applicable only after inspecting evidence—for example, a Scan reveals a Recenter destination—pause at that boundary, fetch the newly applicable owner, and only then act. If compaction removed a fetched reference from active context, treat it as unfetched and retrieve it again.

## Retained navigation and menu state model

Navigation state is conversation-scoped and never notebook state. Retain complete frames, not bare IDs. The minimum live model is:

```yaml
current_frame:
  focus: note-id
  goal: string
  filters: {}
  visited_evidence: []
back: []
forward: []
bookmarks: {}
menu_ui:
  current_menu: Quick actions
  menu_stack: []
```

Every frame carries enough active traversal context to reproduce the positioned view. Preserve direction, link types, status, representation, depth, route witness, impact types/direction/depth, destination query, and any other active graph constraint in `filters` or an explicitly typed equivalent. `visited_evidence` is bounded source evidence retained for interpretation; it is not permission to turn generated content into notebook fact.

Back and Forward contain complete frames in stack order. Every bookmark maps its case-sensitive name to a complete saved frame. Menu UI is a separate ordered stack: changing or dismissing a menu does not change graph history, and graph movement does not masquerade as a menu pop. None of this state may be written to note bodies, frontmatter, links, SQLite, protocol notes, or Git.

Ask snapshots and restores the same complete frame plus the invoking menu and ordered menu stack. Lenses, Peek, and Scan read or interpret retained evidence without changing focus or graph history. Failed, cancelled, unavailable, and complete-frame no-op actions retain all state unless an owning reference explicitly defines a non-state side effect.

If state is missing or incomplete, say it is unknown. Never reconstruct history, filters, traversal witnesses, bookmarks, visited evidence, the current menu, or a menu stack from notebook content or plausible conversation clues. Establish a new landing or ask the human to restate recoverable state.

## Focus mutation invariants

Teleport, Visit, Recenter, and Go to may adopt a new destination; Back and Forward may change focus only by restoring a retained frame. No other action changes focus.

A successful destination-adopting move pushes the prior complete current frame onto Back and clears Forward. An initial cold landing has no prior frame to push. Back moves the current frame to Forward and restores the latest Back frame; Forward performs the symmetric operation. A destination equal to the complete current frame is a no-op. Failed resolution, empty stacks, unknown bookmarks, declined bookmark replacement, Ask, Peek, Scan, lenses, menu navigation, picker dismissal, Arrive, and `Where am I?` never mutate graph history.

Recenter walks one explicit next hop. A typed witness does not permit jumping to the final destination while claiming to walk the path: select a witness, move to its next node, and Orient again. Teleport is the explicit far relocation. Ask never turns a selected note or human artifact into an implicit Recenter. Generated analogies, layouts, groups, classifications, arrows, and human comments never create notes or links automatically.

After every successful focus-changing move, rerun Orient, reset menu UI to Quick actions, and apply the full presentation gate. Arrive stops but retains the final focus and frame. Conversational `navigate` resumes from the retained focus; it does not search for a different entry point merely because the picker had closed or discussion intervened.

## Orient is the canonical positioned read

After a landing or restoration, and whenever an owner requires Orient, acquire the positioned frame topology first, then its lossless body pages. On a color-capable human relay run:

```bash
nn graph show --focus <id> --depth 1 --direction both --zones --presentation-hints --color always --format text
nn graph bodies --focus <id> --depth 1 --direction both --page 1
```

If active link, status, or representation filters exist, append the identical filters to both commands. Read `pages`, `next_page`, and `snapshot` from body page 1, then retrieve every body page in order with the identical traversal:

```bash
nn graph bodies --focus <id> --depth 1 --direction both --page <N> --snapshot <snapshot-from-page-1>
```

Every later page MUST repeat the page-1 snapshot and page count. Verify every selected ID has complete one-based segment ordinals, concatenate its decoded UTF-8 body fragments in order, and retain the explicit empty-body record. A stale or mismatched snapshot means the notebook or traversal changed: discard the partial body batch and restart topology first. The agent MUST NOT make body-derived claims, central-claim summaries, recommendations, or concrete action reasons until every body page has arrived and every body has reconstructed completely. Topology-only facts may be identified while paging, but the body-dependent Focus + Map + Moves presentation remains blocked.

These commands are data sources, not the human-facing response. `graph show` supplies direct directional zones, degree, type/edge markers, and relay budgets; `graph bodies` supplies only IDs, segment ordinals, and stored Markdown body bytes. Preserve that metadata boundary: never treat frontmatter, links, topology fields, snapshot/page metadata, relay hints, or agent commentary as stored prose. Read both sources, then compose Focus + Map + Moves under the `presentation` reference. Never paste raw output as the navigation view. On a surface that cannot render terminal color, retain the stable textual emoji marker/meaning triples; `--color always` is still required for a tool-to-human color-capable relay because tool stdout is commonly non-TTY.

Orient must establish:

- the retained focus's ID, readable title, type, status, degree, body-derived central claim, and neighborhood role;
- direct neighbors and empty zones under their complete local relationship meanings;
- every body page under one snapshot and complete reconstruction for the selected topology IDs;
- bodies and evidence needed to judge concrete next actions relative to the retained goal;
- whether active filters, typed route/impact context, or notebook changes invalidate a cached view; and
- the evidence required to justify at most the contextual shortcut budget rather than promoting generic availability.

If no focus exists, Orient is structurally unavailable. Use the `movement` reference's cold Enter/Teleport contract; do not invent a focus. If a retained frame exists after discussion, `navigate` reruns Orient and resets Quick actions. A compact unchanged transient return may reuse a complete frame already visible only under the exact invalidation and return rules in `presentation`; it may not degrade into a skeletal map.

## Positioned action dispatch

Every human-driven positioned step keeps discoverable these classes:

1. **Recenter — move focus**
2. **Peek — inspect without moving**
3. **Scan — zoom out without moving**
4. **◇ Ask… — suspend for one bounded human decision while retaining the complete frame**
5. **■ Arrive — stop and retain focus**

Peek and Scan are always discoverable unless structurally impossible with the reason stated. Ask is neither look nor move. Arrive remains the final top-level action. A chooser-capable human surface uses the adaptive hierarchy owned by `presentation`; an autonomous or non-interactive run chooses actions from the retained goal without invoking a picker, while still making the action classes discoverable in prose.

Treat `show`, `explain`, `analogize`, `find an analog`, `visualize`, `quiz`, `scan`, `ask`, `arrive`, and `navigate` as direct conversational intents when their preconditions hold. Do not force a human to traverse picker categories before honoring a direct intent. A bare Scan asks for its owned altitude choice; an explicit Local territory or Global landscape request executes that choice. `navigate` is the universal conversational resume/escape and never an `nn` subcommand.

Dispatch sequence:

1. identify the retained frame and goal without mutating them;
2. classify the requested action or seam;
3. fetch every owner from the table above;
4. Orient if required by the movement/state owner or if the cache is absent, stale, or invalidated;
5. run the Navigation presentation blocker checklist before any human-facing positioned view or picker;
6. execute exactly one selected semantic action under its owner;
7. apply the owner's focus/history and deterministic menu-return rules; and
8. report unavailable actions without synthesizing evidence or state.

An action crossing seams repeats dispatch at each boundary. Examples: Scan→Recenter fetches `scan-and-routes` for discovery, then `movement` and `state` before moving, and `presentation` before relay. Graph Ask→Canvas fetches `ask` and `state`; its eventual picker return also fetches `presentation`. A lens finding that merely suggests a link remains non-mutating; any later notebook mutation dispatches to the applicable non-navigation workflow separately.

## Navigation presentation blocker checklist

Fetch `presentation` before evaluating or rendering this gate. Before every human-facing positioned view or chooser, internally verify all applicable items:

- [ ] Focus, Map, and Moves appear in that stable order unless an unchanged compact return is explicitly permitted by `presentation`.
- [ ] topology was fetched first and every body page repeated one snapshot and reconstructed completely
- [ ] no body-derived claim was formed before the complete body batch passed that check
- [ ] focus ID, readable title, type, status, and degree shown
- [ ] focus central claim summarized from its stored body
- [ ] focus neighborhood role explained relative to the retained goal
- [ ] zone/type/edge color markers use the stable relay palette
- [ ] zone positions carry their marker and complete local relationship meaning
- [ ] a visible legend/key explains every carried color, note-type, and edge-family marker
- [ ] positional map rendered rather than raw command output relayed
- [ ] edge meanings and stored directions visible
- [ ] empty zones state both their semantic meaning and what emptiness means here
- [ ] every visible non-focus node has ID, readable title, and body-derived claim
- [ ] summaries are degree-scaled, with expanded treatment for load-bearing neighbors
- [ ] compact labels have an immediately adjacent complete legend; no orphan, ID-only, or title-only nodes
- [ ] every directional human-facing label carries marker, zone name, and local relationship meaning together
- [ ] every concrete quick action states its effect on focus and an evidence-derived reason
- [ ] Recenter available exactly one picker level away when structurally possible
- [ ] Peek available exactly one picker level away
- [ ] Scan available exactly one picker level away
- [ ] ◇ Ask available one level away, and promoted only for a concrete bounded decision justified by evidence
- [ ] Lenses available at the stable top-level row
- [ ] Arrive always visible as final top-level action
- [ ] chooser row limit, breadcrumb, adjacent effect legend, and deterministic return satisfy `presentation`

A missing item blocks presentation unless structurally unavailable and the response explains why. Do not weaken the gate at Teleport landing, Orient, Peek/Lens return, Scan, Back/Forward restoration, `navigate` resume, or Arrive. JSON is marker-free input: the agent manually applies the relay palette without claiming those markers were in JSON. Width pressure permits compact map labels only with an immediately adjacent complete legend.

The complete semantic triples, stable palette, effect markers, menu rows, summary budgets, transient-render invalidation rules, and full Arrive presentation are owned by `presentation`; this checklist enforces their presence but does not replace fetching that reference.

## State, menu, and action return boundaries

Graph focus/history and menu UI are orthogonal. Esc or a declined submenu chooser pops only menu UI. Esc at Quick actions closes only the picker and retains focus/history; report the retained focus and the `navigate` resume affordance rather than implying Arrive. Conversational Back restores graph history and is never an explicit picker row.

Transient actions return to their invoking menu under `presentation`: Peek details return to Peek, lenses to Lenses, scans to Scan, promoted transients to Quick actions, and Ask to its invoking Ask submenu unless promoted. Focus-changing actions, Back/Forward, Go to, Teleport, and `navigate` rerun Orient and reset Quick actions. Arrive stops with focus retained. Quiz may suspend the picker while unanswered; its `navigate` escape resumes through Orient without grading or revealing.

Do not redundantly full-render an unchanged complete frame when the owner permits compact return. Do full-render after focus/filter/traversal/notebook changes, stale or unknown cache, explicit Refresh, or resumption after discussion. Compact return references the complete frame still visible; it cannot redraw a reduced skeleton or omit the navigation affordance.

## Compaction dispatch and recovery

Before compaction, fetch `nn skills get nn-navigate --reference state` and follow it. The handoff MUST serialize the full current frame, Back and Forward stacks in order, every bookmark with its case-sensitive name and complete saved frame, bounded visited evidence, and a separate menu UI state containing the current menu and ordered menu stack. Preserve active direction, link types, status, representation, depth, route, impact, destination-query, and other traversal context—not only the focus ID.

Also record which reference contracts remain actively needed after compaction, but never claim their full text survives: after compaction, rerun this core and refetch every applicable owner before acting. A handoff summary is dispatch evidence, not a replacement for a reference.

If navigation state is absent or incomplete after compaction, report it unknown and never invent it. If only menu UI is absent, retain known graph state but reset menu state only through explicit Orient, a focus-changing move, or conversational `navigate`; do not reconstruct a prior submenu. If all state is absent, ask for a query/new landing or a human restatement. Notebook notes, protocol content, and Git history are never recovery stores for conversational navigation state.

## Epistemic and mutation boundary

Files remain notebook truth and the index remains cache. Navigation displays may combine stored bodies and stored edges with agent-composed maps, relay markers, route choices, human artifacts, analogies, and layouts. Keep those categories explicit. Human feedback is evidence, not notebook truth. Derived geometry, grouping, containment, proposed routes, analogical correspondence, and explanatory arrows remain non-stored unless a separate explicit mutation workflow applies a valid note or typed link in its own operation.

Read selected bodies and stored relationships whenever an owning reference requires source-grounded interpretation. IDs establish provenance but do not substitute for bodies. Search or similarity establishes relevance candidates, not relationship proof. Typed route/impact witnesses preserve stored edge semantics and are movement evidence, not permission to infer unstored closure.

At every seam: retain state first, fetch ownership second, read bounded source evidence third, present only after the blocker passes, and mutate focus or notebook truth only under the action that explicitly owns that mutation.
