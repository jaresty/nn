package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestGraphShowDegree verifies that `graph show` annotates each node with its
// link degree — outgoing and incoming counts — in both text and json output.
//
// property [1a]: text output shows out/in degree for each node
// property [1b]: json output has out_degree and in_degree per node
func TestGraphShowDegree(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	// ego -> a, ego -> b (ego out=2); a -> ego (ego in=1, a out=1)
	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	a := newTestNoteForCLI(note.GenerateID(), "Anode", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Bnode", note.TypeConcept)
	ego.Created, ego.Modified = now, now
	a.Created, a.Modified = now, now
	b.Created, b.Modified = now, now
	ego.Links = []note.Link{
		{TargetID: a.ID, Type: "extends", Annotation: "x"},
		{TargetID: b.ID, Type: "extends", Annotation: "y"},
	}
	a.Links = []note.Link{{TargetID: ego.ID, Type: "questions", Annotation: "z"}}

	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	// property [1b]: json degree fields.
	jsonOut, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--format", "json")
	if err != nil {
		t.Fatalf("graph show json: %v", err)
	}
	var result struct {
		Nodes []struct {
			ID     string `json:"id"`
			OutDeg int    `json:"out_degree"`
			InDeg  int    `json:"in_degree"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("graph show json: %v\n%s", err, jsonOut)
	}
	deg := map[string][2]int{}
	for _, n := range result.Nodes {
		deg[n.ID] = [2]int{n.OutDeg, n.InDeg}
	}
	if deg[ego.ID] != [2]int{2, 1} {
		t.Errorf("property [1b]: ego degree = out%d/in%d, want out2/in1", deg[ego.ID][0], deg[ego.ID][1])
	}
	if deg[a.ID] != [2]int{1, 1} {
		t.Errorf("property [1b]: Anode degree = out%d/in%d, want out1/in1", deg[a.ID][0], deg[a.ID][1])
	}

	// property [1a]: text output shows the degree markers.
	textOut, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--format", "text")
	if err != nil {
		t.Fatalf("graph show text: %v", err)
	}
	// ego line should carry its out=2 in=1 degree in some ↑/↓ or out/in form.
	if !strings.Contains(textOut, "↑2") || !strings.Contains(textOut, "↓1") {
		t.Errorf("property [1a]: ego degree markers (↑2 ↓1) missing from text output:\n%s", textOut)
	}
}
