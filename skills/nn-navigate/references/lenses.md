---
name: lenses
applies_when: "When showing verbatim, explaining in depth, analogizing, finding an analog, finding gaps, visualizing, or quizzing without changing focus."
---

# Reference: Lenses

## Moves versus lenses

Moves operate on the positioned walk; lenses interpret evidence already visited. Focus + Map + Moves remains mandatory whenever the walk presents an onward decision. Find an analog and the other lenses may be reached through Lenses or promoted as concrete top-level shortcuts when they satisfy the evidence criteria. They do not become focus-changing moves: Lens actions never mutate focus or navigation history, and Scan retains focus.

**Find an analog** is human-facing Lens intent implemented internally as a **Scan retrieval move across another region**. It seeks the same **relational structure, not lexical similarity**. Candidate generation may use `nn clusters`, `nn list --search`, `nn list --similar`, and graph neighborhoods, but those retrieval signals identify possibilities rather than establish an analogy. For every serious candidate, compare relational shape and present:

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

### Find gaps

Find gaps interprets what may be missing around the retained exact focus without changing position. It is a Lens over existing evidence primitives, not a new search or traversal command. Retrieve topology first:

```
nn graph show --focus <id> --depth <n> --direction both --format json
```

Then retrieve the identically filtered bodies with `nn graph bodies --focus <id> --depth <n> --direction both`, starting at page 1 and consuming every page under one validated snapshot before making any body-derived claim. Finally run `nn list --similar <id>` to surface lexically related but unlinked candidates. Do not add or invoke `nn gap --focus`; `nn gap <topic>` remains topic-oriented, while this Lens composes exact-focus graph evidence.

Present four explicitly separated sections:

1. **Observed structural gaps** — absences directly established by the bounded topology, such as no edge of a relevant type or one-sided development. State the inspected depth and filters; absence inside the bound is not global absence.
2. **Body-derived gaps** — missing evidence, unanswered questions, unresolved contradictions, or shallow treatment inferred only after all selected bodies reconstruct exactly. Cite the note IDs and distinguish quotation from interpretation.
3. **Candidate missing links** — `nn list --similar <id>` candidates for which visited evidence suggests a plausible absent relationship. Label every candidate **suggested, non-stored**; lexical similarity alone never establishes an edge or its type.
4. **Bounded unknowns** — conclusions the retrieved topology and bodies cannot establish, including questions requiring evidence outside the selected neighborhood.

Find gaps does not mutate focus, history, notes, or links. A later Recenter or typed link proposal is a separate action under its normal contract. Before returning, follow the same transient Lens return policy as Analogize or Visualize.

### Visualize

Visualize **spatializes meaning** in the **retained visited evidence**. Distinguish the **stored Map/graph**—visited notes and stored edges—from any **derived layout** used to explain them. Label derived arrows and groupings as **non-stored**; never imply that proximity, containment, or an explanatory arrow is a notebook edge.

Use ASCII on any surface, or **Pi-supported Mermaid** when it will render. Allowed Mermaid families are `graph`, `flowchart`, `stateDiagram`, `stateDiagram-v2`, `classDiagram`, `erDiagram`, and `sequenceDiagram`; exclude `pie`, `quadrantChart`, and `mindmap`.

### Quiz

Quiz is **source-grounded** and tests a bounded set of **consequential concepts**, not trivia. State an **explicit purpose and stopping condition**, normally bounded to 1–3 concepts. Ask **one question, then wait for a human Predict turn before the reveal**; accept safe `pass`, `skip`, or `I don't know` responses without penalty or pressure. An unanswered Quiz question suspends the picker until the human answers, passes, skips, says `I don't know`, or says `navigate`. Every active Quiz prompt with an unanswered question MUST offer the escape in this wording: `Answer, pass/skip/I don't know, or say navigate to leave the Quiz lens and return to navigation.`

After the Predict turn, compare the prediction with the source-grounded answer, cite the relevant note IDs and stored edge evidence, and explain any misconception and why it matters. Do not invent questions beyond what the retained sources can answer, invent facts or relationships, or test derived-framework recall as though an agent-generated analogy or layout were notebook evidence.

Show verbatim, Explain in depth, Analogize, Find an analog, Find gaps, and Visualize are transient actions. Before returning, fetch `presentation` and follow its invoking-menu and Transient rendering policy: when the complete frame and notebook remain unchanged, Guided mode must show the compact breadcrumb, say focus is unchanged, and reopen Peek, Lenses, Scan, or Quick actions as specified rather than redrawing the frame; Advanced mode retains that semantic return position but keeps Navigation help closed. A completed Quiz follows the same return after a reveal, pass, skip, or `I don't know`, with rendering controlled by the retained interaction mode. While its question is unanswered, do not render another picker. `navigate` remains an interruption and early escape: abort the item without grading or revealing, rerun Orient, reset to Quick actions, and use the rendering policy.

Lens findings may inform a later explicit Recenter or link suggestion, but the lens itself does not mutate focus, history, notes, or links. Any later movement or mutation must be separately proposed and executed under its normal contract.
