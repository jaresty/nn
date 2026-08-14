# ADR-0019: Embedded Datalog rules engine over note Markdown

## Status

Accepted

## Context

We want a general rules/inference capability over the note graph — to derive
implicit relationships (e.g. transitive `governs`), flag invariant violations,
and support open-ended future uses not yet enumerated. Two shaping constraints
emerged from design discussion:

1. **Rules should live in the Markdown**, not in a separate config file, to keep
   faith with the project principle *"Files are truth, index is cache."* Rules
   version alongside notes in Git and the engine stays a pure derivation over
   what is on disk.
2. **Pure-Go only** — no CGO, no JVM subprocess. The CLI must stay portable.

We also decided (design session) that this engine should eventually **absorb the
existing `nn check` structural validations** rather than duplicate them, and that
**validation (violation reporting) is the v1 headline capability**.

### Why not OWL

OWL is a description logic built for open-world class subsumption and ontology
consistency. `nn` is a typed, directed graph with self-defined link semantics
(`refines`, `contradicts`, `governs`, `requires`, `grounded-by`, …). What we want
is closed-world derivation and validation ("no link ⇒ false"), which OWL's
open-world assumption actively fights. There is also **no mature pure-Go OWL
reasoner** — the established reasoners (HermiT, Pellet/Openllet, ELK, Fact++) are
Java/C++, requiring a heavy non-Go dependency.

### Why not an off-the-shelf Go rules engine

- **grule** (pure Go) is an imperative production-rule engine; it does not
  natively give recursion / transitive closure, and derived facts do not
  cleanly feed back as premises — awkward for the derivation use case.
- **GoRules / ZEN** is fast but binds a Rust core via CGO — violates pure-Go.
- **kevinawalsh/datalog** is the only pure-Go Datalog candidate found, but is
  small (≈37★, 35 commits) and **LGPL**, a licensing hazard to embed in the CLI.

### Spike evidence

A ~330-line throwaway prototype (engine core ≈200 lines) proved the full vertical
slice in pure Go with zero dependencies:

- **Recursion / transitive closure** — a recursive rule walked a
  `governs`→`refines`→`refines` chain correctly.
- **Validation as derived facts** — `violation(D, "contradicts a permanent note")`
  was flagged correctly.
- **Stratified negation** — `unreviewed_governed` correctly excluded reviewed and
  permanent notes via `!reviewed_or_perm(N)`.
- **Fence parsing** — rules were extracted from a ` ```nn-rule ` block in a note
  body.

Termination is guaranteed: without function symbols, Datalog's fact universe is
finite, so the fixpoint always halts.

## Decision

Build a **hand-rolled, pure-Go, semi-naive Datalog engine** that evaluates rules
**embedded in note Markdown**, as a pure derivation layer (never a source of
truth; fully rebuildable like the SQLite index).

### 1. Facts — auto-exposed from every note parse (closed-world)

| Predicate | Meaning |
|---|---|
| `note(ID, Type, Status)` | one per note |
| `link(From, To, LinkType)` | one per link |
| `tag(ID, Tag)` | one per tag |
| `open_item(ID, Text)` | one per `- [ ]` checkbox |
| `expires(ID, Date)` | if present |
| `representation(ID, Rep)` | if present |

### 2. Rules — embedded in Markdown, tiered

- **Primary carrier:** ` ```nn-rule ` fenced code blocks in any note body. A
  `type:protocol` note can carry both its prose *and* a machine-checkable rule,
  closing the loop where the LLM interprets `governs` protocols today.
- **Built-in tier:** shipped via `go:embed` (`internal/rules/builtin.dl`) — the
  invariants `nn check` hard-codes today, re-expressed as Datalog `violation(...)`
  clauses.
- **User tier:** any `nn-rule` fence in the user's own notes, evaluated alongside
  built-ins. A violation is a derived `violation(ID, Reason)` fact regardless of
  origin.
- **Frontmatter shorthand** (secondary, optional): `rule: "violation(X) :- ..."`
  compiles to the same clause for one-liners.

### 3. Absorb `nn check` into built-in rules

The existing representation-subgraph invariants (see `cmd/nn/cmd/check.go`) are
ported to `builtin.dl`, and `nn check` becomes a thin caller that runs the engine
and filters `violation` facts for the relevant representation. The invariants to
preserve exactly (same-representation traversal only):

- **ontology**: root must be `type:model`; non-root nodes `type:concept` or
  `type:argument`; no cycles.
- **taxonomy**: ontology rules + all same-rep links must be `refines`/`extends`.
- **axiom**: root must have ≥1 `grounded-by` same-rep link; no cycles.

`nn check` continues to **not** validate section headers within note bodies
(explicitly out of scope, per prior design).

### 4. CLI surface

```
nn rules check              # print all violation(_, _) facts; nonzero exit if any
nn rules query "PRED(...)"  # print derived facts matching a pattern
nn rules list               # loaded rules + provenance (which note each came from)
nn rules explain <fact>     # (stretch) derivation path for a derived fact
```

### 5. Package layout

```
internal/rules/
  engine.go     # semi-naive fixpoint, unify/subst, stratified negation
  parse.go      # atom/rule parser + nn-rule fence extractor
  facts.go      # note.Note -> []fact  (only coupling to the rest of nn)
  builtin.dl    # embedded core invariants (go:embed)
cmd/nn/cmd/
  rules.go      # cobra: check / query / list / explain
```

## Consequences

### Positive

- Single rule system: `nn check` and `nn rules` stop being separate hand-coded
  paths; new invariants are Datalog clauses, not Go.
- Rules version with notes in Git; engine is pure derivation, fully rebuildable.
- Protocol notes can carry enforceable rules, not just LLM-interpreted prose.
- Zero new dependencies; pure Go; deterministic termination.

### Negative / risks

- **We own the engine.** ≈200 lines to maintain, including a small parser.
- **Naive join** (iterate all facts per literal) is fine at notebook scale but
  needs predicate indexing if performance ever matters. Ship naive, optimize on
  evidence.
- **Negation must be stratified.** v1 must **detect and reject non-stratifiable
  rulesets** rather than silently misbehave.
- **Malformed `nn-rule` fences** must warn with note-ID provenance and still let
  the note parse — a bad rule must never break note loading.
- **Absorbing `nn check` raises the correctness bar**: built-in rules must
  reproduce current check semantics exactly; port must be test-covered against
  the existing check behavior before `nn check` is switched to delegate.

## Deferred (v1 scope discipline)

- Arithmetic/comparison predicates (`>`, `<`), aggregation (count/min).
- Rule *scoping* — v1 evaluates all rules globally; revisit only if collisions bite.
- Performance indexing.
- `nn rules explain` (derivation provenance) — stretch, not v1-blocking.

## Alternatives considered

| Alternative | Rejected because |
|---|---|
| OWL + JVM/C++ reasoner | Open-world mismatch; no pure-Go reasoner; heavy dependency |
| grule (pure Go) | Imperative; no native recursion/transitive closure |
| GoRules / ZEN | CGO (Rust core) — violates pure-Go constraint |
| kevinawalsh/datalog | Small, unproven; LGPL licensing hazard |
| External `.dl` config file | Rules would drift from notes; violates "files are truth" |
| Extend `nn check` in Go only | Stays special-purpose; no general derivation substrate |
