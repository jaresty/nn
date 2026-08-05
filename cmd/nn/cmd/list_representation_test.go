package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// properties [1],[2a],[2b],[3a],[3b]: representation field accessible via --fields
func TestListFieldsRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Rep note", note.TypeModel)
	n.Representation = "ontology"
	n.Body = "body text"
	writeNoteFile(t, nbDir, n)

	// property [1]: --fields representation must not error
	// property [2a],[2b]: plain list JSON includes representation
	listOut, err := execute("list", "--json", "--fields", "id,representation")
	if err != nil {
		t.Fatalf("nn list --fields representation: %v", err)
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(listOut)), &results); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, listOut)
	}

	found := false
	for _, r := range results {
		if r["id"] == n.ID {
			found = true
			if r["representation"] != "ontology" {
				t.Errorf("expected representation=ontology, got %v", r["representation"])
			}
		}
	}
	if !found {
		t.Errorf("note %s not found in list output", n.ID)
	}

	// property [3a],[3b]: search JSON includes representation
	searchOut, err := execute("list", "--search", "body text", "--json", "--fields", "id,representation")
	if err != nil {
		t.Fatalf("nn list --search --fields representation: %v", err)
	}

	var searchResults []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(searchOut)), &searchResults); err != nil {
		t.Fatalf("parse search JSON: %v\noutput: %s", err, searchOut)
	}

	found = false
	for _, r := range searchResults {
		if r["id"] == n.ID {
			found = true
			if r["representation"] != "ontology" {
				t.Errorf("search: expected representation=ontology, got %v", r["representation"])
			}
		}
	}
	if !found {
		t.Errorf("note %s not found in search output", n.ID)
	}
}
