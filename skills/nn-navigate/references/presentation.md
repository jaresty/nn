---
name: presentation
applies_when: "When presenting any human-facing positioned view or chooser; fetch before Orient relay, picker rendering, transient return, Scan relay, Teleport landing, Back/Forward restoration, or Arrive presentation."
---

# Reference: Presentation

## Human-driven navigation invariant

Every presented navigation step **MUST KEEP DISCOVERABLE** the four positioned navigation action classes plus the human-consultation class:

1. **Recenter — move focus**
2. **Peek — inspect without moving**
3. **Scan — zoom out without moving**
4. **◇ Ask… — suspend for one bounded human decision while retaining the complete frame**
5. **Arrive — stop**

Ask is neither a look nor a move. An action may be unavailable only when structurally impossible, and the response **MUST** state why. **Peek and Scan MUST be discoverable** in a human-driven walk. When Navigation help is open in a chooser-capable harness, Recenter, Peek, Scan, and Ask are available through `All navigation actions…` exactly one level below the top picker; Arrive is always the final top-level action. A contextual shortcut may also promote one concrete action to the top. Guided keeps that help open; Advanced does not render it unless `:help`, ambiguity, or an unavailable target temporarily requires contextual help.

Before every picker, present:

A. **Focus** — type, status, degree, central claim, neighborhood role
B. **Map** — positional zones, edge meanings, empty zones
C. **Moves** — compact-label references, complete semantic triples, and decision-relevant action reasons

This **MUST PRESENT** rule applies before the mode-specific details below; Teleport, Peek, Scan, and Recenter all reference it rather than weakening it. On an unchanged transient compact return, the complete frame already visible in the conversation satisfies this rule; do not duplicate it merely to reopen the invoking picker.

## Topology-first source acquisition gate

For every full positioned presentation, retrieve topology first with `nn graph show` without `--bodies`, then retrieve the identical focus/depth/direction/link/status/representation set with `nn graph bodies`. Start at page 1, read its total and snapshot, retrieve every page in order using the same snapshot, and reconstruct each ID from complete one-based UTF-8 segments. Preserve the explicit empty-body segment. Any missing page, changed page count, repeated/missing segment, different snapshot, stale-token error, or topology/body ID mismatch blocks presentation.

Topology may establish IDs, zones, stored edges, degrees, and relay budgets while paging. It cannot establish a body-derived central claim, body-based recommendation, summary, or substantive concrete-action reason. Do not draft such claims speculatively and then “verify” them later. Body-derived presentation begins only after all pages and all selected bodies reconstruct exactly. Keep the metadata boundary visible: topology, frontmatter, links, transport envelope/ordinals, relay hints, and agent commentary are not stored body prose.

## Navigation help by interaction mode

Navigation help is the contextual adaptive menu on chooser-capable surfaces and an equivalent compact labeled action list on other human conversational surfaces. It is presentation, not a CLI shell. Guided mode is the default and keeps Navigation help available while another navigation choice is naturally pending. A completed action selected from a Guided picker always reopens its invoking menu. Only a completed action requested directly in conversation may instead use a **quiet return**: report completion, retain the complete frame and semantic menu position, and show the compact `navigate`/`:help` affordance without immediately forcing another chooser. Advanced mode keeps the help closed. `:help` opens a complete canonical command catalog temporarily without changing Advanced mode; dismissal or one completed action closes it again. This catalog is a textual reference, not a picker, so contextual picker row limits do not truncate it. `:guided` makes help persistent when another choice is pending, while `:advanced` closes it immediately. An unanswered Quiz and an awaiting Ask retain their waiting UI rather than claiming the action completed.

