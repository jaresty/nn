package cmd

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// newSearchCmd searches notes using BM25 ranking.
func newSearchCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var sortBy string
	var limit int
	var filterStatus string

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
					if filterStatus != "" && string(n.Status) != filterStatus {
						continue
					}
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
				return printSearchJSON(cmd, filtered, query, scores)
			}
			w := outWriter(cmd)
			for _, n := range filtered {
				fmt.Fprintf(w, "%s  %s\n", n.ID, n.Title)
				if ex := extractExcerpt(n.Body, query); ex != "" {
					fmt.Fprintf(w, "  %s\n", ex)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by: title, modified, created")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by note status")
	return cmd
}

const excerptLen = 120

// extractExcerpt returns a ≤120-char snippet from body containing the first
// query term found, with leading "..." when the snippet is not from the start.
func extractExcerpt(body, query string) string {
	if body == "" || query == "" {
		return ""
	}
	lower := strings.ToLower(body)
	for _, term := range strings.Fields(strings.ToLower(query)) {
		idx := strings.Index(lower, term)
		if idx < 0 {
			continue
		}
		start := idx - 40
		if start < 0 {
			start = 0
		}
		end := start + excerptLen
		if end > len(body) {
			end = len(body)
		}
		// trim to valid UTF-8 rune boundaries
		for start > 0 && !utf8.RuneStart(body[start]) {
			start--
		}
		for end < len(body) && !utf8.RuneStart(body[end]) {
			end++
		}
		snippet := body[start:end]
		if start > 0 {
			snippet = "..." + snippet
		}
		return snippet
	}
	return ""
}
