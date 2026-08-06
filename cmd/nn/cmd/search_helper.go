package cmd

import (
	"github.com/jaresty/nn/internal/config"
	"github.com/jaresty/nn/internal/index"
	"github.com/jaresty/nn/internal/note"
)

// RankedByQuery returns per-field BM25 RRF scores for notes ranked by query relevance.
// It uses a commit-hash-keyed SQLite cache for the per-field IDF so repeated calls
// within the same git state are cheap. repoDir is the git repository root; pass ""
// to skip caching (e.g. in tests without a git repo).
func RankedByQuery(notes []*note.Note, inbound map[string][]string, query, repoDir string) map[string]float64 {
	fieldIDF, _ := index.GetOrComputeFieldIDFPath(config.DefaultIndexDBPath(), repoDir, notes, inbound)
	return note.BM25RRFPerField(notes, fieldIDF, query, inbound)
}
