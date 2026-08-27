# ADR-0038: Conversational navigation DSL

**Status:** Accepted

## Context

`nn-navigate` already defines a human-driven, conversation-scoped graph walk, but its human input contract is distributed across natural-language verbs, adaptive picker labels, and transient-return rules. The picker is useful for learning and discovery, yet an experienced user needs terse deterministic input without turning navigation into a new CLI surface. Unspecified target nicknames would be unsafe because they can outlive the evidence and menu context that grounded them.

The navigation mode, complete frames, history, bookmarks, Ask snapshots, and menu stack are conversation state. Files remain notebook truth and the SQLite index remains cache. Adding Cobra commands, a runtime DSL parser, configuration, environment state, or persisted aliases would violate that boundary. Skill retrieval must also remain byte-for-byte deterministic rather than templated by conversation mode.

## Decision

### Static skill-level grammar

`nn-navigate` owns a static conversational DSL. Its canonical colon-prefixed intents are:

```text
:help
:guided
:advanced
:look
:where
:orient
:recenter "<label>"
:peek "<label>"
:show
:explain
:gaps
:scan local
:scan global
:analogize
:find-analog
:visualize
:quiz
:ask
:back
:forward
:bookmark "<name>"
:goto "<bookmark>"
:arrive
```

These are skill-level intents interpreted by the agent, not `nn` subcommands, flags, shell syntax, or a runtime parser. Existing natural conversational intents remain valid. The direct conversational launch `navigate advanced ...` selects Advanced mode before interpreting the remaining query or intent.

`:look` renders the complete current Focus + Map + Moves view without mutating navigation state. `:where` reports compact focus, traversal/filter, immediate history, and bookmark state. `:orient` performs fresh topology/body retrieval and recomputes the positioned interpretation when evidence may be stale. Unknown colon-prefixed intents never become searches or moves; they retain state, report the unknown shorthand, and teach the location/help commands without guessing.

### Guided and Advanced presentation

Guided is the default. Its Navigation help remains visible while another navigation choice is naturally pending and presents human labels beside canonical shorthand so use teaches the terse grammar. A completed action selected from a Guided picker always reopens its invoking menu. Only a completed action requested directly in conversation may instead use a **quiet return**: report completion, retain the semantic menu position and complete frame, and show the compact `navigate`/`:help` affordance without immediately forcing another chooser. Pending Ask and unanswered Quiz states are not completed actions.

Every submenu visibly displays adjacent `Esc: back to <parent-menu-name>` guidance. The hint is not an option row: canonical submenu rows remain unchanged and every picker keeps its four-row cap. This makes the existing menu-stack pop behavior discoverable without displacing a lens or adding a fifth Back action.

### Contextual notebook integration

Navigation may promote **`✚ Integrate — <bounded change>`** in the single contextual shortcut slot when complete visited evidence supports a concrete edit to notebook truth. Integrate is not a permanent picker row, does not expand `All navigation actions…`, and adds no `:integrate` parser or `nn integrate` command. Direct conversational requests such as “integrate this,” “link these,” “update that note,” or “delete the obsolete note” dispatch to the same owner.

Integrate covers every bounded notebook edit: note creation, body or metadata updates, title/tag/type/status changes, deletion, relationship addition/removal/retyping/annotation revision, supported mixed changes, and no-op. It first presents a non-mutating changeset with evidence, generated-versus-sourced boundaries, affected IDs, before/after intent, canonical edge direction/type/annotation, endpoint support, authority limits, and deletion consequences. Nothing writes without explicit human approval.

Application delegates to existing `nn` workflows and preserves their concurrency, confirmation, typed-edge, deletion, status, and formal-representation safeguards. When an existing workflow supports the approved semantic change atomically, Integrate uses it. Otherwise it must disclose the limitation and require scope reduction or a separately implemented atomic capability; it never represents multiple commands or commits as one operation. Successful mutation retains focus and history, invalidates the cached frame, reruns Orient before presentation, and uses the effect legend **`✚ notebook truth changes; focus and history retained; Orient refreshes`**.

Advanced keeps Navigation help closed. `:help` opens a complete textual catalog of every canonical shorthand temporarily without changing mode; completing an action or dismissing the help closes it. The catalog is distinct from contextual adaptive pickers, so picker hierarchy and row limits do not truncate it. Contextual menus still apply their existing hierarchy and limits. `:guided` and `:advanced` switch mode without changing graph state.

Deterministic transient returns now restore a semantic menu position independently of rendering it. Guided reopens that menu and keeps help visible. Advanced retains the same return position but does not render help. Focus-changing moves and `navigate` still Orient and reset the semantic menu to Quick actions while preserving interaction mode. **Arrive is terminal in both modes: it closes any chooser, preserves mode and the complete retained frame, and shows only the explicit `navigate` resume affordance.** Navigation help reopens only after the human explicitly resumes or requests another action.

### Compact spatial maps with complete adjacent keys

