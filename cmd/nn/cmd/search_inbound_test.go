package cmd

import (
	"slices"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [2] and [3]: both primary search commands use outbound annotations
// from the shared ranker; inbound targets do not receive the disabled signal.
func TestSearchCommandsUseSharedOutboundAnnotationRanking(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	target := newTestNoteForCLI("20260101000000-0030", "Target Note", note.TypeConcept)
	target.Body = "General discussion."

	linker := newTestNoteForCLI("20260101000000-0032", "Linker Note", note.TypeConcept)
	linker.Body = "See target note."
	linker.Links = []note.Link{
		{TargetID: target.ID, Annotation: "eviction policy reference", Type: "related"},
	}

	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, linker)

	commands := [][]string{
		{"list", "--search", "eviction", "--json"},
		{"search", "eviction", "--json"},
	}
	for _, args := range commands {
		out, err := execute(args...)
		if err != nil {
			t.Fatalf("nn %v: %v", args, err)
		}
		titles := orderedTitles(t, out)
		if !slices.Contains(titles, "Linker Note") {
			t.Errorf("nn %v should return outbound source; got %v", args, titles)
		}
		if slices.Contains(titles, "Target Note") {
			t.Errorf("nn %v returned inbound target with inbound weight zero: %v", args, titles)
		}
	}
}
