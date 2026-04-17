package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: BM25 ranks title match above body-only match.
func TestSearchBM25TitleRanksAboveBody(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// bodyOnly has the query only in its body; titleMatch has it in the title.
	// Give bodyOnly an earlier (smaller) ID so default ordering would put it first.
	bodyOnly := newTestNoteForCLI(note.GenerateID(), "Unrelated Subject", note.TypeConcept)
	bodyOnly.Body = "This note mentions zettelkasten somewhere in the body text."
	bodyOnly.Created = time.Now().Add(-2 * time.Second).UTC()

	titleMatch := newTestNoteForCLI(note.GenerateID(), "Zettelkasten Overview", note.TypeConcept)
	titleMatch.Body = "General description without the search term."
	titleMatch.Created = time.Now().Add(-1 * time.Second).UTC()

	writeNoteFile(t, nbDir, bodyOnly)
	writeNoteFile(t, nbDir, titleMatch)

	out, err := execute("search", "zettelkasten", "--json")
	if err != nil {
		t.Fatalf("nn search: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0]["id"] != titleMatch.ID {
		t.Errorf("title match should rank first; got %v first, want %v", results[0]["id"], titleMatch.ID)
	}
}

// Assertion: multi-word query ranks notes containing more terms higher.
func TestSearchBM25MultiWordRanking(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	oneWord := newTestNoteForCLI(note.GenerateID(), "Atomic Notes", note.TypeConcept)
	oneWord.Body = "Notes should be atomic."
	oneWord.Created = time.Now().Add(-2 * time.Second).UTC()

	twoWords := newTestNoteForCLI(note.GenerateID(), "Atomic Zettelkasten", note.TypeConcept)
	twoWords.Body = "Atomic notes in a zettelkasten system."
	twoWords.Created = time.Now().Add(-1 * time.Second).UTC()

	writeNoteFile(t, nbDir, oneWord)
	writeNoteFile(t, nbDir, twoWords)

	out, err := execute("search", "atomic zettelkasten", "--json")
	if err != nil {
		t.Fatalf("nn search: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d: %s", len(results), out)
	}
	if results[0]["id"] != twoWords.ID {
		t.Errorf("note with both terms should rank first; got %v first, want %v", results[0]["id"], twoWords.ID)
	}
}
