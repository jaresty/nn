#!/usr/bin/env bash
# Emits a per-turn system-reminder instructing the agent to output a
# ## Protocols block before each response. Runs on UserPromptSubmit.
printf '<system-reminder>\nBefore responding, output a "## Protocols" block. For each active global protocol, write one sentence stating how it applies to this specific request — or "not applicable" if it does not. Place this block before your substantive response.\n</system-reminder>\n'
