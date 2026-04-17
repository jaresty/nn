package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newBacklinksCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var filterType string

	cmd := &cobra.Command{
		Use:   "backlinks <id>",
		Short: "Show notes that link to a given note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("backlinks: %w", err)
			}

			// Build title index.
			titles := make(map[string]string, len(notes))
			for _, n := range notes {
				titles[n.ID] = n.Title
			}

			type backlinkEntry struct {
				ID         string `json:"id"`
				Title      string `json:"title"`
				Annotation string `json:"annotation"`
				Type       string `json:"type,omitempty"`
			}

			var results []backlinkEntry
			for _, n := range notes {
				for _, lnk := range n.Links {
					if lnk.TargetID != id {
						continue
					}
					if filterType != "" && lnk.Type != filterType {
						continue
					}
					results = append(results, backlinkEntry{
						ID:         n.ID,
						Title:      n.Title,
						Annotation: lnk.Annotation,
						Type:       lnk.Type,
					})
				}
			}

			w := outWriter(cmd)
			if jsonOut {
				if results == nil {
					results = []backlinkEntry{}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}
			for _, e := range results {
				if e.Type != "" {
					fmt.Fprintf(w, "%s  %s\n  [%s] %s\n", e.ID, e.Title, e.Type, e.Annotation)
				} else {
					fmt.Fprintf(w, "%s  %s\n  %s\n", e.ID, e.Title, e.Annotation)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&filterType, "type", "", "Filter by link type")
	return cmd
}
