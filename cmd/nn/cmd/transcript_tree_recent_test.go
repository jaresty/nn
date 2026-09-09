package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHallwayRecentBeforePagination(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	out, e := execute("transcript", "tree", path, "--parent", "ROOT", "--order", "observed-recent", "--limit", "1", "--json")
	if e != nil {
		t.Fatalf("ASSERT_HALLWAY_RECENT: %v", e)
	}
	var p treeChildPage
	if e = json.Unmarshal([]byte(out), &p); e != nil {
		t.Fatal(e)
	}
	if len(p.Children) != 1 || p.Children[0].ID != "B" || p.TotalChildren != 5 {
		t.Fatalf("ASSERT_HALLWAY_RECENT: rank full population before limit: %s", out)
	}
	out, e = execute("transcript", "tree", path, "--parent", "ROOT", "--order", "observed-recent", "--selected", "A", "--limit", "1", "--json")
	if e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal([]byte(out), &p)
	if p.Children[0].ID != "A" {
		t.Fatal("ASSERT_HALLWAY_PIN: selected room not first")
	}
}

func TestHallwayRecentRetentionAndBinding(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	base := []string{"transcript", "tree", path, "--parent", "ROOT", "--order", "observed-recent", "--limit", "1", "--json"}
	out, e := execute(base...)
	if e != nil {
		t.Fatal(e)
	}
	var first treeChildPage
	_ = json.Unmarshal([]byte(out), &first)
	if first.NextCursor == "" {
		t.Fatal("missing cursor")
	}
	for _, extra := range [][]string{{"--selected", "C"}, {"--limit", "2"}, {"--parent", "B"}, {"--strict"}} {
		if _, e = execute(append(append([]string{}, base...), append(extra, "--cursor", first.NextCursor)...)...); e == nil {
			t.Fatalf("ASSERT_HALLWAY_BINDING: option mismatch accepted %v", extra)
		}
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	next, e := execute(append(base, "--cursor", first.NextCursor)...)
	if e != nil {
		t.Fatalf("ASSERT_HALLWAY_RETAINED: %v", e)
	}
	var p treeChildPage
	_ = json.Unmarshal([]byte(next), &p)
	if p.Children[0].ID != "C" || p.Snapshot != first.Snapshot {
		t.Fatalf("ASSERT_HALLWAY_RETAINED: wrong continuation: %s", next)
	}
}

func TestHallwayRecencyAuthorityAndTies(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.WriteString(`{"type":"message","agentId":"C","timestamp":"2099-01-01T00:00:00Z","message":{"role":"user","content":"not work"}}` + "\n" + `{"type":"message","agentId":"C","timestamp":"2026-01-01T00:06:00.000Z","message":{"role":"assistant","content":"tie"}}` + "\n")
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	out, e := execute("transcript", "tree", path, "--parent", "ROOT", "--order", "observed-recent", "--json")
	if e != nil {
		t.Fatal(e)
	}
	var p treeChildPage
	_ = json.Unmarshal([]byte(out), &p)
	for i, id := range []string{"B", "C", "A", "D", "U"} {
		if p.Children[i].ID != id {
			t.Fatalf("ASSERT_HALLWAY_WORK_AUTHORITY: %s", out)
		}
	}
	if p.Children[2].Recency.LastObservedAt != nil {
		t.Fatal("ASSERT_HALLWAY_WORK_AUTHORITY: lifecycle counted as work")
	}
	out, e = execute("transcript", "tree", path, "--parent", "ROOT", "--order", "canonical", "--limit", "1", "--json")
	if e != nil {
		t.Fatal(e)
	}
	_ = json.Unmarshal([]byte(out), &p)
	if p.Children[0].ID != "A" {
		t.Fatal("canonical compatibility lost")
	}
}

func TestHallwayCacheFailsClosed(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for _, mode := range []string{"missing", "corrupt", "expired"} {
		t.Run(mode, func(t *testing.T) {
			path := reviewFixture(t)
			p, e := buildOrderedTreeChildPage(path, "ROOT", "observed-recent", "", 1, "", false)
			if e != nil {
				t.Fatal(e)
			}
			dir, e := captureCacheDir()
			if e != nil {
				t.Fatal(e)
			}
			file := filepath.Join(dir, p.Snapshot+".hallway")
			switch mode {
			case "missing":
				e = os.Remove(file)
			case "corrupt":
				e = os.WriteFile(file, []byte("{}"), 0600)
			case "expired":
				past := time.Now().Add(-25 * time.Hour)
				e = os.Chtimes(file, past, past)
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = buildOrderedTreeChildPage(path, "ROOT", "observed-recent", "", 1, p.NextCursor, false); e == nil {
				t.Fatalf("ASSERT_HALLWAY_CACHE: %s accepted", mode)
			}
			if mode == "expired" {
				cleanupTranscriptCaptures()
				if _, e = os.Stat(file); !os.IsNotExist(e) {
					t.Fatalf("expired hallway not pruned: %v", e)
				}
			}
		})
	}
}

func TestHallwayRecentValidation(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	for _, flags := range [][]string{{"--order", "observed-recent"}, {"--parent", "ROOT", "--order", "bad"}, {"--parent", "ROOT", "--selected", "missing"}, {"--parent", "ROOT", "--selected", "ROOT"}} {
		if _, e := execute(append([]string{"transcript", "tree", path, "--json"}, flags...)...); e == nil {
			t.Fatalf("invalid flags accepted: %v", flags)
		}
	}
	out, e := execute("transcript", "tree", path, "--parent", "ROOT", "--order", "observed-recent", "--json")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out, `"order":"observed-recent"`) || !strings.Contains(out, `"last_observed_at":null`) {
		t.Fatalf("ASSERT_HALLWAY_DISCLOSURE: %s", out)
	}
}
