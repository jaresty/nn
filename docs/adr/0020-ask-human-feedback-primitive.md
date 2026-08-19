# ADR-0020: `nn ask` — routed human-feedback primitive

## Status

Accepted

## Context

We want a way to get **human feedback into the notebook**: an agent (or human)
reaches a point where only a person can supply the missing knowledge — a sketch,
a judgment on a proposal, a marked-up document — and that response needs to come
back into `nn`'s workflow as structured data.

The shaping insight from design discussion is that the primitive is a **job**,
not a UI. Strip away every current tool and the persistent pressure is: *"I have
a knowledge gap only a human can close, and I need to capture their response."*
That job holds whether the answer is spatial (a diagram), textual (annotations on
prose), or a richer bespoke web interaction. The surface is therefore a *routing
decision within one operation*, not the operation itself.

Two concrete surfaces motivated the design:

- **Excalidraw** — an npm React component we can **embed and host** ourselves.
- **Plannotator** (https://docs.plannotator.ai) — *not* an embeddable web app but
  a **self-contained agent tool with its own CLI, hooks, and plugins**. It cannot
  live behind our server; it is a peer we **invoke**.

That asymmetry is load-bearing: it forces two adapter kinds (hosted vs.
delegated) rather than one uniform "surface behind our endpoints" model.

### Constraints and prior decisions

- **Process disposable, session durable.** A persistent daemon is not required.
  The real constraint is that *some* bridge must exist while a browser is live —
  a genuine security boundary, not a tool quirk. A short-lived process that owns a
  `localhost` server for the duration of one blocking call is not a daemon.
- **Thin result envelope; the LLM interprets.** The result is a question/answer,
  not an artifact-to-note pipeline. What becomes of the answer — one note, many,
  updates, a `graph apply` changeset, or *nothing* (pure ephemeral context) — is a
  downstream, agent-side decision, **out of scope** for this mechanism. We do not
  standardize a semantic annotation schema, because Excalidraw's element bag and
  Plannotator's comment/selection model would fit it badly.
- **File-storage-in-`nn` is deferred.** Results live at a known session path and
  are referenced by path. Whether `.excalidraw`/`.png` ever become first-class
  notebook files is a separate, unresolved conversation.

## Decision

Introduce **`nn ask`**: a single primitive that prepares a human-interaction
session, dispatches it to a **surface** chosen by the shape of the question,
blocks until the human submits, and returns a **thin result envelope** at a known
path. The agent decides what, if anything, the answer becomes in the graph.

```
   agent ── "I need to ask X" ──► nn ask ──► router ──► surface adapter ──► FeedbackResult
                                             (question    (hosted or         (thin envelope
                                              shape →       delegated)         at known path)
                                              modality)                            │
                                                                                   ▼
                                                            agent reads → note(s) / update /
                                                            graph apply / nothing (ephemeral)
```

### Modalities (built-in surfaces)

- **spatial** → **canvas / Excalidraw** — *hosted*. `nn` owns an ephemeral
  `localhost` server, an embedded UI bundle, and a small session protocol
  (`GET /session`, `PUT /draft`, `POST /submit`, `POST /cancel`).
- **textual** → **document / Plannotator** — *delegated*. `nn` shells out to the
  `plannotator` binary and reads back the result it writes.
- **web** → **hosted** bespoke React surface (same hosting mechanism as canvas).

### Two adapter kinds

- **Hosted surface** — `nn` owns the server, endpoints, and embedded UI. Its
  dependency is its own code (Excalidraw, web).
- **Delegated surface** — `nn` invokes a peer binary that owns its own lifecycle
  (Plannotator). The interface is a **contract**: given a prepared request spec
  and a place to write, the peer runs the interaction, writes a result there, and
  exits.

### The three-stage shape

- **FeedbackRequest** — prepared *before* launch: intent/instructions, read-only
  context, an optional editable workspace (absent workspace = bootstrap/create),
  and an output spec. The surface renders this; it never infers the agent's intent.
- **FeedbackSession** — the live interaction; drafts persist so browser/process
  failure is recoverable.
- **FeedbackResult** — a thin envelope naming `surface`, `status`, and
  `artifacts[]` (format + path). Surface-specific shape lives *inside* the
  referenced files; the LLM reads them.

```json
{
  "id": "01K2ABC",
  "surface": "canvas",
  "status": "submitted",
  "artifacts": [
    { "format": "excalidraw", "path": "result.excalidraw" },
    { "format": "png",        "path": "result.png" }
  ]
}
```

Plannotator emits a different native format behind the same envelope (e.g.
`{ "format": "markdown-annotations", "path": "result.md" }`).

## Consequences

- **One primitive, many surfaces.** Routing is coherent because it keys on
  question shape (spatial / textual / web), all serving the same job. This is why
  Plannotator is a *dispatch target*, not a competing architectural center.
- **`nn` stays note-agnostic at the boundary.** Context need not be notes (e.g.
  reviewing HTML designs that never enter `nn`), and the result need not become a
  note. The mechanism ends at "structured response at a known path."
- **Delegation keeps `nn` independent without a clunky handoff** — the contract
  (request in, result out, exit) is the entire interface; no `nn` internals are
  exposed to the peer.
- **We do not build document annotation ourselves.** Plannotator already covers
  it; build effort concentrates on the canvas surface it does not cover.

### Scope boundary (what would break the generalization)

Routed-`ask` covers **one-shot, prepared-request surfaces**: prepare a request →
human submits → result at a known path. All three named modalities fit this. A
surface requiring a **live/streaming** loop (continuous real-time collaboration,
no single submit) falls *outside* this abstraction and would need a second
interaction mode before the model is stable. This is the edge to watch.

## Implementation phases

Ordered to de-risk: fix the contract first, then prove the ephemeral-session
invariant on one hosted surface, then prove the abstraction holds by adding the
*delegated* adapter kind (not a second hosted one). Each phase ends with a
runnable `nn ask` invocation.

- **Phase 0 — Contracts (no UI).** Define `FeedbackRequest`, `FeedbackResult`
  (thin envelope: `surface`, `status`, `artifacts[]`), and the session directory
  layout (`~/.config/nn/feedback/<id>/` with `request.json`, `draft.json`,
  `result.json` + native artifacts). Surfaces are built against this fixed target.
- **Phase 1 — Session lifecycle + ephemeral server (canvas first).** Short-lived
  `localhost` server with `GET /session`, `PUT /draft`, `POST /submit`,
  `POST /cancel`; process blocks then exits on submit. Embed Excalidraw
  (`initialData` from request, `onChange` → debounced draft, Done → submit); write
  `result.excalidraw` + `result.png` + envelope. Draft persistence gives
  `nn ask resume <id>` recovery. Proves the ephemeral-process/durable-session
  invariant end-to-end.
- **Phase 2 — Surface abstraction + delegated Plannotator.** Extract the surface
  adapter interface (hosted vs. delegated); implement the delegated adapter by
  invoking `plannotator` on the contract (request path in → result path out →
  exit); normalize its native output under the same envelope.
- **Phase 3 — Web surface.** Reuse Phase 1 hosting for a bespoke React surface,
  confirming hosting generalizes beyond Excalidraw. (Candidate for deferral until
  a concrete web-modality ask exists.)
- **Phase 4 — Agent-facing surface + interpretation boundary.**
  `nn ask --surface <canvas|document|web>` (explicit selection only — inference is
  deferred), blocking call returning the envelope/path; confirm nothing files the
  result automatically — the agent reads and decides.

## Deferred (explicitly out of scope for this ADR)

- **User-extensible routes.** The intended direction is to let routes live **in
  the graph** as ` ```nn-ask-route ` fenced blocks in note bodies — mirroring the
  Datalog rules engine (ADR-0019), where "Rules should live in the Markdown, not
  in a separate config file" and built-in invariants are extended by note-body
  blocks with per-note-ID provenance and skip-on-malformed resilience. This gives
  routes Git versioning, review, and rollback for free, and — because a route's
  `command` is *executable* (unlike an inert Datalog rule) — a graph-native trust
  gate (e.g. load command-bearing routes only from promoted notes) that a plain
  config file could not offer. **Deferred**; not decided here.
- **Routing trigger.** Whether modality is passed explicitly
  (`nn ask --surface canvas`), inferred from question shape, or both
  (explicit overrides inferred). Deferred with extensibility, since addressable
  user routes push toward explicit-or-hinted selection.
- **File storage in `nn`.** Whether result artifacts become first-class notebook
  files. Deferred (separate conversation).
- **Semantic result layer.** No shared annotation schema; each surface writes
  native output and the LLM interprets it.
