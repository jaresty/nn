package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestListStaleReturnsAccessedUnactedNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	n := newTestNoteForCLI(note.GenerateID(), "Stale Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	// Write an access.log entry 1 hour ago for this note.
	accessTime := time.Now().Add(-1 * time.Hour)
	logLine := fmt.Sprintf("%s show %s\n", accessTime.UTC().Format(time.RFC3339), n.ID)
	if err := os.WriteFile(filepath.Join(cfgDir, "access.log"), []byte(logLine), 0o644); err != nil {
		t.Fatal(err)
	}

	// Note has not been committed since access — no git commits in repo at all.
	out, err := execute("list", "--stale")
	if err != nil {
		t.Fatalf("nn list --stale: %v", err)
	}
	if !strings.Contains(out, n.ID) {
		t.Errorf("nn list --stale: expected note %s in output, got %q", n.ID, out)
	}
}

func TestListStaleExcludesRecentlyCommittedNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	n := newTestNoteForCLI(note.GenerateID(), "Fresh Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	// Commit the note (simulate post-access action).
	commitNoteFile(t, nbDir, n)

	// Write an access.log entry 1 hour BEFORE the commit (note was committed after access).
	accessTime := time.Now().Add(-2 * time.Hour)
	logLine := fmt.Sprintf("%s show %s\n", accessTime.UTC().Format(time.RFC3339), n.ID)
	if err := os.WriteFile(filepath.Join(cfgDir, "access.log"), []byte(logLine), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("list", "--stale")
	if err != nil {
		t.Fatalf("nn list --stale: %v", err)
	}
	if strings.Contains(out, n.ID) {
		t.Errorf("nn list --stale: note %s should be excluded (committed after access), got %q", n.ID, out)
	}
}
