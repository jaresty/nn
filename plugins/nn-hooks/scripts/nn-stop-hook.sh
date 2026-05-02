#!/bin/sh
# nn-stop-hook.sh — throttled capture+debrief agent
# Runs at most once per 30 minutes to avoid per-turn latency.

LAST_RUN="$HOME/.claude/nn-stop-last-run"
INTERVAL=1800  # 30 minutes in seconds
NOW=$(date +%s)

# Log which global protocols are active this session — runs unconditionally,
# even when the capture agent is throttled. Used to track protocol utility over time.
LOG_DIR="${NN_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/nn}"
mkdir -p "$LOG_DIR"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
PROTOCOL_IDS=$(nn show --global 2>/dev/null | grep '^id:' | awk '{print $2}' | tr '\n' ' ' | sed 's/ $//')
echo "$TIMESTAMP $PROTOCOL_IDS" >> "$LOG_DIR/protocol-presence.log"

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
