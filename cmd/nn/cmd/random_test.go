package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn random returns a note from the corpus.
func TestRandomReturnsNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("random")
	if err != nil {
		t.Fatalf("nn random: %v", err)
	}
	if !strings.Contains(out, "Some Note") {
		t.Errorf("expected note title in output, got:\n%s", out)
	}
}

// Assertion: nn random --status filters correctly.
func TestRandomStatusFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	draft := newTestNoteForCLI(note.GenerateID(), "Draft Note", note.TypeConcept)
	draft.Status = note.StatusDraft
	perm := newTestNoteForCLI(note.GenerateID(), "Permanent Note", note.TypeConcept)
	perm.Status = note.StatusPermanent
	writeNoteFile(t, nbDir, draft)
	writeNoteFile(t, nbDir, perm)

	// Run many times to ensure only permanent notes are returned.
	for i := 0; i < 20; i++ {
		out, err := execute("random", "--status", "permanent")
		if err != nil {
			t.Fatalf("nn random --status: %v", err)
		}
		if strings.Contains(out, "Draft Note") {
			t.Errorf("draft note appeared despite --status permanent filter:\n%s", out)
		}
		if !strings.Contains(out, "Permanent Note") {
			t.Errorf("permanent note missing from --status permanent output:\n%s", out)
		}
	}
}

// Assertion: nn random --tag filters correctly.
func TestRandomTagFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	tagged := newTestNoteForCLI(note.GenerateID(), "Tagged Note", note.TypeConcept)
	tagged.Tags = []string{"mytag"}
	untagged := newTestNoteForCLI(note.GenerateID(), "Untagged Note", note.TypeConcept)
	writeNoteFile(t, nbDir, tagged)
	writeNoteFile(t, nbDir, untagged)

	for i := 0; i < 20; i++ {
		out, err := execute("random", "--tag", "mytag")
		if err != nil {
			t.Fatalf("nn random --tag: %v", err)
		}
		if strings.Contains(out, "Untagged Note") {
			t.Errorf("untagged note appeared despite --tag filter:\n%s", out)
		}
		if !strings.Contains(out, "Tagged Note") {
			t.Errorf("tagged note missing from --tag output:\n%s", out)
		}
	}
}

// Assertion: nn random --json outputs valid JSON with expected fields.
func TestRandomJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "JSON Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("random", "--json")
	if err != nil {
		t.Fatalf("nn random --json: %v", err)
	}
	var result noteJSON
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("nn random --json output is not valid JSON: %v\n%s", err, out)
	}
	if result.ID == "" {
		t.Errorf("JSON missing id field:\n%s", out)
	}
	if result.Title != "JSON Note" {
		t.Errorf("expected title 'JSON Note', got %q", result.Title)
	}
}
