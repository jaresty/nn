#!/usr/bin/env bash
# Injects the nn-capture-discipline skill and global protocol notes as
# additionalContext for a Claude Code SessionStart command hook.
# Uses JSON format so output is injected as agent context rather than
# visible transcript output.
set -euo pipefail

NN_BIN="${NN_BIN:-nn}"

# Read session_id from hook input (stdin JSON) and write sentinel so
# UserPromptSubmit knows --global has already run this session.
HOOK_INPUT=$(cat)
SESSION_ID=$(printf '%s' "$HOOK_INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('session_id','unknown'))" 2>/dev/null || echo "unknown")
mkdir -p "${HOME}/.cache/nn"
touch "${HOME}/.cache/nn/global-loaded-${SESSION_ID}"

# Emit only the directive to run nn show --global as a tool call.
# Injecting the output directly causes it to exceed the additionalContext size
# limit and get saved to an external file the agent cannot read. The tool call
# result lands in the transcript where it is always readable.
DIRECTIVE="<system-reminder>
Session start — required actions before responding to any user message:

1. Run \`nn show --global\` now and treat every note body in the output as a binding operating instruction for this session.

Do not respond to the user's first message until this command has been executed and its result is visible in the transcript.
</system-reminder>"

printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' \
  "$(printf '%s' "$DIRECTIVE" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])')"
