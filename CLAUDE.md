# Claude Code Reminders — nn

## Project Overview

`nn` is an LLM-driven Zettelkasten CLI. Notes are plain Markdown files in a Git-backed
directory. The CLI is designed for programmatic use by an LLM as well as interactive human use.

## Key Design Principles

- **Non-interactive by default when stdout is not a TTY** — no prompts, no pickers, no editor
- **Every interactive input has a named flag equivalent** — `--content`, `--annotation`, `--confirm`, `--to`
- **Files are truth, index is cache** — SQLite index in `~/.config/nn/index.db` is always rebuildable
- **One operation = one Git commit** — each CLI operation produces a single semantic commit
- **Global config** — notebook location is in `~/.config/nn/config.toml` or `$NN_NOTEBOOK` env var

## Architecture

```
cmd/nn/          CLI entry point (cobra)
internal/
  note/          Note struct, ID generation, frontmatter parsing
  graph/         Link graph, backlink queries
  index/         SQLite index management
  backend/       Backend interface + implementations
    gitlocal/    Git-backed local filesystem
  config/        Config loading (flag > env > project > user)
templates/       Note creation templates (Go template syntax)
docs/adr/        Architecture Decision Records
```

## Available Tools

### GrepAI
- **Tool**: `mcp__grepai__grepai_search` and related grepai MCP tools
- **Purpose**: Semantic code search
- **When to use**: Understanding code by intent, finding implementations

Use grepai as primary tool for code exploration; fall back to Grep/Glob for exact text matching.

## Bar Skills

Bar skills are in `.claude/skills/`. Use `/bar-workflow` for multi-step planning tasks and
`/bar-autopilot` for single-step structured responses.

## Commit Message Convention

```
note: create <id> — <title>
note: link <from-id> → <to-id>
note: promote <id> to <status>
feat: <what changed>
fix: <what was broken>
docs(adr): <adr title>
```

## ADR Convention

ADRs live in `docs/adr/`. Numbered sequentially: `0001-name.md`, `0002-name.md`.
Record every significant architectural decision before implementing it.
