package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// searchMaterializedReference freezes the pre-optimization materialized algorithm.
// Keep this independent from the streaming implementation to detect semantic drift.
func searchMaterializedReference(files []string, query, filter, before string, raw bool, limit int) (transcriptSearchResult, error) {
	result := transcriptSearchResult{Matches: []transcriptSearchMatch{}}
	for _, path := range files {
		recs, err := readRecords(path)
		if err != nil {
			return result, err
		}
		session := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		for _, r := range recs {
			if r.Type == "session" && r.ID != "" {
				session = r.ID
				break
			}
		}
		for i, r := range recs {
			if !isPiEventRecord(r) || len(r.Message) == 0 {
				continue
			}
			owner := r.AgentID
			if owner == "" {
				owner = "ROOT"
			}
			if filter != "" && owner != filter {
				continue
			}
			if before != "" && r.Timestamp != "" && r.Timestamp >= before {
				continue
			}
			var msg struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			}
			if json.Unmarshal(r.Message, &msg) != nil {
				continue
			}
			text := string(r.Message)
			if !raw {
				text = meaningfulContent(msg.Role, msg.Content)
			}
			if !strings.Contains(strings.ToLower(text), strings.ToLower(query)) {
				continue
			}
			if len(result.Matches) == limit {
				result.Truncated = true
				continue
			}
			role := msg.Role
			if role == "" {
				role = r.Type
			}
			id := r.ID
			if id == "" {
				id = fmt.Sprintf("record:%d", i+1)
			}
			result.Matches = append(result.Matches, transcriptSearchMatch{session, owner, id, r.Timestamp, role, strings.TrimSpace(text), path})
		}
	}
	result.Returned = len(result.Matches)
	return result, nil
}

func TestTranscriptSearchStreamingParity(t *testing.T) {
	p := filepath.Join(t.TempDir(), "late.jsonl")
	lines := []string{
		"", "not json", `null`, `{}`, `{"type":"message","message":{"role":"assistant","content":"Needle K Σ İ"}}`,
		`{"type":"message","id":"invalid","timestamp":3,"message":{"role":"assistant","content":"needle"}}`,
		`{"type":"message","message":{"role":42,"content":"needle"}}`,
		`{"type":"message","message":{"role":"assistant","content":42}}`,
		`{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"needle second"}]}}`,
		`{"type":"message","agentId":"child","timestamp":"2026-09-09T00:00:00Z","message":{"role":"toolResult","content":"needle tool"}}`,
		`{"type":"message","agentId":"child","message":{"role":"assistant","content":"needle child"}}`,
		`{"type":"session","id":"late-header"}`, `{"type":"session","id":"not-first"}`,
	}
	writeTranscriptFile(t, p, strings.Join(lines, "\n"))
	other := filepath.Join(t.TempDir(), "no-header.output")
	writeTranscriptFile(t, other, strings.Join(lines[:11], "\n"))
	for _, raw := range []bool{false, true} {
		for _, q := range []string{"needle", "k", "σ", "i", "missing", ""} {
			for _, limit := range []int{1, 2, 50} {
				for _, filter := range []string{"", "ROOT", "child", "absent"} {
					for _, before := range []string{"", "2026-09-09T00:00:00Z"} {
						paths := []string{p, other}
						want, we := searchMaterializedReference(paths, q, filter, before, raw, limit)
						got, ge := searchTranscriptFiles(paths, q, filter, before, raw, limit)
						if we != nil || ge != nil || !reflect.DeepEqual(got, want) {
							t.Fatalf("ASSERT_SEARCH_PARITY raw=%v query=%q limit=%d filter=%s before=%s: got=%+v err=%v want=%+v err=%v", raw, q, limit, filter, before, got, ge, want, we)
						}
					}
				}
			}
		}
	}
	t.Log("ASSERT_SEARCH_PARITY: PASS")
}

func TestTranscriptSearchLateErrors(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.jsonl")
	bad := filepath.Join(dir, "bad.jsonl")
	text := `{"type":"message","message":{"role":"assistant","content":"needle"}}` + "\n"
	writeTranscriptFile(t, good, text+text)
	writeTranscriptFile(t, bad, text+text+strings.Repeat("x", 16*1024*1024)+"\n")
	for _, paths := range [][]string{{bad}, {good, bad}, {good, filepath.Join(dir, "missing")}} {
		want, we := searchMaterializedReference(paths, "needle", "", "", true, 1)
		got, ge := searchTranscriptFiles(paths, "needle", "", "", true, 1)
		if we == nil || ge == nil || we.Error() != ge.Error() || !reflect.DeepEqual(want, got) {
			t.Fatalf("ASSERT_SEARCH_LATE_ERROR: got=%+v/%v want=%+v/%v", got, ge, want, we)
		}
	}
	t.Log("ASSERT_SEARCH_LATE_ERROR: PASS")
}

func searchAllocationFixture(t testing.TB) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "large.jsonl")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	line := `{"type":"message","message":{"role":"assistant","content":"needle ` + strings.Repeat("x", 128*1024) + `"}}` + "\n"
	if _, err = f.WriteString(`{"type":"session","id":"fixed"}` + "\n"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 128; i++ {
		if _, err = f.WriteString(line); err != nil {
			t.Fatal(err)
		}
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTranscriptSearchBoundedAllocations(t *testing.T) {
	p := searchAllocationFixture(t)
	runtime.GC()
	var a, b runtime.MemStats
	runtime.ReadMemStats(&a)
	r, err := searchTranscriptFiles([]string{p}, "needle", "", "", true, 1)
	runtime.ReadMemStats(&b)
	if err != nil || r.Returned != 1 || !r.Truncated {
		t.Fatalf("fixture search: %+v/%v", r, err)
	}
	allocated := b.TotalAlloc - a.TotalAlloc
	// A generous 8 MiB envelope for a 16 MiB corpus and one 128 KiB result.
	// This guards bounded work after truncation, without timing assertions.
	if allocated > 8<<20 {
		t.Fatalf("ASSERT_SEARCH_BOUNDED_ALLOCATION: allocated=%d exceeds 8388608", allocated)
	}
	t.Logf("ASSERT_SEARCH_BOUNDED_ALLOCATION: PASS allocated=%d", allocated)
}

func BenchmarkTranscriptSearchBounded(b *testing.B) {
	p := searchAllocationFixture(b)
	for _, raw := range []bool{false, true} {
		b.Run(fmt.Sprintf("raw=%v", raw), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := searchTranscriptFiles([]string{p}, "needle", "", "", raw, 1); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
