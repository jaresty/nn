package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [2]: --root auto traverses inbound links (root links TO child) to find root
// property [2b]: check passes when found root passes validation
// property [3]: error when no type:model root reachable via inbound links
func TestCheckRootAutoFindsRoot(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Model", note.TypeModel)
	root.Representation = "ontology"

	child := newTestNoteForCLI(note.GenerateID(), "Child Concept", note.TypeConcept)
	child.Representation = "ontology"

	// root links TO child — canonical direction per check_graph_test.go
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "extends it", Type: "extends"}}

	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	// property [2]: check from child with --root auto should find root via inbound link
	out, err := execute("check", child.ID, "--root", "auto")
	if err != nil {
		t.Fatalf("nn check --root auto from child: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("expected 'ok' in output, got: %s", out)
	}
	// property [2]: output should mention the root note, not only the child
	if !strings.Contains(out, root.ID) {
		t.Errorf("expected root ID %s in output, got: %s", root.ID, out)
	}
}

func TestCheckRootAutoNoRoot(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Only a concept note; no model note has an inbound link to it.
	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan Concept", note.TypeConcept)
	orphan.Representation = "ontology"
	writeNoteFile(t, nbDir, orphan)

	// property [3]: should error — no type:model root reachable via inbound links
	_, err := execute("check", orphan.ID, "--root", "auto")
	if err == nil {
		t.Fatalf("expected error when no root reachable, got nil")
	}
	if !strings.Contains(err.Error(), "root") {
		t.Errorf("expected error mentioning 'root', got: %v", err)
	}
}
