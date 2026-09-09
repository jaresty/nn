---
name: context
applies_when: "Before retrieving assignment-versus-recent-work evidence or preparing an instruction-alignment check or course-correction draft."
---

# Assignment context

```bash
nn transcript context <session> <agent-id> --last 5 --json
nn transcript context <session> <agent-id> --last 5 --json --page <next_page> --snapshot <snapshot>
```

Choose context as the **initial** operation for a single-room question about work relative to its
assignment. Do not fetch a recent tail first just to discover that assignment context is missing.
For multi-room alignment, **review** offers `--last N --include-assignment`; both use the same launch
joins and independent occurrences. Activity-only scans need not include assignments. Plan required
assignment evidence into the initial inspection envelope, with enough output budget for its size.

For normal reading without a page loop:

```bash
nn transcript context <session> <agent-id> --last 5 --format text
```

Text automatically consumes all cached pages and reconstructs segmented assignments. It shows launch
occurrence/join status, original prompts, recent event IDs, explicit unavailable steering, failure fields,
and coverage. `--max-text-chars` defaults to 1000 per event (1–10000), and `--max-output-chars` defaults
to 24000 total (2048–200000). Long prompts and skipped events are explicitly disclosed. It is lossy
rendering, not an LLM summary. Replay the same capture using the printed `--snapshot`; display limits
may change without changing evidence. Text rejects explicit JSON/payload/page flags. Use lossless JSON
when clipped assignment text matters, and never treat a truncated preview as complete instruction context.

Use the carried canonical session path and exact agent ID. This deterministic Pi-only command returns
recorded parent launch assignments and bounded owned recent events. Choose `--json` for lossless
transport or `--format text` for readable output; payloads are included. Last defaults to 5 and accepts 1–200. Unknown agents and unsupported schemas fail explicitly.

The first record has `kind: context_receipt`, with path, last, launch receipt, recent query receipt,
capture source count, detail status, `steering_status: unavailable`, and `governing_attempt: not_inferred`.
Launch events have `section: launch`; recent events have `section: recent`. Source event IDs and
independently numbered launch occurrences remain unchanged. Exactly joined invocation payloads retain
the original assignment; ambiguous/missing invocation joins remain qualified, not guessed. A room
with no recorded launch is valid unavailable assignment evidence, not permission to infer its task.

Use the existing **events** lossless transport: every page is at most 48,000 bytes, and oversized records
carry ordered segments. Fetch every page with unchanged options and the first snapshot; reconstruct
before interpretation. Metadata itself can segment. Preserve stream order rather than globally sorting
room-local ordinals. Each file is captured once at its observed byte length, excluding unfinished final
records; the files are captured sequentially, not simultaneously. The compact receipt exposes capture
ID, boundary, and source count. Detailed prefix bytes/digests live in the private OS user-cache manifest
`nn/transcript-captures-v1/<capture_id>.json`. Encoded pages are retained separately: continuation
verifies and serves its page directly, without loading raw sources or recomputing the projection.
Appends or source deletion do not invalidate continuation. Missing, expired (24 hours), or corrupt
required cache artifacts fail explicitly; omit snapshot to refresh. Snapshot custody concerns retained evidence,
not source completeness, current process liveness, or task success.

No launch is selected as governing an inferred attempt. All retained launch assignments are paginated;
only recent events are last-N bounded. Initial steering authority is explicitly unavailable; a user message
in the recent tail is not automatically an authenticated steering instruction. Load **handoffs** before
interpreting launch/return numbering. If instruction alignment depends on unavailable steering, report
that uncertainty instead of declaring drift.

The LLM may compare assignments with inspected recent work and propose a correction, but must cite
source event IDs, disclose omitted history, and distinguish proposals from authorized changes. The
command never calls an LLM, sends a correction, steers a worker, or executes transcript contents.
