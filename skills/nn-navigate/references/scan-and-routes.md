---
name: scan-and-routes
applies_when: "When scanning local territory or the global landscape, discovering a typed destination, inspecting impact, or walking a typed route/path witness."
---

# Reference: Scan and Routes

#### `scan` — look wide (an always-available aside, not a step)

At any point in the walk (Enter, Orient, Read, Recenter) you can stop stepping sideways and *zoom out* to see the landscape around the current node, then return to the walk where you left it. `scan` is deliberately **not zoned**: zones are defined only relative to one ego (design note 20260821155048-7280), so they answer "how do my neighbors relate to me" (relationship) — `scan` answers "what territory am I standing in" (structure at altitude), which zones cannot express. Use it when you feel lost, when Orient shows an unexpectedly dense or sparse neighborhood, or before committing to a direction.

`scan` offers exactly two actions: **Local territory** for the ego at depth 2 and **Global landscape** for the wider graph. Both retain focus; Global landscape receives `↗` because it explores beyond local. Do not raise `--depth` to "look farther out": past depth 2 more hops give an exponential tangle, not a broader view — the real broadening is the **ego→global shift**, not more depth. Render the selected anchor with these existing primitives:

**Your territory (ego, depth 2)** — where you stand and how far you can walk:
- **Degree + reach** — the current node's `↑out ↓in` (hub or leaf?) plus its 2-hop extent, unzoned:
  ```
  nn graph show --focus <id> --depth 2 --direction both --color always --format text
  ```
- **Region + load-bearing** — acquire exact local structural context for the retained ID rather than projecting a goal query onto it:
  ```
  nn clusters --focus <id> --json
  nn graph bridges --focus <id> --format json
  ```
  The cluster envelope reports exact full-graph membership under the active singleton/minimum policy; `cluster: null` means the known focus was omitted by that policy. The bridge envelope reuses the rich full-graph bridge record and witnesses; `bridge: null` means the known focus is not a bridge. Unknown IDs fail. These known negative results are local facts, not failed searches. A focus on a bridge is a crossing point; a focus deep in one cluster is interior.
- **Reachable-but-unlinked** — `nn list --similar <id>`: nearby notes that share vocabulary but have no edge to walk. This is the one signal a step-wise walk can *never* surface — territory the navigation can't reach yet (candidate missing edges).

