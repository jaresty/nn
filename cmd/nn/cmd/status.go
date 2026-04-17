package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)


func newStatusCmd(state *rootState) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
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

			var drafts, broken, unknownTypes int
			var orphanList []*note.Note
			var brokenList []string

			for _, n := range notes {
				if !hasOutbound[n.ID] && !targetIDs[n.ID] {
					orphanList = append(orphanList, n)
				}
				if n.Status == note.StatusDraft {
					drafts++
				}
				for _, lnk := range n.Links {
					if !allIDs[lnk.TargetID] {
						broken++
						brokenList = append(brokenList, fmt.Sprintf("%s→%s", n.ID, lnk.TargetID))
					}
					if !note.IsKnownLinkType(lnk.Type) {
						unknownTypes++
					}
				}
			}

			w := outWriter(cmd)

			if jsonOut {
				type orphanEntry struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}
				type brokenEntry struct {
					From string `json:"from"`
					To   string `json:"to"`
				}
				orphans := make([]orphanEntry, len(orphanList))
				for i, o := range orphanList {
					orphans[i] = orphanEntry{ID: o.ID, Title: o.Title}
				}
				brokens := make([]brokenEntry, len(brokenList))
				for i, b := range brokenList {
					// brokenList entries are "fromID→toID"
					brokens[i] = brokenEntry{From: b}
				}
				out := struct {
					Total            int           `json:"total"`
					Orphans          []orphanEntry `json:"orphans"`
					Drafts           int           `json:"drafts"`
					BrokenLinks      []brokenEntry `json:"broken_links"`
					UnknownLinkTypes int           `json:"unknown_link_types"`
				}{
					Total:            len(notes),
					Orphans:          orphans,
					Drafts:           drafts,
					BrokenLinks:      brokens,
					UnknownLinkTypes: unknownTypes,
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			fmt.Fprintf(w, "total:   %d notes\n", len(notes))
			fmt.Fprintf(w, "orphans: %d\n", len(orphanList))
			for _, o := range orphanList {
				fmt.Fprintf(w, "  %s  %s\n", o.ID, o.Title)
			}
			fmt.Fprintf(w, "drafts:  %d\n", drafts)
			fmt.Fprintf(w, "broken links: %d\n", broken)
			for _, b := range brokenList {
				fmt.Fprintf(w, "  broken: %s\n", b)
			}
			fmt.Fprintf(w, "unknown link types: %d\n", unknownTypes)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
