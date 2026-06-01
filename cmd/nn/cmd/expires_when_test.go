package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestExpiresWhenRoundtrip — expires_when field round-trips through Marshal/Parse
func TestExpiresWhenRoundtrip(t *testing.T) {
	n := newTestNoteForCLI(note.GenerateID(), "Condition Note", note.TypeConcept)
	n.ExpiresWhen = "the auth PR is merged"

	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := note.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.ExpiresWhen != "the auth PR is merged" {
		t.Errorf("want ExpiresWhen %q, got %q", "the auth PR is merged", parsed.ExpiresWhen)
	}
}

// Assertion: TestNewExpiresWhenFlag — nn new --expires-when sets the field
func TestNewExpiresWhenFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Pending Note", "--type", "concept",
		"--content", "Some content.", "--no-edit",
		"--expires-when", "the migration is complete")
	if err != nil {
		t.Fatalf("nn new --expires-when: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")

	n := readNoteByID(t, nbDir, id)
	if n.ExpiresWhen != "the migration is complete" {
		t.Errorf("want ExpiresWhen %q, got %q", "the migration is complete", n.ExpiresWhen)
	}
}

// Assertion: TestUpdateExpiresWhenFlag — nn update --expires-when sets the field
func TestUpdateExpiresWhenFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Update Target", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--expires-when", "when v2 ships", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --expires-when: %v", err)
	}

	updated := readNoteByID(t, nbDir, n.ID)
	if updated.ExpiresWhen != "when v2 ships" {
		t.Errorf("want ExpiresWhen %q, got %q", "when v2 ships", updated.ExpiresWhen)
	}
}

// Assertion: TestReviewPendingConditionsSection — review output contains Pending conditions section
func TestReviewPendingConditionsSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Conditional Note", note.TypeConcept)
	n.ExpiresWhen = "the PR is closed"
	writeNoteFile(t, nbDir, n)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "Pending conditions") {
		t.Errorf("want 'Pending conditions' section in review output, got:\n%s", out)
	}
	if !strings.Contains(out, n.ID) {
		t.Errorf("want note %s in Pending conditions, got:\n%s", n.ID, out)
	}
}

// Assertion: TestReviewExpiryCandidatesSection — review output contains Expiry candidates section
func TestReviewExpiryCandidatesSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	old := newTestNoteForCLI(note.GenerateID(), "Old Observation", note.TypeObservation)
	old.Modified = time.Now().UTC().Add(-35 * 24 * time.Hour)
	old.Status = note.StatusDraft
	writeNoteFile(t, nbDir, old)

	fresh := newTestNoteForCLI(note.GenerateID(), "Fresh Observation", note.TypeObservation)
	fresh.Modified = time.Now().UTC().Add(-1 * time.Hour)
	writeNoteFile(t, nbDir, fresh)

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	// Extract only the Expiry candidates section to avoid false positives from
	// other sections (e.g. Orphans, Note access) that also list all note IDs.
	expirySectionIdx := strings.Index(out, "## Expiry candidates")
	if expirySectionIdx < 0 {
		t.Fatalf("want 'Expiry candidates' section in review output, got:\n%s", out)
	}
	nextSectionIdx := strings.Index(out[expirySectionIdx+1:], "\n## ")
	var expirySection string
	if nextSectionIdx < 0 {
		expirySection = out[expirySectionIdx:]
	} else {
		expirySection = out[expirySectionIdx : expirySectionIdx+1+nextSectionIdx]
	}
	if !strings.Contains(expirySection, old.ID) {
		t.Errorf("want old observation %s in Expiry candidates section, got:\n%s", old.ID, expirySection)
	}
	if strings.Contains(expirySection, fresh.ID) {
		t.Errorf("fresh observation %s should not appear in Expiry candidates section, got:\n%s", fresh.ID, expirySection)
	}
}
