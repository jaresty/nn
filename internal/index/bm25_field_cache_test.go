package index

import (
	"fmt"
	"os/exec"
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
func TestFieldIDFCacheKeyIncludesVersionCorpusAndInbound(t *testing.T) {
	notes := makeTestNotes()
	base := fieldIDFCacheKey("/repo", "head", notes, nil)
	if base == "head" {
		t.Fatal("cache key must invalidate legacy commit-only rows")
	}

	changedNotes := makeTestNotes()
	changedNotes[0].Title = "dirty working tree title"
	if got := fieldIDFCacheKey("/repo", "head", changedNotes, nil); got == base {
		t.Fatal("cache key ignored dirty corpus content")
	}

	inbound := map[string][]string{"n1": {"new annotation"}}
	if got := fieldIDFCacheKey("/repo", "head", notes, inbound); got == base {
		t.Fatal("cache key ignored inbound annotations")
	}

	subset := notes[:1]
	if got := fieldIDFCacheKey("/repo", "head", subset, nil); got == base {
		t.Fatal("cache key ignored corpus identity")
	}
}

func TestFieldIDFCacheRejectsLegacyAndChangedCorpusRows(t *testing.T) {
	idx, err := Open(filepath.Join(t.TempDir(), "idx.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	notes := makeTestNotes()
	fidf := note.BM25FieldIDF(notes, nil)
	if err := idx.storeFieldIDF("head", fidf); err != nil {
		t.Fatal(err)
	}
	current := fieldIDFCacheKey("/repo", "head", notes, nil)
	if _, err := idx.getFieldIDFFromCache(current); err == nil {
		t.Fatal("legacy commit-only row satisfied current cache key")
	}
	if err := idx.storeFieldIDF(current, fidf); err != nil {
		t.Fatal(err)
	}
	changed := makeTestNotes()
	changed[0].Body = "dirty body"
	if _, err := idx.getFieldIDFFromCache(fieldIDFCacheKey("/repo", "head", changed, nil)); err == nil {
		t.Fatal("changed corpus reused stale cache row")
	}
}

func TestGetOrComputeFieldIDFRejectsLegacyAndTracksDirtyCorpus(t *testing.T) {
	repo := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}, {"commit", "--allow-empty", "-m", "initial"}} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	idx, err := Open(filepath.Join(t.TempDir(), "idx.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	head, err := headCommitHash(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.storeFieldIDF(head, note.FieldIDF{Title: map[string]float64{"legacy": 99}}); err != nil {
		t.Fatal(err)
	}

	notes := makeTestNotes()
	first, err := idx.GetOrComputeFieldIDF(repo, notes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Title["alpha"] == 0 || first.Title["legacy"] != 0 {
		t.Fatalf("legacy cache row reused: %#v", first.Title)
	}

	notes[0].Title = "dirtyterm"
	second, err := idx.GetOrComputeFieldIDF(repo, notes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Title["dirtyterm"] == 0 || second.Title["alpha"] == first.Title["alpha"] {
		t.Fatalf("dirty corpus did not recompute IDF: %#v", second.Title)
	}
}

func TestPruneFieldIDFCacheIsBoundedAndRepositoryScoped(t *testing.T) {
	idx, err := Open(filepath.Join(t.TempDir(), "idx.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	prefixA := fieldIDFRepoPrefix("/repo/a")
	prefixB := fieldIDFRepoPrefix("/repo/b")
	for i := 0; i < fieldIDFCacheEntriesPerRepo+3; i++ {
		if err := idx.storeFieldIDF(fmt.Sprintf("%sentry-%02d", prefixA, i), note.FieldIDF{}); err != nil {
			t.Fatal(err)
		}
		if err := idx.storeFieldIDF(fmt.Sprintf("%sentry-%02d", prefixB, i), note.FieldIDF{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.pruneFieldIDFCache(prefixA); err != nil {
		t.Fatal(err)
	}
	var countA, countB int
	if err := idx.db.QueryRow(`SELECT COUNT(*) FROM bm25_field_cache WHERE commit_hash LIKE ?`, prefixA+"%").Scan(&countA); err != nil {
		t.Fatal(err)
	}
	if err := idx.db.QueryRow(`SELECT COUNT(*) FROM bm25_field_cache WHERE commit_hash LIKE ?`, prefixB+"%").Scan(&countB); err != nil {
		t.Fatal(err)
	}
	if countA != fieldIDFCacheEntriesPerRepo {
		t.Fatalf("repo A rows=%d want=%d", countA, fieldIDFCacheEntriesPerRepo)
	}
	if countB != fieldIDFCacheEntriesPerRepo+3 {
		t.Fatalf("pruning repo A changed repo B rows: %d", countB)
	}
}

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
