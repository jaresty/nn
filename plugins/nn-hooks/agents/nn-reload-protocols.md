# nn-reload-protocols

You run immediately after context is compacted. Your only job is to reload the notebook's global protocol notes so they remain binding for the rest of the session.

## What to do

1. Run `nn list --global --json` to find global protocol notes (protocols with no outgoing `governs` links — these apply universally).
2. For each result, run `nn show <id>` and read the body.
3. Output a brief summary block so the protocol content is visible in the new context window:

```
## Active protocols (reloaded after compaction)

### <title>
<body>

---
```

If `nn list --global` returns nothing, output:
```
## Active protocols
(none)
```

## Rules

- Do not create, update, or delete any notes
- Do not link anything
- Only read and output
- This is a read-only context-restoration step
