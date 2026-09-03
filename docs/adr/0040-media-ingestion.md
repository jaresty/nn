# ADR-0040: Media ingestion produces provenance-preserving evidence bundles

**Status:** Proposed
**Date:** 2026-09-02

## Context

`nn` can capture text from files and the web, but audio and video currently require an operator to assemble an external workflow. A recent video review used `ffprobe`/`ffmpeg` metadata inspection, a contact sheet sampled every 45 seconds, and individual frames at selected timestamps. That made a 5:53 recording inspectable without watching it continuously, but the resulting observations, commands, timestamps, and coverage limitations had to be recorded manually.

Transcription can improve coverage, but it is not equivalent to visual review and no single transcription runtime should become a hard dependency. Local engines such as Parakeet may be appropriate on supported machines; other installations may prefer `whisper.cpp`, `faster-whisper`, a hosted service, or no transcription at all.

Integrated notebook claims must not imply that an entire recording was reviewed when only sampled frames were inspected. Generated images, transcripts, and manifests can also be too large or unsuitable for storage directly inside note Markdown. Because files are truth and the SQLite index is a rebuildable cache, media ingestion needs an explicit durable-artifact boundary rather than hiding outputs in the index.

## Decision

Add a media-ingestion workflow that produces a provenance-preserving evidence bundle and exposes that bundle as bounded multimodal context for agent-mediated integration. Media extraction does not directly create notebook notes.

The intended command family is:

```text
nn media inspect <file>
nn media sample <file> [--every DURATION] [--frames TIMESTAMPS] [--contact-sheet]
nn media transcribe <file> [--engine NAME]
nn media prepare <file> [sampling and transcription flags]
nn media context --run <run-id>
```

`prepare` settles the requested extraction stages and publishes a durable run. `context` retrieves a bounded, machine-readable integration packet containing provenance, qualified coverage, transcript segments, and actual image artifacts suitable for multimodal model input. Neither command writes notebook truth. Natural requests such as “integrate this media run” dispatch to the existing agent-owned **✚ Integrate** workflow in `nn-navigate`; `integrate` is not a new Cobra subcommand.

Exact command names and flags may be refined during implementation, but the following boundaries are part of this decision.

### Expose one versioned command-result contract

Every media command emits the same versioned result envelope. It identifies the command, run, source, bundle and manifest locators, overall outcome, per-stage outcomes, warnings, and structured errors. Human output is a projection of that envelope.

Stdout carries only the primary result. Stderr carries progress and diagnostics. JSON mode emits one final object; a future streaming mode must be explicitly selected and use a separately versioned event format. Redirecting stdout changes presentation, never command semantics.

Stages distinguish at least `not_requested`, `pending`, `running`, `succeeded`, `partial`, `inapplicable`, `unavailable`, `failed`, `interrupted`, and `cancelled`. Overall outcomes are derived from requested stage states rather than asserted independently. Complete success, usable partial evidence, invocation or prerequisite failure, processing failure, and cancellation are distinct symbolic outcomes. Exact numeric exit codes remain an implementation choice, but partial completion is non-success and remains machine-detectable while preserving the run and manifest locators.

Every media choice has a flag or configuration equivalent. Non-TTY execution never prompts, launches a picker or editor, or silently selects a different engine. The CLI exposes effective stage planning, engine capability discovery, and run discovery in both human and machine-readable forms.

### Make dependency remediation explicit

`nn media doctor` detects required tools, transcription engines, versions, and operation-specific capabilities. Missing or incompatible dependencies produce structured error codes plus platform- and architecture-specific installation guidance when nn has a known recipe. Unsupported environments receive instructions-only diagnostics rather than guessed commands.

Dependency detection is automatic; installation is never automatic or silent. In a TTY, nn may offer to run a known installation recipe only after showing the exact package-manager command and receiving confirmation. Every prompt has a named flag equivalent. Outside a TTY, nn never prompts: it fails with actionable guidance and prints the equivalent explicit install command.

An explicit install operation may install only a named dependency from a known recipe, for example through `nn media doctor --install <dependency>` or an equivalent command finalized during implementation. After installation, nn reruns capability discovery and reports the effective executable, version, and supported operations. Model downloads, hosted-engine credentials, remote media transfer, and license acceptance require separate explicit consent and are not implied by dependency installation.

### Probe media before analysis

