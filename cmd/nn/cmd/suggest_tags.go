package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newSuggestTagsCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var minNotes int

	cmd := &cobra.Command{
		Use:   "suggest-tags <id>",
		Short: "Suggest tags for a note based on BM25-similar notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("suggest-tags: %w", err)
			}

			var focal *note.Note
			for _, n := range notes {
				if n.ID == id {
					focal = n
					break
				}
			}
			if focal == nil {
				return fmt.Errorf("suggest-tags: note %q not found", id)
			}

			// Build set of tags already on the focal note.
			focalTags := make(map[string]bool)
			for _, t := range focal.Tags {
				focalTags[t] = true
			}

			// Rank all other notes by BM25 similarity.
			var others []*note.Note
			for _, n := range notes {
				if n.ID != focal.ID {
					others = append(others, n)
				}
			}
			query := focal.Title + " " + focal.Body
			scores := note.BM25Scores(others, query, nil)

			// Collect top similar notes (non-zero score).
			var similar []*note.Note
			for _, n := range others {
				if scores[n.ID] > 0 {
					similar = append(similar, n)
				}
			}
			sort.SliceStable(similar, func(i, j int) bool {
				return scores[similar[i].ID] > scores[similar[j].ID]
			})
			// Cap at top 10 similar notes.
			if len(similar) > 10 {
				similar = similar[:10]
			}

			// Aggregate tags from similar notes that focal lacks.
			type tagSuggestion struct {
				Tag       string   `json:"tag"`
				FromNotes []string `json:"from_notes"`
			}
			tagMap := map[string]*tagSuggestion{}
			for _, n := range similar {
				for _, t := range n.Tags {
					if focalTags[t] {
						continue
					}
					if _, ok := tagMap[t]; !ok {
						tagMap[t] = &tagSuggestion{Tag: t}
					}
					tagMap[t].FromNotes = append(tagMap[t].FromNotes, n.ID)
				}
			}

			// Filter to tags appearing in ≥ minNotes similar notes.
			var suggestions []*tagSuggestion
			for _, s := range tagMap {
				if len(s.FromNotes) >= minNotes {
					suggestions = append(suggestions, s)
				}
			}
			sort.Slice(suggestions, func(i, j int) bool {
				if len(suggestions[i].FromNotes) != len(suggestions[j].FromNotes) {
					return len(suggestions[i].FromNotes) > len(suggestions[j].FromNotes)
				}
				return suggestions[i].Tag < suggestions[j].Tag
			})

			w := outWriter(cmd)
			if jsonOut {
				if suggestions == nil {
					suggestions = []*tagSuggestion{}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(suggestions)
			}
			if len(suggestions) == 0 {
				fmt.Fprintln(w, "(no tag suggestions)")
				return nil
			}
			for _, s := range suggestions {
				fmt.Fprintf(w, "%-30s (from %d similar notes: %v)\n", s.Tag, len(s.FromNotes), s.FromNotes)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().IntVar(&minNotes, "min-notes", 2, "Minimum number of similar notes that must share a tag for it to be suggested")
	return cmd
}
