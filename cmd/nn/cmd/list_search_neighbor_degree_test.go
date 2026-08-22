package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestSearchNeighborDegree verifies that `nn list --search --json` graph
// neighbors each carry a degree (out_degree/in_degree) so a hub can be told
// from a leaf without a second lookup.
//
// property [2]: each neighbor in the search JSON has out_degree and in_degree.
func TestSearchNeighborDegree(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	anchor := newTestNoteForCLI(note.GenerateID(), "Zebra anchor note", note.TypeConcept)
	hub := newTestNoteForCLI(note.GenerateID(), "Zebra hub note", note.TypeConcept)
	extra := newTestNoteForCLI(note.GenerateID(), "Zebra extra note", note.TypeConcept)
	anchor.Created, anchor.Modified = now, now
	hub.Created, hub.Modified = now, now
	extra.Created, extra.Modified = now, now
	// anchor -> hub ; hub -> extra (hub out=1, in=1)
	anchor.Links = []note.Link{{TargetID: hub.ID, Type: "refines", Annotation: "a"}}
	hub.Links = []note.Link{{TargetID: extra.ID, Type: "refines", Annotation: "b"}}

	writeNoteFile(t, nbDir, anchor)
	writeNoteFile(t, nbDir, hub)
	writeNoteFile(t, nbDir, extra)

	out, err := execute("list", "--search", "Zebra anchor", "--json")
	if err != nil {
		t.Fatalf("list --search --json: %v", err)
	}

	var results []struct {
		ID        string `json:"id"`
		Neighbors []struct {
			ID     string `json:"id"`
			OutDeg int    `json:"out_degree"`
			InDeg  int    `json:"in_degree"`
			HasOut bool   `json:"-"`
		} `json:"neighbors"`
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("list --search --json: %v\n%s", err, out)
	}

	var found bool
	for _, r := range results {
		for _, nb := range r.Neighbors {
			if nb.ID == hub.ID {
				found = true
				if nb.OutDeg != 1 || nb.InDeg != 1 {
					t.Errorf("property [2]: hub neighbor degree = out%d/in%d, want out1/in1", nb.OutDeg, nb.InDeg)
				}
			}
		}
	}
	if !found {
		t.Fatalf("property [2]: hub neighbor not found in search output:\n%s", out)
	}
}
