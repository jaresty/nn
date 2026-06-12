package cmd

import (
	"encoding/json"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn list --search --json includes applies_when field for protocol notes.
func TestListSearchJSONIncludesAppliesWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Protocol applies_when test", note.TypeProtocol)
	n.AppliesWhen = "before any external lookup"
	n.Status = note.StatusPermanent
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "applies_when test", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	var found map[string]interface{}
	for _, r := range results {
		if r["id"] == n.ID {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatalf("note %q not found in search results", n.ID)
	}
	aw, ok := found["applies_when"]
	if !ok {
		t.Errorf("applies_when field missing from nn list --search --json output")
	} else if aw != "before any external lookup" {
		t.Errorf("applies_when = %q, want %q", aw, "before any external lookup")
	}
}

// Assertion: nn list --json (non-search) includes applies_when field.
func TestListJSONIncludesAppliesWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Protocol plain list test", note.TypeProtocol)
	n.AppliesWhen = "always"
	n.Status = note.StatusPermanent
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	var found map[string]interface{}
	for _, r := range results {
		if r["id"] == n.ID {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatalf("note %q not found in list output", n.ID)
	}
	aw, ok := found["applies_when"]
	if !ok {
		t.Errorf("applies_when field missing from nn list --json output")
	} else if aw != "always" {
		t.Errorf("applies_when = %q, want %q", aw, "always")
	}
}
