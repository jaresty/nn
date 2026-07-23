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

// TestBM25StemmedQueryMatchesStemmedDocument: querying "atomicity" should match a note containing "atomic".
func TestBM25StemmedQueryMatchesStemmedDocument(t *testing.T) {
	notes := []*Note{
		makeNote("a", "atomic token design", "The atomic constraint governs one change per step."),
		makeNote("b", "unrelated note", "Something about potatoes."),
	}
	scores := BM25Scores(notes, "atomicity", nil)
	if scores["a"] == 0 {
		t.Errorf("expected nonzero score for note with stem 'atomic' when querying 'atomicity'; got 0")
	}
	if scores["b"] != 0 {
		t.Errorf("expected zero score for unrelated note; got %f", scores["b"])
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

// TestBM25IDF: rare term has higher IDF than common term.
func TestBM25IDF(t *testing.T) {
	notes := []*Note{
		makeNote("a", "cache invalidation strategy", "cache is key"),
		makeNote("b", "unrelated", "something else"),
		makeNote("c", "another cache note", "cache again"),
	}
	idf := BM25IDF(notes, tokenize("cache unrelated"))
	if idf["cach"] >= idf["unrel"] {
		t.Errorf("rare term 'unrel' should have higher IDF than common 'cach': cach=%f unrel=%f", idf["cach"], idf["unrel"])
	}
}

// TestBM25ScoresWithIDF: pre-computed IDF produces same scores as BM25Scores when corpus==candidates.
func TestBM25ScoresWithIDF(t *testing.T) {
	notes := []*Note{
		makeNote("a", "Caching Strategy", "Discusses cache invalidation."),
		makeNote("b", "Unrelated Topic", "Something about potatoes."),
	}
	query := "cache invalidation"
	idf := BM25IDF(notes, tokenize(query))
	withIDF := BM25ScoresWithIDF(notes, idf, query, nil)
	direct := BM25Scores(notes, query, nil)
	for _, n := range notes {
		if withIDF[n.ID] != direct[n.ID] {
			t.Errorf("BM25ScoresWithIDF(%s) = %f, want %f", n.ID, withIDF[n.ID], direct[n.ID])
		}
	}
}

// TestBM25LinkTypeWeightedInbound: governs annotation boosts more than questions annotation.
func TestBM25LinkTypeWeightedInbound(t *testing.T) {
	governed := makeNote("governed", "Cache Note", "Discusses cache.")
	questioned := makeNote("questioned", "Cache Note", "Discusses cache.")
	notes := []*Note{governed, questioned}
	inbound := map[string][]TypedAnnotation{
		"governed":   {{Text: "cache invalidation strategy", LinkType: "governs"}},
		"questioned": {{Text: "cache invalidation strategy", LinkType: "questions"}},
	}
	scores := BM25ScoresTyped(notes, "cache invalidation", inbound)
	if scores["governed"] <= scores["questioned"] {
		t.Errorf("governs link should boost more than questions: governed=%f questioned=%f", scores["governed"], scores["questioned"])
	}
}

// TestBM25ScorePropagation: neighbor of high-scoring note gets nonzero score via propagation.
func TestBM25ScorePropagation(t *testing.T) {
	direct := makeNote("direct", "Cache Invalidation Strategy", "cache invalidation is important")
	neighbor := makeNote("neighbor", "Linked Note", "completely unrelated content about potatoes")
	notes := []*Note{direct, neighbor}
	links := map[string][]string{"direct": {"neighbor"}}
	scores := BM25ScoresWithPropagation(notes, "cache invalidation", nil, links)
	if scores["direct"] == 0 {
		t.Errorf("direct match should score > 0: got %f", scores["direct"])
	}
	if scores["neighbor"] == 0 {
		t.Errorf("neighbor of high-scoring note should score > 0 via propagation: got %f", scores["neighbor"])
	}
}