`nn media inspect` uses `ffprobe` to emit machine-readable stream and container metadata, including duration, codecs, dimensions, frame rate, and audio-stream presence. The normalized metadata and the source file identity become part of the evidence manifest.

The manifest records at least the source path supplied by the user, source size, modification time, and a stable source identity. The source-identity policy may use an eager, incremental, deferred, or custody-dependent content digest, but it must prevent a later file at the same path from being silently mistaken for the analyzed recording and must disclose the strength of the identity claim.

### Treat visual sampling as bounded evidence

Video sampling uses `ffmpeg` to create:

- contact sheets at a requested interval;
- individual frames at explicit timestamps; and
- a manifest entry for every generated artifact, including its timestamp and effective extraction parameters.

Sampling defaults must remain visible in command output and integration context. A contact sheet or selected frame is evidence about the sampled instant, not evidence that intervening content was reviewed.

Generated visual artifacts are ordinary files in the evidence bundle. They are not embedded as binary data in note Markdown or stored in SQLite.

### Keep transcription behind an adapter

Transcription is optional and is invoked through a model-agnostic adapter interface. An adapter receives an extracted or original audio input plus configuration and returns:

- timestamped transcript segments;
- engine identity and version when available;
- language and confidence metadata when supplied by the engine; and
- the effective invocation configuration without secrets.

Parakeet is a supported adapter candidate, not a required runtime or privileged semantic default. Additional adapters may wrap `whisper.cpp`, `faster-whisper`, hosted services, or a user-configured command. Engine availability is detected explicitly; missing optional engines produce an actionable error rather than silently changing engines.

The first implementation should prefer subprocess adapters over linking model runtimes into the `nn` binary. This keeps the Go CLI small, preserves installation choice, and allows engines to evolve independently.

### Preserve provenance and coverage limitations

Each ingestion run writes a machine-readable manifest alongside its artifacts. The manifest records:

- source identity and probed metadata;
- tool and adapter identities and versions when available;
- requested and effective sampling parameters;
- extracted-frame timestamps;
- transcript segment timestamps;
- generated artifact paths and digests;
- start and completion state for each pipeline stage; and
- warnings, omitted stages, and known coverage gaps.

The manifest is the canonical run record for declared provenance and coverage. It is not independent proof that a tool decoded correctly or that a generated transcript or interpretation is accurate.

Manifest records have stable identifiers and explicit epistemic kinds, including source identity, extracted artifact, generated derivative, tool report, authored observation, and inference. Every derivative identifies its immediate inputs and transformation. Every authored observation or inference records authorship, creation time, method, and supporting evidence identifiers.

Media-relative timestamps identify their timeline and time base and distinguish requested from effective positions when they can differ. Wall-clock event times are a separate timestamp class. Transcript coverage records process completion, attempted temporal span, represented span, known gaps, and confidence availability separately; it never implies semantic accuracy. Video samples cover instants, not intervening intervals. Unknown, known absent, none observed, not requested, inapplicable, and failed to determine are distinct states.

Human-readable rendering is derived from typed manifest and authored claim records. It must preserve run identity, evidence references, authorship, timestamps, failures, and coverage qualifiers without inventing stronger claims.

A bounded integration packet separates source metadata, coverage, transcript evidence, visual artifacts, and manifest provenance. It contains no generated observations or inferences. `Coverage` states the audio span attempted and represented, processing outcome, known decode or transcript gaps, which video instants were sampled, and which requested stages failed or were skipped. It does not imply transcript correctness or coverage of unsampled intervals.

During **✚ Integrate**, the active LLM receives the packet's actual sampled images as multimodal inputs together with bounded transcript segments and provenance. It distinguishes sourced claims from generated interpretation, compares proposed durable claims with existing notebook truth, and presents exact note and relationship changes for human approval. The resulting notebook changes may include a compact source-index note plus atomic concept, argument, observation, or model notes; they must not default to a transcript dump, artifact inventory, or one oversized generated note.

### Store artifacts outside notes and the index

Evidence bundles live in a configurable artifact root outside the notebook by default. Source content identity, acquisition identity, run identity, bundle identity, and artifact identity are distinct. A bundle contains its manifest, transcript representation, frames, contact sheets, and stage logs needed for provenance.

Notes reference stable bundle identities and artifact roles rather than relying only on absolute filesystem paths. Configuration resolves bundle identities to current locations. Bundle-internal paths are relative, normalized, traversal-safe, and portable. The initial implementation supports local filesystem bundles; remote or content-addressed stores may be added later without changing the identity or reference contracts.

