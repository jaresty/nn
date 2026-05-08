package note

import (
	"testing"
)

func makeNote(id, title, body string) *Note {
	return &Note{ID: id, Title: title, Body: body}
}

// AC2: nil inbound map produces same scores as no-inbound call.
func TestBM25ScoresNilInboundIsNoop(t *testing.T) {
	notes := []*Note{
		makeNote("a", "Caching Strategy", "Discusses cache invalidation."),
		makeNote("b", "Unrelated Topic", "Something about potatoes."),
	}
	withNil := BM25Scores(notes, "cache", nil)
	if withNil["a"] == 0 {
		t.Errorf("expected nonzero score for 'a' with nil inbound, got 0")
	}
	if withNil["b"] != 0 {
		t.Errorf("expected zero score for 'b' with nil inbound, got %f", withNil["b"])
	}
}

// AC1/AC5: inbound annotation tokens boost the scored note.
func TestBM25ScoresInboundAnnotationBoostsScore(t *testing.T) {
	// "weak" has "cache" in body only (low score).
	// "strong" has same body but also gets inbound annotation containing "cache invalidation".
	weak := makeNote("weak", "Some Note", "Discusses cache.")
	strong := makeNote("strong", "Some Note", "Discusses cache.")
	notes := []*Note{weak, strong}

	inbound := map[string][]string{
		"strong": {"foundational cache invalidation strategy"},
	}

	scores := BM25Scores(notes, "cache invalidation", inbound)
	if scores["strong"] <= scores["weak"] {
		t.Errorf("inbound annotation should boost strong above weak: strong=%f weak=%f",
			scores["strong"], scores["weak"])
	}
}

// TestBM25TagWeightNonzero: a note matched only by tag term scores nonzero.
func TestBM25TagWeightNonzero(t *testing.T) {
	tagged := &Note{ID: "a", Title: "Some Note", Body: "unrelated content", Tags: []string{"caching"}}
	untagged := &Note{ID: "b", Title: "Other Note", Body: "unrelated content", Tags: []string{}}
	notes := []*Note{tagged, untagged}
	scores := BM25Scores(notes, "caching", nil)
	if scores["a"] == 0 {
		t.Errorf("expected nonzero score for tag-matched note; got 0")
	}
	if scores["b"] != 0 {
		t.Errorf("expected zero score for note with no tag match; got %f", scores["b"])
	}
}

// TestBM25TagWeightExceedsTitleWeight: tag match at tagWeight boosts score meaningfully.
// A tag-only match should score higher than a single body-token match.
func TestBM25TagWeightBoostsOverBody(t *testing.T) {
	tagOnly := &Note{ID: "tag", Title: "Note A", Body: "irrelevant", Tags: []string{"caching"}}
	bodyOnce := &Note{ID: "body", Title: "Note B", Body: "caching is discussed here", Tags: []string{}}
	notes := []*Note{tagOnly, bodyOnce}
	scores := BM25Scores(notes, "caching", nil)
	if scores["tag"] <= scores["body"] {
		t.Errorf("expected tag-matched note to score higher than single body-token match; tag=%f body=%f", scores["tag"], scores["body"])
	}
}
