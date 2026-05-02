#!/usr/bin/env bash
# Re-injects global protocol notes as additionalContext after context compaction.
# Compaction discards the session-start context, so global protocols must be
# reloaded to remain authoritative via path (a) in the Protocols block rule.
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
  PROTOCOLS="## Active protocols (reloaded after compaction)\n\n${GLOBAL}"
fi

DIRECTIVE="<system-reminder>
Context was compacted — global protocols reloaded. Treat every note body below as a binding operating instruction for the remainder of this session.
</system-reminder>"

CONTEXT="${DIRECTIVE}\n\n${CAPTURE_PROTOCOL}\n\n${PROTOCOLS}"

printf '{"hookSpecificOutput":{"hookEventName":"PostCompact","additionalContext":"%s"}}\n' \
  "$(printf '%s' "$CONTEXT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])')"
