# ADR-0055: Standalone delegated-process review protocol

Status: Accepted

## Decision

Publish `virtual-nn-delegated-process` as a standalone conditional virtual
protocol. It applies when a decision about delegated work depends on what the
agent actually established and retained transcript evidence is available.

Require proportional, attributable transcript inspection rather than conclusions
from status, counts, summaries or missing returns alone. Distinguish planned RED
from unexpected failure, reports from inspected results, and producer completion
from successful parent handoff. Normal bounded reads remain direct.

The user requested replacement of the notebook protocol, not preservation of its
source ID or duplicate-suppression machinery. The old notebook note was explicitly
deleted. No notebook provenance field or migration lookup is introduced.

## Verification

Test standalone lookup, conditional wording and global publication. Publication
tests do not establish broader model compliance with the protocol.
