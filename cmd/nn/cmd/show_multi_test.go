package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn show <id1> <id2> outputs both notes separated by ---.
func TestShowMultipleIDs(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n1 := newTestNoteForCLI(note.GenerateID(), "First Note", note.TypeConcept)
	n2 := newTestNoteForCLI(note.GenerateID(), "Second Note", note.TypeArgument)
	writeNoteFile(t, nbDir, n1)
	writeNoteFile(t, nbDir, n2)

	out, err := execute("show", n1.ID, n2.ID)
	if err != nil {
		t.Fatalf("nn show multi: %v", err)
	}
	if !strings.Contains(out, "First Note") {
		t.Errorf("first note missing from output:\n%s", out)
	}
	if !strings.Contains(out, "Second Note") {
		t.Errorf("second note missing from output:\n%s", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("separator --- missing from multi-show output:\n%s", out)
	}
}

// Assertion: nn show with single ID still works (no regression).
func TestShowSingleIDUnchanged(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Solo Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show single: %v", err)
	}
	if !strings.Contains(out, "Solo Note") {
		t.Errorf("note title missing from output:\n%s", out)
	}
}
