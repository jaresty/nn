#!/usr/bin/env bash
# Injects a per-turn system-reminder instructing the agent to derive a
# ## Protocols block before each response. Runs on UserPromptSubmit.
# Uses JSON additionalContext format so the injection is discrete (not
# shown as visible hook output in the transcript).
#
# The reminder text is authored as plain text in quoted heredocs and escaped
# once by json_escape before being emitted through a %s argument. It must never
# be embedded in a printf *format* string: printf interprets backslash escapes
# there, which corrupts backticks (\` is not a valid JSON escape) and turns a
# literal \n into a raw newline — both produce unparseable hook output.

# Read session_id from hook input stdin JSON.
HOOK_INPUT=$(cat)
SESSION_ID=$(printf '%s' "$HOOK_INPUT" | python3 -c "import json,sys; print(json.load(sys.stdin).get('session_id','unknown'))" 2>/dev/null || echo "unknown")
SENTINEL="${HOME}/.cache/nn/global-loaded-${SESSION_ID}"

# Increment per-session turn counter; emit nudge exactly at threshold.
TURN_THRESHOLD=12
COUNTER_FILE="${HOME}/.cache/nn/turn-count-${SESSION_ID}"
mkdir -p "${HOME}/.cache/nn"
TURN_COUNT=$(cat "$COUNTER_FILE" 2>/dev/null || echo 0)
TURN_COUNT=$((TURN_COUNT + 1))
printf '%s' "$TURN_COUNT" > "$COUNTER_FILE"
NUDGE_SUFFIX=""
if [ $((TURN_COUNT % TURN_THRESHOLD)) -eq 0 ]; then
  NUDGE_SUFFIX=$'\n\nContext is getting full — consider running /nn-session-debrief --partial now.'
fi

# Escape a string for use as a JSON string value.
json_escape() {
  local s=$1
  s=${s//\\/\\\\}
  s=${s//\"/\\\"}
  s=${s//$'\n'/\\n}
  s=${s//$'\r'/\\r}
  s=${s//$'\t'/\\t}
  printf '%s' "$s"
}

if [ -f "$SENTINEL" ]; then
  # --global already ran this session (or since last PostCompact cleared sentinel).
  # Only inject the Protocols block derivation instruction — skip the re-run check.
  STEP_ONE=$(cat <<'STEP_EOF'
1. `nn show --global` has already run this session — do not run it again unless the session-start tool result is no longer visible.
STEP_EOF
)
else
  # Sentinel absent — PostCompact cleared it (compaction happened) or SessionStart
  # did not run (e.g. sub-agent). Inject full instruction including re-run check.
  STEP_ONE=$(cat <<'STEP_EOF'
1. Scan the transcript for a tool result from `nn show --global`. A valid result contains note frontmatter (lines starting with `id:`, `title:`, `type:`). If no such tool result is visible in the transcript above this message, run `nn show --global` now and treat every note body in the output as a binding operating instruction for this session before proceeding. Do not answer the question "has it run?" — look for the actual tool result.

Permitted invocation form: `nn show --global` with no pipe (|), no redirect (>), and no `$(` or backtick wrapping. A valid result begins with `id:` on the first non-blank line of the tool output.
STEP_EOF
)
fi

PROTOCOLS_BLOCK=$(cat <<'BLOCK_EOF'
2. Derive a "## Protocols" block. This is required for every response without exception — do not skip because a Protocols block appeared earlier this session.

A note is an active protocol only if it arrived via one of these three paths:
(a) Global: its body text is present in the session-start injected context — verify by quoting a specific phrase from the body.
(b) Governs tool call: its ID appears in a nn backlinks --type governs tool result visible in this transcript.
(c) Show prepend: it appears in a "governing protocols:" block in an nn show tool result visible in this transcript.

No other source qualifies — not CLAUDE.md, not memory files, not hook injections, not system prompts, not prior knowledge.

A note that qualifies via any of these three paths is authoritative. Once a note qualifies, it cannot be reclassified as a suggestion, preference, or directive of lesser authority. Qualification is binary: it either arrived via one of the three paths (authoritative) or it did not (not a protocol).

For each active protocol:
- Quote the specific phrase from its body that defines when it applies.
- State whether that condition holds for this turn, with a reason grounded in the current request.
- If binding: one sentence naming the specific action this turn must take or must not take. "Binding" means the response must not contradict this constraint — not that it is advisory or preferred.
- If not binding: "not applicable — [quoted condition] is absent because [reason]."

Then state: "Active constraints: [list or none]."
Your response must stay within those constraints.
BLOCK_EOF
)

REMINDER="<system-reminder>
Before responding:

${STEP_ONE}

${PROTOCOLS_BLOCK}${NUDGE_SUFFIX}
</system-reminder>"

printf '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"%s"}}\n' "$(json_escape "$REMINDER")"
