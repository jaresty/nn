package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: definition from LinkTypeDescriptions is printed
// property [2]: both note titles appear in output
// property [3]: unknown type returns error
func TestExplainLinkPrintsDefinition(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("explain-link", a.ID, b.ID, "--type", "refines")
	if err != nil {
		t.Fatalf("nn explain-link: %v\noutput: %s", err, out)
	}

	// property [1]: definition text must appear
	def := note.LinkTypeDescriptions["refines"]
	if !strings.Contains(out, def[:30]) {
		t.Errorf("property [1]: expected definition text in output, got:\n%s", out)
	}

	// property [2]: both titles must appear
	if !strings.Contains(out, "Source Note") {
		t.Errorf("property [2]: expected source title in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Target Note") {
		t.Errorf("property [2]: expected target title in output, got:\n%s", out)
	}
}

func TestExplainLinkUnknownType(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	// property [3]: unknown type must error
	_, err := execute("explain-link", a.ID, b.ID, "--type", "invented-type")
	if err == nil {
		t.Fatal("expected error for unknown link type, got nil")
	}
}
