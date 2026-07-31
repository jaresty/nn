package index

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jaresty/nn/internal/note"
)

// GetOrComputeIDF returns the BM25 IDF map for the given notes corpus and query terms.
// It caches the result in SQLite keyed by the git HEAD commit hash of repoDir.
// On a cache hit the IDF is decoded from JSON without re-stemming the corpus.
// If git rev-parse HEAD fails (e.g. fresh repo with no commits), the IDF is
// computed and returned without being stored.
func (idx *Index) GetOrComputeIDF(repoDir string, notes []*note.Note, terms []string) (map[string]float64, error) {
	hash, err := headCommitHash(repoDir)
	if err != nil {
		// Fresh repo or non-git dir — compute without caching.
		return note.BM25IDF(notes, terms), nil
	}

	// Try cache hit.
	var raw string
	err = idx.db.QueryRow(`SELECT idf_json FROM bm25_cache WHERE commit_hash = ?`, hash).Scan(&raw)
	if err == nil {
		idf := make(map[string]float64)
		if jsonErr := json.Unmarshal([]byte(raw), &idf); jsonErr == nil {
			return idf, nil
		}
		// Corrupt JSON — fall through to recompute.
	}

	// Cache miss or corrupt — compute and store.
	idf := note.BM25IDF(notes, terms)
	b, jsonErr := json.Marshal(idf)
	if jsonErr != nil {
		return idf, fmt.Errorf("index.GetOrComputeIDF: marshal: %w", jsonErr)
	}
	if _, storeErr := idx.db.Exec(
		`INSERT OR REPLACE INTO bm25_cache (commit_hash, idf_json) VALUES (?, ?)`,
		hash, string(b),
	); storeErr != nil {
		// Non-fatal — return computed IDF even if we can't cache it.
		return idf, nil
	}
	return idf, nil
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
