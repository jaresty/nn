package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"

	"github.com/jaresty/nn/internal/note"
)

// DefSite is a located symbol definition extracted from a source file.
type DefSite struct {
	Name      string
	Kind      string
	File      string
	StartLine int
	EndLine   int
	StartByte uint32
	EndByte   uint32
	Source    []byte
	CycleMarker string
}

// Index maps symbol names to their definition sites across a directory tree.
type Index struct {
	ByName map[string][]*DefSite
	All    []*DefSite
}

// Node is a call-graph node in the trace result.
type Node struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Kind              string    `json:"kind"`
	File              string    `json:"file"`
	Line              int       `json:"line"`
	Resolved          bool      `json:"resolved"`
	CycleMarker       string    `json:"cycle_marker,omitempty"`
	AmbiguousReceiver bool      `json:"ambiguous_receiver,omitempty"`
	Receiver          string    `json:"receiver,omitempty"`
	NNNotes           []NoteRef `json:"nn_notes"`
}

// Edge is a directed call edge in the trace result.
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Resolved bool   `json:"resolved"`
}

// Result is the full call-graph output of a Trace call.
type Result struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// NoteRef is a reference to an nn note attached to a resolved node.
type NoteRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// BuildIndex walks root, parses all grammar-detected files via gotreesitter, and
// returns an Index of all definition sites.
func BuildIndex(root string) (*Index, error) {
	idx := &Index{ByName: map[string][]*DefSite{}}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if n := d.Name(); n == ".git" || n == "vendor" || n == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if grammars.DetectLanguage(path) == nil {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		entry := grammars.DetectLanguage(path)
		parser := gotreesitter.NewParser(entry.Language())
		tree, err := parser.Parse(src)
		if err != nil || tree == nil {
			return nil
		}
		lines := lineTable(src)
		for _, span := range gotreesitter.ExtractDefinitionSpans(tree) {
			name := string(src[span.NameStartByte:span.NameEndByte])
			def := &DefSite{
				Name:      name,
				Kind:      span.Kind,
				File:      path,
				StartLine: byteToLine(lines, span.StartByte),
				EndLine:   byteToLine(lines, span.EndByte),
				StartByte: span.StartByte,
				EndByte:   span.EndByte,
				Source:    src,
			}
			idx.All = append(idx.All, def)
			idx.ByName[name] = append(idx.ByName[name], def)
		}
		return nil
	})
	return idx, err
}

func lineTable(src []byte) []uint32 {
	offsets := []uint32{0}
	for i, b := range src {
		if b == '\n' {
			offsets = append(offsets, uint32(i+1))
		}
	}
	return offsets
}

func byteToLine(lines []uint32, offset uint32) int {
	lo, hi := 0, len(lines)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if lines[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

// Trace performs a DFS from the named entry-point symbols up to maxDepth, annotates
// each resolved node with related nn notes via BM25, and returns the graph result.
func Trace(idx *Index, symbols []string, maxDepth int, notes []*note.Note) *Result {
	result := &Result{}
	nodeSet := map[string]bool{}
	seen := map[string]bool{}

	var allInbound map[string][]string
	if len(notes) > 0 {
		allInbound = make(map[string][]string)
		for _, n := range notes {
			for _, lnk := range n.Links {
				allInbound[lnk.TargetID] = append(allInbound[lnk.TargetID], lnk.Annotation)
			}
		}
	}

	var dfs func(def *defSiteWithSource, depth int)
	dfs = func(def *defSiteWithSource, depth int) {
		nodeID := def.File + ":" + def.Name
		cycleMarker := ""
		if seen[nodeID] {
			cycleMarker = "already expanded"
		}

		n := Node{
			ID:          nodeID,
			Name:        def.Name,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.StartLine,
			Resolved:    true,
			CycleMarker: cycleMarker,
			NNNotes:     []NoteRef{},
		}

		if len(notes) > 0 {
			query := def.Name
			scores := note.BM25Scores(notes, query, allInbound)
			type scored struct {
				n     *note.Note
				score float64
			}
			var ranked []scored
			for _, nt := range notes {
				if s := scores[nt.ID]; s > 0 {
					ranked = append(ranked, scored{nt, s})
				}
			}
			sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
			if len(ranked) > 2 {
				ranked = ranked[:2]
			}
			for _, r := range ranked {
				n.NNNotes = append(n.NNNotes, NoteRef{ID: r.n.ID, Title: r.n.Title})
			}
		}

		if cycleMarker != "" {
			// Always append cycle-marker nodes so callers can detect the cycle.
			result.Nodes = append(result.Nodes, n)
			return
		}

		if !nodeSet[nodeID] {
			nodeSet[nodeID] = true
			result.Nodes = append(result.Nodes, n)
		}

		if depth >= maxDepth {
			return
		}
		seen[nodeID] = true

		calls := extractCallsInDef(def)
		callSeen := map[string]bool{}
		for _, call := range calls {
			ck := call.receiver + "." + call.name
			if callSeen[ck] {
				continue
			}
			callSeen[ck] = true

			targets := idx.ByName[call.name]
			if len(targets) == 0 {
				edge := Edge{From: nodeID, To: call.name, Resolved: false}
				result.Edges = append(result.Edges, edge)
				continue
			}
			ambiguous := call.receiver != "" && len(targets) > 1
			for _, t := range targets {
				toID := t.File + ":" + t.Name
				result.Edges = append(result.Edges, Edge{From: nodeID, To: toID, Resolved: true})
				child := &defSiteWithSource{DefSite: t, srcBytes: def.srcBytes}
				// Recurse first, then annotate the node that was added.
				prevLen := len(result.Nodes)
				dfs(child, depth+1)
				if ambiguous {
					for i := prevLen; i < len(result.Nodes); i++ {
						if result.Nodes[i].ID == toID {
							result.Nodes[i].AmbiguousReceiver = true
							result.Nodes[i].Receiver = call.receiver
						}
					}
				}
			}
		}
	}

	for _, sym := range symbols {
		defs := idx.ByName[sym]
		if len(defs) == 0 {
			result.Nodes = append(result.Nodes, Node{
				ID:       sym,
				Name:     sym,
				Resolved: false,
				NNNotes:  []NoteRef{},
			})
			continue
		}
		for _, d := range defs {
			dfs(&defSiteWithSource{DefSite: d}, 0)
		}
	}

	return result
}

type defSiteWithSource struct {
	*DefSite
	srcBytes []byte
}

type callRef struct {
	name     string
	receiver string
}

func extractCallsInDef(def *defSiteWithSource) []callRef {
	entry := grammars.DetectLanguage(def.File)
	if entry == nil {
		return nil
	}
	parser := gotreesitter.NewParser(entry.Language())
	tree, err := parser.Parse(def.Source)
	if err != nil || tree == nil {
		return nil
	}
	refs := gotreesitter.ExtractCalls(tree)
	var out []callRef
	for _, ref := range refs {
		if ref.StartByte < def.StartByte || ref.StartByte >= def.EndByte {
			continue
		}
		name := string(def.Source[ref.NameStartByte:ref.NameEndByte])
		out = append(out, callRef{name: name, receiver: ref.Receiver})
	}
	return out
}

// nodeID returns a stable string key for a DefSite.
func nodeID(def *DefSite) string {
	return fmt.Sprintf("%s:%s", def.File, def.Name)
}
