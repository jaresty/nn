---
name: nn-navigate
description: "Use when a human is iteratively navigating the nn graph or asks to teleport, orient, recenter, peek, scan, arrive, use Back/Forward history, or manage navigation bookmarks. Load with `nn skills get nn-navigate`."
when_to_use: "When graph exploration is human-driven or iterative and must retain a positioned focus, including teleport, orient, recenter, peek, scan, arrive, history, and bookmarks."
---

# nn-navigate

Detailed owner for iterative, human-driven navigation of the nn note graph.

## Preflight and activation

Activate this skill when a human is co-navigating a positioned graph walk, when exploration proceeds iteratively across multiple turns, or when the request uses `teleport`, `orient`, `recenter`, `peek`, `scan`, `arrive`, Back/Forward, history, or bookmarks as navigation moves.

Before human-driven iterative navigation, if you have not yet run the session skill inventory, run:

```bash
nn skills list
```

Then load this owner before navigating:

```bash
nn skills get nn-navigate
```

Use `nn-guide` only to look up command syntax or semantics. This skill owns the navigation workflow, human presentation contract, and conversation-scoped navigation state.

### Navigation mode: the zoned navigator with contents

`nn graph show --focus <id> --zones --bodies --color always` is a self-contained **navigation view**: it shows a note's neighbors bucketed by directional zone *and* prints each neighbor's tags and body, degree, and zone-colored headers — so you can both see where you can move and read where you'd land — in one call, without a separate `nn show` per node. Use `--color always` when presenting to a human on a color-capable surface.

Use it to walk the graph as a positioned space rather than dumping the whole structure. Hold a **goal** in mind for the walk ("find what contests this design", "trace where this claim came from", "reach the principle underneath") — every step is judged relative to that goal, not to the whole graph.

## Human-driven navigation invariant

Every presented navigation step **MUST KEEP DISCOVERABLE** four action classes:

1. **Recenter — move focus**
2. **Peek — inspect without moving**
3. **Scan — zoom out without moving**
4. **Arrive — stop**

An action may be unavailable only when structurally impossible, and the response **MUST** state why. **Peek and Scan MUST be discoverable** in a human-driven walk. In a chooser-capable harness, all four classes are available through `All navigation actions…` exactly one level below the top picker; contextual shortcuts may also promote concrete actions to the top.

Before every picker, present:

A. **Focus** — type, status, degree, central claim, neighborhood role
B. **Map** — positional zones, edge meanings, empty zones
C. **Moves** — degree-scaled neighbor summaries and recommended direction

This **MUST PRESENT** rule applies before the mode-specific details below; Teleport, Peek, Scan, and Recenter all reference it rather than weakening it.

## Adaptive hierarchical quick-actions picker

When a human is driving a positioned walk and a chooser is available, the **top-level picker has at most four rows**: **up to two evidence-backed contextual concrete shortcuts**; a stable **`Lenses…`** row; and the **final row is always `All navigation actions…`**. Fewer than two shortcuts is correct when the retained evidence does not justify more. Generic availability is not evidence, and action classes must not be promoted merely to fill rows.

Every promoted shortcut:

1. names its action class (`Recenter`, `Peek`, `Scan`, or `Lens`);
2. states whether selecting it changes or retains focus; and
3. gives a body- or evidence-derived reason why this concrete action matters now.

A promoted shortcut is concrete, such as a named destination, exact note body, specific explanation, particular lens, or bounded scan. Selecting it executes directly rather than opening its category submenu. A specific promoted `Lens` action may occupy a contextual shortcut slot when its evidence criterion is met, and the stable `Lenses…` row remains present. Any directional shortcut or reason still includes the complete semantic triple required below.

Use exactly these promotion criteria:

- Analog candidate survives correspondence mapping, where it holds, and where it breaks.
- Visualize when the evidence has process, state, or relationship structure.
- Quiz when the sources support a consequential distinction.
- Show verbatim when exact wording matters.
- Explain in depth when the focus is connected, contested, complex, or load-bearing.
- Recenter when a specific destination clearly advances the retained goal.

`All navigation actions…` opens the action-class submenu exactly one level away:

1. `Recenter`
2. `Peek`
3. `Scan`
4. `Arrive`

`Recenter` may open a destination submenu, with evidence-backed destinations capped to fit the same row limit and a `Back` row when a return is needed. `Peek` opens exactly:

1. `Show verbatim`
2. `Explain in depth`
3. `Back`

Show verbatim and Explain in depth remain Peek actions. The `Lenses…` row is always present at the top level and opens this exact submenu one picker level away:

1. `Analogize`
2. `Visualize`
3. `Quiz`
4. `Back`

`Scan` opens exactly:

1. `Local territory`
2. `Global landscape`
3. `Find an analog`
4. `Back`

Every picker and submenu has at most four rows. `Back` returns to its parent picker without mutating focus, history, notes, links, the goal, filters, or traversal context. `Arrive` executes directly. During cold Teleport, before any positioned focus exists, use the bounded landing chooser described below; after landing, Orient and use this adaptive picker.

### Semantic-direction enforcement

The policy applies to every human-facing directional map label, move, chooser label and description, recommendation, Peek return, and Recenter return: each **MUST** name the stable emoji marker, zone name, and local relationship meaning together. The required semantic triples are exactly:

