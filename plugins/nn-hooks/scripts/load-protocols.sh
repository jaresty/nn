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

# Load and emit all global protocol notes (type:protocol, no governs links).
# nn show --global emits full note content + derivation instruction per protocol.
GLOBAL=$("$NN_BIN" show --global 2>/dev/null) || true

if [ -z "$GLOBAL" ]; then
  echo "## Active protocols"
  echo "(none)"
else
  echo "## Active protocols (loaded at session start)"
  echo ""
  echo "$GLOBAL"
fi
