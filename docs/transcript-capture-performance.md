# Transcript capture benchmark

Measured locally during development against the retained lsp-trace conversation
`2026-09-04T14-39-04-442Z_01a06cdb-e0fa-77b0-a090-f2e24727c340`.
These are single-run wall-clock observations, not a portable performance guarantee.
The source conversation continued growing between runs.

Command: `nn transcript review <session> --queue open-handoff --limit 3 --last 5 --json`.
Replay adds `--snapshot <first-snapshot>` with unchanged options.

| Operation | Before | After |
|---|---:|---:|
| Fresh bundle | 26.665 s | 3.083 s |
| Retained bundle replay | 29.438 s | 0.010 s |
| Second retained replay | — | 0.009 s |
| Default transport | 3 pages | 1 page |
| First-page bytes | 43,830 | 18,550 |

Phase instrumentation before optimization measured 255 captured sources, 176,054,356 bytes,
and 31,100 decoded records: capture 0.878 s, review projection 25.295 s, selected tails 0.198 s.
After indexing, the same growing conversation measured capture 0.981 s, review 1.900 s,
and selected tails 0.025 s (176,292,746 bytes / 31,127 records).

The dominant cost was repeated parent-record scanning and locator decoding for each room, not
capture I/O. Captured ownership/locator indexes now build once. Bundle replay verifies and reads a
retained encoded page without reloading the raw capture or recomputing queue selection. Large source
inventories remain in the private capture manifest rather than inflating each default receipt.
Fresh capture still processes the conversation's retained population; output limits do not promise
input-I/O limits. Room-cursor advancement can still require loading/projecting the retained capture.

Regression guards cover active appends, partial final records, source deletion, exact native JSON
whitespace preservation, ownership-index parity, encoded-page replay without raw capture availability,
corrupt-page rejection, snapshot option binding, and the 48,000-byte transport limit.
