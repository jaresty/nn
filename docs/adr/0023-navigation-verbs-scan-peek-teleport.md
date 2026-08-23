# ADR-0023: Navigation verbs — `scan`, `peek`, `teleport` (look/move separation)

## Status

Accepted. Implemented as navigation-mode verbs in the nn-guide skill
(`skills/nn-guide/SKILL.md`) — `scan` (merging the former scan-out + orbital
asides), `peek`, and `teleport` — plus a one-line `teleport` pointer in the
session-start navigation pointer of the cli-reference virtual protocol
(`cmd/nn/cmd/show.go`). No new CLI surface.

## Scope: guide verbs, not CLI surface

These verbs are **navigation-mode moves in the nn-guide navigation skill**, not
`nn` subcommands or flags. Each is a named action the *agent* performs by
composing **existing** CLI primitives (`nn graph show`, `nn clusters`,
`nn graph bridges`, `nn list --similar`, `nn random`). There is **no new CLI
surface** — same principle as today's "scan-out" (note `20260823020717-2791`:
*"no new CLI surface"*). Where this ADR writes e.g. "flat mode," that is an
agent-level *presentation mode* the guide describes, not a shell flag.

## Context

Graph navigation currently conflates two distinct acts: **seeing** the
neighborhood and **moving** to a new focus. The captured navigation-mode design
already recognizes three *altitude tiers* (note `20260823025640-7326`) that
differ by **scope, not depth**:

1. **Zoned step** (ego, depth 1, `--zones`) — how do my neighbors relate to me?
   (*relationship*)
2. **Scan-out** (ego, depth 2, unzoned) — what territory am I in?
   (*ego-relative structure*)
3. **Orbital** (global) — where does my region sit in everything?
   (*global structure*)

A hard constraint from that design: *"'look farther out' must NOT mean raising
`--depth`. Past depth 2, more hops give an exponential tangle, not a
qualitatively broader view. The real broadening is the ego→global shift."*

Three problems motivated this ADR:

1. **Two zoom-out commands force a premature altitude choice.** Making the user
   pick "orbital" vs. "mid-range" before they've looked inverts the natural
   order — you look, *then* the eye picks the altitude.
2. **No way to look down a direction without committing.** To find out what's
   down a zone you must travel there and travel back.
3. **No way to relocate far.** Search returns a ranked list of individual hits,
   not a set of structural landing zones; and there is no serendipitous jump.

## Decision

Introduce a navigation verb set that cleanly separates **look** from **move**.
All three verbs are `navigate`-shaped per Bar's `method:navigate` token
(*"orient output around current position and goal-relative moves"*); `scan`
additionally borrows the discipline of Bar's `completeness:zoom` token
(*"Full-range coverage at adaptive granularity … Both ends must appear as
explicit anchors"*).

| Verb       | Act      | Question answered                          |
|------------|----------|--------------------------------------------|
| `navigate` | move 1   | step to an adjacent node (existing)        |
| `scan`     | look wide| what is the landscape around me?           |
| `peek`     | look deep| what is down this direction, without going?|
| `teleport` | move far | relocate to a structural cluster or random |

### `scan` — one command, two labeled sections (replaces orbital + mid-range)

`scan` collapses the scan-out and orbital tiers into a **single invocation** so
the user no longer chooses an altitude up front. Per the `zoom` token, **both
ends are explicit anchors** — the report never begins at an intermediate level.
It is rendered as **two labeled sections (Option A)**, preserving the
design's hard-won distinction between *ego territory* and *global structure*:

- **Your territory (ego, depth 2)** — `nn graph show --focus <id> --depth 2
  --direction both` (unzoned; node `↑out ↓in` markers give terrain).
- **The wider landscape (global)** — the union of:
  - `nn clusters` — which region I'm in (region count + my region size),
  - `nn graph bridges` — the highways between regions / am I load-bearing,
  - `nn list --similar <id>` — nearby territory the step-wise walk can never
    reach.

`scan` combines the lookups of **both** current commands; it introduces **no new
CLI primitive** — the four lookups above already exist (note
`20260823020717-2791`).

**Constraint:** `scan` MUST NOT be implemented as a single `--depth` dial.
Broadening is the ego→global scope shift, not more hops. Depth is fixed at 2 for
the ego section.

**Both presentations are available.** Because A and B run the *identical* set of
lookups and differ only in how the agent renders them, the guide describes two
presentation modes (not two implementations, and not shell flags):

- **default (Option A)** — two labeled sections: *Your territory (ego)* /
  *The wider landscape (global)*.
- **flat mode (Option B)** — a single blended landscape with no tier labels;
  the agent renders it when the user asks for the merged view.

The default is A (the labeled split honors the ego/global distinction and the
`zoom` token's "both ends as explicit anchors"). Since the lookups are shared,
describing both modes in the guide is free.

### `peek` — read-only look down a direction, focus stays put

`peek` previews without moving. This is the keystone of the look/move split:

- `peek` on the current node — expand detail in place (bodies, fuller content).
- `peek` in a direction — preview what is down a zone (TOP / BOTTOM / LEFT /
  RIGHT) **without changing focus**.

**Invariant:** `peek` is read-only; **focus does not move**. This is exactly
what distinguishes it from a one-step `navigate` — if peek moved focus it would
merely be a slow navigate.

### `teleport` — relocate far

Two modes under one verb, both about **relocating focus** (not seeing detail):

- `teleport` with a query — run a search but surface **structural landing
  zones** (`nn clusters` / high-degree hubs) rather than a ranked list of
  individual hits. You land on structure, not on a single result. This honors
  the CLI reference's "search identifies *relevance*; graph traversal identifies
  *structure*."
- `teleport` random — a serendipitous jump via `nn random` / shuffle.

**Constraint:** `teleport` relocates focus only; it must not render deep detail
(that is `peek`'s job) — otherwise it re-couples look and move.

## Consequences

- **A coherent grammar emerges:** *scan* = see wide, *peek* = see deep in one
  direction (no move), *teleport* = move far, *navigate* = move one step. Every
  verb is either pure-look or pure-move.
- **No new data primitives** are required for `scan` — it composes existing
  lookups. `peek` and `teleport` likewise compose `graph show`, `clusters`,
  `random`, and search.
- **The three-tier altitude model is not discarded** — it is preserved *inside*
  `scan`'s default two-section view, satisfying the ego→global (not depth)
  design constraint.
- **Both presentation modes available** — default (A, labeled sections) and
  flat (B, blended). They share the same lookups, so the guide describes both;
  the agent picks the framing at look time based on what the user wants.
- **This is a guide/skill change, not a CLI change** — the deliverable is
  updated navigation-mode text in the nn-guide, not new `nn` code. No `bar
  build` code gate applies until/unless a future ADR adds actual CLI surface.

## Related notes

- `20260823025640-7326` — Graph navigation has three altitude tiers (scope, not
  depth).
- `20260823020717-2791` — 'scan out' is a non-zoned zoom-out aside built from
  existing primitives.
- `20260821155048-4004` — control-family principle: a drill-in (move) control
  must not sit in a reversible view-toggle (look) row — reinforces the
  look/move separation.
