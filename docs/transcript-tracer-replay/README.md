# Controlled conversational replay — ADR-0048

One isolated general-purpose agent (`23d4b203-9d04-430`, handle `tracer-replay`) executed
17 controller-supplied user turns serially, preserving its state through Agent resume. It loaded the
served skill and invoked actual native commands against synthetic transcript fixtures. The controller
inspected tool output, a CLI-argument ledger and the isolated notebook's Git history. This was an actual
model/tool replay, not a fictional expected-answer walkthrough or independent human usability study.

## Environment and artifacts

- Checkout CLI built at `/tmp/nn-tracer-replay/bin/nn`; wrapper isolated HOME, config, cache and notebook
  beneath `/tmp/nn-tracer-replay`. Real notebook was not used.
- `setup.py` records the initial synthetic inputs. `advance.py` appends producer evidence and creates
  the new in-project conversation between investigation and Back/Refresh. These setup scripts write
  only their hardcoded scratch directory; they are retained evidence, not automatically run tests.
- `producer-final.jsonl` and `consumer-final.jsonl` retain the final evidence, including the later
  consumer passing case. The controller appended the two latter consumer records before turn 12.
- `commands.jsonl` records wrapper arguments starting at turn 2, including controller verification
  reads and bootstrap calls. It is not a role-tagged transcript, complete tool-output archive, or
  immutable publication. Turn 1 was inspected through the agent conversation tool before audit logging
  was added. Do not infer call ownership or turn boundaries solely from this argument ledger.
- The text-only test transport used textual contextual choices, not the host's structured picker.

## Executed turns and observed outcomes

| Turn | Actual requested operation | Observed response / check |
|---|---|---|
| 1 | Observe project-a as ongoing project scope | Retrieved discovery, bounded tree, ROOT and A/B; reported the compatibility gap as agent reports, not an established failure. Disclosed nine omitted ROOT events and one child. |
| 2 | Compare producer with explicit project-b consumer, without broadening observation | Retrieved exact producer and consumer events. Explained `task_id` versus `id`, qualified recorded-example support, retained project-a scope. |
| 3 | One-shot: parentage or resolution established? | Answered neither; distinguished fixture reference from authenticated parentage and kept observation context. |
| 4 | Back, after source append and new conversation creation | Restored the old ROOT/A/B observation and old omissions, labeled retained evidence. CLI audit count stayed 5 before/after Back. |
| 5 | Refresh | Rediscovered project-a, selected its new conversation, did not adopt project-b. Said producer lead was “not rechecked—not resolved.” |
| 6 | Capture the earlier mismatch | Searched, presented a concrete single draft/no-links proposal; the empty test notebook also acquired a daily note during normal bootstrap. See setup confound below. |
| 7 | Reject proposal | No capture write; retained packaging observation context. |
| 8 | Repeat after a temporary command-specific rule | Proposed without a write. This trial is not evidence for the final skill: that rule was withdrawn. |
| 9 | Repeat under final instructions after normal bootstrap | Search and proposal; Git HEAD stayed `89c145b22eb0bc4c1c642c5fffc1246f5142cf7a`, working tree clean, only baseline daily note present. |
| 10 | Reject | Same Git HEAD and clean tree; no capture write. |
| 11 | Approve the exact turn-9 title/body, draft, no links | One new note/commit `cf5f36ef352d834109e2d19638ee09f9cdfdd535`; stored body inspected against proposal, draft status and no links preserved. |
| 12 | Recheck later consumer evidence and propose a change, no write | Retrieved exact passing result, existing note, graph topology and bodies. Proposed an append preserving the historical failure and qualifying single-case support. HEAD stayed `cf5f36e…`. |
| 13 | Approve exactly that append | One update commit `e8b88d8`; original body/title/draft retained, later evidence appended, no links. Stored note inspected. |
| 14 | Change observation to exactly producer and consumer files | Inspected both ROOTs and producer A/B, disclosed omissions, explicitly replaced project scope with a fixed two-file set. |
| 15 | Refresh after a newer unselected project-b file was created | Reacquired only the two selected conversations; did not discover/adopt the newer file. |
| 16 | Inspect `missing-event` exactly, no substitution | Reported `events: event id not found`; preserved scope and did not substitute another event. |
| 17 | End | Ended without another acquisition or picker loop. |

Selected verbatim outputs:

> Compatibility is the main open lead—not an established failure.

> The producer compatibility lead is **not rechecked—not resolved**. Project-b remains outside observation scope.

> The earlier statement about no inspected later resolution describes the initial evidence set; this follow-up adds a bounded passing case without negating the historical failure.

## Setup confound and withdrawn workaround

The first capture trial started with an empty, unbootstrapped notebook. Loading the normal notebook
workflow caused `show --global` to create a daily note (`61b4386`), unrelated to the proposed mismatch
note. The controller initially interpreted this as a tracer defect and added a command-specific ban.
The user challenged that interpretation. The ban and its test were removed; no CLI change was made.

The controller removed the test daily note, then deliberately completed normal bootstrap before a new
baseline (`89c145b…`). Under the final, unrestricted command-routing instructions, proposal and rejection
made no further notebook changes, and approval made the single proposed change. This supports treating
the earlier write as a setup confound, not as evidence requiring a tracer-specific command ban. It does
not prove every possible session-initialization situation is side-effect-free.

## Limits

This is one synthetic, stateful model replay. It supports the recorded cases, not general model
compliance, broad usability or a statistically meaningful success rate. In particular:

- Unprompted proactive capture was not demonstrated; capture was requested explicitly.
- No independent test of structured pickers, compaction loss, expired/corrupt captures, large segmented
  payloads, sensitive real data, or schema-wide conversational behavior was run here.
- Graph retrieval covered the isolated note with no linked neighbors; typed link approval and rich
  multi-page graph synthesis remain untested conversationally.
- Known self-exclusion could not be established; the agent disclosed that rather than guessing.
- The agent retained context across instruction reloads; the initialized capture check was not a fresh,
  independent model replication. Host-level Protocols preambles appeared on several turns.
- Fixtures are deliberately synthetic recorded reports/results, not evidence of an actual production
  integration failure or independently rerun application tests.

Native tests separately exercise command behavior and transport. Publication guards separately protect
served wording. None of these three evidence classes should be represented as the others.
