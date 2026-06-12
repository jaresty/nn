package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateSinceReject(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Truncate(time.Second)
	writeNoteFile(t, nbDir, n)

	stale := n.Modified.Add(-5 * time.Second).Format(time.RFC3339)
	_, err := execute("update", n.ID, "--title", "New Title", "--since", stale, "--no-edit")
	if err == nil {
		t.Fatal("expected error when --since is stale, got nil")
	}
	if !strings.Contains(err.Error(), "note was modified since") {
		t.Errorf("expected 'note was modified since' in error, got: %v", err)
	}
}

func TestUpdateSinceMatch(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Truncate(time.Second)
	writeNoteFile(t, nbDir, n)

	since := n.Modified.Format(time.RFC3339)
	_, err := execute("update", n.ID, "--title", "New Title", "--since", since, "--no-edit")
	if err != nil {
		t.Fatalf("expected success when --since matches modified, got: %v", err)
	}
}

func TestUpdateSinceParseFail(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--title", "New Title", "--since", "not-a-timestamp", "--no-edit")
	if err == nil {
		t.Fatal("expected error for malformed --since, got nil")
	}
	if !strings.Contains(err.Error(), "--since") {
		t.Errorf("expected '--since' in error, got: %v", err)
	}
}

// TestUpdateSinceRequired asserts that omitting --since is an error.
func TestUpdateSinceRequired(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--title", "New Title", "--no-edit")
	if err == nil {
		t.Fatal("expected error when --since is omitted, got nil")
	}
	if !strings.Contains(err.Error(), "--since is required") {
		t.Errorf("expected '--since is required' in error, got: %v", err)
	}
}

// Assertion: TestUpdateModifiedUsesLocalTimezone — after nn update, the modified: field in nn show output uses local timezone, not UTC (no trailing Z).
// Uses an injected clock set to America/Los_Angeles so the test is deterministic in UTC CI environments.
func TestUpdateModifiedUsesLocalTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("timezone data unavailable")
	}
	fixedTime := time.Date(2026, 6, 11, 10, 0, 0, 0, loc)
	orig := updateNowFn
	updateNowFn = func() time.Time { return fixedTime }
	defer func() { updateNowFn = orig }()

	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "TZ Test Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Truncate(time.Second)
	writeNoteFile(t, nbDir, n)

	since := n.Modified.Format(time.RFC3339)
	_, err = execute("update", n.ID, "--title", "TZ Test Note Updated", "--since", since, "--no-edit")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}

	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "modified:") {
			if strings.HasSuffix(strings.TrimSpace(line), "Z") {
				t.Errorf("modified: field ends with Z (UTC), want local timezone; got: %s", line)
			}
			return
		}
	}
	t.Error("modified: field not found in nn show output")
}

// TestUpdateSinceStillWorks asserts that providing --since allows the update.
func TestUpdateSinceStillWorks(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Concurrent Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Truncate(time.Second)
	writeNoteFile(t, nbDir, n)

	since := n.Modified.Format(time.RFC3339)
	_, err := execute("update", n.ID, "--title", "New Title", "--since", since, "--no-edit")
	if err != nil {
		t.Fatalf("expected success with valid --since, got: %v", err)
	}
}
