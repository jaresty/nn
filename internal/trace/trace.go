package trace

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
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

// DefaultParseTimeoutMicros bounds each per-file gotreesitter parse. A single
// pathological file can otherwise drive gotreesitter's GLR error-recovery into
// unbounded transient allocation (observed >1GB). On timeout gotreesitter returns
// a partial tree with a nil error, which is acceptable for definition indexing.
// A value of 0 disables the timeout (gotreesitter's default, full-parse behavior).
var DefaultParseTimeoutMicros uint64 = 2_000_000

// BuildIndex walks root, parses all grammar-detected files via gotreesitter, and
// returns an Index of all definition sites. Files are parsed concurrently using a
// bounded goroutine pool.
func BuildIndex(root string) (*Index, error) {
	// Collect all parseable file paths first (WalkDir is not goroutine-safe).
	var paths []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if n := d.Name(); n == ".git" || n == "vendor" || n == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if grammars.DetectLanguage(path) != nil {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	type fileDefs struct {
		defs []*DefSite
	}
	results := make([]fileDefs, len(paths))

	g := new(errgroup.Group)
	g.SetLimit(4)
	for i, path := range paths {
		i, path := i, path
		g.Go(func() error {
			info, err := os.Stat(path)
			if err != nil {
				return nil
			}
			const maxFileBytes = 500 * 1024
			if info.Size() > maxFileBytes {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			entry := grammars.DetectLanguage(path)
			parser := gotreesitter.NewParser(entry.Language())
			// Bound per-file parse time so a single pathological file cannot drive
			// gotreesitter's GLR error-recovery into unbounded transient allocation.
			// On timeout Parse returns a partial tree with nil error, which yields
			// whatever definition spans were recovered before the deadline.
			parser.SetTimeoutMicros(DefaultParseTimeoutMicros)
			tree, err := parser.Parse(src)
			if err != nil || tree == nil {
				return nil
			}
			lines := lineTable(src)
			var defs []*DefSite
			for _, span := range gotreesitter.ExtractDefinitionSpans(tree) {
				name := string(src[span.NameStartByte:span.NameEndByte])
				defs = append(defs, &DefSite{
					Name:      name,
					Kind:      span.Kind,
					File:      path,
					StartLine: byteToLine(lines, span.StartByte),
					EndLine:   byteToLine(lines, span.EndByte),
					StartByte: span.StartByte,
					EndByte:   span.EndByte,
					Source:    src,
				})
			}
			results[i] = fileDefs{defs: defs}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	idx := &Index{ByName: map[string][]*DefSite{}}
	for _, r := range results {
		for _, def := range r.defs {
			idx.All = append(idx.All, def)
			idx.ByName[def.Name] = append(idx.ByName[def.Name], def)
		}
	}
	return idx, nil
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

// Annotator maps a node's source-span text to related nn notes. The caller
// supplies the ranking (for example, the memoizing per-field BM25 scorer used by
// nn grep/ast/shuf), keeping trace decoupled from the note/index packages. A nil
// Annotator leaves every node's NNNotes empty.
type Annotator func(query string) []NoteRef

// Trace performs a DFS from the named entry-point symbols up to maxDepth,
// annotates each resolved node with related nn notes via the supplied annotate
// function, and returns the graph result.
func Trace(idx *Index, symbols []string, maxDepth int, annotate Annotator) *Result {
	result := &Result{}
	nodeSet := map[string]bool{}
	fileCallCache := newCallCache()
	seen := map[string]bool{}

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

		if annotate != nil {
			query := string(def.Source[def.StartByte:def.EndByte])
			if refs := annotate(query); len(refs) > 0 {
				n.NNNotes = refs
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

		calls := fileCallCache.callsForDef(def)
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
	name      string
	receiver  string
	startByte uint32
}

// extractCallsInFile parses a file's source once and returns every call site
// with its start-byte offset, so callers can filter to a specific def's span
// without re-parsing.
func extractCallsInFile(file string, source []byte) []callRef {
	entry := grammars.DetectLanguage(file)
	if entry == nil {
		return nil
	}
	parser := gotreesitter.NewParser(entry.Language())
	tree, err := parser.Parse(source)
	if err != nil || tree == nil {
		return nil
	}
	refs := gotreesitter.ExtractCalls(tree)
	var out []callRef
	for _, ref := range refs {
		name := string(source[ref.NameStartByte:ref.NameEndByte])
		out = append(out, callRef{name: name, receiver: ref.Receiver, startByte: ref.StartByte})
	}
	return out
}

// callCache memoizes per-file call extraction so a file's source is parsed at
// most once per Trace invocation, rather than once per def visited in it.
type callCache struct {
	byFile map[string][]callRef
}

func newCallCache() *callCache {
	return &callCache{byFile: map[string][]callRef{}}
}

// callsForDef returns the calls whose sites fall within def's byte span, parsing
// the file's source at most once and reusing it for every def in that file.
func (c *callCache) callsForDef(def *defSiteWithSource) []callRef {
	fileRefs, ok := c.byFile[def.File]
	if !ok {
		fileRefs = extractCallsInFile(def.File, def.Source)
		c.byFile[def.File] = fileRefs
	}
	var out []callRef
	for _, ref := range fileRefs {
		if ref.startByte < def.StartByte || ref.startByte >= def.EndByte {
			continue
		}
		out = append(out, ref)
	}
	return out
}

// nodeID returns a stable string key for a DefSite.
func nodeID(def *DefSite) string {
	return fmt.Sprintf("%s:%s", def.File, def.Name)
}
