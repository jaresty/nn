# ADR-0022: `nn ask` asynchronous surfaces — open-with-context / submit-out-of-band, via a pluggable opener (MCP, skill, peer-binary)

## Status

Proposed. Design only; no implementation. This ADR captures the shape of the
feature and, more importantly, the **submission problem** it forces — which is
the "second interaction mode" that ADR-0020 and ADR-0021 explicitly deferred.

## The essential shape (not MCP-specific)

The primitive is **not** "invoke an MCP server." MCP is one transport. The
invariant, stated in three moves, is:

1. **Open** — hand some surface the context: the question, its scope, and a
   **correlation id**.
2. **Wait** — the human acts on their own clock, on their own device; the
   invoking call has no reason to stay alive.
3. **Submit** — the result finds its way back to a known path, tied to the
   request by the correlation id.

The *opener* is pluggable. Concrete openers, all interchangeable at move 1:
- an **MCP** `tools/call` that provisions a surface, optionally followed by
  opening it. **wiretext is an MCP server** of this kind: `nn` calls the wiretext
  tool, which produces/hosts a wireframe (Figma-like), and then `nn` **opens the
  browser** on the returned URL for the human to view and annotate. Note the
  opener here is *two moves* — provision (MCP call) then present (browser open) —
  not a single blocking call and not a text-back-from-your-phone flow.
- a **skill** invocation that opens whatever the skill fronts,
- a **peer binary** (the existing delegated pattern, but non-blocking).

Openers therefore split on *how the human is reached*: some (wiretext) call MCP
to provision, then `nn` opens a local browser — the human is present now; others
could hand off entirely to a surface the human reaches on their own later. The
first is browser-open (like `--surface canvas`/`graph`); the second is the
genuinely out-of-band case. **Both still share moves 2 and 3** — the human acts
on their own clock and the result returns by correlation id — which is why they
are one feature; whether a browser is opened locally is an opener detail.

## Context

ADR-0020 established `nn ask` as a routed human-feedback primitive: the *job* is
"ask a human to close a knowledge gap and capture their structured response"; the
*surface* is a routing decision keyed on the shape of the answer. Two adapter
kinds exist today:

- **Hosted** — `nn` owns an ephemeral `localhost` server + embedded UI for the
  duration of one blocking call (canvas/Excalidraw, graph viewer).
- **Delegated (synchronous)** — `nn` invokes a peer binary on a
  request-path-in → result-path-out contract and **blocks until it returns**
  (document/Plannotator, via the `runPlannotator func(argv []string) error`
  injection point in `ask.go`).

Both existing kinds are **one-shot and synchronous**: `runAsk` "prepares a
feedback session, launches its surface via the injected open hook, **blocks until
the human submits or cancels**, and prints the result path." The result lands at
a known path in the session dir before the `nn ask` process exits.

The request here is a **third adapter kind**: point `nn ask` at an arbitrary MCP
server so the human-feedback question can be routed through *any* MCP-exposed
surface (a Slack approval, a ticketing form, a web form, a mobile push a person
answers). The MCP server, not `nn`, owns the UI.

### Why MCP forces a new interaction mode

MCP tool calls come in two shapes, and only one fits the existing model:

- **Synchronous MCP tool** (returns a value in the JSON-RPC response). This is
  just the *delegated adapter generalized* — swap the argv/exit-code transport
  for `tools/call` over JSON-RPC. The call *is* the submit. Buildable today, but
  largely redundant with what an agent already does by calling MCP natively, and
  it isn't really "ask a human" — there is no human. **Out of scope for this
  ADR** (noted only to distinguish it).

- **Asynchronous human surface** (the MCP call *opens* something a person answers
  later; the answer arrives out of band, possibly minutes or hours later, after
  the `nn ask` process would normally have exited). **This is the case this ADR
  is about**, and it does not fit routed-ask, because routed-ask assumes the
  result is available at a known path *before the call returns*. The whole
  difficulty is: **the submission is decoupled in time from the invocation.**

This is exactly the boundary both prior notes drew. From the job/surface concept
(note 4977): *"Routed-ask covers one-shot prepared-request surfaces (prepare →
submit → result at a known path). A live/streaming collaborative surface falls
outside and needs a second interaction mode."* The async MCP surface is a
concrete instance of that deferred second mode.

## The submission problem (the crux)

For a synchronous surface, "submit" is trivial: the surface writes `result.json`
and `nn ask` reads it. For an **asynchronous** surface, the human submits *after*
the invoking call has no reason to still be running. Three possible resolutions,
in increasing order of infrastructure:

