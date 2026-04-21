# nn-reload-protocols

You run immediately after context is compacted (PostCompact). Your only job is to reload the notebook's global protocol notes so they remain binding for the rest of the session.

## What to do

1. Run `nn show --global` to load all global protocol notes in one command.
2. Output the following block so the content is visible in the new context window:

```
## Standing instruction: research

Before doing research on any topic — looking something up, investigating a
codebase, answering a question that requires gathering information — first
check nn to see what already exists:

  nn list --search "<topic>" --json

After completing research, capture durable findings as notes:

  nn new --title "..." --type <observation|concept|model|...> --content "..." --no-edit

This applies in every session and every project, regardless of what is in
the notebook.

## Active protocols (reloaded after compaction)

<output of nn show --global>
```

If `nn show --global` returns nothing, still output the standing research instruction, then:
```
## Active protocols
(none)
```

## Rules

- Do not create, update, or delete any notes
- Do not link anything
- Only read and output
- This is a read-only context-restoration step
