---
name: nn-capture-discipline
description: Enforces the research protocol — search nn before going external, then write or skip with a verifiable artifact after the external action completes.
when_to_use: Before any action that introduces new information not already present in the conversation. Invoke with /nn-capture-discipline.
---

# nn-capture-discipline

The workflow below is not optional. Every external action — spawning an agent, fetching a URL, searching the web, reading an unfamiliar file — is gated by Step 1. The gate is the `nn list --search` tool result in the transcript, not a judgment that the search is unnecessary.

## Workflow

**Step 1 — Search nn (required; blocks all external actions)**

Before planning any external action, run `nn list --search "<topic>" --json` in its own message. The external action is not permitted until this tool result exists in the transcript above it.

This step is not skippable by predicting the result. The only valid basis for proceeding to Step 2 or Step 3 is the actual tool result from this call. "I already know nn won't have this" is not a valid basis — the search must run and return.

After the tool result is visible, quote one specific result title or write "zero results returned." Then answer:
> "What specific claim in the search results, if any, covers the question the external action would answer?"

This question is only answerable by reading the actual results. If the answer requires no inspection of the results, the question has not been engaged with.

**Step 2 — Inspect existing notes (only if results were returned)**

Run `nn show <id>` on the most relevant results. Answer:
> "Does note `<id>` cover the specific question the external action would answer, and if so, which sentence covers it?"

If yes: skip the external action and link or update the existing note instead. If no: proceed to Step 3.

If zero results were returned in Step 1, skip Step 2 and proceed directly to Step 3.

**Step 3 — Do the external action**

Proceed with the fetch, search, file read, or agent spawn — scoped to the topic named in the Step 1 search query. After the action completes, write one sentence stating what the action returned and confirming it falls within that scope.

**Step 4 — Capture or skip (required; based on what the external action returned)**

`nn-capture-skip` is only permitted when ALL of the following hold:
- The external action from Step 3 has completed and its result is visible in the transcript.
- The result was read and a specific claim from it has been named.
- That claim was evaluated against the durability question: "Would this change behavior in a future session with no memory of this one?"
- The answer to the durability question is no, and the reason is stated.

If all four conditions hold: write `Skipping: <named claim> from <source> — <durability reason>.` then run `echo "nn-capture-skip: <that sentence verbatim>"`.

`nn-capture-skip` is not permitted for any other reason — including "no nn results", "result was too long", or "already captured elsewhere" without citing a specific note ID.

If the durability question answers yes: write `Capturing: <finding> from <source>.` then run one of:
- `nn new --title "..." --type <type> --content "..." --no-edit`
- `nn update <id> --append "..." --no-edit`
- `nn link <from> <to> --annotation "..." --type <type>`

## nn commands used

```
nn list --search "<topic>" --json
nn show <id>
nn new --title "..." --type <type> --content "..." --no-edit
nn update <id> [--append "..." | --content "..."] --no-edit
nn link <from> <to> --annotation "..." --type <type>
```

## Success criteria

- Every external action has a `nn list --search` tool result above it in the transcript
- Step 1 answer names a specific result title or states "zero results returned" — not a prediction
- Step 2 answer names the specific sentence in a note that covers the question — not a label
- Step 3 external action result is visible before any capture/skip decision
- Every `nn-capture-skip` names: the specific claim read, the source, and the durability reason
- Every `nn new` / `nn update` / `nn link` is preceded by a `Capturing:` sentence
