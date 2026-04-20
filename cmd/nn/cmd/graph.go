package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
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
		Short: "Output link relationships and graph queries",
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
	cmd.AddCommand(
		newGraphTopCmd(state),
		newGraphOrphansCmd(state),
		newGraphBridgesCmd(state),
		newGraphShowCmd(state),
		newGraphExportCmd(state),
	)
	return cmd
}

// ── nn graph top ──────────────────────────────────────────────────────────────

func newGraphTopCmd(state *rootState) *cobra.Command {
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "top",
		Short: "Notes ranked by inbound link count",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph top: %w", err)
			}

			inbound := make(map[string]int)
			for _, n := range notes {
				for _, lnk := range n.Links {
					inbound[lnk.TargetID]++
				}
			}

			type entry struct {
				id, title string
				count     int
			}
			var entries []entry
			for _, n := range notes {
				if c := inbound[n.ID]; c > 0 {
					entries = append(entries, entry{n.ID, n.Title, c})
				}
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].count != entries[j].count {
					return entries[i].count > entries[j].count
				}
				return entries[i].id < entries[j].id
			})
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}

			w := outWriter(cmd)
			if format == "json" {
				type je struct {
					ID           string `json:"id"`
					Title        string `json:"title"`
					InboundCount int    `json:"inbound_count"`
				}
				out := make([]je, len(entries))
				for i, e := range entries {
					out[i] = je{e.id, e.title, e.count}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, e := range entries {
				fmt.Fprintf(w, "%s  %s  (%d inbound)\n", e.id, e.title, e.count)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum entries to show")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

// ── nn graph orphans ──────────────────────────────────────────────────────────

func newGraphOrphansCmd(state *rootState) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "orphans",
		Short: "Notes with no inbound or outbound links",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph orphans: %w", err)
			}

			targeted := make(map[string]bool)
			hasOutbound := make(map[string]bool)
			for _, n := range notes {
				for _, lnk := range n.Links {
					targeted[lnk.TargetID] = true
					hasOutbound[n.ID] = true
				}
			}

			// Global protocols (type=protocol with no outgoing governs links) are
			// not orphans — they intentionally have no governs targets.
			globalProtocol := make(map[string]bool)
			for _, n := range notes {
				if n.Type != note.TypeProtocol {
					continue
				}
				hasGoverns := false
				for _, lnk := range n.Links {
					if lnk.Type == "governs" {
						hasGoverns = true
						break
					}
				}
				if !hasGoverns {
					globalProtocol[n.ID] = true
				}
			}

			isOrphan := func(n *note.Note) bool {
				return !globalProtocol[n.ID] && !hasOutbound[n.ID] && !targeted[n.ID]
			}

			w := outWriter(cmd)
			if format == "json" {
				type je struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}
				var out []je
				for _, n := range notes {
					if isOrphan(n) {
						out = append(out, je{n.ID, n.Title})
					}
				}
				if out == nil {
					out = []je{}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, n := range notes {
				if isOrphan(n) {
					fmt.Fprintf(w, "%s  %s\n", n.ID, n.Title)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

// ── nn graph bridges ──────────────────────────────────────────────────────────

func newGraphBridgesCmd(state *rootState) *cobra.Command {
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "bridges",
		Short: "Notes that connect otherwise-disconnected parts of the graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph bridges: %w", err)
			}

			// inboundFrom[N] = set of notes that link TO N
			// outboundTo[N]  = set of notes that N links TO
			inboundFrom := make(map[string]map[string]bool)
			outboundTo := make(map[string]map[string]bool)
			for _, n := range notes {
				for _, lnk := range n.Links {
					if inboundFrom[lnk.TargetID] == nil {
						inboundFrom[lnk.TargetID] = make(map[string]bool)
					}
					inboundFrom[lnk.TargetID][n.ID] = true
					if outboundTo[n.ID] == nil {
						outboundTo[n.ID] = make(map[string]bool)
					}
					outboundTo[n.ID][lnk.TargetID] = true
				}
			}

			type entry struct {
				id, title string
				score     int
			}
			var entries []entry
			for _, n := range notes {
				inCount := len(inboundFrom[n.ID])
				outCount := len(outboundTo[n.ID])
				if inCount > 0 && outCount > 0 {
					entries = append(entries, entry{n.ID, n.Title, inCount * outCount})
				}
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].score != entries[j].score {
					return entries[i].score > entries[j].score
				}
				return entries[i].id < entries[j].id
			})
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}

			w := outWriter(cmd)
			if format == "json" {
				type je struct {
					ID    string `json:"id"`
					Title string `json:"title"`
					Score int    `json:"score"`
				}
				out := make([]je, len(entries))
				for i, e := range entries {
					out[i] = je{e.id, e.title, e.score}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, e := range entries {
				fmt.Fprintf(w, "%s  %s  (score %d)\n", e.id, e.title, e.score)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum entries to show")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

// ── nn graph show ─────────────────────────────────────────────────────────────

func newGraphShowCmd(state *rootState) *cobra.Command {
	var focus string
	var depth int
	var format string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Subgraph as structured data (LLM-facing)",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph show: %w", err)
			}

			byID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
			}

			type showNode struct {
				ID    string   `json:"id"`
				Title string   `json:"title"`
				Type  string   `json:"type"`
				Tags  []string `json:"tags"`
			}
			type showEdge struct {
				From       string `json:"from"`
				To         string `json:"to"`
				Annotation string `json:"annotation,omitempty"`
				LinkType   string `json:"type,omitempty"`
			}

			var resultNodes []showNode
			var resultEdges []showEdge

			if focus != "" {
				root, ok := byID[focus]
				if !ok {
					return fmt.Errorf("graph show: note %q not found", focus)
				}
				entries := bfsDepth(root, byID, depth)
				visited := make(map[string]bool, len(entries))
				for _, e := range entries {
					visited[e.n.ID] = true
					tags := e.n.Tags
					if tags == nil {
						tags = []string{}
					}
					resultNodes = append(resultNodes, showNode{e.n.ID, e.n.Title, string(e.n.Type), tags})
				}
				sort.Slice(resultNodes, func(i, j int) bool { return resultNodes[i].ID < resultNodes[j].ID })
				for _, e := range entries {
					for _, lnk := range e.n.Links {
						if visited[lnk.TargetID] {
							resultEdges = append(resultEdges, showEdge{e.n.ID, lnk.TargetID, lnk.Annotation, lnk.Type})
						}
					}
				}
			} else {
				for _, n := range notes {
					tags := n.Tags
					if tags == nil {
						tags = []string{}
					}
					resultNodes = append(resultNodes, showNode{n.ID, n.Title, string(n.Type), tags})
					for _, lnk := range n.Links {
						resultEdges = append(resultEdges, showEdge{n.ID, lnk.TargetID, lnk.Annotation, lnk.Type})
					}
				}
				sort.Slice(resultNodes, func(i, j int) bool { return resultNodes[i].ID < resultNodes[j].ID })
			}
			if resultEdges == nil {
				resultEdges = []showEdge{}
			}

			w := outWriter(cmd)
			if format == "json" {
				if focus != "" {
					out := struct {
						Center string     `json:"center"`
						Nodes  []showNode `json:"nodes"`
						Edges  []showEdge `json:"edges"`
					}{focus, resultNodes, resultEdges}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(out)
				}
				out := struct {
					Nodes []showNode `json:"nodes"`
					Edges []showEdge `json:"edges"`
				}{resultNodes, resultEdges}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, n := range resultNodes {
				fmt.Fprintf(w, "%s  %s\n", n.ID, n.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&focus, "focus", "", "Center note ID for ego-graph")
	cmd.Flags().IntVar(&depth, "depth", 2, "BFS depth from focus note")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

// ── nn graph export ───────────────────────────────────────────────────────────

func newGraphExportCmd(state *rootState) *cobra.Command {
	var format string
	var focus string
	var depth int
	var open bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the graph as DOT or SVG",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph export: %w", err)
			}

			byID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
			}

			var enodes []dotNode
			var eedges []dotEdge

			if focus != "" {
				root, ok := byID[focus]
				if !ok {
					return fmt.Errorf("graph export: note %q not found", focus)
				}
				entries := bfsDepth(root, byID, depth)
				visited := make(map[string]bool, len(entries))
				for _, e := range entries {
					visited[e.n.ID] = true
					enodes = append(enodes, dotNode{e.n.ID, e.n.Title})
				}
				for _, e := range entries {
					for _, lnk := range e.n.Links {
						if visited[lnk.TargetID] {
							eedges = append(eedges, dotEdge{e.n.ID, lnk.TargetID, lnk.Annotation})
						}
					}
				}
			} else {
				for _, n := range notes {
					enodes = append(enodes, dotNode{n.ID, n.Title})
					for _, lnk := range n.Links {
						eedges = append(eedges, dotEdge{n.ID, lnk.TargetID, lnk.Annotation})
					}
				}
			}

			dot := buildDOT(enodes, eedges)
			w := outWriter(cmd)

			switch format {
			case "dot":
				fmt.Fprint(w, dot)
				return nil
			case "svg":
				svg, err := dotToSVG(dot)
				if err != nil {
					return fmt.Errorf("graph export svg: %w (is graphviz installed?)", err)
				}
				if open {
					return openFile([]byte(svg), ".svg")
				}
				fmt.Fprint(w, svg)
				return nil
			default:
				return fmt.Errorf("graph export: unknown format %q (use dot or svg)", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "dot", "Output format: dot or svg")
	cmd.Flags().StringVar(&focus, "focus", "", "Center note ID for ego-graph")
	cmd.Flags().IntVar(&depth, "depth", 2, "BFS depth from focus note (when --focus is set)")
	cmd.Flags().BoolVar(&open, "open", false, "Open output in default viewer (svg only)")
	return cmd
}

// ── DOT / file helpers ────────────────────────────────────────────────────────

type dotNode struct{ id, title string }
type dotEdge struct{ from, to, annotation string }

func buildDOT(nodes []dotNode, edges []dotEdge) string {
	var sb strings.Builder
	sb.WriteString("digraph nn {\n")
	sb.WriteString("  graph [rankdir=LR];\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("  %q [label=%q];\n", n.id, n.title))
	}
	for _, e := range edges {
		if e.annotation != "" {
			sb.WriteString(fmt.Sprintf("  %q -> %q [label=%q];\n", e.from, e.to, e.annotation))
		} else {
			sb.WriteString(fmt.Sprintf("  %q -> %q;\n", e.from, e.to))
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

func dotToSVG(dot string) (string, error) {
	c := exec.Command("dot", "-Tsvg")
	c.Stdin = strings.NewReader(dot)
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func openFile(data []byte, ext string) error {
	f, err := os.CreateTemp("", "nn-graph-*"+ext)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	f.Close()
	openCmd := "xdg-open"
	if runtime.GOOS == "darwin" {
		openCmd = "open"
	}
	return exec.Command(openCmd, f.Name()).Start()
}
