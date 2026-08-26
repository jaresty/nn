# ADR 0035: Graph bodies use a lossless snapshot-bound paginated transport

## Status

Accepted

## Date

2026-08-25

## Context

`nn graph show --bodies` mixes topology, presentation metadata, and unbounded note bodies in one response. A focused neighborhood can therefore exceed an agent tool's 50KB result boundary. Truncation at that boundary is epistemically unsafe: a body can disappear, be cut in the middle, or look complete when later notes were omitted. Loading bodies one note at a time avoids the transport limit but loses the traversal-set parity and one-snapshot discipline that navigation needs.

The replacement must preserve the boundary between stored Markdown body text and frontmatter, links, graph metadata, and agent presentation. It must also let a caller prove that every selected body's UTF-8 bytes were received before making a body-derived claim.

## Decision

### Separate topology from body transport

`nn graph show` remains the topology and presentation-metadata command. A new JSON-only command transports bodies:

```text
nn graph bodies [--focus ID] [--depth N] [--direction outgoing|incoming|both]
                [--links TYPE,...] [--status STATUS,...]
                [--representation VALUE] [--page N] [--snapshot SHA256]
```

Focus, depth, direction, link-type, note-status, and representation filtering have exactly the same traversal semantics as `nn graph show`: filters constrain BFS expansion, the focus is always retained, and traversal flags require `--focus`. Without a focus the selected set is the full graph. Selected notes are ordered by ID.

`nn graph bodies` returns a compact JSON object with this contract:

```json
{
  "snapshot": "<64 lowercase SHA-256 hex characters>",
  "page": 1,
  "pages": 3,
  "next_page": 2,
  "segments": [
    {"id":"<note-id>","segment":1,"segments":2,"body":"<exact body fragment>"}
  ]
}
```

`page` and segment ordinals are one-based. `next_page` is zero on the final page. Every page repeats the same `snapshot` and total `pages`. Page 1 defaults when `--page` is omitted and does not require `--snapshot`. Every later page requires the snapshot returned by page 1. A supplied token on any page must match.

The snapshot hashes a versioned canonical encoding of the complete parsed notebook plus the normalized traversal request. It therefore binds page retrieval both to notebook state and to the selected set. A notebook mutation or a mismatched focus/filter request causes a stale-or-mismatched-snapshot error instead of returning a different page under the old token.

### Lossless bounded segmentation

The serialized response, including its trailing newline and all JSON escaping and envelope overhead, is capped at 48,000 bytes. This leaves explicit safety margin below a 50KB transport boundary. Packing measures the actual compact JSON encoding; it does not estimate body characters.

Bodies are split only at UTF-8 rune boundaries. A body larger than one response is represented by multiple records with the same ID and increasing `segment` values. `segments` gives that body's total segment count. Concatenating each note's decoded `body` values in segment order reconstructs the exact UTF-8 bytes exposed as that note's stored Markdown body, with no omission or duplication. An empty body is represented by one record with `segment: 1`, `segments: 1`, and `body: ""`. Invalid UTF-8 is rejected rather than silently replaced by JSON encoding.

Record order, segment boundaries, page packing, snapshot generation, and compact JSON bytes are deterministic for an unchanged notebook and normalized request. An empty selected set still has one page and an empty `segments` array.

### Preserve the metadata boundary

Body records carry only the note ID, segment ordinals, and body fragment. They do not carry titles, tags, note types/statuses, representation values, links, frontmatter, degree, zones, relay hints, or generated commentary. The ID is the join key to the separately retrieved topology. The `body` value is only the parsed Markdown body between frontmatter and the stored `## Links` section.

A consumer must retrieve every page, verify the repeated snapshot and page count, verify complete ordered segments for every selected ID (including empty bodies), and reconstruct all bodies before making any body-derived central claim, summary, recommendation, or concrete navigation reason. Topology may be presented before body completion, but it must not be embellished with body-derived substance.

### Deprecate inline bodies compatibly

`nn graph show --bodies` is deprecated through Cobra. Help directs callers to `nn graph bodies`, and using the old flag emits a deprecation warning on stderr. The flag remains accepted. Its text and JSON stdout stay byte-for-byte compatible, including full unpaginated bodies, tags, and omission of empty JSON body fields, so existing callers and tests do not break during the transition.

Navigation's canonical positioned read becomes topology first and bodies second: run `graph show` without `--bodies`, then run `graph bodies` with the identical focus/depth/direction/link/status/representation selection and retrieve every page under one snapshot. Navigation presentation remains blocked from body-derived claims until body retrieval and reconstruction are complete.

## Consequences

- Agent-facing pages cannot be silently clipped by the 50KB tool boundary.
- Large and empty bodies have explicit, reconstructable representations.
- A body batch cannot be continued across a notebook or traversal change.
- Topology and bodies remain separately attributable; transport metadata cannot be mistaken for stored prose.
- Callers must implement a short page loop and segment reconstruction before semantic presentation.
- `graph show --bodies` remains potentially unbounded during its compatibility window and should not be used by new workflows.
