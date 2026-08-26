# ADR-0037: Preserve edge semantics in annotation RRF channels

**Status:** Accepted

## Context

The shared BM25/RRF scorer currently concatenates every inbound annotation into one field and every outbound annotation into another. Stored relationship type is discarded before scoring, so annotations on `supports`, `contradicts`, `governs`, `extends`, and legacy untyped edges compete as one bag of words.

Treating every type as an independently additive RRF field would preserve type but reward notes merely for matching more edge-type channels. Title, body, tag, direction policy, tie behavior, and existing flat-map APIs are established compatibility surfaces.

## Decision

Represent graph annotation evidence in canonical channels:

```text
(direction, edge type)
```

Direction is inbound or outbound. Empty legacy type maps to the explicit read-only channel `UNCLASSIFIED`. Any non-empty stored value retains its distinct channel; writable link-type validation remains unchanged.

For each channel `c`, compute its normal BM25 rank and reciprocal-rank contribution:

```text
R_c(note) = 1 / (rrfK + rank_c(note))
```

For each direction `d`, combine only the matching type channels by arithmetic mean:

```text
G_d(note) = mean(R_c(note) for matching c in direction d)
```

No matching channel contributes zero. Equal per-channel evidence therefore has equal directional contribution whether one or many edge types match. Inbound and outbound directions remain independently weighted. Production retains its current outbound-only policy; activating inbound ranking is a separate decision.

Final fusion remains title RRF + body RRF + tag RRF + weighted directional graph contributions. Existing title/body/tag scoring and historical candidate-order tie semantics do not change.

Add typed scorer and IDF entry points. Preserve existing flat annotation APIs as adapters that map their evidence to `UNCLASSIFIED` channels. Production corpus preparation uses typed channels.

Persist typed channel IDFs in canonical direction/type order. Cache fingerprints include direction, edge type, note assignment, and annotation text; type-only changes invalidate the cache. Bump the field-IDF cache version. Token caches key graph tokens by complete channel identity.

## Consequences

- Graph-aware lexical relevance preserves stored edge semantics.
- Legacy untyped annotations remain readable without becoming writable types.
- Matching additional relationship types does not create additional graph votes when evidence is equal.
- Existing lexical fields, outbound-only production policy, and flat compatibility APIs remain stable.
- Typed IDF payloads, cache keys, deterministic channel ordering, and normalization become regression-tested contracts.
