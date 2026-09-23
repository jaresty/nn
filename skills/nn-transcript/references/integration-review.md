---
name: integration-review
description: Compare ongoing work with an overall goal and accepted plan, identify the first missing integration edge, and recommend a bounded course correction.
applies_when: "When assessing progress toward an overall deliverable, comparing cumulative work with an accepted plan or completion path, diagnosing integration or process blockers, or deciding whether to continue, course-correct, pause, or replan."
---

# Overall-goal integration review

Use this reference to judge whether ongoing work is converging on the stated overall deliverable. It owns semantic trajectory assessment, not transcript transport, worker steering, implementation, acceptance, or qualification.

Load only the existing owners needed by the question:

- **investigate** for question-shaped evidence selection;
- **interaction** for exact targets, scope, Back, and Refresh;
- **context** for one worker's assignment-versus-recent-work evidence;
- **review** for bounded multi-room or unclosed-handoff populations;
- **events** for attributable event and payload evidence;
- **handoffs** for launch, return, and lifecycle interpretation;
- **actions** before recommending next actions or creating an integration receipt.

Those references own their native commands, paging, snapshots, segmentation, target precedence, and receipt semantics. Do not duplicate or weaken their contracts here. Retrieve every required page and ordered segment before claims that depend on omitted content.

## Required inputs

Extract from the human's request and inspected evidence, or report as unknown:

1. the overall goal or user-visible outcome;
2. the accepted plan, ADR, architecture, or milestone sequence;
3. the relevant project, workspace, and transcript scope;
4. completion, qualification, compatibility, custody, and claim-ceiling constraints;
5. explicit non-goals and work that is not yet authorized.

Do not infer completion criteria from activity. If only a local assignment is available, state that overall trajectory cannot be decided until the broader goal or plan is supplied or located.

## Completion path

Before judging current activity, render the smallest ordered path from the current inputs to the overall outcome:

```text
current inputs
→ required integration stages
→ verification and qualification gates
→ user-visible outcome
```

Classify each stage as exactly one of:

- `NOT_STARTED` — no inspected evidence of implementation;
- `LOCAL_ONLY` — implemented or tested in isolation;
- `CONNECTED` — an inspected composition path reaches its predecessor or successor;
- `VERIFIED` — the connected behavior passed an applicable objective check;
- `QUALIFIED` — the required independent acceptance or qualification contract passed;
- `ENABLED` — the intended user-facing path is authorized and available;
- `UNKNOWN` — evidence is missing, unavailable, ambiguous, or outside inspected scope.

Do not assign percentages unless the request supplies a weighted rubric. If a directional estimate is useful, label it as judgment and explain why implementation progress differs from qualification or production readiness.

## Evidence procedure

1. Resolve the exact conversation, workers, project, and repository scope through **investigate** and **interaction**.
2. Inspect assignments and recent work through **context**, **review**, or assignment-inclusive **events**, according to whether the question concerns one worker or several.
3. Use **handoffs** before interpreting launches, returns, or lifecycle state.
4. Inspect repository, test, build, or runtime evidence only through an appropriate available tool. Transcript text reporting a command is an agent report until its attributable result is inspected; a branch diff or summary identifies what to inspect but does not substitute for it.
5. Compare each active or completed increment with the completion path:
   - Which exact edge did it establish?
   - Is the result isolated, connected, verified, qualified, or enabled?
   - What bounded claim is newly supported?
   - Did it preserve compatibility and claim ceilings?
   - Is current work attacking the first missing edge?
6. Stop when the trajectory decision and first blocker are supported. Do not read every room merely because it exists.

Preserve source qualifications: producer completion is not task success; missing return is not proof of activity; focused tests are not end-to-end qualification; static source is not runtime execution; bounded evidence is not completeness; commit count and token volume are not progress measures.

## Process diagnoses

Use these labels only when inspected evidence supports them:

- **LOCAL_OPTIMIZATION_LOOP** — repeated hardening of one component while the critical path remains disconnected;
- **INTEGRATION_GAP** — individually tested components lack an evidenced composition path;
- **PLAN_DRIFT** — current work no longer advances the accepted milestone sequence;
- **PREMATURE_EXPOSURE** — public interfaces or migration begin before required admission or qualification gates;
- **FALSE_COMPLETION** — fixtures, commits, or producer completion are treated as overall readiness;
- **COMPATIBILITY_EROSION** — additive work changes historical defaults, bytes, selectors, or behavior;
- **QUALIFICATION_SUBSTITUTION** — focused evidence or design review is treated as formal qualification;
- **EVIDENCE_INFLATION** — evidence is upgraded beyond its custody, authority, runtime, or completeness contract;
- **CORRECTION_CHURN** — repeated correction cycles reveal a missing invariant that should become one persistent guard;
- **OVER_BROAD_MILESTONE** — one phase combines several independently gated edges;
- **STALE_PLAN** — evidence falsifies an assumption underlying the accepted sequence;
- **BLOCKED_HANDOFF** — the next owner lacks an exact seam, invariant, RED condition, or stop boundary.

Do not diagnose drift merely because careful investigation is lengthy. Distinguish planned RED tests from unexpected failures, legitimate correction from unresolved repetition, and producer completion from successful parent integration.

## Decision

Return exactly one trajectory recommendation:

- `CONTINUE` — current work advances the first missing edge with a bounded stopping condition and preserved contracts;
- `COURSE_CORRECT` — useful work exists, but the next phase should change to restore the critical path;
- `PAUSE` — a missing contract, authority decision, source of truth, or qualification criterion makes further implementation speculative;
- `REPLAN` — inspected evidence falsifies the accepted completion path or architecture.

Every recommendation must name:

1. the inspected evidence supporting it;
2. the first missing integration edge;
3. the smallest next milestone;
4. the observable evidence required to call that milestone complete;
5. work that must not happen yet.

Recommendations are proposals, not authenticated steering or acceptance. Do not send, stop, resume, or redirect a worker. Load **actions** before proposing notebook mutation or after consequentially adopting, partially adopting, or rejecting substantive delegated work; integration receipts remain owned by **actions** and the parent-side adjudicator.

## Required report

Lead with the useful answer, then provide concise Markdown:

1. **Overall trajectory** — `ON_TRACK`, `AT_RISK`, `BLOCKED`, or `UNKNOWN`, with a one-sentence evidence basis.
2. **Completion path** — ordered stages with the classifications above.
3. **Verified progress** — only inspected, source-qualified claims.
4. **First missing edge** — the critical blocker, not every open issue.
5. **Process diagnosis** — healthy correction or one or more supported labels.
6. **Recommendation** — `CONTINUE`, `COURSE_CORRECT`, `PAUSE`, or `REPLAN`.
7. **Next checkpoint** — one bounded milestone and its exact completion evidence.
8. **Do not do yet** — work that would skip gates or distract from integration.
9. **Evidence limits** — unavailable transcript detail, omitted history, bounded source coverage, unexecuted tests, or other unknowns.

When no correction is needed, say so directly. When progress is real but far from the overall goal, state both without conflating them.
