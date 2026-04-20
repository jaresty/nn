#!/usr/bin/env bash
# Emits global protocol notes as additionalContext for a Claude Code
# SessionStart command hook. Stdout is injected verbatim into the session.
set -euo pipefail

NN_BIN="${NN_BIN:-nn}"

# Emit the standing research instruction unconditionally.
cat <<'EOF'
## Standing instruction: research

Before doing research on any topic — looking something up, investigating a
codebase, answering a question that requires gathering information — first
check nn to see what already exists:

  nn list --search "<topic>" --json

After completing research, capture durable findings as notes:

  nn new --title "..." --type <observation|concept|model|...> --content "..." --no-edit

This applies in every session and every project, regardless of what is in
the notebook.

EOF

# Load global protocols (no outgoing governs links).
PROTOCOLS=$("$NN_BIN" list --global --json 2>/dev/null) || PROTOCOLS="[]"

COUNT=$(echo "$PROTOCOLS" | grep -c '"id"' || true)

if [ "$COUNT" -eq 0 ]; then
  echo "## Active protocols"
  echo "(none)"
  echo ""
  exit 0
fi

echo "## Active protocols (loaded at session start)"
echo ""

# Extract IDs and show each protocol note.
echo "$PROTOCOLS" | grep '"id"' | sed 's/.*"id":[[:space:]]*"\([^"]*\)".*/\1/' | while read -r id; do
  [ -z "$id" ] && continue

  # Extract title from the JSON list output.
  TITLE=$(echo "$PROTOCOLS" | awk -v id="$id" '
    /"id":/ && $0 ~ id { found=1 }
    found && /"title":/ {
      sub(/.*"title":[[:space:]]*"/, "")
      sub(/".*/, "")
      print
      exit
    }
  ')

  BODY=$("$NN_BIN" show "$id" 2>/dev/null | awk '
    /^---$/ { fm++; next }
    fm >= 2 { print }
  ' | sed '/^## Links/,$ d')

  echo "### ${TITLE}"
  echo "${BODY}"
  echo ""
  echo "---"
  echo ""
done
