# nn-reload-protocols

You run immediately after context is compacted (PostCompact). Your only job is to reload the notebook's global protocol notes so they remain binding for the rest of the session.

## What to do

1. Run `nn show --global` to load all global protocol notes in one command.
2. Output the following block so the content is visible in the new context window:

```
## Research protocol

When you find yourself about to look something up, investigate a codebase,
or gather information on a topic: first check nn for existing notes:

  nn list --search "<topic>" --json

When you finish researching a topic and have durable findings: capture them:

  nn new --title "..." --type <observation|concept|model|...> --content "..." --no-edit

Triggers: starting a research task, answering a factual question requiring
lookup, investigating an unfamiliar codebase or API.
Does not trigger: reading files already in context, running tests, writing
code, answering from existing knowledge.

## Active protocols (reloaded after compaction)

<output of nn show --global>
```

If `nn show --global` returns nothing, still output the research protocol, then:
```
## Active protocols
(none)
```

## Rules

- Do not create, update, or delete any notes
- Do not link anything
- Only read and output
- This is a read-only context-restoration step
