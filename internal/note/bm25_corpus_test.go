package note

import (
	"strings"
	"testing"
)

func TestBM25RRFPerFieldForCorpus_UsesCorpusAverageLength(t *testing.T) {
	short := &Note{ID: "short", Body: "query"}
	dense := &Note{ID: "dense", Body: "query query " + strings.Repeat("filler ", 8)}
	background := &Note{ID: "background", Body: strings.Repeat("filler ", 200)}
	corpus := []*Note{short, dense, background}
	candidates := []*Note{short, dense}

	scores := BM25RRFPerFieldForCorpus(corpus, candidates, BM25FieldIDF(corpus, nil), "query", nil, nil)
	if scores[dense.ID] <= scores[short.ID] {
		t.Fatalf("full-corpus average length should rank dense first: dense=%f short=%f", scores[dense.ID], scores[short.ID])
	}

	candidateOnly := BM25RRFPerFieldForCorpus(candidates, candidates, BM25FieldIDF(candidates, nil), "query", nil, nil)
	if candidateOnly[short.ID] <= candidateOnly[dense.ID] {
		t.Fatalf("fixture must distinguish candidate-only statistics: short=%f dense=%f", candidateOnly[short.ID], candidateOnly[dense.ID])
	}
}

// Equal-scoring documents retain the existing candidate-sequence-dependent
// permutation. This compatibility test intentionally documents, rather than
// endorses, that behavior until tie semantics are redesigned separately.
func TestBM25RRFPerField_PreservesExistingTiePermutation(t *testing.T) {
	a := &Note{ID: "a", Body: "query"}
	b := &Note{ID: "b", Body: "query"}
	c := &Note{ID: "c", Body: "query query"}
	corpus := []*Note{a, b, c}

	scores := BM25RRFPerFieldForCorpus(corpus, corpus, BM25FieldIDF(corpus, nil), "query", nil, nil)
	if !(scores[c.ID] > scores[b.ID] && scores[b.ID] > scores[a.ID]) {
		t.Fatalf("compatibility permutation changed: c=%f b=%f a=%f", scores[c.ID], scores[b.ID], scores[a.ID])
	}
}
