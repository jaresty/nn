package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion E: --search ranks title matches above body-only matches.
func TestListSearchRanksTitleAboveBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	// bodyMatch gets a smaller ID so it comes first in default (filename) order.
	bodyMatch := newTestNoteForCLI("20260101000000-0001", "Network Overview", note.TypeConcept)
	bodyMatch.Body = "This note discusses routing in distributed systems."
	titleMatch := newTestNoteForCLI("20260102000000-0002", "Routing Algorithm", note.TypeConcept)
	titleMatch.Body = "Discusses path selection in networks."
	writeNoteFile(t, nbDir, bodyMatch)
	writeNoteFile(t, nbDir, titleMatch)

	out, err := execute("list", "--search", "routing", "--json")
	if err != nil {
		t.Fatalf("nn list --search routing: %v", err)
	}
	titles := orderedTitles(t, out)
	if len(titles) < 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(titles), titles)
	}
	if titles[0] != "Routing Algorithm" {
		t.Errorf("ranking: first result = %q, want Routing Algorithm (title match should rank first)", titles[0])
	}
}
