package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// Assertion: nn bulk-new creates all notes in the batch.
func TestBulkNewCreatesNotes(t *testing.T) {
	_, execute := setupNotebook(t)

	input := `[
		{"title": "Note Alpha", "type": "concept", "content": "body a"},
		{"title": "Note Beta",  "type": "argument", "content": "body b"}
	]`
	out, err := execute("bulk-new", "--json", input)
	if err != nil {
		t.Fatalf("nn bulk-new: %v", err)
	}
	// Output should list created IDs.
	if !strings.Contains(out, "created") {
		t.Errorf("expected 'created' in output:\n%s", out)
	}

	// Verify both notes exist.
	listOut, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list: %v", err)
	}
	var notes []map[string]interface{}
	if err := json.Unmarshal([]byte(listOut), &notes); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	titles := make(map[string]bool)
	for _, n := range notes {
		titles[n["title"].(string)] = true
	}
	if !titles["Note Alpha"] {
		t.Errorf("Note Alpha not found after bulk-new")
	}
	if !titles["Note Beta"] {
		t.Errorf("Note Beta not found after bulk-new")
	}
}

// Assertion: inline links between batch notes are created.
func TestBulkNewCreatesInlineLinks(t *testing.T) {
	_, execute := setupNotebook(t)

	input := `[
		{"title": "Source", "type": "concept", "content": "src"},
		{"title": "Target", "type": "concept", "content": "tgt",
		 "links": [{"ref": 0, "annotation": "extends it", "type": "extends"}]}
	]`
	_, err := execute("bulk-new", "--json", input)
	if err != nil {
		t.Fatalf("nn bulk-new with links: %v", err)
	}

	// Find Source note ID.
	listOut, _ := execute("list", "--json", "--rich")
	var notes []map[string]interface{}
	json.Unmarshal([]byte(listOut), &notes)

	var sourceID string
	for _, n := range notes {
		if n["title"].(string) == "Source" {
			sourceID = n["id"].(string)
			break
		}
	}
	if sourceID == "" {
		t.Fatal("Source note not found")
	}

	// Verify Target links to Source.
	linksOut, err := execute("backlinks", sourceID)
	if err != nil {
		t.Fatalf("nn backlinks: %v", err)
	}
	if !strings.Contains(linksOut, "Target") {
		t.Errorf("expected Target to link to Source:\n%s", linksOut)
	}
}
