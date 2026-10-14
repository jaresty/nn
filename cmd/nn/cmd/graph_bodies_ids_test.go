package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestGraphBodiesExplicitIDsAreLosslessAndExclusive(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	first := newTestNoteForCLI(note.GenerateID(), "first", note.TypeObservation)
	first.Body = "α" + string(make([]byte, 0))
	second := newTestNoteForCLI(note.GenerateID(), "second", note.TypeObservation)
	second.Body = "selected second body"
	other := newTestNoteForCLI(note.GenerateID(), "other", note.TypeObservation)
	other.Body = "must not be emitted"
	for _, n := range []*note.Note{first, second, other} {
		writeNoteFile(t, nbDir, n)
	}

	_, pages := fetchGraphBodyPages(t, execute, "--ids", first.ID+","+second.ID)
	got := reconstructGraphBodies(t, pages)
	if len(got) != 2 || got[first.ID] != storedGraphBody(t, first) || got[second.ID] != storedGraphBody(t, second) {
		t.Fatalf("explicit bodies = %#v", got)
	}
	if _, ok := got[other.ID]; ok {
		t.Fatalf("unselected note emitted: %#v", got)
	}
	if _, err := execute("graph", "bodies", "--ids", first.ID, "--focus", second.ID); err == nil {
		t.Fatal("--ids with --focus succeeded")
	}
	if _, err := execute("graph", "bodies", "--ids", "missing"); err == nil {
		t.Fatal("unknown explicit ID succeeded")
	}
}
