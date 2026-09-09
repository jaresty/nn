---
name: context
applies_when: "Before retrieving assignment-versus-recent-work evidence or preparing an instruction-alignment check or course-correction draft."
---

# Assignment context

```bash
nn transcript context <session> <agent-id> --last 5 --json
nn transcript context <session> <agent-id> --last 5 --json --page <next_page> --snapshot <snapshot>
```

Use the carried canonical session path and exact agent ID. This deterministic Pi-only command returns
recorded parent launch assignments and bounded owned recent events. JSON is required; payloads are
included. Last defaults to 5 and accepts 1–200. Unknown agents and unsupported schemas fail explicitly.

The first record has `kind: context_receipt`, with path, last, launch receipt, recent query receipt,
source digests, detail status, `steering_status: unavailable`, and `governing_attempt: not_inferred`.
Launch events have `section: launch`; recent events have `section: recent`. Source event IDs and
independently numbered launch occurrences remain unchanged. Exactly joined invocation payloads retain
the original assignment; ambiguous/missing invocation joins remain qualified, not guessed. A room
with no recorded launch is valid unavailable assignment evidence, not permission to infer its task.

Use the existing **events** lossless transport: every page is at most 48,000 bytes, and oversized records
carry ordered segments. Fetch every page with unchanged options and the first snapshot; reconstruct
before interpretation. Metadata itself can segment. Preserve stream order rather than globally sorting
room-local ordinals. Source changes during collection or between pages reject the bundle. Snapshot
custody concerns retained evidence, not source completeness, current process liveness, or task success.

No launch is selected as governing an inferred attempt. All retained launch assignments are paginated;
only recent events are last-N bounded. Initial steering authority is explicitly unavailable; a user message
in the recent tail is not automatically an authenticated steering instruction. Load **handoffs** before
interpreting launch/return numbering. If instruction alignment depends on unavailable steering, report
that uncertainty instead of declaring drift.

The LLM may compare assignments with inspected recent work and propose a correction, but must cite
source event IDs, disclose omitted history, and distinguish proposals from authorized changes. The
command never calls an LLM, sends a correction, steers a worker, or executes transcript contents.
