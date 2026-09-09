---
name: recovery
applies_when: "When a transcript has an unknown schema and native diagnosis or an explicit DuckDB escape-hatch reconstruction is needed. Owns the four reconstructed-join validation assertions."
---

# nn-transcript / recovery — unknown schemas

When scan reports unknown, run `nn transcript doctor`. Use the co-versioned native CLI for supported
schemas; DuckDB is an escape hatch, not an alternative normal workflow. Recovery is available for
one session as well as a cross-session sample; loading patterns is not a prerequisite.

## Reconstructed join validation (four assertions)

When a session has an `unknown` schema and you reconstruct its spawn relation with DuckDB, an
LLM-composed query fails silently-but-plausibly — a wrong join can resolve every row and still be
wrong. Before trusting the reconstructed relation, all four must pass:

- every non-root agent resolves to exactly one parent;
- no agent is its own ancestor (DAG, no cycles);
- each spawn timestamp is at or after its parent's start;
- the resolved spawn-edge count equals the count of spawn tool-calls.

Only emit the relation after all four pass. If any fails, the join is wrong — iterate. Missing evidence
that prevents a check is not a pass. Keep the reconstruction and its source limitations explicit;
structural consistency alone does not supply missing authenticated ownership or establish task success.

The tree is a lossy overview; retrieve complete relevant events before interpreting behavior. Load
`nn skills get nn-transcript --reference events` for supported transport, and
`nn skills get nn-transcript --reference interaction` before expanding an acquisition envelope.

When a new schema is cracked, propose capturing the reusable recipe under
`nn skills get nn-transcript --reference actions` and the normal nn capture discipline. Recovery itself
never writes notebook truth. Return to the originating task; use
`nn skills get nn-transcript --reference patterns` only if that task is a cross-session investigation.
