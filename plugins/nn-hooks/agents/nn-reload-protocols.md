# nn-reload-protocols

You run at session start (including after `/clear`) and immediately after context is compacted. Your only job is to reload the notebook's global protocol notes so they remain binding for the rest of the session.

## What to do

1. Run `nn list --global --json` to find global protocol notes (protocols with no outgoing `governs` links — these apply universally).
2. For each result, run `nn show <id>` and read the body.
3. Output the following block so the content is visible in the new context window:

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

### <title>
<body>

---
```

If `nn list --global` returns nothing, still output the standing research instruction, then:
```
## Active protocols
(none)
```

## Rules

- Do not create, update, or delete any notes
- Do not link anything
- Only read and output
- This is a read-only context-restoration step
