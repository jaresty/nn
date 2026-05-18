package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestListHasURL — --has-url returns notes containing an http/https URL.
func TestListHasURL(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	withURL := newTestNoteForCLI(note.GenerateID(), "Has URL", note.TypeConcept)
	withURL.Body = "See https://example.com for details."
	withoutURL := newTestNoteForCLI(note.GenerateID(), "No URL", note.TypeConcept)
	withoutURL.Body = "No links here."
	writeNoteFile(t, nbDir, withURL)
	writeNoteFile(t, nbDir, withoutURL)

	out, err := execute("list", "--has-url")
	if err != nil {
		t.Fatalf("nn list --has-url: %v", err)
	}
	if !strings.Contains(out, withURL.ID) {
		t.Errorf("expected note with URL in output; got:\n%s", out)
	}
	if strings.Contains(out, withoutURL.ID) {
		t.Errorf("expected note without URL to be excluded; got:\n%s", out)
	}
}

// Assertion: TestListURLContains — --url-contains filters to notes with a URL matching the string.
func TestListURLContains(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	match := newTestNoteForCLI(note.GenerateID(), "Match", note.TypeConcept)
	match.Body = "See https://github.com/foo/bar for context."
	noMatch := newTestNoteForCLI(note.GenerateID(), "NoMatch", note.TypeConcept)
	noMatch.Body = "See https://example.com instead."
	writeNoteFile(t, nbDir, match)
	writeNoteFile(t, nbDir, noMatch)

	out, err := execute("list", "--url-contains", "github.com")
	if err != nil {
		t.Fatalf("nn list --url-contains: %v", err)
	}
	if !strings.Contains(out, match.ID) {
		t.Errorf("expected matching note in output; got:\n%s", out)
	}
	if strings.Contains(out, noMatch.ID) {
		t.Errorf("expected non-matching note excluded; got:\n%s", out)
	}
}

// Assertion: TestListURLContainsOnlyMatchesURLs — --url-contains must match within a URL, not bare text.
func TestListURLContainsOnlyMatchesURLs(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	bareText := newTestNoteForCLI(note.GenerateID(), "BareText", note.TypeConcept)
	bareText.Body = "github.com appears as plain text, not a URL."
	writeNoteFile(t, nbDir, bareText)

	out, err := execute("list", "--url-contains", "github.com")
	if err != nil {
		t.Fatalf("nn list --url-contains: %v", err)
	}
	if strings.Contains(out, bareText.ID) {
		t.Errorf("expected bare-text note excluded (no http/https URL); got:\n%s", out)
	}
}
