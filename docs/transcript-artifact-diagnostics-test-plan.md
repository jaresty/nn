# Transcript artifact diagnostics: regression-test plan

Status: recognition, opt-in JSON CLI and explicit reader tests implemented. This matrix includes residual limits, not a claim that every row has a separately witnessed RED.
Owner: [ADR 0068](adr/0068-transcript-upstream-elision-and-artifact-references.md).

## Evidence-derived fixture

Use a sanitized fixture preserving the observed adapter envelope, not the original
sensitive source or machine-specific path. The exact historical result was a
successful LSP projection whose retained adapter wrapper had this shape:

```json
{
  "ok": true,
  "data": {
    "omitted": true,
    "reason": "Raw MCP result exceeded the details size limit and was replaced with this summary to keep session context bounded.",
    "isError": false,
    "contentBlocks": 1,
    "contentSummary": [
      {"type": "text", "bytes": 12970, "lines": 1, "textOmitted": true}
    ],
    "rawResultBytes": 27290,
    "fullResultPath": "/fixture/mcp-output/mcp-result.txt",
    "structuredContent": {
      "preservedFields": {
        "envelope_version": "1",
        "tool": "lsp_trace_v2_structural_context",
        "request_id": "fixture-projection",
        "outcome": "COMPLETE",
        "operation_status": "SUCCEEDED"
      }
    }
  }
}
```

Embed the envelope as a tool-result text block, as in the retained mcpScript result.
Also cover the direct adapter shape without the `ok/data` wrapper. Use temporary
artifacts created by each test; never depend on historical `/tmp` files. Keep the
reported text truncation marker in a separate synthetic fixture: that original
incident was not independently inspected.

## Acceptance matrix

| Case | Observable expectation |
| --- | --- |
| Adapter elision fixture | One upstream-elision diagnostic; exact recorded path and event/field provenance; producer counts qualified as recorded |
| Direct envelope and known wrapper | Equivalent diagnostic meaning with distinct retained field locations |
| Explicit MCP text truncation marker | Upstream-truncation warning, even with no artifact reference; no invented path or inferred missing-byte count |
| Ordinary nn `[truncated N chars]` display | Remains display clipping, never upstream loss |
| Multi-page/segmented event | Complete reconstruction yields same diagnostic as unfragmented export; no inference from partial retained segments |
| No recognized signal | `not_detected`, never a completeness guarantee |
| Missing/null/non-string reference | Loss warning survives; no fabricated path or coerced reference |
| Malformed/over-budget envelope | Raw payload retained; explicit uninspected status, no crash or unbounded decoding |
| User/assistant prose quoting a marker | No tool-result diagnostic |
| Arbitrary embedded JSON or nested lookalike | Not recursively decoded; deferred extractor scope stays closed |
| `omitted:false`, arbitrary `fullResultPath` key | No adapter-elision assertion without required envelope evidence |
| Pi background `fullOutputPath` fixture | Existing sidechain behavior unchanged; no generic MCP reference fabricated |
| Ordinary commands without opt-in | Existing output, selected events, ordinals, IDs, raw payloads and exports unchanged |
| Diagnostics enabled | Stable ordering, no duplicate references for the same retained location, existing byte/page limits enforced |
| Recognizer/options/snapshot change | Diagnostic continuation rejects mismatches; positional event ID alone cannot authorize a stale reference |
| Detection with unavailable artifact | Detection still works without artifact access or availability claim |
| Path with quotes/control characters | Readable output safely escaped; never shell-executed |

## Explicit reader acceptance matrix

| Case | Observable expectation |
| --- | --- |
| No read operation invoked | Detection/search/export performs zero artifact opens or stats; verify via injected filesystem spy |
| Valid event-bound reference under explicit root | Returns bounded artifact bytes and acquisition provenance; original retained event unchanged |
| Missing `--allow-root` or wrong reference/snapshot | Non-interactive explicit error, no artifact access |
| Absolute path outside approved root | Denied, no bytes leaked |
| Relative path, parent traversal or non-local URI | Rejected before reading |
| Symlinked leaf or directory component | Rejected under initial no-symlink policy |
| Directory, FIFO, socket or device | Rejected as non-regular; never blocks waiting for input |
| Missing artifact | Reports unavailable/missing without claiming expiry cause or rewriting tool success |
| Permission/read error | Typed acquisition failure with no sensitive content in error output |
| Artifact exceeds byte bound | Explicit partial acquisition and acquired-prefix digest; no whole-file completeness claim |
| File changes during reading | Detectable instability is disclosed; no claim of historical byte identity even if a race is undetectable |
| Validation/open race | Adversarial component replacement cannot escape descriptor-bound approved root |
| Nested reference inside artifact | Returned as data; never followed automatically |
| Successful read | No notebook write, index mutation, network request, editor/browser launch, or command execution |

A returned digest proves only the acquired bytes unless it is compared with an
independently recorded digest. Tests must not manufacture historical identity from
path equality. A path-layout match is not a substitute for explicit root admission.

## Intended code seams and test organization

- New table-driven recognizer tests near transcript event projection tests.
- Opt-in JSON output/pagination tests in `transcript_events_diagnostics_test.go`;
  readable diagnostics are explicitly unsupported in this increment.
- Keep `TestTranscriptPiRawPreservesMessagesInEveryRoute` as a compatibility guard;
  it exercises subagent resolution, not generic artifact recovery.
- Reader tests in `transcript_artifact_read_test.go` and the Unix-tagged test file
  use an injected opener for denied-path/no-I/O checks and real temporary
  directories for descriptor/no-follow integration tests.
- The embedded `nn-transcript` events reference documents both shipped commands,
  including the exact-event snapshot and explicit-root requirement.

Test locations do not establish call topology or historical artifact identity.

## Verification order

1. Recognizer tests and assertion-specific CLI REDs precede the implemented JSON projection.
2. Run focused recognition, JSON projection, pagination and compatibility tests.
3. Explicit-reader admission/provenance tests preceded or counterfactually guarded implementation.
4. Run targeted artifact tests plus existing raw/meaningful projection, exports,
   paging, and Pi sidechain ownership compatibility tests.
5. Run the owning Go package suite once, then broader checks required by the eventual
   diff. Record actual commands and results in the implementation handoff.

Do not accept a feature solely because a fixture exists: verify its assertions and results.
No dynamic artifact-open spy has run for diagnostics listing. The reader's injected
opener tests establish no opener call on invalid root, missing reference, wrong event,
stale snapshot, and rejected recorded path; this does not cover every possible I/O path.

## Open acceptance decisions

- JSON `events --diagnostics` and recognizer v1 are implemented; text mode is rejected.
- `transcript artifact read` uses an exact-event snapshot, one-based finding, explicit
  symlink-free root, and a 16,384-byte bound; Darwin/Linux use descriptor-relative
  no-follow opens and other platforms fail closed.
- Source evidence and focused tests cannot prove absence of every filesystem race
  or identity with the historical producer artifact.
