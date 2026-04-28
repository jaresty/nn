---
name: nn-capture-discipline
description: Enforces the research protocol — search nn before going external, then write or skip with a verifiable artifact.
when_to_use: Before any action that introduces new information not already present in the conversation. Invoke with /nn-capture-discipline.
---

# nn-capture-discipline

Enforces the research protocol: search nn before going external, then write or skip with a verifiable artifact.

## When to use

Invoke before any action that introduces new information not already present in the conversation: spawning an agent, fetching a URL, searching the web, or reading an unfamiliar file.

## Workflow

1. **Search nn first** — run `nn list --search "<topic>" --json` before any external action. The tool result must appear in the transcript before the information-gathering action.

2. **Read top results** — if results are returned, run `nn show <id>` on the most relevant ones. Do not proceed to external sources until you have inspected the results.

3. **Durability check** — ask: would this finding change behavior in a future session with no memory of this one? If no, it is not durable.

4. **Write or skip (required)**:
   - **Capturing**: write one sentence `Capturing: <finding> from <source>.` then run one of:
     - `nn new --title "..." --type <type> --content "..." --no-edit`
     - `nn update <id> --append "..." --no-edit`
     - `nn link <from> <to> --annotation "..." --type <type>`
   - **Skipping**: write one sentence naming the specific findings inspected and why none meet the durability bar. Then run:
     - `echo "nn-capture-skip: <that sentence verbatim>"`

Both paths require the derivation sentence to appear in the transcript **before** the tool call.

## nn commands used

```
nn list --search "<topic>" --json
nn show <id>
nn new --title "..." --type <type> --content "..." --no-edit
nn update <id> [--append "..." | --content "..."] --no-edit
nn link <from> <to> --annotation "..." --type <type>
```

## Success criteria

- Every external information-gathering action is preceded by an `nn list --search` tool result
- Every action that returns new information is followed by either an `nn` write operation or a `nn-capture-skip` echo
- No tool call occurs without a preceding derivation sentence in the transcript