The complete catalog lists every core shorthand with its human label and argument shape: `Help — :help`; `Guided mode — :guided`; `Advanced mode — :advanced`; `Look at current positioned view — :look`; `Where am I? — :where`; `Orient or refresh — :orient`; `Recenter — :recenter "<label>"`; `Peek — :peek "<label>"`; `Show verbatim — :show`; `Explain in depth — :explain`; `Find gaps — :gaps`; `Local territory — :scan local`; `Global landscape — :scan global`; `Analogize — :analogize`; `Find an analog — :find-analog`; `Visualize — :visualize`; `Quiz — :quiz`; `Ask… — :ask`; `Back — :back`; `Forward — :forward`; `Bookmark — :bookmark "<name>"`; `Go to bookmark — :goto "<bookmark>"`; and `Arrive — :arrive`. Contextual target rows use their actual grounded labels, for example `Recenter on Replay-safe checkpointing — :recenter "Replay-safe checkpointing"` and `Peek at Replay-safe checkpointing — :peek "Replay-safe checkpointing"`. Contextual menus may hide unavailable actions or summarize controls to respect row limits, but `:help` itself always shows the complete catalog.

For `:recenter "<label>"` and `:peek "<label>"`, resolve only labels in the current complete-frame help/menu context. A full current-context menu label wins; otherwise require one unique case-insensitive fragment among those labels. Never mint aliases from titles, summaries, likely intent, prior hidden menus, or search results. On multiple matches, make no move and temporarily render narrowed help containing only those matches. On no available match, do not guess or search for a substitute: retain state, explain unavailability, and show only applicable contextual help. Bookmark names are excluded from fragment matching and remain exact-case state keys.

## Adaptive hierarchical quick-actions picker

When a human is driving a positioned walk, a chooser is available, and Navigation help is open because mode is Guided or help is temporary, the **top-level picker has at most four rows**: **up to one evidence-backed contextual concrete shortcut**; a stable **`Lenses…`** row; a stable `All navigation actions…` row; and the **final row is always `■ Arrive`**. No shortcut is correct when the retained evidence does not justify one. Generic availability is not evidence, and action classes must not be promoted merely to fill rows.

Every promoted shortcut:

1. names its action class (`Recenter`, `Peek`, `Scan`, or `Lens`); a promoted consultation names `Ask`;
2. states whether selecting it changes or retains focus; and
3. gives a body- or evidence-derived reason why this concrete action matters now.

Every concrete quick-action target MUST include its ID, readable target title, and a substantive body- or evidence-derived reason that explains why the target matters now. A category phrase or relationship label is not evidence: `supporting experiment` alone is not a substantive reason, because it neither states what the experiment found nor how that finding advances the retained goal.

A promoted shortcut is concrete, such as a named destination, exact note body, specific explanation, particular lens, or bounded scan. Selecting it executes directly rather than opening its category submenu. A specific promoted `Lens` action may occupy a contextual shortcut slot when its evidence criterion is met, and the stable `Lenses…` row remains present. Any directional shortcut or reason still includes the complete semantic triple required below.

Use exactly these promotion criteria:

- Analog candidate survives correspondence mapping, where it holds, and where it breaks.
- Visualize when the evidence has process, state, or relationship structure.
- Find gaps when the retained goal needs missing evidence, unanswered questions, unresolved tension, or plausible unlinked context identified around the exact focus.
- Quiz when the sources support a consequential distinction.
- Show verbatim when exact wording matters.
- Explain in depth when the focus is connected, contested, complex, or load-bearing.
- Ask only when human input could materially change the next decision; do not promote it when the move is unambiguous or expected decision improvement is smaller than interruption cost.
- Recenter when a specific destination clearly advances the retained goal.

### Stable menus, breadcrumbs, and effects

The menu model is conversation-scoped UI state: retain the interaction mode plus the **current menu and menu stack**. Its stable menu names are **Quick actions, All actions, Recenter destinations, Peek, Scan, and Lenses**; **Ask** is the stable decision-oriented consultation submenu. The numbered menus below define stable semantic rows; when rendered as Navigation help, actionable labels are decorated beside their canonical shorthand without changing the semantic row or effect marker. Every submenu picker visibly displays `Esc` in adjacent guidance. The guidance names that submenu's parent menu using `Esc: back to <parent-menu-name>`. This is adjacent text, not an option row, canonical submenu rows remain unchanged, and each submenu picker still contains at most four rows. This is not notebook state and MUST NOT be written to note files, frontmatter, links, the index, Git, configuration, or environment.

The top breadcrumb is `<short-id> · Quick actions`. Every submenu shows this breadcrumb form: **`<short-id> · Quick actions › ...`**, using stable menu names for each level, for example `<short-id> · Quick actions › All actions › Peek` and `<short-id> · Quick actions › Lenses`.

