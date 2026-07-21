package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitContextQuery(t *testing.T) {
	t.Run("returns non-empty in git repo with remote", func(t *testing.T) {
		// Run from the nn repo root which has a remote configured.
		repoRoot, err := filepath.Abs("../../../")
		if err != nil {
			t.Fatal(err)
		}
		// Verify assumption: this directory is a git repo with a remote.
		out, err := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin").Output()
		if err != nil || len(out) == 0 {
			t.Skip("test repo has no origin remote — skipping")
		}
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatal(err)
		}
		got := gitContextQuery()
		if got == "" {
			t.Error("gitContextQuery() returned empty string in git repo with remote")
		}
	})

	t.Run("returns empty string outside git repo", func(t *testing.T) {
		tmp := t.TempDir()
		orig, _ := os.Getwd()
		defer os.Chdir(orig)
		if err := os.Chdir(tmp); err != nil {
			t.Fatal(err)
		}
		got := gitContextQuery()
		if got != "" {
			t.Errorf("gitContextQuery() = %q, want empty outside git repo", got)
		}
	})
}
