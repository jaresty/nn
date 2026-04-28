---
name: nn-refine
description: Single-note hygiene — atomicity check, link suggestions, promotion eligibility, and title check.
when_to_use: After nn show <id> when you want to clean up a specific note, or before considering a note for promotion. Invoke with /nn-refine.
---

# nn-refine

Single-note hygiene: atomicity check, link suggestions, promotion eligibility, and title check.

## When to use

Invoke after `nn show <id>` when you want to clean up a specific note. Run this skill on any draft note before considering it for promotion, or on any note that has grown stale.

## Workflow

### 1. Atomicity check

Does the note body contain more than one independent claim?

```
nn show <id>
```

If yes — the note is not atomic. Propose a split:
- `nn new --title "<claim 2>" --type <type> --content "<extracted body>" --no-edit`
- `nn link <new-id> <original-id> --annotation "split from" --type refines`
- `nn update <original-id> --content "<remaining body>" --no-edit`

A note is atomic when removing any sentence would leave only one coherent claim.

### 2. Link discovery

Run the link suggester to surface candidate connections:

```
nn suggest-links <id> [--limit 20]
```

Review candidates and propose `nn link` or `nn bulk-link` commands. Every link requires `--annotation` and `--type`. See `nn-link-suggester` for the full workflow.

### 3. Promotion eligibility

A note is eligible for promotion when:
- Its body is focused (single claim, not a list of loosely related ideas)
- It has at least one inbound reviewed link (run `nn backlinks <id>` to check)
- Its status is `draft` or `reviewed`

If eligible, propose:
```
nn promote <id> --to reviewed      # draft → reviewed
nn promote <id> --to permanent     # reviewed → permanent
```

Or use direct assignment:
```
nn update <id> --status reviewed --no-edit
nn update <id> --status permanent --no-edit
```

### 4. Title check

Does the title name the single claim in the body? A good title is a complete, falsifiable assertion or a precise concept label.

If not, propose:
```
nn update <id> --title "Better title that names the claim" --no-edit
```

## nn commands used

```
nn show <id>
nn suggest-links <id> [--limit N]
nn new --title "..." --type <type> --content "..." --no-edit
nn link <from> <to> --annotation "..." --type TYPE
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> ...]
nn backlinks <id> [--json]
nn promote <id> --to reviewed|permanent
nn update <id> [--title "..."] [--content "..."] [--status STATUS] --no-edit
nn list --long --json
```

## Success criteria

- Atomicity: note body contains exactly one independent claim (or split proposed)
- Links: `nn suggest-links` was run and candidate links were reviewed
- Promotion: eligibility was assessed and a promotion command was proposed or skipped with reason
- Title: title names the single claim or a rename was proposed
