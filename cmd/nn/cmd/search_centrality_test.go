package cmd

import (
	"encoding/json"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: a note with backlinks ranks higher than an otherwise identical note
// with zero backlinks, when BM25 content scores are equal.
// Design: hub and orphan have the same title token count (2 tokens each) and
// identical body — so their BM25 dlen is identical before any centrality boost.
// The linker uses an empty annotation so it adds zero tokens to hub's document
// length, keeping BM25 scores equal. Only the centrality multiplier can break the tie.
func TestSearchCentralityBoostBacklinks(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// hub: 2-token title, linked-to by linker with empty annotation.
	hub := newTestNoteForCLI("20260101000000-0100", "Alpha Hub", note.TypeConcept)
	hub.Body = "alpha content here"
	hub.Status = note.StatusDraft

	// orphan: 2-token title, zero backlinks; body identical to hub.
	orphan := newTestNoteForCLI("20260101000000-0101", "Alpha Zap", note.TypeConcept)
	orphan.Body = "alpha content here"
	orphan.Status = note.StatusDraft

	// linker: empty annotation so it adds zero tokens to hub's BM25 document length.
	linker := newTestNoteForCLI("20260101000000-0102", "Linker Note", note.TypeConcept)
	linker.Body = "unrelated"
	linker.Links = []note.Link{
		{TargetID: hub.ID, Annotation: "", Type: "related"},
	}

	writeNoteFile(t, nbDir, hub)
	writeNoteFile(t, nbDir, orphan)
	writeNoteFile(t, nbDir, linker)

	out, err := execute("list", "--search", "alpha content", "--json")
	if err != nil {
		t.Fatalf("nn list --search: %v", err)
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Find hub and orphan in results.
	var hubScore, orphanScore float64
	var hubFound, orphanFound bool
	for _, r := range results {
		title, _ := r["title"].(string)
		score, _ := r["score"].(float64)
		switch title {
		case "Alpha Hub":
			hubScore = score
			hubFound = true
		case "Alpha Zap":
			orphanScore = score
			orphanFound = true
		}
	}
	if !hubFound {
		t.Fatal("hub note not found in search results")
	}
	if !orphanFound {
		t.Fatal("orphan note not found in search results")
	}
	if hubScore <= orphanScore {
		t.Errorf("centrality boost: hub score %v should be > orphan score %v (hub has 1 backlink, orphan has 0)", hubScore, orphanScore)
	}
}
