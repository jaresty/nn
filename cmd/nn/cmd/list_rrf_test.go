package cmd

import (
	"encoding/json"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [4]: nn list --search uses RRF — note matching in title+body ranks above body-only match
func TestListSearchUsesRRF(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// "both" matches "deploy" in title and body → two-field RRF contribution
	both := newTestNoteForCLI(note.GenerateID(), "deploy pipeline", note.TypeConcept)
	both.Body = "deploy automation workflow"
	writeNoteFile(t, nbDir, both)

	// "bodyOnly" matches "deploy" only in body
	bodyOnly := newTestNoteForCLI(note.GenerateID(), "unrelated subject", note.TypeConcept)
	bodyOnly.Body = "deploy automation workflow"
	writeNoteFile(t, nbDir, bodyOnly)

	out, err := execute("list", "--search", "deploy", "--json", "--fields", "id,score")
	if err != nil {
		t.Fatalf("nn list --search: %v\noutput: %s", err, out)
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, out)
	}

	var bothScore, bodyOnlyScore float64
	for _, r := range results {
		score, _ := r["score"].(float64)
		switch r["id"] {
		case both.ID:
			bothScore = score
		case bodyOnly.ID:
			bodyOnlyScore = score
		}
	}

	if bothScore == 0 {
		t.Errorf("'both' note not found in search results")
	}
	if bodyOnlyScore == 0 {
		t.Errorf("'bodyOnly' note not found in search results")
	}
	if bothScore <= bodyOnlyScore {
		t.Errorf("RRF: expected title+body note (%.4f) > body-only note (%.4f)", bothScore, bodyOnlyScore)
	}
}
