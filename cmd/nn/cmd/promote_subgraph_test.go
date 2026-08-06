package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: --subgraph promotes all reachable same-representation notes
// property [2]: --if-valid blocks promotion when violations exist
// property [3]: --if-valid allows promotion when subgraph is valid
func TestPromoteSubgraphAll(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Model", note.TypeModel)
	root.Representation = "ontology"
	root.Status = note.StatusDraft

	child := newTestNoteForCLI(note.GenerateID(), "Child Concept", note.TypeConcept)
	child.Representation = "ontology"
	child.Status = note.StatusDraft
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "includes", Type: "extends"}}

	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("promote", root.ID, "--subgraph", root.ID, "--to", "reviewed")
	if err != nil {
		t.Fatalf("promote --subgraph: %v\noutput: %s", err, out)
	}

	// Both root and child should now be reviewed.
	listOut, err := execute("list", "--json", "--fields", "id,status")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var notes []map[string]any
	if err := json.Unmarshal([]byte(listOut), &notes); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	statuses := map[string]string{}
	for _, n := range notes {
		statuses[n["id"].(string)] = n["status"].(string)
	}
	if statuses[root.ID] != "reviewed" {
		t.Errorf("root: expected status=reviewed, got %q", statuses[root.ID])
	}
	if statuses[child.ID] != "reviewed" {
		t.Errorf("child: expected status=reviewed, got %q", statuses[child.ID])
	}
}

func TestPromoteSubgraphIfValidBlocks(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Invalid: root is not type:model
	badRoot := newTestNoteForCLI(note.GenerateID(), "Bad Root", note.TypeConcept)
	badRoot.Representation = "ontology"
	badRoot.Status = note.StatusDraft
	writeNoteFile(t, nbDir, badRoot)

	_, err := execute("promote", badRoot.ID, "--subgraph", badRoot.ID, "--to", "reviewed", "--if-valid")
	if err == nil {
		t.Fatal("expected error for invalid subgraph with --if-valid, got nil")
	}
	if !strings.Contains(err.Error(), "fails") && !strings.Contains(err.Error(), "violat") {
		t.Errorf("expected violation message, got: %v", err)
	}
}

func TestPromoteSubgraphIfValidPasses(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Valid Root", note.TypeModel)
	root.Representation = "ontology"
	root.Status = note.StatusDraft
	writeNoteFile(t, nbDir, root)

	out, err := execute("promote", root.ID, "--subgraph", root.ID, "--to", "reviewed", "--if-valid")
	if err != nil {
		t.Fatalf("promote --subgraph --if-valid on valid graph: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "reviewed") && !strings.Contains(out, "ok") && !strings.Contains(out, "promoted") {
		t.Errorf("expected success indicator in output, got: %s", out)
	}
}
