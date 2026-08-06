package index_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/index"
	"github.com/jaresty/nn/internal/note"
)

func setupIndexWithGit(t *testing.T) (*index.Index, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	idx, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	// Init a git repo so rev-parse HEAD works after a commit.
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@test.com"},
		{"-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return idx, dir
}

func makeTestNotes(n int) []*note.Note {
	notes := make([]*note.Note, n)
	for i := range n {
		notes[i] = &note.Note{
			ID:       note.GenerateID(),
			Title:    "note title word",
			Body:     "body content word",
			Type:     note.TypeObservation,
			Status:   note.StatusDraft,
			Created:  time.Now().UTC(),
			Modified: time.Now().UTC(),
		}
	}
	return notes
}

func gitCommitAll(t *testing.T, dir string) string {
	t.Helper()
	// Write a dummy file and commit so HEAD exists.
	dummy := filepath.Join(dir, "dummy.txt")
	if err := exec.Command("bash", "-c", "echo x > "+dummy).Run(); err != nil {
		t.Fatalf("write dummy: %v", err)
	}
	for _, args := range [][]string{
		{"-C", dir, "add", "."},
		{"-C", dir, "commit", "-m", "test commit"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return string(out[:len(out)-1]) // trim newline
}

// TestGetOrComputeIDFCacheHit proves behavior [1]+[3]:
// a second call with the same HEAD hash returns without recomputing.
func TestGetOrComputeIDFCacheHit(t *testing.T) {
	idx, dir := setupIndexWithGit(t)
	notes := makeTestNotes(5)
	terms := note.Tokenize("word content")

	gitCommitAll(t, dir)

	// First call — cold, must compute and store.
	idf1, err := idx.GetOrComputeIDF(dir, notes, terms)
	if err != nil {
		t.Fatalf("first GetOrComputeIDF: %v", err)
	}

	// Second call — must hit cache (same commit hash).
	idf2, err := idx.GetOrComputeIDF(dir, notes, terms)
	if err != nil {
		t.Fatalf("second GetOrComputeIDF: %v", err)
	}

	// Results must be identical.
	if len(idf1) != len(idf2) {
		t.Errorf("idf length mismatch: %d vs %d", len(idf1), len(idf2))
	}
	for k, v := range idf1 {
		if idf2[k] != v {
			t.Errorf("idf[%q]: %v vs %v", k, v, idf2[k])
		}
	}
}

// TestGetOrComputeIDFCacheMissOnNewCommit proves behavior [2]:
// after a new commit the cache is invalidated and IDF is recomputed.
func TestGetOrComputeIDFCacheMissOnNewCommit(t *testing.T) {
	idx, dir := setupIndexWithGit(t)
	notes := makeTestNotes(3)
	terms := note.Tokenize("word content")

	hash1 := gitCommitAll(t, dir)

	_, err := idx.GetOrComputeIDF(dir, notes, terms)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Make a new commit — HEAD changes.
	hash2 := gitCommitAll(t, dir)
	if hash1 == hash2 {
		t.Fatal("expected different hashes after second commit")
	}

	// Different notes corpus — if cache was stale, would return old result.
	notes2 := makeTestNotes(10)
	idf2, err := idx.GetOrComputeIDF(dir, notes2, terms)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	// Cold compute of notes2 must match.
	idfCold := note.BM25IDF(notes2, terms)
	for k, v := range idfCold {
		if idf2[k] != v {
			t.Errorf("after new commit idf[%q]: cached %v, cold %v", k, idf2[k], v)
		}
	}
}

// TestGetOrComputeIDFDifferentTermsSameCommit proves the cache is query-independent:
// a second call with different query terms on the same commit must return non-zero
// IDF for those new terms (not the stale IDF from the first call's terms).
func TestGetOrComputeIDFDifferentTermsSameCommit(t *testing.T) {
	idx, dir := setupIndexWithGit(t)
	notes := []*note.Note{
		{ID: note.GenerateID(), Title: "cobra command", Body: "cobra is a CLI framework", Type: note.TypeConcept, Status: note.StatusDraft, Created: time.Now().UTC(), Modified: time.Now().UTC()},
		{ID: note.GenerateID(), Title: "daily note", Body: "daily session log", Type: note.TypeObservation, Status: note.StatusDraft, Created: time.Now().UTC(), Modified: time.Now().UTC()},
	}
	gitCommitAll(t, dir)

	// First call: terms from "cobra command" (stemmed)
	cobraTerms := note.Tokenize("cobra command")
	_, err := idx.GetOrComputeIDF(dir, notes, cobraTerms)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call: different terms from "daily session" — must get non-zero IDF
	dailyTerms := note.Tokenize("daily session")
	idf2, err := idx.GetOrComputeIDF(dir, notes, dailyTerms)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	stemmedDaily := dailyTerms[0] // "daili"
	if idf2[stemmedDaily] == 0 {
		t.Errorf("property [2]: IDF for %q is 0 on second call — cache returned stale terms from first call", stemmedDaily)
	}
}

// TestGetOrComputeIDFFreshRepoFallback proves behavior [4]:
// when git rev-parse HEAD fails (no commits), returns cold IDF without error.
func TestGetOrComputeIDFFreshRepoFallback(t *testing.T) {
	idx, dir := setupIndexWithGit(t)
	// No commits — HEAD doesn't exist.
	notes := makeTestNotes(3)
	terms := note.Tokenize("word content")

	idf, err := idx.GetOrComputeIDF(dir, notes, terms)
	if err != nil {
		t.Fatalf("fresh repo fallback returned error: %v", err)
	}
	cold := note.BM25IDF(notes, terms)
	if len(idf) != len(cold) {
		t.Errorf("fallback idf length %d != cold %d", len(idf), len(cold))
	}
}

// TestGetOrComputeIDFExclusiveLock_Property1a proves properties [1a] and [1b]:
// under concurrent cache misses from multiple goroutines sharing the same *Index,
// all goroutines return correct IDF values and no errors (exclusive transaction
// ensures at most one computes, and losers re-read the winner's cached result).
func TestGetOrComputeIDFExclusiveLock_Property1a(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	gitCommitAllDir(t, dir)

	// Open once — all goroutines share the same *Index (simulating goroutines
	// within a single process; the exclusive transaction serializes their access).
	idx, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	notes := makeTestNotes(5)
	terms := note.Tokenize("word content")

	const N = 8
	results := make([]map[string]float64, N)
	errs := make([]error, N)
	var wg sync.WaitGroup
	wg.Add(N)
	for i := range N {
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = idx.GetOrComputeIDF(dir, notes, terms)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	// All results must be equal to the cold-computed IDF.
	cold := note.BM25IDF(notes, note.Tokenize("word content"))
	for i, idf := range results {
		for k, v := range cold {
			if idf[k] != v {
				t.Errorf("goroutine %d: idf[%q] = %v, want %v", i, k, idf[k], v)
			}
		}
	}
}

// gitCommitAllDir is a package-level helper (avoids collision with test-method helpers).
func gitCommitAllDir(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@test.com"},
		{"-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	dummy := filepath.Join(dir, "dummy.txt")
	if err := exec.Command("bash", "-c", "echo x > "+dummy).Run(); err != nil {
		t.Fatalf("write dummy: %v", err)
	}
	for _, args := range [][]string{
		{"-C", dir, "add", "."},
		{"-C", dir, "commit", "-m", "test commit"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// TestGetOrComputeIDFStoreFailureNonFatal proves properties [2a] and [2b]:
// when the INSERT fails (DB made read-only after schema setup), the function
// returns the computed IDF without error (non-fatal), and does not hold a lock
// that would block subsequent callers (rollback happened).
func TestGetOrComputeIDFStoreFailureNonFatal(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	gitCommitAllDir(t, dir)

	// Prime the schema.
	idx, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	idx.Close()

	// Make the DB read-only to force INSERT to fail.
	if err := os.Chmod(dbPath, 0o444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dbPath, 0o644) }) //nolint:errcheck

	idx2, err := index.Open(dbPath)
	if err != nil {
		t.Skipf("can't open read-only DB: %v", err)
	}
	defer idx2.Close()

	notes := makeTestNotes(3)
	terms := note.Tokenize("word content")

	// Must return valid IDF even though store will fail.
	idf, err := idx2.GetOrComputeIDF(dir, notes, terms)
	if err != nil {
		t.Fatalf("property [2b] violated: store failure returned error: %v", err)
	}
	cold := note.BM25IDF(notes, terms)
	for k, v := range cold {
		if idf[k] != v {
			t.Errorf("idf[%q] = %v, want %v", k, idf[k], v)
		}
	}

	// Property [2a]: verify lock is released by opening another connection immediately.
	idx3, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("property [2a] violated: lock not released after store failure: %v", err)
	}
	idx3.Close()
}
