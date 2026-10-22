package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTranscriptSearchCacheColdWarmAndEquivalent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	writeTranscriptFile(t, p,
		`{"type":"session","id":"s"}`+"\n"+
			`{"type":"message","id":"a","message":{"role":"assistant","content":"Needle one"}}`+"\n"+
			`{"type":"message","id":"b","agentId":"child","message":{"role":"assistant","content":"needle two"}}`+"\n")
	cache, err := openTranscriptSearchCache(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	match, _ := transcriptSearchMatcher("needle", false)
	want, err := searchTranscriptFilesMatching([]string{p}, match, "", "", false, 1)
	if err != nil {
		t.Fatal(err)
	}
	cold, coldStats, err := cache.SearchSubstring([]string{p}, "needle", "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cold, want) {
		t.Fatalf("ASSERT_TRANSCRIPT_CACHE_COLD_EQUIVALENCE: got=%+v want=%+v", cold, want)
	}
	if coldStats.RefreshedFiles != 1 || coldStats.DecodedRecords != 3 {
		t.Fatalf("ASSERT_TRANSCRIPT_CACHE_COLD_DECODE: %+v", coldStats)
	}

	warm, warmStats, err := cache.SearchSubstring([]string{p}, "needle", "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(warm, want) || warmStats.RefreshedFiles != 0 || warmStats.DecodedRecords != 0 {
		t.Fatalf("ASSERT_TRANSCRIPT_CACHE_WARM_REUSE: result=%+v stats=%+v", warm, warmStats)
	}
}

func TestTranscriptSearchCacheSubstringParityMatrix(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.jsonl")
	b := filepath.Join(dir, "b.jsonl")
	writeTranscriptFile(t, a,
		`{"type":"session","id":"alpha"}`+"\n"+
			`{"type":"message","id":"a1","timestamp":"2026-09-08T00:00:00Z","message":{"role":"assistant","content":"Needle K Σ İ"}}`+"\n"+
			`{"type":"message","id":"a2","agentId":"child","timestamp":"2026-09-10T00:00:00Z","message":{"role":"assistant","content":"needle child"}}`+"\n")
	writeTranscriptFile(t, b,
		`{"type":"message","id":"b1","message":{"role":"assistant","content":[{"type":"text","text":"needle array"}]}}`+"\n"+
			`{"type":"message","id":"b2","message":{"role":"toolResult","content":[{"type":"tool_result","text":"hidden needle"}]}}`+"\n")
	cache, err := openTranscriptSearchCache(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	files := []string{a, b}
	for _, query := range []string{"needle", "k", "σ", "i", "missing", ""} {
		for _, limit := range []int{1, 2, 20} {
			for _, agent := range []string{"", "ROOT", "child", "absent"} {
				for _, before := range []string{"", "2026-09-09T00:00:00Z"} {
					match, _ := transcriptSearchMatcher(query, false)
					want, wantErr := searchTranscriptFilesMatching(files, match, agent, before, false, limit)
					got, _, gotErr := cache.SearchSubstring(files, query, agent, before, limit)
					if wantErr != nil || gotErr != nil || !reflect.DeepEqual(got, want) {
						t.Fatalf("ASSERT_TRANSCRIPT_CACHE_SUBSTRING_PARITY query=%q limit=%d agent=%q before=%q: got=%+v/%v want=%+v/%v", query, limit, agent, before, got, gotErr, want, wantErr)
					}
				}
			}
		}
	}
}

func TestTranscriptSearchCacheInvalidatesAndPrunes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "session.jsonl")
	writeTranscriptFile(t, p, `{"type":"message","id":"old","message":{"role":"assistant","content":"old marker"}}`+"\n")
	cache, err := openTranscriptSearchCache(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	oldMatch, _ := transcriptSearchMatcher("old marker", false)
	if _, _, err = cache.Search([]string{p}, oldMatch, "", "", false, 10); err != nil {
		t.Fatal(err)
	}

	writeTranscriptFile(t, p, `{"type":"message","id":"new","message":{"role":"assistant","content":"new marker longer"}}`+"\n")
	newMatch, _ := transcriptSearchMatcher("new marker", false)
	got, stats, err := cache.Search([]string{p}, newMatch, "", "", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Returned != 1 || got.Matches[0].EventID != "new" || stats.RefreshedFiles != 1 {
		t.Fatalf("ASSERT_TRANSCRIPT_CACHE_INVALIDATION: result=%+v stats=%+v", got, stats)
	}

	if err = os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if _, _, err = cache.Search(nil, newMatch, "", "", false, 10); err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalTranscriptPath(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	canonical = filepath.Join(canonical, filepath.Base(p))
	var count int
	if err = cache.db.QueryRow(`SELECT count(*) FROM transcript_search_files WHERE path = ?`, canonical).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("ASSERT_TRANSCRIPT_CACHE_DELETION_PRUNED: count=%d", count)
	}
}

func TestTranscriptSearchCacheRejectsPartialMalformedReplacement(t *testing.T) {
	p := filepath.Join(t.TempDir(), "session.jsonl")
	writeTranscriptFile(t, p, `{"type":"message","id":"old","message":{"role":"assistant","content":"stable"}}`+"\n")
	cache, err := openTranscriptSearchCache(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	match, _ := transcriptSearchMatcher("stable", false)
	if _, _, err = cache.Search([]string{p}, match, "", "", false, 10); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.WriteString(strings.Repeat("x", 16*1024*1024) + "\n")
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("fixture write: %v/%v", writeErr, closeErr)
	}
	if _, _, err = cache.Search([]string{p}, match, "", "", false, 10); err == nil {
		t.Fatal("ASSERT_TRANSCRIPT_CACHE_MALFORMED_REJECTED: expected refresh failure")
	}

	canonical, err := canonicalTranscriptPath(p)
	if err != nil {
		t.Fatal(err)
	}
	var eventID string
	if err = cache.db.QueryRow(`SELECT event_id FROM transcript_search_events WHERE path = ?`, canonical).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	if eventID != "old" {
		t.Fatalf("ASSERT_TRANSCRIPT_CACHE_TRANSACTIONAL_REPLACEMENT: event=%q", eventID)
	}
}
