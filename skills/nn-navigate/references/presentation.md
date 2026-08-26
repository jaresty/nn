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
C. **Moves** — degree-scaled neighbor summaries and recommended direction

This **MUST PRESENT** rule applies before the mode-specific details below; Teleport, Peek, Scan, and Recenter all reference it rather than weakening it. On an unchanged transient compact return, the complete frame already visible in the conversation satisfies this rule; do not duplicate it merely to reopen the invoking picker.

## Topology-first source acquisition gate

For every full positioned presentation, retrieve topology first with `nn graph show` without `--bodies`, then retrieve the identical focus/depth/direction/link/status/representation set with `nn graph bodies`. Start at page 1, read its total and snapshot, retrieve every page in order using the same snapshot, and reconstruct each ID from complete one-based UTF-8 segments. Preserve the explicit empty-body segment. Any missing page, changed page count, repeated/missing segment, different snapshot, stale-token error, or topology/body ID mismatch blocks presentation.

Topology may establish IDs, zones, stored edges, degrees, and relay budgets while paging. It cannot establish a body-derived central claim, body-based recommendation, summary, or substantive concrete-action reason. Do not draft such claims speculatively and then “verify” them later. Body-derived presentation begins only after all pages and all selected bodies reconstruct exactly. Keep the metadata boundary visible: topology, frontmatter, links, transport envelope/ordinals, relay hints, and agent commentary are not stored body prose.

## Navigation help by interaction mode

Navigation help is the contextual adaptive menu on chooser-capable surfaces and an equivalent compact labeled action list on other human conversational surfaces. It is presentation, not a CLI shell. Guided mode is the default and keeps Navigation help persistent: render it after every completed action, even when the positioned frame can use a compact unchanged return. Advanced mode keeps the help closed. `:help` opens the current contextual help temporarily without changing Advanced mode; dismissal or one completed action closes it again. `:guided` makes help persistent, while `:advanced` closes it immediately. An unanswered Quiz and an awaiting Ask retain their waiting UI rather than claiming the action completed.

Every rendered help action shows its human label beside the canonical shorthand from the core. Examples are `Where am I? — :where`, `Orient — :orient`, `Show verbatim — :show`, `Explain in depth — :explain`, `Find gaps — :gaps`, `Local territory — :scan local`, `Global landscape — :scan global`, `Analogize — :analogize`, `Find an analog — :find-analog`, `Visualize — :visualize`, `Quiz — :quiz`, `Ask… — :ask`, and `Arrive — :arrive`. Target rows use their actual grounded labels, for example `Recenter on Replay-safe checkpointing — :recenter "Replay-safe checkpointing"` and `Peek at Replay-safe checkpointing — :peek "Replay-safe checkpointing"`. A compact control key may teach `:back`, `:forward`, `:bookmark "<name>"`, `:goto "<bookmark>"`, `:guided`, `:advanced`, and `:help` without turning Back into a picker row or exceeding picker row limits.

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

The menu model is conversation-scoped UI state: retain the interaction mode plus the **current menu and menu stack**. Its stable menu names are **Quick actions, All actions, Recenter destinations, Peek, Scan, and Lenses**; **Ask** is the stable decision-oriented consultation submenu. The numbered menus below define stable semantic rows; when rendered as Navigation help, actionable labels are decorated beside their canonical shorthand without changing the semantic row or effect marker. This is not notebook state and MUST NOT be written to note files, frontmatter, links, the index, Git, configuration, or environment.

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

Show verbatim and Explain in depth remain Peek actions. The `Lenses…` row is always present at the top level and opens this exact submenu one picker level away under `<short-id> · Quick actions › Lenses`:

1. `○ Analogize`
2. `↗ Find an analog`
3. `○ Find gaps`
4. `○ Visualize`
5. `○ Quiz`

Find an analog is a human-intent Lens and internally uses Scan retrieval across another region. That implementation detail does not place it in the Scan picker.

`Scan` opens `<short-id> · Quick actions › All actions › Scan` with exactly:

1. `○ Local territory`
2. `↗ Global landscape`

Scan contains exactly Local territory and Global landscape; it has no Find an analog row.

### Direct intents and menu return transitions

Treat **show, explain, analogize, find an analog, find gaps, visualize, quiz, scan, and arrive** as direct conversational intents from any menu, not `nn` subcommands. Treat **ask** the same way when a positioned frame exists. Resolve them without forcing the human to traverse the hierarchy. A bare `scan` opens Scan so the human can choose altitude; an explicit local or global scan executes directly. Direct action vocabulary is presentation-level only and adds no CLI surface.

Returns are deterministic and restore the relevant semantic menu position without changing focus, history, or interaction mode:

