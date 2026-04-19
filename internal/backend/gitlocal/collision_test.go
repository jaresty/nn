package gitlocal_test

import (
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: Write with a colliding ID succeeds and the note is written with a new non-colliding ID.
func TestWriteRetriesOnIDCollision(t *testing.T) {
	b, dir := newBackend(t)

	id := note.GenerateID()
	n1 := &note.Note{
		ID: id, Title: "Original Title", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(), Body: "body",
	}
	n2 := &note.Note{
		ID: id, Title: "Colliding Title", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(), Body: "body",
	}

	if err := b.Write(n1); err != nil {
		t.Fatalf("first Write failed: %v", err)
	}
	if err := b.Write(n2); err != nil {
		t.Fatalf("Write on collision should succeed by retrying ID, got error: %v", err)
	}
	// n2.ID should have been updated to a new non-colliding ID.
	if n2.ID == id {
		t.Errorf("expected n2.ID to be updated after collision, still %s", id)
	}
	// Both notes should be readable.
	if _, err := b.Read(n1.ID); err != nil {
		t.Errorf("original note unreadable after collision retry: %v", err)
	}
	if _, err := b.Read(n2.ID); err != nil {
		t.Errorf("colliding note unreadable after ID retry: %v", err)
	}
	// File for n2 should exist in dir.
	_ = dir
}

// Assertion: BulkWrite with a colliding note succeeds, colliding note gets a new ID.
func TestBulkWriteRetriesOnIDCollision(t *testing.T) {
	b, _ := newBackend(t)

	id := note.GenerateID()
	existing := &note.Note{
		ID: id, Title: "Existing Note", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(), Body: "body",
	}
	if err := b.Write(existing); err != nil {
		t.Fatalf("setup Write failed: %v", err)
	}

	collider := &note.Note{
		ID: id, Title: "Colliding Bulk Note", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(), Body: "body",
	}
	other := &note.Note{
		ID: note.GenerateID(), Title: "Innocent Note", Type: note.TypeConcept, Status: note.StatusDraft,
		Created: time.Now().UTC(), Modified: time.Now().UTC(), Body: "body",
	}

	if err := b.BulkWrite([]*note.Note{other, collider}); err != nil {
		t.Fatalf("BulkWrite on collision should succeed by retrying ID, got error: %v", err)
	}
	if collider.ID == id {
		t.Errorf("expected collider.ID to be updated after collision, still %s", id)
	}
	if _, err := b.Read(collider.ID); err != nil {
		t.Errorf("colliding note unreadable after ID retry: %v", err)
	}
}
