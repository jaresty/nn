# nn

An LLM-driven Zettelkasten CLI. Notes are plain Markdown files in a Git-backed directory. Every operation is designed for programmatic use by an LLM as well as interactive human use.

## Install

### Homebrew (macOS / Linux)

```sh
brew tap jaresty/nn
brew install nn
```

> **macOS Gatekeeper:** On first install, macOS may block the binary as unverified. Run:
> ```sh
> xattr -d com.apple.quarantine $(which nn)
> ```

### Go

```sh
go install github.com/jaresty/nn/cmd/nn@latest
```

### Download

Download a pre-built binary from the [releases page](https://github.com/jaresty/nn/releases).

## Setup

Create `~/.config/nn/config.toml`:

```toml
[notebooks]
default = "personal"

[notebooks.personal]
path = "~/notes"
backend = "gitlocal"
```

Initialise the notebook directory as a git repo:

```sh
mkdir ~/notes && git -C ~/notes init
```

## Usage

```sh
nn new --title "The Atomicity Principle" --type concept --no-edit
nn show <id>
nn show <id> --depth 2          # note + all notes reachable within 2 hops
nn list
nn list --type concept --status draft --json
nn list --similar <id>          # notes ranked by similarity to a given note
nn random --status permanent    # serendipitous re-encounter
nn link <from-id> <to-id> --annotation "builds on this" --type extends
nn unlink <from-id> <to-id>
nn graph --json
nn status
nn path <id-a> <id-b>           # shortest link path between two notes
nn clusters                     # topological clusters via label propagation
nn promote <id> --to reviewed
nn delete <id> --confirm
nn install-skills
```

### Note types

`concept` · `argument` · `model` · `hypothesis` · `observation`

### Note statuses

`draft` → `reviewed` → `permanent`

## LLM use

Every command accepts named flags — no prompts, no editor, no TTY required:

```sh
nn new --title "..." --type concept --content "..." --no-edit
nn link <from> <to> --annotation "..."
nn list --json
```

Install the Claude Code skills for guided LLM workflows:

```sh
nn install-skills
```

This copies `nn-workflow` and `nn-guide` into `~/.claude/skills/`.

## Note format

```markdown
---
id: 20260411120045-3821
title: "The Atomicity Principle"
type: concept
status: draft
tags: [zettelkasten, methodology]
created: 2026-04-11T12:00:45Z
modified: 2026-04-11T12:05:00Z
---

Body text.

## Links

- [[20260411090000-1234]] — provides the foundational philosophy this principle implements
```

## Multiple notebooks

```toml
[notebooks]
default = "personal"

[notebooks.personal]
path = "~/notes"
backend = "gitlocal"

[notebooks.work]
path = "~/work/notes"
backend = "gitlocal"
```

Select with `--notebook work` or `NN_NOTEBOOK=work nn list`.