Each run records an explicit source-custody policy: reference in place, copy into managed custody, move into managed custody, or identity-only. Runs are assembled in a staging state and become discoverable as completed bundles only at an atomic publication point after required records and artifacts are durable and validated. Interrupted, failed, completed, superseded, repaired, and intentionally deleted states remain distinguishable.

Retention is policy-driven and conservative. Temporary stage files may be reclaimed automatically. Completed evidence is never deleted solely because no current note references it; deletion requires explainable eligibility, a dry run, and explicit confirmation. Repair never silently rewrites historical evidence: relocation updates resolution state, while regenerated artifacts create a new run or an auditable repair record. Privacy policy covers generated transcripts, images, logs, paths, permissions, remote transfer, trust domains, and intentional erasure, including tombstones where needed to distinguish erasure from accidental breakage.

The SQLite index may index bounded textual projections, but it does not become the only copy of a transcript, manifest, claim record, or generated artifact.

### Make integration agent-mediated, proposed, and atomic

Inspection, sampling, transcription, preparation, run discovery, and context retrieval never create a note. The media CLI owns deterministic extraction, durable artifacts, qualified coverage, and bounded context transport. It does not decide what the media means or which claims belong in the notebook.

The existing agent-owned **✚ Integrate** workflow owns all notebook mutation. Given a media run, it reads the complete bounded evidence packet, inspects actual sampled images, reasons over transcript and visual evidence together, searches and reads relevant existing notes, and presents a non-mutating proposal. The proposal identifies exact evidence boundaries, sourced claims versus interpretation, affected notes and relationships, complete before/after intent, uncertainty, authority limits, and expected semantic operation and commit count. Every mutation requires explicit human approval.

Approved changes execute only through existing `nn` mutation workflows and retain their safeguards. Prefer one supported atomic changeset, such as `nn graph apply`, when it can represent the complete approved creation-and-link operation. If a heterogeneous proposal cannot be applied atomically, report that boundary and narrow or separately authorize operations rather than hiding multiple commits. Generated images, transcripts, and manifests remain outside the notebook; notes refer to stable run, bundle, and evidence identities when provenance is needed.

Re-running ingestion creates a new run rather than mutating prior evidence in place. Runs may record `retry-of`, `resumed-from`, `extends-coverage-of`, or `supersedes` lineage without mutating predecessors. Run discovery resolves a run identifier to its current manifest and reports source identity and stage states. Context retrieval accepts an existing completed or partial run, so a later integration attempt never forces media reprocessing. Resume may reuse validated artifacts only when source identity, effective parameters, and manifest state match; reuse is recorded explicitly.

### Security and operational boundaries

All external tools execute through a shared runner contract. A runner receives an argument vector, filtered environment, controlled working directory, explicit streams and paths, deadline and cancellation signal, log limits, and resource policy. It returns structured startup, exit, signal, timeout, cancellation, diagnostic, and artifact-validation results. Process separation is not presented as sandboxing.

Capability discovery is operation-specific and side-effect-bounded. It records executable identity and the capabilities relied upon without downloading models, allocating accelerators, contacting hosted services, or mutating caches unless explicitly authorized. `ffmpeg`, `ffprobe`, and transcription engines are replaceable implementations that must pass conformance tests for malformed output, unsupported capabilities, partial output, cancellation, timeout, cleanup, secret redaction, timestamps, and resource limits.

Transcription adapters declare accepted input modes, timestamp origin, streaming and partial-output behavior, confidence semantics, model/cache requirements, and trust class. The core may provide canonical prepared audio, while original-media input remains an explicitly declared capability. Parakeet remains experimental until a concrete distribution proves conformance, timestamp behavior, resource use, packaging, and licensing.

Manifests omit credentials and redact configured secret values. Hosted transcription is opt-in because it transfers media outside the machine; local execution is preferred only when an explicitly configured local adapter is available.

The workflow does not make semantic claims from frames or transcripts by itself. It extracts evidence and formats provenance. LLM- or human-authored observations and inferences remain a later, explicit and attributed step.

## Consequences

