package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn backlinks <id> returns notes that link to <id>.
func TestBacklinksText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "builds on"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("backlinks", dst.ID)
	if err != nil {
		t.Fatalf("nn backlinks: %v", err)
	}
	if !strings.Contains(out, src.ID) {
		t.Errorf("expected source ID %q in backlinks output:\n%s", src.ID, out)
	}
	if !strings.Contains(out, "Source Note") {
		t.Errorf("expected source title in backlinks output:\n%s", out)
	}
}

// Assertion: --type TYPE filters backlinks to only those with matching link type.
func TestBacklinksTypeFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src1 := newTestNoteForCLI(note.GenerateID(), "Refiner", note.TypeConcept)
	src2 := newTestNoteForCLI(note.GenerateID(), "Contradictor", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	src1.Links = []note.Link{{TargetID: dst.ID, Annotation: "narrows", Type: "refines"}}
	src2.Links = []note.Link{{TargetID: dst.ID, Annotation: "disputes", Type: "contradicts"}}
	writeNoteFile(t, nbDir, src1)
	writeNoteFile(t, nbDir, src2)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("backlinks", dst.ID, "--type", "refines")
	if err != nil {
		t.Fatalf("nn backlinks --type: %v", err)
	}
	if !strings.Contains(out, src1.ID) {
		t.Errorf("expected refines source in output:\n%s", out)
	}
	if strings.Contains(out, src2.ID) {
		t.Errorf("contradicts source leaked into --type refines output:\n%s", out)
	}
}

// Assertion: --json emits JSON array with id/title/annotation/type fields.
func TestBacklinksJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "Linker", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "Linkee", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "supports", Type: "supports"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("backlinks", dst.ID, "--json")
	if err != nil {
		t.Fatalf("nn backlinks --json: %v", err)
	}
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 backlink, got %d", len(result))
	}
	if result[0]["id"] != src.ID {
		t.Errorf("id = %v, want %v", result[0]["id"], src.ID)
	}
	if result[0]["annotation"] != "supports" {
		t.Errorf("annotation = %v, want supports", result[0]["annotation"])
	}
}

// Assertion: notes with no inbound links produce empty output (not an error).
func TestBacklinksEmpty(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Lonely Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("backlinks", n.ID)
	if err != nil {
		t.Fatalf("nn backlinks on orphan should not error: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output for orphan, got %q", out)
	}
}
