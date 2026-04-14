package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion C: nn links <id> text output includes linked note title and annotation.
func TestLinksText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	ann := "explains the foundational concept"
	from.Links = []note.Link{{TargetID: to.ID, Annotation: ann}}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)

	out, err := execute("links", from.ID)
	if err != nil {
		t.Fatalf("nn links %s: %v", from.ID, err)
	}
	if !strings.Contains(out, to.ID) {
		t.Errorf("links text missing target ID %q:\n%s", to.ID, out)
	}
	if !strings.Contains(out, to.Title) {
		t.Errorf("links text missing target title %q:\n%s", to.Title, out)
	}
	if !strings.Contains(out, ann) {
		t.Errorf("links text missing annotation %q:\n%s", ann, out)
	}
}

// --type filter: only returns links matching the given type.
func TestLinksTypeFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	to1 := newTestNoteForCLI(note.GenerateID(), "Refines Target", note.TypeConcept)
	to2 := newTestNoteForCLI(note.GenerateID(), "Contradicts Target", note.TypeConcept)
	from.Links = []note.Link{
		{TargetID: to1.ID, Annotation: "narrows scope", Type: "refines"},
		{TargetID: to2.ID, Annotation: "opposes claim", Type: "contradicts"},
	}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to1)
	writeNoteFile(t, nbDir, to2)

	out, err := execute("links", from.ID, "--type", "refines")
	if err != nil {
		t.Fatalf("nn links --type refines: %v", err)
	}
	if !strings.Contains(out, "Refines Target") {
		t.Errorf("missing refines link:\n%s", out)
	}
	if strings.Contains(out, "Contradicts Target") {
		t.Errorf("contradicts link should be filtered out:\n%s", out)
	}
}

// Assertion D: nn links <id> --json produces valid JSON array with id/title/annotation.
func TestLinksJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	ann := "explains the foundational concept"
	from.Links = []note.Link{{TargetID: to.ID, Annotation: ann}}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)

	out, err := execute("links", from.ID, "--json")
	if err != nil {
		t.Fatalf("nn links %s --json: %v", from.ID, err)
	}
	var result []struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Annotation string `json:"annotation"`
	}
	mustJSON(t, out, &result)
	if len(result) != 1 {
		t.Fatalf("links JSON: got %d items, want 1", len(result))
	}
	if result[0].ID != to.ID {
		t.Errorf("id: got %q, want %q", result[0].ID, to.ID)
	}
	if result[0].Title != to.Title {
		t.Errorf("title: got %q, want %q", result[0].Title, to.Title)
	}
	if result[0].Annotation != ann {
		t.Errorf("annotation: got %q, want %q", result[0].Annotation, ann)
	}
}
