# Transcript artifact diagnostics: frozen implementation handoff v1

Parent phase: transcript-recognizer-plan. Scope: increment 1 only.
Requirements: ADR 0068 and transcript-artifact-diagnostics-test-plan.md, narrowed by
this handoff. CLI and artifact reading remain separate increments, not authorized
by this handoff. This is a semantic implementation assignment, not a mechanical
execution exemption; the writer must derive its own applicable Bar phase.

## Delivery sequence

1. Pure bounded diagnostic recognizer and tests (this assignment).
2. Separate plan for opt-in events --diagnostics, readable/JSON projection,
   pagination/snapshot binding, compatibility tests, embedded reference update.
3. Separate plan for explicit artifact read, caller-approved root, descriptor-bound
   no-follow admission, provenance, unavailable/partial outcomes, security tests.

No CLI syntax is shipped by increment 1. Do not change ADR status to Accepted.

## Established seam / evidence boundary

Parent inspected projectLedger's retained source: ledgerEvent preserves raw message
or content-block payload in payload; tool_result events are distinguishable by kind.
ledgerText only decodes strings/content blocks, not arbitrary embedded MCP JSON.
Parent directly retrieved readOwnedPiSidechain, ownedPiRecords and
validatePiSidechainPath through LSP: existing Pi resolution enforces a different
layout/agent contract and must not be reused or modified here. These are source
observations, not runtime tests or repository-wide absence claims.

## Exclusive writer scope

Create only:
- cmd/nn/cmd/transcript_artifact_diagnostics.go
- cmd/nn/cmd/transcript_artifact_diagnostics_test.go

Use existing package cmd and ledgerEvent type. No CLI wiring, edits to existing
production files, dependencies, notebook changes, Git commits, generated binaries,
LSP configuration/session mutations, or docs updates. Stop if the two-file scope
cannot implement the contract; do not expand it silently.

## API and output contract

Internal entry point: diagnoseTranscriptArtifactEvent(e ledgerEvent)
returns a typed diagnostic result; choose idiomatic unexported Go type names.

Result fields / JSON names:
- version: fixed nn.transcript.artifact-diagnostics/v1
- event_id: input event ID (never fabricate one)
- status: detected | not_detected | uninspected
- findings: ordered non-null array
- inspection_complete: boolean
- issues: ordered non-null array of location/reason records

Non-tool_result events return not_detected with empty findings without inspecting
payload content. Supported payload representations are json.RawMessage and maps
already used by ledger events; unsupported representations return uninspected.
Do not marshal the entire input into a new huge buffer just to apply a size bound.

Each finding carries kind (upstream_elision or upstream_truncation), recognizer
identifier/version, payload-relative JSON Pointer location, and optional artifact
reference. Reference fields: recorded_path (verbatim), location (exact pointer),
trust="recorded_unverified". Do not stat or normalize the path. Missing/null/
non-string/empty paths do not erase an otherwise valid loss finding.

For JSON decoded inside a text string, preserve two locations: the outer payload
pointer to that string and an inner JSON pointer for the recognized field. Never
pretend decoded inner fields are direct fields of the original payload.

Status is detected when findings exist, even if inspection was incomplete;
inspection_complete=false plus issues discloses incomplete coverage. With no
findings, status is uninspected if eligible content could not be inspected,
otherwise not_detected. not_detected is never called complete source evidence.

## Recognition v1

Operate only on tool_result payload content: a plain string or an array of text
blocks (type=text and text string). Never scan assistant/user messages or arbitrary
fields. Decode a candidate text as JSON at most once; inspect either that root
object or exactly its known ok/data wrapper (ok is boolean and data is object).
No recursive decoding of strings, arbitrary wrappers or artifact following.

Adapter-elision shape requires ALL:
- omitted is boolean true;
- reason is nonempty string;
- contentBlocks is nonnegative integral JSON number;
- contentSummary is an array with at least one object whose textOmitted is true;
- rawResultBytes is nonnegative integral JSON number.

fullResultPath is optional and recognized only within that admitted envelope.
Numeric fields are producer-reported metadata, not measured omitted bytes.
Do not require particular English reason wording or an LSP-specific tool name.

