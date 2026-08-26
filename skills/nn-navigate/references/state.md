---
name: state
applies_when: "When focus/history/bookmarks/menu state may change, be restored, fail, survive Ask, or cross a compaction boundary."
---

# Reference: State

## Conversational navigation history and bookmarks

These are skill-level conversational moves and conversation-scoped state, not `nn` subcommands or persisted notebook data. A **navigation frame** is the retained focus plus its active traversal context and filters: preserve enough information to reproduce the positioned view, including the walk goal/query and any active direction, link-type, status, representation, depth, route, or impact context. Restoring only a note ID is not sufficient when filters or traversal context were active.

Maintain a current frame, a Back stack, a Forward stack, and bookmarks. Separately retain `interaction_mode` (`guided` or `advanced`), the current menu, and its ordered stack as conversation-scoped UI state. Interaction mode is not part of a navigation frame: moves, Back/Forward restoration, lenses, and bookmark travel preserve the current mode rather than restoring an older one.

- After a successful Teleport, Visit, Recenter, or Go to, push the prior frame onto Back and clear Forward. `Visit` here means an independently requested move when that vocabulary is already present; it is never the prohibited second confirmation after a completed Teleport landing. If there is no prior/current focus, an initial landing cannot push a frame.
- **Back** moves the current frame onto Forward and restores the latest Back frame. **Forward** moves the current frame onto Back and restores the latest Forward frame. After either restoration, fetch `movement` and `presentation`, rerun Orient, and present Focus + Map + Moves under the Presentation discipline.
- **Bookmark <name>** stores the complete current frame under a case-sensitive name. Creating a new name needs no confirmation; an existing exact-case name requires explicit confirmation before replacing it. A declined replacement changes nothing.
- **Go to <name>** restores its saved frame as a Teleport landing. It therefore follows the successful-move rule: push the prior current frame onto Back, clear Forward, rerun Orient, and present Focus + Map + Moves.
- **Where am I?** reports the current focus, active traversal context and filters, immediate Back and Forward destinations (if any), and bookmark names. It does not mutate navigation state.

Failed or no-op operations never mutate history or bookmarks. If the Back or Forward stack is empty, say, for example, “Back stack is empty” or “Forward stack is empty,” retain the current frame, and do not rerun a fictitious landing. For an unknown bookmark, say the name was not found, list the available case-sensitive bookmark names, and retain all state. A move that resolves to the complete current frame is a no-op.

This state lasts only for the conversation. Any compaction handoff MUST include the full current frame, Back and Forward stacks, and every bookmark, preserving stack order and each bookmark's case-sensitive name and complete saved frame. The compaction handoff MUST include a separate menu UI state containing interaction mode, the current menu, and ordered menu stack. Preserve interaction mode through moves, lenses, Ask, and compaction. Menu UI state is conversation-scoped UI state, not notebook state: never persist interaction mode, menu position, or navigation state into notes, frontmatter, links, SQLite, protocol notes, Git, configuration, or environment.

Missing interaction mode defaults to Guided; this is the only reconstructive default and does not recover any graph state. If menu UI state is absent after compaction, it is unknown with respect to menu position: reset that position only through an explicit Orient, focus-changing move, or `navigate`, rather than inventing a prior submenu. If navigation state is missing or incomplete after compaction, state is unknown: never invent history, filters, destinations, or bookmarks; report that navigation state cannot be recovered and ask the human to establish a new landing or restate it. Skill retrieval remains deterministic and untemplated: restored mode changes presentation only and is never passed into skill/reference retrieval.
