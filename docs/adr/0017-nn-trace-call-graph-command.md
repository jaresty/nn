# ADR-0017: nn trace — Syntax-Aware Call Graph Command

## Status

Proposed

## Context

LLM agents working in a codebase need on-demand contextual discovery of how symbols
relate across files. `nn grep` surfaces note context around text matches but does not
traverse call edges. A dedicated `nn trace` command can walk a call graph from one or
more entry-point symbols, annotate each resolved node with related nn notes via BM25,
and emit either a human-readable tree or a JSON graph for downstream tooling.

A spike (`spike/asttrace/`) validated the core approach using
[gotreesitter](https://github.com/odvcencio/gotreesitter) (pure Go, 206 grammars, no
CGO). The spike demonstrated: language detection via `grammars.DetectLanguage`,
definition extraction via `gotreesitter.ExtractDefinitionSpans`, and call extraction
via `gotreesitter.ExtractCalls`. Cross-file resolution requires a pre-built name→[]defSite
index; cycle guards use a `file:name` key.

## Decision

Add `nn trace` as a new top-level command. It is **not** an extension of `nn grep`
because it has a fundamentally different latency profile (index phase + DFS vs. per-file
BM25 scan).

### Command signature

```
nn trace <root-dir> --symbol <name> [--symbol <name> ...] [--depth N] [--json] [--show-unresolved]
```

- `<root-dir>`: directory to index (required positional)
- `--symbol`: entry-point symbol name; repeatable; merged into a single graph
- `--depth`: DFS depth limit (default 3)
- `--json`: emit JSON graph instead of tree
- `--show-unresolved`: include unresolved (stdlib/external) leaves in human output (default off; always included in JSON with `resolved: false`)

### Index phase

Walk `<root-dir>`, skip `.git`, `vendor`, `node_modules`. For each file where
`grammars.DetectLanguage` returns non-nil, parse with `gotreesitter.NewParser` and
call `gotreesitter.ExtractDefinitionSpans`. Build `name → []*defSite` map. Index is
built once per invocation (no caching in v1).

### Entry resolution

Match each `--symbol` value against the index by exact name (case-sensitive). When a
name resolves to N > 1 definitions, annotate as `[N candidates]` and expand all
branches. No definition is silently dropped.

### DFS traversal

For each resolved entry point, extract calls within the definition's byte range via
`gotreesitter.ExtractCalls`. Resolve each call name against the index. Recurse up to
`--depth`. Cycle guard: skip a node whose `file:name` key has already been expanded in
this DFS path (mark as `[already expanded]` in output).

### BM25 annotation

For each resolved node in the graph, query the nn SQLite index with the symbol name and
surrounding source context (same mechanism as `nn grep`). Attach matching note IDs and
titles to the node. In JSON output these appear as a `nn_notes` array; in human output
as indented `note:` lines below the symbol.

### Human-readable output (default)

```
AddLink (function) [internal/backend/gitlocal/notes.go:42]
  note: gitlocal RMW lock pattern (20260630…)
  → acquireLock (function) [internal/backend/gitlocal/lock.go:18]
  → json.Marshal (function) [unresolved]    ← only with --show-unresolved
```

### JSON output (`--json`)

```json
{
  "nodes": [
    { "id": "internal/backend/gitlocal/notes.go:AddLink", "name": "AddLink",
      "kind": "function", "file": "...", "line": 42,
      "resolved": true, "nn_notes": [{"id": "...", "title": "..."}] }
  ],
  "edges": [
    { "from": "…:AddLink", "to": "…:acquireLock", "resolved": true }
  ]
}
```

## Consequences

**Positive**
- LLM agents get structured call-graph output with nn note context in a single command.
- Pure Go (no CGO), cross-compiles cleanly alongside the rest of nn.
- Multi-symbol merged graph enables tracing interaction paths between two symbols.

**Negative / risks**
- Index phase is O(files × parse time); no caching in v1. Large repos (>10k files) may
  be slow.
- Name-only resolution means methods on different receivers share a graph node when names
  collide. Receiver-qualified resolution is deferred to v2.
- gotreesitter's `ExtractCalls` populates `Receiver` but v1 does not use it for filtering
  (verified as a known gap in the spike).

## Deferred

- Index caching (file-mtime keyed, stored in `~/.config/nn/trace-cache/`)
- `--callers` flag for reverse graph (who calls this symbol?)
- Receiver-qualified resolution to disambiguate `Read` on `http.Response` vs `os.File`
- Incremental re-index on file change

## Falsifying scenarios

- Two methods named `Read` on different receivers → both shown as `[2 candidates]`, neither silently dropped.
- `--symbol A --symbol B` where A calls B → B appears once in the graph, not duplicated.
- `--json` on a 50-file repo → valid JSON, `nn_notes` array present on resolved nodes.
- `--depth 1 --show-unresolved` off → no stdlib leaves, no depth-2 nodes in output.
