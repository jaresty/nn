package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// newSearchCmd is an alias for nn list --search <query>.
func newSearchCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var sortBy string
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search notes by title and body (alias for nn list --search)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("search: %w", err)
			}

			var filtered []*note.Note
			for _, n := range notes {
				if containsFold(n.Title, query) || containsFold(n.Body, query) {
					filtered = append(filtered, n)
				}
			}

			// Rank: title match scores higher than body-only match.
			scores := make(map[string]int, len(filtered))
			for _, n := range filtered {
				if containsFold(n.Title, query) {
					scores[n.ID] += 10
				}
				if containsFold(n.Body, query) {
					scores[n.ID] += 1
				}
			}
			sort.SliceStable(filtered, func(i, j int) bool {
				return scores[filtered[i].ID] > scores[filtered[j].ID]
			})

			switch sortBy {
			case "modified":
				sort.Slice(filtered, func(i, j int) bool {
					return filtered[i].Modified.After(filtered[j].Modified)
				})
			case "title":
				sort.Slice(filtered, func(i, j int) bool {
					return filtered[i].Title < filtered[j].Title
				})
			case "created":
				sort.Slice(filtered, func(i, j int) bool {
					return filtered[i].Created.After(filtered[j].Created)
				})
			}

			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			if jsonOut {
				return printNotesJSON(cmd, filtered)
			}
			for _, n := range filtered {
				fmt.Fprintf(outWriter(cmd), "%s  %s\n", n.ID, n.Title)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by: title, modified, created")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	return cmd
}
