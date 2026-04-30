package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestTagsPlainOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	a.Tags = []string{"hooks", "architecture"}
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	b.Tags = []string{"hooks"}
	c := newTestNoteForCLI(note.GenerateID(), "Gamma", note.TypeConcept)
	c.Tags = []string{"architecture"}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, c)

	out, err := execute("tags")
	if err != nil {
		t.Fatalf("nn tags: %v", err)
	}
	if !strings.Contains(out, "hooks") {
		t.Errorf("output missing 'hooks': %q", out)
	}
	if !strings.Contains(out, "architecture") {
		t.Errorf("output missing 'architecture': %q", out)
	}
}

func TestTagsJSONOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept)
	a.Tags = []string{"hooks", "architecture"}
	b := newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeConcept)
	b.Tags = []string{"hooks"}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("tags", "--json")
	if err != nil {
		t.Fatalf("nn tags --json: %v", err)
	}
	var result []struct {
		Tag   string   `json:"tag"`
		Count int      `json:"count"`
		Notes []string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("nn tags --json: invalid JSON: %v\n%s", err, out)
	}
	counts := map[string]int{}
	for _, r := range result {
		counts[r.Tag] = r.Count
	}
	if counts["hooks"] != 2 {
		t.Errorf("hooks count: want 2, got %d", counts["hooks"])
	}
	if counts["architecture"] != 1 {
		t.Errorf("architecture count: want 1, got %d", counts["architecture"])
	}
}
