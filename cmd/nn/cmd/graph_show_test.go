package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func makeLinkedNotes(t *testing.T, nbDir string) (root, child1, child2 *note.Note) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)

	root = newTestNoteForCLI(note.GenerateID(), "Root note", note.TypeModel)
	root.Created, root.Modified = now, now
	root.Body = "root body"

	child1 = newTestNoteForCLI(note.GenerateID(), "Child one", note.TypeConcept)
	child1.Created, child1.Modified = now, now
	child1.Body = "child one body"

	child2 = newTestNoteForCLI(note.GenerateID(), "Child two", note.TypeArgument)
	child2.Created, child2.Modified = now, now
	child2.Body = "child two body"

	root.Links = []note.Link{
		{TargetID: child1.ID, Type: "supports", Annotation: "first link"},
		{TargetID: child2.ID, Type: "extends", Annotation: ""},
	}

	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child1)
	writeNoteFile(t, nbDir, child2)
	return
}

// property [1]: first line is focus note
// property [2a]: children are indented
// property [2b]: children show link type
func TestGraphShow_TreeHierarchy(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root, child1, child2 := makeLinkedNotes(t, nbDir)

	out, err := execute("graph", "show", "--focus", root.ID, "--depth", "2")
	if err != nil {
		t.Fatalf("graph show: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")

	// property [1]: first line contains root ID and title
	if len(lines) == 0 || !strings.Contains(lines[0], root.ID) {
		t.Errorf("property [1]: expected first line to contain root ID %s, got:\n%s", root.ID, out)
	}

	// property [2a]: indented lines exist (children indented)
	hasIndent := false
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
			hasIndent = true
			break
		}
	}
	if !hasIndent {
		t.Errorf("property [2a]: expected indented child lines, got:\n%s", out)
	}

	// property [2b]: link type shown
	if !strings.Contains(out, "supports") && !strings.Contains(out, "extends") {
		t.Errorf("property [2b]: expected link types in output, got:\n%s", out)
	}

	// children appear in output
	if !strings.Contains(out, child1.ID) {
		t.Errorf("expected child1 %s in output:\n%s", child1.ID, out)
	}
	if !strings.Contains(out, child2.ID) {
		t.Errorf("expected child2 %s in output:\n%s", child2.ID, out)
	}
}

// property [3]: cycle guard — note appearing in multiple paths only rendered once
func TestGraphShow_CycleGuard(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	a := newTestNoteForCLI(note.GenerateID(), "Note A", note.TypeModel)
	a.Created, a.Modified = now, now
	b := newTestNoteForCLI(note.GenerateID(), "Note B", note.TypeConcept)
	b.Created, b.Modified = now, now

	// A → B → A (cycle)
	a.Links = []note.Link{{TargetID: b.ID, Type: "supports", Annotation: ""}}
	b.Links = []note.Link{{TargetID: a.ID, Type: "supports", Annotation: ""}}

	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("graph", "show", "--focus", a.ID, "--depth", "3")
	if err != nil {
		t.Fatalf("graph show cycle: %v", err)
	}

	// A should appear exactly once as root
	count := strings.Count(out, a.ID)
	if count != 1 {
		t.Errorf("property [3]: expected note A to appear once, got %d times:\n%s", count, out)
	}
}

// property [4]: no --focus falls back to flat list (no regression)
func TestGraphShow_NoFocusFlatList(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root, _, _ := makeLinkedNotes(t, nbDir)

	out, err := execute("graph", "show")
	if err != nil {
		t.Fatalf("graph show no focus: %v", err)
	}

	// Should contain root ID without tree structure
	if !strings.Contains(out, root.ID) {
		t.Errorf("property [4]: expected root ID in flat output, got:\n%s", out)
	}
}
