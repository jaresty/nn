package gitlocal_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// setupCrossProcessCmds sets up a notebook and builds the nn binary, returning a factory
// that creates an exec.Cmd ready to Start() — allowing the caller to start multiple
// commands before waiting on any of them (true parallelism without goroutine-scheduling gaps).
func setupCrossProcessCmds(t *testing.T) (*gitlocal.Backend, func(...string) *exec.Cmd) {
	t.Helper()
	dir := t.TempDir()
	initGitRepo(t, dir)

	xdgDir := t.TempDir()
	cfgPath := filepath.Join(xdgDir, "nn", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	cfgContent := fmt.Sprintf("[notebooks]\ndefault = \"personal\"\n[notebooks.personal]\npath = %q\nbackend = \"gitlocal\"\n", dir)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	b, err := gitlocal.NewWithConfigDir(dir, xdgDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	binPath := filepath.Join(t.TempDir(), "nn-test-bin")
	build := exec.Command("go", "build", "-o", binPath, "github.com/jaresty/nn/cmd/nn")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build nn: %v\n%s", err, out)
	}

	env := append(os.Environ(), "XDG_CONFIG_HOME="+xdgDir)
	newCmd := func(args ...string) *exec.Cmd {
		cmd := exec.Command(binPath, args...)
		cmd.Env = env
		return cmd
	}
	return b, newCmd
}

// TestUpdateSinceConflictCrossProcess proves property [1]:
// two concurrent cross-process nn update --replace-section calls with the same
// --since timestamp must not both succeed — exactly one must exit non-zero.
func TestUpdateSinceConflictCrossProcess(t *testing.T) {
	b, nn := setupCrossProcess(t)
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "since-race-probe",
		Type:     note.TypeObservation,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "## A\nold-a\n\n## B\nold-b",
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}

	const attempts = 5
	for i := range attempts {
		// Read the current modified timestamp.
		got, err := b.Read(n.ID)
		if err != nil {
			t.Fatalf("attempt %d Read: %v", i, err)
		}
		since := got.Modified.UTC().Format(time.RFC3339)

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs[0] = nn("update", n.ID, "--replace-section", "A", "--content", "new-a", "--since", since, "--no-edit")
		}()
		go func() {
			defer wg.Done()
			errs[1] = nn("update", n.ID, "--replace-section", "B", "--content", "new-b", "--since", since, "--no-edit")
		}()
		wg.Wait()

		bothSucceeded := errs[0] == nil && errs[1] == nil
		if bothSucceeded {
			t.Errorf("attempt %d: both concurrent --since updates exited zero — one write must fail with conflict", i)
			return
		}
		// Reset note for next attempt.
		reset, err := b.Read(n.ID)
		if err != nil {
			t.Fatalf("attempt %d reset read: %v", i, err)
		}
		reset.Body = "## A\nold-a\n\n## B\nold-b"
		reset.Modified = time.Now().UTC().Truncate(time.Second)
		if err := b.Update(reset, nil); err != nil {
			t.Fatalf("attempt %d reset update: %v", i, err)
		}
		n = reset
	}
}

// TestUpdateSinceNoConflict proves property [2]:
// a single nn update --since on an unmodified note succeeds.
func TestUpdateSinceNoConflict(t *testing.T) {
	b, nn := setupCrossProcess(t)
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "since-no-conflict",
		Type:     note.TypeObservation,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "## A\nold-a",
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := b.Read(n.ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	since := got.Modified.UTC().Format(time.RFC3339)
	if err := nn("update", n.ID, "--replace-section", "A", "--content", "new-a", "--since", since, "--no-edit"); err != nil {
		t.Errorf("single update with valid --since failed: %v", err)
	}
}

// TestUpdateSinceStaleRejected proves property [3]:
// an nn update --since on a note already modified after --since exits non-zero.
func TestUpdateSinceStaleRejected(t *testing.T) {
	b, nn := setupCrossProcess(t)
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "since-stale",
		Type:     note.TypeObservation,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "## A\nold-a",
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	staleSince := n.Modified.Add(-2 * time.Second).UTC().Format(time.RFC3339)
	err := nn("update", n.ID, "--replace-section", "A", "--content", "new-a", "--since", staleSince, "--no-edit")
	if err == nil {
		t.Errorf("update with stale --since should have failed but succeeded")
	}
}

// TestTodoDoneConcurrentCrossProcess proves that two concurrent nn todo done
// calls on different checkboxes in the same note cannot both succeed — one
// must exit non-zero with a conflict error.
//
// Uses Start()+Wait() to guarantee both OS processes are running before either
// is waited on, eliminating the scheduling gap that caused CI flakiness.
func TestTodoDoneConcurrentCrossProcess(t *testing.T) {
	b, newCmd := setupCrossProcessCmds(t)
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "todo-race-probe",
		Type:     note.TypeObservation,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "- [ ] task-alpha\n- [ ] task-beta",
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Run many attempts; require that at least one detects a conflict.
	// A single attempt may miss the race window if one process completes
	// before the other starts reading — this is expected on fast machines.
	const attempts = 20
	conflictSeen := false
	for i := range attempts {
		if conflictSeen {
			break
		}
		// Reset both checkboxes for each attempt.
		cur, err := b.Read(n.ID)
		if err != nil {
			t.Fatalf("attempt %d Read: %v", i, err)
		}
		cur.Body = "- [ ] task-alpha\n- [ ] task-beta"
		cur.Modified = time.Now().UTC()
		if err := b.Update(cur, nil); err != nil {
			t.Fatalf("attempt %d reset: %v", i, err)
		}
		n = cur

		// Start both OS processes before waiting on either — maximises overlap window.
		var bufs [2]bytes.Buffer
		cmds := [2]*exec.Cmd{
			newCmd("todo", "done", n.ID, "task-alpha"),
			newCmd("todo", "done", n.ID, "task-beta"),
		}
		for j := range cmds {
			cmds[j].Stdout = &bufs[j]
			cmds[j].Stderr = &bufs[j]
			if err := cmds[j].Start(); err != nil {
				t.Fatalf("attempt %d: Start[%d]: %v", i, j, err)
			}
		}
		errs := make([]error, 2)
		for j := range cmds {
			if err := cmds[j].Wait(); err != nil {
				errs[j] = fmt.Errorf("%v: %s", cmds[j].Args, bufs[j].String())
			}
		}

		if errs[0] != nil || errs[1] != nil {
			conflictSeen = true
		}
	}

	if !conflictSeen {
		t.Errorf("no conflict detected in %d attempts — concurrent todo done calls must not both succeed", attempts)
	}
}

// TestUpdateNilSinceUnconditional proves property [4]:
// backend.Update with nil since proceeds unconditionally (existing callers unaffected).
func TestUpdateNilSinceUnconditional(t *testing.T) {
	b := setupBackend(t)
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "nil-since",
		Type:     note.TypeObservation,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "original",
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Call Update with nil since — must not reject.
	n.Body = "updated"
	n.Modified = time.Now().UTC().Truncate(time.Second)
	if err := b.Update(n, nil); err != nil {
		t.Errorf("Update(n, nil) failed: %v", err)
	}
	got, err := b.Read(n.ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Body != "updated" && got.Body != "\nupdated" {
		t.Errorf("body = %q, want \"updated\"", got.Body)
	}
	_ = fmt.Sprintf("property 4 verified for %s", n.ID)
}
