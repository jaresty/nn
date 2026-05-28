package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func readNoteByID(t *testing.T, nbDir, id string) *note.Note {
	t.Helper()
	entries, err := os.ReadDir(nbDir)
	if err != nil {
		t.Fatalf("readNoteByID: ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id) {
			data, err := os.ReadFile(filepath.Join(nbDir, e.Name()))
			if err != nil {
				t.Fatalf("readNoteByID: ReadFile: %v", err)
			}
			n, err := note.Parse(data)
			if err != nil {
				t.Fatalf("readNoteByID: Parse: %v", err)
			}
			return n
		}
	}
	t.Fatalf("readNoteByID: no file found for ID %s", id)
	return nil
}

// Assertion: TestRemindCreatesTaggedPermanentObservation — nn remind creates observation tagged reminder with permanent status
func TestRemindCreatesTaggedPermanentObservation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	out, err := execute("remind", "Check in with the mobile team about the release branch")
	if err != nil {
		t.Fatalf("nn remind: %v", err)
	}

	id := strings.TrimPrefix(strings.TrimSpace(strings.Split(out, " (expires")[0]), "created ")
	if id == "" || id == out {
		t.Fatalf("nn remind: expected 'created <id> (expires ...)' output, got %q", out)
	}

	n := readNoteByID(t, nbDir, id)
	if n.Type != note.TypeObservation {
		t.Errorf("want type observation, got %s", n.Type)
	}
	if n.Status != note.StatusPermanent {
		t.Errorf("want status permanent, got %s", n.Status)
	}
	hasReminder := false
	for _, tag := range n.Tags {
		if tag == "reminder" {
			hasReminder = true
		}
	}
	if !hasReminder {
		t.Errorf("want tag 'reminder', got tags %v", n.Tags)
	}
}

// Assertion: TestRemindOutputIncludesExpiry — nn remind output includes expiry date
func TestRemindOutputIncludesExpiry(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("remind", "Check the deploy")
	if err != nil {
		t.Fatalf("nn remind: %v", err)
	}
	if !strings.Contains(out, "(expires ") {
		t.Errorf("want expiry date in output, got %q", out)
	}
}

// Assertion: TestRemindExpiresDefaultOneDay — nn remind without --for or --expires sets expires to tomorrow
func TestRemindExpiresDefaultOneDay(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	out, err := execute("remind", "Temporary context note")
	if err != nil {
		t.Fatalf("nn remind: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(strings.Split(out, " (expires")[0]), "created ")

	n := readNoteByID(t, nbDir, id)
	if n.Expires == nil {
		t.Fatal("want Expires non-nil, got nil")
	}
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	diff := n.Expires.Sub(tomorrow)
	if diff < -2*time.Minute || diff > 2*time.Minute {
		t.Errorf("want expires ~tomorrow, got %v", *n.Expires)
	}
}

// Assertion: TestRemindTitleTruncated — nn remind sets title to first 60 chars of content
func TestRemindTitleTruncated(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	content := "This is a very long reminder message that exceeds sixty characters easily"
	out, err := execute("remind", content)
	if err != nil {
		t.Fatalf("nn remind: %v", err)
	}
	id := strings.TrimPrefix(strings.TrimSpace(strings.Split(out, " (expires")[0]), "created ")

	n := readNoteByID(t, nbDir, id)
	want := content[:60]
	if n.Title != want {
		t.Errorf("want title %q, got %q", want, n.Title)
	}
}

// Assertion: TestGlobalShowRemindersBlock — nn show --global appends ## Reminders block with non-expired reminder bodies
func TestGlobalShowRemindersBlock(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Active Reminder", note.TypeObservation)
	n.Status = note.StatusPermanent
	n.Tags = []string{"reminder"}
	n.Expires = &exp
	n.Body = "Remember to check the deploy pipeline."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "## Reminders") {
		t.Errorf("want '## Reminders' block in --global output, got:\n%s", out)
	}
	if !strings.Contains(out, "Remember to check the deploy pipeline.") {
		t.Errorf("want reminder body in --global output, got:\n%s", out)
	}
}

// Assertion: TestGlobalShowRemindersExcludesExpired — nn show --global omits expired reminder notes
func TestGlobalShowRemindersExcludesExpired(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Old Reminder", note.TypeObservation)
	n.Status = note.StatusPermanent
	n.Tags = []string{"reminder"}
	n.Expires = &exp
	n.Body = "This reminder has expired."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if strings.Contains(out, "This reminder has expired.") {
		t.Errorf("expired reminder body should be absent from --global output, got:\n%s", out)
	}
}
