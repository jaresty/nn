package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: RankedByQuery returns same scores as BM25RRFPerField with BM25FieldIDF.
func TestRankedByQuery_Correctness(t *testing.T) {
	notes := []*note.Note{
		{ID: "n1", Title: "alpha beta gamma", Body: "delta epsilon", Tags: []string{"zeta"}, Status: note.StatusDraft},
		{ID: "n2", Title: "epsilon zeta", Body: "alpha gamma omega", Tags: []string{"beta"}, Status: note.StatusPermanent},
	}
	inbound := map[string][]string{"n1": {"n2"}}
	query := "alpha gamma"

	got := RankedByQuery(notes, inbound, query, "")

	fidf := note.BM25FieldIDF(notes, inbound)
	want := note.BM25RRFPerField(notes, fidf, query, inbound)

	for id, wantScore := range want {
		if gotScore := got[id]; gotScore != wantScore {
			t.Errorf("RankedByQuery[%q]: got %.6f want %.6f", id, gotScore, wantScore)
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			t.Errorf("RankedByQuery returned extra key %q not in BM25RRFPerField result", id)
		}
	}
}