Every picker that contains a concrete action carries this adjacent legend: **→ focus changes; ○ focus retained; ■ stops; ↗ explores beyond local. ◇ human consultation; retained focus and history**. Every concrete action row includes exactly one applicable effect marker. Category rows such as `Recenter`, `Peek`, `Scan`, `Lenses…`, and `All navigation actions…` do not execute an action and therefore have no effect marker. Promoted shortcuts and Recenter destinations still include readable titles and substantive body- or evidence-derived reasons; the marker never replaces that content.

`All navigation actions…` opens the action-class submenu exactly one level away, under `<short-id> · Quick actions › All actions`, with exactly these rows and no duplicate Arrive:

1. `Recenter`
2. `Peek`
3. `Scan`
4. `Ask…`

`Recenter` may open `<short-id> · Quick actions › All actions › Recenter destinations`. Its evidence-backed destinations are capped at four rows and each begins with `→` because selecting it changes focus.

`Peek` opens `<short-id> · Quick actions › All actions › Peek` with exactly:

1. `○ Show verbatim`
2. `○ Explain in depth`
3. `↗ Find an analog`

Show verbatim, Explain in depth, and Find an analog remain Peek actions. Find an analog is a human-intent Peek-family action that retains focus and internally uses Scan retrieval across another region. That implementation detail does not place it in the Scan picker.

The `Lenses…` row is always present at the top level and opens this exact submenu one picker level away under `<short-id> · Quick actions › Lenses`:

1. `○ Analogize`
2. `○ Find gaps`
3. `○ Visualize`
4. `○ Quiz`

`Scan` opens `<short-id> · Quick actions › All actions › Scan` with exactly:

1. `○ Local territory`
2. `↗ Global landscape`

Scan contains exactly Local territory and Global landscape; it has no Find an analog row.

### Direct intents and menu return transitions

Treat **show, explain, analogize, find an analog, find gaps, visualize, quiz, scan, and arrive** as direct conversational intents from any menu, not `nn` subcommands. Treat **ask** the same way when a positioned frame exists. Resolve them without forcing the human to traverse the hierarchy. A bare `scan` opens Scan so the human can choose altitude; an explicit local or global scan executes directly. Direct action vocabulary is presentation-level only and adds no CLI surface.

Returns are deterministic and restore the relevant semantic menu position without changing focus, history, or interaction mode:

- A Lens invoked from Lenses returns to Lenses.
- Show verbatim, Explain in depth, and Find an analog return to Peek.
- Local territory and Global landscape return to Scan.
- Ask completion or Cancellation reopens the invoking Ask submenu in Guided mode; a promoted Ask returns to Quick actions, always with the complete prior frame unchanged.
- A promoted top-level transient returns to Quick actions.
- Quiz suspends the picker while unanswered; completion, pass, skip, or `I don't know` returns to the Lenses or Quick actions menu that invoked it.
- `navigate` aborts Quiz and returns to Quick actions.

In Guided mode, a completed return reopens Navigation help at that menu when another choice is naturally pending. A completed action selected from a Guided picker always reopens its invoking menu. Only a completed action requested directly in conversation may use a quiet return: retain the semantic menu position, report completion and unchanged focus, and show `navigate`/`:help` without forcing a picker. In Advanced mode, the same return position is retained but the help stays closed; if the action was selected from temporary `:help`, completion closes that help. Ambiguity and unavailability may open narrowed temporary help but never switch mode.

Focus-changing Recenter, Teleport, Back, Forward, and Go to rerun Orient and reset the menu to Quick actions. `navigate` always reruns Orient and resets to Quick actions. Both preserve interaction mode: Guided renders Quick actions help and Advanced leaves it closed. Arrive stops and retains focus; Guided still presents contextual Navigation help after the completed Arrive report, while Advanced uses only the navigation-return affordance.

### Transient rendering policy

