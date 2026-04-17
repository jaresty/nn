package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// executeWithStderr runs a command and returns (stdout, stderr, error).
func executeWithStderr(t *testing.T, cfgFile string, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := NewRootCmdForTest(cfgFile)
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

// Assertion: body > 2000 chars prints warning to stderr.
func TestAtomicityWarnStderr(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)

	largeBody := strings.Repeat("x", 2001)
	_, stderr, err := executeWithStderr(t, cfgFile, "new", "--type", "concept", "--title", "Big Note", "--content", largeBody, "--no-edit")
	if err != nil {
		t.Fatalf("nn new large body: %v", err)
	}
	if !strings.Contains(stderr, "warning") {
		t.Errorf("expected warning on stderr for large body, got: %q", stderr)
	}
}

// Assertion: body <= 2000 chars produces no warning.
func TestAtomicityNoWarnSmallBody(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)

	smallBody := strings.Repeat("x", 100)
	_, stderr, err := executeWithStderr(t, cfgFile, "new", "--type", "concept", "--title", "Small Note", "--content", smallBody, "--no-edit")
	if err != nil {
		t.Fatalf("nn new small body: %v", err)
	}
	if strings.Contains(stderr, "warning") {
		t.Errorf("unexpected warning for small body: %q", stderr)
	}
}

// Assertion: nn update also warns on large body.
func TestAtomicityWarnOnUpdate(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)

	n := newTestNoteForCLI(note.GenerateID(), "Update Target", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	largeBody := strings.Repeat("x", 2001)
	_, stderr, err := executeWithStderr(t, cfgFile, "update", n.ID, "--content", largeBody, "--no-edit")
	if err != nil {
		t.Fatalf("nn update large body: %v", err)
	}
	if !strings.Contains(stderr, "warning") {
		t.Errorf("expected warning on stderr for large update body, got: %q", stderr)
	}
}
