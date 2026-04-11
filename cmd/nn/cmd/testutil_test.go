package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// setupNotebook creates a temp directory with a git repo and a config file,
// and returns (notebookDir, executeCmd).
// executeCmd runs the nn root command against the temp notebook and returns stdout.
func setupNotebook(t *testing.T) (string, func(...string) (string, error)) {
	t.Helper()
	nbDir := t.TempDir()
	initGitRepoForCLI(t, nbDir)

	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	os.WriteFile(cfgFile, []byte(fmt.Sprintf(`
[notebooks]
default = "test"

[notebooks.test]
path = %q
backend = "gitlocal"
`, nbDir)), 0o644)

	execute := func(args ...string) (string, error) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		root := NewRootCmdForTest(cfgFile)
		root.SetOut(&stdout)
		root.SetErr(&stderr)
		root.SetArgs(args)
		err := root.Execute()
		return stdout.String(), err
	}
	return nbDir, execute
}

func initGitRepoForCLI(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
}

// writeNote writes a note file directly to nbDir (bypasses git) for test setup.
func writeNoteFile(t *testing.T, nbDir string, n *note.Note) {
	t.Helper()
	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	os.WriteFile(filepath.Join(nbDir, n.Filename()), data, 0o644)
}

func newTestNoteForCLI(id, title string, typ note.Type) *note.Note {
	return &note.Note{
		ID: id, Title: title, Type: typ, Status: note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "Test body.",
	}
}

func mustJSON(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatalf("invalid JSON %q: %v", s, err)
	}
}
