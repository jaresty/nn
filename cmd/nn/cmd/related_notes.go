package cmd

import (
	"fmt"
	"io"
)

const resolveInstructionText = "Resolve each unread related note — run `nn show <id>` to open, or write `skip-related: <id> [<id> ...] — <reason>` to dismiss. Notes marked [read] have already been loaded this session and do not require action."

// printResolveInstruction writes the resolve instruction only when hasUnread is true.
func printResolveInstruction(w io.Writer, hasUnread bool) {
	if hasUnread {
		fmt.Fprintln(w, resolveInstructionText)
	}
}
