package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestBulkUpdateLinkRejectsMissingAndUnknownNewTypes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Type: "supports", Annotation: "original"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	for _, linkType := range []string{"", "invented"} {
		t.Run(linkType, func(t *testing.T) {
			if _, err := execute("bulk-update-link", src.ID, "--to", dst.ID, "--type", linkType); err == nil {
				t.Fatalf("bulk-update-link type %q: want error, got nil", linkType)
			}
		})
	}
}

func TestBulkUpdateLinkTypes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeConcept)
	src.Links = []note.Link{
		{TargetID: dst1.ID, Annotation: "narrows scope"},
		{TargetID: dst2.ID, Annotation: "opposes claim"},
	}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst1)
	writeNoteFile(t, nbDir, dst2)

	_, err := execute("bulk-update-link", src.ID,
		"--to", dst1.ID, "--type", "refines",
		"--to", dst2.ID, "--type", "contradicts",
	)
	if err != nil {
		t.Fatalf("nn bulk-update-link: %v", err)
	}

	out, _ := execute("show", src.ID)
	if !strings.Contains(out, "[refines]") {
		t.Errorf("refines type not set:\n%s", out)
	}
	if !strings.Contains(out, "[contradicts]") {
		t.Errorf("contradicts type not set:\n%s", out)
	}
	// Annotations must be preserved.
	if !strings.Contains(out, "narrows scope") {
		t.Errorf("annotation lost:\n%s", out)
	}
}

func TestBulkUpdateLinkRequiresTo(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	writeNoteFile(t, nbDir, src)

	_, err := execute("bulk-update-link", src.ID)
	if err == nil {
		t.Fatal("bulk-update-link with no --to: want error, got nil")
	}
}
