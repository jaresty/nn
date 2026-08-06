package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: --root auto finds the root model node via backlinks
// property [2]: check passes when root passes validation
// property [3]: error when no type:model root reachable
func TestCheckRootAutoFindsRoot(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Model", note.TypeModel)
	root.Representation = "ontology"

	child := newTestNoteForCLI(note.GenerateID(), "Child Concept", note.TypeConcept)
	child.Representation = "ontology"
	child.Links = []note.Link{{TargetID: root.ID, Annotation: "refines it", Type: "refines"}}

	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	// property [1]+[2]: check from child with --root auto should pass (root is valid)
	out, err := execute("check", child.ID, "--root", "auto")
	if err != nil {
		t.Fatalf("nn check --root auto from child: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("expected 'ok' in output, got: %s", out)
	}
	// property [1]: output should mention the root note, not the child
	if !strings.Contains(out, root.ID) {
		t.Errorf("expected root ID %s in output, got: %s", root.ID, out)
	}
}

func TestCheckRootAutoNoRoot(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Only a concept note, no model root reachable via backlinks.
	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan Concept", note.TypeConcept)
	orphan.Representation = "ontology"
	writeNoteFile(t, nbDir, orphan)

	// property [3]: should error — no type:model root reachable
	_, err := execute("check", orphan.ID, "--root", "auto")
	if err == nil {
		t.Fatalf("expected error when no root reachable, got nil")
	}
	if !strings.Contains(err.Error(), "no root") && !strings.Contains(err.Error(), "root") {
		t.Errorf("expected error mentioning 'root', got: %v", err)
	}
}