Compact labels are the default map representation for every full positioned Navigation view. This keeps the Map spatially legible while preserving the evidence pressure that previously forced full descriptions into every occurrence. Labels are deterministic within the view by semantic zone and local order, such as `[T1]`, `[L1]`, `[R1]`, and `[B1]`.

Every compact map has one immediately adjacent complete evidence index. Each label maps exactly once to the note ID, readable title, note type, degree, and body-derived central claim. No label may be orphaned, duplicated, ID-only, or title-only. Summary prominence is **view-local importance**, not degree: at most two `★` decision-shaping nodes receive expanded treatment, `◆` marks decision-supporting evidence, and `·` marks orienting context. Degree remains visible connectivity and is only a final tie-breaker. A full inline map is allowed only for a one- or two-neighbor view that explicitly states why inline form is clearer; each inline node must preserve the same identity, evidence, degree, and importance semantics.

Spatial placement remains literal and invariant: TOP is above focus, LEFT is left, RIGHT is right, and BOTTOM is below. Every empty zone visibly occupies its slot as `[∅]`; prose-only emptiness is insufficient. Each compact label keeps its zone emoji and importance marker directly beside it. TOP and BOTTOM use one post-classification vertical renderer: zone classification and semantic gloss remain distinct, while placement never changes stored source, canonical type, target, or target-adjacent arrow. One edge uses an attached typed stem; one reciprocal pair uses a separately typed reciprocal stem; every other ambiguous edge set uses a centered borderless attached rail with one endpoint-complete unit per stored edge and unit-local narrow wrapping. Direct focus relationships remain in geometry, while neighbor-to-neighbor stored relationships move to an immediately adjacent complete secondary-edge ledger. Moves reference labels rather than repeating the evidence index; they retain the complete semantic triple and action effect, then add only the decision-relevant reason for acting. Concrete quick actions retain target ID, readable title, effect, and evidence-derived reason because they can be selected independently of the map.

`Find an analog` remains a focus-retaining, globally retrieving interpretive action, but its menu home moves beside the other Peek-family inspections. Peek therefore contains Show verbatim, Explain in depth, and Find an analog; Lenses contains Analogize, Find gaps, Visualize, and Quiz. Both menus fit the four-row chooser limit without pagination.

This changes presentation structure, not evidence acquisition, graph semantics, focus/history behavior, runtime parsing, or notebook persistence.

### Context-grounded targets

`:recenter "<label>"` and `:peek "<label>"` resolve only against human labels in the current complete-frame menu/help context. A complete label or unique case-insensitive fragment may resolve. The agent does not generate or retain note aliases, widen the candidates, consult hidden stale menus, or guess from likely intent.

Ambiguity performs no action and opens narrowed contextual help containing only matching labels, including temporarily in Advanced. An unavailable target performs no search or substitution; it explains the unavailability and shows only applicable contextual help. Bookmark lookup is not label-fragment resolution: bookmark names remain exact and case-sensitive.

### Conversation-scoped mode state

Interaction mode is UI state separate from navigation frames. Preserve it through moves, Back/Forward, bookmarks, lenses, Ask preparation and handoffs, cancellation, and compaction. A compaction handoff carries mode with current menu and ordered menu stack. Missing mode alone defaults to Guided; missing graph state remains unknown and is never reconstructed.

Mode, menus, aliases, and navigation state are never persisted to note bodies, frontmatter, links, SQLite, protocol notes, Git, configuration, or environment. `nn skills get nn-navigate` and lazy-reference retrieval remain deterministic and untemplated; mode changes presentation, not retrieved skill bytes.

### Ownership and dispatch

The compact `nn-navigate` core owns activation, canonical grammar, state invariants, and lazy-reference dispatch. `presentation` owns mode-aware help, contextual labels, ambiguity, and transient rendering. `movement` owns launch/resume/Arrive behavior. `state` owns conversation-only preservation and compaction defaults. `ask` owns exact mode preservation while consultation is suspended.

`nn-guide` and the generated global CLI protocol contain summary dispatch only. They point to `nn-navigate` and state that the DSL is skill-level rather than duplicating grammar, target resolution, menu behavior, or owner-specific return rules.

No Cobra command, runtime parser, configuration key, persistence layer, or generated alias registry is added.

## Consequences

- New and occasional users get persistent, self-teaching Guided navigation.
- Experienced users can use concise Advanced navigation without repeated menus.
- Natural language and canonical shorthand share one semantic action contract.
- Target selection remains auditable because it is bounded to current grounded labels; ambiguity and unavailability cannot silently relocate focus.
- Menu return semantics remain deterministic while rendering differs coherently by interaction mode.
- Ask, lenses, moves, and compaction retain one conversation mode without contaminating notebook truth.
- Missing mode has a safe presentation default, while missing graph state still fails closed.
- The implementation is documentation/static-skill-only; runtime CLI compatibility and deterministic skill retrieval are unchanged.
