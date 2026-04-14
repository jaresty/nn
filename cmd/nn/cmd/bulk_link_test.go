package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion C: bulk-link creates all specified links.
func TestBulkLinkCreatesAllLinks(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst1 := newTestNoteForCLI(note.GenerateID(), "Target One", note.TypeConcept)
	dst2 := newTestNoteForCLI(note.GenerateID(), "Target Two", note.TypeConcept)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst1)
	writeNoteFile(t, nbDir, dst2)

	_, err := execute("bulk-link", src.ID,
		"--to", dst1.ID, "--annotation", "extends this",
		"--to", dst2.ID, "--annotation", "contradicts that",
	)
	if err != nil {
		t.Fatalf("nn bulk-link: %v", err)
	}

	out, _ := execute("show", src.ID)
	if !strings.Contains(out, dst1.ID) {
		t.Errorf("bulk-link: source missing link to %s:\n%s", dst1.ID, out)
	}
	if !strings.Contains(out, "extends this") {
		t.Errorf("bulk-link: source missing annotation 'extends this':\n%s", out)
	}
	if !strings.Contains(out, dst2.ID) {
		t.Errorf("bulk-link: source missing link to %s:\n%s", dst2.ID, out)
	}
	if !strings.Contains(out, "contradicts that") {
		t.Errorf("bulk-link: source missing annotation 'contradicts that':\n%s", out)
	}
}

// Assertion D: bulk-link errors on mismatched --to/--annotation counts.
func TestBulkLinkMismatchedCountsErrors(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("bulk-link", src.ID,
		"--to", dst.ID,
		// missing --annotation
	)
	if err == nil {
		t.Fatal("nn bulk-link with missing --annotation: want error, got nil")
	}
}
