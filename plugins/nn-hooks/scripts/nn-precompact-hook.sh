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

AGENT_PROMPT_FILE="$HOME/.local/share/nn/plugins/nn-hooks/agents/nn-session-debrief.md"
if [ ! -f "$AGENT_PROMPT_FILE" ]; then
  exit 0
fi

# Extract transcript_path from stdin JSON
HOOK_INPUT=$(cat)
TRANSCRIPT_PATH=$(echo "$HOOK_INPUT" | grep -o '"transcript_path":"[^"]*"' | sed 's/"transcript_path":"//;s/"//')
TRANSCRIPT=""
if [ -n "$TRANSCRIPT_PATH" ] && [ -f "$TRANSCRIPT_PATH" ]; then
  TRANSCRIPT=$(cat "$TRANSCRIPT_PATH")
fi

echo "$NOW" > "$LAST_RUN"

PROMPT_FILE=$(mktemp)
cat "$AGENT_PROMPT_FILE" > "$PROMPT_FILE"
if [ -n "$TRANSCRIPT_PATH" ] && [ -f "$TRANSCRIPT_PATH" ]; then
  printf '\n\n## Session transcript (last 50 lines)\n\n' >> "$PROMPT_FILE"
  tail -50 "$TRANSCRIPT_PATH" >> "$PROMPT_FILE"
fi

claude --print - \
  --allowedTools "Bash" \
  --output-format text \
  < "$PROMPT_FILE" \
  2>/dev/null || true

rm -f "$PROMPT_FILE"
