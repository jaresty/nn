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

			// Apply explicit filters before shared relevance scoring.
			var candidates []*note.Note
			for _, n := range notes {
				if filterStatus != "" && string(n.Status) != filterStatus {
					continue
				}
				candidates = append(candidates, n)
			}

			scores := RankedByQuery(notes, candidates, query, state.notebookDir)
			filtered := make([]*note.Note, 0, len(candidates))
			for _, n := range candidates {
				if scores[n.ID] > 0 {
					filtered = append(filtered, n)
				}
			}

			// Default: sort by RRF score descending.
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
				return printSearchJSON(cmd, filtered, notes, query, scores, nil, nil, false)
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

// extractHeadingBreadcrumb returns the heading breadcrumb (e.g. "## Foo > ### Bar")
// for the position matchIdx in body, by walking backward to collect ancestor headings.
func extractHeadingBreadcrumb(body string, matchIdx int) string {
	lines := strings.Split(body[:matchIdx], "\n")
	// headings[level] = most recent heading text at that level (1=H1, 2=H2, 3=H3)
	headings := make(map[int]string)
	for _, line := range lines {
		for level := 1; level <= 6; level++ {
			prefix := strings.Repeat("#", level) + " "
			if strings.HasPrefix(line, prefix) {
				headings[level] = strings.TrimSpace(line)
				// clear all deeper levels when a shallower heading is found
				for deeper := level + 1; deeper <= 6; deeper++ {
					delete(headings, deeper)
				}
				break
			}
		}
	}
	if len(headings) == 0 {
		return ""
	}
	var parts []string
	for level := 1; level <= 6; level++ {
		if h, ok := headings[level]; ok {
			parts = append(parts, h)
		}
	}
	return strings.Join(parts, " > ")
}

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
		if crumb := extractHeadingBreadcrumb(body, idx); crumb != "" {
			snippet = crumb + " | " + snippet
		}
		return snippet
	}
	return ""
}
