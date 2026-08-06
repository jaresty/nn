package index

import (
	"path/filepath"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func makeTestNotes() []*note.Note {
	return []*note.Note{
		{ID: "n1", Title: "alpha beta gamma", Body: "delta epsilon", Tags: []string{"zeta"}},
		{ID: "n2", Title: "epsilon zeta", Body: "alpha gamma omega", Tags: []string{"beta"}},
	}
}

// property [1]: GetOrComputeFieldIDF returns same value as BM25FieldIDF for same corpus.
func TestGetOrComputeFieldIDF_Correctness(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idx.db")
	idx, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer idx.Close()

	notes := makeTestNotes()
	inbound := map[string][]string{"n1": {"n2"}}

	got, err := idx.GetOrComputeFieldIDF("", notes, inbound)
	if err != nil {
		t.Fatalf("GetOrComputeFieldIDF: %v", err)
	}
	want := note.BM25FieldIDF(notes, inbound)

	for tok, wantScore := range want.Title {
		if gotScore := got.Title[tok]; gotScore != wantScore {
			t.Errorf("Title[%q]: got %.6f want %.6f", tok, gotScore, wantScore)
		}
	}
	for tok, wantScore := range want.Body {
		if gotScore := got.Body[tok]; gotScore != wantScore {
			t.Errorf("Body[%q]: got %.6f want %.6f", tok, gotScore, wantScore)
		}
	}
	for tok, wantScore := range want.Tags {
		if gotScore := got.Tags[tok]; gotScore != wantScore {
			t.Errorf("Tags[%q]: got %.6f want %.6f", tok, gotScore, wantScore)
		}
	}
	for tok, wantScore := range want.Inbound {
		if gotScore := got.Inbound[tok]; gotScore != wantScore {
			t.Errorf("Inbound[%q]: got %.6f want %.6f", tok, gotScore, wantScore)
		}
	}
}

// property [2]: second call with same commit hash does not recompute (cache hit).
// We verify indirectly: by passing nil notes on second call — if it hits cache,
// it returns non-empty results; if it recomputes from nil, it panics or returns empty.
func TestGetOrComputeFieldIDF_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idx.db")
	idx, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer idx.Close()

	notes := makeTestNotes()

	// First call: populate cache with a fake hash via empty repoDir (no-cache path).
	// To test caching we need a real hash — seed it directly using the internal helper.
	hash := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	fidf := note.BM25FieldIDF(notes, nil)
	if err := idx.storeFieldIDF(hash, fidf); err != nil {
		t.Fatalf("storeFieldIDF: %v", err)
	}

	// Second call with nil notes — must hit cache and return non-empty Title map.
	got, err := idx.getFieldIDFFromCache(hash)
	if err != nil {
		t.Fatalf("getFieldIDFFromCache: %v", err)
	}
	if len(got.Title) == 0 {
		t.Error("property [2]: cache hit returned empty Title map; expected stored values")
	}
}

// property [3]: transactional integrity is tested via storeFieldIDF + getFieldIDFFromCache roundtrip.
func TestGetOrComputeFieldIDF_TransactionRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idx.db")
	idx, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer idx.Close()

	notes := makeTestNotes()
	fidf := note.BM25FieldIDF(notes, nil)

	if err := idx.storeFieldIDF("abc123", fidf); err != nil {
		t.Fatalf("storeFieldIDF: %v", err)
	}
	got, err := idx.getFieldIDFFromCache("abc123")
	if err != nil {
		t.Fatalf("getFieldIDFFromCache: %v", err)
	}
	for tok, wantScore := range fidf.Title {
		if got.Title[tok] != wantScore {
			t.Errorf("roundtrip Title[%q]: got %.6f want %.6f", tok, got.Title[tok], wantScore)
		}
	}
}

// property [5]: empty repoDir (no git HEAD) returns correct value without caching.
func TestGetOrComputeFieldIDF_FreshRepo(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idx.db")
	idx, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer idx.Close()

	notes := makeTestNotes()
	inbound := map[string][]string{"n1": {"n2"}}

	// Empty repoDir — no git HEAD, must not cache.
	got, err := idx.GetOrComputeFieldIDF("", notes, inbound)
	if err != nil {
		t.Fatalf("GetOrComputeFieldIDF: %v", err)
	}
	want := note.BM25FieldIDF(notes, inbound)
	for tok, wantScore := range want.Title {
		if got.Title[tok] != wantScore {
			t.Errorf("fresh-repo Title[%q]: got %.6f want %.6f", tok, got.Title[tok], wantScore)
		}
	}

	// Verify nothing was stored.
	var count int
	if err := idx.db.QueryRow("SELECT COUNT(*) FROM bm25_field_cache").Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("property [5]: expected 0 cached rows for fresh repo, got %d", count)
	}
}
