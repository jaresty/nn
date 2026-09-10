package cmd

import (
	"math"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestRankedByQueriesUsesWeightedReciprocalRanks(t *testing.T) {
	corpus := []*note.Note{
		{ID: "source", Title: "sourceonly"},
		{ID: "intent", Title: "intentonly"},
		{ID: "both", Title: "sourceonly intentonly"},
	}
	prepared := prepareCorpus(corpus, "")
	got := prepared.rankedByQueries(corpus, []rankingQuery{
		{Name: "source", Text: "sourceonly", Weight: 1},
		{Name: "intent", Text: "intentonly", Weight: 2},
	})

	if len(got) != 3 {
		t.Fatalf("ranked results = %d, want 3", len(got))
	}
	if got[0].Note.ID != "both" {
		t.Fatalf("first note = %q, want both", got[0].Note.ID)
	}
	byID := map[string]fusedRanking{}
	for _, result := range got {
		byID[result.Note.ID] = result
	}
	wantIntentOnly := 2 * prepared.rankedByQuery(corpus, "intentonly")["intent"]
	if diff := math.Abs(byID["intent"].Score - wantIntentOnly); diff > 1e-12 {
		t.Fatalf("intent-only score = %.12f, want %.12f", byID["intent"].Score, wantIntentOnly)
	}
	if len(byID["intent"].Provenance) != 1 || byID["intent"].Provenance[0].Channel != "intent" {
		t.Fatalf("intent provenance = %#v", byID["intent"].Provenance)
	}
}

func TestRankedByQueriesOmitsBlankChannelsAndPreservesCandidateTies(t *testing.T) {
	corpus := []*note.Note{
		{ID: "first", Title: "shared"},
		{ID: "second", Title: "shared"},
	}
	prepared := prepareCorpus(corpus, "")
	got := prepared.rankedByQueries(corpus, []rankingQuery{
		{Name: "blank", Text: "  ", Weight: 100},
		{Name: "source", Text: "shared", Weight: 1},
	})
	if len(got) != 2 || got[0].Note.ID != "first" || got[1].Note.ID != "second" {
		t.Fatalf("tie order = %#v", got)
	}
	for _, result := range got {
		if len(result.Provenance) != 1 || result.Provenance[0].Channel != "source" {
			t.Fatalf("provenance = %#v", result.Provenance)
		}
	}
}
