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

CONTEXT="## Research protocol\n\nBefore pulling in any knowledge that did not exist in this conversation's\ncontext at the start of the turn, you must produce a visible nn search\nartifact in the transcript:\n\n  nn list --search \"<topic>\" --json\n\nThis covers any tool or action that introduces new information: spawning an\nagent, fetching a URL, searching the web, reading unfamiliar files, or any\nother means. The artifact must appear BEFORE that action. Skipping it is\ndetectable: an information-gathering action with no preceding nn result in\nthe transcript is a protocol violation, regardless of the reason.\n\nWhat counts as the artifact: a tool result block from nn list --search (even\nan empty result). A statement that you checked nn does not count.\n\nAfter the search: if results are present, read the relevant ones before\ndeciding whether to go external. If results are absent or insufficient,\nproceed to the information-gathering action.\n\nAfter any information-gathering action returns, you may not take any further\naction (including responding to the user) until one of the following tool\nresults appears in the transcript:\n\n  - A capture: nn new, nn update, nn link, or any other nn write operation\n  - An explicit skip: Bash(echo \"nn-capture-skip: <reason>\")\n\nBoth are falsifiable by transcript inspection. \"I decided nothing was worth\ncapturing\" is not a valid skip — it must be a tool result. The skip exists\nso that genuine no-finding cases are distinguishable from silent omissions.\n\n${PROTOCOLS}"

printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' \
  "$(printf '%s' "$CONTEXT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])')"
