# ADR 0034: Skill references are lazy, safe, and recursively packaged

## Status

Accepted

## Date

2026-08-25

## Context

`nn skills get <skill>` currently returns one embedded `SKILL.md`. That keeps the installed dispatch stub version-matched with the binary, but it forces large owner skills to load every specialized workflow at once. `nn-navigate` has grown to 74,859 bytes and combines presentation, Ask handoffs, movement, scans and route overlays, lenses, and retained-state recovery even when one action needs only one of those contracts.

A split must not weaken binding navigation semantics. It must also work from the checked-out filesystem, the Go embedded filesystem, and recursively copied or packaged skill trees. Treating a user-provided reference as a path would create traversal, platform-separator, non-Markdown, and symlink escape hazards.

## Decision

### Core and lazy references

A skill may contain a compact `SKILL.md` plus direct reference files under `references/*.md`. `nn skills get <skill>` remains compatibility-preserving: without a reference flag it prints only `SKILL.md`, byte for byte, and does not inline references.

Two mutually exclusive retrieval modes are added:

```text
nn skills get <skill> --list-references
nn skills get <skill> --reference <name>
```

`--list-references` prints stable, lexically sorted logical names and each reference's `applies_when` frontmatter value. `--reference <name>` prints exactly the selected Markdown file. A logical name is a strict lowercase ASCII stem containing only letters, digits, and internal hyphens; callers do not supply a path or `.md` suffix.

The `nn-navigate` bundle has exactly these references:

- `presentation` — human-facing maps, picker hierarchy, semantic direction, relay colors, and blocker checklist;
- `ask` — bounded consultation and Graph/Canvas/Document handoffs;
- `movement` — Enter, Orient, Recenter, Peek, Teleport, Arrive, and navigation resume;
- `scan-and-routes` — local/global Scan and typed route, impact, and path overlays;
- `lenses` — Show verbatim, Explain in depth, Analogize, Find an analog, Visualize, and Quiz;
- `state` — complete navigation frames, history, bookmarks, menu state, and recovery/compaction.

The compact core owns activation and dispatch. Before an applicable action, an agent MUST fetch that action's owning reference with `nn skills get nn-navigate --reference <name>`. When an action crosses ownership boundaries, it fetches every owning reference before acting. The core retains the complete state model, focus-mutation invariants, canonical Orient command, action-to-reference dispatch, the blocking presentation checklist, and compaction dispatch so lazy loading cannot erase those invariants.

### Safe discovery and retrieval

Reference operations are implemented against an `fs.FS` rooted at the skill collection so the same code serves source, embedded, and installed trees.

Discovery and retrieval:

1. validate the skill and reference as logical names, never paths;
2. inspect only the skill's direct `references` directory;
3. include only direct, regular, non-symlink files whose names are exactly `<valid-name>.md`;
4. reject traversal tokens, `/`, `\\`, supplied extensions, unknown names, directories, non-Markdown files, and symlinks;
5. inspect path components before opening them so an OS-backed `fs.FS` cannot follow a symlinked skill or `references` directory outside its root; and
6. return generic not-found/invalid-reference errors without attempting a fallback path.

Reference frontmatter must contain a nonblank, single-line `applies_when` value. Invalid files make listing fail rather than silently publishing unstable routing metadata.

### Embedding and installation packaging

Go embedding names whole skill directories, which recursively embeds their reference subdirectories. Tests must verify both the core and all six references through the embedded source.

Any materialization of a skill tree uses recursive `fs.WalkDir` copying and preserves the relative tree, including `references/`. The installed single `nn` dispatch stub remains the general CLI contract and asks the version-matched binary for cores and references. The Pi preset additionally materializes the standard direct skill trees recursively alongside that stub, matching Pi package discovery. The Pi npm package recursively ships only its declared extension and skill roots, and package tests verify that every `nn-navigate` reference is present. Documentation uses `nn install --for pi`; the removed `install-pi` command is not revived.

## Consequences

- Loading `nn-navigate` no longer consumes every detailed action contract.
- Agents must perform one explicit reference fetch before an applicable action; this is deliberate action ownership, not optional reading.
- Default `skills get` consumers remain compatible and receive no concatenated reference payload.
- The same safe API can be tested with `os.DirFS`, `embed.FS`, and a recursively copied installation tree.
- Symlinked skill/reference paths are rejected even when their target remains readable to the host process.
- Splitting the document creates cross-file dispatch risk; retained-property tests therefore inspect the full bundle while ownership tests ensure specialized details remain in the owning reference.
