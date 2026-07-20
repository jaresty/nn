package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jaresty/nn/internal/trace"
	"github.com/spf13/cobra"
)

func newTraceCmd(state *rootState) *cobra.Command {
	var symbols []string
	var depth int
	var asJSON bool
	var showUnresolved bool

	cmd := &cobra.Command{
		Use:   "trace <root-dir>",
		Short: "Syntax-aware call graph from entry-point symbols",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := args[0]

			idx, err := trace.BuildIndex(root)
			if err != nil {
				return fmt.Errorf("build index: %w", err)
			}

			notes, _ := state.backend.List()

			result := trace.Trace(idx, symbols, depth, notes)

			w := outWriter(cmd)

			if asJSON {
				b, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(w, string(b))
				return nil
			}

			// Human-readable tree: print nodes in order with indented edges.
			nodeByID := map[string]trace.Node{}
			for _, n := range result.Nodes {
				nodeByID[n.ID] = n
			}
			// Collect outgoing edges per node.
			edges := map[string][]trace.Edge{}
			for _, e := range result.Edges {
				edges[e.From] = append(edges[e.From], e)
			}

			printed := map[string]bool{}
			var printNode func(nodeID string, indent int)
			printNode = func(nodeID string, indent int) {
				n, ok := nodeByID[nodeID]
				if !ok {
					return
				}
				prefix := strings.Repeat("  ", indent)
				marker := ""
				if n.CycleMarker != "" {
					marker = " [" + n.CycleMarker + "]"
				}
				fmt.Fprintf(w, "%s%s (%s) [%s:%d]%s\n", prefix, n.Name, n.Kind, n.File, n.Line, marker)
				for _, ref := range n.NNNotes {
					fmt.Fprintf(w, "%s  note: [[%s|%s]]\n", prefix, ref.ID, ref.Title)
				}
				if printed[nodeID] || n.CycleMarker != "" {
					return
				}
				printed[nodeID] = true
				for _, e := range edges[nodeID] {
					if !e.Resolved && !showUnresolved {
						continue
					}
					if !e.Resolved {
						fmt.Fprintf(w, "%s  → %s [unresolved]\n", prefix, e.To)
						continue
					}
					printNode(e.To, indent+1)
				}
			}

			// Print entry points first.
			for _, sym := range symbols {
				for _, n := range result.Nodes {
					if n.Name == sym && !printed[n.ID] {
						printNode(n.ID, 0)
					}
				}
			}

			// Collect all unique notes surfaced across every node.
			seenNotes := map[string]bool{}
			var allNotes []trace.NoteRef
			for _, n := range result.Nodes {
				for _, ref := range n.NNNotes {
					if !seenNotes[ref.ID] {
						seenNotes[ref.ID] = true
						allNotes = append(allNotes, ref)
					}
				}
			}
			fmt.Fprintln(w, "\n## Related notes")
			for _, ref := range allNotes {
				fmt.Fprintf(w, "- [[%s|%s]] [likely relevant]\n", ref.ID, ref.Title)
			}
			fmt.Fprintln(w, "Resolve each related note before the next action — run `nn show <id>` to open, or write `skip-related: <id> [<id> ...] — <reason>` to dismiss. Continuing without resolving is a protocol violation.")

			return nil
		},
	}

	cmd.Flags().StringArrayVar(&symbols, "symbol", nil, "Entry-point symbol name (repeatable)")
	cmd.Flags().IntVar(&depth, "depth", 3, "DFS depth limit")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON graph")
	cmd.Flags().BoolVar(&showUnresolved, "show-unresolved", false, "Show unresolved (stdlib/external) leaves")
	_ = cmd.MarkFlagRequired("symbol")
	return cmd
}
