# ADR-0070: Read Together keeps document position separate from notebook truth

**Status:** Accepted

## Context

`nn read` can print selected lines and append BM25-related notes, while `nn-navigate` can present a graph-positioned Focus + Map + Moves view. Neither supports an interactive reading session that advances through a document, retains a stable cursor, and lets a human inspect relevant notebook context without confusing lexical relevance with a stored graph relationship.

A document passage is not a notebook note. BM25 similarity is generated retrieval evidence, not an edge, and a bounded candidate set cannot establish exhaustive absence. Any reading workflow must therefore preserve three distinct things: source text and cursor position, generated notebook relevance, and stored note topology.

## Decision

### Add a bounded read-context transport

Add a JSON-only command:

```text
nn read-context <file> [--chunk N] [--snapshot SHA256]
                       [--chunk-bytes N] [--matches N]
```

The MVP accepts UTF-8 `.md`, `.markdown`, and `.txt` files. It splits source text deterministically into heading-aware passages, preserving source order and line ranges. Oversized sections split at paragraph or line boundaries, with UTF-8-safe fallback segmentation. `--chunk` is one-based and defaults to 1. `--chunk-bytes` controls the target source size and is part of the snapshot request; it has a bounded valid range. The command returns exactly one passage.

The compact JSON response contains:

- a SHA-256 snapshot binding the exact source bytes, format version, and chunking options;
- source, chunk, total chunks, previous/next chunk, heading, and line range;
- the exact passage text;
- up to `--matches` ranked notebook candidates with ID, title, type, status, score, match reason, and bounded excerpt;
- an explicit `generated-relevance` relationship label and non-exhaustive coverage statement.

Chunk 1 does not require a supplied snapshot. Every later chunk requires the snapshot returned by chunk 1. A supplied token on any chunk must match. Source mutation, changed chunking options, or another source rejects continuation. Every encoded response, including its final newline, is at most 48,000 bytes.

The command is read-only. It does not persist a cursor, reading history, pins, generated matches, notes, or links.

### Add Read Together as a Navigate mode

Read Together is an `nn-navigate` skill workflow, not an `nn read-together` runtime command. It owns conversation-scoped document state:

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
  visited_evidence: []
```

The document frame is parallel to graph navigation state. Advancing, reversing, jumping to a section, pinning, hiding, or refreshing document context does not mutate graph focus/history. Peeking at a relevant note retains the document frame. Explicitly following a note creates a graph excursion under the existing movement/state rules; `Back to reading` restores the exact document frame without pretending that the passage is a graph node.

The canonical conversational intents are skill-level actions: `:read <document>`, `:next`, `:previous`, `:section "<heading>"`, `:context`, `:deeper`, `:pin "<label>"`, `:hide "<label>"`, `:back-to-reading`, and `:finish-reading`. Natural-language equivalents remain valid. They are not Cobra commands or a runtime parser.

### Present Passage + Context + Map + Reading actions

A full Read Together view uses this order:

1. **Passage** — source identity, heading, chunk progress, line range, and exact text;
2. **Context** — bounded generated-relevance candidates with reasons and explicit non-exhaustive labeling;
3. **Map** — stored topology only among selected notebook notes, with typed directed edges visually distinct from generated match lines;
4. **Reading actions** — next/previous, inspect/follow a note, pin/hide context, deeper graph context, Integrate, or finish.

The document passage must never occupy a semantic TOP/LEFT/RIGHT/BOTTOM graph zone. Generated relevance uses a dashed or otherwise explicitly labeled connector. Solid typed arrows are reserved for stored notebook links. Selecting a note may open the ordinary Focus + Map + Moves presentation without losing the document cursor.

Read Together never creates a note or link automatically. Any proposed notebook mutation dispatches through Navigate Integrate, preserves sourced-versus-generated boundaries, and requires explicit approval.

## Consequences

- A human can advance through a document while retaining relevant notebook context.
- Source position, lexical relevance, and stored graph truth remain distinguishable.
- Reading sessions survive ordinary conversational turns but are not persisted into notebook files, the index, configuration, or environment.
- Initial format support is intentionally narrow; PDF, DOCX, URLs, and durable resumability can later build on `nn media` rather than weakening the MVP contract.
- Callers must retain the source snapshot and pass it when advancing beyond chunk 1.
- Bounded matches improve orientation but cannot establish that omitted notebook context is irrelevant or absent.
