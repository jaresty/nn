package gitlocal_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend"
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

func TestNewNoteWritesRejectMissingAndUnknownLinkTypes(t *testing.T) {
	for _, linkType := range []string{"", "invented"} {
		t.Run(linkType, func(t *testing.T) {
			newLinkedNote := func(title string) *note.Note {
				n, _ := newNoteWithLinks(t)
				n.Title = title
				n.Links = []note.Link{{TargetID: "legacy-target", Annotation: "context", Type: linkType}}
				return n
			}

			b, _ := newBackend(t)
			if err := b.Write(newLinkedNote("single")); err == nil {
				t.Fatalf("Write type %q: want error, got nil", linkType)
			}
			if err := b.BulkWrite([]*note.Note{newLinkedNote("bulk")}); err == nil {
				t.Fatalf("BulkWrite type %q: want error, got nil", linkType)
			}
			if err := b.BulkApply([]*note.Note{newLinkedNote("apply")}, nil); err == nil {
				t.Fatalf("BulkApply new-note type %q: want error, got nil", linkType)
			}
		})
	}
}

func TestAddLink(t *testing.T) {
	b, _ := newBackend(t)
	n1, n2 := newNoteWithLinks(t)
	b.Write(n1)
	b.Write(n2)

	if err := b.AddLink(n1.ID, n2.ID, "provides context for", "supports", "draft"); err != nil {
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

func TestAddLinkRejectsMissingAndUnknownTypes(t *testing.T) {
	for _, linkType := range []string{"", "invented"} {
		t.Run(linkType, func(t *testing.T) {
			b, _ := newBackend(t)
			n1, n2 := newNoteWithLinks(t)
			if err := b.Write(n1); err != nil {
				t.Fatal(err)
			}
			if err := b.Write(n2); err != nil {
				t.Fatal(err)
			}
			if err := b.AddLink(n1.ID, n2.ID, "context", linkType, "draft"); err == nil {
				t.Fatalf("AddLink type %q: want error, got nil", linkType)
			}
		})
	}
}

func TestAddLinksRejectsMissingAndUnknownTypes(t *testing.T) {
	for _, linkType := range []string{"", "invented"} {
		t.Run(linkType, func(t *testing.T) {
			b, _ := newBackend(t)
			n1, n2 := newNoteWithLinks(t)
			if err := b.Write(n1); err != nil {
				t.Fatal(err)
			}
			if err := b.Write(n2); err != nil {
				t.Fatal(err)
			}
			err := b.AddLinks(n1.ID, []backend.LinkTarget{{ToID: n2.ID, Annotation: "context", Type: linkType, Status: "draft"}})
			if err == nil {
				t.Fatalf("AddLinks type %q: want error, got nil", linkType)
			}
		})
	}
}

func TestLinkTypeUpdatesRejectMissingAndUnknownTypes(t *testing.T) {
	for _, linkType := range []string{"", "invented"} {
		t.Run(linkType, func(t *testing.T) {
			b, _ := newBackend(t)
			n1, n2 := newNoteWithLinks(t)
			if err := b.Write(n1); err != nil {
				t.Fatal(err)
			}
			if err := b.Write(n2); err != nil {
				t.Fatal(err)
			}
			if err := b.AddLink(n1.ID, n2.ID, "context", "supports", "draft"); err != nil {
				t.Fatal(err)
			}
			if err := b.UpdateLink(n1.ID, n2.ID, nil, &linkType, nil); err == nil {
				t.Fatalf("UpdateLink type %q: want error, got nil", linkType)
			}
			updates := []backend.LinkUpdate{{ToID: n2.ID, Type: &linkType}}
			if err := b.BulkUpdateLinks(n1.ID, updates); err == nil {
				t.Fatalf("BulkUpdateLinks type %q: want error, got nil", linkType)
			}
		})
	}
}

func TestAddLinkCommitMessage(t *testing.T) {
	b, dir := newBackend(t)
	n1, n2 := newNoteWithLinks(t)
	b.Write(n1)
	b.Write(n2)
	b.AddLink(n1.ID, n2.ID, "provides context for", "supports", "draft")

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
	b.AddLink(n1.ID, n2.ID, "provides context for", "supports", "draft")

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

	if err := b.Promote(n.ID, n.Modified, note.StatusReviewed); err != nil {
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
	b.Promote(n.ID, n.Modified, note.StatusReviewed)

	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = dir
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "note: promote") {
		t.Errorf("commit %q does not contain 'note: promote'", strings.TrimSpace(string(out)))
	}
}
