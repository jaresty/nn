---
name: movement
applies_when: "When entering, orienting, recentering, peeking, teleporting, arriving, or resuming a positioned navigation walk."
---

# Reference: Movement

### Navigation mode: the zoned navigator with contents

Build the navigation view from two bounded sources: `nn graph show --focus <id> --zones --color always` supplies topology, degree, and zone-colored headers; `nn graph bodies --focus <id> ...` supplies the identical traversal set's stored bodies as lossless snapshot-bound pages. Retrieve and reconstruct every body page before deriving central claims or move reasons. This lets you see where you can move and read where you would land without risking an unbounded inline-body response. Use `--color always` when presenting topology to a human on a color-capable surface.

Use the combined, fully received sources to walk the graph as a positioned space rather than dumping the whole structure. Hold a **goal** in mind for the walk ("find what contests this design", "trace where this claim came from", "reach the principle underneath") — every step is judged relative to that goal, not to the whole graph.

## Arrive report

State:
- starting query or focus;
- final note ID, type, and status;
- region reached;
- what was learned relative to the starting goal;
- which paths remain unexplored;
- that focus remains at the final note.

Arrive does not clear interaction mode. After the completed report, Guided mode renders persistent Navigation help with canonical shorthand labels; Advanced mode keeps it closed and uses only the navigation-return footer below.

### Arrival depth

An Arrive report **MUST** explain the landing note, not merely identify it. Provide enough explanation for the human to understand the destination without opening the note separately. Scale depth to the destination's complexity, degree, and role; **2–3 sentences is a minimum for simple notes, not a cap**.

- **Leaf or simple note:** at least one substantive paragraph explaining its central claim and why it answers or reframes the starting goal.
- **Connected note:** usually 2–3 paragraphs covering its claim, neighborhood role, most important relationships, and practical takeaway.
- **Hub, model, protocol, or contested note:** provide a fuller synthesis of its central claim, why it is load-bearing, major supporting and opposing relationships, visible tensions or uncertainty, implications, and what remains unresolved.

A field-only checklist or one-line summary is not a compliant arrival. The report should optimize for understanding rather than sentence count, while avoiding a raw dump of the note body or graph output.

## `navigate` — resume navigation (universal escape)

`navigate` is a **conversational shortcut, not an `nn` subcommand**. It universally leaves the current lens or navigation discussion and reopens the positioned walk from the retained frame. When the human says `navigate` after Arrive, during or after a lens, or after discussing visited material, resume from the retained final focus: restore the complete retained navigation frame—including its focus, goal, filters, and traversal context—re-run Orient for that focus, and reset the menu stack to Quick actions without changing interaction mode. In Guided mode, invoke the adaptive quick-actions picker when the harness supports it, or render equivalent labeled Navigation help; in Advanced mode, keep help closed. After discussion, render Focus + Map + Moves in full; otherwise follow the transient rendering policy when Orient confirms the complete cache and notebook are unchanged. Keep all four positioned navigation action classes plus Ask discoverable without inventing an interactive picker. Do not search for a new entry point or change focus merely because navigation was reopened.

A direct conversational launch of `navigate advanced` switches interaction mode to Advanced before the landing or resume; any remaining text is the ordinary query or navigation intent. This is still agent-owned conversational syntax, not an `nn navigate advanced` command. A later bare `navigate` preserves that Advanced mode, while `:guided` and `:advanced` remain the explicit mode switches.

`navigate` during an unanswered Quiz aborts the current item without grading, revealing the answer, or forcing Quiz completion. It does not mutate focus or navigation history. Immediately restore the complete retained navigation frame, re-run Orient, and reset to Quick actions. Guided reopens that picker using the rendering policy; Advanced keeps it closed. This is the same conversational escape, never an `nn navigate` command.

Use a navigation-return footer at a seam where the positioned frame remains available but navigation has not already reopened: after every Arrive report and when exiting an extended navigation discussion. End with this exact footer, or a close equivalent that preserves the `navigate` trigger, all three restored view parts, and the retained focus:

> Say navigate to reopen Focus + Map + Moves at the retained focus.

Do not use this footer as a substitute for the deterministic transient-action return. In Guided mode, after Show verbatim, Explain in depth, Analogize, Find an analog, Find gaps, Visualize, or a completed Quiz, reopen the invoking picker with the compact or full presentation required by the rendering policy. In Advanced mode, retain the same semantic return position but keep Navigation help closed.

If no focus is retained, say that resume is structurally unavailable and treat `navigate <query>` as a cold Teleport request. A bare `navigate` with no retained focus should ask for a query or offer Scan and Arrive rather than inventing a destination.

## Virtual navigation protocol seed

A compact virtual protocol should use `applies_when: human-driven nn graph navigation` and require:

