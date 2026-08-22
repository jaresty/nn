package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestGraphShowBodies verifies the --bodies flag on `graph show`.
//
// property [2a]: present(bodies) && format=text => every node's body appears in output
// property [2b]: present(bodies) && format=json => every node's body appears in output
// property [4]:  present(bodies) && format=text => every node's tags appear in output
// property [1]:  absent(bodies) => output byte-identical to current behavior (covered by
//                existing zone/filter tests remaining green)
func TestGraphShowBodies(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	neighbor := newTestNoteForCLI(note.GenerateID(), "Neighbor", note.TypeConcept)
	ego.Created, ego.Modified = now, now
	neighbor.Created, neighbor.Modified = now, now
	ego.Body = "EGO_DISTINCTIVE_BODY"
	neighbor.Body = "NEIGHBOR_DISTINCTIVE_BODY"
	neighbor.Tags = []string{"zonetag"}
	ego.Links = []note.Link{{TargetID: neighbor.ID, Type: "extends", Annotation: "builds on"}}

	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, neighbor)

	// property [2a] + [4]: text output includes each node's body and tags.
	textOut, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--bodies", "--format", "text")
	if err != nil {
		t.Fatalf("graph show --bodies --format text: %v", err)
	}
	if !strings.Contains(textOut, "EGO_DISTINCTIVE_BODY") {
		t.Errorf("property [2a]: ego body missing from text output:\n%s", textOut)
	}
	if !strings.Contains(textOut, "NEIGHBOR_DISTINCTIVE_BODY") {
		t.Errorf("property [2a]: neighbor body missing from text output:\n%s", textOut)
	}
	if !strings.Contains(textOut, "zonetag") {
		t.Errorf("property [4]: neighbor tag missing from text output:\n%s", textOut)
	}

	// property [2b]: json output includes each node's body.
	jsonOut, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--bodies", "--format", "json")
	if err != nil {
		t.Fatalf("graph show --bodies --format json: %v", err)
	}
	var result struct {
		Nodes []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("graph show --bodies --format json: output not valid JSON: %v\n%s", err, jsonOut)
	}
	bodyByID := map[string]string{}
	for _, n := range result.Nodes {
		bodyByID[n.ID] = n.Body
	}
	if !strings.Contains(bodyByID[ego.ID], "EGO_DISTINCTIVE_BODY") {
		t.Errorf("property [2b]: ego body = %q, want to contain %q", bodyByID[ego.ID], "EGO_DISTINCTIVE_BODY")
	}
	if !strings.Contains(bodyByID[neighbor.ID], "NEIGHBOR_DISTINCTIVE_BODY") {
		t.Errorf("property [2b]: neighbor body = %q, want to contain %q", bodyByID[neighbor.ID], "NEIGHBOR_DISTINCTIVE_BODY")
	}
}