Truncation recognition: an explicit bracketed text marker beginning
[MCP text output truncated: and ending ] with a nonempty interior. Preserve the
marker location and text offset; do not parse a missing-byte count or guess an
artifact path. The unrelated nn [truncated N chars] marker is not recognized.
A quoted marker inside a tool-result text is reported only as a recorded signal,
not an authenticated producer statement. User/assistant events are excluded.

Do not treat Pi fullOutputPath or a lone fullResultPath key as adapter evidence.
One candidate contributes at most one finding of each kind; no duplicated findings
for the same kind/location. Preserve content-block order and a fixed kind order.

## Bounds

Fixed v1 bounds, checked before expensive decoding/scanning:
- candidate text: 65536 bytes, equality accepted;
- content blocks inspected: 32, equality accepted;
- JSON nesting: 16 (root container depth 1), equality accepted;
- findings retained: 16, equality accepted.

Bound top-level raw payload decoding independently at 4 MiB; over-limit raw payload
returns uninspected without decoding it. Already-decoded map content still uses
block/text bounds; do not recurse through unrelated map values.
Use a string/escape-aware pre-scan or bounded decoder for depth, not an unrestricted
unmarshal followed by a depth check. Check limits across wrapper and candidate
parsing. Malformed JSON-looking candidates (first nonspace { or [) yield an issue;
ordinary non-JSON text without a marker is not_detected. Mark remaining coverage
incomplete on block/finding limits; no silent truncation or inferred completeness.

## Tests and counterfactual evidence

Persist tests named TestTranscriptArtifactDiagnostics... in the allowed test file.
Use the sanitized observed envelope in the ADR test plan as the positive fixture.
Include direct and ok/data shapes, exact inner/outer provenance, multiple blocks,
marker-only detection, missing/malformed path, omitted:false, lookalike fields,
non-tool_result messages, Pi fullOutputPath, nested-string/wrapper non-recognition,
malformed input, unsupported payload, every bound at and beyond equality, escaped
brackets/braces/quotes for depth, stable ordering and immutable input.

Use in-memory fixtures only. Prove no artifact I/O by keeping the recognizer free
of filesystem/network/process APIs; final parent diff review checks that boundary.
Input equality assertions must cover both map and raw payload forms.

Establish an assertion-specific RED against a compiling minimal implementation;
missing symbols/compile errors do not qualify. Record assertion identity and actual
failure, then GREEN. For negative/boundary assertions already passing under the
minimal baseline, use focused reversible perturbations of the implementation to
show those guards reject incorrect behavior; restore production before finishing.
Do not claim all assertions witnessed unless their actual records support it.

## Verification

Inner loop / exact feature acceptance:
  go test ./cmd/nn/cmd -run '^TestTranscriptArtifactDiagnostics' -count=1

Before handoff:
  gofmt -w cmd/nn/cmd/transcript_artifact_diagnostics.go cmd/nn/cmd/transcript_artifact_diagnostics_test.go
  go test ./cmd/nn/cmd -run '^TestTranscriptArtifactDiagnostics' -count=1
  go vet ./cmd/nn/cmd
  git diff --check

Parent will run the owning package suite at integrated acceptance; do not repeatedly
run exhaustive suites in the RED/GREEN loop. Stop and report unrelated failures
without fixing them. No race suite unless new concurrency-sensitive code appears
(which would itself exceed this pure-function scope).

## Stop / handoff

If payload ownership, type shape, parsing limits, or semantics conflict with the
frozen contract, stop and return exact evidence; do not redesign the contract.
Writer must re-establish its exact READY MCP session and make only minimal seam/API
confirmation using source projection, not repeat broad architecture discovery.
Use adequate independent LSP budgets; after a typed traversal/cancellation failure
check full session state, and stop if POISONED. No blind retries.

Report changed paths, exact RED/GREEN commands/results, any unwitnessed assertions,
limits/ambiguities and acceptance status. Parent must inspect retained transcript,
actual diff, focused tests and integration checks before accepting this increment.
