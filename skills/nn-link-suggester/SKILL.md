---
name: nn-link-suggester
description: After any capture, surface candidate notes and propose nn link commands with annotations.
when_to_use: After nn new or nn update, to discover notes that should be linked to the newly created or modified note. Invoke with /nn-link-suggester.
---

# nn-link-suggester

After any capture, surface candidate notes and propose `nn link` commands with annotations.

## When to use

Invoke after `nn new` or `nn update` to discover notes that should be linked to the newly created or modified note.

## Workflow

1. **Get candidates** — run `nn suggest-links <id>` to get a BM25-ranked list of candidate notes. The output includes already-linked notes (marked) and notes with no term overlap (excluded with count).

2. **Inspect candidates** — read the candidate summaries. For each candidate, assess whether a meaningful relationship exists.

3. **Filter already-linked** — the output marks notes already linked; skip those unless an additional link type is warranted.

4. **Propose links** — for each accepted candidate, name the relationship type and write an annotation explaining why. Use `nn bulk-link` for multiple targets (single commit):
   ```
   nn bulk-link <id> \
     --to <id1> --annotation "..." --type <type> \
     --to <id2> --annotation "..." --type <type>
   ```
   Or single links:
   ```
   nn link <id> <other-id> --annotation "..." --type <type>
   ```

5. **Bidirectional check** — consider whether the target note should also link back. If so, add reverse links with appropriate types.

## nn commands used

```
nn suggest-links <id> [--limit N]
nn link <from> <to> --annotation "..." --type TYPE
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> --annotation "..." --type TYPE]...
nn update-link <from> <to> --annotation "..." --type TYPE [--status reviewed]
```

## Canonical link types

`refines` | `contradicts` | `source-of` | `extends` | `supports` | `questions` | `governs`

## Success criteria

- `nn suggest-links` is run on the focal note
- Every proposed `nn link` includes both `--annotation` and `--type`
- No bare links (annotation is required — it is a schema violation to omit it)
- Already-linked notes are only re-linked when a distinct additional type is warranted
