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

CONTEXT="## Research protocol\n\nBefore calling any external knowledge tool — Task[Explore], WebFetch, WebSearch,\nor Grep/Glob over unfamiliar territory — you must produce a visible nn search\nartifact in the transcript:\n\n  nn list --search \"<topic>\" --json\n\nThis artifact must appear BEFORE the external tool call. Skipping it is\ndetectable: an external tool call with no preceding nn result in the transcript\nis a protocol violation, regardless of the reason.\n\nWhat counts as the artifact: a tool result block from nn list --search (even\nan empty result). A statement that you checked nn does not count.\n\nAfter the search: if results are present, read the relevant ones before\ndeciding whether to go external. If results are absent or insufficient,\nproceed to the external tool.\n\nWhen you finish and have durable findings, capture them:\n\n  nn new --title \"...\" --type <observation|concept|model|...> --content \"...\" --no-edit\n\n${PROTOCOLS}"

printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' \
  "$(printf '%s' "$CONTEXT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])')"
