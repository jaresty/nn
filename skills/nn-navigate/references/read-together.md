---
name: read-together
applies_when: "When entering, advancing, reversing, refreshing, inspecting context for, leaving, or returning to an interactive document reading session."
---

# Reference: Read Together

## Purpose and activation

Read Together is a human-driven `nn-navigate` mode for moving through a document while surfacing bounded notebook context. Activate it for `read together`, `:read <document>`, `:overview`, `:next`, `:previous`, `:section`, `:context`, `:deeper`, `:pin`, `:hide`, `:back-to-reading`, or `:finish-reading` when a document frame exists. These are conversational skill intents, not `nn` subcommands. The machine-readable primitive is `nn read-context`.

Read Together preserves three distinct evidence classes:

1. **Source passage** — exact text from the selected source and line range;
2. **Generated relevance** — bounded BM25 candidates that may help interpret the passage;
3. **Stored notebook truth** — note bodies, metadata, and typed directed links retrieved through their owning commands.

Never describe a generated match as a link, support relation, contradiction, provenance edge, or exhaustive context. A missing match never establishes notebook absence.

## Document frame

Retain this conversation-scoped state independently of graph navigation state:

```yaml
document_frame:
  source: path
  snapshot: sha256
  chunk: integer
  chunks: integer
  heading: string
  line_start: integer
  line_end: integer
  pinned_note_ids: []
  hidden_note_ids: []
  visited_evidence: []
  graph_excursion_origin: optional-complete-document-frame
```

Do not persist this state to notes, links, SQLite, Git, configuration, environment, or protocol notes. Compaction must preserve the complete frame or report it unknown. Never reconstruct a missing snapshot, cursor, pin, hidden set, or excursion origin from notebook content.

## Enter and passage acquisition

Enter with:

```bash
nn read-context <file> --chunk 1
```

The source must be a supported UTF-8 Markdown or text file. Read `snapshot`, `chunk`, `chunks`, `next_chunk`, heading, line range, passage, coverage, and matches. Reading windows default to 8 KiB: headings remain structural metadata, adjacent short sections pack together, and only oversized sections split internally. The command response is evidence input, not the human-facing presentation.

For every later chunk, preserve source and chunking options and pass the original snapshot:

```bash
nn read-context <file> --chunk <N> --snapshot <snapshot-from-chunk-1>
```

A stale or mismatched snapshot invalidates the document frame. Do not continue from partial or newly chunked source state: restart at chunk 1 and explain that the source changed.

`:next` selects `next_chunk`; `:previous` selects `previous_chunk`. At either boundary, retain state and state that the beginning or end has been reached. `:section "<heading>"` resolves only against headings already observed in the current session unless the implementation has retrieved a deterministic section index; never guess an unseen destination. `:context` refreshes generated relevance for the current passage. `:deeper` graph-expands explicitly selected or pinned note IDs; it does not increase the claim strength of BM25 matches.

`:overview` retrieves the complete bounded structural map. Start with:

```bash
nn read-context <file> --overview --chunk <current-chunk> --page 1
```

If `pages` is greater than one, retrieve every later page in order with the page-1 snapshot and identical source/chunking options. Do not present or resolve unseen headings until all pages arrive under one snapshot. Each returned item is `document-structure`, not generated relevance or a stored graph relationship. Render headings in source order with indentation from `level`, line and chunk ranges, and exactly one current marker. This observational action does not change the document frame or graph history. After a complete overview, `:section "<heading>"` may resolve against its retrieved headings using exact text or a unique case-insensitive fragment; ambiguity never guesses.

## Pins, hides, and note inspection

`:pin "<label>"` and `:hide "<label>"` resolve only current Context labels using the same full-label-or-unique-fragment discipline as Navigate presentation. Pins remain visible across passages but retain their original provenance as generated relevance or manually selected context. Hiding suppresses a candidate from presentation only; it does not alter retrieval, notebook truth, or future independent matches. Ambiguity and unavailable labels retain all state and show narrowed contextual help without guessing.

Peek at a note by retrieving its exact body and relevant stored topology under ordinary Navigate ownership while retaining the complete document frame. Peek never changes graph focus or the document cursor.

Following or Recenter-on-note starts an explicit graph excursion. Save the complete document frame as `graph_excursion_origin`, then apply ordinary graph movement/state rules. The passage is not a graph node and is never pushed into graph Back/Forward stacks. `:back-to-reading` restores the exact saved document frame, clears only the excursion marker, and rerenders Read Together; it does not fabricate a graph move.

## Presentation: Passage → Context → Map → Reading actions

A full human-facing Read Together view uses this stable order.

### 1. Passage

Show source identity, heading, `chunk/chunks`, line range, and exact passage text. Clearly distinguish exact source text from agent summary or commentary. If the passage is long, preserve it as a quoted or fenced block rather than paraphrasing it away.

### 2. Context

List bounded generated-relevance candidates with stable local labels such as `[N1]`, note ID, readable title, type/status, normalized score, match reason, and a bounded excerpt. Mark pinned candidates. State visibly:

> Dashed match lines are generated relevance, not stored links. This is bounded context, not an exhaustive notebook search.

Do not apply TOP/LEFT/RIGHT/BOTTOM semantics to the passage-to-note match.

### 3. Map

Draw a two-domain view:

```text
DOCUMENT                              NOTEBOOK CONTEXT
§ current passage  - - match - - >    [N1] candidate
                                      [N2] candidate

                                      [N1] ──typed-link──→ [N2]
```

Use dashed or explicitly labeled `match` connectors only for passage-to-note relevance. Use solid arrows only for verified stored typed relationships, with arrowheads beside stored targets. If stored topology has not been retrieved, say `Stored map not expanded` rather than inferring edges from similarity. When a selected note becomes graph focus, switch to the ordinary Focus + Map + Moves presentation and its zone palette.

### 4. Reading actions

Keep these action classes discoverable when structurally available:

- `Document overview — :overview` without moving;
- `Next passage — :next` and `Previous passage — :previous`;
- `Inspect note — :peek "<label>"` without moving;
- `Follow note — :recenter "<label>"` for an explicit graph excursion;
- `Refresh context — :context`;
- `Expand stored map — :deeper`;
- `Pin context — :pin "<label>"` and `Hide context — :hide "<label>"`;
- `Back to reading — :back-to-reading` during an excursion;
- `Integrate` only when visited evidence supports a concrete bounded proposal;
- `Finish reading — :finish-reading`.

Guided mode presents a compact adaptive picker; Advanced mode keeps it closed unless help is requested. Read Together actions do not replace ordinary Navigate help during a graph excursion.

## Finish and Integrate

`:finish-reading` stops advancing while retaining the final document frame for `read together` resumption. Report sections/chunks covered, pinned context, inspected notes, unresolved questions, and possible integrations. Distinguish exact source claims, notebook claims, and generated interpretations.

Read Together never mutates notebook truth automatically. Any capture, update, or link proposal fetches `integrate`, names its source passage and note evidence, preserves generated-versus-stored boundaries, and writes only after explicit approval. Similarity alone is insufficient evidence for a link.

## Return and compaction

Transient note inspection returns to the current Read Together frame without rereading an unchanged passage. Full rerender is required after passage movement, source refresh, pin/hide changes that alter the visible frame, returning from a graph excursion, notebook mutation, stale state, or explicit refresh.

A compaction handoff carries the entire document frame, interaction mode, current Read Together menu position, and any complete graph excursion frame separately. If either frame is incomplete, report that part unknown; never merge or reconstruct the two state domains.
