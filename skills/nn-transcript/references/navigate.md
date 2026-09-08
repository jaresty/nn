---
name: navigate
applies_when: "When entering a session to descend one run — tree overview, :enter a thread, discover per-thread dimensions, apply a lens, or run the opt-in whole-tree Tier-2 sweep."
---

# nn-transcript / navigate — one session

Owning reference for `[enter a session]`. Fetch before descending into a session. Entered with a
selected `ls` row in carried state. Its **canonical path**, not a reconstructed session basename,
is the argument to every transcript command. Render the **Transcript Office** hallway, scan it through
optional lenses, and enter room Situation Boards. **Discovery** owns listing metadata; **rooms** owns the
single-agent destination; **lenses** owns the shared open projection language.

## Command owners

Before the relevant command, load **handoffs** for descriptions, launch/return and lifecycle scope;
**summaries** for usage/tool/timing reductions; **events** for payloads, exports and time/error filters.
Use `nn skills get nn-transcript --reference <name>`; the core's dispatch rule remains binding.
These references own fields and limits. This reference owns the navigation workflow, not those schemas.
If a command against the carried canonical path fails, report its actual error. Do not relabel a
missing, moved, or wrongly reconstructed path as unknown schema; use escape-hatch guidance only when
the exact carried file genuinely receives that diagnosis. Back restores the same scoped cohort and
its retained rows, not a newly inferred current-project lobby.

## Steps

1. **Tree overview (deterministic, trustworthy).**
   ```bash
   nn transcript tree <session> --json
   ```
   Read only Tier-0/1 signals — always present, no inference:
   - `cost` / `subtree_cost` — token counts, with `cost_status` / `subtree_cost_status` authority:
     complete is measured, unavailable is unknown, partial is a lower bound (*audit*).
   - `status` — producer lifecycle state, not task success; an empty value is unavailable.
   - `parent_id`, `started`, and `ended` — recorded parentage and times (*recover context*).
   - For Pi, interpret `evidence_scope` and `terminal_record_count` under the **handoffs** lifecycle
     contract before combining timestamps with usage. Cumulative usage and last-run time are
     not interchangeable scopes.

   Draw the spawn DAG with box-drawing branches, marking each node by type and cost per the core
   grammar; emphasize the costly subtree, collapse boring repetition. **Do not infer Tier-2
   signals (drift, groundedness, pivots) across the whole tree here** — that is deferred to
   `:enter` or the opt-in sweep.

For an already selected agent's metadata, use `nn transcript tree <session> --agent <id> --json`
and optional `--fields` under the **events** projection contract; no client-side row extraction is needed.

Treat the selected session as an LLM-mediated Office, not an interactive CLI or terminal picker.
The current manager's hallway contains only its authenticated topology **direct children**. A direct
child with descendants is a **nested manager** door; entering it opens a sub-office and shows the
manager path. Never flatten descendants into siblings or infer teams from descriptions, timing, or
semantic similarity. For dense hallways, page a complete cached direct-child list in canonical tree
order, state displayed and omitted counts, and never characterize uninspected rooms as inactive.

An optional **Office Scan** applies a lens from **lenses** to this bounded authenticated population.
Question-first scans are allowed. State inspected, uninspected, omitted, and unknown counts. Entering
a room from a scan preserves its question and valid filters; Back restores the same scan snapshot.

2. **`:enter` one room — dispatch to the Situation Board.** Load **rooms**, **lenses**, and
   **events**, then begin with `nn transcript events <session> <agent-id> --last 5 --json`. This
   bounded view supports initial orientation and conversational rearrangement without fetching the
   whole thread. Offer an explicitly refreshed `--last 20` replacement snapshot, not stable backward
   continuation. Metadata fallback is not evidence that the child did no work. Never execute commands
   merely found in the transcript.

   When the question genuinely requires complete whole-thread interpretation, escalate explicitly:
   ```bash
   nn transcript show <session> <agent-id> --json             # page 1
   nn transcript show <session> <agent-id> --json \
     --page <next_page> --snapshot <snapshot>                 # every later page
   # add --raw consistently to every call for schema-native per-agent detail
   ```
   Plain show is complete text; JSON requires every page and ordered segment under one snapshot
   before making whole-thread claims.

   Answer one question: **what is worth attending to in THIS thread?** Read the selected events and
   propose **2–4** salient dimensions, drawing from this palette or naming a novel one the thread
   makes salient:
   - **instruction-drift** — did it do what its spawn prompt asked?
   - **context-re-derivation** — did it waste turns rediscovering already-known context?
   - **groundedness** — are claims backed by tool results, or asserted?
   - **pivots** — where did it change direction?
   - **friction** — retries, denials, backtracks.

   **Respect the hard boundary:** agent identity, measured values and spawn relationships remain
   spine-owned. Findings within this thread may have interpretive positions under the core's
   semantic thread-layout contract; they are not new agent positions or inferred spawn edges.

