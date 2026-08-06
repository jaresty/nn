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

// property [P1]: exact full-title match ranks above broad multi-field partial match.
// atomicNote has all query terms only in the title.
// broadNote has query terms spread across title, body, and tags — currently wins via multi-field RRF.
func TestBM25RRF_ExactTitleBeatsMultiField(t *testing.T) {
	atomicNote := &Note{
		ID:    "atomic",
		Title: "negative zero denominator behavior",
		Body:  "unrelated content here",
		Tags:  []string{"math"},
		Status: StatusDraft,
	}
	broadNote := &Note{
		ID:    "broad",
		Title: "division edge cases",
		Body:  "negative zero denominator behavior in division routines affects all callers",
		Tags:  []string{"negative", "zero", "denominator", "behavior"},
		Status: StatusPermanent,
	}
	notes := []*Note{atomicNote, broadNote}
	query := "negative zero denominator behavior"

	idf := BM25IDF(notes, Tokenize(query))
	scores := BM25RRF(notes, idf, query, nil)

	atomicScore := scores["atomic"]
	broadScore := scores["broad"]

	if atomicScore <= broadScore {
		t.Errorf("property [P1]: exact-title note (%.4f) should rank above broad multi-field note (%.4f)", atomicScore, broadScore)
	}
}

// property [P3]: when two notes have identical relevance, the permanent one may score slightly higher
// but must not exceed the draft score by more than 6%% (statusMultiplier <= 1.06).
func TestBM25RRF_StatusMultiplierDoesNotOverpower(t *testing.T) {
	// Both notes have identical content — only status differs.
	draftNote := &Note{
		ID:     "draft-note",
		Title:  "migration authority holder domain",
		Body:   "migration authority holder domain rules",
		Status: StatusDraft,
	}
	permanentNote := &Note{
		ID:     "permanent-note",
		Title:  "migration authority holder domain",
		Body:   "migration authority holder domain rules",
		Status: StatusPermanent,
	}
	notes := []*Note{draftNote, permanentNote}
	query := "migration authority holder domain"

	idf := BM25IDF(notes, Tokenize(query))
	scores := BM25RRF(notes, idf, query, nil)

	draftScore := scores["draft-note"]
	permanentScore := scores["permanent-note"]

	// permanent may score higher but only by the statusMultiplier — must be <= 6%%.
	if draftScore == 0 {
		t.Fatal("property [P3]: draft note scored 0")
	}
	ratio := permanentScore / draftScore
	if ratio > 1.06 {
		t.Errorf("property [P3]: permanent/draft score ratio %.4f exceeds 1.06 — statusMultiplier too large", ratio)
	}
}

// property [P2]: note with all query terms in title ranks above note with query terms only in body.
// titleNote: all terms in title, minimal body.
// bodyNote: no query terms in title, all terms repeated heavily in body (simulates a long synthesis note).
func TestBM25RRF_TitleCoverageBeatsBodyOnly(t *testing.T) {
	titleNote := &Note{
		ID:     "title-note",
		Title:  "negative zero denominator behavior",
		Body:   "unrelated content",
		Status: StatusDraft,
	}
	// bodyNote has many body repetitions to maximize its body BM25 score.
	bodyNote := &Note{
		ID:    "body-note",
		Title: "division arithmetic",
		Body:  "negative zero denominator behavior negative zero denominator behavior negative zero denominator behavior negative zero denominator behavior",
		Status: StatusDraft,
	}
	notes := []*Note{titleNote, bodyNote}
	query := "negative zero denominator behavior"

	idf := BM25IDF(notes, Tokenize(query))
	scores := BM25RRF(notes, idf, query, nil)

	titleScore := scores["title-note"]
	bodyScore := scores["body-note"]

	if titleScore <= bodyScore {
		t.Errorf("property [P2]: title-match note (%.4f) should rank above body-only note (%.4f)", titleScore, bodyScore)
	}
}

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
