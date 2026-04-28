package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// AC5 (integration): a note with no query terms in its own title/body appears
// in search results when an inbound link annotation contains the query term.
func TestListSearchInboundAnnotationIncludesNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// "target" has no query term anywhere in its own text — previously invisible.
	target := newTestNoteForCLI("20260101000000-0030", "Target Note", note.TypeConcept)
	target.Body = "General discussion."

	// "linker" links to target with an annotation containing the query term.
	linker := newTestNoteForCLI("20260101000000-0032", "Linker Note", note.TypeConcept)
	linker.Body = "See target note."
	linker.Links = []note.Link{
		{TargetID: target.ID, Annotation: "eviction policy reference", Type: "related"},
	}

	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, linker)

	out, err := execute("list", "--search", "eviction", "--json")
	if err != nil {
		t.Fatalf("nn list --search eviction: %v", err)
	}
	titles := orderedTitles(t, out)
	for _, title := range titles {
		if title == "Target Note" {
			return // found — inbound annotation worked
		}
	}
	t.Errorf("Target Note should appear via inbound annotation but got: %v", titles)
}
