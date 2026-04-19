package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn show <id> --depth 1 returns the note and its directly linked notes.
func TestShowDepth1(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Note", note.TypeConcept)
	child := newTestNoteForCLI(note.GenerateID(), "Child Note", note.TypeConcept)
	grandchild := newTestNoteForCLI(note.GenerateID(), "Grandchild Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "links to child"}}
	child.Links = []note.Link{{TargetID: grandchild.ID, Annotation: "links to grandchild"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)
	writeNoteFile(t, nbDir, grandchild)

	out, err := execute("show", root.ID, "--depth", "1")
	if err != nil {
		t.Fatalf("nn show --depth 1: %v", err)
	}
	if !strings.Contains(out, "Root Note") {
		t.Errorf("root note missing from --depth 1 output:\n%s", out)
	}
	if !strings.Contains(out, "Child Note") {
		t.Errorf("child note missing from --depth 1 output:\n%s", out)
	}
	if strings.Contains(out, "Grandchild Note") {
		t.Errorf("grandchild note should not appear at --depth 1:\n%s", out)
	}
}

// Assertion: nn show <id> --depth 2 returns note + depth-1 + depth-2 linked notes.
func TestShowDepth2(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Note", note.TypeConcept)
	child := newTestNoteForCLI(note.GenerateID(), "Child Note", note.TypeConcept)
	grandchild := newTestNoteForCLI(note.GenerateID(), "Grandchild Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "links to child"}}
	child.Links = []note.Link{{TargetID: grandchild.ID, Annotation: "links to grandchild"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)
	writeNoteFile(t, nbDir, grandchild)

	out, err := execute("show", root.ID, "--depth", "2")
	if err != nil {
		t.Fatalf("nn show --depth 2: %v", err)
	}
	if !strings.Contains(out, "Root Note") {
		t.Errorf("root note missing from --depth 2 output:\n%s", out)
	}
	if !strings.Contains(out, "Child Note") {
		t.Errorf("child note missing from --depth 2 output:\n%s", out)
	}
	if !strings.Contains(out, "Grandchild Note") {
		t.Errorf("grandchild note missing from --depth 2 output:\n%s", out)
	}
}

// Assertion: nn show --depth separates notes with ---
func TestShowDepthSeparator(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Note", note.TypeConcept)
	child := newTestNoteForCLI(note.GenerateID(), "Child Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "links to child"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("show", root.ID, "--depth", "1")
	if err != nil {
		t.Fatalf("nn show --depth 1: %v", err)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("separator --- missing from --depth output:\n%s", out)
	}
}

// Assertion: nn show --depth --json returns array of note objects in BFS order.
func TestShowDepthJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Root Note", note.TypeConcept)
	child := newTestNoteForCLI(note.GenerateID(), "Child Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "links to child"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("show", root.ID, "--depth", "1", "--json")
	if err != nil {
		t.Fatalf("nn show --depth --json: %v", err)
	}
	type depthNoteJSON struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Depth int    `json:"depth"`
	}
	var results []depthNoteJSON
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("nn show --depth --json output is not valid JSON: %v\n%s", err, out)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 notes, got %d:\n%s", len(results), out)
	}
	if results[0].ID != root.ID {
		t.Errorf("first result should be root note, got %s", results[0].ID)
	}
	if results[0].Depth != 0 {
		t.Errorf("root note should have depth 0, got %d", results[0].Depth)
	}
	if results[1].ID != child.ID {
		t.Errorf("second result should be child note, got %s", results[1].ID)
	}
	if results[1].Depth != 1 {
		t.Errorf("child note should have depth 1, got %d", results[1].Depth)
	}
}
