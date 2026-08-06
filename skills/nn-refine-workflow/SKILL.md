---
name: nn-refine-workflow
description: Use when running a hygiene pass across multiple notes at once — drafts, orphans, long notes, or a topic cluster. Load with `nn skills get nn-refine-workflow`.
when_to_use: When you want to run a hygiene pass across multiple notes at once — drafts, orphans, long notes, or a topic cluster. Invoke with /nn-refine-workflow.
---

# nn-refine-workflow

Batch hygiene pass: reason over a filtered set of notes, group all proposed changes, then execute as bulk operations — one pass, one batch, one health check.

## When to use

Invoke when you want to clean up multiple notes in one pass — for example:
- All draft notes in a topic cluster
- Long notes that exceed the atomicity threshold
- Orphan notes with no links
- Notes last modified before a given date
- A coherent multi-note representation change (ontology, taxonomy, axiom subgraph)

## Execution principle

**Reason first, execute second.** Work through all notes before issuing any commands. Collect every proposed change — splits, links, renames, promotions — into a single grouped proposal. Execute only after the full proposal is confirmed. Do not summarise after each note; summarise once after all notes are processed.

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

# Representation subgraph
nn graph show --focus <root-id> --depth 3
```

### 2. Load all notes before proposing any changes

For each note in the set, run `nn show <id>` to load its body, links, and backlinks. Load the full set before forming any proposals — do not propose changes note-by-note as you read.

### 3. Reason across the full set

After loading all notes, work through the four `nn-refine` dimensions **across the whole set**:

1. **Atomicity** — which notes contain more than one independent claim? For each: propose a split using `bulk-new` if creating multiple new notes, or `nn new` for a single split.
2. **Links** — which notes are missing connections to other notes in the set, or to the broader graph? Propose `nn bulk-link` for each source note's full link batch. Prefer `nn bulk-update-link` for correcting existing link types or annotations (one commit per source note).
3. **Promotion** — which notes are eligible (single claim, at least one inbound reviewed link)? For a representation subgraph, use `nn promote --subgraph <root> --to reviewed --if-valid` rather than promoting notes individually.
4. **Title** — which titles do not name their single claim? Collect all renames.

### 4. Present one grouped proposal

Group all proposed changes by note ID. Present the complete proposal before executing any of it:

```
## Splits (bulk-new)
nn bulk-new --json '[
  {"title": "...", "type": "concept", "content": "..."},
  {"title": "...", "type": "argument", "content": "..."}
]'

## Link corrections (bulk-update-link, one per source note)
nn bulk-update-link <id-a> \
  --to <id-b> --type extends --annotation "..." \
  --to <id-c> --type supports --annotation "..."

## New links (bulk-link, one per source note)
nn bulk-link <id-a> \
  --to <id-d> --annotation "..." --type refines \
  --to <id-e> --annotation "..." --type source-of

## Renames
nn update <id> --title "Better title" --no-edit --since <modified>

## Subgraph promotion (validates first, then promotes in dependency order)
nn promote --subgraph <root-id> --to reviewed --if-valid

## Representation check (after promotion)
nn check <root-id>
```

Execute only after the user confirms the proposal is correct.

### 5. Health check

After executing the batch, run a health report to verify improvement:

```
nn status --json
nn review --format json
```

## nn commands used

```
nn list [--status STATUS] [--long] [--orphan] [--search TOPIC] [--json]
nn show <id>
nn graph show --focus <id> --depth N
nn suggest-links <id> [--limit N]
nn backlinks <id> [--json]
nn bulk-new --json '[{"title":"...","type":"...","content":"...","applies_when":"..."}]'
nn bulk-update --json '[{"id":"...","applies_when":"...","title":"...","content":"..."}]'
nn new --title "..." --type <type> --content "..." --no-edit
nn graph apply <manifest.yaml> [--dry-run|--commit]
nn link <from> <to> --annotation "..." --type TYPE
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> ...]
nn bulk-update-link <from> --to <id> [--type TYPE] [--annotation "..."] [--status reviewed] [--to ...]
nn promote --subgraph <root-id> --to reviewed --if-valid
nn promote <id> --to reviewed|permanent
nn check <id>
nn check <id> --root auto
nn update <id> [--title "..."] [--content "..."] [--status STATUS] --no-edit --since <modified>
nn status [--json]
nn review [--format json]
```

## Success criteria

- All notes in the target set were loaded before any proposals were made
- All proposed changes were grouped by note and presented as a single batch before execution
- Splits used `bulk-new`; link corrections used `bulk-update-link`; subgraph promotion used `nn promote --subgraph --if-valid`
- A health report confirmed improvement after the batch pass
- Only one summary was produced (after all notes were processed, not after each note)
