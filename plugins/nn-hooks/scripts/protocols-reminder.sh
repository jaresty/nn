#!/usr/bin/env bash
# Injects a per-turn system-reminder instructing the agent to derive a
# ## Protocols block before each response. Runs on UserPromptSubmit.
# Uses JSON additionalContext format so the injection is discrete (not
# shown as visible hook output in the transcript).
printf '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"<system-reminder>\\nBefore responding, derive a \\"## Protocols\\" block as follows:\\nFor each active global protocol, name the specific condition that would make it binding on this turn, and state whether that condition holds.\\n- If binding: write one sentence naming the specific constraint it imposes on this turn.\\n- If not binding: write \\"not applicable — [condition] is absent because [reason].\\\"\\nThen state: \\"Active constraints: [list or none].\\\"\\nYour substantive response must stay within those active constraints.\\n</system-reminder>"}}\n'
