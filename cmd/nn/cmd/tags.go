package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func newTagsCmd(state *rootState) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List all tags in the notebook with note counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("tags: %w", err)
			}

			type tagEntry struct {
				Tag   string   `json:"tag"`
				Count int      `json:"count"`
				Notes []string `json:"notes"`
			}

			tagMap := map[string]*tagEntry{}
			for _, n := range notes {
				for _, t := range n.Tags {
					if _, ok := tagMap[t]; !ok {
						tagMap[t] = &tagEntry{Tag: t}
					}
					tagMap[t].Count++
					tagMap[t].Notes = append(tagMap[t].Notes, n.ID)
				}
			}

			entries := make([]*tagEntry, 0, len(tagMap))
			for _, e := range tagMap {
				entries = append(entries, e)
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].Count != entries[j].Count {
					return entries[i].Count > entries[j].Count
				}
				return entries[i].Tag < entries[j].Tag
			})

			w := outWriter(cmd)
			if jsonOut {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}
			for _, e := range entries {
				fmt.Fprintf(w, "%-30s %d\n", e.Tag, e.Count)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	return cmd
}
