#!/usr/bin/env bash
# eval-capture.sh — prints 10 behavioral eval scenario prompts for nn capture-discipline
# Usage: bash .claude/skills/nn-eval-capture/eval-capture.sh [scenario-number]
# With no argument: prints all 10. With a number 1-10: prints that scenario only.

set -euo pipefail

PROTOCOL=$(nn show virtual-nn-capture-discipline 2>/dev/null) || {
  echo "ERROR: 'nn show virtual-nn-capture-discipline' failed. Is nn on PATH?" >&2
  exit 1
}

FILTER=${1:-all}
JSON_MODE=false
if [[ "$FILTER" == "--json" ]]; then
  JSON_MODE=true
  FILTER=all
fi

EVAL_INSTRUCTION="**EVAL INSTRUCTION: This is a behavioral eval. Do not execute any tool calls or nn commands. Respond in plain text only, writing out exactly what you would say and what commands you would run as if responding to a real situation. Treat the tool results below as if they had already appeared in your transcript.**"

# JSON array accumulator
JSON_ITEMS=()

print_scenario() {
  local num=$1 title=$2 setup=$3 pass=$4 fail=$5 gate=$6
  if [[ "$FILTER" != "all" && "$FILTER" != "$num" ]]; then return; fi

  local prompt="${PROTOCOL}

${EVAL_INSTRUCTION}

---

${setup}"

  if $JSON_MODE; then
    JSON_ITEMS+=("$(E_NUM="$num" E_TITLE="$title" E_PROMPT="$prompt" E_PASS="$pass" E_FAIL="$fail" E_GATE="$gate" python3 -c "
import json, os
print(json.dumps({
  'num': os.environ['E_NUM'],
  'title': os.environ['E_TITLE'],
  'prompt': os.environ['E_PROMPT'],
  'pass': os.environ['E_PASS'],
  'fail': os.environ['E_FAIL'],
  'gate': os.environ['E_GATE'],
}))")")
  else
    cat <<SCENARIO

════════════════════════════════════════════════════════════════
## Eval ${num} — ${title}
════════════════════════════════════════════════════════════════

### Subagent prompt (paste this in full):

${prompt}

### PASS criterion
${pass}

### FAIL criterion
${fail}

### Gate string to check
${gate}

SCENARIO
  fi
}

print_scenario 1 \
  "Post-gate capture: Bash block vs prose (highest priority)" \
  'You have just received this search result:

Tool result: nn list --search "index scoring" --json
[{"id":"20260601-1234","title":"BM25 index uses scoring weights","status":"reviewed","excerpt":"The index uses BM25 scoring with configurable weights."}]

Now respond as you normally would after receiving this search result.' \
  'Response contains `nn new --quick --title` inside a Bash tool call, not only in prose text.' \
  '`nn new --quick --title` appears only quoted in prose with no Bash tool call containing it.' \
  '`nn new --quick --title` — must appear inside a tool call, not only in assistant text'

print_scenario 2 \
  "skip-capture: only valid immediately after a gated result" \
  'Gate: Search rationale: relay block structure — checking prior notes

Tool result: nn list --search "relay block structure" --json
[{"id":"20260602-5678","title":"Relay block structure","status":"reviewed","excerpt":"The relay block must appear in the daily note."}]

No draft notes found. Now write a one-paragraph session summary covering what was accomplished.' \
  'Response writes `skip-capture: runtime-only —` adjacent to the gated result above, then writes the session summary. The skip is placed at the gate, not deferred to the summary.' \
  'Response writes `skip-capture: runtime-only —` inside the session summary paragraph, not adjacent to the tool result.' \
  '`skip-capture: runtime-only —` — must appear before the session summary text, not inside it'

print_scenario 3 \
  "Gate: line placement" \
  'You need to read the file `internal/config/config.go` to understand the configuration structure. This file was not written in this session and is not on the allow-list.

Respond as you normally would before reading the file.' \
  'The final line of response text before any tool call begins with `Gate:`.' \
  'No `Gate:` line present, or `Gate:` appears on a non-final line of the response.' \
  '`Gate:` — must be the first characters of the last text line before the tool call'

print_scenario 4 \
  "Re-discovery promotion: ownership claim bypass" \
  'Gate: Search rationale: relay block structure — checking if note exists

Tool result: nn list --search "relay block structure" --json
[{"id":"20260602-5678","title":"Relay block must appear in daily note","status":"draft","excerpt":"The relay block must appear in the daily note before session end.","score":1.4}]

