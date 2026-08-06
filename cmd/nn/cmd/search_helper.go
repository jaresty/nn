package cmd

import (
	"github.com/jaresty/nn/internal/config"
	"github.com/jaresty/nn/internal/index"
	"github.com/jaresty/nn/internal/note"
)

// RankedByQuery returns positive per-field BM25 RRF scores for candidates.
// Field statistics and annotation projections come from the full corpus. It uses
// a commit-hash-keyed SQLite cache for established field IDF; pass an empty
// repoDir to skip caching (for example, in tests without a git repository).
func RankedByQuery(corpus, candidates []*note.Note, query, repoDir string) map[string]float64 {
	inbound := make(map[string][]string)
	outbound := make(map[string][]string)
	for _, n := range corpus {
		for _, lnk := range n.Links {
			inbound[lnk.TargetID] = append(inbound[lnk.TargetID], lnk.Annotation)
			outbound[n.ID] = append(outbound[n.ID], lnk.Annotation)
		}
	}
	fieldIDF, _ := index.GetOrComputeFieldIDFPath(config.DefaultIndexDBPath(), repoDir, corpus, inbound)
	return note.BM25RRFPerFieldForCorpus(corpus, candidates, fieldIDF, query, inbound, outbound)
}
