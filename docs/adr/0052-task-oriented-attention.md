# ADR-0052: Task-oriented attention and automatic signal evaluation

Status: Accepted

## Decision

Fresh attention transport is version 2 (metric_version remains 2). For every selected agent, run all
bundled policies. Initially the collection contains only low-edit-ratio. No signal selector, plugin
framework, new evaluator or background monitor. Observe and attention share this runner.

Separate applicability (applicable / not_applicable / unknown), measured condition (the existing
match / no_match / indeterminate result), and outcome (triggered / not_triggered / needs_context /
not_applicable / insufficient_evidence / error). Unknown task scope never disables measurements or
becomes no-match. An error takes precedence, followed by known inapplicability, insufficient measurement
evidence, unknown applicability, then the measured matching outcome. Each signal carries policy
identity/version/digest, window, metrics, reasons and retained evidence. One signal's evaluation error
does not erase other signal results. Shared acquisition/identity failures remain explicit failures.

--task is an optional cohort classification override, not activation. Repeatable --agent-task ID=TASK
adds exact per-agent overrides for mixed cohorts; reject duplicates, malformed inputs and unselected
IDs. Do not guess task classes from labels, language, tool counts or free-text keyword rules.

Return bounded task context from captured unique Pi launch/invocation joins, or owned user-message
text where launch context is absent. Preserve source/event references, uncertain joins, total/omitted
counts and clipping. At most two entries per room and 1024 UTF-8 text bytes per entry; newest canonical
entries are display selection, never inferred governing attempts. Do not emit reasoning or opaque
payloads. The skill may interpret complete applicable task evidence alongside measured conditions,
but must distinguish its interpretation from the retained native applicability/outcome. Retrieve more
evidence only when genuinely missing; do not rerun the detector merely to attach an interpretation.

Keep the 1–20 agent bounds, independently scoped observation sample, 200KB JSON / 100K-character native
attention limits, 200KB observe limit and 4MiB retained attention limit. Output caps do not bound source
processing. Count agent selections separately from signal evaluations. Snapshot retention remains
private and expiring; no notebook mutations. Inspect provides the union of successful signal work
windows plus per-signal window references and task context, with existing evidence pagination.

Compatibility: load and render v1 captures using their historical semantics without re-evaluation.
Fresh v2 pages add policies and signals; singular policy / room metrics, window and result remain
explicit first-signal compatibility projections (unknown scope maps to needs_context, never historical
inapplicable). Clients must consult version and plural fields; no silent fallback to v1 semantics.
The legacy Policy.Evaluate API stays unchanged; a separate condition/applicability API composes it.

## Verification

Test automatic unknown-task measurements, task overrides/mixed cohorts, all status distinctions,
multiple test-only policies and error isolation, per-policy windows, task-evidence provenance and
bounds, old replay and new inspect, independent observation scope, output limits, and the served
single-command recipe on Pi/SDK/Claude. Distinguish executable/native checks from conversational replay.
