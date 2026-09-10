package cmd

import (
	"sort"
	"strings"

	"github.com/jaresty/nn/internal/config"
	"github.com/jaresty/nn/internal/index"
	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/trace"
)

// traceAnnotator returns a trace.Annotator that ranks the prepared corpus for a
// query via the memoizing per-field scorer and returns the top-k related notes.
// This gives nn trace / grep --trace the same ranking as grep/ast/shuf and reuses
// the corpus tokenization cache across every traced node. Returns nil when there
// is no corpus, so trace leaves NNNotes empty.
func traceAnnotator(prepared preparedCorpus, k int) trace.Annotator {
	if len(prepared.corpus) == 0 {
		return nil
	}
	if k <= 0 {
		k = 2
	}
	return func(query string) []trace.NoteRef {
		scores := prepared.rankedByQuery(prepared.corpus, query)
		type scored struct {
			n     *note.Note
			score float64
		}
		var ranked []scored
		for _, n := range prepared.corpus {
			if s := scores[n.ID]; s > 0 {
				ranked = append(ranked, scored{n, s})
			}
		}
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
		if len(ranked) > k {
			ranked = ranked[:k]
		}
		refs := make([]trace.NoteRef, 0, len(ranked))
		for _, r := range ranked {
			refs = append(refs, trace.NoteRef{ID: r.n.ID, Title: r.n.Title})
		}
		return refs
	}
}

// preparedCorpus holds the query-invariant BM25 inputs derived from a corpus:
// typed direction/edge annotation channels and their per-field IDF. These
// depend only on the corpus, not on the query, so callers that rank many queries
// against the same corpus (nn grep per match, nn ast per reference, nn shuf per
// sample) should build this once and reuse it across queries.
type preparedCorpus struct {
	corpus     []*note.Note
	channels   note.AnnotationChannels
	fieldIDF   note.TypedFieldIDF
	scorer     *note.TypedCorpusScorer
	queryCache map[string]map[string]float64
}

type rankingQuery struct {
	Name   string
	Text   string
	Weight float64
}

type rankingContribution struct {
	Channel string  `json:"channel"`
	Rank    int     `json:"rank"`
	Weight  float64 `json:"weight"`
}

type fusedRanking struct {
	Note       *note.Note
	Score      float64
	Provenance []rankingContribution
}

// prepareCorpus computes the query-invariant BM25 inputs for a corpus once. It
// performs the expensive work — link-map projection, plus fieldIDF resolution
// which opens the SQLite cache, runs git rev-parse, and hashes the whole corpus
// for the cache key. Pass an empty repoDir to skip caching (for example, in
// tests without a git repository).
func prepareCorpus(corpus []*note.Note, repoDir string) preparedCorpus {
	channels := make(note.AnnotationChannels)
	for _, n := range corpus {
		for _, lnk := range n.Links {
			channels.Add(note.AnnotationInbound, lnk.Type, lnk.TargetID, lnk.Annotation)
			channels.Add(note.AnnotationOutbound, lnk.Type, n.ID, lnk.Annotation)
		}
	}
	fieldIDF, _ := index.GetOrComputeTypedFieldIDFPath(config.DefaultIndexDBPath(), repoDir, corpus, channels)
	scorer := note.NewTypedCorpusScorer(corpus, fieldIDF, channels)
	return preparedCorpus{corpus: corpus, channels: channels, fieldIDF: fieldIDF, scorer: scorer, queryCache: make(map[string]map[string]float64)}
}

// rankedByQuery scores candidates for query using pre-computed corpus inputs.
// It is the query-only half of the ranking path: the scorer reuses tokenization
// across queries, so repeated calls against the same prepared corpus do not
// re-tokenize the corpus per query.
func (p preparedCorpus) rankedByQuery(candidates []*note.Note, query string) map[string]float64 {
	return p.scorer.Score(candidates, query)
}

func (p preparedCorpus) cachedRankedByQuery(candidates []*note.Note, query string) map[string]float64 {
	var key strings.Builder
	key.WriteString(query)
	for _, candidate := range candidates {
		key.WriteByte(0)
		key.WriteString(candidate.ID)
	}
	cacheKey := key.String()
	if scores, ok := p.queryCache[cacheKey]; ok {
		return scores
	}
	scores := p.rankedByQuery(candidates, query)
	p.queryCache[cacheKey] = scores
	return scores
}

func (p preparedCorpus) rankedByQueries(candidates []*note.Note, queries []rankingQuery) []fusedRanking {
	candidateOrder := make(map[string]int, len(candidates))
	byID := make(map[string]*fusedRanking)
	for i, candidate := range candidates {
		candidateOrder[candidate.ID] = i
	}

	for _, query := range queries {
		if strings.TrimSpace(query.Text) == "" || query.Weight <= 0 {
			continue
		}
		scores := p.cachedRankedByQuery(candidates, query.Text)
		ranked := make([]*note.Note, 0, len(scores))
		for _, candidate := range candidates {
			if scores[candidate.ID] > 0 {
				ranked = append(ranked, candidate)
			}
		}
		sort.SliceStable(ranked, func(i, j int) bool {
			left, right := scores[ranked[i].ID], scores[ranked[j].ID]
			if left != right {
				return left > right
			}
			return candidateOrder[ranked[i].ID] < candidateOrder[ranked[j].ID]
		})
		for i, candidate := range ranked {
			result := byID[candidate.ID]
			if result == nil {
				result = &fusedRanking{Note: candidate}
				byID[candidate.ID] = result
			}
			rank := i + 1
			// Each per-query score is already the sum of field-level RRF
			// contributions. Weighted addition is one final accumulator over
			// (query, field) channels; do not reciprocal-rank these scores again.
			result.Score += query.Weight * scores[candidate.ID]
			result.Provenance = append(result.Provenance, rankingContribution{
				Channel: query.Name,
				Rank:    rank,
				Weight:  query.Weight,
			})
		}
	}

	results := make([]fusedRanking, 0, len(byID))
	for _, candidate := range candidates {
		if result := byID[candidate.ID]; result != nil {
			results = append(results, *result)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return candidateOrder[results[i].Note.ID] < candidateOrder[results[j].Note.ID]
	})
	return results
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
