# ADR-0022: `nn ask` MCP/skill surface — an async, LLM-driven handshake (no new command)

## Status

Accepted (documentation-only). Implemented as a documented mode in the `nn ask`
block of the cli-reference virtual protocol (`cmd/nn/cmd/show.go`), so it appears
in `nn show --global` and the CLI reference next to `--surface canvas|document|
graph`. Supersedes earlier drafts of this ADR that proposed `nn` commands
(`start`/`submit`/`resume`), an opener interface, or `nn` acting as an MCP client
— all abandoned.

## Context

ADR-0020 established `nn ask` as a routed human-feedback primitive: the *job* is
"ask a human to close a knowledge gap and capture their structured response"; the
*surface* is a routing decision keyed on the shape of the answer. The three
surfaces that exist today — canvas, document (plannotator), graph — are all
**nn-driven and synchronous**: `nn` hosts (canvas/graph) or delegates
(document) the surface and **blocks until the human submits**, leaving a thin
`result.json` the agent reads.

The desire: reach a human through an **arbitrary MCP server or skill** — e.g. a
wireframe MCP (call the tool to provision a wireframe, then a browser opens for
the human to annotate). Answers here can be **asynchronous** (the human works on
their own clock), and the tool is one the **LLM**, not `nn`, knows how to call.

## The collapse (why there is no new command)

The design went through several simplifications, each removing machinery:

1. **`nn` is not the MCP client.** The LLM driving `nn` already is one — it can
   call any MCP tool or skill natively. So `nn` should not call MCP, spawn
   servers, or know about openers/tools/transports. This removed
   `--async`/`--server`/`--tool`/`--open`.
2. **Same agent throughout.** The agent that opens the surface is the one that
   reads the answer, so there is no cross-agent handoff and no `resume`/poll.
3. **"Me" is the LLM.** The "tell me when you're done so I can read it after"
   reader is the *LLM*, not `nn` and not the human directly. So `nn` neither
   stores nor executes anything — no session, no `result.json`, no correlation
   id, no `start`/`submit` commands.

What remains is a **handshake convention** the LLM runs. `nn`'s only role is to
*carry* that convention so it is discoverable — exactly how `nn show --global`
carries protocols the LLM then follows.

## Decision

Document an **MCP/skill surface** in the `nn ask` cli-reference block as an
async, LLM-driven handshake, distinct from the nn-driven synchronous surfaces:

1. **Invoke** the MCP tool or skill and populate it with the question + context.
2. **Wait** for the human to finish in that surface.
3. **Read back** the surface's contents via the same MCP/skill.
4. **Decide** via the thin-envelope discipline (ADR-0020) what, if anything,
   becomes a note.

- **No `nn` command, no session, no `result.json`, no `--surface` flag.** The
  LLM does all four steps; `nn` only documents the convention.
- **Discoverability is the deliverable.** The mode lives in the `nn ask` block of
  the cli-reference virtual protocol (`show.go`; virtual protocols are defined
  there per note 5108), so it surfaces at session start and in the reference
  alongside `--surface canvas|document|graph`. Marker string: `MCP/skill`.
- **Thin envelope unchanged (ADR-0020).** The read-back result names artifacts;
  nothing is filed automatically — step 4 is the agent's decision.

## Consequences

- **Near-zero code.** One documentation line in `show.go`; no MCP client library,
  no session store, no new verbs. Delivered through the bar build gate as a
  virtual-protocol edit.
- **MCP and skills work identically** because `nn` is ignorant of the opener by
  construction — the handshake never names a specific transport.
- The async human-feedback need is met **without** `nn` growing an async
  execution model; continuity across the human's wait lives in the LLM, which is
  present throughout.
- Establishes a fourth ask surface *class* (LLM-driven) beside the nn-driven
  ones, keeping the ADR-0020 job/surface model coherent.

## Deferred / rejected (explicitly out of scope)

- **`nn` as MCP client / opener interface / `start`-`submit`-`resume` commands.**
  Rejected — the LLM is already the client and is present throughout.
- **Cross-agent handoff** (answer arrives after the opening agent is gone). Would
  need the surface to write to an `nn`-watched path and a real poll/resume; not
  modeled here.
- **Synchronous compute MCP tools.** Not "ask a human"; the LLM calls those
  natively. Out of scope.
- **Routing inference.** As in ADR-0020/0021, surface selection is explicit.