Do not redundantly rerender Focus + Map + Moves after a transient action when the complete retained frame and notebook are unchanged. In Guided mode, show the compact breadcrumb, say focus is unchanged, and reopen Peek, Lenses, Scan, or Quick actions when another choice is naturally pending. A completed picker selection always reopens that invoking menu. Only when an action requested directly in conversation clearly answers that request may it use a quiet return: retain the semantic menu position and complete frame, report completion and unchanged focus, and show the compact `navigate`/`:help` affordance without invoking a chooser. This compact return is not a degraded full map; it reuses the complete frame already visible in the conversation. In Advanced mode, report the transient result and unchanged focus without reopening a picker; temporary help closes on completion. Thus deterministic return selects a semantic menu position, while completion context and interaction mode control whether that menu is rendered.

Perform a full render only when the focus, filters, traversal context, or notebook changed; when navigation resumes after discussion; after an explicit Refresh; or when the cached frame is stale or unknown. A full render preserves the readable full map and all presentation discipline: visible nodes keep IDs, readable titles, and substantive body-derived reasons, with immediately adjacent legends for compact labels. Full frame rendering does not force Advanced Navigation help open. `navigate` still reruns Orient and resets Quick actions; after discussion it full-renders, while an unchanged Quiz abort may compactly reopen Quick actions in Guided mode after Orient confirms the cache.

### Full-frame visual hierarchy

Keep the stable section order **Focus → Map → Moves**. Within each section and concrete action, layer information for scanning in this order:

1. **Identity and action** — what or where this is, and what selecting it does;
2. **Relationship meaning** — the complete marker, zone, and local semantic meaning;
3. **Evidence and state effect** — the body-derived reason, view-local importance depth, visible connectivity, and whether focus/history changes.

Use headings, line breaks, and typographic emphasis to expose those layers instead of flattening every field into one dense sentence. **This hierarchy changes emphasis, not content**: it never removes required IDs, readable titles, body-derived claims, semantic triples, marker legends, degrees, evidence reasons, or one-level access to Recenter, Peek, Scan, and Arrive.

### Semantic-direction enforcement

The policy applies to every human-facing move, chooser label and description, recommendation, Peek return, and Recenter return: each **MUST** name the stable emoji marker, zone name, and local relationship meaning together. The aligned map geometry is the one compact exception: each bracketed node label keeps its zone emoji directly beside it, while an immediately adjacent zone key supplies the full zone name and local meaning without putting long prose inside alignment columns. The required semantic triples are exactly:

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
- → Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> (changes focus) because <substantive body-derived reason tied to the goal>
- Lenses…
- All navigation actions…
- ■ Arrive

Legend: → focus changes; ○ focus retained; ■ stops; ↗ explores beyond local. ◇ human consultation; retained focus and history.

Relationship templates embedded in a complete promoted label include `→ Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> because <substantive body-derived reason>` and `○ Peek 🔴 LEFT — what contests or questions this focus: inspect <id> — <readable target title> because <substantive body-derived reason>`; the complete label must also state the applicable focus mutation.

**Navigation presentation check:**
- [ ] focus type and status shown
- [ ] zone/type/edge color markers applied (stable relay palette)
- [ ] compact colored labels occupy their true positions while semantic prose stays outside alignment columns
- [ ] an immediately adjacent zone key maps every occupied label to its zone name and local meaning
- [ ] focus degree shown
- [ ] central claim summarized
- [ ] neighborhood role explained
- [ ] positional map uses compact stable labels by default, with TOP above, LEFT left, RIGHT right, and BOTTOM below focus
- [ ] every empty zone visibly shows `[∅]` in its geometric slot
- [ ] direct focus relationships show canonical type and stored direction in geometry; secondary stored relationships appear in the adjacent complete ledger
- [ ] every arrowhead sits beside its stored target, including separately typed reciprocal vertical relationships
- [ ] an immediately adjacent complete evidence index maps every label exactly once to ID, readable title, note type, degree, importance marker, and body-derived claim
- [ ] at most two `★` decision-shaping nodes receive expanded summaries; `◆` supporting nodes receive one sentence; `·` context nodes receive one clause
- [ ] degree remains visible connectivity and never establishes importance by itself
- [ ] no orphan, duplicate, title-only, or ID-only labels
- [ ] Recenter available one level away
- [ ] Peek available one level away
- [ ] Scan available one level away
- [ ] ◇ Ask available one level away with a concrete bounded decision when promoted
- [ ] Arrive always visible as final top-level action

