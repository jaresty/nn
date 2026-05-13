package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestSearchExcerptPlain(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept)
	n.Body = "The quick brown fox jumps over the lazy dog near the riverbank."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "riverbank")
	if err != nil {
		t.Fatalf("nn list --search: %v", err)
	}
	if !strings.Contains(out, "riverbank") {
		t.Errorf("plain search output missing excerpt containing matched term: %q", out)
	}
	// Excerpt should appear on a second line indented under the result
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (result + excerpt), got %d: %q", len(lines), out)
	}
	excerptLine := lines[1]
	if !strings.Contains(excerptLine, "riverbank") {
		t.Errorf("second line does not contain matched term: %q", excerptLine)
	}
}

func TestSearchExcerptJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept)
	n.Body = "The quick brown fox jumps over the lazy dog near the riverbank."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "riverbank", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	excerpt, ok := results[0]["excerpt"].(string)
	if !ok {
		t.Errorf("'excerpt' field missing or not a string in JSON result: %v", results[0])
	}
	if !strings.Contains(excerpt, "riverbank") {
		t.Errorf("excerpt does not contain matched term: %q", excerpt)
	}
}
