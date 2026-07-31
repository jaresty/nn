package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateClearExpires(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Ephemeral Note", note.TypeObservation)
	exp := time.Now().Add(7 * 24 * time.Hour)
	n.Expires = &exp
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--clear-expires", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --clear-expires: %v", err)
	}
	out, _ := execute("show", n.ID)
	if strings.Contains(out, "expires:") {
		t.Errorf("expires field still present after --clear-expires: %q", out)
	}
}

func TestUpdateClearExpiresWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Conditional Note", note.TypeObservation)
	n.ExpiresWhen = "when the feature ships"
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--clear-expires-when", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --clear-expires-when: %v", err)
	}
	out, _ := execute("show", n.ID)
	if strings.Contains(out, "expires_when:") {
		t.Errorf("expires_when field still present after --clear-expires-when: %q", out)
	}
}
