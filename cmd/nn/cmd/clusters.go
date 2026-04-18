package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newClustersCmd(state *rootState) *cobra.Command {
	var minSize int
	var singletons bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Detect topological clusters of notes using label propagation",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("clusters: %w", err)
			}

			// Build undirected adjacency list.
			adj := make(map[string][]string, len(notes))
			for _, n := range notes {
				for _, lnk := range n.Links {
					adj[n.ID] = append(adj[n.ID], lnk.TargetID)
					adj[lnk.TargetID] = append(adj[lnk.TargetID], n.ID)
				}
			}

			// Label propagation: each note starts with its own label.
			labels := make(map[string]string, len(notes))
			for _, n := range notes {
				labels[n.ID] = n.ID
			}

			// Iterate until stable (max 20 iterations).
			for iter := 0; iter < 20; iter++ {
				changed := false
				// Process in deterministic order (sorted IDs).
				ids := make([]string, 0, len(notes))
				for _, n := range notes {
					ids = append(ids, n.ID)
				}
				sort.Strings(ids)

				for _, id := range ids {
					neighbors := adj[id]
					if len(neighbors) == 0 {
						continue
					}
					// Count neighbor labels.
					freq := make(map[string]int)
					for _, nb := range neighbors {
						freq[labels[nb]]++
					}
					// Find most common label (ties: pick lexicographically smallest).
					bestLabel := labels[id]
					bestCount := 0
					for lbl, cnt := range freq {
						if cnt > bestCount || (cnt == bestCount && lbl < bestLabel) {
							bestLabel = lbl
							bestCount = cnt
						}
					}
					if bestLabel != labels[id] {
						labels[id] = bestLabel
						changed = true
					}
				}
				if !changed {
					break
				}
			}

			// Group notes by label.
			groups := make(map[string][]*note.Note)
			for _, n := range notes {
				lbl := labels[n.ID]
				groups[lbl] = append(groups[lbl], n)
			}

			// Build sorted cluster list.
			type cluster struct {
				notes []*note.Note
			}
			var clusters []cluster
			for _, members := range groups {
				size := len(members)
				effectiveMin := minSize
				if effectiveMin <= 0 {
					effectiveMin = 2
				}
				if !singletons && size < 2 {
					continue
				}
				if size < effectiveMin {
					continue
				}
				clusters = append(clusters, cluster{notes: members})
			}
			// Sort clusters by size descending, then by first member ID for stability.
			sort.Slice(clusters, func(i, j int) bool {
				if len(clusters[i].notes) != len(clusters[j].notes) {
					return len(clusters[i].notes) > len(clusters[j].notes)
				}
				return clusters[i].notes[0].ID < clusters[j].notes[0].ID
			})

			w := outWriter(cmd)
			if jsonOut {
				type noteEntry struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}
				type clusterEntry struct {
					Notes []noteEntry `json:"notes"`
				}
				out := make([]clusterEntry, len(clusters))
				for i, c := range clusters {
					entries := make([]noteEntry, len(c.notes))
					for j, n := range c.notes {
						entries[j] = noteEntry{ID: n.ID, Title: n.Title}
					}
					out[i] = clusterEntry{Notes: entries}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			for i, c := range clusters {
				entries := make([]string, len(c.notes))
				for j, n := range c.notes {
					entries[j] = n.ID + "  " + n.Title
				}
				fmt.Fprintf(w, "cluster %d (%d notes):\n", i+1, len(c.notes))
				for _, e := range entries {
					fmt.Fprintf(w, "  %s\n", e)
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&minSize, "min", 2, "Omit clusters smaller than N notes")
	cmd.Flags().BoolVar(&singletons, "singletons", false, "Include singleton clusters (notes with no links)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
