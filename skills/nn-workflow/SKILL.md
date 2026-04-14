# nn-workflow

A multi-step workflow skill for operating the `nn` Zettelkasten CLI as an LLM agent.

## When to use

Use this skill when asked to organise, capture, or link notes in the user's Zettelkasten.
Invoke it with `/nn-workflow`.

## Workflow

1. **Capture**: Identify the atomic idea to record. Choose a `type`:
   - `concept` — a single defined idea or principle
   - `argument` — a claim with supporting reasoning
   - `model` — a framework or mental model
   - `hypothesis` — an untested conjecture
   - `observation` — a concrete empirical note

2. **Create**: Run `nn new` with all flags (non-interactive):
   ```
   nn new --title "..." --type <type> --content "..." --no-edit
   ```

3. **Link**: For each relevant existing note, add annotated links. Use `--type` when the relationship is specific:
   ```
   nn link <new-id> <existing-id> --annotation "..." [--type refines|contradicts|source-of|extends|supports|questions]
   nn link <existing-id> <new-id> --annotation "..."
   ```
   For multiple targets at once (single commit):
   ```
   nn bulk-link <new-id> \
     --to <id1> --annotation "..." \
     --to <id2> --annotation "..."
   ```
   Annotations must explain the relationship — never bare links.

4. **Review**: Run `nn status` to check for orphans (printed with IDs and titles) or broken links.
   To audit link annotations for a specific note: `nn links <id>`

## Non-interactive rules

- Always pass `--no-edit` when creating notes
- Always pass `--annotation` when linking
- Always pass `--confirm` when deleting
- Use `--json` for machine-readable output when parsing results

## Key commands

| Command | Usage |
|---|---|
| `nn new` | Create a note |
| `nn show <id>` | Read a note |
| `nn list [--search TEXT] [--sort modified|title|created]` | List/filter/rank notes |
| `nn link <from> <to> --annotation "..." [--type TYPE]` | Add a link |
| `nn bulk-link <from> --to <id> --annotation "..."...` | Add multiple links (1 commit) |
| `nn unlink <from> <to>` | Remove a link |
| `nn graph --json` | Export link graph |
| `nn status [--json]` | Notebook health (orphans listed with IDs/titles) |
| `nn links <id> [--type TYPE] [--json]` | Outgoing links with annotations (filterable by type) |
| `nn update-link <from> <to> [--annotation "..."] [--type TYPE]` | Update link metadata in place |
| `nn bulk-update-link <from> --to <id> --type TYPE ...` | Update multiple links (1 commit) |
| `nn update <id> --content "..." --no-edit` | Replace note body |
| `nn update <id> --append "..." --no-edit` | Append to note body |
| `nn update <id> --title "..." --no-edit` | Rename note |
| `nn promote <id> --to reviewed` | Advance review status |
| `nn delete <id> --confirm` | Delete a note |
