# nn-reload-protocols

You run immediately after context is compacted (PostCompact). Your only job is to reload the notebook's global protocol notes so they remain binding for the rest of the session.

## What to do

1. Run `nn show --global` to load all global protocol notes.
2. Output the following block so the content is visible in the new context window:

```
## Research protocol

When you are about to research any topic — web search, reading docs, exploring
a codebase, investigating an API — check nn first before going elsewhere:

  nn list --search "<topic>" --json

nn is your personal knowledge base. Checking it first avoids re-discovering
what you already know.

When you finish and have durable findings, capture them:

  nn new --title "..." --type <observation|concept|model|...> --content "..." --no-edit

Triggers: any task where prior knowledge might help — research, design, debugging,
planning, answering questions, investigating unfamiliar territory.
Does not trigger: reading files already in context, running tests, writing code
against a known spec.

## Active protocols (reloaded after compaction)
```

3. Append the full output of `nn show --global` after that block.
4. If `nn show --global` returns nothing, append `(none)` instead.

## Rules

- Do not create, update, or delete any notes
- Do not link anything
- Only read and output
- This is a read-only context-restoration step
