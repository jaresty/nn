package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestLinkAddsAnnotation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("link", src.ID, dst.ID, "--annotation", "builds on this", "--type", "extends")
	if err != nil {
		t.Fatalf("nn link: %v", err)
	}

	out, _ := execute("show", src.ID)
	if !strings.Contains(out, dst.ID) {
		t.Errorf("source note does not contain link to target: %q", out)
	}
	if !strings.Contains(out, "builds on this") {
		t.Errorf("source note does not contain annotation: %q", out)
	}
}

func TestLinkRequiresAnnotation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("link", src.ID, dst.ID)
	if err == nil {
		t.Fatal("nn link without --annotation: want error, got nil")
	}
}

// Assertion: nn link without --type returns an error.
func TestLinkRequiresType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("link", src.ID, dst.ID, "--annotation", "builds on this")
	if err == nil {
		t.Fatal("nn link without --type: want error, got nil")
	}
}

func TestUnlinkRemovesLink(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeArgument)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("unlink", src.ID, dst.ID)
	if err != nil {
		t.Fatalf("nn unlink: %v", err)
	}

	out, _ := execute("show", src.ID)
	if strings.Contains(out, dst.ID) {
		t.Errorf("source note still contains link after unlink: %q", out)
	}
}
