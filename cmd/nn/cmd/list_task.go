package cmd

import (
	"fmt"
	"os"

	"github.com/jaresty/nn/internal/note"
)

// isUnblocked reports whether n has at least one requires link and all its
// requires targets are done (no unchecked checkboxes). Notes with no requires
// links are not unblocked — they are not task-blocking anything.
func isUnblocked(n *note.Note, byID map[string]*note.Note) bool {
	var requiresTargets []*note.Note
	for _, lnk := range n.Links {
		if lnk.Type != "requires" {
			continue
		}
		target, ok := byID[lnk.TargetID]
		if !ok {
			continue
		}
		requiresTargets = append(requiresTargets, target)
	}
	if len(requiresTargets) == 0 {
		return false
	}
	for _, target := range requiresTargets {
		if !note.IsDone(target.Body) {
			return false
		}
		if !note.HasCheckbox(target.Body) {
			fmt.Fprintf(os.Stderr, "warning: note %s (%q) has no checkboxes — it unblocks %s (%q) vacuously; add checkboxes to confirm intentional completion\n",
				target.ID, target.Title, n.ID, n.Title)
		}
	}
	return true
}
