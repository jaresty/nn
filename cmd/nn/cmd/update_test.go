package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestUpdateTitle(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Old Title", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--title", "New Title", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --title: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "New Title") {
		t.Errorf("title not updated:\n%s", out)
	}
}

func TestUpdateContent(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	n.Body = "original body"
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--content", "replaced body", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --content: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "replaced body") {
		t.Errorf("content not replaced:\n%s", out)
	}
	if strings.Contains(out, "original body") {
		t.Errorf("old content still present:\n%s", out)
	}
}

func TestUpdateAppend(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	n.Body = "original body"
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--append", "appended text", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --append: %v", err)
	}
	out, _ := execute("show", n.ID)
	if !strings.Contains(out, "original body") {
		t.Errorf("original body missing after append:\n%s", out)
	}
	if !strings.Contains(out, "appended text") {
		t.Errorf("appended text missing:\n%s", out)
	}
}

func TestUpdateTags(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--tags", "foo,bar", "--since", sinceFor(n), "--no-edit")
	if err != nil {
		t.Fatalf("nn update --tags: %v", err)
	}
	out, _ := execute("list", "--tag", "foo", "--json")
	if !strings.Contains(out, n.ID) {
		t.Errorf("note not found by updated tag:\n%s", out)
	}
}

func TestUpdateContentAndAppendMutuallyExclusive(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--content", "new", "--append", "more", "--since", sinceFor(n), "--no-edit")
	if err == nil {
		t.Fatal("nn update --content --append: want error, got nil")
	}
}

func TestUpdateRequiresFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--since", sinceFor(n), "--no-edit")
	if err == nil {
		t.Fatal("nn update with no change flags: want error, got nil")
	}
}

func TestUpdateMultipleLinkTo(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	a := newTestNoteForCLI(note.GenerateID(), "Target A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Target B", note.TypeConcept)
	writeNoteFile(t, nbDir, n)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	_, err := execute("update", n.ID, "--since", sinceFor(n), "--no-edit",
		"--link-to", a.ID, "--link-type", "grounded-by", "--annotation", "link to a",
		"--link-to", b.ID, "--link-type", "extends", "--annotation", "link to b")
	if err != nil {
		t.Fatalf("nn update with multiple --link-to: %v", err)
	}

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show after update: %v", err)
	}
	if !strings.Contains(out, a.ID) {
		t.Errorf("show output does not contain link target %s:\n%s", a.ID, out)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("show output does not contain link target %s:\n%s", b.ID, out)
	}
	if !strings.Contains(out, "[grounded-by]") || !strings.Contains(out, "[extends]") {
		t.Errorf("show output does not contain paired link types:\n%s", out)
	}
}

func TestUpdateLinkToRequiresPairedKnownLinkType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing", []string{"--link-to", "someid", "--annotation", "context"}},
		{"unknown", []string{"--link-to", "someid", "--link-type", "invented", "--annotation", "context"}},
		{"mismatched", []string{"--link-to", "a", "--link-type", "supports", "--annotation", "a", "--link-to", "b", "--annotation", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"update", n.ID, "--since", sinceFor(n), "--no-edit"}
			_, err := execute(append(args, tc.args...)...)
			if err == nil {
				t.Fatalf("nn update with %s link type: want error, got nil", tc.name)
			}
		})
	}
}

func TestUpdateLinkToMismatchedAnnotation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--since", sinceFor(n), "--no-edit",
		"--link-to", "someid", "--link-to", "otherid", "--annotation", "only one")
	if err == nil {
		t.Fatal("nn update with mismatched --link-to/--annotation: want error, got nil")
	}
}

func TestUpdateCheckFlagNoRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("update", n.ID, "--since", sinceFor(n), "--no-edit",
		"--title", "My Note Updated", "--check")
	if err != nil {
		t.Fatalf("nn update --check with no representation: %v", err)
	}
}

// Assertion: TestUpdateDailyUpserts — nn update daily --content replaces body of today's daily note (upsert: creates if absent).
func TestUpdateDailyUpserts(t *testing.T) {
	_, execute := setupNotebook(t)
	today := time.Now().Format("2006-01-02")
	todayTitle := "Daily: " + today

	_, err := execute("update", "daily", "--content", "session work today", "--no-edit")
	if err != nil {
		t.Fatalf("nn update daily (upsert): %v", err)
	}
	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily after update: %v", err)
	}
	if !strings.Contains(out, todayTitle) {
		t.Errorf("nn update daily: want note titled %q, got:\n%s", todayTitle, out)
	}
	if !strings.Contains(out, "session work today") {
		t.Errorf("nn update daily: want body 'session work today', got:\n%s", out)
	}
}
