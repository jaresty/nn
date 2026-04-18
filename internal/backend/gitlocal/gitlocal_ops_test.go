package gitlocal_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func newNoteWithLinks(t *testing.T) (*note.Note, *note.Note) {
	t.Helper()
	n1 := &note.Note{
		ID: note.GenerateID(), Title: "Source", Type: note.TypeConcept,
		Status: note.StatusDraft, Created: time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
	}
	n2 := &note.Note{
		ID: note.GenerateID(), Title: "Target", Type: note.TypeArgument,
		Status: note.StatusDraft, Created: time.Now().UTC().Truncate(time.Second),
		Modified: time.Now().UTC().Truncate(time.Second),
	}
	return n1, n2
}

func TestAddLink(t *testing.T) {
	b, _ := newBackend(t)
	n1, n2 := newNoteWithLinks(t)
	b.Write(n1)
	b.Write(n2)

	if err := b.AddLink(n1.ID, n2.ID, "provides context for", "", "draft"); err != nil {
		t.Fatalf("AddLink: %v", err)
	}

	got, err := b.Read(n1.ID)
	if err != nil {
		t.Fatalf("Read after AddLink: %v", err)
	}
	if len(got.Links) != 1 {
		t.Fatalf("Links count = %d, want 1", len(got.Links))
	}
	if got.Links[0].TargetID != n2.ID {
		t.Errorf("Link TargetID = %q, want %q", got.Links[0].TargetID, n2.ID)
	}
}

func TestAddLinkCommitMessage(t *testing.T) {
	b, dir := newBackend(t)
	n1, n2 := newNoteWithLinks(t)
	b.Write(n1)
	b.Write(n2)
	b.AddLink(n1.ID, n2.ID, "provides context for", "", "draft")

	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "note: link") {
		t.Errorf("commit %q does not contain 'note: link'", strings.TrimSpace(string(out)))
	}
}

func TestRemoveLink(t *testing.T) {
	b, _ := newBackend(t)
	n1, n2 := newNoteWithLinks(t)
	b.Write(n1)
	b.Write(n2)
	b.AddLink(n1.ID, n2.ID, "provides context for", "", "draft")

	if err := b.RemoveLink(n1.ID, n2.ID); err != nil {
		t.Fatalf("RemoveLink: %v", err)
	}

	got, _ := b.Read(n1.ID)
	if len(got.Links) != 0 {
		t.Errorf("Links after RemoveLink = %d, want 0", len(got.Links))
	}
}

func TestPromote(t *testing.T) {
	b, _ := newBackend(t)
	n := newTestNote(t)
	b.Write(n)

	if err := b.Promote(n.ID, note.StatusReviewed); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	got, _ := b.Read(n.ID)
	if got.Status != note.StatusReviewed {
		t.Errorf("Status after Promote = %q, want reviewed", got.Status)
	}
}

func TestPromoteCommitMessage(t *testing.T) {
	b, dir := newBackend(t)
	n := newTestNote(t)
	b.Write(n)
	b.Promote(n.ID, note.StatusReviewed)

	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "note: promote") {
		t.Errorf("commit %q does not contain 'note: promote'", strings.TrimSpace(string(out)))
	}
}
