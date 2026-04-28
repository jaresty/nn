---
name: nn-session-debrief
description: End-of-session review — surface what was captured, flag un-promoted drafts with new links, and propose a session summary note.
when_to_use: At the end of a working session to consolidate captures, assess promotion eligibility, and record what was learned. Invoke with /nn-session-debrief.
---

# nn-session-debrief

End-of-session review: surface what was captured, flag un-promoted drafts with new links, and propose a session summary note.

## When to use

Invoke at the end of a working session to:
- Review what was captured
- Identify draft notes that now have enough inbound links to be promoted
- Propose a session summary note linking the session's captures

## Workflow

### 1. Review captures

List recently created notes to see what was captured this session:

```
nn list --sort created --status draft --json | head -20
```

For each captured note, run `nn show <id>` to verify the content is accurate and the title is precise.

### 2. Find promotion candidates

Find draft notes that have acquired inbound links (suggesting others have referenced them):

```
nn list --type observation --status draft --json
nn list --type concept --status draft --json
```

For each, check inbound links:

```
nn backlinks <id> --json
```

If a draft has reviewed inbound links and a focused body, propose:

```
nn promote <id> --to reviewed
```

### 3. Run nn-refine on key captures

For each significant capture from this session, invoke the `nn-refine` workflow to check atomicity, links, and title quality.

### 4. Propose session summary note

Create a session summary note linking the key captures:

```
nn new --title "Session: <date> — <topic>" --type observation --content "## What was captured\n\n..." --no-edit
nn bulk-link <summary-id> \
  --to <id1> --annotation "captured this session" --type source-of \
  --to <id2> --annotation "captured this session" --type source-of
```

## nn commands used

```
nn list --sort created --status draft --json
nn show <id>
nn backlinks <id> --json
nn promote <id> --to reviewed|permanent
nn new --title "..." --type observation --content "..." --no-edit
nn bulk-link <from> --to <id> --annotation "..." --type TYPE [--to <id> ...]
nn review [--format json]
```

## Success criteria

- All captures from the session have been reviewed for accuracy
- Promotion-eligible drafts have been promoted or a reason given for deferral
- A session summary note exists linking the key captures, or a reason given for skipping it
