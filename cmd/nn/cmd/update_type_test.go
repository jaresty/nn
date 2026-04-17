package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn update <id> --type <type> changes the note's type.
func TestUpdateType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Type Changer", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--type", "argument", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --type: %v", err)
	}

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show after update: %v", err)
	}
	if !strings.Contains(out, "type: argument") {
		t.Errorf("expected type to be argument after update:\n%s", out)
	}
}

// Assertion: nn update <id> --type <invalid> returns an error.
func TestUpdateTypeInvalid(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Type Changer", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--type", "bogus", "--no-edit")
	if err == nil {
		t.Errorf("expected error for invalid type, got nil")
	}
}
