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

3. **Link**: For each relevant existing note, add an annotated link:
   ```
   nn link <new-id> <existing-id> --annotation "..."
   nn link <existing-id> <new-id> --annotation "..."
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
| `nn list` | List/filter notes |
| `nn link <from> <to> --annotation "..."` | Add a link |
| `nn unlink <from> <to>` | Remove a link |
| `nn graph --json` | Export link graph |
| `nn status [--json]` | Notebook health (orphans listed with IDs/titles) |
| `nn links <id> [--json]` | Outgoing links with annotations |
| `nn promote <id> --to reviewed` | Advance review status |
| `nn delete <id> --confirm` | Delete a note |
