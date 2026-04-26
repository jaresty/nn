#!/usr/bin/env bash
# Injects global protocol notes as additionalContext for a Claude Code
# SessionStart command hook. Uses JSON format so output is injected as
# agent context rather than visible transcript output.
set -euo pipefail

NN_BIN="${NN_BIN:-nn}"

GLOBAL=$("$NN_BIN" show --global 2>/dev/null) || true

if [ -z "$GLOBAL" ]; then
  PROTOCOLS="## Active protocols\n(none)"
else
  PROTOCOLS="## Active protocols (loaded at session start)\n\n${GLOBAL}"
fi

CONTEXT="## Research protocol\n\nWhen you are about to research any topic — web search, reading docs, exploring\na codebase, investigating an API — check nn first before going elsewhere:\n\n  nn list --search \"<topic>\" --json\n\nnn is your personal knowledge base. Checking it first avoids re-discovering\nwhat you already know.\n\nWhen you finish and have durable findings, capture them:\n\n  nn new --title \"...\" --type <observation|concept|model|...> --content \"...\" --no-edit\n\nBefore deciding whether to check nn: ask — \"Is there a topic here where I might\nhave prior captured knowledge that would change what I do?\" If yes, check nn first.\nReason from the specific request — do not match it against a category label.\n\n${PROTOCOLS}"

printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' \
  "$(printf '%s' "$CONTEXT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])')"
