package gitlocal_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// TestGitLockReleasedAfterCommit verifies the git-commit.lock file is gone after commit.
func TestGitLockReleasedAfterCommit(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n := &note.Note{
		ID: note.GenerateID(), Title: "Lock Test", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC().Truncate(time.Second), Modified: time.Now().UTC().Truncate(time.Second),
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	lockPath := filepath.Join(configDir, "git-commit.lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("git-commit.lock still exists after Write completed")
	}
}

// TestGitLockStolenWhenHolderDead verifies a stale lock from a dead PID is stolen.
func TestGitLockStolenWhenHolderDead(t *testing.T) {
	configDir := t.TempDir()
	lockPath := filepath.Join(configDir, "git-commit.lock")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a lock file with a PID that is guaranteed dead (PID 0 is never valid).
	if err := os.WriteFile(lockPath, []byte("0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// AcquireGitLock must succeed by stealing the stale lock.
	if err := gitlocal.AcquireGitLock(configDir); err != nil {
		t.Fatalf("AcquireGitLock with stale lock: %v", err)
	}
	gitlocal.ReleaseGitLock(configDir)
}

