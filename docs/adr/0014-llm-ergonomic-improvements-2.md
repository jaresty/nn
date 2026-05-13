# ADR 0014: LLM Ergonomic Improvements — Search Excerpts, Graph Links, applies_when Flag, Declared-Fail Skip, show --global No-Pipe Rule

## Status

Accepted

## Context

Five usability issues surfaced in session review that degrade LLM agent ergonomics:

1. **`nn list --search` shows no match excerpt.** The LLM cannot judge relevance without
   a follow-up `nn show`, which doubles tool calls for every search.

2. **`nn list --search` does not surface graph links.** The search result row contains no
   signal about how well-connected a note is, even though link density is a strong
   relevance proxy. The search amplifier may also be boosting the wrong signal (link
   count vs. backlink count).

3. **`applies_when` is frontmatter-only.** Setting or updating the `applies_when` field
   on a protocol note requires editing YAML directly. nn's design principle is that every
   field has a named flag equivalent so LLMs never need to hand-edit frontmatter.

4. **PostToolUseFailure hook fires on declared failures.** When an LLM preemptively
   declares that a command is expected to fail (e.g., "Expected FAIL — TDD gate"), the
   hook still requires an `nn list --search` lookup. This creates false-positive
   interruptions and erodes trust in the protocol.

5. **LLMs pipe `nn show --global` through `head` or `tail`.** The session-start prompt
   does not specify the only permitted invocation form. LLMs have been observed running
   `nn show --global | head` or similar, silently truncating the protocol list. All
   global protocols must be visible for the session-start contract to hold.

## Decision

### 1. Match excerpts in `nn list --search`

Add a `matched_excerpt` field to each search result row: the verbatim snippet (≤ 120
characters) from the note body that produced the BM25 hit, with the matched term(s)
highlighted (bold in plain output, `matched_text` in JSON). This eliminates the
follow-up `nn show` for relevance evaluation.

Plain output example:
```
20260417061000-8858  Session start           [protocol]
  excerpt: "...run `nn show --global` now and treat every note body..."
```

JSON: add `"excerpt": "..."` alongside existing fields.

### 2. Graph link signal in `nn list --search`

**First:** audit whether boosting link count or backlink count produces better search
ranking. Run an offline evaluation before changing the amplifier. Only after the audit:

Add a link-count indicator to plain search output (e.g., `↔3` for 3 links). Keep it
opt-in via `--links` flag initially to avoid noise for human readers.

### 3. `--applies-when` flag on `nn update`

Add `--applies-when <value>` to `nn update` as a shorthand for
`--field applies_when --value "..."`. Also support it on `nn new` for protocol notes.
This makes the field settable without touching frontmatter.

```
nn update <id> --applies-when "before any action that reads from an external source"
nn new --type protocol --title "..." --applies-when "..." --content "..."
```

### 4. Declared-fail skip for PostToolUseFailure hook — prompt change

Update the PostToolUseFailure hook prompt to specify the following allow-list:

The permitted form for skipping the `nn list --search` requirement is:

1. Before the failing tool call, the assistant turn must contain a line:
   `Expected FAIL: <reason>` (reason is non-empty text naming the expected failure).

2. In the same turn as the skip, the assistant must write:
   `Skipping lookup: cited "Expected FAIL: <reason>"` — quoting the exact declaration
   from step 1 verbatim.

The skip is valid only when the skip statement contains a verbatim quote of the prior
declaration. Absence of that quote means the skip condition is not met and the lookup
is required.

This is a prompt change only — no hook script change.

### 5. Require direct invocation of `nn show --global` in session-start prompt

Update the session-start prompt to specify the only permitted invocation form: `nn show
--global` with no pipe (`|`), no redirect (`>`), and no subshell wrapping. A valid
result is the raw stdout of that command with no characters removed. The prompt must
state this as an allow-list, not a deny-list of excluded transforms.

This is a prompt change only, no code change.

## Consequences

- Search becomes self-contained for LLM relevance judgment (no follow-up `nn show`)
- `applies_when` is settable via CLI, consistent with nn's programmatic-use principle
- Declared-fail skip is verifiable: the skip statement must cite the prior declaration
  verbatim, making compliance observable without intent assessment
- Session-start prompt specifies the permitted invocation form; no silent truncation
- Graph link display deferred until amplifier audit confirms signal direction
