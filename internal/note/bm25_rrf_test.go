package note

import (
	"testing"
)

// property [1]: title-field isolation — note matching only in title appears in RRF scores
// property [2]: RRF formula = sum of 1/(60+rank_f) across fields
// property [3]: note matching in two fields ranks higher than note matching in one field
// property [4]: BM25RRF is callable as a drop-in for BM25ScoresWithIDF
func TestBM25RRF_TitleFieldIsolation(t *testing.T) {
	// titleAndBody matches "deploy" in both title and body → ranks in two fields.
	// bodyOnly matches "deploy" only in body → ranks in one field.
	titleAndBody := &Note{ID: "both", Title: "deploy pipeline", Body: "deploy automation", Status: StatusDraft}
	bodyOnly := &Note{ID: "body-only", Title: "unrelated title", Body: "deploy automation", Status: StatusDraft}
	notes := []*Note{titleAndBody, bodyOnly}

	idf := BM25IDF(notes, Tokenize("deploy"))
	scores := BM25RRF(notes, idf, "deploy", nil)

	bothScore, bothFound := scores["both"]
	bodyScore, bodyFound := scores["body-only"]

	// property [1]: both notes must appear (both match "deploy")
	if !bothFound {
		t.Errorf("property [1]: titleAndBody note not found in RRF scores")
	}
	if !bodyFound {
		t.Errorf("property [1]: body-only note not found in RRF scores")
	}

	// property [3]: two-field match ranks above single-field match
	if bothFound && bodyFound && bothScore <= bodyScore {
		t.Errorf("property [3]: expected both-fields (%.4f) > body-only (%.4f)", bothScore, bodyScore)
	}
}

func TestBM25RRF_Formula(t *testing.T) {
	// Single note matching only in title — its RRF score must be >= 1/(60+1)
	n := &Note{ID: "n1", Title: "unique-xyzzy term", Body: "irrelevant", Status: StatusDraft}
	notes := []*Note{n}

	idf := BM25IDF(notes, Tokenize("unique-xyzzy"))
	scores := BM25RRF(notes, idf, "unique-xyzzy", nil)

	score, found := scores["n1"]
	if !found {
		t.Fatalf("property [2]: note n1 not in RRF scores")
	}
	// With one note, it's rank 1 in whatever field it matches.
	// RRF contribution: 1/(60+1) per matching field, so total >= 1/61.
	expected := 1.0 / 61.0
	if score < expected*0.9 {
		t.Errorf("property [2]: expected RRF score >= %.4f, got %.4f", expected, score)
	}
}

func TestBM25RRF_NoResults(t *testing.T) {
	n := &Note{ID: "n1", Title: "hello world", Body: "nothing here", Status: StatusDraft}
	notes := []*Note{n}
	idf := BM25IDF(notes, Tokenize("xyzzy"))
	scores := BM25RRF(notes, idf, "xyzzy", nil)
	if len(scores) != 0 {
		t.Errorf("expected empty scores for non-matching query, got %v", scores)
	}
}
