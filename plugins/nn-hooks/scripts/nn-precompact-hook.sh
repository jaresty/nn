#!/bin/sh
# nn-precompact-hook.sh — capture+debrief agent before context compaction
# Runs at most once per 10 minutes to avoid redundant runs in heavy sessions.
# Fires before compaction so the agent sees the full raw transcript.

LAST_RUN="$HOME/.claude/nn-precompact-last-run"
INTERVAL=600  # 10 minutes in seconds
NOW=$(date +%s)

if [ -f "$LAST_RUN" ]; then
  LAST=$(cat "$LAST_RUN")
  if [ $((NOW - LAST)) -lt $INTERVAL ]; then
    exit 0  # too soon — skip agent
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
