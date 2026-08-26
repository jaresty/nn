# ADR-0025: Require typed relationships for new writes and expose legacy untyped links

## Status

Accepted — implemented

## Date

2026-08-25

## Context

ADR-0003 introduced typed links as optional metadata so existing untyped Markdown
links remained valid. Later graph and navigation features made relationship types
semantic: they determine zones, typed routes, impact traversal, graph validation,
and the meaning an agent relays to a human.

The write APIs now enforce contradictory contracts:

- `nn link` requires `--type`;
- `nn new --link-to … --annotation …` and
  `nn update --link-to … --annotation …` expose no type flag and write
  `note.Link{Type: ""}`;
- `IsKnownLinkType` treats the empty string as valid;
- `nn status` therefore reports no unknown type;
- unfiltered graph traversal preserves these relationships, but zoning cannot
  classify them.

Untyped relationships are consequently not corrupt files or merely ancient
leftovers. They are currently supported output from active capture commands. This
is a gap between the capture API and the typed graph/navigation model.

Choosing a default type would hide the gap by manufacturing semantics the caller
did not provide. Dropping or refusing to read old untyped links would lose durable
annotations and break the principle that files are truth.

## Decision

### 1. Require a known, non-empty type for every newly written relationship

All command paths that create a link must provide a canonical link type. Add a
repeatable `--link-type` flag to `nn new` and `nn update`, paired positionally with
`--link-to` and `--annotation`:

```bash
nn new \
  --link-to <id> \
  --link-type grounded-by \
  --annotation "Evidence basis"

nn update <id> \
  --link-to <id> \
  --link-type extends \
  --annotation "Builds on the model"
```

For multiple relationships:

```bash
nn new \
  --link-to a --link-type grounded-by --annotation "Evidence basis" \
  --link-to b --link-type extends     --annotation "Adds scope"
```

Before writing, commands require:

```text
len(linkTo) == len(linkType) == len(annotation)
```

A missing or unknown type is a hard error. No default type is inferred.

This decision supersedes ADR-0003 only where it made link types optional for new
writes. It preserves ADR-0003's Markdown compatibility for existing files.

### 2. Separate readability, known semantics, and write validity

`IsKnownLinkType` answers only whether a non-empty type belongs to the canonical
allowlist:

```go
func IsKnownLinkType(t string) bool {
    return KnownLinkTypes[t]
}
```

Empty types remain readable from Markdown for backward compatibility, but they are
neither known nor writable. Parsing preserves them losslessly rather than refusing
to load the note.

The model distinguishes:

- **readable:** the stored relationship can be parsed and retained;
- **known:** its type belongs to the canonical allowlist;
- **writable:** a command may create or update it under current invariants.

An empty type is readable but not known or writable.

### 3. Report missing and unknown types separately

`nn status` reports two conditions:

```json
{
  "missing_link_types": 17,
  "unknown_link_types": 0
}
```

Definitions:

```text
missing_link_types = type == ""
unknown_link_types = type != "" && !IsKnownLinkType(type)
```

Detailed output identifies source and target IDs so the relationships can be
reviewed. Empty types are not silently counted as valid and are not conflated with
unknown experimental strings.

### 4. Keep untyped relationships visible but semantically unclassified

Untyped links remain available to unfiltered structural traversal. They must not
silently disappear from human-facing graph navigation, and they must not be placed
into a semantic zone by inference.

Graph and navigation presentation adds an explicit category:

```text
⚪ UNCLASSIFIED — stored relationships without navigation semantics
```

Untyped links:

- remain visible in unfiltered neighborhoods;
- contribute to structural degree;
- are labeled as unclassified;
- do not imply support, refinement, provenance, tension, or task dependency;
- are excluded from typed route, impact, and path queries unless a future command
  explicitly requests unclassified edges;
- are surfaced as migration candidates.

Zoned views must not omit a directly connected note solely because its edge is
untyped. The unclassified category is additional to TOP, BOTTOM, LEFT, and RIGHT;
it does not assign false directional meaning.

