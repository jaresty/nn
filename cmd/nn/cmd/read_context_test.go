package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type readContextTestResponse struct {
	Snapshot      string `json:"snapshot"`
	Source        string `json:"source"`
	Chunk         int    `json:"chunk"`
	Chunks        int    `json:"chunks"`
	PreviousChunk int    `json:"previous_chunk"`
	NextChunk     int    `json:"next_chunk"`
	Heading       string `json:"heading"`
	LineStart     int    `json:"line_start"`
	LineEnd       int    `json:"line_end"`
	Passage       string `json:"passage"`
	Coverage      string `json:"coverage"`
	Matches       []struct {
		ID           string  `json:"id"`
		Title        string  `json:"title"`
		Score        float64 `json:"score"`
		MatchReason  string  `json:"match_reason"`
		Relationship string  `json:"relationship"`
	} `json:"matches"`
}

func TestReadContextChunksMarkdownAndRanksGeneratedRelevance(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	relevant := newTestNoteForCLI("20990101000000-0001", "Cache invalidation policy", note.TypeConcept)
	relevant.Body = "Cache invalidation requires deterministic keys and explicit expiry."
	writeNoteFile(t, nbDir, relevant)
	other := newTestNoteForCLI("20990101000000-0002", "Garden layout", note.TypeConcept)
	other.Body = "Tomatoes grow beside basil."
	writeNoteFile(t, nbDir, other)

	path := filepath.Join(t.TempDir(), "guide.md")
	content := "# Introduction\n\nCache invalidation needs deterministic keys.\n\n" + strings.Repeat("More cache expiry detail. ", 20) + "\n\n# Operations\n\nRotate keys carefully.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("read-context", path, "--chunk", "1", "--chunk-bytes", "256")
	if err != nil {
		t.Fatalf("read-context chunk 1: %v", err)
	}
	if len(out) > readContextPageMaxBytes {
		t.Fatalf("encoded response = %d bytes, limit %d", len(out), readContextPageMaxBytes)
	}
	var first readContextTestResponse
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatalf("chunk 1 JSON: %v\n%s", err, out)
	}
	if len(first.Snapshot) != 64 || first.Chunk != 1 || first.Chunks < 2 || first.NextChunk != 2 {
		t.Fatalf("chunk metadata = %+v", first)
	}
	if first.Heading != "Introduction" || first.LineStart != 1 || !strings.Contains(first.Passage, "Cache invalidation") {
		t.Fatalf("passage metadata/content = %+v", first)
	}
	if first.Coverage == "" || len(first.Matches) == 0 {
		t.Fatalf("coverage/matches missing: %+v", first)
	}
	if first.Matches[0].ID != relevant.ID || first.Matches[0].Relationship != "generated-relevance" || first.Matches[0].MatchReason == "" {
		t.Fatalf("top match = %+v", first.Matches[0])
	}

	if _, err := execute("read-context", path, "--chunk", "2", "--chunk-bytes", "256"); err == nil || !strings.Contains(err.Error(), "--snapshot is required") {
		t.Fatalf("chunk 2 without snapshot error = %v", err)
	}
	out, err = execute("read-context", path, "--chunk", "2", "--chunk-bytes", "256", "--snapshot", first.Snapshot)
	if err != nil {
		t.Fatalf("read-context chunk 2: %v", err)
	}
	var second readContextTestResponse
	if err := json.Unmarshal([]byte(out), &second); err != nil {
		t.Fatal(err)
	}
	if second.Snapshot != first.Snapshot || second.PreviousChunk != 1 || second.Chunk != 2 {
		t.Fatalf("chunk 2 metadata = %+v", second)
	}
}

func TestReadContextRejectsStaleSnapshotAndUnsupportedInput(t *testing.T) {
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("bounded passage text\n\n", 40)), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := execute("read-context", path, "--chunk-bytes", "256")
	if err != nil {
		t.Fatal(err)
	}
	var first readContextTestResponse
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed\n"+strings.Repeat("bounded passage text\n\n", 40)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("read-context", path, "--chunk", "2", "--chunk-bytes", "256", "--snapshot", first.Snapshot); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
		t.Fatalf("stale snapshot error = %v", err)
	}

	unsupported := filepath.Join(dir, "notes.pdf")
	if err := os.WriteFile(unsupported, []byte("not a pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("read-context", unsupported); err == nil || !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("unsupported extension error = %v", err)
	}
}