- `🔵 TOP — what the focus answers to`;
- `🟢 BOTTOM — what builds on the focus`;
- `🔴 LEFT — what contests or questions the focus`;
- `🔷 RIGHT — lateral provenance or task relationships`.

Geometry words such as `upward`, `downward`, `left`, `right`, `above`, and `below` may supplement this semantic triple but never suffice alone. A note title or ID cannot replace the local relationship meaning. Empty zones still require the marker, zone name, and meaning, for example: `🔴 LEFT — what contests or questions the focus: empty; nothing contests or questions this focus.` This is a guide-only presentation policy; it adds no CLI surface.

**Bad:**
- Recenter available
- Show note
- More…

**Why bad:** generic availability and vague labels are not evidence-backed concrete shortcuts; they omit focus mutation, body/evidence reasons, and semantic triples. A neighbor-only picker that hides Peek, Scan, or Arrive beyond `All navigation actions…` is also noncompliant.

**Compliant top-level picker:**
- Recenter 🔵 TOP — what this focus answers to: move to <id> (changes focus) because <body-derived reason tied to the goal>
- Lens — Visualize <named process> (retains focus) because <evidence-derived structural reason>
- Lenses…
- All navigation actions…

Relationship templates embedded in a complete promoted label include `Recenter 🔵 TOP — what this focus answers to: move to <id> because <body-derived reason>` and `Peek 🔴 LEFT — what contests or questions this focus: inspect <id> because <body-derived reason>`; the complete label must also state the applicable focus mutation.

**Navigation presentation check:**
- [ ] focus type and status shown
- [ ] zone/type/edge color markers applied (stable relay palette)
- [ ] zone positions carry their color markers in the map
- [ ] legend/key for the color markers shown so they are interpretable
- [ ] focus degree shown
- [ ] central claim summarized
- [ ] neighborhood role explained
- [ ] positional map rendered
- [ ] edge meanings visible
- [ ] high-degree neighbors receive expanded summaries
- [ ] Recenter available one level away
- [ ] Peek available one level away
- [ ] Scan available one level away
- [ ] Arrive available one level away

Run this check internally before invoking a picker. A missing item blocks presentation unless it is structurally unavailable and the response explains why.

## Arrive report

State:
- starting query or focus;
- final note ID, type, and status;
- region reached;
- what was learned relative to the starting goal;
- which paths remain unexplored;
- that focus remains at the final note.

### Arrival depth

An Arrive report **MUST** explain the landing note, not merely identify it. Provide enough explanation for the human to understand the destination without opening the note separately. Scale depth to the destination's complexity, degree, and role; **2–3 sentences is a minimum for simple notes, not a cap**.

- **Leaf or simple note:** at least one substantive paragraph explaining its central claim and why it answers or reframes the starting goal.
- **Connected note:** usually 2–3 paragraphs covering its claim, neighborhood role, most important relationships, and practical takeaway.
- **Hub, model, protocol, or contested note:** provide a fuller synthesis of its central claim, why it is load-bearing, major supporting and opposing relationships, visible tensions or uncertainty, implications, and what remains unresolved.

A field-only checklist or one-line summary is not a compliant arrival. The report should optimize for understanding rather than sentence count, while avoiding a raw dump of the note body or graph output.

## Moves versus lenses

Moves operate on the positioned walk; lenses interpret evidence already visited. Focus + Map + Moves remains mandatory whenever the walk presents an onward decision. Find an analog and lenses may be reached through the hierarchy or promoted as concrete top-level shortcuts when they satisfy the evidence criteria. They do not become focus-changing moves: Lens actions never mutate focus or navigation history, and Scan retains focus.

**Find an analog** is a **Scan retrieval move across another region**. It seeks the same **relational structure, not lexical similarity**. Candidate generation may use `nn clusters`, `nn list --search`, `nn list --similar`, and graph neighborhoods, but those retrieval signals identify possibilities rather than establish an analogy. For every serious candidate, compare relational shape and present:

- a correspondence mapping between the focus-side roles, nodes, and edges and their candidate-side counterparts;
- where the analogy holds and what that correspondence explains;
- where it breaks, including unmatched roles or differently typed/directed edges; and
- a classification of any proposed connection as a **missing-edge suggestion** when visited evidence supports a plausible absent stored relationship, or **comparison-only** when the resemblance is explanatory but does not justify a link.

Preserve the retained focus until an explicit Recenter. Finding or comparing an analog does not move focus, push Back, clear Forward, or create the suggested edge.

Peek actions and lenses **never mutate focus or navigation history**. They inspect or interpret retained evidence whether selected through the hierarchy or promoted directly.

### Show verbatim

Show the complete stored body verbatim, without truncation. Clearly separate metadata, stored links, and display-only injected material from that body so frontmatter, graph annotations, relay hints, inferred backlinks, or agent commentary are never presented as stored prose. Preserve whitespace and wording; do not summarize inside the verbatim body.

### Explain in depth

Give a source-bounded full treatment of the visited material's claims, details, relationships, implications, and uncertainty. Scale to the evidence rather than a fixed paragraph count, and separate direct quotation, stored-edge evidence, and interpretation. Do not silently turn display-only context, generated layouts, or analogies into notebook claims.

### Analogize

Generate a **familiar analogy** that helps interpret the visited material. Give an explicit correspondence, state what it clarifies, and state where it breaks. Label the analogy **generated, non-evidence**: it is an explanatory aid, not a notebook fact, stored edge, or substitute for cited evidence.

