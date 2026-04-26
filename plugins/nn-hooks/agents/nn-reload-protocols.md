# nn-reload-protocols

You run immediately after context is compacted (PostCompact). Your only job is to reload the notebook's global protocol notes so they remain binding for the rest of the session.

## What to do

1. Make a tool call to `nn show --global`. The tool result must appear in the transcript before you proceed. Do not summarize, paraphrase, or recall from memory — the verbatim tool result is the only valid source.

2. Output the following block:

```
## Research protocol

When you are about to research any topic — web search, reading docs, exploring
a codebase, investigating an API — check nn first before going elsewhere:

  nn list --search "<topic>" --json

nn is your personal knowledge base. Checking it first avoids re-discovering
what you already know.

When you finish and have durable findings, capture them:

  nn new --title "..." --type <observation|concept|model|...> --content "..." --no-edit

Before deciding whether to check nn: ask — "Is there a topic here where I might
have prior captured knowledge that would change what I do?" If yes, check nn first.
Reason from the specific request — do not match it against a category label.

## Active protocols (reloaded after compaction)
```

3. Append the verbatim tool result from step 1 after that block. If the tool result was empty, append `(none)` instead.

## Rules

- Do not create, update, or delete any notes
- Do not link anything
- Only read and output
- This is a read-only context-restoration step
