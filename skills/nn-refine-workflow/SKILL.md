---
name: nn-refine-workflow
description: Batch hygiene pass — invoke nn-refine across a filtered set of notes and present proposed changes grouped by note.
when_to_use: When you want to run a hygiene pass across multiple notes at once — drafts, orphans, long notes, or a topic cluster. Invoke with /nn-refine-workflow.
---

# nn-refine-workflow

Batch hygiene pass: invoke `nn-refine` across a filtered set of notes and present proposed changes grouped by note.

## When to use

Invoke when you want to clean up multiple notes in one pass — for example:
- All draft notes in a topic cluster
- Long notes that exceed the atomicity threshold
- Orphan notes with no links
- Notes last modified before a given date

## Workflow

### 1. Select the target set

Choose a filter to identify the notes to refine:

```
# All drafts
nn list --status draft --json

# Long notes (atomicity candidates)
nn list --long --json

# Orphans
nn list --orphan --json

# Topic cluster
nn list --search "<topic>" --json

# Combined: draft orphans
nn list --status draft --orphan --json
```

### 2. Load each note

For each note in the set, run:

```
nn show <id>
```

### 3. Apply nn-refine steps

For each note, work through the four `nn-refine` steps:

1. **Atomicity** — does the body contain more than one independent claim? Propose a split.
2. **Links** — run `nn suggest-links <id>` and propose `nn link` commands.
3. **Promotion** — check `nn backlinks <id>` and propose `nn promote` if eligible.
4. **Title** — does the title name the single claim? Propose `nn update --title` if not.

### 4. Present grouped proposals

Group all proposed changes by note ID before executing. Present them as a list of `nn` commands for review:

```
## <id> — <title>
nn update <id> --title "..." --no-edit
nn bulk-link <id> --to <other-id> --annotation "..." --type <type>
nn promote <id> --to reviewed
```

Execute only after confirming the proposals are correct.

### 5. Health check

After the batch pass, run a health report to verify improvement:

```
nn status --json
nn review --format json
```

## nn commands used

```
nn list [--status STATUS] [--long] [--orphan] [--search TOPIC] [--json]
nn show <id>
nn suggest-links <id> [--limit N]
nn backlinks <id> [--json]
nn new --title "..." --type <type> --content "..." --no-edit
nn link <from> <to> --annotation "..." --type TYPE
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> ...]
nn promote <id> --to reviewed|permanent
nn update <id> [--title "..."] [--content "..."] [--status STATUS] --no-edit
nn status [--json]
nn review [--format json]
```

## Success criteria

- Every note in the target set has been processed through all four nn-refine steps
- All proposed changes are grouped by note and presented before execution
- A health report is run after the batch pass confirming reduction in orphans, drafts, or long notes
