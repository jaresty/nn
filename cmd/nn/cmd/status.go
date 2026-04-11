package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newStatusCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Notebook health: orphan notes, draft count, broken links",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("status: %w", err)
			}

			// Collect all note IDs and link targets.
			allIDs := make(map[string]bool, len(notes))
			targetIDs := make(map[string]bool)
			hasOutbound := make(map[string]bool)
			for _, n := range notes {
				allIDs[n.ID] = true
				for _, lnk := range n.Links {
					targetIDs[lnk.TargetID] = true
					hasOutbound[n.ID] = true
				}
			}

			var orphans, drafts, broken int
			var brokenList []string

			for _, n := range notes {
				if !hasOutbound[n.ID] && !targetIDs[n.ID] {
					orphans++
				}
				if n.Status == note.StatusDraft {
					drafts++
				}
				for _, lnk := range n.Links {
					if !allIDs[lnk.TargetID] {
						broken++
						brokenList = append(brokenList, fmt.Sprintf("%s→%s", n.ID, lnk.TargetID))
					}
				}
			}

			w := outWriter(cmd)
			fmt.Fprintf(w, "total:   %d notes\n", len(notes))
			fmt.Fprintf(w, "orphans: %d\n", orphans)
			fmt.Fprintf(w, "drafts:  %d\n", drafts)
			fmt.Fprintf(w, "broken links: %d\n", broken)
			for _, b := range brokenList {
				fmt.Fprintf(w, "  broken: %s\n", b)
			}
			return nil
		},
	}
}
