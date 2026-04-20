package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)


func newRandomCmd(state *rootState) *cobra.Command {
	var (
		filterTag    string
		filterType   string
		filterStatus string
		jsonOut      bool
		depth        int
	)

	cmd := &cobra.Command{
		Use:   "random",
		Short: "Return a random note, optionally filtered",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("random: %w", err)
			}

			var filtered []*note.Note
			for _, n := range notes {
				if filterTag != "" && !hasTag(n, filterTag) {
					continue
				}
				if filterType != "" && string(n.Type) != filterType {
					continue
				}
				if filterStatus != "" && string(n.Status) != filterStatus {
					continue
				}
				filtered = append(filtered, n)
			}

			if len(filtered) == 0 {
				return fmt.Errorf("random: no notes match filters")
			}

			n := filtered[rand.Intn(len(filtered))]
			w := outWriter(cmd)

			if depth > 0 {
				all, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("random: %w", err)
				}
				byID := make(map[string]*note.Note, len(all))
				for _, nn := range all {
					byID[nn.ID] = nn
				}
				entries := bfsDepth(n, byID, depth)
				if jsonOut {
					return printDepthJSON(w, entries)
				}
				return printDepthMarkdown(w, entries)
			}

			if jsonOut {
				tags := n.Tags
				if tags == nil {
					tags = []string{}
				}
				out := noteJSON{
					ID:     n.ID,
					Title:  n.Title,
					Type:   string(n.Type),
					Status: string(n.Status),
					Tags:   tags,
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			data, err := n.Marshal()
			if err != nil {
				return fmt.Errorf("random: marshal: %w", err)
			}
			fmt.Fprint(w, string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&filterTag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&filterType, "type", "", "Filter by type")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by status")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output note as JSON")
	cmd.Flags().IntVar(&depth, "depth", 0, "Traverse outgoing links to this depth and print all reachable notes")
	return cmd
}
