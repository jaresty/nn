package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: --boost-recent with zero search results must not panic
func TestListBoostRecentZeroResults(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Unrelated Note", note.TypeConcept)
	n.Body = "nothing matching here"
	writeNoteFile(t, nbDir, n)

	// Query that matches nothing; --boost-recent should not panic on nil searchScores
	_, err := execute("list", "--search", "xyzzy-nonexistent-term", "--boost-recent", "--json")
	if err != nil {
		t.Fatalf("nn list --boost-recent with zero results: %v", err)
	}
}
