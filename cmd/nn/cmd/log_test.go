package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestLogRegister(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("log", "someid")
	if err == nil {
		t.Fatal("expected error for unknown id, got nil")
	}
	if strings.Contains(err.Error(), "unknown command") {
		t.Errorf("nn log is not registered: %v", err)
	}
}

func TestLogResolve(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("log", "nonexistent-note-id")
	if err == nil {
		t.Fatal("expected error for unknown id, got nil")
	}
	if !strings.Contains(err.Error(), "no note found") {
		t.Errorf("expected 'no note found' error, got: %v", err)
	}
}

func TestLogInvoke(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Log Test Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)
	commitNoteFile(t, nbDir, n)

	out, err := execute("log", n.ID)
	if err != nil {
		t.Fatalf("nn log: %v", err)
	}
	if !strings.Contains(out, n.Filename()) {
		t.Errorf("expected note filename in log output, got:\n%s", out)
	}
}

func TestLogSince(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Log Test Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)
	commitNoteFile(t, nbDir, n)

	// --since far future should produce no commits
	out, err := execute("log", n.ID, "--since", "2099-01-01")
	if err != nil {
		t.Fatalf("nn log --since: %v", err)
	}
	if strings.Contains(out, "commit") {
		t.Errorf("expected no commits with future --since, got:\n%s", out)
	}
}
