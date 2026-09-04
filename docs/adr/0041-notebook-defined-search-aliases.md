# 0041. Notebook-defined search aliases (acronym expansion)

## Status

Proposed

## Context

nn's search and similarity ranking use a hand-rolled BM25 tokenizer
(`internal/note/bm25.go`, `tokenize()`) that lowercases, splits on
non-alphanumeric runes, drops single-character tokens, and applies Snowball
English stemming (`english.Stem`). This is already at the ceiling of what pure
lexical BM25 can do morphologically.

An empirically confirmed residual gap remains: **abbreviation/acronym misses**.
`nn list --search "TDD"` and `nn list --search "test driven development"` return
disjoint result sets, because `tdd` shares no morpheme with `test`, `driven`, or
`development`, so stemming cannot bridge them. This is a *semantic* gap, not a
morphological one.

Options considered and rejected:

- **WordNet libraries** (`fluhus/gostuff`, `wnram`): maintained, but research
  confirms WordNet lacks technical acronyms while expanding general terms
  (`test` -> `trial`), adding noise and missing the actual targets. Net negative
  for a code-notebook corpus.
- **Bleve** (`SynonymIndex`): would host field-weighted RRF and per-field BM25,
  but has no concept of the note graph. Link-type-weighted inbound annotations,
  1-hop score propagation, and status multipliers would become external pre/post
  processing, and it introduces a second persistent index — violating the
  "files are truth, index is cache" principle for a single feature.
- **External acronym datasets** (Peruma ICSME 2019, AMAP): identifier-focused or
  static; none contain project-specific coinages (`atomic`, `ground`, `falsify`,
  `craft pack`). A comprehensive generic list *hurts* precision.
- **Hardcoded Go map / `~/.config` file**: introduces a second source of truth
  outside the notebook, subject to drift, outside Git and the "files are truth"
  guarantee.

## Decision

**Let the notebook define its own acronyms.** A note declares its abbreviations
via a frontmatter `aliases` field:

```yaml
---
id: 20260101000000-0000
title: Test-driven development
type: permanent
aliases: [TDD]
---
```

At index build, all `aliases` across the corpus are collected into an expansion
table (`tdd -> [test, driven, development]`, where the expansion is the tokenized
title of the declaring note). `tokenize()` applies this table **before** stemming,
so expansions are themselves stemmed, and both query and document tokens expand
identically. `nn list --search "TDD"` and `"test driven development"` then
converge.

The alias table is a **projection of note content**, rebuilt with the index. No
second source of truth; it is notebook truth, versioned in Git, one-operation-
one-commit.

### Mechanics

- Add `Aliases []string` to `Note` and `aliases` (yaml, omitempty) to
  `frontmatterYAML` in `internal/note/note.go`.
- Alias expansion is applied inside `tokenize()` via a **package-level alias
  table** set once at corpus-load time (`note.SetAliases(map)`), mirroring the
  existing global `english.Stem` pattern. This avoids threading a corpus map
  through all ~35 `tokenize()`/`Tokenize()` call sites across `bm25.go`,
  `bm25_typed.go`, `list.go`, and `graph_routes.go`.
- Expansion runs before `english.Stem` so expanded tokens stem consistently.
- Expansion is bidirectional by construction: because both queries and stored
  bodies pass through the same `tokenize()`, an alias and its expansion collapse
  to a shared token set regardless of which form the user typed.

## Consequences

**Positive**
- Closes the acronym gap for exactly the vocabulary the notebook cares about,
  with zero external curation and zero noise from generic terms.
- Fits "files are truth, index is cache": the dictionary is notes; rebuildable.
- Colocates a concept's definition and its acronyms as one fact.
- Non-invasive: one package-level table, no call-site churn.

**Negative / risks**
- Package-level mutable state in `tokenize()` requires care for test isolation
  and concurrent index builds; must be set deterministically at load and treated
  as read-only during scoring.
- Requires per-concept manual tagging (once). Acceptable: only the acronyms you
  actually miss need declaring.
- Alias-target ambiguity (one acronym, two concepts) is resolved by policy below.

## Resolved policy

- **Expansion target: title-derived.** `aliases: [TDD]` expands to the tokenized
  title of the declaring note. The acronym and its expansion are one fact
  colocated on the concept note; the field carries only the short forms, not a
  `key = value` pair. Consequence: renaming a note's title changes what its
  acronyms expand to — acceptable, because the title *is* the canonical
  expansion, and a title rename is a deliberate redefinition.
- **Duplicate alias keys: reject at parse/index time.** If the same alias key is
  declared as an `aliases` entry on two different notes, index build fails with a
  clear error naming both note IDs. This keeps the expansion table a function
  (one key -> one expansion), avoids silent union ambiguity, and surfaces the
  collision to the human to resolve deliberately rather than guessing. Case
  folding matches `tokenize()` (aliases are lowercased before keying).

## Alternatives kept open

- **Auto-mined parenthetical acronyms** (`test-driven development (TDD)` regex over
  bodies) could later *suggest* `aliases` entries without auto-writing them,
  preserving precision while reducing manual effort.
- **Embeddings** remain the eventual path for full semantic similarity, out of
  scope here.
