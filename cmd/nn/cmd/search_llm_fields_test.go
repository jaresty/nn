package cmd

import (
	"encoding/json"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// D1: score field — normalized float in [0,1] in each search JSON result.
func TestSearchJSONScore(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "BM25 Relevance Scoring", note.TypeConcept)
	n.Body = "Describes how BM25 relevance scoring works."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "relevance", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	scoreVal, ok := results[0]["score"]
	if !ok {
		t.Fatalf("'score' field missing from search JSON result: %v", results[0])
	}
	score, ok := scoreVal.(float64)
	if !ok {
		t.Fatalf("'score' field is not a number: %T %v", scoreVal, scoreVal)
	}
	if score <= 0 || score > 1 {
		t.Errorf("'score' = %f, want value in (0, 1]", score)
	}
}

// D2: modified field — ISO 8601 datetime string in each search JSON result.
func TestSearchJSONModified(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Modification Timestamp Note", note.TypeConcept)
	n.Body = "A note about timestamps."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "timestamps", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	modVal, ok := results[0]["modified"]
	if !ok {
		t.Fatalf("'modified' field missing from search JSON result: %v", results[0])
	}
	mod, ok := modVal.(string)
	if !ok || mod == "" {
		t.Errorf("'modified' field is empty or not a string: %v", modVal)
	}
}

// D3: match_reason field — non-empty string naming matched fields.
func TestSearchJSONMatchReasonTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Consensus Algorithm", note.TypeConcept)
	n.Body = "Describes leader election in distributed systems."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "consensus", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	mrVal, ok := results[0]["match_reason"]
	if !ok {
		t.Fatalf("'match_reason' field missing from search JSON result: %v", results[0])
	}
	mr, ok := mrVal.(string)
	if !ok || mr == "" {
		t.Errorf("'match_reason' is empty or not a string: %v", mrVal)
	}
}

// D9 removed: status-based ranking was removed — permanent notes no longer receive
// a score boost over draft notes with identical content.
