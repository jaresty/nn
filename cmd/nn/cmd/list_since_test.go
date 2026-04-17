package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: --since filters out notes modified before the date.
func TestListSinceFiltersOldNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	old := newTestNoteForCLI(note.GenerateID(), "Old Note", note.TypeConcept)
	old.Modified = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := newTestNoteForCLI(note.GenerateID(), "Recent Note", note.TypeConcept)
	recent.Modified = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeNoteFile(t, nbDir, old)
	writeNoteFile(t, nbDir, recent)

	out, err := execute("list", "--since", "2025-01-01")
	if err != nil {
		t.Fatalf("nn list --since: %v", err)
	}
	if strings.Contains(out, "Old Note") {
		t.Errorf("old note should be filtered out by --since:\n%s", out)
	}
	if !strings.Contains(out, "Recent Note") {
		t.Errorf("recent note should be in --since output:\n%s", out)
	}
}

// Assertion: --since with future date returns empty (no error).
func TestListSinceFutureEmpty(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Any Note", note.TypeConcept)
	n.Modified = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--since", "2099-01-01", "--json")
	if err != nil {
		t.Fatalf("nn list --since future should not error: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results for future --since, got %d", len(results))
	}
}
