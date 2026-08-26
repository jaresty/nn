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

// GetOrComputeTypedFieldIDF returns typed lexical/channel IDFs from the
// versioned corpus cache, or computes them without caching when HEAD is absent.
func (idx *Index) GetOrComputeTypedFieldIDF(repoDir string, notes []*note.Note, channels note.AnnotationChannels) (note.TypedFieldIDF, error) {
	if repoDir == "" {
		return note.BM25TypedFieldIDF(notes, channels), nil
	}
	head, err := headCommitHash(repoDir)
	if err != nil {
		return note.BM25TypedFieldIDF(notes, channels), nil
	}
	repoPrefix := fieldIDFRepoPrefix(repoDir)
	cacheKey := typedFieldIDFCacheKey(repoDir, head, notes, channels)

	if fidf, err := idx.getTypedFieldIDFFromCache(cacheKey); err == nil {
		return fidf, nil
	}

	tx, txErr := idx.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if txErr != nil {
		return note.BM25TypedFieldIDF(notes, channels), nil
	}
	defer tx.Rollback() //nolint:errcheck

	var raw string
	if scanErr := tx.QueryRow(`SELECT field_idf_json FROM bm25_field_cache WHERE commit_hash = ?`, cacheKey).Scan(&raw); scanErr == nil {
		var fidf note.TypedFieldIDF
		if jsonErr := json.Unmarshal([]byte(raw), &fidf); jsonErr == nil {
			tx.Rollback() //nolint:errcheck
			return fidf.Canonicalized(), nil
		}
	}

	fidf := note.BM25TypedFieldIDF(notes, channels).Canonicalized()
	encoded, jsonErr := json.Marshal(fidf)
	if jsonErr != nil {
		return fidf, fmt.Errorf("index.GetOrComputeTypedFieldIDF: marshal: %w", jsonErr)
	}
	if _, storeErr := tx.Exec(
		`INSERT OR REPLACE INTO bm25_field_cache (commit_hash, field_idf_json) VALUES (?, ?)`,
		cacheKey, string(encoded),
	); storeErr != nil {
		return fidf, nil
	}
	if pruneErr := pruneFieldIDFCacheTx(tx, repoPrefix); pruneErr != nil {
		return fidf, nil
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return fidf, nil
	}
	return fidf, nil
}

func typedFieldIDFCacheKey(repoDir, head string, notes []*note.Note, channels note.AnnotationChannels) string {
	type cacheDocument struct {
		ID    string   `json:"id"`
		Title string   `json:"title"`
		Body  string   `json:"body"`
		Tags  []string `json:"tags"`
	}
	type cacheAssignment struct {
		NoteID      string   `json:"note_id"`
		Annotations []string `json:"annotations"`
	}
	type cacheChannel struct {
		Direction   note.AnnotationDirection `json:"direction"`
		EdgeType    string                   `json:"edge_type"`
		Assignments []cacheAssignment        `json:"assignments"`
	}
	type cacheCorpus struct {
		Documents []cacheDocument `json:"documents"`
		Channels  []cacheChannel  `json:"channels"`
	}

	orderedNotes := append([]*note.Note(nil), notes...)
	sort.Slice(orderedNotes, func(i, j int) bool { return orderedNotes[i].ID < orderedNotes[j].ID })
	fingerprintInput := cacheCorpus{Documents: make([]cacheDocument, 0, len(orderedNotes))}
	for _, n := range orderedNotes {
		tags := append([]string(nil), n.Tags...)
		sort.Strings(tags)
		fingerprintInput.Documents = append(fingerprintInput.Documents, cacheDocument{
			ID: n.ID, Title: n.Title, Body: n.Body, Tags: tags,
		})
	}

	canonical := channels.Canonicalized()
	for _, channel := range canonical.CanonicalChannels() {
		cachedChannel := cacheChannel{Direction: channel.Direction, EdgeType: channel.EdgeType}
		noteIDs := make([]string, 0, len(canonical[channel]))
		for noteID := range canonical[channel] {
			noteIDs = append(noteIDs, noteID)
		}
		sort.Strings(noteIDs)
		for _, noteID := range noteIDs {
			annotations := append([]string(nil), canonical[channel][noteID]...)
			sort.Strings(annotations)
			cachedChannel.Assignments = append(cachedChannel.Assignments, cacheAssignment{
				NoteID: noteID, Annotations: annotations,
			})
		}
		fingerprintInput.Channels = append(fingerprintInput.Channels, cachedChannel)
	}

	encoded, _ := json.Marshal(fingerprintInput)
	fingerprint := sha256.Sum256(encoded)
	return fmt.Sprintf("%s%s:%x", fieldIDFRepoPrefix(repoDir), head, fingerprint)
}

func (idx *Index) storeTypedFieldIDF(hash string, fidf note.TypedFieldIDF) error {
	encoded, err := json.Marshal(fidf.Canonicalized())
	if err != nil {
		return fmt.Errorf("storeTypedFieldIDF: marshal: %w", err)
	}
	_, err = idx.db.Exec(`INSERT OR REPLACE INTO bm25_field_cache (commit_hash, field_idf_json) VALUES (?, ?)`, hash, string(encoded))
	return err
}

func (idx *Index) getTypedFieldIDFFromCache(hash string) (note.TypedFieldIDF, error) {
	var raw string
	if err := idx.db.QueryRow(`SELECT field_idf_json FROM bm25_field_cache WHERE commit_hash = ?`, hash).Scan(&raw); err != nil {
		return note.TypedFieldIDF{}, err
	}
	var fidf note.TypedFieldIDF
	if err := json.Unmarshal([]byte(raw), &fidf); err != nil {
		return note.TypedFieldIDF{}, err
	}
	return fidf.Canonicalized(), nil
}

// GetOrComputeTypedFieldIDFPath opens the shared index cache for a typed IDF
// lookup. An unavailable cache degrades to a direct computation.
func GetOrComputeTypedFieldIDFPath(dbPath, repoDir string, notes []*note.Note, channels note.AnnotationChannels) (note.TypedFieldIDF, error) {
	idx, err := Open(dbPath)
	if err != nil {
		return note.BM25TypedFieldIDF(notes, channels), nil
	}
	defer idx.Close()
	return idx.GetOrComputeTypedFieldIDF(repoDir, notes, channels)
}
