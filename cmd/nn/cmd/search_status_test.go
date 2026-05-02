package cmd

import (
	"encoding/json"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: search --status filters results to notes with the given status.
func TestSearchStatusFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	permanent := newTestNoteForCLI(note.GenerateID(), "Permanent Concept", note.TypeConcept)
	permanent.Status = note.StatusPermanent
	permanent.Body = "This note is about zettelkasten."

	draft := newTestNoteForCLI(note.GenerateID(), "Draft Idea", note.TypeConcept)
	draft.Status = note.StatusDraft
	draft.Body = "This note is about zettelkasten too."

	writeNoteFile(t, nbDir, permanent)
	writeNoteFile(t, nbDir, draft)

	out, err := execute("search", "zettelkasten", "--status", "permanent", "--json")
	if err != nil {
		t.Fatalf("nn search: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %s", len(results), out)
	}
	if results[0]["id"] != permanent.ID {
		t.Errorf("expected permanent note %v, got %v", permanent.ID, results[0]["id"])
	}
}

// Assertion: search without --status returns all matching notes regardless of status.
func TestSearchStatusFilterAbsent(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	permanent := newTestNoteForCLI(note.GenerateID(), "Permanent Concept", note.TypeConcept)
	permanent.Status = note.StatusPermanent
	permanent.Body = "This note is about zettelkasten."

	draft := newTestNoteForCLI(note.GenerateID(), "Draft Idea", note.TypeConcept)
	draft.Status = note.StatusDraft
	draft.Body = "This note is about zettelkasten too."

	writeNoteFile(t, nbDir, permanent)
	writeNoteFile(t, nbDir, draft)

	out, err := execute("search", "zettelkasten", "--json")
	if err != nil {
		t.Fatalf("nn search: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results without status filter, got %d: %s", len(results), out)
	}
}
