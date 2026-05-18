package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion (SH1): nn show <id> output contains the resolved title of each outgoing
// link, not just the raw target ID.
func TestShowResolvesOutgoingLinkTitles(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	target := newTestNoteForCLI("20260101000000-0200", "Target Note Title", note.TypeConcept)
	target.Body = "target body"

	src := newTestNoteForCLI("20260101000000-0201", "Source Note", note.TypeConcept)
	src.Body = "source body"
	src.Links = []note.Link{
		{TargetID: target.ID, Annotation: "see also", Type: "related"},
	}

	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, src)

	out, err := execute("show", src.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}

	if !strings.Contains(out, "Target Note Title") {
		t.Errorf("show output should contain resolved link title %q, got:\n%s", "Target Note Title", out)
	}
}

// Assertion (SH2): nn show <id> output contains a Backlinks section listing notes
// that link to the shown note.
func TestShowBacklinksSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	shown := newTestNoteForCLI("20260101000000-0210", "Shown Note", note.TypeConcept)
	shown.Body = "shown body"

	backer := newTestNoteForCLI("20260101000000-0211", "Backer Note", note.TypeConcept)
	backer.Body = "backer body"
	backer.Links = []note.Link{
		{TargetID: shown.ID, Annotation: "related to shown", Type: "related"},
	}

	writeNoteFile(t, nbDir, shown)
	writeNoteFile(t, nbDir, backer)

	out, err := execute("show", shown.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}

	if !strings.Contains(out, "## Backlinks") {
		t.Errorf("show output should contain '## Backlinks' section, got:\n%s", out)
	}
	if !strings.Contains(out, "Backer Note") {
		t.Errorf("show backlinks section should contain backer note title %q, got:\n%s", "Backer Note", out)
	}
}