1. **Poll a rendezvous path (fits the file-based model best).**
   - `nn ask --surface mcp` calls the MCP tool to *open* the surface, passing it
     a **correlation id** and a **result path** (or a callback command).
     Then `nn ask` **blocks and polls** the session dir for `result.json`, on a
     timeout.
   - The MCP surface (or a small companion) writes `result.json` at that path
     when the human answers. This is the same known-path contract as delegated —
     only the *writer* is remote and the *wait* is long.
   - Pro: reuses the entire ADR-0020 envelope; the only new thing is a bounded
     blocking poll. Con: `nn ask` must stay alive for the human's whole latency,
     which is fine for minutes, wrong for hours.

2. **Detach + resume (two-phase ask).**
   - `nn ask --surface mcp ... --detach` returns *immediately* with a
     **pending session id** after firing the MCP open-call; it does not block.
   - A second command — `nn ask resume <session-id>` — reads `result.json` once
     the human has answered (erroring if still pending). The agent (or a hook)
     calls resume later.
   - Pro: no long-lived process; supports hour/day latencies. Con: introduces a
     durable *pending* session state and a resume verb — the first real break
     from "one-shot prepare→submit→result".

3. **Callback endpoint (`nn` briefly becomes a server for the reply).**
   - `nn ask` (or a lightweight `nn ask serve`) exposes a local callback the MCP
     surface hits on submit, mirroring the graph surface's `/event` relay hook —
     but for a *remote, delayed* submit rather than a local one-shot.
   - Pro: event-driven, no polling. Con: requires the MCP surface to reach the
     callback (networking, auth), heaviest option.

The **correlation id** is common to all three and is the load-bearing new
concept: because submission is decoupled from invocation, every async surface
needs a token that ties a later inbound result back to the specific `ask`
request. Synchronous surfaces never needed one (the blocking call *is* the
correlation).

## Decision

**Deferred pending a chosen submission model.** The decision that matters is the
**async submission mechanism**, not the opener. Recommend **option 2 (detach +
resume)** as the target if pursued, because it is the only one that honestly
supports human latencies (minutes to hours — an approval someone gets to
tomorrow, a wireframe review left open in a tab) without holding a process open,
and it generalizes: a
`pending` session + `nn ask resume` is reusable by *every* opener. Options 1 and
3 are optimizations of the wait, not new capabilities.

Model the **opener as an interface**, not a fixed surface flag:

```
type opener interface {
    // Open hands the surface the prepared request (question + scope +
    // correlationID + resultPath) and returns once the surface has been
    // *opened* — NOT once the human has answered.
    Open(req askRequest) error
}
```

with implementations `mcpOpener` (`tools/call`, optionally followed by a browser
open on a returned URL — this is the wiretext pattern: call the MCP tool to
provision a wireframe, then open the browser on it), `skillOpener`, and
`binaryOpener` (delegated, non-blocking). Each is injected for testing, mirroring
the existing `runPlannotator` hook. `Open` returns once the surface is *up*, not
once the human has answered — so an opener that also opens a local browser (like
wiretext) still returns promptly and the submission still arrives via moves 2–3.

Invocation sketch (not final) — the opener is selected, the async contract is
uniform:

```
nn ask --async --open mcp --server wiretext --tool <open-tool> [--args <json>]
nn ask --async --open mcp --server <name> --tool <open-tool> [--args <json>]
nn ask --async --open skill --skill <name> [--args <json>]
  → opener fires with a correlation id + result path
  → prints: pending session <id>       (does NOT block)
  ... (human answers on their own surface, minutes/hours later) ...
nn ask resume <id>
  → reads result.json (or: "still pending")
```

- The **thin-envelope discipline is unchanged** (ADR-0020): `result.json` names
  native artifacts by path; nothing is filed automatically.
- MCP-, skill-, and link-specific config (which server/skill/URL, transport,
  auth) lives behind its opener; the async machinery (pending session,
  correlation id, resume) is opener-agnostic.

## Consequences

- Introduces the first **asynchronous, durable-pending** ask session — a genuine
  new interaction mode, not another surface. This is the cost, and it is why the
  synchronous MCP case (which needs none of this) is explicitly excluded.
- Adds a **correlation id** to the ask contract for async surfaces.
- Once `detach`/`resume` + correlation exist, other async surfaces (email, SMS,
  long-running human review) become cheap — the MCP surface is the forcing
  function, not the only beneficiary.
- MCP-client configuration (which servers, transport, auth) becomes part of
  `nn`'s config surface — new territory for a tool that has so far only *been*
  called, not *called out* to MCP.

## Deferred (explicitly out of scope for this ADR)

- **Synchronous MCP tools as a surface.** Buildable under the existing delegated
  contract, but redundant with native agent tool-calling and not "ask a human".
  Not pursued here.
- **Which submission model (1/2/3).** Recommended: option 2. Not decided.
- **Routing inference.** As in ADR-0020/0021, `--surface mcp` would be explicit
  selection only.
- **Write-back / live loop.** Unchanged from prior ADRs.
