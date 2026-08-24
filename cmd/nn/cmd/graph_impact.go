package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

type graphImpactEntry struct {
	Node      typedWitnessNode `json:"node"`
	Depth     int              `json:"depth"`
	Witnesses []typedWitness   `json:"witnesses"`
}

type graphImpactOutput struct {
	Focus     typedWitnessNode   `json:"focus"`
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
			titles := make(map[string]string, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
				titles[n.ID] = n.Title
			}
			if byID[focus] == nil {
				return fmt.Errorf("graph impact: note %q not found", focus)
			}

			adj := make(map[string][]typedTraversalEdge, len(notes))
			for _, n := range notes {
				for _, lnk := range n.Links {
					if !allowedTypes[lnk.Type] || byID[lnk.TargetID] == nil {
						continue
					}
					storedEdge := typedWitnessEdge{
						From: n.ID, To: lnk.TargetID, Type: lnk.Type, Annotation: lnk.Annotation,
					}
					if direction == "outgoing" {
						adj[n.ID] = append(adj[n.ID], typedTraversalEdge{next: lnk.TargetID, edge: storedEdge})
					} else {
						adj[lnk.TargetID] = append(adj[lnk.TargetID], typedTraversalEdge{next: n.ID, edge: storedEdge})
					}
				}
			}
			witnessSearch := findShortestTypedWitnesses(focus, titles, adj, depth)

			impactIDs := make([]string, 0, len(witnessSearch.depthByID)-1)
			for id := range witnessSearch.depthByID {
				if id != focus {
					impactIDs = append(impactIDs, id)
				}
			}
			sort.Slice(impactIDs, func(i, j int) bool {
				if witnessSearch.depthByID[impactIDs[i]] != witnessSearch.depthByID[impactIDs[j]] {
					return witnessSearch.depthByID[impactIDs[i]] < witnessSearch.depthByID[impactIDs[j]]
				}
				return impactIDs[i] < impactIDs[j]
			})

			out := graphImpactOutput{
				Focus:     typedWitnessNode{ID: focus, Title: byID[focus].Title},
				Direction: direction,
				Links:     normalizedLinks,
				Depth:     depth,
				Impacts:   make([]graphImpactEntry, 0, len(impactIDs)),
			}
			for _, impactID := range impactIDs {
				out.Impacts = append(out.Impacts, graphImpactEntry{
					Node:      typedWitnessNode{ID: impactID, Title: byID[impactID].Title},
					Depth:     witnessSearch.depthByID[impactID],
					Witnesses: witnessSearch.witnessesTo(impactID),
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
