---
name: navigate
applies_when: "When entering a session to descend one run — tree overview, :enter a thread, discover per-thread dimensions, apply a lens, or run the opt-in whole-tree Tier-2 sweep."
---

# nn-transcript / navigate — one session

Owning reference for `[enter a session]`. Fetch before descending into a session. Entered with a
`session id` in carried state. Descend, discover per-thread dimensions, offer lenses. The visual
grammar and discovery contract live in the core (`nn skills get nn-transcript`) — this reference
does not restate them.

## Timing and failure inspection

For delay questions, prefer `nn transcript events <session> <agent-id> --summary timing` before
custom arithmetic. It reports observed message gaps and uniquely matched tool intervals, **not execution time**
or inferred retries. Keep missing/invalid and negative clocks visible; native message and record clocks differ.
Use `events ... --errors-only` for recorded failures (independent of payload visibility), then an inclusive
`events ... --since <RFC3339> --until <RFC3339> --payload` window around the event. Preserve original IDs and
ordinals; retrieve every page/segment with identical filters and the page-1 snapshot. The query receipt
reports excluded unknown timestamps and binds out-of-window evidence too. Complete window transport is
not complete historical source evidence. See the core for exact fields, limits, and incompatibilities.

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
   - For Pi, interpret `evidence_scope` and `terminal_record_count` under the core's lifecycle
     contract before combining timestamps with usage. Cumulative usage and last-run time are
     not interchangeable scopes.

   Draw the spawn DAG with box-drawing branches, marking each node by type and cost per the core
   grammar; emphasize the costly subtree, collapse boring repetition. **Do not infer Tier-2
   signals (drift, groundedness, pivots) across the whole tree here** — that is deferred to
   `:enter` or the opt-in sweep.

For an already selected agent's metadata, use `nn transcript tree <session> --agent <id> --json`
and optional `--fields` under the core projection contract; no client-side row extraction is needed.

2. **`:enter` one thread — pay inference, scoped to this thread.**
   ```bash
   nn transcript show <session> <agent-id> --json             # page 1
   nn transcript show <session> <agent-id> --json \
     --page <next_page> --snapshot <snapshot>                 # every later page
   # add --raw consistently to every call for schema-native per-agent detail
   ```
   For an explicit unbounded JSON export, use `show --all --json` (complete text) or
   `events --all` (complete event objects) under the core export contract. Otherwise:
   Retrieve every page under the page-1 snapshot, concatenate `segments[].text` by global
   `segment` ordinal, and verify all `segments` ordinals are present before interpreting events.
   Never mix snapshots or make event-derived claims from a partial page set. The reconstruction is
   exactly the legacy text `show` projection for the selected meaningful/raw mode.
   The snapshot binds the request and projected output, not original source bytes. JSON decoding
   may replace malformed source UTF-8 with U+FFFD; JSON pagination rejects invalid projected UTF-8.
   Resolved Pi sidechain events require an explicit matching `agentId`, including in `--raw` mode;
   foreign or missing-owner records are not attributable detail. If none match, metadata fallback
   means event detail is unavailable, not that the agent did no work.
   For Pi, raw detail contains complete owned message payloads, not outer JSONL wrappers.
   Meaningful show and search omit tool-result roles and typed tool-result blocks; use `--raw`
   when inspecting or locating tool-result errors or payloads.

   Answer one question: **what is worth attending to in THIS thread?** Read the events and
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
`nn transcript events <session> <agent-id> --summary usage --bucket-size 10` under the core summary
contract, rather than writing another aggregation. It covers the complete selected ledger in one
bounded result and discloses missing and zero records; bucket boundaries are not inferred task phases.

For tool counts, result sizes, or what enlarged the thread, use
`nn transcript events <session> <agent-id> --summary tools --limit 8 --group-by tool` under the core
tool-volume contract. Use returned joins and command previews instead of client-side ranking and lookup.
Sizes are not token attribution; judging necessity still requires inspecting relevant evidence.

For custom per-record analysis or lifecycle accounting, use
`nn transcript events <session> <agent-id> --select identity,message,usage,tools,lifecycle --json`
under the core ledger contract. Complete the page set before summing message usage; extracted tool
events do not carry usage. Inspect a selected event with `--event <event-id> --payload` rather than
fetching every native payload. Size measurements do not establish token attribution or wasted work.

4. **Offer a lens** (change emphasis within the declared layout; explain any warranted remapping
   under the core contract, and never alter spawn topology):
   - **debug** — errors, friction, drift, pivots.
   - **audit** — subtree_cost, tools, re-derivation.
   - **harvest** — notes-touched, groundedness, pivots.
   - **recover** — pivots, joins, lifespan ordering.

5. **Return to the core picker** (loop invariant — never terminate the branch on its own).

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
