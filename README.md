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
nn clusters --focus <id> --json # exact note's full-graph cluster (or null)
nn graph bridges --focus <id> --format json # exact note's bridge record (or null)
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
nn new --title "..." --type concept --content "..." --no-edit \
  --link-to <id> --link-type grounded-by --annotation "evidence basis"
nn link <from> <to> --type extends --annotation "builds on this"
nn link set-type <from> <to> --type supports  # migrate one legacy untyped link
nn list --json
```

Install guided LLM workflows:

```sh
nn install-skills              # Claude Code dispatch skill in ~/.claude/skills/
nn install-skills --for pi     # Pi dispatch skill + recursive skill sources
nn install --for pi            # Pi skills plus the nn global-context extension
```

`nn install --for pi` installs the Pi extension that loads `nn show --global` at session start and injects the global protocol context before each agent turn. Restart Pi or run `/reload` after installing.

Skills are served version-matched from the binary. Large skills expose lazy references without changing default output:

```sh
nn skills get nn-navigate                     # compact core only
nn skills get nn-navigate --list-references   # sorted names + applicability
nn skills get nn-navigate --reference movement
```

### Lossless graph body transport

Read topology first, then retrieve the same traversal set's bodies as bounded JSON pages:

```sh
nn graph show --focus <id> --depth 1 --direction both --zones --format json
nn graph bodies --focus <id> --depth 1 --direction both --page 1
nn graph bodies --focus <id> --depth 1 --direction both --page 2 --snapshot <sha256-from-page-1>
```

Each compact page is at most 48,000 bytes including JSON overhead. Page 1 returns `snapshot`, `pages`, `next_page`, and ordered body `segments`; pass that snapshot to every later page. Retrieve every page before reconstructing bodies or making body-derived claims. Concatenate a note's UTF-8 segments by their one-based `segment` ordinal for exact bytes; huge bodies span segments and empty bodies have one explicit empty segment. A notebook or traversal change rejects the old snapshot. Body records contain no frontmatter, links, or presentation metadata. `nn graph show --bodies` remains accepted for exact legacy output but is deprecated and unbounded.

### Graph Ask

```sh
nn ask --surface graph --focus <note-id> [--nodes <id,...>] --instructions "Review this neighborhood."
```

Graph Ask stays inside the bounded notebook neighborhood and offers three terminal buttons: `Send` (`handoff: null`), `Send to Canvas` (`handoff: canvas`), and `Send to Document` (`handoff: document`). Clicking one submits exactly once and closes the Graph Ask surface. The result uses exactly `groups`, `overall_comment`, and `handoff`; Canvas alone adds a non-stored canvas seed, while Document has no separate seed.

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