### Visualize

Visualize **spatializes meaning** in the **retained visited evidence**. Distinguish the **stored Map/graph**—visited notes and stored edges—from any **derived layout** used to explain them. Label derived arrows and groupings as **non-stored**; never imply that proximity, containment, or an explanatory arrow is a notebook edge.

Use ASCII on any surface, or **Pi-supported Mermaid** when it will render. Allowed Mermaid families are `graph`, `flowchart`, `stateDiagram`, `stateDiagram-v2`, `classDiagram`, `erDiagram`, and `sequenceDiagram`; exclude `pie`, `quadrantChart`, and `mindmap`.

### Quiz

Quiz is **source-grounded** and tests a bounded set of **consequential concepts**, not trivia. State an **explicit purpose and stopping condition**, normally bounded to 1–3 concepts. Ask **one question, then wait for a human Predict turn before the reveal**; accept safe `pass`, `skip`, or `I don't know` responses without penalty or pressure. An unanswered Quiz question suspends the picker until the human answers, passes, skips, says `I don't know`, or says `navigate`. Every active Quiz prompt with an unanswered question MUST offer the escape in this wording: `Answer, pass/skip/I don't know, or say navigate to leave the Quiz lens and return to navigation.`

After the Predict turn, compare the prediction with the source-grounded answer, cite the relevant note IDs and stored edge evidence, and explain any misconception and why it matters. Do not invent questions beyond what the retained sources can answer, invent facts or relationships, or test derived-framework recall as though an agent-generated analogy or layout were notebook evidence.

Show verbatim, Explain in depth, Analogize, and Visualize are transient actions: after each result, automatically restore the complete retained navigation frame, re-run Orient, render Focus + Map + Moves, and reopen the adaptive quick-actions picker when the harness supports it. A completed Quiz is transient too: after a reveal, pass, skip, or `I don't know`, perform the same automatic return. While its question is unanswered, do not render another picker. `navigate` remains an interruption and early escape: abort the item without grading or revealing, then perform the same return immediately.

Lens findings may inform a later explicit Recenter or link suggestion, but the lens itself does not mutate focus, history, notes, or links. Any later movement or mutation must be separately proposed and executed under its normal contract.

## `navigate` — resume navigation (universal escape)

`navigate` is a **conversational shortcut, not an `nn` subcommand**. It universally leaves the current lens or navigation discussion and reopens the positioned walk from the retained frame. When the human says `navigate` after Arrive, during or after a lens, or after discussing visited material, resume from the retained final focus: restore the complete retained navigation frame—including its focus, goal, filters, and traversal context—re-run Orient for that focus, render Focus + Map + Moves, and invoke the adaptive quick-actions picker when the harness supports it. Otherwise keep all four action classes discoverable without inventing an interactive picker. Do not search for a new entry point or change focus merely because navigation was reopened.

`navigate` during an unanswered Quiz aborts the current item without grading, revealing the answer, or forcing Quiz completion. It does not mutate focus or navigation history. Immediately restore the complete retained navigation frame, re-run Orient, render Focus + Map + Moves, and invoke the adaptive quick-actions picker when the harness supports it. This is the same conversational escape, never an `nn navigate` command.

Use a navigation-return footer at a seam where the positioned frame remains available but navigation has not already reopened: after every Arrive report and when exiting an extended navigation discussion. End with this exact footer, or a close equivalent that preserves the `navigate` trigger, all three restored view parts, and the retained focus:

> Say navigate to reopen Focus + Map + Moves at the retained focus.

Do not use this footer as a substitute for the automatic transient-action return. After Show verbatim, Explain in depth, Analogize, Visualize, or a completed Quiz, the picker is already reopened.

If no focus is retained, say that resume is structurally unavailable and treat `navigate <query>` as a cold Teleport request. A bare `navigate` with no retained focus should ask for a query or offer Scan and Arrive rather than inventing a destination.

## Conversational navigation history and bookmarks

These are skill-level conversational moves and conversation-scoped state, not `nn` subcommands or persisted notebook data. A **navigation frame** is the retained focus plus its active traversal context and filters: preserve enough information to reproduce the positioned view, including the walk goal/query and any active direction, link-type, status, representation, depth, route, or impact context. Restoring only a note ID is not sufficient when filters or traversal context were active.

Maintain a current frame, a Back stack, a Forward stack, and bookmarks:

