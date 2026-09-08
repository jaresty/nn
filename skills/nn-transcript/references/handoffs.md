---
name: handoffs
applies_when: "Before interpreting Pi lifecycle scope, launch descriptions, or parent-side launch/return occurrences."
---

# nn-transcript / handoffs

## Select by recorded description

A launch name is tree metadata. Use `nn transcript ls <root> --json` to select the parent session,
then `nn transcript tree <session> --json` and match `description`; if the parent is known, start at
tree. For agent selection, do not use `nn transcript search`: it searches event content rather than
the authoritative description and may match copied prompts or later discussion. After selecting the
agent ID, use the launch/return commands below.

## Launch descriptions and parent-side handoffs

Pi `tree` nodes expose optional `description`, the latest nonempty recorded launch label in file
order, preserved after terminal records replace provisional state. Structured launch acknowledgment
metadata wins; otherwise use only a uniquely matched Agent invocation's description. No prompt-based
label invention. Text trees show an escaped quoted description beside the ID, type, and status.
`tree --agent <id> --fields id,description,status --json` supports the new field; unknown descriptions
are omitted normally and null when explicitly selected. Other schemas currently have no description
projection. A label never changes authoritative topology, cost, or completion semantics.

For a selected child, target its **parent transcript**, not its own first/last message:

```bash
nn transcript events <parent-session> <agent-id> --at launch --payload
nn transcript events <parent-session> <agent-id> --at return --payload
```

These shortcuts currently support Pi. They read only the supplied canonical parent file—no inferred
sidechain or parent discovery. Launches are structured Agent background acknowledgments; return events
are exact matching `subagents:record` producer terminal records, not inferred text notifications or
proof of task success. Launch invocation matching requires a unique Agent call ID and unique
acknowledgment within the same recorded owner scope. Duplicate IDs are ambiguous, absent candidates
missing, absent IDs unavailable; never use `parentId` sequencing or adjacent text to guess a join.

The normal bounded ledger envelope adds `handoff` with `at`, `status`, `launches`, `returns`,
`occurrences` (all occurrences of the requested kind, before an optional exact-event filter), and
`pairing`. Status is observed, not_observed (other handoff evidence exists), unavailable (known
ROOT/inline child but no handoff evidence), or unsupported_schema. Unknown children in a Pi parent
reject; unsupported/unrecognized schemas explicitly report unsupported_schema. Counts describe
records in this parent file, not authenticated historical completeness or outstanding attempts.

Each entry has kind launch/return, child `agent_id`, native `record_owner`, source provenance,
`event_id`, and independent per-kind `occurrence`. `ordinal` is its position in this child's combined
parent-handoff ledger—not its child-message ledger ordinal. Returns retain the positional event ID
of the corresponding ordinary lifecycle event; launches use a distinct handoff slot. Multiple returns
or resumed launches are never collapsed, paired by ordinal, or silently treated as the latest attempt.

The `lifecycle` facet adds recorded status and launch description/call ID/match status. A matched
invocation includes its exact parent-side tool event ID and source-record provenance. `--payload`
adds `{acknowledgment, invocation}` for launches (invocation null unless uniquely matched), and native
terminal data for returns. Other ordinary facets do not synthesize message/usage/tool projections
for handoff entries. Use `--at launch --event <id>` or `--at return --event <id>` for an exact occurrence.

`--at` rejects summaries and time/error filters, accepts normal facets/payload and exact-event selection,
and uses existing snapshot/page/fragment/--all rules. Retrieve every page and ordered segment before
interpreting the complete batch. Snapshots bind the selected handoff projection/options and receipt;
changed invocation payloads invalidate payload continuation. `--all` is unbounded and has the same
snapshot. No return observed does not establish that a child is currently running. For a missing
parent-side record, use discovery to locate the correct parent rather than substituting child activity.

## Pi lifecycle scope

Pi `tree --json` nodes include `evidence_scope`; other schemas omit this Pi-specific object.
Its `status` is `last_terminal_record`, `background_spawn_record`, or `unavailable` for ROOT.
Its `timestamps` is `last_terminal_record`, `root_message_history`, or `unavailable` for
provisional children. Last means last in file order, not latest by timestamp; missing timestamps
remain empty. Its `cost` is `retained_sidechain_history` after authenticated owned hydration,
`root_message_history` for ROOT, or `unavailable`. Its `subtree_cost` is `subtree_aggregate`:
that total combines the included nodes' own scopes, not a single run window.
`terminal_record_count` counts matching producer records, including duplicates—not distinct attempts.

Do not divide cumulative costs by last-run duration or label those costs as latest-run usage.
Producer `completed` means a terminal status, not task success. Read the complete thread to assess
outcome; never silently rewrite the producer status from prose. These provenance labels neither
certify source completeness nor replace `cost_status` / `subtree_cost_status` authority.