Run this check internally before invoking a picker. A missing item blocks presentation unless it is structurally unavailable and the response explains why.

<a id="presentation-discipline"></a>
#### Presentation discipline (the named block every seam cites)

**This is the single source of truth for how any walk view is presented to a human. Every point that presents a positioned view — Orient, the return after `peek`, the landing after `teleport`, `scan` — cites this block by name rather than restating it. A rule stated only at one step definition does not travel across a seam that re-enters the walk from elsewhere; centralizing it here is what stops per-seam drift.** When presenting to a human on a capable surface, all four apply jointly:

- **P1 — Colors and relay budgets on.** Every color-capable human-facing navigation view uses the stable markers below—not only post-landing Orient, but also the JSON-backed pre-landing Teleport chooser, the return after Peek, Scan at both altitudes, and Arrive. Run topology first with the underlying `nn graph show … --zones --presentation-hints --color always`, then complete the identically filtered `nn graph bodies` page loop before deriving body claims. Graph text sources MUST use `--color always` for a color-capable human relay; do not trust `auto`, because tool stdout is commonly non-TTY. JSON sources are marker-free by design: parse them, then manually apply the relay palette to the human-facing chooser, headings, map, focus, and region labels—never mutate or claim markers exist in the JSON. The stable emoji marker and textual zone/meaning triple remains mandatory even when the surrounding surface cannot render terminal color; `--presentation-hints` keeps each topology node's relay budget in context while its body arrives losslessly through the separate transport.
- **P2 — Focus + Map + Moves.** Give the Focus summary (substance), positional/ASCII Map plus immediately adjacent importance-scaled evidence index and secondary stored-relationship ledger (structure and neighbor substance), and compact Moves (direction and decision reason) defined above. Every directional use in all three parts follows [Semantic-direction enforcement](#semantic-direction-enforcement). None is sufficient alone: a map without its index is a skeleton you can't read; an index without a map loses the spatial relationships. Never relay raw command output. (Detailed as (a)/(b)/(c) below.)
- **P3 — Adaptive hierarchical quick-actions picker.** When the harness exposes a chooser affordance, a human is co-navigating, and Navigation help is open under Guided mode or temporary help, show up to one evidence-backed contextual concrete shortcut, a stable `Lenses…` row, a stable `All navigation actions…` row, and the final row is always `■ Arrive`; keep Lenses and Recenter / Peek / Scan / Arrive exactly one level away through those stable rows. Apply breadcrumbs, shorthand labels, effect markers, menu-stack transitions, and mode-aware compact-return rules from the picker contract. In Advanced mode do not open the picker automatically. On a surface without a chooser, use the equivalent labeled help only when mode requires it; an autonomous run picks the move per the goal while keeping all four action classes discoverable in prose. This applies wherever a positioned walk presents an onward decision, including after `peek` and a completed `teleport` landing.
- **P4 — View-local importance summaries.** Classify every neighbor from the retained goal and complete evidence as at most two `★` decision-shaping nodes, `◆` decision-supporting nodes, or `·` orienting context. Degree remains connectivity, not importance, and is only a final tie-breaker. Scale evidence-index depth by this importance class.

##### Stable emoji relay palette

Use this exact palette in every color-capable human-facing navigation view. It extends the graph command's actual zone convention and remains stable across source formats:

- `🔵 TOP` — what the focus answers to;
- `🟢 BOTTOM` — what builds on the focus;
- `🔴 LEFT` — what contests or questions the focus;
- `🔷 RIGHT` — lateral provenance or task relationships;
- `🟠 FOCUS / REGION` — the retained focus and its current region. Before a Teleport landing exists, use the same orange marker for each candidate landing region and label the recommended candidate explicitly.

Preserve note-type markers and edge-family markers already emitted by graph text instead of remapping them. Agent-drawn positional or region maps must use this palette too. You must **reproduce this legend** in the presentation itself — a short key mapping each marker to its meaning (the five zone markers above, plus the note-type and edge-family markers you carried through) — because a colored marker the reader cannot decode carries no information; the markers and their legend travel together. Plain, uncolored Focus + Map + Moves is noncompliant on a color-capable surface even when its prose and geometry are otherwise correct.

The three parts, in detail:

##### Compact spatial map and complete evidence index

Compact stable labels are the default map representation for every full positioned view. Assign deterministic labels within the view by semantic zone and local order—for example `[T1]`, `[L1]`, `[R1]`, and `[B1]`—and use those labels consistently in Map and Moves. A full inline map is permitted only when the view has one or two neighbors and explicitly states why inline form is clearer; in that exception each inline node still carries its note ID, readable title, note type, degree, importance marker, and importance-scaled body-derived central claim. The inline form replaces compact labels only for that justified view; it never weakens identity, substance, connectivity, importance, edge, geometry, or semantic-direction requirements.

TOP is above focus, LEFT is left, RIGHT is right, and BOTTOM is below. Geometry is semantic, not decorative: never stack RIGHT or LEFT beneath focus merely because prose is long. Every empty zone visibly occupies its geometric slot as `[∅]`; a prose-only empty-zone statement is insufficient. The aligned geometry block keeps each zone emoji and `★`/`◆`/`·` importance marker directly beside its compact label while semantic prose stays in the immediately adjacent zone key.

Direct focus relationships appear in the geometry. Classify stored edges into semantic zones before drawing them, then use **one post-classification vertical renderer** for TOP and BOTTOM. The classifier and each zone's semantic gloss remain distinct; only the rendering grammar is shared. Mirroring placement never mirrors graph meaning: every stored source, canonical type, and stored target remain invariant, and each form keeps its arrowhead beside its stored target.

Select the vertical form from the exact classified edge set:

- an empty set visibly renders `[∅]`;
- exactly one stored edge uses an attached typed stem;
- exactly one reciprocal pair over one focus/node pair uses one attached stem with separately named canonical types and distinct target arrows;
- every other edge set uses a **centered borderless attached rail** with one endpoint-complete unit per stored edge.

The rail remains attached to the vertical-zone axis; it is direct-edge geometry, not a detached box or a secondary ledger. Each endpoint-complete unit renders stored order explicitly, even when that repeats `[FOCUS]` or a compact node label. Repeated labels are endpoint references; the evidence index still maps each note label exactly once. For example, a dense BOTTOM rail may contain:

```text
                    🟠 [FOCUS]
                         │
                    🟢 BOTTOM
                         │
        ★ [B1] ──grounded-by──→ [FOCUS]
        ◆ [B2] ──grounded-by──→ [FOCUS]
        [FOCUS] ──refines──────→ ★ [B3]
        [FOCUS] ──refines──────→ ◆ [B4]
        [FOCUS] ──refines──────→ ★ [B5]
```

A dense TOP rail uses the same post-classification renderer above focus with zone-valid edges; never place an identical stored edge in another zone merely to mirror the picture. At narrow width, wrap only inside that edge unit so the arrow and stored target remain on the same line:

```text
        [FOCUS]
          refines
          → ◆ [B2]
```

Direct focus relationships remain in geometry; neighbor-to-neighbor stored relationships remain in the secondary stored-relationship ledger with canonical type and stored source-to-target direction. The centered rail is required whenever a shared stem would merge types, hide multiplicity, or make edge-to-node correspondence depend on row order.

Immediately below the map, render one complete evidence index that maps every compact label exactly once to its note ID, readable title, note type, degree, importance marker, and body-derived central claim. Scale each claim under P4. The index is the single identity-and-substance surface for mapped neighbors: no orphan, duplicate, title-only, or ID-only labels, and no deferred key in another section, picker, or turn. A compact unchanged transient return may reference the complete frame already visible instead of redrawing it.

Moves reference the compact labels, retain the complete marker/zone/local-meaning semantic triple and action effect, and add only the decision-relevant action reason instead of repeating complete evidence-index entries. Focus retains its full identity and central claim in the Focus section. Concrete quick actions still name their target ID and readable title plus the substantive evidence-derived reason required by the picker contract, because they may be acted on independently of the map.

Empty zones have no evidence-index entry, but retain `[∅]` in their literal geometric slot plus the complete semantic gloss: marker, zone name, local relationship meaning, and what the emptiness means for this focus. Region blobs that do not represent individual notes likewise need a semantic region gloss; any individual representative, bridge, match, or candidate note shown inside them receives a compact label and complete adjacent index entry.

**Compliant default compact map, evidence index, and ledger:**

```text
                              🔵 TOP
                              ◆ [T1]
                       source-of ▲
                                 │
                                 ▼ supports
🔴 LEFT · [∅] ───────────── 🟠 [FOCUS] ──source-of──→ ★ [R1] 🔷 RIGHT
                                 │
                             🟢 BOTTOM
                               · [∅]

Zone key
🔵 [T1] = TOP — what the focus answers to
🔴 [∅] = LEFT — what contests or questions the focus: empty
🔷 [R1] = RIGHT — lateral provenance or task relationships
🟢 [∅] = BOTTOM — what builds on the focus: empty
🟠 [FOCUS] = retained focus

Evidence index
◆ [T1] = 20250101120000-t001 — Accepted recovery policy — concept · ↑1 ↓5 —
         Defines the durable boundary that retries must restore.
★ [R1] = 20250101120000-r001 — Recovery design review — observation · ↑2 ↓1 —
         Supplies the decision-changing review evidence. Its constraints determine
         whether the retained policy can be adopted and what must change first.

Secondary stored relationships
[R1] ──supports──→ [T1]

Moves
- [R1] — Recenter 🔷 RIGHT — lateral provenance or task relationships because its review evidence changes the retained goal's next decision.

Legend: ★ decision-shaping; ◆ decision-supporting; · orienting context. ▲/▼ and → point to stored targets.
```

**Noncompliant skeletal presentation:**

```text
Map: FOCUS -> [N1] -> 4302
Moves: 4302; supporting experiment
Quick action: Recenter BOTTOM to 4302 because supporting experiment
```

This is bad because `[N1]` is orphaned, `4302` is ID-only, neither node has a readable title and body-derived central claim, the direction omits its semantic triple, and `supporting experiment` names only a vague role rather than substantive evidence.

**(a) Summarize the current note.** Before the map, characterize where the reader is standing: what this note *is* (type/status and body-derived central claim) and how it sits in all four semantic zones. Scale the focus explanation to its actual complexity and role in the retained goal, not to degree alone.

**(b) Draw the map.** Follow the compliant example and shared vertical renderer above: all four zone slots remain literal, `[∅]` marks emptiness, long prose stays outside alignment columns, direct focus edges remain in geometry, and every arrowhead sits beside its stored target. Select vertical stems or rails from the exact classified edge set rather than node count, and preserve one endpoint-complete unit per dense stored edge. Put the complete zone key, evidence index, and secondary stored-relationship ledger immediately below the geometry. Do not serialize zones into a prose list, detach a dense vertical comb from its nodes, or draw neighbor-to-neighbor edges through the positional cross.

**(c) Assign importance, build the complete evidence index, and name Moves.** Evaluate each neighbor in this order: explicit retained goal or human selection; whether removing its evidence changes the next action; relationship consequence; unresolved leverage; specificity; then degree only as a final tie-breaker. Use at most two `★` decision-shaping nodes (2–3 sentences), `◆` decision-supporting nodes (one sentence), and `·` orienting context nodes (one claim clause). Every entry still includes ID, readable title, note type, and `↑out ↓in` connectivity. Put identity and substance once in the adjacent evidence index, then let Moves reference compact labels and add only why the action matters relative to the goal.

The body transport remains lossless: it splits large bodies into bounded segments, and the presentation gate waits for all pages before summarizing. `--presentation-hints` supplies degree-based source hints, but those hints do not establish view-local importance or final summary length. Empty zones retain `[∅]` in geometry and the full semantic gloss beside it.

Keep `--depth 1` when zoning—only direct neighbors carry a zone, and the zoned text view omits unzoned nodes. Use topology alone only when you need shape without claims. When substance is required, use `nn graph bodies` with the identical traversal, retrieve every page under the same snapshot, reconstruct every body, and only then read and relay it. The deprecated `nn graph show --bodies` compatibility path is unbounded and is not a navigation source.
