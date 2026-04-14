package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// orderedTitles parses JSON list output and returns note titles in output order.
func orderedTitles(t *testing.T, out string) []string {
	t.Helper()
	var result []struct {
		Title string `json:"title"`
	}
	mustJSON(t, out, &result)
	titles := make([]string, len(result))
	for i, r := range result {
		titles[i] = r.Title
	}
	return titles
}

func noteWithModified(id, title string, mod time.Time) *note.Note {
	n := newTestNoteForCLI(id, title, note.TypeConcept)
	n.Modified = mod
	n.Created = mod
	return n
}

// Assertion A: --sort modified returns notes most-recently-modified first.
func TestListSortModified(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	oldest := noteWithModified("20260101000000-0001", "Oldest", base)
	middle := noteWithModified("20260102000000-0002", "Middle", base.Add(24*time.Hour))
	newest := noteWithModified("20260103000000-0003", "Newest", base.Add(48*time.Hour))
	// Write in ID order (oldest first) so default order is oldest→newest.
	writeNoteFile(t, nbDir, oldest)
	writeNoteFile(t, nbDir, middle)
	writeNoteFile(t, nbDir, newest)

	out, err := execute("list", "--sort", "modified", "--json")
	if err != nil {
		t.Fatalf("nn list --sort modified: %v", err)
	}
	titles := orderedTitles(t, out)
	if len(titles) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(titles))
	}
	if titles[0] != "Newest" {
		t.Errorf("--sort modified: first = %q, want Newest", titles[0])
	}
	if titles[2] != "Oldest" {
		t.Errorf("--sort modified: last = %q, want Oldest", titles[2])
	}
}

// Assertion B: --sort title returns notes alphabetically.
func TestListSortTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	// Write in reverse alpha order so default order != alpha.
	writeNoteFile(t, nbDir, newTestNoteForCLI("20260103000000-0003", "Zebra", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI("20260102000000-0002", "Mango", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI("20260101000000-0001", "Apple", note.TypeConcept))

	out, err := execute("list", "--sort", "title", "--json")
	if err != nil {
		t.Fatalf("nn list --sort title: %v", err)
	}

	var result []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(result))
	}
	if result[0].Title != "Apple" {
		t.Errorf("--sort title: first = %q, want Apple", result[0].Title)
	}
	if result[2].Title != "Zebra" {
		t.Errorf("--sort title: last = %q, want Zebra", result[2].Title)
	}
}
