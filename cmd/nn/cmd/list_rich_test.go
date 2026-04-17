package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: --rich adds modified, link_count, body_preview to JSON output.
func TestListRichFields(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Rich Note", note.TypeConcept)
	n.Body = "This is the body content of the note."
	n.Modified = time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	dst := newTestNoteForCLI(note.GenerateID(), "Linked", note.TypeConcept)
	n.Links = []note.Link{{TargetID: dst.ID, Annotation: "relates"}}
	writeNoteFile(t, nbDir, n)
	writeNoteFile(t, nbDir, dst)

	out, err := execute("list", "--json", "--rich")
	if err != nil {
		t.Fatalf("nn list --json --rich: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	var found map[string]interface{}
	for _, r := range results {
		if r["id"] == n.ID {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatalf("note %q not found in output", n.ID)
	}
	if _, ok := found["modified"]; !ok {
		t.Errorf("modified field missing from --rich output")
	}
	if lc, ok := found["link_count"]; !ok {
		t.Errorf("link_count field missing from --rich output")
	} else if lc.(float64) != 1 {
		t.Errorf("link_count = %v, want 1", lc)
	}
	if bp, ok := found["body_preview"]; !ok {
		t.Errorf("body_preview field missing from --rich output")
	} else if !strings.Contains(bp.(string), "This is") {
		t.Errorf("body_preview = %q, want prefix of body", bp)
	}
}

// Assertion: without --rich, modified/link_count/body_preview are absent.
func TestListJSONWithoutRichLacksExtraFields(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Plain Note", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list --json: %v", err)
	}
	if strings.Contains(out, "body_preview") {
		t.Errorf("body_preview should not appear in plain --json output")
	}
	if strings.Contains(out, "link_count") {
		t.Errorf("link_count should not appear in plain --json output")
	}
}
