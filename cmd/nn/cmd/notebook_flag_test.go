package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --notebook selects a named notebook, beats $NN_NOTEBOOK, and unknown names error.
func TestNotebookFlagSelectsNamedNotebook(t *testing.T) {
	defDir, cfgFile := setupNotebookWithCfg(t)
	otherDir := t.TempDir()
	initGitRepoForCLI(t, otherDir)
	f, err := os.OpenFile(cfgFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(f, "\n[notebooks.other]\npath = %q\nbackend = \"gitlocal\"\n", otherDir)
	f.Close()

	run := func(args ...string) (string, error) {
		root := NewRootCmdForTest(cfgFile)
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		err := root.Execute()
		return out.String(), err
	}

	if _, err := run("--notebook", "other", "new", "--title", "In Other", "--content", "x", "--type", "concept"); err != nil {
		t.Fatalf("--notebook other new: %v", err)
	}
	if n := countMD(t, otherDir); n != 1 {
		t.Fatalf("other notebook: want 1 note, got %d", n)
	}
	if n := countMD(t, defDir); n != 0 {
		t.Fatalf("default notebook: want 0 notes, got %d", n)
	}

	// flag beats env
	t.Setenv("NN_NOTEBOOK", "test")
	if _, err := run("--notebook", "other", "new", "--title", "Second", "--content", "y", "--type", "concept"); err != nil {
		t.Fatalf("flag over env: %v", err)
	}
	if n := countMD(t, otherDir); n != 2 {
		t.Fatalf("flag should beat env; other has %d notes", n)
	}

	_, err = run("--notebook", "missing", "list")
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unknown notebook: want error naming it, got %v", err)
	}
}

func countMD(t *testing.T, dir string) int {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	return len(m)
}
