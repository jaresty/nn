# ADR-0021: `nn ask --surface graph` — the graph viewer as a scoped hosted feedback surface

## Status

Proposed

## Context

ADR-0020 established `nn ask` as a routed human-feedback primitive: the *job* is
"ask a human to close a knowledge gap and capture their structured response"; the
*surface* is a routing decision keyed on the shape of the answer. Two adapter
kinds exist — **hosted** (`nn` owns an ephemeral `localhost` server + embedded
UI, e.g. canvas/Excalidraw) and **delegated** (`nn` invokes a peer binary on a
request-path-in → result-path-out contract, e.g. document/Plannotator).

Separately, `nn` has grown an **interactive graph viewer** (`nn graph export
--format html --serve`) that renders the notebook as a force/grouped/zoned graph.
The recent "zoned" overhaul gave it directional ego layouts (relationships map to
screen zones), depth-1 focus with breadcrumb navigation, node/edge tooltips, a
selection tray, and a "Review & Export" flow.

The shaping observation: **some human-feedback questions are inherently
relational** — "which of these notes actually support the claim?", "is this the
right note to link X to?", "mark the tension in this neighborhood." Those are
badly served by a blank canvas or a prose document; they want the *graph itself*
as the response surface. The viewer already provides scoped exploration and a
selection/annotation affordance — the missing pieces are the ask request/result
envelope and a way for the agent to bound what the human sees.

### Constraints and prior decisions

- **Scope is the load-bearing constraint.** The full notebook is hundreds of
  nodes; an unbounded graph surface drowns the human ("you get lost in the whole
  graph"). The agent **must** supply a scope. The depth-1 ego subgraph around a
  `--focus` node — the constraint mechanism the zoned viewer already implements —
  is exactly this bound. Without agent-supplied scoping the surface does not work.
- **Hosted adapter, one-shot.** Like canvas, this is a hosted surface: `nn` owns
  an ephemeral server for the duration of one blocking call. It is **routed-ask**
  (prepare → submit → result at a known path), *not* a live conversational loop.
  A streaming loop where the LLM interactively re-focuses the human falls outside
  routed-ask (noted in the design concept as needing a second interaction mode)
  and is **deferred**.
- **Thin result envelope; the LLM interprets** (inherited from ADR-0020). The
  result names native artifacts (selected node ids, per-node annotations, a
  free-text answer) by path; nothing is filed automatically. What becomes a note,
  an `nn link`, a `graph apply` changeset, or nothing, is a downstream agent
  decision.
- **Reuse, don't rebuild.** The viewer's serve mode already has the relay hooks
  (`/event` node-click POSTs, `/messages` poll) and the selection tray + export.
  The graph surface reuses these for the one-shot submit rather than inventing a
  new UI.

## Decision

Add a **`graph`** surface to `nn ask`, as a **hosted adapter** parallel to
`canvas`:

```
  agent ── "react to this neighborhood" ──► nn ask --surface graph --focus <id>
                                                   │
                                                   ▼
                          prepare request (question + focus id + allowed node set)
                          written to session dir BEFORE the surface opens
                                                   │
                                                   ▼
                          host the graph viewer, seeded to the SCOPED subgraph,
                          opened focused on <id>; human explores / selects /
                          annotates within that bound; submits
                                                   │
                                                   ▼
                          result.json (selected ids + annotations + answer),
                          referenced by path → agent reads and decides
```

- **Invocation:** `nn ask --surface graph --focus <id> [--instructions "..."]
  [--nodes <id,id,...>]`. `--focus` (required) sets the ego. Scope defaults to the
  focus's depth-1 ego subgraph; `--nodes` optionally supplies an explicit
  allowlist for a hand-picked set.
- **Prepare (property [2], as canvas):** write the request — question/instructions
  + focus id + the resolved allowed-node set — to the session dir before the
  browser opens.
- **Serve:** launch the existing `graph export --serve`, but seeded to *only* the
  scoped subgraph (not the full graph) and opened in zoned/focused mode on the
  ego. The human uses the existing exploration (breadcrumb, tooltips) and
  selection tray within that bound.
- **Submit → result:** on the human's submit, write `result.json` under the
  ADR-0020 envelope, with an artifact naming the selected node ids, any per-node
  annotations, and a free-text answer. The `nn ask` call unblocks and prints the
  result path; the agent reads it.

The agent-supplied scope is what distinguishes this from "just open the graph
viewer": the graph surface is a **prepared, bounded request**, not an open-ended
explorer.

## Consequences

- Adds a third built-in surface with **no new UI framework** — it reuses the
  viewer and the ADR-0020 hosted-adapter machinery (ephemeral server, session
  dir, thin envelope).
- The scoping input (`--focus`, optional `--nodes`) becomes part of the ask
  contract; the server's `/graph?focus` already produces the depth-1 subgraph, so
  the bound is cheap to enforce.
- Relational feedback (support/tension/link judgments) gets a native surface,
  reducing the impedance mismatch of forcing graph questions through canvas/prose.
- The result schema stays surface-native (node ids + annotations) — consistent
  with ADR-0020's refusal to standardize a cross-surface semantic schema.

## Deferred (explicitly out of scope for this ADR)

- **Live conversational loop.** LLM interactively re-focusing the human, back and
  forth. Falls outside routed-ask; needs a second interaction mode. Deferred.
- **Write-back mutations from the surface.** Letting the human *create* links or
  notes directly in the graph surface (vs. returning a result the agent applies).
  The one-shot envelope returns intent; whether the surface can mutate the graph
  in place is a separate decision.
- **Routing inference.** As in ADR-0020, `--surface graph` is explicit selection
  only; inferring "this question is relational, use the graph" is deferred.