- A Lens invoked from Lenses returns to Lenses.
- Show verbatim and Explain in depth return to Peek.
- Local territory and Global landscape return to Scan.
- Ask completion or Cancellation reopens the invoking Ask submenu in Guided mode; a promoted Ask returns to Quick actions, always with the complete prior frame unchanged.
- A promoted top-level transient returns to Quick actions.
- Quiz suspends the picker while unanswered; completion, pass, skip, or `I don't know` returns to the Lenses or Quick actions menu that invoked it.
- `navigate` aborts Quiz and returns to Quick actions.

In Guided mode, each completed return renders Navigation help at that menu, so the help/menu remains persistent and continues teaching shorthand. In Advanced mode, the same return position is retained but the help stays closed; if the action was selected from temporary `:help`, completion closes that help. Ambiguity and unavailability may open narrowed temporary help but never switch mode.

Focus-changing Recenter, Teleport, Back, Forward, and Go to rerun Orient and reset the menu to Quick actions. `navigate` always reruns Orient and resets to Quick actions. Both preserve interaction mode: Guided renders Quick actions help and Advanced leaves it closed. Arrive stops and retains focus; Guided still presents contextual Navigation help after the completed Arrive report, while Advanced uses only the navigation-return affordance.

### Transient rendering policy

Do not redundantly rerender Focus + Map + Moves after a transient action when the complete retained frame and notebook are unchanged. In Guided mode, return with the compact breadcrumb, state that focus is unchanged, and reopen the invoking picker. This compact return is not a degraded full map; it reuses the complete frame already visible in the conversation. In Advanced mode, report the transient result and unchanged focus without reopening a picker; temporary help closes on completion. Thus deterministic return selects a semantic menu position, while interaction mode controls whether that menu is rendered.

Perform a full render only when the focus, filters, traversal context, or notebook changed; when navigation resumes after discussion; after an explicit Refresh; or when the cached frame is stale or unknown. A full render preserves the readable full map and all presentation discipline: visible nodes keep IDs, readable titles, and substantive body-derived reasons, with immediately adjacent legends for compact labels. Full frame rendering does not force Advanced Navigation help open. `navigate` still reruns Orient and resets Quick actions; after discussion it full-renders, while an unchanged Quiz abort may compactly reopen Quick actions in Guided mode after Orient confirms the cache.

### Full-frame visual hierarchy

Keep the stable section order **Focus → Map → Moves**. Within each section and concrete action, layer information for scanning in this order:

1. **Identity and action** — what or where this is, and what selecting it does;
2. **Relationship meaning** — the complete marker, zone, and local semantic meaning;
3. **Evidence and state effect** — the body-derived reason, degree-scaled detail, and whether focus/history changes.

Use headings, line breaks, and typographic emphasis to expose those layers instead of flattening every field into one dense sentence. **This hierarchy changes emphasis, not content**: it never removes required IDs, readable titles, body-derived claims, semantic triples, marker legends, degrees, evidence reasons, or one-level access to Recenter, Peek, Scan, and Arrive.

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
- → Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> (changes focus) because <substantive body-derived reason tied to the goal>
- Lenses…
- All navigation actions…
- ■ Arrive

Legend: → focus changes; ○ focus retained; ■ stops; ↗ explores beyond local. ◇ human consultation; retained focus and history.

