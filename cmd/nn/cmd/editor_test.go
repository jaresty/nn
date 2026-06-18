package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestNewNoteOpensEditorWhenTTY(t *testing.T) {
	_, execute := setupNotebook(t)
	origTTY := isTTYFn
	origEditor := openEditorFn
	t.Cleanup(func() {
		isTTYFn = origTTY
		openEditorFn = origEditor
	})
	isTTYFn = func() bool { return true }
	openEditorFn = func(initial string) (string, error) {
		return "editor-written body", nil
	}

	out, err := execute("new", "--title", "Editor Note", "--type", "concept")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.TrimSpace(strings.TrimPrefix(strings.Split(out, "\n")[0], "created "))
	showOut, _ := execute("show", id)
	if !strings.Contains(showOut, "editor-written body") {
		t.Errorf("expected editor-written body in note, got:\n%s", showOut)
	}
}

func TestNewNoteSkipsEditorWithNoEdit(t *testing.T) {
	_, execute := setupNotebook(t)
	origTTY := isTTYFn
	origEditor := openEditorFn
	t.Cleanup(func() {
		isTTYFn = origTTY
		openEditorFn = origEditor
	})
	isTTYFn = func() bool { return true }
	editorCalled := false
	openEditorFn = func(initial string) (string, error) {
		editorCalled = true
		return initial, nil
	}

	_, err := execute("new", "--title", "No Editor Note", "--type", "concept", "--no-edit", "--content", "static body")
	if err != nil {
		t.Fatalf("nn new --no-edit: %v", err)
	}
	if editorCalled {
		t.Errorf("openEditorFn was called despite --no-edit")
	}
}

func TestUpdateOpensEditorWhenTTY(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	origTTY := isTTYFn
	origEditor := openEditorFn
	t.Cleanup(func() {
		isTTYFn = origTTY
		openEditorFn = origEditor
	})
	isTTYFn = func() bool { return true }
	openEditorFn = func(initial string) (string, error) {
		return "editor-updated body", nil
	}

	n := newTestNoteForCLI(note.GenerateID(), "Editable Note", note.TypeConcept)
	n.Body = "original"
	writeNoteFile(t, nbDir, n)
	commitNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--since", sinceFor(n))
	if err != nil {
		t.Fatalf("nn update: %v", err)
	}
	showOut, _ := execute("show", n.ID)
	if !strings.Contains(showOut, "editor-updated body") {
		t.Errorf("expected editor-updated body in note, got:\n%s", showOut)
	}
}

func TestUpdateSkipsEditorWithNoEdit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	origTTY := isTTYFn
	origEditor := openEditorFn
	t.Cleanup(func() {
		isTTYFn = origTTY
		openEditorFn = origEditor
	})
	isTTYFn = func() bool { return true }
	editorCalled := false
	openEditorFn = func(initial string) (string, error) {
		editorCalled = true
		return initial, nil
	}

	n := newTestNoteForCLI(note.GenerateID(), "Static Note", note.TypeConcept)
	n.Body = "original"
	writeNoteFile(t, nbDir, n)
	commitNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--since", sinceFor(n), "--no-edit", "--content", "direct body")
	if err != nil {
		t.Fatalf("nn update --no-edit: %v", err)
	}
	if editorCalled {
		t.Errorf("openEditorFn was called despite --no-edit")
	}
}

func TestEditorReceivesInitialBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	origTTY := isTTYFn
	origEditor := openEditorFn
	t.Cleanup(func() {
		isTTYFn = origTTY
		openEditorFn = origEditor
	})
	isTTYFn = func() bool { return true }
	var receivedInitial string
	openEditorFn = func(initial string) (string, error) {
		receivedInitial = initial
		return initial, nil
	}

	n := newTestNoteForCLI(note.GenerateID(), "Seed Note", note.TypeConcept)
	n.Body = "seed content"
	writeNoteFile(t, nbDir, n)
	commitNoteFile(t, nbDir, n)

	execute("update", n.ID, "--since", sinceFor(n))
	if receivedInitial != "seed content" {
		t.Errorf("editor not seeded with note body: got %q", receivedInitial)
	}
}
