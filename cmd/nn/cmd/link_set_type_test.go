package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestLinkSetTypeAtomicallyTypesLegacyLink(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "To", note.TypeObservation)
	from.Links = []note.Link{{TargetID: to.ID, Annotation: "evidence basis", Status: "reviewed"}}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)
	commitNoteFile(t, nbDir, from)
	commitNoteFile(t, nbDir, to)
	before := gitCommitCount(t, nbDir)

	out, err := execute("link", "set-type", from.ID, to.ID, "--type", "grounded-by")
	if err != nil {
		t.Fatalf("nn link set-type: %v", err)
	}
	if !strings.Contains(out, "typed link "+from.ID+" → "+to.ID) {
		t.Errorf("unexpected output: %s", out)
	}
	if got := gitCommitCount(t, nbDir) - before; got != 1 {
		t.Fatalf("set-type commits = %d, want 1", got)
	}

	show, err := execute("show", from.ID)
	if err != nil {
		t.Fatalf("show after set-type: %v", err)
	}
	for _, want := range []string{to.ID, "[grounded-by]", "{reviewed}", "evidence basis"} {
		if !strings.Contains(show, want) {
			t.Errorf("set-type did not preserve link field %q:\n%s", want, show)
		}
	}
	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = nbDir
	msg, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	wantMessage := "note: type link " + from.ID + " → " + to.ID + " as grounded-by"
	if strings.TrimSpace(string(msg)) != wantMessage {
		t.Errorf("commit message = %q, want %q", strings.TrimSpace(string(msg)), wantMessage)
	}
}

func TestLinkSetTypeRejectsAmbiguityAndSupportsAnnotationDiscriminator(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	from.Links = []note.Link{
		{TargetID: to.ID, Annotation: "first evidence"},
		{TargetID: to.ID, Annotation: "second evidence"},
	}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)

	if _, err := execute("link", "set-type", from.ID, to.ID, "--type", "supports"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("set-type ambiguous error = %v, want ambiguity rejection", err)
	}
	if _, err := execute("link", "set-type", from.ID, to.ID, "--annotation-matches", "second", "--type", "supports"); err != nil {
		t.Fatalf("set-type with annotation discriminator: %v", err)
	}

	show, err := execute("show", from.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(show, "[["+to.ID+"|To]] — first evidence") {
		t.Errorf("non-matching legacy link changed:\n%s", show)
	}
	if !strings.Contains(show, "[["+to.ID+"|To]] [supports] — second evidence") {
		t.Errorf("matching link not typed or annotation changed:\n%s", show)
	}
}

func TestLinkSetTypeRejectsUnknownOrAlreadyTypedRelationship(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	from := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	to := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	from.Links = []note.Link{{TargetID: to.ID, Type: "extends", Annotation: "already semantic"}}
	writeNoteFile(t, nbDir, from)
	writeNoteFile(t, nbDir, to)

	if _, err := execute("link", "set-type", from.ID, to.ID, "--type", "invented"); err == nil {
		t.Fatal("set-type unknown type: want error, got nil")
	}
	if _, err := execute("link", "set-type", from.ID, to.ID, "--type", "supports"); err == nil || !strings.Contains(err.Error(), "already typed") {
		t.Fatalf("set-type already typed error = %v, want rejection", err)
	}
}