1. if `nn skills list` has not yet run this session, run it, then load `nn-navigate` with `nn skills get nn-navigate`;
2. run the required graph command;
3. render Focus + Map + Moves;
4. in Guided mode, open the adaptive quick-actions picker on capable human-driven surfaces, keeping Recenter, Peek, and Scan exactly one level away and Arrive at top level, with Ask also exactly one level away under All actions; in Advanced mode keep help closed except for temporary `:help` or narrowed resolution help;
5. preserve focus for Peek and Scan;
6. adopt a new destination only after a successful Teleport, Visit, Recenter, or Go to; Back and Forward may change focus only by restoring a retained frame;
7. when the relationship family is known but the destination is unknown, fetch `scan-and-routes` and use its typed-destination workflow, treating bounded witnesses as route candidates rather than relationship proof;
8. preserve complete-frame Back/Forward and case-sensitive bookmark semantics exactly as specified above, mutating them only after successful focus-changing moves;
9. preserve the conversation-scoped interaction mode, current menu, and ordered menu stack as separate compaction UI state, never notebook state;
10. fetch `ask` before consultation and execute only its complete retained-frame lifecycle, with no implicit movement or mutation; and
11. carry the complete navigation and menu UI state through compaction, default only a missing interaction mode to Guided, and report missing graph state unknown rather than reconstructing it.

This seed is the compact enforcement contract for virtual protocols. The detailed contract remains owned by this skill rather than the broad command reference or a virtual protocol.

0. **Enter** — if you have no starting note, find one first: `nn list --search "<topic>" --json --fields id,title,score,link_count` and pick the highest-scoring, best-connected hit as your entry focus. If you already have a note ID (from a prior answer, a link, a backlink), start there.
1. **Orient** — fetch topology first, then every body page for the identical depth-1/both traversal (use `--color always` for every color-capable human relay):
   ```
   nn graph show --focus <id> --depth 1 --direction both --zones --presentation-hints --color always --format text
   nn graph bodies --focus <id> --depth 1 --direction both --page 1
   nn graph bodies --focus <id> --depth 1 --direction both --page <N> --snapshot <snapshot-from-page-1>
   ```
   Repeat active link/status/representation filters on both surfaces. Continue through the reported page count, require one repeated snapshot, and reconstruct every selected body's ordered segments—including empty bodies—before presentation. If paging reports a stale or mismatched snapshot, discard the partial batch and restart Orient. Until completion, do not make a body-derived claim or reason. Once complete, you know for the current node: 🔵 TOP — what the focus answers to; 🟢 BOTTOM — what builds on the focus; 🔴 LEFT — what contests or questions the focus; and 🔷 RIGHT — lateral provenance or task relationships. Each neighbor can be joined by ID to its complete body, `↑out ↓in` degree, and zone. Empty zones are information too: `🔴 LEFT — what contests or questions the focus: empty` means nothing contests or questions this focus; `🔵 TOP — what the focus answers to: empty` means the focus is a root. A neighbor with high inbound degree is a hub worth weighting even if it is off your direct path.
2. **Read from here** — the completely reconstructed bodies let you evaluate a move without leaving the view. Zones tell you *what kind* of move each neighbor is; bodies tell you *whether it's the one you want*. Choose the move that closes distance to your goal:

   | Your goal | Zone to step into |
   |-----------|-------------------|
   | What challenges or questions this? | **🔴 LEFT — what contests or questions the focus** |
   | What does this answer to, or where did it come from? | **🔵 TOP — what the focus answers to** or **🔷 RIGHT — lateral provenance or task relationships** |
   | What builds on or operationalizes this? | **🟢 BOTTOM — what builds on the focus** |
   | What is laterally related by provenance or task context? | **🔷 RIGHT — lateral provenance or task relationships** |

