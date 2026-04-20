package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// helper: create a note with specific body and write it to nbDir.
func writeSuggestNote(t *testing.T, nbDir string, id, title, body string) *note.Note {
	t.Helper()
	n := newTestNoteForCLI(id, title, note.TypeConcept)
	n.Body = body
	writeNoteFile(t, nbDir, n)
	return n
}

// Assertion: TestSuggestLinksCommandExists — command is registered and runs without error.
func TestSuggestLinksCommandExists(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	_, err := execute("suggest-links", focal.ID)
	if err != nil {
		t.Fatalf("suggest-links command failed: %v", err)
	}
}

// Assertion: TestSuggestLinksFocalNoteSection — output contains ## Focal note block.
func TestSuggestLinksFocalNoteSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	out, err := execute("suggest-links", focal.ID)
	if err != nil {
		t.Fatalf("suggest-links: %v", err)
	}
	if !strings.Contains(out, "## Focal note") {
		t.Errorf("expected '## Focal note' in output; got:\n%s", out)
	}
	if !strings.Contains(out, focal.ID) {
		t.Errorf("expected focal note ID %q in output; got:\n%s", focal.ID, out)
	}
	if !strings.Contains(out, focal.Title) {
		t.Errorf("expected focal note title %q in output; got:\n%s", focal.Title, out)
	}
}

// Assertion: TestSuggestLinksCandidatesSection — output contains ## Candidate notes block.
func TestSuggestLinksCandidatesSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Overview", "zettelkasten is a method for atomic note taking")
	out, err := execute("suggest-links", focal.ID)
	if err != nil {
		t.Fatalf("suggest-links: %v", err)
	}
	if !strings.Contains(out, "## Candidate notes") {
		t.Errorf("expected '## Candidate notes' in output; got:\n%s", out)
	}
}

// Assertion: TestSuggestLinksExistingLinksMarked — already-linked note is marked.
func TestSuggestLinksExistingLinksMarked(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Overview", "zettelkasten is a method for atomic note taking")

	focal := newTestNoteForCLI("20240101000001", "Atomic Notes", note.TypeConcept)
	focal.Body = "atomic notes are single-idea notes"
	focal.Links = []note.Link{{TargetID: target.ID, Type: "extends", Annotation: "extends the zettelkasten concept"}}
	writeNoteFile(t, nbDir, focal)

	out, err := execute("suggest-links", focal.ID)
	if err != nil {
		t.Fatalf("suggest-links: %v", err)
	}
	if !strings.Contains(out, "already linked") {
		t.Errorf("expected 'already linked' marker for existing link; got:\n%s", out)
	}
}

// Assertion: TestSuggestLinksFormatJSON — --format json produces valid JSON with required fields.
func TestSuggestLinksFormatJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Overview", "zettelkasten is a method for atomic note taking")

	out, err := execute("suggest-links", focal.ID, "--format", "json")
	if err != nil {
		t.Fatalf("suggest-links --format json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON; got:\n%s\nerr: %v", out, err)
	}
	if _, ok := result["focal_note"]; !ok {
		t.Errorf("expected 'focal_note' key in JSON; got keys: %v", jsonKeys(result))
	}
	if _, ok := result["candidates"]; !ok {
		t.Errorf("expected 'candidates' key in JSON; got keys: %v", jsonKeys(result))
	}
}

// Assertion: TestSuggestLinksLimit — --limit 1 returns at most 1 candidate.
func TestSuggestLinksLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Overview", "zettelkasten is a method for atomic note taking")
	writeSuggestNote(t, nbDir, "20240101000003", "Evergreen Notes", "evergreen notes are atomic and linked")

	out, err := execute("suggest-links", focal.ID, "--limit", "1")
	if err != nil {
		t.Fatalf("suggest-links --limit 1: %v", err)
	}
	// Count "### " headers in candidates section (each candidate starts with ### <id>)
	count := strings.Count(out, "\n### ")
	if count > 1 {
		t.Errorf("expected at most 1 candidate with --limit 1; got %d", count)
	}
}

// Assertion: TestSuggestLinksExcludesZeroScore — zero-score note absent from candidates section.
func TestSuggestLinksExcludesZeroScore(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	related := writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Overview", "zettelkasten is a method for atomic note taking")
	unrelated := writeSuggestNote(t, nbDir, "20240101000003", "Completely Unrelated Topic", "xyzzy plugh thud wumpus frobozz")

	out, err := execute("suggest-links", focal.ID)
	if err != nil {
		t.Fatalf("suggest-links: %v", err)
	}
	if !strings.Contains(out, related.ID) {
		t.Errorf("expected related note %q in candidates; got:\n%s", related.ID, out)
	}
	if strings.Contains(out, unrelated.ID) {
		t.Errorf("expected zero-score note %q excluded from candidates; got:\n%s", unrelated.ID, out)
	}
}

// Assertion: TestSuggestLinksReportsExcludedCount — header reports excluded count when notes are filtered.
func TestSuggestLinksReportsExcludedCount(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focal := writeSuggestNote(t, nbDir, "20240101000001", "Atomic Notes", "atomic notes are single-idea notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Overview", "zettelkasten is a method for atomic note taking")
	writeSuggestNote(t, nbDir, "20240101000003", "Completely Unrelated Topic", "xyzzy plugh thud wumpus frobozz")

	out, err := execute("suggest-links", focal.ID)
	if err != nil {
		t.Fatalf("suggest-links: %v", err)
	}
	if !strings.Contains(out, "excluded") {
		t.Errorf("expected 'excluded' count in header; got:\n%s", out)
	}
}

func jsonKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
