package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion F: nn link --type refines stores type in note file.
func TestLinkTypeStoredInFile(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	_, err := execute("link", src.ID, dst.ID, "--annotation", "narrows the scope", "--type", "refines")
	if err != nil {
		t.Fatalf("nn link --type: %v", err)
	}

	out, _ := execute("show", src.ID)
	if !strings.Contains(out, "[refines]") {
		t.Errorf("link type not present in note file:\n%s", out)
	}
}

// Assertion G: nn links <id> output includes link type when present.
func TestLinksShowsType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "narrows the scope", Type: "refines"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("links", src.ID)
	if err != nil {
		t.Fatalf("nn links: %v", err)
	}
	if !strings.Contains(out, "refines") {
		t.Errorf("nn links output missing type 'refines':\n%s", out)
	}
}

// Assertion H: existing links without type continue to parse cleanly.
func TestLinkNoTypeBackwardCompat(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	// Link has no Type field — should parse and round-trip cleanly.
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("links", src.ID)
	if err != nil {
		t.Fatalf("nn links (no type): %v", err)
	}
	if !strings.Contains(out, "relates to") {
		t.Errorf("nn links backward compat: annotation missing:\n%s", out)
	}
}
