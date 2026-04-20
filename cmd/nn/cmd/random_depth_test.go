package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn random --depth 1 returns root note and its directly linked notes.
func TestRandomDepth(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Note", note.TypeConcept)
	child := newTestNoteForCLI(note.GenerateID(), "Child Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "links to child", Type: "extends"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	// Run several times — the only note with a link is root, so it must always be selected.
	for i := 0; i < 5; i++ {
		out, err := execute("random", "--status", "draft", "--depth", "1")
		if err != nil {
			t.Fatalf("nn random --depth 1: %v", err)
		}
		if strings.Contains(out, "Root Note") {
			// Root was selected; child must also appear.
			if !strings.Contains(out, "Child Note") {
				t.Errorf("child note missing from --depth 1 output when root selected:\n%s", out)
			}
			if !strings.Contains(out, "---") {
				t.Errorf("separator missing from --depth output:\n%s", out)
			}
		}
		// If child was selected, only child appears (it has no outgoing links at depth 1).
	}
}

// Assertion: nn random --depth 1 --json returns array with depth field.
func TestRandomDepthJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Note", note.TypeConcept)
	child := newTestNoteForCLI(note.GenerateID(), "Child Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "links to child", Type: "extends"}}
	// Only write root so it's always selected.
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	// Force selection of root by filtering to a unique tag.
	root.Tags = []string{"unique-root-tag"}
	writeNoteFile(t, nbDir, root)

	out, err := execute("random", "--tag", "unique-root-tag", "--depth", "1", "--json")
	if err != nil {
		t.Fatalf("nn random --depth --json: %v", err)
	}
	type depthNote struct {
		ID    string `json:"id"`
		Depth int    `json:"depth"`
	}
	var results []depthNote
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) < 1 {
		t.Fatalf("expected at least 1 result, got 0:\n%s", out)
	}
	if results[0].Depth != 0 {
		t.Errorf("first result should have depth 0, got %d", results[0].Depth)
	}
}
