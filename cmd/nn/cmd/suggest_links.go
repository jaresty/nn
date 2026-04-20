package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// newSuggestLinksCmd returns the suggest-links command.
func newSuggestLinksCmd(state *rootState) *cobra.Command {
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "suggest-links <id>",
		Short: "Format context for LLM-driven link suggestion",
		Long: `Loads the focal note and ranked BM25 candidate notes, then emits a
structured context block for an LLM to reason over and suggest links.

The LLM receiving this output is expected to call 'nn link' or 'nn bulk-link'
to create accepted suggestions.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("suggest-links: %w", err)
			}

			// Find focal note.
			var focal *note.Note
			for _, n := range notes {
				if n.ID == id {
					focal = n
					break
				}
			}
			if focal == nil {
				return fmt.Errorf("suggest-links: note %q not found", id)
			}

			// Build existing link set for focal note (both directions).
			linkedIDs := make(map[string]string) // targetID -> link type
			for _, lnk := range focal.Links {
				linkedIDs[lnk.TargetID] = lnk.Type
			}
			// Also check backlinks: notes that link TO focal.
			for _, n := range notes {
				for _, lnk := range n.Links {
					if lnk.TargetID == focal.ID {
						linkedIDs[n.ID] = lnk.Type
					}
				}
			}

			// Build candidate list: all notes except focal, ranked by BM25.
			var others []*note.Note
			for _, n := range notes {
				if n.ID != focal.ID {
					others = append(others, n)
				}
			}

			// BM25 score candidates against focal note's content.
			query := focal.Title + " " + focal.Body
			scores := note.BM25Scores(others, query)

			// Variant F: exclude zero-score notes, report excluded count.
			var scored []*note.Note
			var excludedCount int
			for _, n := range others {
				if scores[n.ID] > 0 {
					scored = append(scored, n)
				} else {
					excludedCount++
				}
			}
			others = scored

			sort.SliceStable(others, func(i, j int) bool {
				return scores[others[i].ID] > scores[others[j].ID]
			})

			if limit > 0 && len(others) > limit {
				others = others[:limit]
			}

			w := outWriter(cmd)

			if format == "json" {
				type candidateJSON struct {
					ID          string `json:"id"`
					Title       string `json:"title"`
					Type        string `json:"type"`
					Tags        []string `json:"tags"`
					Summary     string `json:"summary"`
					AlreadyLinked bool   `json:"already_linked,omitempty"`
					LinkType    string `json:"link_type,omitempty"`
				}
				type focalJSON struct {
					ID    string   `json:"id"`
					Title string   `json:"title"`
					Type  string   `json:"type"`
					Tags  []string `json:"tags"`
					Body  string   `json:"body"`
				}
				type outputJSON struct {
					FocalNote     focalJSON      `json:"focal_note"`
					Candidates    []candidateJSON `json:"candidates"`
					ExcludedCount int            `json:"excluded_count,omitempty"`
				}

				out := outputJSON{
					ExcludedCount: excludedCount,
					FocalNote: focalJSON{
						ID:    focal.ID,
						Title: focal.Title,
						Type:  string(focal.Type),
						Tags:  tagsOrEmpty(focal.Tags),
						Body:  focal.Body,
					},
				}
				for _, n := range others {
					c := candidateJSON{
						ID:      n.ID,
						Title:   n.Title,
						Type:    string(n.Type),
						Tags:    tagsOrEmpty(n.Tags),
						Summary: summarize(n.Body, 200),
					}
					if lt, ok := linkedIDs[n.ID]; ok {
						c.AlreadyLinked = true
						c.LinkType = lt
					}
					out.Candidates = append(out.Candidates, c)
				}
				if out.Candidates == nil {
					out.Candidates = []candidateJSON{}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			// Plain text format (default).
			tags := strings.Join(focal.Tags, ", ")
			fmt.Fprintf(w, "## Focal note\n")
			fmt.Fprintf(w, "id: %s\n", focal.ID)
			fmt.Fprintf(w, "title: %s\n", focal.Title)
			fmt.Fprintf(w, "type: %s\n", focal.Type)
			fmt.Fprintf(w, "tags: %s\n", tags)
			fmt.Fprintf(w, "body:\n%s\n", focal.Body)
			excludedSuffix := ""
			if excludedCount > 0 {
				excludedSuffix = fmt.Sprintf(", %d excluded — no term overlap", excludedCount)
			}
			fmt.Fprintf(w, "\n## Candidate notes (%d total%s)\n", len(others), excludedSuffix)
			for _, n := range others {
				linkedMarker := ""
				if lt, ok := linkedIDs[n.ID]; ok {
					if lt != "" {
						linkedMarker = fmt.Sprintf(" (already linked: %s)", lt)
					} else {
						linkedMarker = " (already linked)"
					}
				}
				fmt.Fprintf(w, "### %s — %s [%s]%s\n", n.ID, n.Title, n.Type, linkedMarker)
				fmt.Fprintf(w, "tags: %s\n", strings.Join(n.Tags, ", "))
				fmt.Fprintf(w, "summary: %s\n\n", summarize(n.Body, 200))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of candidate notes to include")
	cmd.Flags().StringVar(&format, "format", "", "Output format: json")
	return cmd
}

// summarize returns the first n characters of s, trimmed cleanly.
func summarize(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// tagsOrEmpty returns the tags slice or an empty slice if nil.
func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}
