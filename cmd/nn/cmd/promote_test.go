package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestPromoteDraftToReviewed(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Promotable", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("promote", n.ID, "--to", "reviewed")
	if err != nil {
		t.Fatalf("nn promote: %v", err)
	}

	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "reviewed") {
		t.Errorf("note status not updated to reviewed: %q", out)
	}
}

func TestPromoteReviewedToPermanent(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Permanentable", note.TypeConcept)
	n.Status = note.StatusReviewed
	writeNoteFile(t, nbDir, n)

	_, err := execute("promote", n.ID, "--to", "permanent")
	if err != nil {
		t.Fatalf("nn promote: %v", err)
	}

	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "permanent") {
		t.Errorf("note status not updated to permanent: %q", out)
	}
}

func TestPromoteRequiresTo(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Promotable", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("promote", n.ID)
	if err == nil {
		t.Fatal("nn promote without --to: want error, got nil")
	}
}
