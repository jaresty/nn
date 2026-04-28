package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// newSearchCmd searches notes using BM25 ranking.
func newSearchCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var sortBy string
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search notes by title and body with BM25 ranking",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("search: %w", err)
			}

			// BM25: include any note matching at least one query term.
			scores := note.BM25Scores(notes, query, nil)
			var filtered []*note.Note
			for _, n := range notes {
				if scores[n.ID] > 0 {
					filtered = append(filtered, n)
				}
			}

			// Default: sort by BM25 score descending.
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