Relationship templates embedded in a complete promoted label include `→ Recenter 🔵 TOP — what this focus answers to: move to <id> — <readable target title> because <substantive body-derived reason>` and `○ Peek 🔴 LEFT — what contests or questions this focus: inspect <id> — <readable target title> because <substantive body-derived reason>`; the complete label must also state the applicable focus mutation.

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
- [ ] every visible non-focus node has ID, readable title, and body-derived claim
- [ ] compact labels have an immediately adjacent complete legend; no orphan or ID-only nodes
- [ ] high-degree neighbors receive expanded summaries
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
- **P2 — Focus + Map + Moves.** Give the Focus summary (substance), positional/ASCII Map (structure), and degree-scaled Moves (direction) defined above. Every directional use in all three parts follows [Semantic-direction enforcement](#semantic-direction-enforcement). None is sufficient alone: a map without summaries is a skeleton you can't read; summaries without a map lose the spatial relationships. Never relay raw command output. (Detailed as (a)/(b)/(c) below.)
- **P3 — Adaptive hierarchical quick-actions picker.** When the harness exposes a chooser affordance, a human is co-navigating, and Navigation help is open under Guided mode or temporary help, show up to one evidence-backed contextual concrete shortcut, a stable `Lenses…` row, a stable `All navigation actions…` row, and the final row is always `■ Arrive`; keep Lenses and Recenter / Peek / Scan / Arrive exactly one level away through those stable rows. Apply breadcrumbs, shorthand labels, effect markers, menu-stack transitions, and mode-aware compact-return rules from the picker contract. In Advanced mode do not open the picker automatically. On a surface without a chooser, use the equivalent labeled help only when mode requires it; an autonomous run picks the move per the goal while keeping all four action classes discoverable in prose. This applies wherever a positioned walk presents an onward decision, including after `peek` and a completed `teleport` landing.
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

##### Complete visible-node description

Every visible non-focus node in Focus, Map, or Moves MUST carry the complete identity-and-substance form `<id> — <readable title> — <body-derived central claim>`, with the claim scaled to that node's inbound degree under P4. This is one cross-seam rule for Orient, a full Peek or Lens return, Scan, Teleport landing, Back/Forward restoration, and a full `navigate` resume; no full re-entry path may degrade a previously descriptive view into a skeletal one. A compact unchanged transient return references the complete frame already visible instead of redrawing a skeleton. The rule also applies to candidate notes in a pre-landing Teleport chooser and to note nodes composed from JSON sources. IDs supplement identity and substance; they never replace either one.

Width pressure may replace map node text with compact labels only when the map has an immediately adjacent complete legend mapping every compact label to that node's ID, readable title, and body-derived central claim, still scaled to that node's inbound degree. The legend must be in the same view and beside the compact map—not deferred to another section, picker, turn, or command. Orphan labels and ID-only nodes are prohibited. A title without a claim is also skeletal. A compact reference in Focus or Moves is allowed only when that same complete legend is immediately adjacent; otherwise each occurrence uses the complete form.

Empty zones are exempt from node description because they contain no node, but they retain the complete semantic gloss: marker, zone name, local relationship meaning, and what the emptiness means for this focus. Region blobs that do not represent individual notes likewise need a semantic region gloss; any individual representative, bridge, match, or candidate note shown inside them remains subject to the complete node form.

Concrete quick actions obey the same rule: a note target needs its ID and readable target title plus the substantive body- or evidence-derived reason required by the picker contract. A generic role such as `supporting experiment` cannot stand in for the target's claim or result.

**Compliant descriptive node and compact-map fallback:**

```text
Moves
- 20250101120000-a1b2 — Replay-safe checkpointing — Failed replays restore the
  last durable boundary before retry, preventing partial state from becoming the
  new baseline. (↓6; hub summary continues with why this is load-bearing.)

Map
🟠 FOCUS ──▶ [N1]
Legend (immediately adjacent)
[N1] = 20250101120000-a1b2 — Replay-safe checkpointing — Failed replays restore
       the last durable boundary before retry; this recovery rule supports six
       inbound dependents.

Quick action
→ Recenter 🟢 BOTTOM — what builds on the focus: move to 20250101120000-a1b2 —
Replay-safe checkpointing (changes focus) because the body says failed replays restore the last durable boundary
before retry, directly advancing the safe-recovery goal.
Legend: → focus changes; ○ focus retained; ■ stops; ↗ explores beyond local. ◇ human consultation; retained focus and history.
```

**Noncompliant skeletal presentation:**

```text
Map: FOCUS -> [N1] -> 4302
Moves: 4302; supporting experiment
Quick action: Recenter BOTTOM to 4302 because supporting experiment
```

This is bad because `[N1]` is orphaned, `4302` is ID-only, neither node has a readable title and body-derived central claim, the direction omits its semantic triple, and `supporting experiment` names only a vague role rather than substantive evidence.

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
| leaf (↓0–1) | ID + readable title + type + one body-derived claim clause |
| connected (↓2–4) | ID + readable title + one full sentence on its body-derived claim |
| hub (↓5+, or the highest-degree node in view) | ID + readable title + 2–3 sentences: its body-derived claim *and* why it's load-bearing |

The body transport never truncates semantic content: it splits large bodies into lossless bounded segments, and the presentation gate waits for all pages before summarizing. Degree tells *you* how much to relay. With `--presentation-hints`, each topology text node carries a `relay budget:` line and each topology JSON node carries structured `summary_budget` metadata mirroring these tiers, so the policy remains visible while bodies travel separately. Two overrides: a high-degree *daily* or index note is a hub by connectivity, not substance—treat it as connected (one sentence) and say so; and a low-degree axiom whose body is clearly central gets promoted a tier—let the body's actual claim override the degree. Empty zones carry meaning and the full semantic triple: say, for example, `🔴 LEFT — what contests or questions the focus: empty; nothing contests or questions this focus.`

Keep `--depth 1` when zoning—only direct neighbors carry a zone, and the zoned text view omits unzoned nodes. Use topology alone only when you need shape without claims. When substance is required, use `nn graph bodies` with the identical traversal, retrieve every page under the same snapshot, reconstruct every body, and only then read and relay it. The deprecated `nn graph show --bodies` compatibility path is unbounded and is not a navigation source.
