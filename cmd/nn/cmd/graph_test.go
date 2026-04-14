package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestGraphTextIncludesType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "opposes claim", Type: "contradicts"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("graph")
	if err != nil {
		t.Fatalf("nn graph: %v", err)
	}
	if !strings.Contains(out, "[contradicts]") {
		t.Errorf("graph text output missing type:\n%s", out)
	}
}

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

func TestGraphJSONEdgeIncludesType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "opposes claim", Type: "contradicts"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("graph", "--json")
	if err != nil {
		t.Fatalf("nn graph --json: %v", err)
	}
	var result struct {
		Edges []struct {
			From       string `json:"from"`
			To         string `json:"to"`
			Annotation string `json:"annotation"`
			Type       string `json:"type"`
		} `json:"edges"`
	}
	mustJSON(t, out, &result)
	if len(result.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(result.Edges))
	}
	if result.Edges[0].Type != "contradicts" {
		t.Errorf("edge type = %q, want contradicts", result.Edges[0].Type)
	}
}
