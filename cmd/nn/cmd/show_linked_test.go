package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn show --linked-from <id> outputs all notes linked from that note.
func TestShowLinkedFrom(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeArgument)
	src.Links = []note.Link{
		{TargetID: dst1.ID, Annotation: "first link"},
		{TargetID: dst2.ID, Annotation: "second link"},
	}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst1)
	writeNoteFile(t, nbDir, dst2)

	out, err := execute("show", "--linked-from", src.ID)
	if err != nil {
		t.Fatalf("nn show --linked-from: %v", err)
	}
	if !strings.Contains(out, "Target One") {
		t.Errorf("Target One missing from --linked-from output:\n%s", out)
	}
	if !strings.Contains(out, "Target Two") {
		t.Errorf("Target Two missing from --linked-from output:\n%s", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("separator missing from --linked-from output:\n%s", out)
	}
}

// Assertion: --linked-from on note with no links returns empty (not an error).
func TestShowLinkedFromEmpty(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "No Links", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--linked-from", n.ID)
	if err != nil {
		t.Fatalf("nn show --linked-from empty should not error: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output for note with no links, got: %q", out)
	}
}
