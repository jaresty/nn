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
}

func newBackend(t *testing.T) (*gitlocal.Backend, string) {
	t.Helper()
	dir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.New(dir)
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
	if err := b.Update(n); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, oldFilename)); !os.IsNotExist(err) {
		t.Errorf("old file %q still exists after rename", oldFilename)
	}
	newFilename := n.Filename()
	if _, err := os.Stat(filepath.Join(dir, newFilename)); err != nil {
		t.Errorf("new file %q not found after rename: %v", newFilename, err)
	}
}
