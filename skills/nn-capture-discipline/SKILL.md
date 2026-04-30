---
name: nn-capture-discipline
description: Enforces the research protocol — search nn before going external, then write or skip with a verifiable artifact after the external action completes.
when_to_use: Before any action that introduces new information not already present in the conversation. Invoke with /nn-capture-discipline.
---

# nn-capture-discipline

Enforces the research protocol: search nn for duplicates before going external, do the external action, then capture or skip based on what was found.

## When to use

Invoke before any action that introduces new information not already present in the conversation: spawning an agent, fetching a URL, searching the web, or reading an unfamiliar file.

## Workflow

**Step 1 — Search nn (separate message, before planning the external action)**

Run `nn list --search "<topic>" --json`. This call must be in its own message, completed and returned, before the external action is planned. Planning the search and the external action in the same message collapses the gating relationship — the search result cannot constrain a decision already made.

After the search result is visible in the transcript, answer this question before proceeding:
> "Given what the search returned, is there a meaningful gap that justifies the external action?"

If yes: proceed to Step 2. If no: the external action is not warranted; state why and stop.

**Step 2 — Inspect existing notes (only if results were returned)**

Run `nn show <id>` on the most relevant results. Answer:
> "Does any existing note already cover the specific question the external action would answer?"

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

- Step 1 search is in its own message, completed before the external action is planned
- Step 3 external action result is visible in the transcript before any capture/skip decision
- Every `nn-capture-skip` names: the specific claim read, the source, and the durability reason
- Every `nn new` / `nn update` / `nn link` is preceded by a `Capturing:` sentence
