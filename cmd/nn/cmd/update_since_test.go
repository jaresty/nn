package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateSinceReject(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Truncate(time.Second)
	writeNoteFile(t, nbDir, n)

	stale := n.Modified.Add(-5 * time.Second).Format(time.RFC3339)
	_, err := execute("update", n.ID, "--title", "New Title", "--since", stale, "--no-edit")
	if err == nil {
		t.Fatal("expected error when --since is stale, got nil")
	}
	if !strings.Contains(err.Error(), "note was modified since") {
		t.Errorf("expected 'note was modified since' in error, got: %v", err)
	}
}

func TestUpdateSinceMatch(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Truncate(time.Second)
	writeNoteFile(t, nbDir, n)

	since := n.Modified.Format(time.RFC3339)
	_, err := execute("update", n.ID, "--title", "New Title", "--since", since, "--no-edit")
	if err != nil {
		t.Fatalf("expected success when --since matches modified, got: %v", err)
	}
}

func TestUpdateSinceParseFail(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--title", "New Title", "--since", "not-a-timestamp", "--no-edit")
	if err == nil {
		t.Fatal("expected error for malformed --since, got nil")
	}
	if !strings.Contains(err.Error(), "--since") {
		t.Errorf("expected '--since' in error, got: %v", err)
	}
}

func TestUpdateSinceOptional(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--title", "New Title", "--no-edit")
	if err != nil {
		t.Fatalf("expected success without --since, got: %v", err)
	}
}
