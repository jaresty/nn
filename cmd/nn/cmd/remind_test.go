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

// Assertion: TestGlobalShowRemindersBlock — nn show --global appends ## Reminders block with non-expired reminder bodies and expiry date
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
	if !strings.Contains(out, "expires: "+exp.Format("2006-01-02")) {
		t.Errorf("want 'expires: YYYY-MM-DD' in --global reminders output, got:\n%s", out)
	}
}

// Assertion: TestGlobalShowRemindersExpiresWhen — nn show --global shows expires_when in reminders block
func TestGlobalShowRemindersExpiresWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Conditional Reminder", note.TypeObservation)
	n.Status = note.StatusPermanent
	n.Tags = []string{"reminder"}
	n.Expires = &exp
	n.ExpiresWhen = "when the PR is merged"
	n.Body = "Hold off on deploys."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "expires_when: when the PR is merged") {
		t.Errorf("want 'expires_when: ...' in --global reminders output, got:\n%s", out)
	}
}

// Assertion: TestRemindFindFlag — nn remind --find FRAGMENT returns matching reminder IDs by title substring
func TestRemindFindFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	n1 := newTestNoteForCLI(note.GenerateID(), "Check deploy pipeline", note.TypeObservation)
	n1.Status = note.StatusPermanent
	n1.Tags = []string{"reminder"}
	n1.Expires = &exp
	n1.Body = "deploy"
	writeNoteFile(t, nbDir, n1)

	n2 := newTestNoteForCLI(note.GenerateID(), "Review PR queue", note.TypeObservation)
	n2.Status = note.StatusPermanent
	n2.Tags = []string{"reminder"}
	n2.Expires = &exp
	n2.Body = "pr"
	writeNoteFile(t, nbDir, n2)

	out, err := execute("remind", "--find", "deploy")
	if err != nil {
		t.Fatalf("nn remind --find deploy: %v", err)
	}
	if !strings.Contains(out, n1.ID) {
		t.Errorf("want ID %s in output, got:\n%s", n1.ID, out)
	}
	if strings.Contains(out, n2.ID) {
		t.Errorf("want ID %s absent from output, got:\n%s", n2.ID, out)
	}
}

// Assertion: TestRemindFindAmbiguous — nn remind --find aborts with error when multiple matches
func TestRemindFindAmbiguous(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	n1 := newTestNoteForCLI(note.GenerateID(), "Check deploy pipeline", note.TypeObservation)
	n1.Status = note.StatusPermanent
	n1.Tags = []string{"reminder"}
	n1.Expires = &exp
	writeNoteFile(t, nbDir, n1)

	n2 := newTestNoteForCLI(note.GenerateID(), "Check PR queue", note.TypeObservation)
	n2.Status = note.StatusPermanent
	n2.Tags = []string{"reminder"}
	n2.Expires = &exp
	writeNoteFile(t, nbDir, n2)

	_, err := execute("remind", "--find", "Check")
	if err == nil {
		t.Fatal("nn remind --find with ambiguous match: want error, got nil")
	}
}

// Assertion: TestRemindUpdateFlag — nn remind --update ID replaces body of existing reminder
func TestRemindUpdateFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Old reminder body", note.TypeObservation)
	n.Status = note.StatusPermanent
	n.Tags = []string{"reminder"}
	n.Expires = &exp
	n.Body = "Old reminder body"
	writeNoteFile(t, nbDir, n)

	_, err := execute("remind", "New body text", "--update", n.ID)
	if err != nil {
		t.Fatalf("nn remind --update: %v", err)
	}
	updated := readNoteByID(t, nbDir, n.ID)
	if !strings.Contains(updated.Body, "New body text") {
		t.Errorf("want body 'New body text', got %q", updated.Body)
	}
}

// Assertion: TestRemindUpdatePreservesExpiry — nn remind --update preserves existing expiry
func TestRemindUpdatePreservesExpiry(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Some reminder", note.TypeObservation)
	n.Status = note.StatusPermanent
	n.Tags = []string{"reminder"}
	n.Expires = &exp
	n.Body = "original"
	writeNoteFile(t, nbDir, n)

	_, err := execute("remind", "updated body", "--update", n.ID)
	if err != nil {
		t.Fatalf("nn remind --update: %v", err)
	}
	updated := readNoteByID(t, nbDir, n.ID)
	if updated.Expires == nil {
		t.Fatal("want Expires preserved, got nil")
	}
	if !updated.Expires.Equal(exp) {
		t.Errorf("want expires %v, got %v", exp, *updated.Expires)
	}
}

// Assertion: TestRemindUpdateNoNewNote — nn remind --update does not create a new note
func TestRemindUpdateNoNewNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	exp := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Existing reminder", note.TypeObservation)
	n.Status = note.StatusPermanent
	n.Tags = []string{"reminder"}
	n.Expires = &exp
	n.Body = "original"
	writeNoteFile(t, nbDir, n)

	_, err := execute("remind", "new body", "--update", n.ID)
	if err != nil {
		t.Fatalf("nn remind --update: %v", err)
	}
	entries, _ := os.ReadDir(nbDir)
	var mdCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount != 1 {
		t.Errorf("want 1 note file after --update, got %d", mdCount)
	}
}

// Assertion: TestGlobalShowRelayWarningWhenAbsent — nn show --global emits relay warning when daily note has no ## Relay section
func TestGlobalShowRelayWarningWhenAbsent(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	today := time.Now().Format("2006-01-02")
	n := newTestNoteForCLI(note.GenerateID(), "Daily: "+today, note.TypeObservation)
	n.Status = note.StatusDraft
	n.Tags = []string{"daily"}
	n.Body = "## Open\n\n- some task\n"
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Warning: relay block missing") {
		t.Errorf("expected relay warning when ## Relay absent; got:\n%s", out)
	}
}

// Assertion: TestGlobalShowNoRelayWarningWhenPresent — nn show --global suppresses relay warning when ## Relay section exists
func TestGlobalShowNoRelayWarningWhenPresent(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	today := time.Now().Format("2006-01-02")
	n := newTestNoteForCLI(note.GenerateID(), "Daily: "+today, note.TypeObservation)
	n.Status = note.StatusDraft
	n.Tags = []string{"daily"}
	n.Body = "## Relay\n\nSome relay content.\n"
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if strings.Contains(out, "Warning: relay block missing") {
		t.Errorf("expected no relay warning when ## Relay present; got:\n%s", out)
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
