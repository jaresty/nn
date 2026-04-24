#!/usr/bin/env bash
# Emits global protocol notes as additionalContext for a Claude Code
# SessionStart command hook. Stdout is injected verbatim into the session.
set -euo pipefail

NN_BIN="${NN_BIN:-nn}"

# Emit the standing research instruction unconditionally.
cat <<'EOF'
## Research protocol

When you are about to research any topic — web search, reading docs, exploring
a codebase, investigating an API — check nn first before going elsewhere:

  nn list --search "<topic>" --json

nn is your personal knowledge base. Checking it first avoids re-discovering
what you already know.

When you finish and have durable findings, capture them:

  nn new --title "..." --type <observation|concept|model|...> --content "..." --no-edit

Triggers: any research task, regardless of topic or project.
Does not trigger: reading files already in context, running tests, writing code.

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
