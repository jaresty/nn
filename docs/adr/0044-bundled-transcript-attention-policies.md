# ADR-0044: Bundled transcript attention policies in an extensible rule language

## Status

Accepted architectural direction; not implemented. The illustrative language, metrics, thresholds,
and evaluator integration below require validation before becoming a public contract.

Related: [ADR-0043: Intent-preserving transcript discovery](0043-intent-preserving-transcript-discovery.md)
and [ADR-0042: Transcript navigator](0042-transcript-navigator-duckdb-spine.md).

## Context

Transcript users have different reasons to inspect a thread. Command-heavy work may be normal for
research, diagnosis, or verification, but may warrant inspection during an implementation task.
A universal alert threshold would conflate these needs and could imply productivity, failure, or
runtime state that transcript evidence cannot establish.

The initial candidate is a low ratio of recognized edit operations to command operations within a
bounded window. This is a ratio signal, not independent comparisons of absolute command and edit
counts. A minimum denominator can guard against tiny samples without replacing the ratio criterion.

We want to learn from one built-in policy before offering general configuration. However, implementing
a hard-coded detector now and introducing a policy language later would defer the central abstraction.
The agreed direction is to implement the first policy in the proposed language itself and bundle that
definition with nn. General notebook configuration is a later exposure of the same mechanism, not a
second condition engine.

## Decision

### 1. Ship one definition through an extensible evaluator

The built-in policy is a versioned definition containing scope, evidence window, parameters, and a
Datalog-style rule. The runtime parses and evaluates that definition; it must not bypass the rule
with a policy-specific Go threshold check.

Native code still owns collection, authenticated ownership, metric calculation, and bounded evidence.
The evaluator owns rule matching. The Office owns qualified presentation and offers inspection.
A match is an attention suggestion, not a diagnosis or an instruction to act on a worker.

Initially, load only bundled definitions. Do not add user-policy discovery, notebook activation,
general configuration commands, background monitoring, or automatic notifications in this increment.
Do not make this a mandatory global alert: evaluation is explicitly scoped to selected work.

### 2. Retain the wrapper plus Datalog-style rule shape

Illustrative definition, **not executable syntax or calibrated defaults**:

````markdown
```nn-attention
scope:
  task: implementation

window:
  last_work_events: 100

parameters:
  minimum_commands: 30
  minimum_edit_ratio: 0.05

rule: |
  attention(Thread, "Low observed edit-to-command ratio") :-
    command_count(Thread, Commands),
    recognized_edit_count(Thread, Edits),
    Commands >= minimum_commands,
    Edits / Commands < minimum_edit_ratio,
    classification_complete(Thread).
```
````

A project restriction could be supplied in the scope when evaluation is requested; the bundled
policy must not embed a developer-specific absolute project path. Task applicability must be explicit,
not silently inferred from prose. Missing required scope produces an inapplicable result, not a match.

Keep the rule, rather than replacing it with a YAML condition tree. YAML-like fields describe the
context in which the rule runs. Parameters must be typed, validated, and bound as values, not inserted
into executable rule text by string substitution. Definition identity/version or digest must accompany
results so later changes cannot silently reinterpret an earlier match.

Reuse nn's existing Datalog parser for the `rule` body; do not write a second Datalog parser or
parallel rule-language implementation. The scope/window/parameter wrapper is a separate configuration
layer and delegates rule parsing to that existing parser. Reuse the existing evaluator with an isolated
transcript fact context.

The existing engine's aggregation, numeric comparison, parameter, and arithmetic support has not been
established by this decision. Check those capabilities before fixing the grammar. The example expresses
intended semantics; it does not authorize publishing unsupported syntax. If necessary, extend the
existing implementation narrowly or express the operation through a supported safe numeric relation,
with regression tests preserving existing notebook behavior. Neither a fresh parser nor a hard-coded
policy-specific detector is an acceptable workaround for unsupported illustrative syntax.

### 3. Define evidence before interpreting the ratio

The intended quantity is recognized edit operations divided by command operations. Before implementing
the built-in, publish and test an operational definition for each metric, including:

