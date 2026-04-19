package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Setup: global protocol = type protocol, no outgoing governs links, no inbound links.
func globalProtocolNote() *note.Note {
	n := newTestNoteForCLI(note.GenerateID(), "My Global Protocol", note.TypeProtocol)
	n.Links = nil
	return n
}

// Assertion: global protocol does NOT appear in the orphans section.
// Natural FAIL state: no — passes trivially without the feature. Perturbation: run against
// current code where global protocols ARE counted as orphans.
func TestStatusGlobalProtocolNotInOrphans(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := globalProtocolNote()
	writeNoteFile(t, nbDir, proto)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}

	// Find orphans section and confirm the protocol ID is not listed there.
	lines := strings.Split(out, "\n")
	inOrphans := false
	for _, line := range lines {
		if strings.HasPrefix(line, "orphans:") {
			inOrphans = true
			continue
		}
		if inOrphans {
			// Stop at next top-level section.
			if len(line) > 0 && line[0] != ' ' {
				break
			}
			if strings.Contains(line, proto.ID) {
				t.Errorf("global protocol %s should not appear in orphans section:\n%s", proto.ID, out)
			}
		}
	}
}

// Assertion: global protocol appears in a separate "global protocols:" section.
func TestStatusGlobalProtocolSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := globalProtocolNote()
	writeNoteFile(t, nbDir, proto)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(out, "global protocols:") {
		t.Errorf("expected 'global protocols:' section in status output:\n%s", out)
	}
	if !strings.Contains(out, proto.ID) {
		t.Errorf("expected global protocol ID %s in status output:\n%s", proto.ID, out)
	}
}

// Assertion: JSON output contains global_protocols array with the global protocol.
func TestStatusGlobalProtocolJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := globalProtocolNote()
	writeNoteFile(t, nbDir, proto)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result struct {
		Orphans []struct {
			ID string `json:"id"`
		} `json:"orphans"`
		GlobalProtocols []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"global_protocols"`
	}
	mustJSON(t, out, &result)
	if len(result.GlobalProtocols) != 1 {
		t.Fatalf("expected 1 global_protocol, got %d:\n%s", len(result.GlobalProtocols), out)
	}
	if result.GlobalProtocols[0].ID != proto.ID {
		t.Errorf("global_protocols[0].id: got %q, want %q", result.GlobalProtocols[0].ID, proto.ID)
	}
	for _, o := range result.Orphans {
		if o.ID == proto.ID {
			t.Errorf("global protocol %s should not appear in orphans array", proto.ID)
		}
	}
}

// Assertion: a protocol WITH a governs link is not in global protocols.
func TestStatusProtocolWithGovernsNotGlobal(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	proto := newTestNoteForCLI(note.GenerateID(), "Scoped Protocol", note.TypeProtocol)
	proto.Links = []note.Link{
		{TargetID: target.ID, Annotation: "governs this note", Type: "governs"},
	}
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, proto)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result struct {
		GlobalProtocols []struct {
			ID string `json:"id"`
		} `json:"global_protocols"`
	}
	mustJSON(t, out, &result)
	for _, gp := range result.GlobalProtocols {
		if gp.ID == proto.ID {
			t.Errorf("protocol with governs link should not appear in global_protocols: %s", proto.ID)
		}
	}
}
