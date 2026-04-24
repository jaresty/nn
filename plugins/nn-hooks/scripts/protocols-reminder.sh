#!/usr/bin/env bash
# Injects a per-turn system-reminder instructing the agent to output a
# ## Protocols block before each response. Runs on UserPromptSubmit.
# Uses JSON additionalContext format so the injection is discrete (not
# shown as visible hook output in the transcript).
printf '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"<system-reminder>\\nBefore responding, output a \\"## Protocols\\" block. For each active global protocol, write one sentence stating how it applies to this specific request — or \\"not applicable\\" if it does not. Place this block before your substantive response.\\n</system-reminder>"}}\n'