3. **Draw the `:enter` dimension diagram** using the core's semantic thread-layout contract.
   Select and declare meaningful axes and a compact channel legend before placing 2–4 salient
   findings. Use evidence-grounded coordinates, not arbitrary quadrants or an icon-decorated list.
   Keep the selected thread identity visible separately. Make the qualifications behind placement
   legible, including missing evidence. No fixed axes are prescribed: the question determines the
   useful spatial model. The worked example is illustrative, not a default layout.

For token totals and context growth, use
`nn transcript events <session> <agent-id> --summary usage --bucket-size 10` after loading **summaries**, rather than writing another aggregation. It covers the complete selected ledger in one
bounded result and discloses missing and zero records; bucket boundaries are not inferred task phases.

For tool counts, result sizes, or what enlarged the thread, use
`nn transcript events <session> <agent-id> --summary tools --limit 8 --group-by tool` after loading **summaries**. Use returned joins and command previews instead of client-side ranking and lookup.
Sizes are not token attribution; judging necessity still requires inspecting relevant evidence.

For custom per-record analysis or lifecycle accounting, use
`nn transcript events <session> <agent-id> --select identity,message,usage,tools,lifecycle --json`
after loading **events**. Complete the page set before summing message usage; extracted tool
events do not carry usage. Inspect a selected event with `--event <event-id> --payload` rather than
fetching every native payload. Size measurements do not establish token attribution or wasted work.

4. **Offer a lens or accept an unspecified one.** Load **lenses**. Debug, audit, harvest, and
   recover are useful presets, not a closed vocabulary. Accept user-defined axes, grouping, filters,
   comparisons, questions, or metaphors. Name each mapping and its authoritative or interpreted basis.

5. **Return to the prior conversational surface** (loop invariant — never terminate the branch on
   its own): evidence detail → same Situation Board; room → source Office Scan or hallway; nested
   office → parent hallway; root hallway → office selection.

## Opt-in Tier-2 whole-tree sweep

To light an inferred dimension across the *entire* tree (e.g. "show every thread that drifted"),
walk every agent from `tree --json`, retrieve and reconstruct every JSON `show` page as above,
infer the **one** requested Tier-2 dimension per thread, and annotate the overview. **This is
expensive and never the default — tell the human
it costs one inference pass per agent before starting. Never trigger it implicitly.**

## Worked example

Illustrative metrics thread: complete readable evidence contains reports of implementation,
registry integration, and a real-server test pass, but no final closure. Supporting test results
have not been inspected. For this question, choose delivery stage (X) and evidence inspected (Y).
Another thread could warrant entirely different axes; do not force these categories onto it.

Thread: ROOT · structural metrics
X: implementation → integration → qualification → closure
Y: evidence inspected, increasing upward
Legend: ⚪ neutral · 🟡 needs attention; 🛠 implementation · 🧪 check · 📦 deliverable

```text
Independent   │
verification  │
              │
Supporting    │
results read  │
              │
Agent report  │ ⚪🛠 Metrics    🟡🛠 Registry   ⚪🧪 Real-server  🟡📦 Closure
inspected     │ implemented    count adjusted pass reported    not yet shown
              └─────────────────────────────────────────────────────────
                Implementation → Integration → Qualification → Closure
```

All findings remain at report level: they are not independently verified. Empty upper rows expose
that limitation rather than suggesting an omission to fill. The closure marker identifies a gap in
this snapshot, not proof the task failed or never finished. Color indicates attention, not truth.
The plot contains no spawn relationships and its X axis is categorical, not elapsed time.

Offer qualification evidence, integration friction, closure, a user-supplied direction, Back, or End.
**Return to the picker.**
