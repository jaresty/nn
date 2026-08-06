package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: bulk-update sets applies_when on existing notes
func TestBulkUpdateAppliesWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Protocol Note", note.TypeProtocol)
	writeNoteFile(t, nbDir, n)

	spec := `[{"id":"` + n.ID + `","applies_when":"when context is X"}]`
	out, err := execute("bulk-update", "--json", spec)
	if err != nil {
		t.Fatalf("bulk-update: %v\noutput: %s", err, out)
	}

	showOut, _ := execute("show", n.ID)
	if !strings.Contains(showOut, "when context is X") {
		t.Errorf("property [1]: applies_when not updated; show output: %s", showOut)
	}
}

// property [1]: bulk-update sets title
func TestBulkUpdateTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Old Title", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	spec := `[{"id":"` + n.ID + `","title":"New Title"}]`
	out, err := execute("bulk-update", "--json", spec)
	if err != nil {
		t.Fatalf("bulk-update: %v\noutput: %s", err, out)
	}

	all, _ := execute("list", "--json")
	if !strings.Contains(all, "New Title") {
		t.Errorf("property [1]: title not updated; list output: %s", all)
	}
}

// property [2]: unknown id returns error, no writes
func TestBulkUpdateUnknownID(t *testing.T) {
	_, execute := setupNotebook(t)

	spec := `[{"id":"nonexistent-id","applies_when":"x"}]`
	_, err := execute("bulk-update", "--json", spec)
	if err == nil {
		t.Error("property [2]: expected error for unknown note ID")
	}
}

// property [3]: multiple updates in one batch
func TestBulkUpdateMultiple(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Note A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Note B", note.TypeConcept)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	spec := `[{"id":"` + a.ID + `","title":"Note A Updated"},{"id":"` + b.ID + `","title":"Note B Updated"}]`
	out, err := execute("bulk-update", "--json", spec)
	if err != nil {
		t.Fatalf("bulk-update multiple: %v\noutput: %s", err, out)
	}

	all, _ := execute("list", "--json")
	if !strings.Contains(all, "Note A Updated") || !strings.Contains(all, "Note B Updated") {
		t.Errorf("property [3]: not all notes updated; list: %s", all)
	}
}
