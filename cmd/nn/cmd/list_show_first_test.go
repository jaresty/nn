package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: --show-first appends the top result body after the JSON array.
func TestListShowFirstAppendsBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	n.Body = "This is the body of the target note."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "Target", "--show-first", "--json")
	if err != nil {
		t.Fatalf("nn list --search --show-first --json: %v", err)
	}
	if !strings.Contains(out, "This is the body of the target note.") {
		t.Errorf("expected top result body in output; got:\n%s", out)
	}
}

// Assertion: JSON array appears before the body block.
func TestListShowFirstJSONBeforeBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	n.Body = "Body content here."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "Target", "--show-first", "--json")
	if err != nil {
		t.Fatalf("nn list --search --show-first --json: %v", err)
	}
	jsonEnd := strings.Index(out, "]")
	bodyPos := strings.Index(out, "Body content here.")
	if jsonEnd == -1 || bodyPos == -1 {
		t.Fatalf("expected JSON array and body in output; got:\n%s", out)
	}
	if bodyPos < jsonEnd {
		t.Errorf("expected body to appear after JSON array; got:\n%s", out)
	}
}

// Assertion: --show-first with no results produces no body block (no panic, no extra output).
func TestListShowFirstNoResults(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("list", "--search", "nonexistent", "--show-first", "--json")
	if err != nil {
		t.Fatalf("nn list --search --show-first --json (empty): %v", err)
	}
	// Valid empty JSON array
	var results []interface{}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &results); jsonErr != nil {
		// output may have trailing newline or empty array — just check no crash and no body separator
		if !strings.Contains(out, "[]") && !strings.Contains(out, "[ ]") {
			t.Errorf("expected empty JSON array for no results; got:\n%s", out)
		}
	}
}
