package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

type graphImpactAdjacencyEdge struct {
	next    string
	witness pathWitnessEdge
}

type graphImpactNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type graphImpactEntry struct {
	Node  graphImpactNode   `json:"node"`
	Depth int               `json:"depth"`
	Nodes []graphImpactNode `json:"nodes"`
	Edges []pathWitnessEdge `json:"edges"`
}

type graphImpactOutput struct {
	Focus     graphImpactNode    `json:"focus"`
	Direction string             `json:"direction"`
	Links     []string           `json:"links"`
	Depth     int                `json:"depth"`
	Impacts   []graphImpactEntry `json:"impacts"`
}

func newGraphImpactCmd(state *rootState) *cobra.Command {
	var focus string
	var links string
	var direction string
	var depth int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "impact",
		Short: "Traverse typed incoming or outgoing impact from a focus note",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("focus") {
				return fmt.Errorf("graph impact: --focus is required")
			}
			focus = strings.TrimSpace(focus)
			if focus == "" {
				return fmt.Errorf("graph impact: --focus requires a non-blank ID")
			}
			if !cmd.Flags().Changed("links") {
				return fmt.Errorf("graph impact: --links is required")
			}
			if strings.TrimSpace(links) == "" {
				return fmt.Errorf("graph impact: --links requires at least one link type")
			}
			allowedTypes := make(map[string]bool)
			for _, raw := range strings.Split(links, ",") {
				linkType := strings.TrimSpace(raw)
				if linkType == "" {
					return fmt.Errorf("graph impact: --links contains an empty value")
				}
				if !note.IsKnownLinkType(linkType) {
					return fmt.Errorf("graph impact: unknown link type %q (known: %s)", linkType, strings.Join(note.LinkTypeOrder, ", "))
				}
				allowedTypes[linkType] = true
			}
			normalizedLinks := make([]string, 0, len(allowedTypes))
			for linkType := range allowedTypes {
				normalizedLinks = append(normalizedLinks, linkType)
			}
			sort.Strings(normalizedLinks)

			if !cmd.Flags().Changed("direction") {
				return fmt.Errorf("graph impact: --direction is required")
			}
			if direction != "incoming" && direction != "outgoing" {
				return fmt.Errorf("graph impact: --direction must be exactly incoming or outgoing")
			}
			if !cmd.Flags().Changed("depth") {
				return fmt.Errorf("graph impact: --depth is required")
			}
			if depth <= 0 {
				return fmt.Errorf("graph impact: --depth must be greater than zero")
			}
			if !cmd.Flags().Changed("json") || !jsonOut {
				return fmt.Errorf("graph impact: requires --json")
			}

			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph impact: %w", err)
			}
			byID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
			}
			if byID[focus] == nil {
				return fmt.Errorf("graph impact: note %q not found", focus)
			}

			adj := make(map[string][]graphImpactAdjacencyEdge, len(notes))
			for _, n := range notes {
				for _, lnk := range n.Links {
					if !allowedTypes[lnk.Type] || byID[lnk.TargetID] == nil {
						continue
					}
					witness := pathWitnessEdge{
						From: n.ID, To: lnk.TargetID, Type: lnk.Type, Annotation: lnk.Annotation,
					}
					if direction == "outgoing" {
						adj[n.ID] = append(adj[n.ID], graphImpactAdjacencyEdge{next: lnk.TargetID, witness: witness})
					} else {
						adj[lnk.TargetID] = append(adj[lnk.TargetID], graphImpactAdjacencyEdge{next: n.ID, witness: witness})
					}
				}
			}
			for id := range adj {
				sort.Slice(adj[id], func(i, j int) bool {
					a, b := adj[id][i], adj[id][j]
					if a.next != b.next {
						return a.next < b.next
					}
					if a.witness.Type != b.witness.Type {
						return a.witness.Type < b.witness.Type
					}
					return a.witness.Annotation < b.witness.Annotation
				})
			}

			predecessor := map[string]string{focus: ""}
			predecessorEdge := make(map[string]pathWitnessEdge)
			depthByID := map[string]int{focus: 0}
			queue := []string{focus}
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				if depthByID[current] >= depth {
					continue
				}
				for _, edge := range adj[current] {
					if _, visited := predecessor[edge.next]; visited {
						continue
					}
					predecessor[edge.next] = current
					predecessorEdge[edge.next] = edge.witness
					depthByID[edge.next] = depthByID[current] + 1
					queue = append(queue, edge.next)
				}
			}

			impactIDs := make([]string, 0, len(predecessor)-1)
			for id := range predecessor {
				if id != focus {
					impactIDs = append(impactIDs, id)
				}
			}
			sort.Slice(impactIDs, func(i, j int) bool {
				if depthByID[impactIDs[i]] != depthByID[impactIDs[j]] {
					return depthByID[impactIDs[i]] < depthByID[impactIDs[j]]
				}
				return impactIDs[i] < impactIDs[j]
			})

			out := graphImpactOutput{
				Focus:     graphImpactNode{ID: focus, Title: byID[focus].Title},
				Direction: direction,
				Links:     normalizedLinks,
				Depth:     depth,
				Impacts:   make([]graphImpactEntry, 0, len(impactIDs)),
			}
			for _, impactID := range impactIDs {
				var reversedIDs []string
				var reversedEdges []pathWitnessEdge
				for current := impactID; current != ""; current = predecessor[current] {
					reversedIDs = append(reversedIDs, current)
					if edge, ok := predecessorEdge[current]; ok {
						reversedEdges = append(reversedEdges, edge)
					}
				}
				for left, right := 0, len(reversedIDs)-1; left < right; left, right = left+1, right-1 {
					reversedIDs[left], reversedIDs[right] = reversedIDs[right], reversedIDs[left]
				}
				for left, right := 0, len(reversedEdges)-1; left < right; left, right = left+1, right-1 {
					reversedEdges[left], reversedEdges[right] = reversedEdges[right], reversedEdges[left]
				}
				pathNodes := make([]graphImpactNode, len(reversedIDs))
				for i, id := range reversedIDs {
					pathNodes[i] = graphImpactNode{ID: id, Title: byID[id].Title}
				}
				out.Impacts = append(out.Impacts, graphImpactEntry{
					Node:  graphImpactNode{ID: impactID, Title: byID[impactID].Title},
					Depth: depthByID[impactID],
					Nodes: pathNodes,
					Edges: reversedEdges,
				})
			}

			enc := json.NewEncoder(outWriter(cmd))
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	cmd.Flags().StringVar(&focus, "focus", "", "Focus note ID (required)")
	cmd.Flags().StringVar(&links, "links", "", "Comma-separated canonical link types (required)")
	cmd.Flags().StringVar(&direction, "direction", "", "Traversal direction: incoming or outgoing (required)")
	cmd.Flags().IntVar(&depth, "depth", 0, "Positive traversal depth (required)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output impact witnesses as JSON (required)")
	return cmd
}
