package index

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jaresty/nn/internal/note"
)

// GetOrComputeIDF returns the BM25 IDF map for the given notes corpus and query terms.
// It caches the full-corpus IDF (all tokens) in SQLite keyed by the git HEAD commit
// hash of repoDir, then returns the subset relevant to terms. This makes the cache
// query-independent: different queries against the same commit share one cached entry.
// If git rev-parse HEAD fails (e.g. fresh repo with no commits), the IDF is
// computed and returned without being stored.
func (idx *Index) GetOrComputeIDF(repoDir string, notes []*note.Note, terms []string) (map[string]float64, error) {
	hash, err := headCommitHash(repoDir)
	if err != nil {
		// Fresh repo or non-git dir — compute without caching.
		return note.BM25IDF(notes, terms), nil
	}

	// Try cache hit — stored as full-corpus IDF.
	var raw string
	err = idx.db.QueryRow(`SELECT idf_json FROM bm25_cache WHERE commit_hash = ?`, hash).Scan(&raw)
	if err == nil {
		fullIDF := make(map[string]float64)
		if jsonErr := json.Unmarshal([]byte(raw), &fullIDF); jsonErr == nil {
			return subsetIDF(fullIDF, terms), nil
		}
		// Corrupt JSON — fall through to recompute.
	}

	// Cache miss or corrupt — compute full-corpus IDF and store it.
	allTerms := corpusTerms(notes)
	fullIDF := note.BM25IDF(notes, allTerms)
	b, jsonErr := json.Marshal(fullIDF)
	if jsonErr != nil {
		return subsetIDF(fullIDF, terms), fmt.Errorf("index.GetOrComputeIDF: marshal: %w", jsonErr)
	}
	if _, storeErr := idx.db.Exec(
		`INSERT OR REPLACE INTO bm25_cache (commit_hash, idf_json) VALUES (?, ?)`,
		hash, string(b),
	); storeErr != nil {
		// Non-fatal — return computed IDF even if we can't cache it.
		return subsetIDF(fullIDF, terms), nil
	}
	return subsetIDF(fullIDF, terms), nil
}

// corpusTerms returns all unique tokens present in the notes corpus.
func corpusTerms(notes []*note.Note) []string {
	seen := make(map[string]struct{})
	for _, n := range notes {
		for _, t := range note.Tokenize(n.Title) {
			seen[t] = struct{}{}
		}
		for _, t := range note.Tokenize(n.Body) {
			seen[t] = struct{}{}
		}
		for _, tag := range n.Tags {
			for _, t := range note.Tokenize(tag) {
				seen[t] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	return out
}

// subsetIDF extracts only the requested terms from a full-corpus IDF map.
func subsetIDF(full map[string]float64, terms []string) map[string]float64 {
	out := make(map[string]float64, len(terms))
	for _, t := range terms {
		out[t] = full[t]
	}
	return out
}

// GetOrComputeIDFPath is a convenience wrapper that opens the index at dbPath,
// calls GetOrComputeIDF, and closes the index. Use when no *Index is already open.
func GetOrComputeIDFPath(dbPath, repoDir string, notes []*note.Note, terms []string) (map[string]float64, error) {
	idx, err := Open(dbPath)
	if err != nil {
		return note.BM25IDF(notes, terms), nil
	}
	defer idx.Close()
	return idx.GetOrComputeIDF(repoDir, notes, terms)
}

func headCommitHash(repoDir string) (string, error) {
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
