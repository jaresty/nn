# nn-workflow

A multi-step workflow skill for operating the `nn` Zettelkasten CLI as an LLM agent.

## When to use

Use this skill when asked to organise, capture, or link notes in the user's Zettelkasten.
Invoke it with `/nn-workflow`.

## Workflow

0. **Session Start**: Load relevant protocols before doing other work:
   - **Global protocols** (no outgoing `governs` links): `nn list --global --json` — load each with `nn show <id>`
   - **Contextual protocols** (linked to notes in scope): when the user names specific notes or topics, run `nn backlinks <note-id> --type governs` for each; load any protocol notes returned
   - Treat all loaded protocol bodies as binding operating instructions for this session. An empty result is a no-op.

1. **Capture**: Identify the atomic idea to record. Choose a `type`:
   - `concept` — a single defined idea or principle
   - `argument` — a claim with supporting reasoning
   - `model` — a framework or mental model
   - `hypothesis` — an untested conjecture
   - `observation` — a concrete empirical note
   - `question` — an open question the graph should eventually answer
   - `protocol` — an imperative operating instruction for the LLM (loaded at session start)

2. **Create**: Run `nn new` with all flags (non-interactive):
   ```
   nn new --title "..." --type <type> --content "..." --no-edit
   ```
   After each `nn new`, `nn update`, or `nn link`, print one sentence to the user summarising what was recorded and why (e.g. "Captured *X* as a concept note — it defines the core invariant driving Y.").

3. **Link**: For each relevant existing note, add annotated links. `--type` is required. New links default to `--status draft`.
   ```
   nn link <new-id> <existing-id> --annotation "..." --type refines|contradicts|source-of|extends|supports|questions|governs
   nn link <existing-id> <new-id> --annotation "..." --type <type>
   ```
   Pass `--status reviewed` when you (as a human) are explicitly creating and endorsing the link.
   For multiple targets at once (single commit):
   ```
   nn bulk-link <new-id> \
     --to <id1> --annotation "..." --type <type> \
     --to <id2> --annotation "..." --type <type>
   ```
   Annotations must explain the relationship — never bare links.

4. **Review**: Run `nn status` to check for orphans, broken links, long notes (candidates for splitting), and hub notes (high-connectivity anchors). Draft links count is also reported.
   - Triage unendorsed links: `nn links <id> --status draft` — review each and run `nn update-link <from> <to> --status reviewed` once verified.
   - Find notes that have grown too large: `nn list --long`
   - Explore how ideas cluster: `nn clusters`
   - Find shortest path between two ideas: `nn path <id-a> <id-b>`
   - Discover unlinked related notes: `nn list --similar <id> --limit 10` — surfaces notes sharing vocabulary with a given note. Good for finding connections the graph doesn't yet capture.
   - Load a topic subgraph as context: `nn show <id> --depth 2` — prints the note and all notes reachable within 2 hops as a single Markdown document.
   - Serendipitous re-encounter: `nn random --status permanent` — revisit a permanent note and consider whether it connects to current work.

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
| `nn list [--search TEXT] [--sort modified|title|created] [--type TYPE]` | List/filter/rank notes |
| `nn list --global [--json]` | Global protocols (no outgoing governs links) |
| `nn link <from> <to> --annotation "..." --type TYPE` | Add a link |
| `nn bulk-link <from> --to <id> --annotation "..." --type TYPE ...` | Add multiple links (1 commit) |
| `nn unlink <from> <to>` | Remove a link |
| `nn graph --json` | Export link graph |
| `nn status [--json] [--hubs N]` | Notebook health: orphans, drafts, broken links, draft links, long notes, hub notes |
| `nn links <id> [--type TYPE] [--status draft\|reviewed] [--json]` | Outgoing links; filter by type or status |
| `nn backlinks <id> [--type TYPE] [--json]` | Notes that link TO this note (inbound links) |
| `nn update-link <from> <to> [--annotation "..."] [--type TYPE] [--status reviewed]` | Update link metadata; use --status reviewed to endorse a draft link |
| `nn bulk-update-link <from> --to <id> [--type TYPE] [--annotation "..."] [--status reviewed] ...` | Update multiple links (1 commit) |
| `nn list --long [--json]` | Notes exceeding atomicity threshold |
| `nn list --similar <id> [--limit N] [--json]` | Notes ranked by BM25 similarity to a given note (excludes source) |
| `nn random [--status STATUS] [--tag TAG] [--json]` | Random note, optionally filtered |
| `nn show <id> --depth N [--json]` | Note + all reachable notes within N hops (BFS, outgoing links) |
| `nn path <id-a> <id-b> [--json]` | Shortest undirected path between two notes |
| `nn clusters [--min N] [--singletons] [--json]` | Topological clusters via label propagation |
| `nn ast <file> [--trace] [--root DIR] [--json]` | Structural outline of a source file; --trace finds references to all symbols |
| `nn new --from-stdin` | Create note with body from stdin |
| `nn new --from-file PATH` | Create note scaffolded from ast outline of a source file |
| `nn update <id> --content "..." --no-edit` | Replace note body |
| `nn update <id> --append "..." --no-edit` | Append to note body |
| `nn update <id> --title "..." --no-edit` | Rename note |
| `nn update <id> --type TYPE --no-edit` | Change note type |
| `nn promote <id> --to reviewed` | Advance review status |
| `nn delete <id> --confirm` | Delete a note |
