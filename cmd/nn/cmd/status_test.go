package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestStatusOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan", note.TypeConcept)
	draft := newTestNoteForCLI(note.GenerateID(), "Draft", note.TypeConcept)
	writeNoteFile(t, nbDir, orphan)
	writeNoteFile(t, nbDir, draft)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "orphan") && !strings.Contains(lower, "draft") {
		t.Errorf("status output missing health info: %q", out)
	}
}
