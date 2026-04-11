package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestGraphJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("graph", "--json")
	if err != nil {
		t.Fatalf("nn graph --json: %v", err)
	}
	var result map[string]any
	mustJSON(t, out, &result)

	nodes, ok := result["nodes"]
	if !ok {
		t.Fatal("graph JSON missing 'nodes' key")
	}
	edges, ok := result["edges"]
	if !ok {
		t.Fatal("graph JSON missing 'edges' key")
	}
	_ = nodes
	_ = edges
}
