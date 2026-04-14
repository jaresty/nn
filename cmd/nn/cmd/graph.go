package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type graphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type graphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Annotation string `json:"annotation"`
	LinkType   string `json:"type,omitempty"`
}

type graphOutput struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

func newGraphCmd(state *rootState) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Output link relationships",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph: %w", err)
			}

			g := graphOutput{
				Nodes: make([]graphNode, 0, len(notes)),
				Edges: []graphEdge{},
			}
			for _, n := range notes {
				g.Nodes = append(g.Nodes, graphNode{
					ID:    n.ID,
					Title: n.Title,
					Type:  string(n.Type),
				})
				for _, lnk := range n.Links {
					g.Edges = append(g.Edges, graphEdge{
						From:       n.ID,
						To:         lnk.TargetID,
						Annotation: lnk.Annotation,
						LinkType:   lnk.Type,
					})
				}
			}

			if jsonOut {
				enc := json.NewEncoder(outWriter(cmd))
				enc.SetIndent("", "  ")
				return enc.Encode(g)
			}
			// Plain text: one edge per line
			for _, e := range g.Edges {
				if e.LinkType != "" {
					fmt.Fprintf(outWriter(cmd), "%s -> %s [%s] -- %s\n", e.From, e.To, e.LinkType, e.Annotation)
				} else {
					fmt.Fprintf(outWriter(cmd), "%s -> %s -- %s\n", e.From, e.To, e.Annotation)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
