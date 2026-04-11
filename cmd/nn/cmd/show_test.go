package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestShowNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Show Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "Show Me") {
		t.Errorf("output %q does not contain title 'Show Me'", out)
	}
}

func TestShowNoteNotFound(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("show", "99999999999999-0000")
	if err == nil {
		t.Fatal("nn show nonexistent: want error, got nil")
	}
}
