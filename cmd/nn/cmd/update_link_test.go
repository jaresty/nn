package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateLinkType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("update-link", src.ID, dst.ID, "--type", "refines")
	if err != nil {
		t.Fatalf("nn update-link --type: %v", err)
	}
	out, _ := execute("show", src.ID)
	if !strings.Contains(out, "[refines]") {
		t.Errorf("type not updated in note:\n%s", out)
	}
	if !strings.Contains(out, "relates to") {
		t.Errorf("annotation lost after update-link:\n%s", out)
	}
}

func TestUpdateLinkAnnotation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "old annotation"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("update-link", src.ID, dst.ID, "--annotation", "new annotation")
	if err != nil {
		t.Fatalf("nn update-link --annotation: %v", err)
	}
	out, _ := execute("show", src.ID)
	if !strings.Contains(out, "new annotation") {
		t.Errorf("annotation not updated:\n%s", out)
	}
	if strings.Contains(out, "old annotation") {
		t.Errorf("old annotation still present:\n%s", out)
	}
}

func TestUpdateLinkRequiresFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("update-link", src.ID, dst.ID)
	if err == nil {
		t.Fatal("nn update-link with no flags: want error, got nil")
	}
}

func TestUpdateLinkNotFoundErrors(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	// No link between src and dst.
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("update-link", src.ID, dst.ID, "--type", "refines")
	if err == nil {
		t.Fatal("nn update-link on non-existent link: want error, got nil")
	}
}
