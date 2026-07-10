package gitlocal_test

import (
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
)

// TestNoOpCommitIsIdempotent verifies that writing/promoting a note to the same
// state twice succeeds on the second call instead of failing with "nothing to commit".
func TestNoOpCommitIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "Idempotent Note",
		Type:     note.TypeConcept,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	// Writing the same note again should not error.
	if err := b.Write(n); err != nil {
		t.Errorf("second Write (no-op): %v", err)
	}
	// Promoting to the same status should not error.
	if err := b.Promote(n.ID, note.StatusDraft); err != nil {
		t.Errorf("Promote to same status (no-op): %v", err)
	}
}

// TestNoOpBulkWriteIsIdempotent verifies that BulkWrite succeeds even when the
// written files produce no staged changes (nothing to commit).
func TestNoOpBulkWriteIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	configDir := t.TempDir()
	initGitRepo(t, dir)
	b, err := gitlocal.NewWithConfigDir(dir, configDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Write the note via Write first so it's tracked by git.
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    "Bulk Idempotent A",
		Type:     note.TypeConcept,
		Status:   note.StatusDraft,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
	}
	if err := b.Write(n); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// BulkWrite the identical note — file bytes unchanged, nothing staged.
	if err := b.BulkWrite([]*note.Note{n}); err != nil {
		t.Errorf("BulkWrite no-op: %v", err)
	}
}
