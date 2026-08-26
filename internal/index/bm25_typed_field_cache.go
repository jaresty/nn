package index

import (
	"encoding/json"

	"github.com/jaresty/nn/internal/note"
)

// typedFieldIDFCacheKey is a compatibility scaffold for the typed cache key.
// It deliberately projects channels to the legacy flat shape so red tests can
// demonstrate that direction and edge-type identity are not yet represented.
func typedFieldIDFCacheKey(repoDir, head string, notes []*note.Note, channels note.AnnotationChannels) string {
	inbound, outbound := channels.FlatMaps()
	for noteID, annotations := range outbound {
		inbound[noteID] = append(inbound[noteID], annotations...)
	}
	return fieldIDFCacheKey(repoDir, head, notes, inbound)
}

func (idx *Index) storeTypedFieldIDF(hash string, fidf note.TypedFieldIDF) error {
	encoded, err := json.Marshal(fidf)
	if err != nil {
		return err
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
	return fidf, nil
}
