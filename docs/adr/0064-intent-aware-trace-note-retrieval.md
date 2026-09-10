# ADR-0064: Intent-aware note retrieval for source tracing

**Status:** Accepted

## Context

`nn trace` builds a syntax-aware call graph and annotates resolved nodes with related notebook notes. `nn grep --trace` combines source matches with inline structural traces and match-level note retrieval.

Current retrieval uses only symbol source or grep context as its query. The shared `TypedCorpusScorer` ranks title, body, tags, and typed annotation channels with BM25 and combines field ranks using reciprocal rank fusion (RRF), as established by ADR-0037. Source-only retrieval therefore answers which notes resemble the implementation, but not necessarily which notes help with the investigation that motivated the trace.

This caused a trace of `resolveTranscriptSession` to select an unrelated note sharing `String` vocabulary while missing an existing note about generic-name trace misresolution. Separately, commit `54d0c02` made `nn grep --trace` 2.42× faster by avoiding per-node BM25 annotations that its inline renderer discarded. Intent-aware retrieval must preserve that optimization.

## Decision

### Optional intent flag

Add optional `--intent <text>` flags to both tracing surfaces:

```text
nn trace ... --intent "investigate ambiguous call resolution"
nn grep ... --trace --intent "investigate ambiguous call resolution"
```

Intent affects note retrieval only. It must not alter grep matches, symbol resolution, graph nodes or edges, traversal, ambiguity classification, or unresolved-call classification. Omission preserves source-only behavior and never prompts. `nn grep --intent` without `--trace` fails clearly because the initial contract is trace-specific.

Agent guidance should require intent for purposeful investigations while allowing omission for structural exploration and compatibility.

### Hierarchical RRF

Preserve the existing field-level RRF independently for each query. Run the existing scorer for these query channels:

| Channel | Input | Weight |
|---|---|---:|
| `source` | Symbol body or grep context | 1 |
| `intent` | Operation identity (`nn trace` or `nn grep trace`) plus exact nonblank `--intent` value | 5 |
| `diagnostic` | Bounded generated trace conditions | 1 |

The intent channel is absent when omitted. The diagnostic channel is absent when no qualifying condition occurs. The existing scorer returns the sum of field-level RRF contributions for one query. Multiply those contributions by the query weight and add them directly:

```text
Q(note) = Σ(query q) weight(q) × Σ(field f) fieldWeight(f) / (K + rank(q, f, note))
```

This is one RRF accumulator over `(query, field)` channels. Query-result ranks may be retained as explanatory provenance, but they are never passed through a second reciprocal-rank transform. A missing note contributes zero for that query. Existing candidate order remains the final tie-break.

The fixed operation identity is included in the intent query because an unscoped investigation phrase ranked an unrelated domain note above an existing note about the tracing operation. The intent multiplier is 5: live direct-trace and grep-trace checks showed that 2 still allowed accumulated source and diagnostic fields to displace the intent-rank-2 limitation note, while 5 retained that note on both surfaces. Field-level evidence, rather than a second compressed rank layer, determines the final difference.

### Bounded diagnostics

Generate diagnostic query text only from conditions present in the trace and from a fixed vocabulary:

- unresolved call
- ambiguous receiver
- multiple name-matched candidates
- external definition unavailable
- cycle detected

Do not serialize the graph into the diagnostic query.

### Provenance

Intent-aware results disclose contributing query channels and their ranks. Human output may use a compact suffix such as:

```text
[intent #1, diagnostic #2, source #8]
```

JSON adds ranking provenance fields without changing graph identity. Raw intent text is retained once at command-result scope rather than repeated on every note.

### Enrichment boundary

Build the graph before enriching it. Retrieval failures must not invalidate an otherwise valid graph.

`nn grep --trace` continues calling `trace.Trace` without a node annotator. It derives bounded diagnostics from the result and ranks notes once per grep match from source context, optional intent, and diagnostics. This preserves the `54d0c02` optimization.

Direct `nn trace` may attach notes to nodes, but enrichment occurs after traversal so diagnostics are available and retrieval cannot influence graph construction.

## Consequences

### Positive

- Investigation-specific notes can win without source vocabulary overlap.
- Source relevance remains independently represented.
- Existing field IDF, typed annotation channels, tokenization caches, and RRF semantics are reused.
- Graph authority remains separate from notebook enrichment.
- Ranking decisions become inspectable.
- Existing invocations remain valid and non-interactive.
- The optimized `grep --trace` execution boundary remains intact.

### Negative

- Ranking has two explicit RRF layers.
- Query weights and provenance become compatibility surfaces.
- Direct trace enrichment requires post-traversal refactoring.
- Intent can bias retrieval when it is vague or leading.

## Implementation plan

1. Add a shared named-query ranking primitive over `preparedCorpus.rankedByQuery`, returning fused scores and per-channel rank provenance.
2. Reuse or expose the established RRF constant and preserve deterministic candidate-order ties.
3. Add optional `--intent` flags to `nn trace` and `nn grep --trace`; reject grep intent without trace.
4. Separate direct graph construction from post-traversal node enrichment.
5. Derive fixed-vocabulary diagnostics from existing graph records.
6. Keep `grep --trace` node annotation disabled and perform one match-level fused retrieval.
7. Add concise human provenance and additive JSON provenance.
8. Update the virtual CLI reference and `skills/nn-guide/SKILL.md` without naming or privileging another tracer.
9. Add deterministic ranking, CLI compatibility, graph-identity, grep-match, provenance, and performance tests.

## Validation

- `go test ./...`
- `git diff --check`
- Graph nodes and edges must be identical with and without intent.
- Grep matches and source windows must be identical with and without intent.
- Omitted intent must preserve established output unless additive provenance is explicitly enabled.
- A trace-limitation note must outrank a source-only lexical false positive under a matching intent.
- The representative `grep --trace` benchmark must not materially regress from revision `54d0c02`.

## Current state and next action

Revision `54d0c02` contains neutral tracing guidance and the discarded-annotation performance fix. Existing field-level and typed-annotation RRF infrastructure is available through `preparedCorpus` and `TypedCorpusScorer`. Query-level fusion, CLI flags, post-trace enrichment, and provenance are not yet implemented.

The next action is to add failing deterministic tests for weighted query-level RRF, then implement the shared fusion primitive before changing command behavior.
