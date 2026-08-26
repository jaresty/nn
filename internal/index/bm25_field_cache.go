package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/jaresty/nn/internal/note"
)

const fieldIDFCacheVersion = "field-idf-v4"
const fieldIDFCacheEntriesPerRepo = 8

// GetOrComputeFieldIDF preserves the flat inbound API as an adapter to the
// typed UNCLASSIFIED channel cache.
func (idx *Index) GetOrComputeFieldIDF(repoDir string, notes []*note.Note, inbound map[string][]string) (note.FieldIDF, error) {
	fidf, err := idx.GetOrComputeTypedFieldIDF(repoDir, notes, note.FlatAnnotationChannels(inbound, nil))
	return fidf.FlatFieldIDF(note.AnnotationInbound), err
}

func fieldIDFRepoPrefix(repoDir string) string {
	absolute, err := filepath.Abs(repoDir)
	if err != nil {
		absolute = filepath.Clean(repoDir)
	}
	repoHash := sha256.Sum256([]byte(absolute))
	return fmt.Sprintf("%s:%x:", fieldIDFCacheVersion, repoHash)
}

func fieldIDFCacheKey(repoDir, head string, notes []*note.Note, inbound map[string][]string) string {
	return typedFieldIDFCacheKey(repoDir, head, notes, note.FlatAnnotationChannels(inbound, nil))
}

// pruneFieldIDFCacheTx retains the newest bounded set for one repository
// namespace; rows belonging to other repositories and legacy formats remain untouched.
func pruneFieldIDFCacheTx(tx *sql.Tx, repoPrefix string) error {
	_, err := tx.Exec(`DELETE FROM bm25_field_cache
		WHERE commit_hash LIKE ?
		AND rowid NOT IN (
			SELECT rowid FROM bm25_field_cache
			WHERE commit_hash LIKE ?
			ORDER BY rowid DESC LIMIT ?
		)`, repoPrefix+"%", repoPrefix+"%", fieldIDFCacheEntriesPerRepo)
	return err
}

func (idx *Index) pruneFieldIDFCache(repoPrefix string) error {
	tx, err := idx.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := pruneFieldIDFCacheTx(tx, repoPrefix); err != nil {
		return err
	}
	return tx.Commit()
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