- Which dedicated tools and shell invocations contribute to each count, and whether categories overlap.
- Whether an operation counts an invocation, a successful result, or a verified change; do not conflate them.
- How duplicate message/tool-result representations are prevented from counting the same operation twice.
- How the work window is selected, including boundary-crossing invocations and partial evidence.
- How unsupported tools, shell side effects, unknown timestamps, and missing detail affect eligibility.

Native code supplies bounded facts with canonical thread identity and evidence provenance. The policy
must not parse raw transcript payloads or infer ownership itself. Preserve unknown values rather than
converting them to zero. A zero or unavailable denominator cannot produce a valid ratio.

`classification_complete` is a proposed eligibility predicate, not proof of complete filesystem
observation. Its meaning must be specified against the supported classification vocabulary. Recognized
edit counts must never be presented as the total number of actual edits. Decide explicitly which unknown
classifications make the signal indeterminate; do not suppress missing evidence behind a low ratio.

Ratios, window size, and minimum sample size are provisional. A low ratio does not establish looping,
lack of progress, wasted work, task failure, or a need to steer. Even implementation work can legitimately
spend a long interval investigating or verifying without edits.

### 4. Isolate transcript evaluation from notebook validity

Use a separate transcript fact namespace and evaluation context. Existing `nn-rule` blocks, notebook
facts, structural invariants, and `nn rules check` behavior must remain unchanged. An attention match
must not become a notebook violation or change the notebook check command's exit status.

A future `nn-attention` block in a notebook would be a separate opt-in policy source, not automatically
loaded merely because a note was retrieved. That future source should feed the same parser/evaluator
used by the bundled definition. Activation, precedence, trust, overrides, and policy composition are
explicitly deferred.

Evaluation is read-only, bounded, and deterministic for identical retained evidence and policy inputs.
No arbitrary code execution, file access, network access, notebook writes, correction delivery, or
worker control is available to a rule. Transcript payload instructions remain evidence, never authority.
Invalid definitions and evaluation limits must fail explicitly rather than masquerade as no match.

### 5. Make results inspectable

A result distinguishes match, no match, inapplicable, and indeterminate evidence; execution errors are
separate from those outcomes. Include policy identity, effective scope and parameters, retained evidence
reference, window coverage, numerator, denominator, ratio when defined, and classification limitations.

The Office presents a match as **worth inspecting**, with an evidence inspection action. Do not expose
an unexplained red/green state, productivity score, or inferred liveness. No automatic intervention follows.

## Alternatives considered

- **Hard-code one detector, add configuration later:** rejected; it postpones exercising the intended
  definition/evaluation boundary and risks two implementations of the same policy.
- **Use a YAML-only ratio condition engine:** rejected; the agreed shape retains a Datalog-style rule
  within the scope/window/parameter wrapper.
- **Offer arbitrary notebook policies immediately:** deferred; one built-in can expose vocabulary and
  evidence problems before making a broad compatibility promise.
- **Put attention rules into existing notebook validity checks:** rejected; attention is contextual and
  advisory, not an invariant over notebook truth.
- **Adopt a universal low-ratio alarm:** rejected; usefulness depends on task, window, and evidence quality.

## Consequences and verification

The first increment includes a small language/evaluation boundary, not just a metric comparison. That
cost is intentional: the bundled rule exercises the extension path before it becomes user-configurable.
It does not yet establish that the signal is useful or that any threshold is calibrated.

Before shipping:

1. Inspect and reuse the existing Datalog parser and evaluator; verify their capabilities and freeze
   the minimal supported wrapper, rule grammar, and metric definitions. Test that bundled rule bodies
   actually pass through the existing parser rather than a parallel implementation.
2. Test parser/type errors, positive thresholds, zero/unknown denominators, scope mismatch, window
   boundaries, duplicate representations, unknown classifications, and bounded evaluation failure.
3. Prove the bundled definition controls matching: change its ratio threshold or rule in an isolated
   test and observe the expected result without changing detector code.
4. Verify retained-evidence reproducibility and unchanged notebook rule/check behavior.
5. Evaluate bounded implementation, research, and verification examples; retain false positives,
   missed cases, and limitations rather than claiming semantic accuracy from parser tests.

Only after that experience should a later decision expose notebook-defined policies using this same
mechanism. This ADR does not authorize additional detectors, dashboards, monitoring, or automatic actions.
