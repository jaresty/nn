# ADR-0048 implementation verification

## Status

Implementation with native/publication verification and one bounded controlled conversational replay.
This is not broad usability validation. No production Go changes were required. The compact core, references, historical ADR boundaries and publication guards were
updated together. Two new owners are `observe` and `investigate`.

## Executed checks

The following commands passed after the final recipe mutation was restored:

```text
go test ./... -count=1
go test -race ./cmd/nn/cmd -run 'Transcript|Attention|Review|Awaiting|Assignment' -count=1
go test -race ./internal/attention -count=1
go vet ./...
git diff --check
TRACER_FINAL_NATIVE_VERIFICATION_PASS
```

The full suite covered all packages; relevant command race tests and the complete internal attention
race suite ran separately. Existing native command/evidence tests were retained. Changed old assertions
are publication wording checks for superseded interaction defaults, not relaxations of the native
rejection classifier, ownership, pagination, capture or evaluation tests.

`TestTranscriptTracerNativeRecipe` executes the four command templates extracted from the CLI-served
`observe` reference against Pi, SDK-layout and inline Claude fixtures. It covers discovery, exact
canonical path selection, a bounded direct-child page, ROOT and each selected child's readable event
window. It does not exercise an LLM's choice of population or interpretation of the results.

The initial handwritten native test omitted discovery and therefore missed a nonexistent `ls --sort`
flag. CLI inspection exposed the gap. The published recipe was corrected to explicit source-root,
conversation-only, mtime-ordered discovery, and the test now executes the served recipe rather than a
separate handwritten copy. Reintroducing `--sort observed-recent` produced `TRACER_RECIPE FAIL` with
`unknown flag: --sort`; restoring the source produced PASS for all three adapters.

Eight isolated sentence-removal mutations produced their own attributable publication failures and
passed after byte-for-byte restoration:

| Guard | Contract |
|---|---|
| TRACER_DIRECT | Bounded reads before optional choices |
| TRACER_SCOPE | Investigative reach does not expand observation scope |
| TRACER_BACK | Back does not acquire fresh evidence |
| TRACER_ROOT | ROOT included in initial selected-conversation sample |
| TRACER_COVERAGE | Candidate page differs from scope |
| TRACER_LEARNING | Cache identity alone is not durable learning |
| TRACER_CAPTURE | Concrete approval before notebook writes |
| TRACER_QUESTIONS | Question-shaped retrieval, not mandatory tree descent |

A final reference audit removed the remaining mandatory navigation descent and return-to-picker from
`patterns`. Its ownership guard now requires investigate/interaction dispatch and rejects those obsolete
directives. Reintroducing the mandatory picker produced `DOC_PATTERNS_OWNERS FAIL`; restoring the
reference passed, followed by a fresh full/race/vet run.

These are exact-publication tripwires. Deleting a sentence demonstrates sensitivity to that sentence's
absence; it does not prove semantic completeness, equivalence under arbitrary paraphrase, or absence
of conflicting instructions. Native recipe execution is stronger evidence for command compatibility,
not for model compliance. The broader retained properties' semantic adequacy remains unestablished.

Final release logs are retained under `transcript-tracer-replay/checks/` (full, race and vet).
Mutation session logs: `/tmp/nn-tracer-mutations.log` and individual
`/tmp/nn-tracer-<guard>-{wrong,restored}.log` / recipe logs. These local logs are temporary; the executed
commands, findings and limitations above are the durable record, not claims of immutable log custody.

## Executed conversational evaluation

See the [17-turn controlled replay](transcript-tracer-replay/README.md) for actual requests, observed
responses, tool-call evidence, notebook baselines/changes and limitations. The replay exercised:

- Observe scoped work including ROOT; disclose sample and omitted descendants/conversations.
- Investigate an explicit cross-thread question without navigating a compulsory hierarchy.
- Ask successive direct questions without intervening confirmation gates.
- Back restores retained context; Refresh reacquires the current activity's scope and intent.
- Consult outside sources without expanding observation scope; explicit project Refresh can admit new
  in-project conversations, while explicit A/B remains A/B.
- Missing ownership, clipped evidence and unavailable captures remain qualified unknowns.
- Recognize a useful learning; search and inspect relevant graph context; propose exact mutations;
  rejection causes no write and approval writes only the proposed change in an isolated notebook.
- Preserve a true historical observation while distinguishing a later bounded correction.

The replay caught an initialization confound: normal notebook bootstrap created a daily note before
the first capture proposal in an empty notebook. A proposed command-specific workaround was withdrawn
following user review. With bootstrap completed before the measurement baseline, proposal/rejection
made no notebook changes and approval made only the proposed capture or append. No tracer-specific
command ban or native CLI change was retained.

Not all items above were exhaustively exercised: missing exact evidence was tested, but clipped large
payloads and expired captures were not replayed; the graph case had no linked neighbors. Unprompted
proactive capture, structured pickers and compaction recovery remain conversationally untested. Read
the replay's limitations before extending these observations into broader claims. Fictional examples,
static publication assertions and native fixtures remain separate evidence classes.
