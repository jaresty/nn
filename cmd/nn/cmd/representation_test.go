package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [3]: nn new --representation creates a note with Representation set.
func TestNewRepresentationFlag(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Domain Ontology", "--type", "model", "--representation", "ontology", "--no-edit")
	if err != nil {
		t.Fatalf("nn new --representation: %v\n%s", err, out)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	id = strings.Fields(id)[0]

	shown, err := execute("show", id)
	if err != nil {
		t.Fatalf("nn show %s: %v", id, err)
	}
	if !strings.Contains(shown, "representation: ontology") {
		t.Errorf("nn show output missing representation field:\n%s", shown)
	}
}

// property [4]: nn list --representation filters to notes with matching Representation.
func TestListRepresentationFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	match := newTestNoteForCLI(note.GenerateID(), "Ontology note", note.TypeModel)
	match.Representation = "ontology"
	writeNoteFile(t, nbDir, match)

	other := newTestNoteForCLI(note.GenerateID(), "Taxonomy note", note.TypeModel)
	other.Representation = "taxonomy"
	writeNoteFile(t, nbDir, other)

	out, err := execute("list", "--representation", "ontology")
	if err != nil {
		t.Fatalf("nn list --representation: %v\n%s", err, out)
	}
	if !strings.Contains(out, match.ID) {
		t.Errorf("nn list --representation=ontology missing matching note %q", match.ID)
	}
	if strings.Contains(out, other.ID) {
		t.Errorf("nn list --representation=ontology unexpectedly includes taxonomy note %q", other.ID)
	}
}
