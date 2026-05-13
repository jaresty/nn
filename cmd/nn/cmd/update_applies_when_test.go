package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateAppliesWhen(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--applies-when", "before any external action", "--no-edit")
	if err != nil {
		t.Fatalf("nn update --applies-when: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "before any external action") {
		t.Errorf("applies_when not set: %q", out)
	}
}

func TestNewAppliesWhen(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "My Protocol", "--type", "protocol",
		"--applies-when", "when a command fails unexpectedly", "--content", "body", "--no-edit")
	if err != nil {
		t.Fatalf("nn new --applies-when: %v", err)
	}
	// out is "created <id>\n"; extract the ID
	id := strings.TrimPrefix(strings.TrimSpace(out), "created ")
	shown, _ := execute("show", id)
	if !strings.Contains(shown, "when a command fails unexpectedly") {
		t.Errorf("applies_when not set on new note: %q", shown)
	}
}
