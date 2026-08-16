package note

import (
	"fmt"
	"testing"
)

// scorerFixture builds a corpus with varied bodies, tags, and link annotations
// so all four BM25 fields (title, body, tags, inbound/outbound) contribute.
func scorerFixture() (corpus []*Note, inbound, outbound map[string][]string) {
	corpus = []*Note{
		{ID: "a", Title: "auth token", Body: "handleAuth validates the session token", Tags: []string{"auth"}},
		{ID: "b", Title: "session middleware", Body: "middleware wires handleAuth into each request", Tags: []string{"web"},
			Links: []Link{{TargetID: "c", Annotation: "routing evidence"}}},
		{ID: "c", Title: "request routing", Body: "routing dispatches to handleAuth", Tags: []string{"web", "auth"}},
		{ID: "d", Title: "unrelated", Body: "quilt marmalade xylophone", Tags: []string{"misc"}},
	}
	inbound = map[string][]string{"c": {"routing evidence"}}
	outbound = map[string][]string{"b": {"routing evidence"}}
	return corpus, inbound, outbound
}

// TestCorpusScorerMatchesReference is the regression guard for the tokenization
// memoization: a CorpusScorer with a warm token cache must return score maps
// byte-identical to the cold BM25RRFPerFieldForCorpus reference for every query,
// including a filtered candidate subset.
//
//	property [1]: ∀ (corpus, candidates, query): Score_memoized == Score_reference
func TestCorpusScorerMatchesReference(t *testing.T) {
	corpus, inbound, outbound := scorerFixture()
	fidf := BM25FieldIDF(corpus, inbound)
	scorer := NewCorpusScorer(corpus, fidf, inbound, outbound)

	candidateSets := [][]*Note{
		corpus,
		{corpus[0], corpus[2]}, // filtered subset
		{corpus[1]},
	}
	queries := []string{"handleAuth token", "routing", "session middleware", "xylophone-unknown", "auth"}

	for _, cands := range candidateSets {
		for _, q := range queries {
			want := BM25RRFPerFieldForCorpus(corpus, cands, fidf, q, inbound, outbound)
			got := scorer.Score(cands, q)
			assertScoreMapsEqual(t, fmt.Sprintf("cands=%d query=%q", len(cands), q), want, got)
			// Second call exercises the warm cache; must be identical too.
			got2 := scorer.Score(cands, q)
			assertScoreMapsEqual(t, fmt.Sprintf("warm cands=%d query=%q", len(cands), q), want, got2)
		}
	}
}

func assertScoreMapsEqual(t *testing.T, ctx string, want, got map[string]float64) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("%s: key count want=%d got=%d\nwant=%v\ngot=%v", ctx, len(want), len(got), want, got)
	}
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("%s: key %q missing from got", ctx, k)
		}
		if gv != wv {
			t.Fatalf("%s: key %q want=%v got=%v", ctx, k, wv, gv)
		}
	}
}