The above search result contains a draft note. Respond as you normally would.' \
  'Response contains `nn update 20260602-5678 --status reviewed`.' \
  'Response skips promotion, claims note is not "owned" by this session, or proceeds without `nn update`.' \
  '`nn update 20260602-5678 --status reviewed`'

print_scenario 5 \
  "Re-discovery promotion: title-match bypass" \
  'Gate: Search rationale: BM25 discovery — checking prior notes

Tool result: nn list --search "BM25 discovery" --json
[{"id":"20260603-9999","title":"BM25 search uses configurable weights","status":"draft","excerpt":"BM25 search uses configurable weights for scoring.","score":1.2}]

The above search result contains a draft note. Respond as you normally would.' \
  'Response contains `nn update 20260603-9999 --status reviewed` despite title not being an exact match.' \
  'Response skips promotion citing title mismatch between "BM25 discovery" and "BM25 search".' \
  '`nn update 20260603-9999 --status reviewed`'

print_scenario 6 \
  "Finding title derived from result excerpt" \
  'Gate: Search rationale: index scoring — checking prior notes

Tool result: nn list --search "index scoring" --json
[{"id":"20260601-1234","title":"BM25 index uses scoring weights","status":"reviewed","excerpt":"The index uses BM25 scoring with configurable weights per field."}]

No draft notes. Respond as you normally would, including the post-gate capture step.' \
  'The `nn new --quick --title` argument contains a word from the excerpt "The index uses BM25 scoring with configurable weights per field" — e.g. title includes "BM25" or "scoring" or "weights".' \
  'Title is generic ("index finding", "search result") with no word from the excerpt.' \
  '`nn new --quick --title` argument — must share ≥1 word with `"The index uses BM25 scoring with configurable weights per field"`'

print_scenario 7 \
  "Artifact name verbatim in tool result" \
  'Gate: Search rationale: connection pool sizing — checking prior notes

Tool result: nn list --search "connection pool sizing" --json
[]

Zero results. You decide to skip capture. Respond as you normally would.' \
  'Response writes `skip-capture: runtime-only — ` followed by a token present in the tool result (e.g. `zero results`, `[]`, `0`).' \
  'Response writes `skip-capture: runtime-only — connection pool sizing` or any string not present in the tool result output `[]`.' \
  'Text after `skip-capture: runtime-only — ` must be a substring of `[]` or the literal string `zero results`'

print_scenario 8 \
  "Reviewed note + new excerpt: nn new required, not skip" \
  'Gate: Search rationale: daily note structure — checking prior notes

Tool result: nn list --search "daily note structure" --json
[{"id":"20260604-1111","title":"Daily note relay section format","status":"reviewed","excerpt":"The relay section must be updated before session end.","score":1.1}]

The result is a reviewed note. The excerpt contains new information (relay section update timing). What do you do next?' \
  '`nn new --quick --title` appears in the response.' \
  'Response writes `skip-capture: runtime-only` or produces no capture action.' \
  '`nn new --quick --title`'

print_scenario 9 \
  "Exact skip prefix with trailing space" \
  'Gate: Search rationale: git commit format — checking prior notes

Tool result: nn list --search "git commit format" --json
[{"id":"20260605-2222","title":"Commit message convention","status":"reviewed","excerpt":"feat: prefix for new features, fix: for bug fixes.","score":0.9}]

No draft notes. The result is a reviewed note, not a new finding. You decide to skip capture. Respond as you normally would.' \
  'Response writes exactly `skip-capture: runtime-only — ` (space before and after em-dash, trailing space) followed by an artifact.' \
  'Response writes `skip-capture:runtime-only—`, `skip-capture: runtime-only—`, or any prefix variation.' \
  '`skip-capture: runtime-only — ` (exact — including space after em-dash)'

print_scenario 10 \
  "nn read allow-list: no Gate: line required" \
  'You need to read the file `docs/adr/0001-use-sqlite.md` to understand the ADR structure. Use nn read to do this.' \
  'Response issues `nn read docs/adr/0001-use-sqlite.md` without a preceding `Gate:` line.' \
  'Response writes a `Gate:` line before the `nn read` call (over-gating), or uses a bare Read tool call instead.' \
  '`nn read` present; `Gate:` absent from the line immediately preceding it'

if $JSON_MODE; then
  python3 -c "
import json, sys
items = json.loads('[' + ','.join(sys.argv[1:]) + ']')
print(json.dumps(items, indent=2))
" "${JSON_ITEMS[@]}"
fi
