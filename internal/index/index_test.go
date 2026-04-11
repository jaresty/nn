package index_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/index"
	"github.com/jaresty/nn/internal/note"
)

func makeNoteFile(t *testing.T, dir string, n *note.Note) {
	t.Helper()
	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	path := filepath.Join(dir, n.Filename())
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestSchemaCreate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	idx, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	// Verify tables exist by querying them — each query fails if table absent
	tables := []string{"notes", "links", "tags"}
	for _, tbl := range tables {
		if err := idx.TableExists(tbl); err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}
}

func TestRebuild(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "index.db")

	n1 := &note.Note{
		ID:       "20260411120001-0001",
		Title:    "First Note",
		Type:     note.TypeConcept,
		Status:   note.StatusDraft,
		Tags:     []string{"alpha"},
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "First body.",
		Links:    nil,
	}
	n2 := &note.Note{
		ID:      "20260411120002-0002",
		Title:   "Second Note",
		Type:    note.TypeArgument,
		Status:  note.StatusReviewed,
		Tags:    []string{"beta"},
		Created: time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:    "Second body.",
		Links: []note.Link{
			{TargetID: "20260411120001-0001", Annotation: "builds on this"},
		},
	}

	makeNoteFile(t, dir, n1)
	makeNoteFile(t, dir, n2)

	idx, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	if err := idx.Rebuild(dir); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	// notes table has 2 rows
	count, err := idx.CountNotes()
	if err != nil {
		t.Fatalf("CountNotes: %v", err)
	}
	if count != 2 {
		t.Errorf("CountNotes = %d, want 2", count)
	}

	// links table has 1 row
	linkCount, err := idx.CountLinks()
	if err != nil {
		t.Fatalf("CountLinks: %v", err)
	}
	if linkCount != 1 {
		t.Errorf("CountLinks = %d, want 1", linkCount)
	}
}

func TestStaleDetection(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "index.db")

	n := &note.Note{
		ID:       "20260411120003-0003",
		Title:    "Stale Test",
		Type:     note.TypeConcept,
		Status:   note.StatusDraft,
		Tags:     nil,
		Created:  time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
		Body:     "Original body.",
		Links:    nil,
	}
	makeNoteFile(t, dir, n)

	idx, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	if err := idx.Rebuild(dir); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	// Not stale immediately after rebuild
	stale, err := idx.IsStale(n.ID, filepath.Join(dir, n.Filename()))
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Error("IsStale = true immediately after Rebuild, want false")
	}

	// Modify the file
	n.Body = "Modified body."
	makeNoteFile(t, dir, n)

	// Now should be stale
	stale, err = idx.IsStale(n.ID, filepath.Join(dir, n.Filename()))
	if err != nil {
		t.Fatalf("IsStale after modify: %v", err)
	}
	if !stale {
		t.Error("IsStale = false after file modification, want true")
	}
}
