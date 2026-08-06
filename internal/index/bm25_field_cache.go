package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jaresty/nn/internal/note"
)

const fieldIDFCacheVersion = "field-idf-v2"

// GetOrComputeFieldIDF returns the per-field BM25 IDF struct for the given notes corpus.
// It caches the result in SQLite using a versioned HEAD and corpus fingerprint.
// If git rev-parse HEAD fails (fresh repo or non-git dir), the IDF is computed and
// returned without caching.
func (idx *Index) GetOrComputeFieldIDF(repoDir string, notes []*note.Note, inbound map[string][]string) (note.FieldIDF, error) {
	if repoDir == "" {
		return note.BM25FieldIDF(notes, inbound), nil
	}
	head, err := headCommitHash(repoDir)
	if err != nil {
		return note.BM25FieldIDF(notes, inbound), nil
	}
	cacheKey := fieldIDFCacheKey(head, notes, inbound)

	// Fast path: try cache without a lock.
	if fidf, err := idx.getFieldIDFFromCache(cacheKey); err == nil {
		return fidf, nil
	}

	// Cache miss — acquire exclusive lock, double-check, compute and store.
	tx, txErr := idx.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if txErr != nil {
		return note.BM25FieldIDF(notes, inbound), nil
	}
	defer tx.Rollback() //nolint:errcheck

	// Double-check under lock.
	var raw string
	if scanErr := tx.QueryRow(`SELECT field_idf_json FROM bm25_field_cache WHERE commit_hash = ?`, cacheKey).Scan(&raw); scanErr == nil {
		var fidf note.FieldIDF
		if jsonErr := json.Unmarshal([]byte(raw), &fidf); jsonErr == nil {
			tx.Rollback() //nolint:errcheck
			return fidf, nil
		}
	}

	// Compute and store under lock.
	fidf := note.BM25FieldIDF(notes, inbound)
	b, jsonErr := json.Marshal(fidf)
	if jsonErr != nil {
		return fidf, fmt.Errorf("index.GetOrComputeFieldIDF: marshal: %w", jsonErr)
	}
	if _, storeErr := tx.Exec(
		`INSERT OR REPLACE INTO bm25_field_cache (commit_hash, field_idf_json) VALUES (?, ?)`,
		cacheKey, string(b),
	); storeErr != nil {
		return fidf, nil
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fidf, nil
	}
	return fidf, nil
}

func fieldIDFCacheKey(head string, notes []*note.Note, inbound map[string][]string) string {
	type cacheDocument struct {
		ID      string   `json:"id"`
		Title   string   `json:"title"`
		Body    string   `json:"body"`
		Tags    []string `json:"tags"`
		Inbound []string `json:"inbound"`
	}
	ordered := append([]*note.Note(nil), notes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	documents := make([]cacheDocument, 0, len(ordered))
	for _, n := range ordered {
		tags := append([]string(nil), n.Tags...)
		annotations := append([]string(nil), inbound[n.ID]...)
		sort.Strings(tags)
		sort.Strings(annotations)
		documents = append(documents, cacheDocument{
			ID: n.ID, Title: n.Title, Body: n.Body, Tags: tags, Inbound: annotations,
		})
	}
	encoded, _ := json.Marshal(documents)
	fingerprint := sha256.Sum256(encoded)
	return fmt.Sprintf("%s:%s:%x", fieldIDFCacheVersion, head, fingerprint)
}

// GetOrComputeFieldIDFPath is a convenience wrapper that opens the index at dbPath,
// calls GetOrComputeFieldIDF, and closes the index. Use when no *Index is already open.
func GetOrComputeFieldIDFPath(dbPath, repoDir string, notes []*note.Note, inbound map[string][]string) (note.FieldIDF, error) {
	idx, err := Open(dbPath)
	if err != nil {
		return note.BM25FieldIDF(notes, inbound), nil
	}
	defer idx.Close()
	return idx.GetOrComputeFieldIDF(repoDir, notes, inbound)
}

// getFieldIDFFromCache retrieves a stored FieldIDF from the cache for the given hash.
// Returns an error if not found or if JSON is corrupt.
func (idx *Index) getFieldIDFFromCache(hash string) (note.FieldIDF, error) {
	var raw string
	if err := idx.db.QueryRow(`SELECT field_idf_json FROM bm25_field_cache WHERE commit_hash = ?`, hash).Scan(&raw); err != nil {
		return note.FieldIDF{}, err
	}
	var fidf note.FieldIDF
	if err := json.Unmarshal([]byte(raw), &fidf); err != nil {
		return note.FieldIDF{}, err
	}
	return fidf, nil
}

// storeFieldIDF writes a FieldIDF to the cache under the given hash.
// Exported for test seeding; normal callers use GetOrComputeFieldIDF.
func (idx *Index) storeFieldIDF(hash string, fidf note.FieldIDF) error {
	b, err := json.Marshal(fidf)
	if err != nil {
		return fmt.Errorf("storeFieldIDF: marshal: %w", err)
	}
	_, err = idx.db.Exec(`INSERT OR REPLACE INTO bm25_field_cache (commit_hash, field_idf_json) VALUES (?, ?)`, hash, string(b))
	return err
}
