package cmd

import (
	"encoding/json"
	"reflect"
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

func TestSearchPresentationMatchesListSearch(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	var notes []*note.Note
	for i := 0; i < 7; i++ {
		n := newTestNoteForCLI("20260101000000-01"+string(rune('0'+i)), "Alpha Note "+string(rune('A'+i)), note.TypeConcept)
		n.Body = "alpha shared body"
		if i%2 == 0 {
			n.Status = note.StatusReviewed
		}
		notes = append(notes, n)
	}
	notes[0].Links = []note.Link{{TargetID: notes[1].ID, Annotation: "alpha connection", Type: "related"}}
	for _, n := range notes {
		writeNoteFile(t, nbDir, n)
	}

	listOut, err := execute("list", "--search", "alpha", "--json")
	if err != nil {
		t.Fatalf("list search: %v", err)
	}
	searchOut, err := execute("search", "alpha", "--json")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	var listRows, searchRows []map[string]any
	if err := json.Unmarshal([]byte(listOut), &listRows); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if err := json.Unmarshal([]byte(searchOut), &searchRows); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if !reflect.DeepEqual(searchRows, listRows) {
		t.Fatalf("search presentation differs from list --search\nsearch=%v\nlist=%v", searchRows, listRows)
	}

	listFiltered, err := execute("list", "--search", "alpha", "--status", "reviewed", "--json")
	if err != nil {
		t.Fatal(err)
	}
	searchFiltered, err := execute("search", "alpha", "--status", "reviewed", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if searchFiltered != listFiltered {
		t.Fatalf("status-filtered JSON differs\nsearch=%s\nlist=%s", searchFiltered, listFiltered)
	}

	listText, err := execute("list", "--search", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	searchText, err := execute("search", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if searchText != listText {
		t.Fatalf("text presentation differs\nsearch=%s\nlist=%s", searchText, listText)
	}
}
