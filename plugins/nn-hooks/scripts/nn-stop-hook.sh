#!/bin/sh
# nn-stop-hook.sh — throttled capture+debrief agent
# Runs at most once per 30 minutes to avoid per-turn latency.

LAST_RUN="$HOME/.claude/nn-stop-last-run"
INTERVAL=1800  # 30 minutes in seconds
NOW=$(date +%s)

if [ -f "$LAST_RUN" ]; then
  LAST=$(cat "$LAST_RUN")
  if [ $((NOW - LAST)) -lt $INTERVAL ]; then
    exit 0  # too soon — skip
  fi
fi

AGENT_PROMPT_FILE="$HOME/.local/share/nn/plugins/nn-hooks/agents/nn-stop-agent.md"
if [ ! -f "$AGENT_PROMPT_FILE" ]; then
  exit 0
fi

echo "$NOW" > "$LAST_RUN"

claude --print "$(cat "$AGENT_PROMPT_FILE")" \
  --allowedTools "Bash" \
  --output-format text \
  2>/dev/null || true
