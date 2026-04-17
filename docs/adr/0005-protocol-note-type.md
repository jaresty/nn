# ADR-0005: Protocol Note Type and nn-workflow Meta-Skill

**Status:** Accepted — implemented
**Date:** 2026-04-16
**Authors:** jaresty

**Implementation log:**
- 2026-04-16 `protocol` added as seventh note type; nn-workflow updated to load protocols at session start

---

## Context

Notes in the Zettelkasten currently serve as declarative knowledge (concepts, arguments,
models, hypotheses, observations, questions). There is no way to encode *operating
instructions* — heuristics the LLM should follow when working in this specific notebook.

The existing skill system (`nn install-skills`, `~/.claude/skills/`) is global and requires
file copying to deploy. This is too heavyweight for experimental or notebook-specific
procedures.

The request: make notes usable as live, per-notebook skill prototypes. Write a note, the
LLM picks it up immediately without any install step. When a protocol matures, graduate it
to a proper skill file.

---

## Decisions

### 1. New note type: `protocol`

A `protocol` note contains imperative instructions the LLM should follow when operating
in this notebook. It is distinct from all existing types:

| Type | Epistemic role |
|---|---|
| `concept` | declarative — defines an idea |
| `argument` | declarative — makes a claim |
| `model` | declarative — describes a framework |
| `hypothesis` | declarative — states a conjecture |
| `observation` | declarative — records a fact |
| `question` | interrogative — poses an open question |
| `protocol` | **imperative** — specifies a procedure to follow |

A `protocol` note is written in second-person imperative ("When you create a new hypothesis
note, immediately link it to…"). The LLM treats the body as a binding operating instruction
for the session.

### 2. `nn-workflow` loads protocols at session start

The `nn-workflow` skill is updated with a **Session Start** step that runs before any other
work:

```
nn list --type protocol --json
```

For each result, `nn show <id>` loads the protocol body into context. The LLM treats all
loaded protocols as additional constraints for the session, scoped to this notebook.

### 3. Scope and lifecycle

- Protocols are **per-notebook** — they live with the knowledge they govern
- Protocols are **ephemeral by design** — experimental procedures belong here, not in global skills
- **Graduation path**: when a protocol proves stable, extract it to `skills/<name>/SKILL.md`
  and run `nn install-skills`
- Protocols participate in the normal Zettelkasten graph — they can be linked to the notes
  they govern, annotated with `source-of` or `refines` relationships

---

## Consequences

- The notebook becomes self-instructing — operating procedures co-evolve with knowledge
- No file copying or install step needed to prototype a new LLM behavior
- Protocol notes are first-class Zettelkasten citizens: versioned in git, linkable, searchable
- The LLM always checks for protocols; an empty result is a no-op

---

## Alternatives Considered

**Tag-based (`--tag protocol` on existing types):** Rejected. A protocol is semantically
distinct — it is imperative, not declarative. Forcing it into `concept` or `model` loses
the type signal that triggers special handling in `nn-workflow`.

**Separate skill file per notebook (`.nn/skills/`):** Rejected. Adds infrastructure without
adding capability. A note is simpler, already versioned, and participates in the graph.

**Global skill with notebook-scoped content:** Rejected. Protocols are per-notebook by
nature; a global skill with conditional logic per notebook defeats the purpose.
