# ADR 0024: Bulk unlink is source-scoped, typed, and atomic

## Status

Accepted

## Context

`nn` provides `bulk-link` and `bulk-update-link`, but removing several outgoing links requires repeated `unlink` operations and therefore several Git commits. That asymmetry makes graph cleanup needlessly expensive and can leave a requested batch partially applied.

Single `unlink` accepts an optional type: with `--type`, it removes only matching `(from, to, type)` edges; without it, it removes all edges between the source and target. Existing bulk link commands are source-scoped and use repeated `--to` flags with broadcast-or-positional metadata flags.

## Decision

Add:

```text
nn bulk-unlink <from-id> --to <id> [--to <id> ...] [--type <type> ...]
```

The command is source-scoped and uses raw IDs, matching `bulk-link` and `bulk-update-link`. At least one `--to` is required.

Type semantics preserve single `unlink` behavior:

- no `--type`: remove all edge types for every target;
- one `--type`: broadcast it to every target;
- one `--type` per `--to`: pair values positionally;
- any other count: reject the batch before mutation.

The backend validates the whole requested batch against one in-memory source note, filters links once, writes the source once, and creates at most one semantic Git commit. Missing edges are idempotent no-ops. Raw target IDs remain valid even when a target note file is missing, allowing broken-edge cleanup.

## Consequences

- Multi-edge cleanup becomes symmetric with bulk creation and update.
- Invalid flag shapes cannot partially mutate the graph.
- Parallel typed edges can be removed selectively or all at once.
- Successful non-empty mutations produce one commit: `note: bulk-unlink <from> → <N> notes`.
- The command inherits existing link-operation durability: file replacement is atomic and mutations are serialized by process and Git locks, but a Git commit failure may leave the working tree modified.
