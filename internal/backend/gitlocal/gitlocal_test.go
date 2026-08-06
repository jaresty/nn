package gitlocal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

func newTestNote(t *testing.T) *note.Note {
	t.Helper()
	return &note.Note{
		ID:       note.GenerateID(),
		Title:    "Test Note",
		Type:     note.TypeConcept,
		Status:   note.StatusDraft,
		Tags:     []string{"test"},
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "This is the body.",
		Links:    nil,
	}
}

func initGitRepo(t *testing.T, dir string) {
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
	// Git leaves read-only object files under .git on Linux, which prevents
	// TempDir's RemoveAll from cleaning up. Chmod everything writable then
	// remove explicitly so new read-only objects written between the walk and
	// TempDir's own RemoveAll don't cause cleanup failures.
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			return os.Chmod(path, 0700)
		})
		_ = os.RemoveAll(dir)
	})
}

func newBackend(t *testing.T) (*gitlocal.Backend, string) {
	t.Helper()
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("gitlocal.New: %v", err)
	}
	return b, dir
}

func TestWriteNote(t *testing.T) {
	b, dir := newBackend(t)
	n := newTestNote(t)
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	expected := filepath.Join(dir, n.Filename())
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected file %s not found: %v", expected, err)
	}
}

func TestReadNote(t *testing.T) {
	b, _ := newBackend(t)
	n := newTestNote(t)
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := b.Read(n.ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.ID != n.ID {
		t.Errorf("Read ID = %q, want %q", got.ID, n.ID)
	}
	if got.Title != n.Title {
		t.Errorf("Read Title = %q, want %q", got.Title, n.Title)
	}
	if got.Type != n.Type {
		t.Errorf("Read Type = %q, want %q", got.Type, n.Type)
	}
}

func TestDeleteNote(t *testing.T) {
	b, dir := newBackend(t)
	n := newTestNote(t)
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := b.Delete(n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	expected := filepath.Join(dir, n.Filename())
	if _, err := os.Stat(expected); !os.IsNotExist(err) {
		t.Fatalf("file %s still exists after Delete", expected)
	}
}

func TestWriteProducesGitCommit(t *testing.T) {
	b, dir := newBackend(t)
	n := newTestNote(t)
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	msg := string(out)
	wantPrefix := "note: create " + n.ID
	if !strings.Contains(msg, wantPrefix) {
		t.Errorf("commit message %q does not contain %q", strings.TrimSpace(msg), wantPrefix)
	}
}

func TestDeleteProducesGitCommit(t *testing.T) {
	b, dir := newBackend(t)
	n := newTestNote(t)
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := b.Delete(n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	msg := string(out)
	wantPrefix := "note: delete " + n.ID
	if !strings.Contains(msg, wantPrefix) {
		t.Errorf("commit message %q does not contain %q", strings.TrimSpace(msg), wantPrefix)
	}
}

func TestListNotes(t *testing.T) {
	b, _ := newBackend(t)
	n1 := newTestNote(t)
	n2 := newTestNote(t)
	n2.Title = "Second Note"
	if err := b.Write(n1); err != nil {
		t.Fatalf("Write n1: %v", err)
	}
	if err := b.Write(n2); err != nil {
		t.Fatalf("Write n2: %v", err)
	}
	notes, err := b.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("List() count = %d, want 2", len(notes))
	}
}

func TestUpdateDeletesOldFileOnRename(t *testing.T) {
	b, dir := newBackend(t)
	n := newTestNote(t)
	n.Title = "Old Title"
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	oldFilename := n.Filename()

	n.Title = "New Title"
	updateDone := make(chan error, 1)
	go func() { updateDone <- b.Update(n, nil) }()
	select {
	case err := <-updateDone:
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Update deadlocked while renaming a note")
	}

	if _, err := os.Stat(filepath.Join(dir, oldFilename)); !os.IsNotExist(err) {
		t.Errorf("old file %q still exists after rename", oldFilename)
	}
	newFilename := n.Filename()
	if _, err := os.Stat(filepath.Join(dir, newFilename)); err != nil {
		t.Errorf("new file %q not found after rename: %v", newFilename, err)
	}
}

// TestListSkipsDeletedFile confirms that List tolerates a file that disappears
// after ReadDir but before ReadFile (TOCTOU race). We simulate this by writing a
// valid .md file directly to the notebook directory and then removing it before
// calling List — so ReadDir would have seen it in a real race window but ReadFile
// returns ENOENT. We use a symlink trick: create the file, create a symlink to it
// with the .md name that ReadDir will enumerate, delete the real file so the
// symlink is dangling, then call List. ReadDir returns the symlink name; ReadFile
// on a dangling symlink returns ENOENT on Linux/macOS.
func TestListSkipsDeletedFile(t *testing.T) {
	b, dir := newBackend(t)

	// Write a real note so List has something to return.
	n := newTestNote(t)
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Create a dangling symlink whose name looks like a valid note file.
	// ReadDir enumerates it; os.ReadFile on a dangling symlink → ENOENT.
	target := filepath.Join(dir, "20990101000000-9999-ghost-target.md")
	link := filepath.Join(dir, "20990101000000-9999-ghost.md")
	if err := os.WriteFile(target, []byte("---\nid: 20990101000000-9999\ntitle: ghost\n---\n"), 0o644); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove target: %v", err)
	}

	notes, err := b.List()
	if err != nil {
		t.Fatalf("List returned error on dangling symlink (simulated TOCTOU): %v", err)
	}
	if len(notes) != 1 || notes[0].ID != n.ID {
		t.Errorf("expected exactly note %s, got %d notes", n.ID, len(notes))
	}
}
