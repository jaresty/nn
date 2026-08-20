package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/jaresty/nn/internal/note"
)

type depthEntry struct {
	n     *note.Note
	level int
}

// bfsDepth performs a BFS from root following outgoing links up to maxDepth hops.
// byID is a pre-built map of all notes indexed by ID.
func bfsDepth(root *note.Note, byID map[string]*note.Note, maxDepth int) []depthEntry {
	visited := map[string]bool{root.ID: true}
	queue := []depthEntry{{root, 0}}
	var ordered []depthEntry
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ordered = append(ordered, cur)
		if cur.level >= maxDepth {
			continue
		}
		for _, lnk := range cur.n.Links {
			if visited[lnk.TargetID] {
				continue
			}
			visited[lnk.TargetID] = true
			if target, ok := byID[lnk.TargetID]; ok {
				queue = append(queue, depthEntry{target, cur.level + 1})
			}
		}
	}
	return ordered
}

// bfsDepthBoth performs a BFS from root following links in BOTH directions
// (outgoing root->N and incoming N->root) up to maxDepth hops. Used where a
// focused ego subgraph must include the notes that link TO the root (e.g. the
// graph viewer's zoned layout), not just what the root links out to.
func bfsDepthBoth(root *note.Note, byID map[string]*note.Note, maxDepth int) []depthEntry {
	// Precompute inbound adjacency: target ID -> source notes linking to it.
	inbound := make(map[string][]*note.Note)
	for _, n := range byID {
		for _, lnk := range n.Links {
			if _, ok := byID[lnk.TargetID]; ok {
				inbound[lnk.TargetID] = append(inbound[lnk.TargetID], n)
			}
		}
	}

	visited := map[string]bool{root.ID: true}
	queue := []depthEntry{{root, 0}}
	var ordered []depthEntry
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ordered = append(ordered, cur)
		if cur.level >= maxDepth {
			continue
		}
		visit := func(next *note.Note) {
			if next == nil || visited[next.ID] {
				return
			}
			visited[next.ID] = true
			queue = append(queue, depthEntry{next, cur.level + 1})
		}
		for _, lnk := range cur.n.Links {
			visit(byID[lnk.TargetID])
		}
		for _, src := range inbound[cur.n.ID] {
			visit(src)
		}
	}
	return ordered
}

type depthNoteJSON struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
	Body     string   `json:"body"`
	Depth    int      `json:"depth"`
}

// printDepthJSON encodes a BFS result as a JSON array.
func printDepthJSON(w io.Writer, entries []depthEntry) error {
	out := make([]depthNoteJSON, len(entries))
	for i, e := range entries {
		tags := e.n.Tags
		if tags == nil {
			tags = []string{}
		}
		out[i] = depthNoteJSON{
			ID:       e.n.ID,
			Title:    e.n.Title,
			Type:     string(e.n.Type),
			Status:   string(e.n.Status),
			Tags:     tags,
			Created:  e.n.Created.UTC().Format(time.RFC3339Nano),
			Modified: e.n.Modified.UTC().Format(time.RFC3339Nano),
			Body:     e.n.Body,
			Depth:    e.level,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// printDepthMarkdown prints BFS entries as concatenated Markdown separated by ---.
func printDepthMarkdown(w io.Writer, entries []depthEntry) error {
	for i, e := range entries {
		if i > 0 {
			fmt.Fprintln(w, "---")
		}
		data, err := e.n.Marshal()
		if err != nil {
			return fmt.Errorf("marshal %s: %w", e.n.ID, err)
		}
		fmt.Fprint(w, string(data))
	}
	return nil
}
