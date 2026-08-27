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

Guided is the default. Its Navigation help remains visible after each completed action and presents human labels beside canonical shorthand so use teaches the terse grammar. Pending Ask and unanswered Quiz states are not completed actions.

Advanced keeps Navigation help closed. `:help` opens a complete textual catalog of every canonical shorthand temporarily without changing mode; completing an action or dismissing the help closes it. The catalog is distinct from contextual adaptive pickers, so picker hierarchy and row limits do not truncate it. Contextual menus still apply their existing hierarchy and limits. `:guided` and `:advanced` switch mode without changing graph state.

Deterministic transient returns now restore a semantic menu position independently of rendering it. Guided reopens that menu and keeps help visible. Advanced retains the same return position but does not render help. Focus-changing moves and `navigate` still Orient and reset the semantic menu to Quick actions while preserving interaction mode. Arrive preserves mode and focus; Guided shows help after the report and Advanced shows only the resume affordance.

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
