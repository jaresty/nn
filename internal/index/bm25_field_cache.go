package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jaresty/nn/internal/note"
)

// GetOrComputeFieldIDF returns the per-field BM25 IDF struct for the given notes corpus.
// It caches the result in SQLite keyed by the git HEAD commit hash of repoDir.
// If git rev-parse HEAD fails (fresh repo or non-git dir), the IDF is computed and
// returned without caching.
func (idx *Index) GetOrComputeFieldIDF(repoDir string, notes []*note.Note, inbound map[string][]string) (note.FieldIDF, error) {
	if repoDir == "" {
		return note.BM25FieldIDF(notes, inbound), nil
	}
	hash, err := headCommitHash(repoDir)
	if err != nil {
		return note.BM25FieldIDF(notes, inbound), nil
	}

	// Fast path: try cache without a lock.
	if fidf, err := idx.getFieldIDFFromCache(hash); err == nil {
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
	if scanErr := tx.QueryRow(`SELECT field_idf_json FROM bm25_field_cache WHERE commit_hash = ?`, hash).Scan(&raw); scanErr == nil {
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
		hash, string(b),
	); storeErr != nil {
		return fidf, nil
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fidf, nil
	}
	return fidf, nil
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
