package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestNoteExpiresFieldParsed — expires field round-trips through Marshal/Parse
func TestNoteExpiresFieldParsed(t *testing.T) {
	exp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	n := newTestNoteForCLI(note.GenerateID(), "Expires Test", note.TypeConcept)
	n.Expires = &exp

	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := note.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.Expires == nil {
		t.Fatal("want Expires non-nil after parse, got nil")
	}
	if !parsed.Expires.Equal(exp) {
		t.Errorf("want Expires %v, got %v", exp, *parsed.Expires)
	}
}

// Assertion: TestListExpiredReturnsExpiredNotes — --expired returns notes with expires in the past
func TestListExpiredReturnsExpiredNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Expired Note", note.TypeConcept)
	n.Expires = &exp
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--expired")
	if err != nil {
		t.Fatalf("nn list --expired: %v", err)
	}
	if !strings.Contains(out, n.ID) {
		t.Errorf("want expired note %s in --expired output, got:\n%s", n.ID, out)
	}
}

// Assertion: TestListExpiredExcludesFutureNotes — --expired excludes notes with future expires
func TestListExpiredExcludesFutureNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Future Note", note.TypeConcept)
	n.Expires = &exp
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--expired")
	if err != nil {
		t.Fatalf("nn list --expired: %v", err)
	}
	if strings.Contains(out, n.ID) {
		t.Errorf("future note %s should be excluded from --expired output, got:\n%s", n.ID, out)
	}
}

// Assertion: TestListHasExpiresIncludesSet — --has-expires returns notes with Expires set
func TestListHasExpiresIncludesSet(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Future Expires Note", note.TypeConcept)
	n.Expires = &exp
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--has-expires")
	if err != nil {
		t.Fatalf("nn list --has-expires: %v", err)
	}
	if !strings.Contains(out, n.ID) {
		t.Errorf("want note with expires %s in --has-expires output, got:\n%s", n.ID, out)
	}
}

// Assertion: TestListHasExpiresExcludesUnset — --has-expires excludes notes with no Expires field
func TestListHasExpiresExcludesUnset(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "No Expires Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--has-expires")
	if err != nil {
		t.Fatalf("nn list --has-expires: %v", err)
	}
	if strings.Contains(out, n.ID) {
		t.Errorf("note without expires %s should be excluded from --has-expires output, got:\n%s", n.ID, out)
	}
}

// Assertion: TestReviewExpiredNotesSection — review output contains expiring notes section
func TestReviewExpiredNotesSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Expired Note", note.TypeConcept)
	n.Expires = &exp
	writeNoteFile(t, nbDir, n)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "Expiring notes") {
		t.Errorf("want 'Expiring notes' section in review output, got:\n%s", out)
	}
	if !strings.Contains(out, n.ID) {
		t.Errorf("want expired note %s in review output, got:\n%s", n.ID, out)
	}
}
