package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [4]: --root with a non-"auto" value returns an error
func TestCheckRootInvalidValue(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeModel)
	n.Representation = "ontology"
	writeNoteFile(t, nbDir, n)

	_, err := execute("check", n.ID, "--root", "someID")
	if err == nil {
		t.Fatal("expected error for --root with non-auto value, got nil")
	}
}
