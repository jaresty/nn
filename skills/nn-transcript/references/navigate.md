---
name: navigate
applies_when: "When entering a session to descend one run — tree overview, :enter a thread, discover per-thread dimensions, apply a lens, or run the opt-in whole-tree Tier-2 sweep."
---

# nn-transcript / navigate — one session

Owning reference for `[enter a session]`. Fetch before descending into a session. Entered with a
`session id` in carried state. Descend, discover per-thread dimensions, offer lenses. The visual
grammar and discovery contract live in the core (`nn skills get nn-transcript`) — this reference
does not restate them.

## Steps

1. **Tree overview (deterministic, trustworthy).**
   ```bash
   nn transcript tree <session> --json
   ```
   Read only Tier-0/1 signals — always present, no inference:
   - `cost` / `subtree_cost` — which branch spent the tokens (*audit*).
   - `status` — active vs completed.
   - `hierarchy` + `lifespans` — who spawned what, when (*recover context*).

   Draw the spawn DAG with box-drawing branches, marking each node by type and cost per the core
   grammar; emphasize the costly subtree, collapse boring repetition. **Do not infer Tier-2
   signals (drift, groundedness, pivots) across the whole tree here** — that is deferred to
   `:enter` or the opt-in sweep.

2. **`:enter` one thread — pay inference, scoped to this thread.**
   ```bash
   nn transcript show <session> <agent-id> --json             # page 1
   nn transcript show <session> <agent-id> --json \
     --page <next_page> --snapshot <snapshot>                 # every later page
   # add --raw consistently to every call for the full per-agent record
   ```
   Retrieve every page under the page-1 snapshot, concatenate `segments[].text` by global
   `segment` ordinal, and verify all `segments` ordinals are present before interpreting events.
   Never mix snapshots or make event-derived claims from a partial page set. The reconstruction is
   exactly the legacy text `show` projection for the selected meaningful/raw mode.
   The snapshot binds the request and projected output, not original source bytes. JSON decoding
   may replace malformed source UTF-8 with U+FFFD; JSON pagination rejects invalid projected UTF-8.
   Resolved Pi sidechain events require an explicit matching `agentId`, including in `--raw` mode;
   foreign or missing-owner records are not attributable detail. If none match, metadata fallback
   means event detail is unavailable, not that the agent did no work.

   Answer one question: **what is worth attending to in THIS thread?** Read the events and
   propose **2–4** salient dimensions, drawing from this palette or naming a novel one the thread
   makes salient:
   - **instruction-drift** — did it do what its spawn prompt asked?
   - **context-re-derivation** — did it waste turns rediscovering already-known context?
   - **groundedness** — are claims backed by tool results, or asserted?
   - **pivots** — where did it change direction?
   - **friction** — retries, denials, backtracks.

   **Respect the hard boundary:** you MAY propose *appearance* (state on the node) and *emphasis*
   (which lens). You may NEVER touch *position* (geography) or *connection* (edge types) — those
   belong to the spine. If a thread seems to need a new position dimension, that is a request to
   change the base geography — surface it explicitly to the human, never apply it silently.

3. **Draw the `:enter` dimension diagram** — the entered node with its 2–4 dimensions arranged
   spatially and visually treated (core grammar: color = emphasis/tension, width = cost,
   shape/label = type). Interpretation-bearing and different each time — draw *this* thread to
   surface *what matters here*.

4. **Offer a lens** (emphasis only — a lens changes *what is emphasized*, never node position):
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

Human enters agent `a7665a` of session `sdk-2026-08-10`. `tree --json` shows it is a cheap leaf
under a heavy main thread. `show` reveals real URLs and a node snippet, no prompt deviation.
Proposed dimensions, drawn:

```
        ┌─────────────────────────────┐
   🟦   │  a7665a  ·  nn-capture       │
  spawn │  purpose: browser-demo shots │
   ↑    └─────────────────────────────┘
        cost  ██▏ $24.5k
        🟢 groundedness: high (real URLs, node snippet)
        🟢 instruction-drift: none (did the screenshot task)
        🟡 friction: one interrupted tool call
```

Offer the debug lens → nothing lights. Offer the harvest lens → capture "browser-demo screenshots
are grounded in live URLs" with provenance `sdk-2026-08-10/a7665a`. **Return to the picker.**
