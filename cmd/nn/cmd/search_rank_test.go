package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: --search with multi-term query ranks notes by BM25 score (more term
// overlap = higher rank), not by a fixed per-field integer.
func TestListSearchBM25MultiTermRanking(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	// weak: contains only "cache" — one term overlap with query.
	weak := newTestNoteForCLI("20260101000000-0010", "Cache Basics", note.TypeConcept)
	weak.Body = "Describes a cache layer."
	// strong: contains "cache", "invalidation", "distributed" — three term overlaps.
	strong := newTestNoteForCLI("20260101000000-0011", "Cache Invalidation Strategies", note.TypeConcept)
	strong.Body = "Discusses cache invalidation in distributed systems."
	// noise: no overlap with query terms.
	noise := newTestNoteForCLI("20260101000000-0012", "Unrelated Topic", note.TypeConcept)
	noise.Body = "Something about potatoes."
	writeNoteFile(t, nbDir, weak)
	writeNoteFile(t, nbDir, strong)
	writeNoteFile(t, nbDir, noise)

	out, err := execute("list", "--search", "cache invalidation distributed", "--json")
	if err != nil {
		t.Fatalf("nn list --search multi-term: %v", err)
	}
	titles := orderedTitles(t, out)
	if len(titles) < 2 {
		t.Fatalf("expected ≥2 results, got %d: %v", len(titles), titles)
	}
	if titles[0] != "Cache Invalidation Strategies" {
		t.Errorf("multi-term ranking: first = %q, want Cache Invalidation Strategies", titles[0])
	}
	for _, title := range titles {
		if title == "Unrelated Topic" {
			t.Errorf("zero-score note should be excluded, but got it in results: %v", titles)
		}
	}
}

// Assertion: --search composed with --type still filters correctly (BM25 runs
// over the already-type-filtered set, not the full corpus).
func TestListSearchBM25WithTypeFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	match := newTestNoteForCLI("20260101000000-0020", "Routing Concept", note.TypeConcept)
	match.Body = "Discusses routing algorithms."
	wrongType := newTestNoteForCLI("20260101000000-0021", "Routing Argument", note.TypeArgument)
	wrongType.Body = "Argues about routing protocols."
	writeNoteFile(t, nbDir, match)
	writeNoteFile(t, nbDir, wrongType)

	out, err := execute("list", "--search", "routing", "--type", "concept", "--json")
	if err != nil {
		t.Fatalf("nn list --search --type: %v", err)
	}
	titles := orderedTitles(t, out)
	if len(titles) != 1 || titles[0] != "Routing Concept" {
		t.Errorf("--search + --type: got %v, want [Routing Concept]", titles)
	}
}

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