3. **Recenter** — pick a neighbor's ID and re-run step 1 with that as `--focus`. Navigation is a chain of ego-hops (the ExcaliBrain model in design note 20260820154156-6576), not a pan over a hairball. Whenever a step (this one, or a return from `peek`/`teleport`) presents the view to a human, first fetch `presentation` and apply its Presentation discipline (P1–P4). Peek returns and Recenter returns MUST restate the semantic triple for every directional option; returning to a title, ID, arrow, or geometry word alone is noncompliant.
4. **Arrive** — stop when the current node answers your goal, or when the zone you were following is empty (for example, `🔴 LEFT — what contests or questions the focus: empty` ends a walk seeking contesting material). Emit the complete [Arrive report](#arrive-report), including the region reached and what changed relative to the starting goal.

These three verbs are **not** `nn` subcommands or flags — they are named moves *you* (the agent) perform by composing existing primitives, exactly like the walk steps above. There is **no new CLI surface**. Modes and options named below (e.g. "flat mode") are presentation choices you make, not shell flags.

#### `peek` — look deep without moving (read-only, focus stays put)

`peek` previews detail **without changing focus** — this is what makes it distinct from a one-step recenter. If it moved focus it would just be a slow walk. Two forms:

- **`peek` on the current node** — expand detail in place: use Orient's topology-first `graph show` plus complete snapshot-bound `graph bodies` page loop to read the current node's full body and its neighbors' bodies without recentering. Peeking consumes the reconstructed evidence without committing to a move.
- **`peek` in a direction** — preview what is down one zone (TOP / BOTTOM / LEFT / RIGHT) before deciding to go there: `nn show <neighbor-id> --depth 2` reads that neighbor and *its* onward links, so you see where the move leads without taking it. Focus stays on your current node throughout.

Use `peek` when Read-from-here isn't enough to judge a move — you want to see one step past the neighbor before committing.

Because `peek` does not move, it returns to the retained positioned frame rather than creating a new one. After a transient Peek result, first fetch `presentation` and follow its Transient rendering policy: in Guided mode, normally show the compact Peek breadcrumb, say focus is unchanged, and reopen Peek; in Advanced mode, retain Peek as the semantic return position but keep help closed. Full-render only for one of the named invalidation conditions. The finding may update or justify a promoted shortcut (for example, `→ Recenter 🔷 RIGHT — lateral provenance or task relationships: move to 4302 — Provenance gaps in generated maps (changes focus) because the peek confirms its body identifies the missing source boundary that the retained goal asks us to resolve`); it does not exempt the return from evidence criteria, effect markers, or semantic-direction enforcement.

#### `teleport` — move far (relocate focus)

`teleport` is how you *get* a focus when you have none, or jump to a distant region. Unlike `scan`/`peek` (which look), `teleport` **relocates** — it changes where you are. Two modes:

- **`teleport` with a query** — land on **structure, not a ranked hit list**. Use `nn clusters --search "<query>" --json --summary` as the **default landing-zone source**: it clusters the complete graph, returns only regions containing search hits, ranks them by top-three normalized matching evidence, supplies a representative without dumping full region membership, and defaults to at most 3 ranked `matches` per region. Total `match_count`, score, ranking, density, and representative remain exhaustive; inspect `matches_returned`/`matches_truncated`, or use `--match-limit 0` when every evidence match is explicitly needed. Recenter on the chosen region's `representative.id`, then re-enter Orient from that note; alternatively recenter on a ranked `matches[].id` to enter through specific evidence. Use `nn list --search "<query>"` separately only when you need note-level evidence beyond the region result, or rerun without `--summary` when complete region membership is explicitly needed.
- **`teleport` random** — a serendipitous jump: `nn random` (optionally `--tag`/`--type`) or `nn shuf` to land somewhere unplanned, then start a fresh walk from there.

`teleport` only relocates — it does not render deep detail (that is `peek`'s job). After landing, **re-enter the walk at Orient (step 1) from the new focus, carrying the full `presentation` reference's Presentation discipline (P1–P4)** — the landing is not a shortcut past it; landing then narrating in prose without colors, map, or chooser is exactly the seam-drift that discipline exists to prevent. This is the one move worth invoking from cold — it is named in the session-start CLI-reference pointer for that reason.

**The landing itself is a chooser point** — teleport's landing *is* a Recenter decision ("which region/hub do I land on?"). When a human is driving on a chooser-capable harness, render genuinely ambiguous candidate landings in a chooser of at most four rows. The source is JSON, so the pre-landing Teleport chooser must manually mark each candidate landing region with the stable relay palette before any focus exists. Each candidate is a concrete focus-changing action, so prefix it with `→` and include the adjacent effect legend. Its bounded option set is the recommended hub, additional cluster/sub-territory candidates as capacity permits, and a conversational `Teleport random` escape as the "somewhere else" option. Because no positioned frame exists yet, Focus + Map + Moves and the adaptive picker begin immediately after the landing, not before it.

**Selection completes the landing decision.** Once the human selects a candidate—or the goal makes one candidate unambiguous—the selected landing automatically becomes the retained focus. Immediately run Orient from it and present Focus + Map + Moves. Teleport **MUST NOT ask for a second confirmation** such as "visit this focus?" after selection and **MUST NOT offer a separate `Visit` action**. Ask for human choice only while candidates remain genuinely ambiguous and no landing has yet been selected.

**Offer the adaptive picker when the harness supports one, a human is driving the walk, and Navigation help should be open.** A harness picker renders contextual decisions as clickable actions rather than a wall of prose. Use it only when the harness exposes a chooser affordance, a human is co-navigating this walk, and mode is Guided or Advanced has temporary help open. Advanced mode otherwise keeps it closed. When those conditions are false—because help is closed, navigation is autonomous, or stdout is not a TTY—do **not** invoke a picker: pick the move yourself per the goal and continue (nn is non-interactive by default when stdout is not a TTY).

When you offer it, first fetch `presentation` and use its Adaptive hierarchical quick-actions picker, not a neighbor-only or strict-flat chooser. A promoted Recenter targets the goal-relevant semantic triple—`🔴 LEFT — what contests or questions the focus`, `🔵 TOP — what the focus answers to`, `🔷 RIGHT — lateral provenance or task relationships`, or `🟢 BOTTOM — what builds on the focus`—and its description includes that complete triple, target ID and readable title, `↑out ↓in` degree, focus mutation, and a one-line substantive body-derived reason. Other promoted actions likewise name their class, focus behavior, and evidence reason. Then execute the selected concrete action directly. Teleport, Visit, Recenter, and Go to may adopt a new destination; Back and Forward may only restore retained frames. No other action changes focus.
