package gitlocal_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jaresty/nn/internal/backend/gitlocal"
)

// queueDir returns the commit queue dir for the given config dir.
func queueDir(configDir string) string {
	return filepath.Join(configDir, "commit-queue")
}

// lockFile returns the drain lock path for the given config dir.
func lockFile(configDir string) string {
	return filepath.Join(configDir, "commit-queue.lock")
}

// TestEnqueueCreatesItem verifies that Enqueue writes an atomic JSON item into
// the queue directory with the expected fields.
func TestEnqueueCreatesItem(t *testing.T) {
	configDir := t.TempDir()
	item := gitlocal.CommitItem{
		Op:      "write",
		Message: "note: create 123 — test",
		Files:   []string{"/tmp/foo.md"},
	}
	if err := gitlocal.Enqueue(configDir, item); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	entries, err := os.ReadDir(queueDir(configDir))
	if err != nil {
		t.Fatalf("ReadDir queue: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 queue item, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(queueDir(configDir), entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got gitlocal.CommitItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Message != item.Message {
		t.Errorf("message: got %q want %q", got.Message, item.Message)
	}
}

// TestDrainCommitsItems verifies that DrainQueue commits all queued items in
// order and leaves the queue empty.
func TestDrainCommitsItems(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	configDir := t.TempDir()

	// Write two note files and enqueue their commits.
	files := []string{
		filepath.Join(dir, "note-a.md"),
		filepath.Join(dir, "note-b.md"),
	}
	for i, f := range files {
		if err := os.WriteFile(f, []byte(fmt.Sprintf("---\nid: %d\ntitle: note %d\n---\n", i, i)), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if err := gitlocal.Enqueue(configDir, gitlocal.CommitItem{
			Op:      "write",
			Message: fmt.Sprintf("note: create %d", i),
			Files:   []string{f},
		}); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}

	if err := gitlocal.DrainQueue(configDir, dir); err != nil {
		t.Fatalf("DrainQueue: %v", err)
	}

	// Queue should be empty.
	entries, err := os.ReadDir(queueDir(configDir))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty queue, got %d items", len(entries))
	}

	// Both files should be committed.
	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	log := string(out)
	if !strings.Contains(log, "note: create 0") || !strings.Contains(log, "note: create 1") {
		t.Errorf("expected both commits in git log, got:\n%s", log)
	}
}

// TestStaleLockRecovery verifies that a lock file referencing a dead pid is
// stolen and the drain proceeds.
func TestStaleLockRecovery(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	configDir := t.TempDir()

	// Plant a stale lock with a pid that cannot be alive (pid 0 is never a user process).
	if err := os.MkdirAll(filepath.Dir(lockFile(configDir)), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(lockFile(configDir), []byte("0\n"), 0o644); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}

	f := filepath.Join(dir, "note-stale.md")
	if err := os.WriteFile(f, []byte("---\nid: stale\ntitle: stale\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := gitlocal.Enqueue(configDir, gitlocal.CommitItem{
		Op:      "write",
		Message: "note: create stale",
		Files:   []string{f},
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	if err := gitlocal.DrainQueue(configDir, dir); err != nil {
		t.Fatalf("DrainQueue with stale lock: %v", err)
	}

	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "note: create stale") {
		t.Errorf("expected stale commit in git log, got:\n%s", out)
	}
}

// TestDrainLockElection verifies that when N goroutines call EnqueueAndDrain
// concurrently, exactly all items are committed and no errors occur.
func TestDrainLockElection(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	configDir := t.TempDir()

	const n = 5
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			f := filepath.Join(dir, fmt.Sprintf("note-%d.md", idx))
			if err := os.WriteFile(f, []byte(fmt.Sprintf("---\nid: %d\ntitle: note %d\n---\n", idx, idx)), 0o644); err != nil {
				errs[idx] = err
				return
			}
			errs[idx] = gitlocal.EnqueueAndDrain(configDir, dir, gitlocal.CommitItem{
				Op:      "write",
				Message: fmt.Sprintf("note: create %d", idx),
				Files:   []string{f},
			})
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}

	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	log := string(out)
	for i := range n {
		if !strings.Contains(log, fmt.Sprintf("note: create %d", i)) {
			t.Errorf("missing commit for note %d in git log:\n%s", i, log)
		}
	}
}

// TestCommitItemSkipsOutsideRepoFiles verifies that DrainQueue silently skips
// items whose files are outside the repo directory (stale cross-repo queue items).
func TestCommitItemSkipsOutsideRepoFiles(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	configDir := t.TempDir()

	// Enqueue an item pointing to a path in a completely different temp dir.
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "20990101000000-0000-outside.md")
	if err := os.WriteFile(outsideFile, []byte("---\nid: outside\ntitle: outside\n---\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := gitlocal.Enqueue(configDir, gitlocal.CommitItem{
		Op:      "write",
		Message: "note: create outside",
		Files:   []string{outsideFile},
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// DrainQueue must not return an error for an outside-repo item.
	if err := gitlocal.DrainQueue(configDir, dir); err != nil {
		t.Fatalf("DrainQueue returned error for outside-repo item: %v", err)
	}

	// The queue should be empty (item was consumed and skipped).
	entries, _ := os.ReadDir(queueDir(configDir))
	if len(entries) != 0 {
		t.Errorf("expected empty queue after drain, got %d items", len(entries))
	}
}
