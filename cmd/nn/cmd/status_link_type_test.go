package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: plain text output includes unknown link type count.
func TestStatusUnknownLinkTypesText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "test", Type: "bogus-type"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(out, "unknown link types") {
		t.Errorf("expected 'unknown link types' in status output:\n%s", out)
	}
	if !strings.Contains(out, "1") {
		t.Errorf("expected count 1 in status output:\n%s", out)
	}
}

// Assertion: --json includes unknown_link_types count.
func TestStatusUnknownLinkTypesJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "test", Type: "bogus-type"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if _, ok := result["unknown_link_types"]; !ok {
		t.Errorf("unknown_link_types field missing from status JSON:\n%s", out)
	}
	if result["unknown_link_types"].(float64) != 1 {
		t.Errorf("unknown_link_types = %v, want 1", result["unknown_link_types"])
	}
}

func TestStatusSeparatesMissingFromUnknownLinkTypes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	missingDst := newTestNoteForCLI(note.GenerateID(), "Missing Type Target", note.TypeConcept)
	unknownDst := newTestNoteForCLI(note.GenerateID(), "Unknown Type Target", note.TypeConcept)
	src.Links = []note.Link{
		{TargetID: missingDst.ID, Annotation: "legacy", Type: ""},
		{TargetID: unknownDst.ID, Annotation: "experimental", Type: "invented"},
	}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, missingDst)
	writeNoteFile(t, nbDir, unknownDst)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if result["missing_link_types"] != float64(1) {
		t.Errorf("missing_link_types = %v, want 1", result["missing_link_types"])
	}
	if result["unknown_link_types"] != float64(1) {
		t.Errorf("unknown_link_types = %v, want 1", result["unknown_link_types"])
	}

	text, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(text, "missing link types: 1") || !strings.Contains(text, src.ID+"→"+missingDst.ID) {
		t.Errorf("missing type detail lacks count or endpoints:\n%s", text)
	}
	if !strings.Contains(text, "unknown link types: 1") || !strings.Contains(text, src.ID+"→"+unknownDst.ID) {
		t.Errorf("unknown type detail lacks count or endpoints:\n%s", text)
	}
}

// Assertion: notes with only known types report 0 unknown.
func TestStatusKnownTypesZeroUnknown(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	src := newTestNoteForCLI(note.GenerateID(), "From", note.TypeConcept)
	dst := newTestNoteForCLI(note.GenerateID(), "To", note.TypeConcept)
	src.Links = []note.Link{{TargetID: dst.ID, Annotation: "test", Type: "refines"}}
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if result["unknown_link_types"].(float64) != 0 {
		t.Errorf("unknown_link_types = %v, want 0 for known type", result["unknown_link_types"])
	}
}
