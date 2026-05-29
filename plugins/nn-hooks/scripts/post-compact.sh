#!/usr/bin/env bash
# post-compact.sh — clears the nn show --global sentinel after compaction.
# On the next UserPromptSubmit, protocols-reminder.sh will inject the full
# re-run instruction so the agent reloads global protocols from the notebook.

HOOK_INPUT=$(cat)
SESSION_ID=$(printf '%s' "$HOOK_INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('session_id','unknown'))" 2>/dev/null || echo "unknown")
SENTINEL="${HOME}/.cache/nn/global-loaded-${SESSION_ID}"

rm -f "$SENTINEL"
