package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// noteWithBodyAndModified returns a test note with the given body and timestamps.
func noteWithBodyAndModified(id, title, body string, mod time.Time) *note.Note {
	n := newTestNoteForCLI(id, title, note.TypeConcept)
	n.Body = body
	n.Modified = mod
	n.Created = mod
	return n
}

// TestListBoostRecentRankingOrder verifies that --boost-recent reorders search
// results so a recently-modified note with the same query terms ranks above an
// older note with identical BM25 relevance.
func TestListBoostRecentRankingOrder(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	body := "quantum entanglement physics"
	// Both notes share the same Created time so the default created-desc sort
	// does not break ties; only Modified differs, which --boost-recent uses.
	sharedCreated := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	oldMod := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	recentMod := time.Now()

	// Old gets the lower ID so it sorts first in backend list order (alphabetical by ID).
	old := noteWithBodyAndModified("20240601000000-0001", "Old Note", body, oldMod)
	old.Created = sharedCreated
	recent := noteWithBodyAndModified("20240601000000-0002", "Recent Note", body, recentMod)
	recent.Created = sharedCreated

	// Write old first: without boost, stable BM25 sort + created-desc tiebreak
	// produces a tie (same Created), leaving old first in list order.
	writeNoteFile(t, nbDir, old)
	writeNoteFile(t, nbDir, recent)

	out, err := execute("list", "--search", "quantum", "--boost-recent", "--json")
	if err != nil {
		t.Fatalf("nn list --search quantum --boost-recent: %v", err)
	}

	var results []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if results[0].Title != "Recent Note" {
		t.Errorf("--boost-recent: first result = %q, want Recent Note", results[0].Title)
	}
}

// TestListBoostRecentNoEffectWithoutSearch verifies that --boost-recent does
// not affect listing when --search is absent (flag is accepted without error).
func TestListBoostRecentNoEffectWithoutSearch(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI("20260101000000-0001", "Alpha", note.TypeConcept))

	_, err := execute("list", "--boost-recent", "--json")
	if err != nil {
		t.Errorf("nn list --boost-recent without --search should not error: %v", err)
	}
}
