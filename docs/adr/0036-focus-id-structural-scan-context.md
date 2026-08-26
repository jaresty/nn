# ADR-0036: Add focus-ID structural context to clusters and bridges

**Status:** Accepted

## Context

Human-driven local navigation begins from an exact retained note ID. `nn clusters --search` and `nn graph bridges --search` instead project freeform lexical relevance onto full-graph structures. Using those query-conditioned commands to describe the current note can confuse query placement with exact cluster membership or bridge status.

The existing unconditioned and query-conditioned commands are established compatibility surfaces. Exact focus lookup must therefore be additive and must reuse the same full-graph label propagation, bridge scoring, region summaries, and deterministic witnesses.

## Decision

Add mutually exclusive focus modes:

```bash
nn clusters --focus ID --json
nn graph bridges --focus ID --format json
```

`nn clusters --focus` returns one envelope:

```json
{
  "focus": {"id": "N1", "title": "Focus"},
  "cluster": {
    "size": 3,
    "representative": {"id": "N2", "title": "Hub"},
    "notes": [{"id": "N1", "title": "Focus"}]
  }
}
```

A known note omitted by the active `--min`/`--singletons` policy returns `"cluster": null`. Unknown IDs fail. Focus mode rejects `--search`, `--summary`, and `--match-limit`; `--min` and `--singletons` retain their existing meanings.

`nn graph bridges --focus` returns:

```json
{
  "focus": {"id": "B1", "title": "Bridge"},
  "bridge": {
    "id": "B1",
    "title": "Bridge",
    "score": 4,
    "relevance_score": null,
    "witnesses": []
  }
}
```

The nested bridge is the existing rich bridge record. A known non-bridge returns `"bridge": null`; an unknown ID fails. Focus mode bypasses ranking and the default limit, and rejects `--search`, `--exclude`, or an explicitly supplied `--limit`.

Without `--focus`, existing text, JSON, ranking, filtering, and schemas remain unchanged.

`nn-navigate` Local territory uses these ID-aware modes. Freeform `--search` remains the Global landscape projection mechanism.

## Consequences

- Local scans can distinguish exact structural membership from lexical relevance.
- Known negative outcomes are explicit rather than conflated with lookup failure.
- Cluster membership and bridge witnesses remain derived from one full-graph implementation.
- Two additive JSON envelopes and CLI flag-combination rules become compatibility surfaces.
