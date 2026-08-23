# ADR-0022: `nn ask` resumable sessions — nn owns a durable pending-question envelope; the LLM couriers the result

## Status

Proposed. Design only; no implementation. Supersedes an earlier draft of this
ADR that made `nn` an MCP client with a pluggable opener interface — that design
is abandoned in favor of the simpler courier model below.

## Context

ADR-0020 established `nn ask` as a routed human-feedback primitive: the *job* is
"ask a human to close a knowledge gap and capture their structured response"; the
*surface* is a routing decision keyed on the shape of the answer. The two adapter
kinds that exist today — **hosted** (canvas, graph) and **delegated** (document/
Plannotator) — are both **synchronous**: `runAsk` "prepares a feedback session,
launches its surface via the injected open hook, blocks until the human submits
or cancels, and prints the result path." The result is at a known path *before*
the call returns.

The desire: let `nn ask` reach a human through an **arbitrary MCP server or
skill** — e.g. wiretext (a wireframe/Figma-style MCP: call the tool to provision
a wireframe, then open a browser on it for the human to annotate). Human answers
here can be **asynchronous** — the person works on their own clock, so the answer
is decoupled in time from the invocation.

### The key simplification: nn is not the MCP client

An earlier version of this design had `nn` act as the MCP client — spawning
servers, calling `tools/call`, with `--server`/`--tool`/`--open` flags and a
pluggable `opener` interface. That is unnecessary complexity: **the LLM driving
`nn` is already an MCP (and skill) client.** It can call any MCP tool or invoke
any skill natively.

So `nn` should not call MCP, spawn servers, or know about openers, tools, or
transports at all. `nn` owns **only a resumable session envelope**; the LLM does
the calling and couriers the result back. This makes MCP and skills work
*identically* — `nn` is opener-agnostic because it never touches the opener — and
it removes the need for `--async`, `--server`, `--tool`, and `--open` entirely.

## Decision

`nn ask` gains a **resumable-session (courier) mode**. `nn`'s entire
responsibility is a durable scratchpad for an in-flight human question:

```
1. START   nn ask start --instructions "review this wireframe" [--context ...]
                └─ nn creates a durable PENDING session, returns a session <id>.
                   nn calls nothing and never blocks.

2. LLM acts   the LLM calls the wiretext MCP tool (or a skill) ITSELF, with its
              own client. the surface opens; the human works; the LLM receives
              the answer back in its own context.

3. SUBMIT  nn ask submit <id> --from <file>        (or --result "<json>")
                └─ the LLM couriers the answer into the session; nn stores it
                   and marks the session resolved.
```

**No `resume` verb.** The agent that calls `start` is the same agent that calls
the MCP/skill and receives the answer — so it already holds the result when it
calls `submit`. It never needs to *read back* a result it is holding, so a
`resume`/continue verb is dead weight. The word "resume" also smuggles in a
false assumption: that `nn` suspends and continues a flow. `nn` runs no flow — it
holds a slot. A genuine `resume`/poll belongs only in the *other* model, where
the answer arrives after the opening agent is gone (cross-agent handoff) and the
surface must write to an `nn`-watched path — the model rejected below. For simple
read-back convenience, a plain getter (`nn ask show <id>`, `nn ask list`) covers
it without suspend/continue semantics.

- **No `--async` flag.** `start` never blocks, so there is no synchronous variant
  to distinguish it from. Sync-vs-async was only a distinction when *nn* did the
  opening; since the LLM opens, `start` is always non-blocking.
- **No `--server`/`--tool`/`--open`.** The LLM is the client; `nn` needs to know
  nothing about MCP or skills. This is what makes both work with zero MCP code in
  `nn`.
- **Courier submission (chosen).** The LLM calls the tool, receives the result in
  its own context, and pushes it into the session via `submit`. `nn` never
  watches a path, polls, or waits for a file that might never arrive. (Rejected
  alternative: the surface writes to an `nn`-owned `resultPath` and `resume`
  reads it — that reintroduces path-watching and a may-never-arrive file.)
- **Correlation id = session id.** Because `start` returns before the answer
  exists, a token ties the later `submit` back to the request; the session id
  serves as that token. Synchronous surfaces never needed one (the blocking call
  *was* the correlation).
- **Thin envelope unchanged (ADR-0020).** The couriered result names native
  artifacts; nothing is filed automatically. What becomes a note, an `nn link`,
  or nothing is a downstream agent decision.

Command surface:

| Command | Does |
|---|---|
| `nn ask start` | create a pending session; return its id (and the prepared request) |
| `nn ask submit <id>` | store the LLM-couriered result; mark resolved |
| `nn ask show <id>` (optional) | plain read-back of a stored result |
| `nn ask list` (optional) | show pending sessions |

**Possible further collapse.** `start` earns its place only if a durable
"open question" slot is wanted *between* the tool call and the submit — trackable
via `list`, surviving a crash. If that durability is not needed, the whole thing
collapses to a single create-and-store `submit`. Whether to keep `start` is the
one open question; everything else is fixed.

## Consequences

- `nn`'s new code is small and MCP-free: a durable session store plus two verbs
  (`start`/`submit`, with optional `show`/`list` getters). No MCP client library,
  no opener interface, no `runMCP` hook, no server/transport config.
- The first **durable, pending** ask session (existing sessions are ephemeral and
  one-shot). Session dirs already have a 7-day retention window (`feedbackRetention`
  in `ask.go`); a pending session reuses it.
- Works for **any** opener — MCP, skill, or a future one — because `nn` is
  ignorant of the opener by construction. The forcing example is wiretext, but
  nothing is wiretext-specific.
- Moves the "call the surface" responsibility to the LLM, where the client
  already lives — avoiding a second MCP client implementation inside `nn`.

## Deferred (explicitly out of scope for this ADR)

- **Synchronous MCP tools as a surface.** A pure-compute MCP call is not "ask a
  human" and the LLM can already do it natively; not modeled here.
- **`nn` opening the surface itself.** Explicitly rejected — the LLM opens.
- **Routing inference.** As in ADR-0020/0021, mode selection is explicit.
- **Write-back / live conversational loop.** Unchanged from prior ADRs.
