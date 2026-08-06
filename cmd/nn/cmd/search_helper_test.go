package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: the shared ranker uses the full corpus while scoring only candidates.
func TestRankedByQuery_UsesCorpusAndCandidateSubset(t *testing.T) {
	corpus := []*note.Note{
		{ID: "candidate", Title: "alpha"},
		{ID: "excluded", Title: "alpha"},
		{ID: "background", Title: "omega"},
	}
	candidates := []*note.Note{corpus[0]}

	scores := RankedByQuery(corpus, candidates, "alpha", "")

	if scores["candidate"] <= 0 {
		t.Fatal("candidate should have a positive score")
	}
	if _, ok := scores["excluded"]; ok {
		t.Fatal("excluded corpus note must not be scored")
	}
}

// property [3]: outbound annotations retrieve their source note, while unknown queries stay empty.
func TestRankedByQuery_OutboundOnlyAndUnknownEligibility(t *testing.T) {
	corpus := []*note.Note{
		{ID: "source", Links: []note.Link{{TargetID: "target", Annotation: "quasar marmalade"}}},
		{ID: "target"},
		{ID: "other", Title: "ordinary note"},
	}

	scores := RankedByQuery(corpus, corpus, "quasar", "")
	if scores["source"] <= 0 {
		t.Fatal("outbound-only source note should have a positive score")
	}
	if scores["target"] > 0 {
		t.Fatal("inbound annotation must not contribute when inbound weight is zero")
	}
	if got := RankedByQuery(corpus, corpus, "xylophone-unknown", ""); len(got) != 0 {
		t.Fatalf("unknown query returned %d scores, want 0", len(got))
	}
}

func TestComputeMatchReason_OutboundAnnotation(t *testing.T) {
	n := &note.Note{
		ID:    "source",
		Title: "Quasar source",
		Links: []note.Link{{TargetID: "target", Annotation: "marmalade evidence"}},
	}
	if got := computeMatchReason(n, "marmalade"); got != "outbound" {
		t.Fatalf("outbound-only reason = %q, want outbound", got)
	}
	if got := computeMatchReason(n, "quasar marmalade"); got != "title, outbound" {
		t.Fatalf("mixed reason = %q, want title, outbound", got)
	}
}
