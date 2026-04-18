package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: text output includes governing protocols section when a governs-backlink exists.
func TestShowGoverningProtocolsText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	proto.Links = []note.Link{{TargetID: target.ID, Annotation: "governs this", Type: "governs"}}
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", target.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "governing protocols") {
		t.Errorf("expected 'governing protocols' section in output:\n%s", out)
	}
	if !strings.Contains(out, "My Protocol") {
		t.Errorf("expected protocol title 'My Protocol' in output:\n%s", out)
	}
}

// Assertion: text output omits governing protocols section when none exist.
func TestShowNoGoverningProtocolsText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Ungoverned Note", note.TypeConcept)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", target.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "governing protocols") {
		t.Errorf("expected no 'governing protocols' section when none exist:\n%s", out)
	}
}

// Assertion: JSON output includes governing_protocols array with protocol ID and title.
func TestShowGoverningProtocolsJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "JSON Protocol", note.TypeProtocol)
	target := newTestNoteForCLI(note.GenerateID(), "JSON Target", note.TypeConcept)
	proto.Links = []note.Link{{TargetID: target.ID, Annotation: "governs this", Type: "governs"}}
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", "--json", target.ID)
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	gp, ok := result["governing_protocols"]
	if !ok {
		t.Fatalf("JSON missing 'governing_protocols' key:\n%s", out)
	}
	arr, ok := gp.([]any)
	if !ok || len(arr) == 0 {
		t.Fatalf("expected non-empty governing_protocols array, got: %v", gp)
	}
	entry, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("governing_protocols entry is not an object: %v", arr[0])
	}
	if entry["title"] != "JSON Protocol" {
		t.Errorf("expected title 'JSON Protocol', got %v", entry["title"])
	}
}

// Assertion: JSON output governing_protocols is empty array when none exist.
func TestShowNoGoverningProtocolsJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Ungoverned JSON Note", note.TypeConcept)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", "--json", target.ID)
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	gp, ok := result["governing_protocols"]
	if !ok {
		t.Fatalf("JSON missing 'governing_protocols' key:\n%s", out)
	}
	arr, ok := gp.([]any)
	if !ok {
		t.Fatalf("governing_protocols should be an array, got: %T", gp)
	}
	if len(arr) != 0 {
		t.Errorf("expected empty governing_protocols array, got: %v", arr)
	}
}