### 5. Add an atomic relationship-retyping command

Add:

```bash
nn link set-type <from> <to> --type <type>
```

The command:

- finds an existing relationship between the endpoints;
- by default requires its current type to be empty;
- rejects ambiguous matches rather than guessing;
- accepts an annotation-matching discriminator when multiple relationships exist;
- preserves endpoints and annotation;
- changes only the type;
- performs one backend write and one Git commit.

Suggested disambiguation:

```bash
nn link set-type <from> <to> \
  --annotation-matches "fragment" \
  --type grounded-by
```

Suggested commit message:

```text
note: type link <from-id> → <to-id> as <type>
```

Retyping an already typed relationship requires an explicit current-type guard in
a future extension; the initial command is primarily a safe migration operation.

### 6. Use Graph Ask to review groups of untyped relationships without automatic mutation

ADR-0024 defines Graph Ask results as annotated selections and groups. Untyped
relationships are a first-class review use case: render them distinctly and ask a
human to group edges by proposed semantic type or mark them unresolved.

Example result:

```yaml
groups:
  - proposed_type: grounded-by
    edges: [a-to-b, c-to-d]
  - proposed_type: extends
    edges: [e-to-f]
  - proposed_type: null
    classification: unresolved
    edges: [g-to-h]
    comment: The stored annotation does not establish meaning or direction.
```

The result is a proposal. The agent verifies it against note bodies, link
direction, and annotations, then invokes explicit `nn link set-type` operations.
Graph Ask never mutates the notebook automatically.

### 7. Tighten validation in stages

The rollout preserves readability while stopping new debt:

1. reject untyped links from all new writes;
2. report existing missing types;
3. show them explicitly in graph/navigation views;
4. support atomic retyping;
5. migrate resolvable relationships;
6. consider stronger notebook-wide rule violations only after migration tooling is
   proven.

Old notebooks remain loadable throughout.

## Consequences

- Capture and standalone link commands converge on one typed relationship
  invariant.
- New navigation edges always have enough semantics for zoning and typed graph
  operations.
- Existing annotations and endpoints remain durable and readable.
- `nn status` exposes real semantic debt instead of reporting empty types as known.
- Navigation gains an explicit unclassified category that avoids both omission and
  false inference.
- Callers of `nn new` and `nn update` that currently use `--link-to` must add
  `--link-type`; this is an intentional CLI compatibility break at the write
  boundary.
- Migration requires human or agent judgment because type cannot be inferred
  safely from connectivity alone.

## Implementation order

1. Add `--link-type` to `nn new` and `nn update`; validate complete triples and
   known types before writing.
2. Change `IsKnownLinkType` so empty is not known while keeping Markdown parsing
   backward-compatible.
3. Add `missing_link_types` to human and JSON status output.
4. Add unclassified-edge rendering and ensure directly connected untyped
   neighbors remain visible in navigation.
5. Add `nn link set-type` with ambiguity and annotation-preservation tests.
6. Add a Graph Ask review workflow for grouping untyped edges by proposed type.
7. Evaluate a later `nn rules check` violation after migration coverage is known.

## Alternatives considered

**Default missing types to `related`:** Rejected. `related` is not a canonical
neutral fact; assigning it would invent semantics and direction.

**Continue accepting empty types because Markdown supports them:** Rejected.
Storage compatibility does not imply that current write APIs should keep creating
relationships unusable by typed navigation.

**Reject notebooks containing any untyped relationship:** Rejected. Existing files
are truth, and old relationships may contain valuable annotations even when their
semantic type is absent.

**Infer types automatically from annotations or neighboring notes:** Rejected as a
mutation policy. Such inference may produce review candidates, but it requires
human or agent verification before write-back.

**Repair by unlinking and recreating relationships:** Rejected. It risks losing
annotations, creates unnecessary intermediate states, and fails the one-operation,
one-semantic-commit principle.

**Assign untyped edges to RIGHT or another existing zone:** Rejected. Every zone
has semantic meaning; arbitrary placement would misrepresent the stored
relationship.
