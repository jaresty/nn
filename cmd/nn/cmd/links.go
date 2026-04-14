package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newLinksCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var filterType string

	cmd := &cobra.Command{
		Use:   "links <id>",
		Short: "Show outgoing links for a note with annotations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("links: %w", err)
			}

			// Build a title index.
			titles := make(map[string]string, len(notes))
			for _, n := range notes {
				titles[n.ID] = n.Title
			}

			// Find the requested note.
			var found bool
			w := outWriter(cmd)
			for _, n := range notes {
				if n.ID != id {
					continue
				}
				found = true

				if jsonOut {
					type linkEntry struct {
						ID         string `json:"id"`
						Title      string `json:"title"`
						Annotation string `json:"annotation"`
						Type       string `json:"type,omitempty"`
					}
					var entries []linkEntry
					for _, lnk := range n.Links {
						if filterType != "" && lnk.Type != filterType {
							continue
						}
						entries = append(entries, linkEntry{
							ID:         lnk.TargetID,
							Title:      titles[lnk.TargetID],
							Annotation: lnk.Annotation,
							Type:       lnk.Type,
						})
					}
					if entries == nil {
						entries = []linkEntry{}
					}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(entries)
				}

				for _, lnk := range n.Links {
					if filterType != "" && lnk.Type != filterType {
						continue
					}
					if lnk.Type != "" {
						fmt.Fprintf(w, "%s  %s\n  [%s] %s\n", lnk.TargetID, titles[lnk.TargetID], lnk.Type, lnk.Annotation)
					} else {
						fmt.Fprintf(w, "%s  %s\n  %s\n", lnk.TargetID, titles[lnk.TargetID], lnk.Annotation)
					}
				}
				break
			}
			if !found {
				return fmt.Errorf("links: note %q not found", id)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&filterType, "type", "", "Filter by link type (e.g. refines, contradicts)")
	return cmd
}
