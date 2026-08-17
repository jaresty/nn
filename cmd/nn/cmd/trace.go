package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
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

			// Detect file:line input: resolve symbol from AST index.
			if sym, dir, ok := resolveFileLineSymbol(root); ok {
				root = dir
				if len(symbols) == 0 {
					symbols = []string{sym}
				}
			} else if len(symbols) == 0 {
				return fmt.Errorf("required flag(s) \"symbol\" not set")
			}

			idx, err := trace.BuildIndex(root)
			if err != nil {
				return fmt.Errorf("build index: %w", err)
			}

			sessionReads := loadSessionReads(resolveCfgDir())
			notes, _ := state.backend.List()

			prepared := prepareCorpus(notes, state.notebookDir)
			result := trace.Trace(idx, symbols, depth, traceAnnotator(prepared, 2))

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
				if n.AmbiguousReceiver {
					candidates := 0
					for _, e := range edges[n.ID] {
						if e.Resolved {
							candidates++
						}
					}
					marker += fmt.Sprintf(" [receiver: %s, %d candidates — type-unqualified]", n.Receiver, candidates+1)
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
			hasUnread := false
			for _, ref := range allNotes {
				readMarker := ""
				if sessionReads[ref.ID] {
					readMarker = " [read]"
				} else {
					hasUnread = true
				}
				fmt.Fprintf(w, "- [[%s|%s]] [likely relevant]%s\n", ref.ID, ref.Title, readMarker)
			}
			printResolveInstruction(w, hasUnread)

			return nil
		},
	}

	cmd.Flags().StringArrayVar(&symbols, "symbol", nil, "Entry-point symbol name (repeatable)")
	cmd.Flags().IntVar(&depth, "depth", 3, "DFS depth limit")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON graph")
	cmd.Flags().BoolVar(&showUnresolved, "show-unresolved", false, "Show unresolved (stdlib/external) leaves")
	return cmd
}

// resolveFileLineSymbol detects "file:line" format in arg, builds an AST index
// on the file's directory, and returns the symbol name spanning that line.
func resolveFileLineSymbol(arg string) (symbol, dir string, ok bool) {
	colon := strings.LastIndex(arg, ":")
	if colon < 0 {
		return "", "", false
	}
	lineNum, err := strconv.Atoi(arg[colon+1:])
	if err != nil {
		return "", "", false
	}
	file := arg[:colon]
	dir = filepath.Dir(file)

	idx, err := trace.BuildIndex(dir)
	if err != nil {
		return "", "", false
	}
	for _, sites := range idx.ByName {
		for _, s := range sites {
			if s.File == file && s.StartLine <= lineNum && s.EndLine >= lineNum {
				return s.Name, dir, true
			}
		}
	}
	return "", "", false
}
