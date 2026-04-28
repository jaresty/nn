#!/usr/bin/env bash
# Injects the nn-capture-discipline skill and global protocol notes as
# additionalContext for a Claude Code SessionStart command hook.
# Uses JSON format so output is injected as agent context rather than
# visible transcript output.
set -euo pipefail

NN_BIN="${NN_BIN:-nn}"

# Load the research/capture protocol from the installed nn-capture-discipline skill.
# Strips YAML frontmatter (everything between the first two --- lines).
SKILL_FILE="${HOME}/.claude/skills/nn-capture-discipline/SKILL.md"
if [ -f "$SKILL_FILE" ]; then
  CAPTURE_PROTOCOL=$(awk '
    BEGIN { in_front=0; past_front=0; count=0 }
    /^---$/ {
      count++
      if (count == 1) { in_front=1; next }
      if (count == 2) { in_front=0; past_front=1; next }
    }
    past_front { print }
  ' "$SKILL_FILE")
else
  CAPTURE_PROTOCOL="## Research protocol\n\nInstall the nn-capture-discipline skill (/nn-capture-discipline) for the full research and capture protocol."
fi

GLOBAL=$("$NN_BIN" show --global 2>/dev/null) || true

if [ -z "$GLOBAL" ]; then
  PROTOCOLS="## Active protocols\n(none)"
else
  PROTOCOLS="## Active protocols (loaded at session start)\n\n${GLOBAL}"
fi

CONTEXT="${CAPTURE_PROTOCOL}\n\n${PROTOCOLS}"

printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' \
  "$(printf '%s' "$CONTEXT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])')"
