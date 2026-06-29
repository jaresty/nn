package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestListFieldsRequiresJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))

	_, err := execute("list", "--fields", "id,title")
	if err == nil {
		t.Fatal("expected error when --fields used without --json, got nil")
	}
	if !strings.Contains(err.Error(), "--fields requires --json") {
		t.Errorf("error should mention '--fields requires --json', got: %v", err)
	}
}

func TestListFieldsUnknownErrors(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))

	_, err := execute("list", "--json", "--fields", "id,bogus")
	if err == nil {
		t.Fatal("expected error for unknown field 'bogus', got nil")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should name the unknown field 'bogus', got: %v", err)
	}
}

func TestListFieldsProjects(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))

	out, err := execute("list", "--json", "--fields", "id,title")
	if err != nil {
		t.Fatalf("nn list --json --fields id,title: %v", err)
	}
	var results []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &results); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	for _, r := range results {
		if _, ok := r["id"]; !ok {
			t.Errorf("result missing 'id': %v", r)
		}
		if _, ok := r["title"]; !ok {
			t.Errorf("result missing 'title': %v", r)
		}
		if _, ok := r["type"]; ok {
			t.Errorf("result should not contain 'type': %v", r)
		}
		if _, ok := r["status"]; ok {
			t.Errorf("result should not contain 'status': %v", r)
		}
	}
}

func TestListFieldsPassthrough(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))

	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list --json: %v", err)
	}
	var results []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &results); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if _, ok := results[0]["id"]; !ok {
		t.Errorf("passthrough result missing 'id'")
	}
	if _, ok := results[0]["type"]; !ok {
		t.Errorf("passthrough result missing 'type'")
	}
}
