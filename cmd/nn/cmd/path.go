package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

)

func newPathCmd(state *rootState) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "path <id-a> <id-b>",
		Short: "Find the shortest undirected path between two notes via their link graph",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			idA, idB := args[0], args[1]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("path: %w", err)
			}

			// Build title index and adjacency list (undirected).
			titles := make(map[string]string, len(notes))
			adj := make(map[string][]string, len(notes))
			allIDs := make(map[string]bool, len(notes))
			for _, n := range notes {
				titles[n.ID] = n.Title
				allIDs[n.ID] = true
				for _, lnk := range n.Links {
					adj[n.ID] = append(adj[n.ID], lnk.TargetID)
					adj[lnk.TargetID] = append(adj[lnk.TargetID], n.ID)
				}
			}

			if !allIDs[idA] {
				return fmt.Errorf("path: note %q not found", idA)
			}
			if !allIDs[idB] {
				return fmt.Errorf("path: note %q not found", idB)
			}
			if idA == idB {
				return printPath(cmd, jsonOut, []string{idA}, titles)
			}

			// BFS from idA to idB.
			prev := map[string]string{idA: ""}
			queue := []string{idA}
			found := false
			for len(queue) > 0 && !found {
				cur := queue[0]
				queue = queue[1:]
				for _, nb := range adj[cur] {
					if _, visited := prev[nb]; visited {
						continue
					}
					prev[nb] = cur
					if nb == idB {
						found = true
						break
					}
					queue = append(queue, nb)
				}
			}

			if !found {
				return fmt.Errorf("path: no path found between %q and %q", idA, idB)
			}

			// Reconstruct path from prev map.
			var path []string
			for cur := idB; cur != ""; cur = prev[cur] {
				path = append([]string{cur}, path...)
			}

			return printPath(cmd, jsonOut, path, titles)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func printPath(cmd *cobra.Command, jsonOut bool, path []string, titles map[string]string) error {
	w := outWriter(cmd)
	if jsonOut {
		type step struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		steps := make([]step, len(path))
		for i, id := range path {
			steps[i] = step{ID: id, Title: titles[id]}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(steps)
	}
	for i, id := range path {
		if i > 0 {
			fmt.Fprintf(w, "  →\n")
		}
		fmt.Fprintf(w, "%s  %s\n", id, titles[id])
	}
	return nil
}

