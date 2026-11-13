# ADR-0072: Move weekly review into a dedicated skill with tiered autonomy

**Status:** Accepted
**Date:** 2026-11-13

## Context

Weekly review is currently the `--weekly` mode of `nn-session-debrief`. Its step 5 reports stale drafts and expired notes but ends each disposition with "surface" or "propose", so a model following it literally lists candidates and stops. The expired-note sweep also reads, searches, appends and deletes one note at a time, which exhausts context on a large backlog before the count reaches zero. Step 5 has no completion artifact and no terminal state, and the weekly instructions sit inside a session-oriented skill whose steps 1-4 do not apply to a notebook-wide pass.

Users differ in how much deletion they will delegate. Because every operation is one Git commit, deletion is recoverable, which makes unattended cleanup affordable for users who want it, but it must not be the shared default.

## Decision

Add an embedded skill, `nn-weekly-review`, and make `nn-session-debrief --weekly` delegate to it.

The skill defines two modes. The default classifies each bucket and asks once per bucket, not once per note. `--auto` runs without confirmation.

Cleanup is tiered. Cheap cases (expired `daily` notes, and situational notes with no backlinks and no durable principle) are deleted in bulk by one filtered command. Notes carrying a durable principle are batched into append-or-new before deletion. Stale drafts follow the existing link-count rule: no links in either direction is deletable, outbound links only is linked or promoted.

Termination is observable, not descriptive. A loop re-runs `nn list --expired --count` (or the bucket's own filtered count) after each page and stops when the count is `0`, when a full pass leaves the count unchanged, when the page budget is spent, or when the user says stop. Items needing judgment accumulate in a residual list instead of interrupting.

Guards hold in every mode. `protocol` and `permanent` notes are never deleted automatically, and a note with backlinks is not deleted until its key claim is inlined into each dependent. Every run ends with a receipt of before and after counts, deleted IDs and the residual list.

Which mode a user gets by default is a user policy, expressed in a global protocol note, so the shared skill stays conservative.

## Consequences

- `--weekly` in `nn-session-debrief` shrinks to a pointer, removing duplicated sweep logic.
- Skills are embedded with `//go:embed nn-*`, so no Go registration change is needed.
- The loop is resumable: its state is the filtered query count, so a context reset loses nothing.
- A user who sets `--auto` as their default accepts that deletions happen without review. They remain recoverable through Git history.
