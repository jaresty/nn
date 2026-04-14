package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Old Title", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--title", "New Title", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --title: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "New Title") {
		t.Errorf("title not updated:\n%s", out)
	}
}

func TestUpdateContent(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	n.Body = "original body"
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--content", "replaced body", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --content: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "replaced body") {
		t.Errorf("content not replaced:\n%s", out)
	}
	if strings.Contains(out, "original body") {
		t.Errorf("old content still present:\n%s", out)
	}
}

func TestUpdateAppend(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	n.Body = "original body"
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--append", "appended text", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --append: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "original body") {
		t.Errorf("original body missing after append:\n%s", out)
	}
	if !strings.Contains(out, "appended text") {
		t.Errorf("appended text missing:\n%s", out)
	}
}

func TestUpdateTags(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--tags", "foo,bar", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --tags: %v", err)
	}
	out, _ := execute("list", "--tag", "foo", "--json")
	if !strings.Contains(out, n.ID) {
		t.Errorf("note not found by updated tag:\n%s", out)
	}
}

func TestUpdateContentAndAppendMutuallyExclusive(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--content", "new", "--append", "more", "--no-edit")
	if err == nil {
		t.Fatal("nn update --content --append: want error, got nil")
	}
}

func TestUpdateRequiresFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--no-edit")
	if err == nil {
		t.Fatal("nn update with no change flags: want error, got nil")
	}
}