func TestReadContextRejectsInvalidFlagsAndUTF8(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(path, []byte("# Notes\n\ntext"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"read-context", path, "--chunk", "0"},
		{"read-context", path, "--chunk-bytes", "255"},
		{"read-context", path, "--matches", "21"},
		{"read-context", path, "--snapshot", strings.Repeat("a", 64)},
	} {
		if _, err := execute(args...); err == nil {
			t.Fatalf("expected rejection for %v", args)
		}
	}
	bad := filepath.Join(t.TempDir(), "bad.txt")
	if err := os.WriteFile(bad, []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("read-context", bad); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}

func TestReadContextSupportedFormatsAreDeterministicAndSourceOrdered(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, ext := range []string{".md", ".markdown", ".txt"} {
		t.Run(ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "source"+ext)
			content := "# First\n\none\n\n# Second\n\ntwo\n"
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			first, err := execute("read-context", path, "--matches", "0")
			if err != nil {
				t.Fatal(err)
			}
			second, err := execute("read-context", path, "--matches", "0")
			if err != nil {
				t.Fatal(err)
			}
			if first != second {
				t.Fatalf("nondeterministic output for %s", ext)
			}
			if len(first) > readContextPageMaxBytes {
				t.Fatalf("encoded response = %d bytes, limit %d", len(first), readContextPageMaxBytes)
			}
			var got readContextTestResponse
			if err := json.Unmarshal([]byte(first), &got); err != nil {
				t.Fatal(err)
			}
			if got.LineStart != 1 || got.LineEnd < got.LineStart || got.Passage == "" {
				t.Fatalf("source-order line metadata = %+v", got)
			}
			if ext != ".txt" && got.Heading != "First" {
				t.Fatalf("heading = %q, want First", got.Heading)
			}
		})
	}
}

func TestReadContextSnapshotBindsChunkingOptions(t *testing.T) {
	_, execute := setupNotebook(t)
	path := filepath.Join(t.TempDir(), "source.md")
	if err := os.WriteFile(path, []byte(strings.Repeat("# Heading\n\nparagraph text\n\n", 40)), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := execute("read-context", path, "--chunk", "1", "--chunk-bytes", "256", "--matches", "0")
	if err != nil {
		t.Fatal(err)
	}
	var first readContextTestResponse
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("read-context", path, "--chunk", "2", "--chunk-bytes", "512", "--snapshot", first.Snapshot, "--matches", "0"); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
		t.Fatalf("changed chunking options error = %v", err)
	}
}

func TestReadContextBoundsAndLabelsGeneratedRelevance(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	for i, title := range []string{"Alpha context", "Alpha policy", "Alpha operations"} {
		n := newTestNoteForCLI("20990101000000-000"+string(rune('1'+i)), title, note.TypeConcept)
		n.Body = "alpha shared relevance"
		writeNoteFile(t, nbDir, n)
	}
	path := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(path, []byte("alpha shared relevance"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := execute("read-context", path, "--matches", "2")
	if err != nil {
		t.Fatal(err)
	}
	var got readContextTestResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Matches) > 2 {
		t.Fatalf("matches = %d, want at most 2", len(got.Matches))
	}
	if !strings.Contains(got.Coverage, "not exhaustive") || !strings.Contains(got.Coverage, "not a stored notebook relationship") {
		t.Fatalf("coverage = %q", got.Coverage)
	}
	for _, match := range got.Matches {
		if match.Relationship != "generated-relevance" {
			t.Fatalf("relationship = %q", match.Relationship)
		}
	}
}

func TestReadTogetherRegistrationDocumentationAndStateBoundaries(t *testing.T) {
	root := NewRootCmdForTest("")
	if command, _, err := root.Find([]string{"read-context"}); err != nil || command.Name() != "read-context" {
		t.Fatalf("read-context registration: command=%v err=%v", command, err)
	}

	files := map[string][]string{
		"../../../README.md": {
			"nn read-context", "48,000", "generated-relevance", "not a stored graph relationship", "without persisting reading state or creating links automatically",
		},
		"../../../skills/nn-guide/SKILL.md": {
			"nn read-context", "generated-relevance", "read-together", "not a stored", "non-exhaustive",
		},
		"show.go": {
			"**nn read-context**", "generated-relevance", "not a stored", "non-exhaustive",
		},
		"../../../skills/nn-navigate/SKILL.md": {
			"`read-together`", "document passages, generated relevance, and stored graph truth remain separate", "only an explicit note excursion enters graph navigation",
		},
		"../../../skills/nn-navigate/references/read-together.md": {
			"generated match", "not stored links", "graph_excursion_origin", "restores the exact saved document frame", "never mutates notebook truth automatically",
		},
		"../../../skills/nn-navigate/references/state.md": {
			"document passage is never a graph focus or graph-history entry", "state domain separate from graph history", "never persist",
		},
	}
	for path, required := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range required {
			if !strings.Contains(string(content), text) {
				t.Errorf("%s missing %q", path, text)
			}
		}
	}
}
