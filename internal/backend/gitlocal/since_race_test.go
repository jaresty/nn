package gitlocal_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

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
