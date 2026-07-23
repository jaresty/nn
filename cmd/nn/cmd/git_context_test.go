package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func makeGitRepo(t *testing.T, branch, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0o755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/"+branch+"\n"), 0o644)
	config := "[core]\n\trepositoryformatversion = 0\n[remote \"origin\"]\n\turl = " + remoteURL + "\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n"
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0o644)
	return dir
}

func TestGitContextQueryFromFiles(t *testing.T) {
	t.Run("reads branch and repo name from git files", func(t *testing.T) {
		dir := makeGitRepo(t, "my-branch", "https://github.com/user/my-repo.git")
		got := gitContextQueryFromFiles(dir)
		if got == "" {
			t.Fatal("expected non-empty result")
		}
		if want := "my-repo"; !containsStr(got, want) {
			t.Errorf("got %q, want it to contain %q", got, want)
		}
		if want := "my-branch"; !containsStr(got, want) {
			t.Errorf("got %q, want it to contain %q", got, want)
		}
	})

	t.Run("returns empty when no .git dir", func(t *testing.T) {
		got := gitContextQueryFromFiles(t.TempDir())
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("handles detached HEAD gracefully", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0o755)
		os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc123def456\n"), 0o644)
		config := "[remote \"origin\"]\n\turl = https://github.com/user/repo.git\n"
		os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0o644)
		got := gitContextQueryFromFiles(dir)
		// Should still return repo name even without branch
		if want := "repo"; !containsStr(got, want) {
			t.Errorf("got %q, want it to contain %q", got, want)
		}
	})

	t.Run("returns empty when no remote", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		os.MkdirAll(gitDir, 0o755)
		os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)
		os.WriteFile(filepath.Join(gitDir, "config"), []byte("[core]\n\trepositoryformatversion = 0\n"), 0o644)
		got := gitContextQueryFromFiles(dir)
		if got != "" {
			t.Errorf("got %q, want empty (no remote)", got)
		}
	})
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

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
