package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: --max-backlinks 0 returns only notes with zero inbound links.
func TestRandomMaxBacklinksZero(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	isolated := newTestNoteForCLI(note.GenerateID(), "Isolated Note", note.TypeConcept)
	linked := newTestNoteForCLI(note.GenerateID(), "Linked Note", note.TypeConcept)
	linker := newTestNoteForCLI(note.GenerateID(), "Linker Note", note.TypeConcept)
	linker.Links = []note.Link{{TargetID: linked.ID, Annotation: "points to linked", Type: "extends"}}
	writeNoteFile(t, nbDir, isolated)
	writeNoteFile(t, nbDir, linked)
	writeNoteFile(t, nbDir, linker)

	for i := 0; i < 30; i++ {
		out, err := execute("random", "--max-backlinks", "0")
		if err != nil {
			t.Fatalf("nn random --max-backlinks 0: %v", err)
		}
		if strings.Contains(out, "Linked Note") {
			t.Errorf("iteration %d: expected only notes with 0 backlinks, got note with 1 backlink:\n%s", i, out)
		}
	}
}

// property [2]: omitting --max-backlinks returns all notes (no filter applied).
func TestRandomMaxBacklinksDefault(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	linked := newTestNoteForCLI(note.GenerateID(), "Linked Note", note.TypeConcept)
	linker := newTestNoteForCLI(note.GenerateID(), "Linker Note", note.TypeConcept)
	linker.Links = []note.Link{{TargetID: linked.ID, Annotation: "points to linked", Type: "extends"}}
	writeNoteFile(t, nbDir, linked)
	writeNoteFile(t, nbDir, linker)

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		out, err := execute("random")
		if err != nil {
			t.Fatalf("nn random: %v", err)
		}
		if strings.Contains(out, "Linked Note") {
			seen["linked"] = true
		}
		if strings.Contains(out, "Linker Note") {
			seen["linker"] = true
		}
	}
	if !seen["linked"] {
		t.Errorf("expected Linked Note to appear in random output without --max-backlinks filter, but it never did")
	}
}

// property [3]: --max-backlinks N with no qualifying notes errors.
func TestRandomMaxBacklinksNoMatch(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	linked := newTestNoteForCLI(note.GenerateID(), "Linked Note", note.TypeConcept)
	linker := newTestNoteForCLI(note.GenerateID(), "Linker Note", note.TypeConcept)
	linker.Links = []note.Link{{TargetID: linked.ID, Annotation: "points to linked", Type: "extends"}}
	writeNoteFile(t, nbDir, linked)
	writeNoteFile(t, nbDir, linker)

	// Both notes have ≥1 inbound or outbound, but linker has 0 inbound and linked has 1.
	// Use --max-backlinks 0 --type argument to ensure no notes match (no argument-type notes exist).
	_, err := execute("random", "--max-backlinks", "0", "--type", "argument")
	if err == nil {
		t.Errorf("expected error when no notes match --max-backlinks filter, got none")
	}
}

