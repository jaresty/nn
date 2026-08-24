package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newPathCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var links string

	cmd := &cobra.Command{
		Use:   "path <id-a> <id-b>",
		Short: "Find the shortest path between two notes via their link graph",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			idA, idB := args[0], args[1]
			typedMode := cmd.Flags().Changed("links")
			allowedTypes := make(map[string]bool)
			if typedMode {
				for _, raw := range strings.Split(links, ",") {
					linkType := strings.TrimSpace(raw)
					if linkType == "" {
						continue
					}
					if !note.IsKnownLinkType(linkType) {
						return fmt.Errorf("path: unknown link type %q (known: %s)", linkType, strings.Join(note.LinkTypeOrder, ", "))
					}
					allowedTypes[linkType] = true
				}
				if len(allowedTypes) == 0 {
					return fmt.Errorf("path: --links requires at least one link type")
				}
			}

			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("path: %w", err)
			}

			titles := make(map[string]string, len(notes))
			allIDs := make(map[string]bool, len(notes))
			legacyAdj := make(map[string][]string, len(notes))
			typedAdj := make(map[string][]typedTraversalEdge, len(notes))
			for _, n := range notes {
				titles[n.ID] = n.Title
				allIDs[n.ID] = true
				for _, lnk := range n.Links {
					if typedMode {
						if allowedTypes[lnk.Type] {
							typedAdj[n.ID] = append(typedAdj[n.ID], typedTraversalEdge{
								next: lnk.TargetID,
								edge: typedWitnessEdge{From: n.ID, To: lnk.TargetID, Type: lnk.Type, Annotation: lnk.Annotation},
							})
						}
						continue
					}
					legacyAdj[n.ID] = append(legacyAdj[n.ID], lnk.TargetID)
					legacyAdj[lnk.TargetID] = append(legacyAdj[lnk.TargetID], n.ID)
				}
			}

			if !allIDs[idA] {
				return fmt.Errorf("path: note %q not found", idA)
			}
			if !allIDs[idB] {
				return fmt.Errorf("path: note %q not found", idB)
			}
			if typedMode {
				witnesses := findShortestTypedWitnesses(idA, titles, typedAdj, 0).witnessesTo(idB)
				if len(witnesses) == 0 {
					return fmt.Errorf("path: no path found between %q and %q", idA, idB)
				}
				if jsonOut {
					return printTypedPath(cmd, witnesses)
				}
				return printTypedPathText(cmd, witnesses)
			}

			if idA == idB {
				return printPath(cmd, jsonOut, []string{idA}, titles)
			}
			path, found := findLegacyPath(idA, idB, legacyAdj)
			if !found {
				return fmt.Errorf("path: no path found between %q and %q", idA, idB)
			}
			return printPath(cmd, jsonOut, path, titles)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&links, "links", "", "Follow only these comma-separated link types in stored direction")
	return cmd
}

func findLegacyPath(idA, idB string, adj map[string][]string) ([]string, bool) {
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
		return nil, false
	}
	var path []string
	for cur := idB; cur != ""; cur = prev[cur] {
		path = append([]string{cur}, path...)
	}
	return path, true
}

func printTypedPath(cmd *cobra.Command, witnesses []typedWitness) error {
	out := struct {
		Witnesses []typedWitness `json:"witnesses"`
	}{Witnesses: witnesses}
	enc := json.NewEncoder(outWriter(cmd))
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printTypedPathText(cmd *cobra.Command, witnesses []typedWitness) error {
	w := outWriter(cmd)
	for witnessIndex, witness := range witnesses {
		if witnessIndex > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Witness %d:\n", witnessIndex+1)
		for nodeIndex, node := range witness.Nodes {
			fmt.Fprintf(w, "%s  %s\n", node.ID, node.Title)
			if nodeIndex >= len(witness.Edges) {
				continue
			}
			edge := witness.Edges[nodeIndex]
			if edge.Annotation == "" {
				fmt.Fprintf(w, "  → [%s]\n", edge.Type)
			} else {
				fmt.Fprintf(w, "  → [%s] %s\n", edge.Type, edge.Annotation)
			}
		}
	}
	return nil
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