- Long recordings become inspectable through repeatable, timestamped sampling instead of requiring continuous playback.
- Transcription can improve audio coverage without coupling `nn` to one model runtime.
- Notes remain small and auditable because large artifacts live in evidence bundles.
- Agent-mediated proposals preserve sampled intervals and failed or omitted stages while preventing raw transcript or artifact dumps from masquerading as integrated knowledge.
- Users must install `ffmpeg`/`ffprobe` for media operations and separately install any selected transcription engine.
- External artifact roots introduce lifecycle concerns: bundles can be moved or deleted independently of notes, so future maintenance tooling should detect broken artifact references.
- Content digests and generated images consume additional time and storage, especially for large recordings.
- Partial pipeline success requires richer status reporting than a single success/failure exit message.

## Implementation handoff

Implementation is blocked until four versioned contracts and one recovery path are specified and tested with fixtures:

1. Define the command-result contract, symbolic outcomes, stage-state machine, non-TTY behavior, and run discovery.
2. Define the typed evidence-manifest and claim model, including derivations, authorship, timestamps, and qualified coverage.
3. Define bundle identity, resolution, custody, publication, retention, repair, erasure, and lineage.
4. Define runner, capability-provider, and transcription-adapter conformance contracts.
5. Prove that a completed or partial run can be discovered and transported as bounded multimodal integration context without media reprocessing or notebook mutation.

After that gate, implement this decision in atomic stages:

1. Implement the shared runner, capability model, manifest state machine, and artifact publication boundary.
2. Add source probing and normalized metadata output using `ffprobe`.
3. Add deterministic frame extraction and contact-sheet generation using `ffmpeg`.
4. Implement the transcription-adapter conformance harness and evaluate Parakeet as an experimental candidate.
5. Add bounded context retrieval from existing runs, including actual image attachments and transcript segmentation, with no note-write capability.
6. Connect media-run context to the existing agent-owned **✚ Integrate** workflow and its proposal/approval boundary.
7. Add run discovery, artifact-reference health checks, conservative maintenance tooling, and documentation.

The media package should own process execution, manifest state, artifact generation, and bounded evidence retrieval. The command package should own flags, TTY behavior, and rendering. The agent-owned **✚ Integrate** workflow should mediate interpretation and dispatch approved notebook changes through existing note, graph, and backend operations so Git atomicity remains explicit.

Tests must cover:

- `ffprobe` JSON normalization from checked-in fixtures;
- deterministic sampling plans and timestamp formatting without requiring a codec runtime;
- subprocess argument construction without shell interpolation;
- adapter discovery, explicit missing-engine errors, and secret redaction;
- dependency diagnostics for supported and unsupported OS/architecture combinations;
- exact-command display, confirmation, non-TTY refusal to prompt, and post-install capability recheck;
- separation of package installation from model downloads, credentials, license acceptance, and remote transfer consent;
- manifest transitions for success, partial success, and failure;
- bounded integration packets that distinguish transcript evidence, image evidence, provenance, and coverage;
- refusal to represent sampled video as fully reviewed;
- proof that media commands do not mutate notebook truth;
- **✚ Integrate** proposal, approval, and atomic-application behavior for media evidence; and
- integration tests gated on locally available `ffmpeg`/`ffprobe` using small generated media fixtures.

Before implementation, decide the artifact-root default, manifest versioning policy, generic transcription-adapter conformance contract, and acceptance criteria for a Parakeet experiment. Parakeet becomes supported only after a concrete distribution passes those criteria. The mechanism-level choices remain intentionally open because they depend on packaging and runtime experiments rather than the architectural boundary decided here.

## Alternatives rejected

### Add only `nn transcribe`

Rejected because transcription cannot recover visual layout, labels, alerts, drawers, or other screen state, and it would omit the provenance needed to distinguish complete audio coverage from sampled video coverage.

### Make Parakeet a required dependency

Rejected because hardware, packaging, language support, and model runtimes vary. The durable interface is timestamped transcription with provenance, not one engine.

### Embed generated media and full transcripts in notes

Rejected because large binaries and transcripts would inflate the Git-backed notebook, degrade review ergonomics, and blur the boundary between atomic notes and source evidence.

### Store artifacts only in SQLite

Rejected because the index is a rebuildable cache and must not become the sole source of truth for evidence.

### Automatically infer observations from sampled frames

Rejected because extraction and interpretation have different epistemic status. Automatic inference may be added as an explicit downstream adapter, but its claims must remain separate from timestamped source evidence.

### Treat any successful sample as full review

Rejected because unsampled intervals may contain materially different content. Coverage must be represented explicitly and conservatively.
