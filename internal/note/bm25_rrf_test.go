package note

import (
	"math"
	"testing"
)

// property [F1]: BM25RRFPerField returns non-empty results when query terms appear only in note titles.
// With the shared-IDF bug, a title-only match can produce IDF=0 for the field and score 0.
func TestBM25RRFPerField_TitleOnlyMatch(t *testing.T) {
	titleOnly := &Note{ID: "title-only", Title: "xyloquartz resonance", Body: "unrelated body content here", Status: StatusDraft}
	distractor := &Note{ID: "distractor", Title: "something else entirely", Body: "more unrelated content", Status: StatusDraft}
	notes := []*Note{titleOnly, distractor}

	fieldIDF := BM25FieldIDF(notes, nil)
	scores := BM25RRFPerField(notes, fieldIDF, "xyloquartz resonance", nil)

	if len(scores) == 0 {
		t.Fatal("property [F1]: BM25RRFPerField returned no results for title-only match")
	}
	if scores["title-only"] == 0 {
		t.Errorf("property [F1]: title-only note scored 0; scores=%v", scores)
	}
}

// property [F2]: note matching query only in title ranks above note matching only in body,
// when per-field IDF is used. Title IDF is computed over the title corpus only.
func TestBM25RRFPerField_TitleBeatsBody(t *testing.T) {
	titleNote := &Note{ID: "title-note", Title: "xyloquartz resonance phenomenon", Body: "unrelated", Status: StatusDraft}
	bodyNote := &Note{ID: "body-note", Title: "general overview", Body: "xyloquartz resonance phenomenon xyloquartz resonance phenomenon xyloquartz resonance phenomenon", Status: StatusDraft}
	notes := []*Note{titleNote, bodyNote}

	fieldIDF := BM25FieldIDF(notes, nil)
	scores := BM25RRFPerField(notes, fieldIDF, "xyloquartz resonance phenomenon", nil)

	titleScore := scores["title-note"]
	bodyScore := scores["body-note"]
	if titleScore <= bodyScore {
		t.Errorf("property [F2]: title-match (%.4f) should beat body-only (%.4f) with per-field IDF", titleScore, bodyScore)
	}
}

// property [2]: BM25RRFPerField score for a single note equals the raw weighted RRF sum
// with no status multiplier. For a one-note corpus matching query in title only (rank 0),
// the expected score is titleWeight/(60+1) = 5/61. If statusMultiplier were applied
// to a permanent note, the result would be 5/61 * 1.05.
func TestBM25RRFPerField_NoStatusMultiplier(t *testing.T) {
	n := &Note{
		ID: "n1", Title: "xyloquartz resonance phenomenon",
		Body: "unrelated body here", Status: StatusPermanent,
	}
	notes := []*Note{n}
	fidf := BM25FieldIDF(notes, nil)
	scores := BM25RRFPerField(notes, fidf, "xyloquartz resonance", nil)

	got := scores["n1"]
	if got == 0 {
		t.Fatal("property [2]: note scored 0")
	}
	// With statusMultiplier(permanent)=1.05, score = 5/61 * 1.05 ≈ 0.08607.
	// Without multiplier, score = 5/61 ≈ 0.08197.
	// Both title terms match → title rank 0 → RRF = 5/(60+1).
	// (Body, tags, inbound contribute 0 since no query terms there.)
	withoutMultiplier := 5.0 / 61.0
	withMultiplier := withoutMultiplier * 1.05
	const eps = 0.0001
	if math.Abs(got-withMultiplier) < eps {
		t.Errorf("property [2]: score %.6f matches statusMultiplier=1.05 value %.6f — multiplier still applied", got, withMultiplier)
	}
	if math.Abs(got-withoutMultiplier) > eps {
		t.Errorf("property [2]: score %.6f does not match expected raw RRF value %.6f (eps=%.4f)", got, withoutMultiplier, eps)
	}
}
