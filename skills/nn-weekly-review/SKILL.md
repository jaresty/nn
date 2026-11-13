---
name: nn-weekly-review
description: Use when running the recurring notebook cleanup — sweeping expired notes, stale drafts, orphans and long notes — or when asked for a weekly review or to clear the backlog. Load with `nn skills get nn-weekly-review`.
when_to_use: Weekly, or whenever expired or stale notes pile up. Invoke as the weekly review; add --auto to run the cleanup loop without confirmation.
---

# nn-weekly-review

Clear the notebook's expired and stale backlog until it is done, not until the first report. Session capture belongs to `nn-session-debrief`; this skill covers the notebook-wide pass.

## Modes

- **Default** — classify each bucket, then confirm once per bucket (not once per note), then apply and continue.
- **`--auto`** — run the loop below with no confirmation. A user's default for this mode is a user policy kept in a global protocol note; the skill does not assume it.

Both modes use the same triage, loop and guards. Only the confirmation step differs.

## 1. Measure

Report each bucket as its exact filtered query plus its count before touching anything. Keep the query unchanged for the rest of the run; never replace a selected set with a broader inventory.

```
nn review
nn list --expired --count
nn list --expired --tag daily --count
nn list --older-than 14 --status draft --count
nn list --orphan --count
```

## 2. Triage tiers

Handle cheap cases in bulk and spend per-note effort only where it changes the outcome.

| Tier | Selector | Action |
| --- | --- | --- |
| Cheap | expired notes tagged `daily` | delete the whole filtered set |
| Cheap | expired notes with `backlink_count` 0 and no durable principle | delete in bulk |
| Principle | expired notes whose body holds an abstract, reusable claim | batch: search for an existing home, `nn update <id> --append "..." --since <modified>` or `nn new --no-edit`, then delete |
| Stale draft | no links in either direction | delete |
| Stale draft | outbound links, coherent body | `nn promote` |
| Stale draft | outbound links, no inbound | link it with `nn link`; do not delete |
| Residual | anything needing judgment | add to the residual list; do not interrupt the loop |

Select bodies for the Principle tier with `nn graph bodies --ids ID1,ID2,... --page 1` and every continuation page, never by broadening the traversal.

## 3. Loop

For each bucket, repeat until a stop condition holds:

1. Take the next page of the exact bucket query (`--json --envelope --fields id,title,modified,type,tags,backlink_count --page N --snapshot <sha256-from-page-1>`).
2. Classify every note on the page with the triage table.
3. Apply the tier actions. In default mode, confirm once per bucket on the first page.
4. Re-run the bucket's count, for example `nn list --expired --count`.

**Stop when any of these holds:**

- the count command prints `0`;
- no-progress — a full pass over the bucket leaves the count unchanged, so what remains is residual;
- the page budget is spent (default 20 pages per bucket, or the budget the user set);
- the user says stop.

A stop condition is a command result or a user message, never your own judgment that the work "feels done". The loop's only state is the bucket count, so a restart after a context reset recomputes it and continues.

## Guards

These hold in every mode, including `--auto`.

- Never auto-delete a note of type `protocol`, a note with status `permanent`, or a note tagged `reminder`.
- Never delete a note with backlinks until you inline the key claim into each note that depends on it.
- Never drop a durable principle: absorb it (Principle tier) before deleting its source.
- Deletion is `nn delete <id> --confirm`, one Git commit each, so every deletion is recoverable from history.

## Structural checks (optional)

The two stale-draft structural buckets are expressible as `nn-rule` fences and queryable with `nn rules query`, complementing `nn list --orphan` and `--no-inbound`:

```nn-rule
has_out(N) :- link(N, _, _).
has_in(N) :- link(_, N, _).
orphan_draft(N) :- note(N, _, "draft"), !has_out(N), !has_in(N).
dead_end_draft(N) :- note(N, _, "draft"), has_out(N), !has_in(N).
```

`nn rules query orphan_draft` lists deletion candidates and `nn rules query dead_end_draft` lists link-it candidates.

Review the week's friction notes with `nn list --search "friction" --status draft --json`. If a notebook-health check was run by hand more than once, propose promoting it to a built-in or shared `nn-rule` so `nn rules check` surfaces it automatically. If the check needs facts the engine does not expose, keep it as a `rules-engine`-tagged feature-request note.

## Receipt

End every run, including a stopped or budget-limited one, with a `## Receipt` containing:

- before and after counts for each bucket, with the stop condition that ended it;
- deleted IDs, grouped by tier;
- notes promoted or linked;
- the residual list, each entry with the reason it needs judgment.

If a bucket ended on the budget or a user stop, say so and give the command to resume it.

## Commands

```
nn review                                          # health report
nn list --expired --count                          # backlog cardinality
nn list --expired --tag daily --count              # semantic slice
nn list --older-than 14 --status draft --rich --json
nn graph bodies --ids ID1,ID2 --page 1
nn delete <id> --confirm
nn promote <id>
```
