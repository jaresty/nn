package feedback

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeSession creates a feedback session dir with one file, backdated to age.
func makeSession(t *testing.T, root, id string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f := filepath.Join(dir, "result.json")
	if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mtime := time.Now().Add(-age)
	if err := os.Chtimes(f, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return dir
}

// property [24]: a session dir whose newest file is older than the retention
// window is removed. property [25]: a recent one is retained.
func TestCleanupSessionsRemovesOldKeepsRecent(t *testing.T) {
	root := t.TempDir()
	old := makeSession(t, root, "20200101000000-0001", 30*24*time.Hour)
	recent := makeSession(t, root, "20260101000000-0002", 1*time.Hour)

	if err := CleanupSessions(root, 7*24*time.Hour); err != nil {
		t.Fatalf("CleanupSessions: %v", err)
	}

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old session dir not removed (err=%v)", err)
	}
	if _, err := os.Stat(recent); err != nil {
		t.Fatalf("recent session dir removed: %v", err)
	}
}

// property [26]: CleanupSessions is best-effort — a missing root is not an error,
// and it does not fail the caller.
func TestCleanupSessionsMissingRootIsNoError(t *testing.T) {
	if err := CleanupSessions(filepath.Join(t.TempDir(), "does-not-exist"), time.Hour); err != nil {
		t.Fatalf("CleanupSessions on missing root = %v, want nil", err)
	}
}
