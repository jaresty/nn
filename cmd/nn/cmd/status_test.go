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

// Assertion A: status text output includes orphan ID and title inline.
func TestStatusOrphanNamesInline(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	orphan := newTestNoteForCLI(note.GenerateID(), "The Lost Note", note.TypeConcept)
	writeNoteFile(t, nbDir, orphan)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(out, orphan.ID) {
		t.Errorf("status text output missing orphan ID %q:\n%s", orphan.ID, out)
	}
	if !strings.Contains(out, orphan.Title) {
		t.Errorf("status text output missing orphan title %q:\n%s", orphan.Title, out)
	}
}

// Assertion B: status --json produces valid JSON with orphans as array of objects.
func TestStatusJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	orphan := newTestNoteForCLI(note.GenerateID(), "JSON Orphan", note.TypeConcept)
	writeNoteFile(t, nbDir, orphan)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result struct {
		Total   int `json:"total"`
		Orphans []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"orphans"`
		Drafts      int              `json:"drafts"`
		BrokenLinks []map[string]any `json:"broken_links"`
	}
	mustJSON(t, out, &result)
	if result.Total != 1 {
		t.Errorf("total: got %d, want 1", result.Total)
	}
	if len(result.Orphans) != 1 {
		t.Errorf("orphans: got %d, want 1", len(result.Orphans))
	} else {
		if result.Orphans[0].ID != orphan.ID {
			t.Errorf("orphan ID: got %q, want %q", result.Orphans[0].ID, orphan.ID)
		}
		if result.Orphans[0].Title != orphan.Title {
			t.Errorf("orphan title: got %q, want %q", result.Orphans[0].Title, orphan.Title)
		}
	}
}
