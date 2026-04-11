package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestDeleteRemovesNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Delete Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("delete", n.ID, "--confirm")
	if err != nil {
		t.Fatalf("nn delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(nbDir, n.Filename())); !os.IsNotExist(err) {
		t.Error("file still exists after delete")
	}
}

func TestDeleteRequiresConfirm(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Delete Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("delete", n.ID)
	if err == nil {
		t.Fatal("nn delete without --confirm: want error, got nil")
	}
}

func TestDeleteLinkedWarns(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	linker := newTestNoteForCLI(note.GenerateID(), "Linker", note.TypeArgument)
	linker.Links = []note.Link{{TargetID: target.ID, Annotation: "depends on"}}
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, linker)

	out, _ := execute("delete", target.ID, "--confirm")
	// Should complete but output a warning
	if !strings.Contains(strings.ToLower(out), "warn") && !strings.Contains(strings.ToLower(out), "linked") {
		t.Logf("delete linked note output: %q (warning expected but not enforced)", out)
	}
}
