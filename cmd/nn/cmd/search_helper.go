package cmd

import (
	"github.com/jaresty/nn/internal/config"
	"github.com/jaresty/nn/internal/index"
	"github.com/jaresty/nn/internal/note"
)

// preparedCorpus holds the query-invariant BM25 inputs derived from a corpus:
// the inbound/outbound link-annotation projections and the per-field IDF. These
// depend only on the corpus, not on the query, so callers that rank many queries
// against the same corpus (nn grep per match, nn ast per reference, nn shuf per
// sample) should build this once and reuse it across queries.
type preparedCorpus struct {
	corpus   []*note.Note
	inbound  map[string][]string
	outbound map[string][]string
	fieldIDF note.FieldIDF
}

// prepareCorpus computes the query-invariant BM25 inputs for a corpus once. It
// performs the expensive work — link-map projection, plus fieldIDF resolution
// which opens the SQLite cache, runs git rev-parse, and hashes the whole corpus
// for the cache key. Pass an empty repoDir to skip caching (for example, in
// tests without a git repository).
func prepareCorpus(corpus []*note.Note, repoDir string) preparedCorpus {
	inbound := make(map[string][]string)
	outbound := make(map[string][]string)
	for _, n := range corpus {
		for _, lnk := range n.Links {
			inbound[lnk.TargetID] = append(inbound[lnk.TargetID], lnk.Annotation)
			outbound[n.ID] = append(outbound[n.ID], lnk.Annotation)
		}
	}
	fieldIDF, _ := index.GetOrComputeFieldIDFPath(config.DefaultIndexDBPath(), repoDir, corpus, inbound)
	return preparedCorpus{corpus: corpus, inbound: inbound, outbound: outbound, fieldIDF: fieldIDF}
}

// rankedByQuery scores candidates for query using pre-computed corpus inputs.
// It is the query-only half of the ranking path and performs no per-call corpus
// preparation.
func (p preparedCorpus) rankedByQuery(candidates []*note.Note, query string) map[string]float64 {
	return note.BM25RRFPerFieldForCorpus(p.corpus, candidates, p.fieldIDF, query, p.inbound, p.outbound)
}

// RankedByQuery returns positive per-field BM25 RRF scores for candidates.
// Field statistics and annotation projections come from the full corpus. It uses
// a commit-hash-keyed SQLite cache for established field IDF; pass an empty
// repoDir to skip caching (for example, in tests without a git repository).
//
// This is the single-call convenience wrapper: it prepares the corpus and ranks
// one query. Callers ranking many queries against the same corpus should call
// prepareCorpus once and reuse rankedByQuery to avoid recomputing the invariant
// corpus work per query.
func RankedByQuery(corpus, candidates []*note.Note, query, repoDir string) map[string]float64 {
	return prepareCorpus(corpus, repoDir).rankedByQuery(candidates, query)
}
