package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn list --similar <id> returns notes ranked by BM25, excluding the source note.
func TestListSimilar(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	src := &note.Note{
		ID: note.GenerateID(), Title: "Zettelkasten note-taking method",
		Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(),
		Body: "The Zettelkasten method is a note-taking system that connects ideas through links.",
	}
	related := &note.Note{
		ID: note.GenerateID(), Title: "Note-taking systems overview",
		Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(),
		Body: "Note-taking systems help organise ideas. The Zettelkasten method is one approach.",
	}
	unrelated := &note.Note{
		ID: note.GenerateID(), Title: "Bicycle maintenance guide",
		Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(),
		Body: "How to oil the chain and adjust the brakes on a bicycle.",
	}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, related)
	writeNoteFile(t, nbDir, unrelated)

	out, err := execute("list", "--similar", src.ID)
	if err != nil {
		t.Fatalf("nn list --similar: %v", err)
	}
	// Source note must not appear.
	if strings.Contains(out, src.ID) {
		t.Errorf("source note %s should be excluded from --similar output:\n%s", src.ID, out)
	}
	// Related note should appear before unrelated.
	relIdx := strings.Index(out, related.ID)
	unrelIdx := strings.Index(out, unrelated.ID)
	if relIdx == -1 {
		t.Errorf("related note missing from --similar output:\n%s", out)
	}
	if unrelIdx != -1 && relIdx > unrelIdx {
		t.Errorf("unrelated note ranked above related note in --similar output:\n%s", out)
	}
}

// Assertion: nn list --similar composes with --limit.
func TestListSimilarLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	src := &note.Note{
		ID: note.GenerateID(), Title: "Machine learning concepts",
		Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(),
		Body: "Machine learning is a subset of artificial intelligence.",
	}
	for i := 0; i < 5; i++ {
		n := &note.Note{
			ID:       note.GenerateID(),
			Title:    "Machine learning topic",
			Type:     note.TypeConcept,
			Status:   note.StatusDraft,
			Created:  time.Now().UTC(),
			Modified: time.Now().UTC(),
			Body:     "Machine learning and AI concepts.",
		}
		writeNoteFile(t, nbDir, n)
	}
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--similar", src.ID, "--limit", "2", "--json")
	if err != nil {
		t.Fatalf("nn list --similar --limit: %v", err)
	}
	var results []noteJSON
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results with --limit 2, got %d", len(results))
	}
}