**The wider landscape (global)** — where your region sits in everything (drops the ego entirely). Freeform `--search` belongs at this Global landscape altitude; do not substitute a query-conditioned projection for the exact local `--focus` lookups above:
- **The landmass — `nn clusters`** — every topic cluster and its size. When the walk has a goal query, prefer `nn clusters --search "<query>" --json --summary` to project that query onto full-graph regions without loading every unrelated cluster or every member of a matching cluster. Summary output defaults to the top 3 ranked matches per region while retaining total `match_count`; read `matches_returned` and `matches_truncated`, and use `--match-limit 0` only when Scan genuinely needs every matching note. The match limit is per region and never limits the number of regions.
- **The highways — `nn graph bridges`** — integration points whose links join otherwise-separate regions, ranked by load-bearing weight. When the walk has a goal query, use `nn graph bridges --search "<query>" --format json --exclude <focus-id>` to project relevance onto bridges computed from the complete graph without offering the retained focus as a movement candidate. `--exclude` is repeatable and is applied before `--limit`, so excluded results are replaced rather than shortening the candidate list. Read each returned record's bounded crossing witnesses and region context to explain why it is a plausible crossing before acting; the connector evidence is not proof of territorial separation. Peek through a returned bridge `id` to inspect where it leads without moving, or Recenter on that `id` to cross into its neighborhood. (Daily/index notes often rank high because they touch many topics — treat those as connectors-by-aggregation, not substantive bridges.)
- **The whole shape — `nn graph show --color always`** *(no `--focus`)* — the entire graph, a last resort (it's large); usually the two above are enough.

Keep the selected action labeled as either *Local territory (ego)* or *Global landscape*. If the human explicitly asks to compare both, render two labeled sections rather than inventing a third Scan action or blending away the ego/global distinction.

**Make it spatial — you draw the map, the command is just the data.** The `--format text` trees are the *data source*, not the thing you relay. Don't paste raw output. Read it — indentation is reach (hops out), `↑out ↓in` markers are terrain (hubs vs. leaves) — then *draw* a map in whatever form fits the surface (the same "CLI is a faithful data source, the agent is the presenter" split the zoned step uses). Cheapest first:

- **Indented / positional text** — an annotated tree, or a rough sketch placing the hub at the center, with `🔵 TOP — what the focus answers to` geometrically above and `🟢 BOTTOM — what builds on the focus` fanning below. Geometry supplements rather than replaces those semantic labels. Works on any surface, including plain terminals and agent relay. This is the default.
- **A Mermaid diagram** — only when relaying to a human on a surface that *renders* Mermaid. Build it yourself; keep it compact (drop edge annotations, abbreviate titles). An unrendered Mermaid block is just a wall of source.

For the **global** anchor the map is a **different shape** — not a hop-tree from one ego, but **regions as blobs sized by note-count, with bridges as the lines joining them** (a continent map, not a family tree). Mark where you are. On a plain surface, an ASCII sketch:

```
  ┌──────────────────────────┐
  │  cluster 1  (259) ●●●●●●  │  central continent (prompting/token core)
  └───────────┬──────────────┘
        bridge: [B1]
              │
     ┌────────┴─────┐   ┌──────────┐   ┌──────────┐
     │ c3 (25) 🟠 YOU │   │ c2 (28)  │   │ c4 (24)  │   … 35 more small regions
     │ graph-tooling│   └──────────┘   └──────────┘
     └──────────────┘

  Legend: [B1] = 5853 — Vocab is a composition barrier — Shared vocabulary can
          prevent otherwise compatible regions from composing cleanly.
```

Before relaying either altitude, fetch `presentation` and apply its Presentation discipline (P1–P4) to the composed Scan; this is where text-source markers and manually marked JSON region/bridge data become one consistent human-facing view. Pick the rendering by surface capability, exactly as you pick whether to offer the recenter chooser — richer form only when the surface supports it, text otherwise. After scanning, resume the walk: recenter on a neighbor as before, or hop to a similar-but-unlinked note by making it the new `--focus` (and consider linking it, since the absence of an edge is what `scan` just exposed).

#### Typed destination discovery

When the walk has a current focus and a goal query but no destination yet, run:

```
nn graph routes --focus ID --links TYPES --search QUERY --limit N --json
nn graph routes --focus ID --links TYPES --search QUERY --limit N --json --explain
```

This intersects semantic relevance with actual directed reachability under the selected relationship types. A result is a candidate landing plus `witnesses`, at most 3 deterministic shortest paths selected by first-hop diversity, then edge type-sequence diversity, then lexical full-path order; none proves that the destination is the only or best conceptual landing. Without `--explain`, JSON remains a top-level route array. Opt-in `--explain` (which requires `--json`) wraps that same array as `{routes:[...], diagnostics:{...}}` with bounded aggregate diagnostics only—never note bodies, titles, or candidate dumps.

Diagnostics report normalized `query_tokens`; `total_notes`; `direct_lexical_matches` over title, body, and tags without annotations; `focus_excluded` (0 or 1); `typed_reachable` excluding focus; `eligible_destinations` (direct lexical matches intersected with typed reachability, excluding focus); `returned`; and boolean `truncated_by_limit`. `graph_scored_matches` is a separate count from the full graph-aware relevance scorer and may exceed direct lexical matches because inbound/outbound annotations can score. Annotation-only scores never make a route eligible. Routes has no depth flag, so the diagnostics do not report or imply a traversal depth.

- **Orient** — run typed destination discovery when the goal implies a relationship family but the destination is unknown; present the highest-ranked reachable destinations beside the local map.
- **Scan** — use the ranked destination set to see query-relevant territory reachable under `TYPES`; absence means no directed route to a destination with positive direct lexical evidence under that filter, not global disconnection.
- **Peek** — inspect a selected result's complete `witnesses`; each member has aligned `nodes` and `edges`. Compare alternatives without changing focus.
- **Recenter** — choose one witness, move to `witnesses[k].nodes[1]`, and rerun Orient; do not jump directly to the destination while claiming to walk that witness.
- **Arrive** — when focus reaches `destination.id`, explain the traversed edge types and annotations and report the destination's `relevance_score` as discovery evidence, not relationship strength.

#### Explicit typed impact overlay

When the walk asks what a retained focus affects or what relies on it through a known relationship family, run:

```
nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json
```

This is a bounded structural impact set, not relevance ranking or inferred closure. Choose direction from stored semantics: `grounded-by` incoming from an evidence focus finds claims that depend on it, while `supports` outgoing from an evidence focus finds claims it corroborates.

- **Scan** — read `summary` first for `total_impacts`, depth distribution, overlapping first-hop branch counts, and the honest `witnesses_truncated` signal, then read the complete returned impact set to the requested depth; absence under one type/direction filter is not proof of global isolation.
- **Peek** — inspect a selected impact's full `witnesses` array without changing focus. For every incoming witness, `nodes` run focus→impact while each stored edge retains source→target orientation and therefore points opposite its consecutive traversal nodes.
- **Recenter** — choose one witness and move only to `witnesses[k].nodes[1]`, retain the selected types/direction/depth deliberately, and rerun from the new focus; do not jump to a distant impact while claiming to walk the witness.
- **Arrive** — when focus reaches the selected `node.id`, explain the traversed edge types and annotations. Read each incoming edge in its stored source→target orientation rather than reversing its claim.

#### Typed path route overlay

When the walk has both a current focus `<a>` and a known destination `<b>`, use `nn path <a> <b> --links <types> --json` as an optional semantic route overlay. It returns `witnesses`, at most 3 shortest directed paths whose edges use only the requested relationship types. Selection prioritizes distinct first-hop nodes, then distinct edge type-sequence values, then lexical full-path order. These are concrete routes, not Datalog closure: Datalog answers whether and what follows transitively; typed path shows ordered hops that Navigation can execute and explain.

Integrate the overlay into the navigation actions as follows:

- **Orient** — compute the typed path when the goal implies a relationship family and show the route beside the local map. Relationship direction matters: use `grounded-by` to walk claim→evidence, `supports` to walk evidence→claim, and `requires` for source task→required dependency. Do not combine opposite semantic directions merely because both concern evidence.
- **Teleport** — keep instant relocation as the default. Offer the typed path only when the human wants an explainable semantic route instead of jumping directly to a distant landing.
- **Scan** — assess whether candidate targets or regions have a semantically coherent route from the current focus; do not confuse absence under one filter with global disconnection.
- **Peek** — preview every complete node-and-edge witness without changing focus.
- **Recenter** — choose one witness and move to its `nodes[1]`, the next hop, then rerun Orient. Never jump to the final node while claiming to walk the route.
- **Arrive** — explain the traversed `edges` sequence, including relationship types and annotations, as the semantic account of how origin and destination connect.
