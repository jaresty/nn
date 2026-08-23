# ADR 0031: Graph output carries opt-in presentation budgets

## Status

Accepted

## Context

`nn graph show --zones --bodies` intentionally returns complete note bodies so an LLM can preserve qualifications and provenance while summarizing. The navigation guide defines degree-scaled relay lengths, but an integrated trial relayed raw output because those rules were no longer present near the command result.

Truncating bodies in the CLI would reduce evidence quality. Requiring the model to recover presentation policy from earlier guide context is fragile.

## Decision

`nn graph show` accepts `--presentation-hints`. When enabled, every node receives a deterministic budget based on full-graph inbound degree:

- 0–1: leaf, one clause;
- 2–4: connected, one sentence;
- 5+: hub, 2–3 sentences including why it is load-bearing.

High-degree notes tagged `daily` or `index` are aggregation hubs and receive the connected one-sentence budget plus an explicit override note.

JSON nodes add `summary_budget` with `tier`, `length`, `include`, and optional `note`. Text output adds a `relay budget:` line beside each node. Bodies are never altered or truncated.

The flag is opt-in; output remains unchanged when absent. Navigation Orient enables it by default.

## Consequences

- Presentation policy travels with the evidence it governs.
- LLMs retain complete bodies while receiving local instructions about how much to relay.
- The hints describe budgets rather than generating summaries; semantic synthesis and justified overrides remain the LLM's responsibility.