- After a successful Teleport, Visit, Recenter, or Go to, push the prior frame onto Back and clear Forward. `Visit` here means an independently requested move when that vocabulary is already present; it is never the prohibited second confirmation after a completed Teleport landing. If there is no prior/current focus, an initial landing cannot push a frame.
- **Back** moves the current frame onto Forward and restores the latest Back frame. **Forward** moves the current frame onto Back and restores the latest Forward frame. After either restoration, rerun Orient and present Focus + Map + Moves under the [Presentation discipline](#presentation-discipline).
- **Bookmark <name>** stores the complete current frame under a case-sensitive name. Creating a new name needs no confirmation; an existing exact-case name requires explicit confirmation before replacing it. A declined replacement changes nothing.
- **Go to <name>** restores its saved frame as a Teleport landing. It therefore follows the successful-move rule: push the prior current frame onto Back, clear Forward, rerun Orient, and present Focus + Map + Moves.
- **Where am I?** reports the current focus, active traversal context and filters, immediate Back and Forward destinations (if any), and bookmark names. It does not mutate navigation state.

Failed or no-op operations never mutate history or bookmarks. If the Back or Forward stack is empty, say, for example, “Back stack is empty” or “Forward stack is empty,” retain the current frame, and do not rerun a fictitious landing. For an unknown bookmark, say the name was not found, list the available case-sensitive bookmark names, and retain all state. A move that resolves to the complete current frame is a no-op.

This state lasts only for the conversation. Any compaction handoff MUST include the full current frame, Back and Forward stacks, and every bookmark, preserving stack order and each bookmark's case-sensitive name and complete saved frame. If that handoff is missing or incomplete after compaction, state is unknown: never invent history, filters, destinations, or bookmarks; report that navigation state cannot be recovered and ask the human to establish a new landing or restate it.

## Virtual navigation protocol seed

A compact virtual protocol should use `applies_when: human-driven nn graph navigation` and require:

1. if `nn skills list` has not yet run this session, run it, then load `nn-navigate` with `nn skills get nn-navigate`;
2. run the required graph command;
3. render Focus + Map + Moves;
4. open the adaptive quick-actions picker on capable human-driven surfaces, keeping Recenter, Peek, Scan, and Arrive exactly one level away;
5. preserve focus for Peek and Scan;
6. adopt a new destination only after a successful Teleport, Visit, Recenter, or Go to; Back and Forward may change focus only by restoring a retained frame;
7. when the relationship family is known but the destination is unknown, run `nn graph routes --focus ID --links TYPES --search QUERY --limit N --json` and treat its bounded witnesses as route candidates, not relationship proof;
8. preserve complete-frame Back/Forward and case-sensitive bookmark semantics exactly as specified above, mutating them only after successful focus-changing moves; and
9. carry the complete navigation state through compaction, or report it unknown rather than reconstructing it.

This seed is the compact enforcement contract for virtual protocols. The detailed contract remains owned by this skill rather than the broad command reference or a virtual protocol.

0. **Enter** — if you have no starting note, find one first: `nn list --search "<topic>" --json --fields id,title,score,link_count` and pick the highest-scoring, best-connected hit as your entry focus. If you already have a note ID (from a prior answer, a link, a backlink), start there.
1. **Orient** — render the current node's zoned neighborhood with bodies (use `--color always` for every color-capable human relay):
   ```
   nn graph show --focus <id> --depth 1 --direction both --zones --bodies --presentation-hints --color always --format text
   ```
   You now see, for the current node: 🔵 TOP — what the focus answers to; 🟢 BOTTOM — what builds on the focus; 🔴 LEFT — what contests or questions the focus; and 🔷 RIGHT — lateral provenance or task relationships. Each neighbor includes its body, its `↑out ↓in` degree, and a zone key. Empty zones are information too: `🔴 LEFT — what contests or questions the focus: empty` means nothing contests or questions this focus; `🔵 TOP — what the focus answers to: empty` means the focus is a root. A neighbor with high inbound degree is a hub worth weighting even if it is off your direct path.
2. **Read from here** — the bodies let you evaluate a move without leaving the view. Zones tell you *what kind* of move each neighbor is; bodies tell you *whether it's the one you want*. Choose the move that closes distance to your goal:

   | Your goal | Zone to step into |
   |-----------|-------------------|
   | What challenges or questions this? | **🔴 LEFT — what contests or questions the focus** |
   | What does this answer to, or where did it come from? | **🔵 TOP — what the focus answers to** or **🔷 RIGHT — lateral provenance or task relationships** |
   | What builds on or operationalizes this? | **🟢 BOTTOM — what builds on the focus** |
   | What is laterally related by provenance or task context? | **🔷 RIGHT — lateral provenance or task relationships** |

3. **Recenter** — pick a neighbor's ID and re-run step 1 with that as `--focus`. Navigation is a chain of ego-hops (the ExcaliBrain model in design note 20260820154156-6576), not a pan over a hairball. Whenever a step (this one, or a return from `peek`/`teleport`) presents the view to a human, apply the [Presentation discipline](#presentation-discipline) (P1–P4). Peek returns and Recenter returns MUST restate the semantic triple for every directional option; returning to a title, ID, arrow, or geometry word alone is noncompliant.
4. **Arrive** — stop when the current node answers your goal, or when the zone you were following is empty (for example, `🔴 LEFT — what contests or questions the focus: empty` ends a walk seeking contesting material). Emit the complete [Arrive report](#arrive-report), including the region reached and what changed relative to the starting goal.

These three verbs are **not** `nn` subcommands or flags — they are named moves *you* (the agent) perform by composing existing primitives, exactly like the walk steps above. There is **no new CLI surface**. Modes and options named below (e.g. "flat mode") are presentation choices you make, not shell flags.

#### `scan` — look wide (an always-available aside, not a step)

At any point in the walk (Enter, Orient, Read, Recenter) you can stop stepping sideways and *zoom out* to see the landscape around the current node, then return to the walk where you left it. `scan` is deliberately **not zoned**: zones are defined only relative to one ego (design note 20260821155048-7280), so they answer "how do my neighbors relate to me" (relationship) — `scan` answers "what territory am I standing in" (structure at altitude), which zones cannot express. Use it when you feel lost, when Orient shows an unexpectedly dense or sparse neighborhood, or before committing to a direction.

`scan` spans **both altitudes in one look** — your ego territory *and* the global landscape. Do not raise `--depth` to "look farther out": past depth 2 more hops give an exponential tangle, not a broader view — the real broadening is the **ego→global shift**, not more depth. Render both anchors (never start at an intermediate level), combining these existing primitives:

**Your territory (ego, depth 2)** — where you stand and how far you can walk:
- **Degree + reach** — the current node's `↑out ↓in` (hub or leaf?) plus its 2-hop extent, unzoned:
  ```
  nn graph show --focus <id> --depth 2 --direction both --color always --format text
  ```
- **Region + load-bearing** — which cluster this node sits in (`nn clusters`) and whether it sits on a bridge between clusters (`nn graph bridges`). A node on a bridge is a crossing point; a node deep in one cluster is interior.
- **Reachable-but-unlinked** — `nn list --similar <id>`: nearby notes that share vocabulary but have no edge to walk. This is the one signal a step-wise walk can *never* surface — territory the navigation can't reach yet (candidate missing edges).

**The wider landscape (global)** — where your region sits in everything (drops the ego entirely):
- **The landmass — `nn clusters`** — every topic cluster and its size. When the walk has a goal query, prefer `nn clusters --search "<query>" --json --summary` to project that query onto full-graph regions without loading every unrelated cluster or every member of a matching cluster. Summary output defaults to the top 3 ranked matches per region while retaining total `match_count`; read `matches_returned` and `matches_truncated`, and use `--match-limit 0` only when Scan genuinely needs every matching note. The match limit is per region and never limits the number of regions.
- **The highways — `nn graph bridges`** — integration points whose links join otherwise-separate regions, ranked by load-bearing weight. When the walk has a goal query, use `nn graph bridges --search "<query>" --format json --exclude <focus-id>` to project relevance onto bridges computed from the complete graph without offering the retained focus as a movement candidate. `--exclude` is repeatable and is applied before `--limit`, so excluded results are replaced rather than shortening the candidate list. Read each returned record's bounded crossing witnesses and region context to explain why it is a plausible crossing before acting; the connector evidence is not proof of territorial separation. Peek through a returned bridge `id` to inspect where it leads without moving, or Recenter on that `id` to cross into its neighborhood. (Daily/index notes often rank high because they touch many topics — treat those as connectors-by-aggregation, not substantive bridges.)
- **The whole shape — `nn graph show --color always`** *(no `--focus`)* — the entire graph, a last resort (it's large); usually the two above are enough.

**Two presentation modes — you choose at look time:**
- **Default (labeled sections)** — render the two anchors above as *Your territory (ego)* and *The wider landscape (global)*, kept distinct. This is the default: the labels honor the relationship-vs-territory and ego-vs-global distinctions.
- **Flat mode (blended)** — merge both anchors into one landscape with no tier labels, when the user asks for the merged view. Same lookups, different rendering.

**Make it spatial — you draw the map, the command is just the data.** The `--format text` trees are the *data source*, not the thing you relay. Don't paste raw output. Read it — indentation is reach (hops out), `↑out ↓in` markers are terrain (hubs vs. leaves) — then *draw* a map in whatever form fits the surface (the same "CLI is a faithful data source, the agent is the presenter" split the zoned step uses). Cheapest first:

- **Indented / positional text** — an annotated tree, or a rough sketch placing the hub at the center, with `🔵 TOP — what the focus answers to` geometrically above and `🟢 BOTTOM — what builds on the focus` fanning below. Geometry supplements rather than replaces those semantic labels. Works on any surface, including plain terminals and agent relay. This is the default.
- **A Mermaid diagram** — only when relaying to a human on a surface that *renders* Mermaid. Build it yourself; keep it compact (drop edge annotations, abbreviate titles). An unrendered Mermaid block is just a wall of source.

For the **global** anchor the map is a **different shape** — not a hop-tree from one ego, but **regions as blobs sized by note-count, with bridges as the lines joining them** (a continent map, not a family tree). Mark where you are. On a plain surface, an ASCII sketch:

```
  ┌──────────────────────────┐
  │  cluster 1  (259) ●●●●●●  │  central continent (prompting/token core)
  └───────────┬──────────────┘
        bridge: 5853 "vocab is a composition barrier"
              │
     ┌────────┴─────┐   ┌──────────┐   ┌──────────┐
     │ c3 (25) 🟠 YOU │   │ c2 (28)  │   │ c4 (24)  │   … 35 more small regions
     │ graph-tooling│   └──────────┘   └──────────┘
     └──────────────┘
```

Before relaying either altitude, apply the [Presentation discipline](#presentation-discipline) (P1–P4) to the composed Scan; this is where text-source markers and manually marked JSON region/bridge data become one consistent human-facing view. Pick the rendering by surface capability, exactly as you pick whether to offer the recenter chooser — richer form only when the surface supports it, text otherwise. After scanning, resume the walk: recenter on a neighbor as before, or hop to a similar-but-unlinked note by making it the new `--focus` (and consider linking it, since the absence of an edge is what `scan` just exposed).

#### Typed destination discovery

When the walk has a current focus and a goal query but no destination yet, run:

```
nn graph routes --focus ID --links TYPES --search QUERY --limit N --json
nn graph routes --focus ID --links TYPES --search QUERY --limit N --json --explain
```

This intersects semantic relevance with actual directed reachability under the selected relationship types. A result is a candidate landing plus `witnesses`, at most 3 deterministic shortest paths selected by first-hop diversity, then edge type-sequence diversity, then lexical full-path order; none proves that the destination is the only or best conceptual landing. Without `--explain`, JSON remains a top-level route array. Opt-in `--explain` (which requires `--json`) wraps that same array as `{routes:[...], diagnostics:{...}}` with bounded aggregate diagnostics only—never note bodies, titles, or candidate dumps.

Diagnostics report normalized `query_tokens`; `total_notes`; `direct_lexical_matches` over title, body, and tags without annotations; `focus_excluded` (0 or 1); `typed_reachable` excluding focus; `eligible_destinations` (direct lexical matches intersected with typed reachability, excluding focus); `returned`; and boolean `truncated_by_limit`. `graph_scored_matches` is a separate count from the full graph-aware relevance scorer and may exceed direct lexical matches because inbound/outbound annotations can score. Annotation-only scores never make a route eligible. Routes has no depth flag, so the diagnostics do not report or imply a traversal depth.

- **Orient** — run typed destination discovery when the goal implies a relationship family but the destination is unknown; present the highest-ranked reachable destinations beside the local map.
- **Scan** — use the ranked destination set to see query-relevant territory reachable under `TYPES`; absence means no directed route to a destination with positive direct lexical evidence under that filter, not global disconnection.
- **Peek** — inspect a selected result's complete `witnesses`; each member has aligned `nodes` and `edges`. Compare alternatives without changing focus.
- **Recenter** — choose one witness, move to `witnesses[k].nodes[1]`, and rerun Orient; do not jump directly to the destination while claiming to walk that witness.
- **Arrive** — when focus reaches `destination.id`, explain the traversed edge types and annotations and report the destination's `relevance_score` as discovery evidence, not relationship strength.

#### Explicit typed impact overlay

When the walk asks what a retained focus affects or what relies on it through a known relationship family, run:

```
nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json
```

This is a bounded structural impact set, not relevance ranking or inferred closure. Choose direction from stored semantics: `grounded-by` incoming from an evidence focus finds claims that depend on it, while `supports` outgoing from an evidence focus finds claims it corroborates.

- **Scan** — read `summary` first for `total_impacts`, depth distribution, overlapping first-hop branch counts, and the honest `witnesses_truncated` signal, then read the complete returned impact set to the requested depth; absence under one type/direction filter is not proof of global isolation.
- **Peek** — inspect a selected impact's full `witnesses` array without changing focus. For every incoming witness, `nodes` run focus→impact while each stored edge retains source→target orientation and therefore points opposite its consecutive traversal nodes.
- **Recenter** — choose one witness and move only to `witnesses[k].nodes[1]`, retain the selected types/direction/depth deliberately, and rerun from the new focus; do not jump to a distant impact while claiming to walk the witness.
- **Arrive** — when focus reaches the selected `node.id`, explain the traversed edge types and annotations. Read each incoming edge in its stored source→target orientation rather than reversing its claim.

#### Typed path route overlay

When the walk has both a current focus `<a>` and a known destination `<b>`, use `nn path <a> <b> --links <types> --json` as an optional semantic route overlay. It returns `witnesses`, at most 3 shortest directed paths whose edges use only the requested relationship types. Selection prioritizes distinct first-hop nodes, then distinct edge type-sequence values, then lexical full-path order. These are concrete routes, not Datalog closure: Datalog answers whether and what follows transitively; typed path shows ordered hops that Navigation can execute and explain.

Integrate the overlay into the navigation actions as follows:

- **Orient** — compute the typed path when the goal implies a relationship family and show the route beside the local map. Relationship direction matters: use `grounded-by` to walk claim→evidence, `supports` to walk evidence→claim, and `requires` for source task→required dependency. Do not combine opposite semantic directions merely because both concern evidence.
- **Teleport** — keep instant relocation as the default. Offer the typed path only when the human wants an explainable semantic route instead of jumping directly to a distant landing.
- **Scan** — assess whether candidate targets or regions have a semantically coherent route from the current focus; do not confuse absence under one filter with global disconnection.
- **Peek** — preview every complete node-and-edge witness without changing focus.
- **Recenter** — choose one witness and move to its `nodes[1]`, the next hop, then rerun Orient. Never jump to the final node while claiming to walk the route.
- **Arrive** — explain the traversed `edges` sequence, including relationship types and annotations, as the semantic account of how origin and destination connect.

#### `peek` — look deep without moving (read-only, focus stays put)

`peek` previews detail **without changing focus** — this is what makes it distinct from a one-step recenter. If it moved focus it would just be a slow walk. Two forms:

- **`peek` on the current node** — expand detail in place: read the current node's full body and its neighbors' bodies without recentering (`nn graph show --focus <id> --depth 1 --direction both --zones --bodies --color always` is already the read-in-place view; peeking just means you consume it without committing to a move).
- **`peek` in a direction** — preview what is down one zone (TOP / BOTTOM / LEFT / RIGHT) before deciding to go there: `nn show <neighbor-id> --depth 2` reads that neighbor and *its* onward links, so you see where the move leads without taking it. Focus stays on your current node throughout.

Use `peek` when Read-from-here isn't enough to judge a move — you want to see one step past the neighbor before committing.

Because `peek` does not move, it returns to the retained positioned frame rather than creating a new one. After a transient Peek result, apply the full [Presentation discipline](#presentation-discipline) (P1–P4): re-run Orient, render Focus + Map + Moves, and reopen the adaptive picker rather than dropping into prose. The finding may update or justify a promoted shortcut (for example, `Recenter 🔷 RIGHT — lateral provenance or task relationships: move to 4302 (changes focus) because the peek confirms its body resolves the open provenance question`); it does not exempt the return from evidence criteria or semantic-direction enforcement.

#### `teleport` — move far (relocate focus)

`teleport` is how you *get* a focus when you have none, or jump to a distant region. Unlike `scan`/`peek` (which look), `teleport` **relocates** — it changes where you are. Two modes:

- **`teleport` with a query** — land on **structure, not a ranked hit list**. Use `nn clusters --search "<query>" --json --summary` as the **default landing-zone source**: it clusters the complete graph, returns only regions containing search hits, ranks them by top-three normalized matching evidence, supplies a representative without dumping full region membership, and defaults to at most 3 ranked `matches` per region. Total `match_count`, score, ranking, density, and representative remain exhaustive; inspect `matches_returned`/`matches_truncated`, or use `--match-limit 0` when every evidence match is explicitly needed. Recenter on the chosen region's `representative.id`, then re-enter Orient from that note; alternatively recenter on a ranked `matches[].id` to enter through specific evidence. Use `nn list --search "<query>"` separately only when you need note-level evidence beyond the region result, or rerun without `--summary` when complete region membership is explicitly needed.
- **`teleport` random** — a serendipitous jump: `nn random` (optionally `--tag`/`--type`) or `nn shuf` to land somewhere unplanned, then start a fresh walk from there.

`teleport` only relocates — it does not render deep detail (that is `peek`'s job). After landing, **re-enter the walk at Orient (step 1) from the new focus, carrying the full [Presentation discipline](#presentation-discipline) (P1–P4)** — the landing is not a shortcut past it; landing then narrating in prose without colors, map, or chooser is exactly the seam-drift that discipline exists to prevent. This is the one move worth invoking from cold — it is named in the session-start CLI-reference pointer for that reason.

**The landing itself is a chooser point** — teleport's landing *is* a Recenter decision ("which region/hub do I land on?"). When a human is driving on a chooser-capable harness, render genuinely ambiguous candidate landings in a chooser of at most four rows. The source is JSON, so the pre-landing Teleport chooser must manually mark each candidate landing region with the stable relay palette before any focus exists. Its bounded option set is the recommended hub, additional cluster/sub-territory candidates as capacity permits, and a `teleport --random` escape as the "somewhere else" option. Because no positioned frame exists yet, Focus + Map + Moves and the adaptive picker begin immediately after the landing, not before it.

**Selection completes the landing decision.** Once the human selects a candidate—or the goal makes one candidate unambiguous—the selected landing automatically becomes the retained focus. Immediately run Orient from it and present Focus + Map + Moves. Teleport **MUST NOT ask for a second confirmation** such as "visit this focus?" after selection and **MUST NOT offer a separate `Visit` action**. Ask for human choice only while candidates remain genuinely ambiguous and no landing has yet been selected.

**Offer the adaptive picker when the harness supports one and a human is driving the walk.** A harness picker renders contextual decisions as clickable actions rather than a wall of prose. Use it *only* when both hold: the harness exposes a chooser affordance, **and** a human is co-navigating this walk. When either is false — you are navigating autonomously toward a goal, or stdout is not a TTY — do **not** invoke a picker: pick the move yourself per the goal and continue (nn is non-interactive by default when stdout is not a TTY).

When you offer it, use the [Adaptive hierarchical quick-actions picker](#adaptive-hierarchical-quick-actions-picker), not a neighbor-only or strict-flat chooser. A promoted Recenter targets the goal-relevant semantic triple—`🔴 LEFT — what contests or questions the focus`, `🔵 TOP — what the focus answers to`, `🔷 RIGHT — lateral provenance or task relationships`, or `🟢 BOTTOM — what builds on the focus`—and its description includes that complete triple, `↑out ↓in` degree, focus mutation, and a one-line body-derived reason. Other promoted actions likewise name their class, focus behavior, and evidence reason. Then execute the selected concrete action directly. Teleport, Visit, Recenter, and Go to may adopt a new destination; Back and Forward may only restore retained frames. No other action changes focus.

<a id="presentation-discipline"></a>
#### Presentation discipline (the named block every seam cites)

**This is the single source of truth for how any walk view is presented to a human. Every point that presents a positioned view — Orient, the return after `peek`, the landing after `teleport`, `scan` — cites this block by name rather than restating it. A rule stated only at one step definition does not travel across a seam that re-enters the walk from elsewhere; centralizing it here is what stops per-seam drift.** When presenting to a human on a capable surface, all four apply jointly:

- **P1 — Colors and relay budgets on.** Every color-capable human-facing navigation view uses the stable markers below—not only post-landing Orient, but also the JSON-backed pre-landing Teleport chooser, the return after Peek, Scan at both altitudes, and Arrive. Run the underlying `nn graph show … --zones --bodies --presentation-hints --color always` so zone/type/edge markers and degree-based summary budgets survive relay. Graph text sources MUST use `--color always` for a color-capable human relay; do not trust `auto`, because tool stdout is commonly non-TTY. JSON sources are marker-free by design: parse them, then manually apply the relay palette to the human-facing chooser, headings, map, focus, and region labels—never mutate or claim markers exist in the JSON. The stable emoji marker and textual zone/meaning triple remains mandatory even when the surrounding surface cannot render terminal color; keep `--presentation-hints` so each complete body travels with an in-context relay budget.
- **P2 — Focus + Map + Moves.** Give the Focus summary (substance), positional/ASCII Map (structure), and degree-scaled Moves (direction) defined above. Every directional use in all three parts follows [Semantic-direction enforcement](#semantic-direction-enforcement). None is sufficient alone: a map without summaries is a skeleton you can't read; summaries without a map lose the spatial relationships. Never relay raw command output. (Detailed as (a)/(b)/(c) below.)
- **P3 — Adaptive hierarchical quick-actions picker.** When the harness exposes a chooser affordance **and** a human is co-navigating, show up to two justified concrete shortcuts, stable `Lenses…`, and final `All navigation actions…`; keep the Lens submenu and Recenter / Peek / Scan / Arrive exactly one level away through those stable rows. Otherwise pick the move yourself per the goal while keeping all four action classes discoverable in the presentation. This applies wherever a positioned walk presents an onward decision, including after `peek` and a completed `teleport` landing.
- **P4 — Degree-scaled summaries.** Scale each summary's length to the node's inbound degree (the tiers in (c)); daily/index hubs are connectors-by-aggregation, not substance.

##### Stable emoji relay palette

Use this exact palette in every color-capable human-facing navigation view. It extends the graph command's actual zone convention and remains stable across source formats:

- `🔵 TOP` — what the focus answers to;
- `🟢 BOTTOM` — what builds on the focus;
- `🔴 LEFT` — what contests or questions the focus;
- `🔷 RIGHT` — lateral provenance or task relationships;
- `🟠 FOCUS / REGION` — the retained focus and its current region. Before a Teleport landing exists, use the same orange marker for each candidate landing region and label the recommended candidate explicitly.

Preserve note-type markers and edge-family markers already emitted by graph text instead of remapping them. Agent-drawn positional or region maps must use this palette too. You must **reproduce this legend** in the presentation itself — a short key mapping each marker to its meaning (the five zone markers above, plus the note-type and edge-family markers you carried through) — because a colored marker the reader cannot decode carries no information; the markers and their legend travel together. Plain, uncolored Focus + Map + Moves is noncompliant on a color-capable surface even when its prose and geometry are otherwise correct.

The three parts, in detail:

**(a) Summarize the current note — length scaled to its degree (see the tiers in (c)).** Before the map, characterize where the reader is standing: what this note *is* (its type/status and its central claim, drawn from its body — what it argues, not just its title), and how it sits in `🔵 TOP — what the focus answers to`, `🟢 BOTTOM — what builds on the focus`, `🔴 LEFT — what contests or questions the focus`, and `🔷 RIGHT — lateral provenance or task relationships`. Apply the same degree tiers as the neighbors: a high-degree focus (a hub you've landed on) earns the fuller 2–3 sentence treatment; a low-degree focus (a leaf you're passing through) gets a brief one. This is the anchor the rest of the view hangs off.

**(b) Draw the map.** Lay the zones out by position so the layout itself encodes the relationships:

```
            🔵 TOP — what the focus answers to
                         ▲
 🔴 LEFT — what contests or questions the focus ◀── 🟠 YOU ARE HERE <focus id + title> ──▶ 🔷 RIGHT — lateral provenance or task relationships
                         ▼
            🟢 BOTTOM — what builds on the focus

  🟠 focus   ──▶/◀── edge direction
  🔴 LEFT — what contests or questions the focus: empty
```

**(c) Summarize each neighbor + name the moves — scale each summary's length to that neighbor's inbound degree (`↓`).** Draw each summary from the body (what it *claims*, not just what it's *called*), then flag the recommended next move relative to the goal. Do not give every neighbor a uniform one-liner — that is the failure this step exists to prevent. Use these tiers:

| Inbound degree | Length |
|----------------|--------|
| leaf (↓0–1) | title + type + one clause |
| connected (↓2–4) | one full sentence on its claim |
| hub (↓5+, or the highest-degree node in view) | 2–3 sentences: its claim *and* why it's load-bearing |

The CLI never truncates — it always gives the full body; degree only tells *you* how much to relay. With `--presentation-hints`, each text node carries a `relay budget:` line and each JSON node carries structured `summary_budget` metadata mirroring these tiers, so this policy remains visible without looking back at the guide. Two overrides: a high-degree *daily* or index note is a hub by connectivity, not substance — treat it as connected (one sentence) and say so; and a low-degree axiom whose body is clearly central gets promoted a tier — let the body's actual claim override the degree. Empty zones carry meaning and the full semantic triple: say, for example, `🔴 LEFT — what contests or questions the focus: empty; nothing contests or questions this focus.`

Without `--bodies` this is the older search→zone→per-node-`nn show` loop; `--bodies` collapses that into a single call. Keep `--depth 1` when zoning — only direct neighbors carry a zone, and the zoned text view omits unzoned nodes. Bodies can be long; drop `--bodies` (titles only) when you just need the shape, add it back when you need to read.

