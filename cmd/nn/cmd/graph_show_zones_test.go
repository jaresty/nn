package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// TestGraphShowZones verifies that `graph show --focus <id> --zones --format json`
// annotates each node with its directional zone, computed via zoneOf from the
// node's direct edge to the ego. See design note 20260820154156-6576.
//
// property Z1: E->N edge of type T  => N.zone == zoneOf(T, dirOut)
// property Z2: N->E edge of type T  => N.zone == zoneOf(T, dirIn)
// property Z3: the ego node E has empty zone
func TestGraphShowZones(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	up := newTestNoteForCLI(note.GenerateID(), "Up", note.TypeConcept)                  // ego -> up (extends out => TOP)
	challenger := newTestNoteForCLI(note.GenerateID(), "Challenger", note.TypeArgument) // challenger -> ego (questions in => LEFT)
	ego.Created, ego.Modified = now, now
	up.Created, up.Modified = now, now
	challenger.Created, challenger.Modified = now, now

	ego.Links = []note.Link{{TargetID: up.ID, Type: "extends", Annotation: "builds on"}}
	challenger.Links = []note.Link{{TargetID: ego.ID, Type: "questions", Annotation: "challenges"}}

	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, up)
	writeNoteFile(t, nbDir, challenger)

	out, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--format", "json")
	if err != nil {
		t.Fatalf("graph show --zones: %v", err)
	}

	var result struct {
		Nodes []struct {
			ID   string `json:"id"`
			Zone string `json:"zone"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("graph show --zones: output not valid JSON: %v\n%s", err, out)
	}

	zoneByID := map[string]string{}
	for _, n := range result.Nodes {
		zoneByID[n.ID] = n.Zone
	}

	// property Z1: ego -> up via extends (out) => top
	if zoneByID[up.ID] != "top" {
		t.Errorf("property Z1: node Up zone = %q, want %q", zoneByID[up.ID], "top")
	}
	// property Z2: challenger -> ego via questions (in) => left
	if zoneByID[challenger.ID] != "left" {
		t.Errorf("property Z2: node Challenger zone = %q, want %q", zoneByID[challenger.ID], "left")
	}
	// property Z3: ego itself has empty zone
	if zoneByID[ego.ID] != "" {
		t.Errorf("property Z3: ego zone = %q, want empty", zoneByID[ego.ID])
	}
}

func TestGraphShowZonesReciprocalEvidenceEdgesAgree(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	claim := newTestNoteForCLI(note.GenerateID(), "Claim", note.TypeConcept)
	evidence := newTestNoteForCLI(note.GenerateID(), "Evidence", note.TypeObservation)
	claim.Links = []note.Link{{TargetID: evidence.ID, Type: "grounded-by", Annotation: "depends on evidence"}}
	evidence.Links = []note.Link{{TargetID: claim.ID, Type: "supports", Annotation: "corroborates claim"}}
	writeNoteFile(t, nbDir, claim)
	writeNoteFile(t, nbDir, evidence)

	zoneFor := func(focusID, relatedID string) string {
		out, err := execute("graph", "show", "--focus", focusID, "--depth", "1", "--direction", "both", "--zones", "--format", "json")
		if err != nil {
			t.Fatalf("graph show reciprocal evidence zones: %v", err)
		}
		var result struct {
			Nodes []struct {
				ID   string `json:"id"`
				Zone string `json:"zone"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("graph show reciprocal evidence JSON: %v\n%s", err, out)
		}
		for _, n := range result.Nodes {
			if n.ID == relatedID {
				return n.Zone
			}
		}
		return "missing"
	}

	if got := zoneFor(claim.ID, evidence.ID); got != "top" {
		t.Errorf("claim should see reciprocal evidence in TOP, got %q", got)
	}
	if got := zoneFor(evidence.ID, claim.ID); got != "bottom" {
		t.Errorf("evidence should see reciprocal claim in BOTTOM, got %q", got)
	}
}

func TestGraphShowZonesExposesDirectUntypedLinksAsUnclassified(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	legacy := newTestNoteForCLI(note.GenerateID(), "Legacy Neighbor", note.TypeConcept)
	ego.Links = []note.Link{{TargetID: legacy.ID, Annotation: "stored context"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, legacy)

	jsonOut, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--format", "json")
	if err != nil {
		t.Fatalf("graph show unclassified JSON: %v", err)
	}
	var result struct {
		Nodes []struct {
			ID   string `json:"id"`
			Zone string `json:"zone"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("decode graph show unclassified JSON: %v\n%s", err, jsonOut)
	}
	zone := "missing"
	for _, n := range result.Nodes {
		if n.ID == legacy.ID {
			zone = n.Zone
		}
	}
	if zone != "unclassified" {
		t.Errorf("legacy neighbor zone = %q, want unclassified", zone)
	}

	text, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--format", "text")
	if err != nil {
		t.Fatalf("graph show unclassified text: %v", err)
	}
	if !strings.Contains(text, "UNCLASSIFIED") || !strings.Contains(text, legacy.ID) {
		t.Errorf("untyped direct neighbor omitted from UNCLASSIFIED category:\n%s", text)
	}
	if !strings.Contains(text, "stored relationships without navigation semantics") {
		t.Errorf("untyped neighborhood omitted UNCLASSIFIED legend row:\n%s", text)
	}
	for _, semantic := range []string{"TOP\n  " + legacy.ID, "BOTTOM\n  " + legacy.ID, "LEFT\n  " + legacy.ID, "RIGHT\n  " + legacy.ID} {
		if strings.Contains(text, semantic) {
			t.Errorf("untyped neighbor assigned semantic zone %q:\n%s", semantic, text)
		}
	}
}

// TestGraphShowZonesText verifies that `--zones --format text` groups nodes
// under directional zone headers.
//
// property T1: a zoned node's line appears under its zone's header (TOP/BOTTOM/LEFT/RIGHT).
func TestGraphShowZonesTextOmitsUnclassifiedLegendWithoutVisibleUntypedEdge(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	typed := newTestNoteForCLI(note.GenerateID(), "Typed Neighbor", note.TypeConcept)
	ego.Links = []note.Link{{TargetID: typed.ID, Type: "extends", Annotation: "typed context"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, typed)

	text, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--format", "text")
	if err != nil {
		t.Fatalf("graph show typed-only text: %v", err)
	}
	if strings.Contains(text, "stored relationships without navigation semantics") {
		t.Errorf("typed-only neighborhood included misleading UNCLASSIFIED legend row:\n%s", text)
	}
}

func TestGraphShowZonesText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	up := newTestNoteForCLI(note.GenerateID(), "Upnote", note.TypeConcept)
	challenger := newTestNoteForCLI(note.GenerateID(), "Challengernote", note.TypeArgument)
	ego.Created, ego.Modified = now, now
	up.Created, up.Modified = now, now
	challenger.Created, challenger.Modified = now, now

	ego.Links = []note.Link{{TargetID: up.ID, Type: "extends", Annotation: "builds on"}}
	challenger.Links = []note.Link{{TargetID: ego.ID, Type: "questions", Annotation: "challenges"}}

	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, up)
	writeNoteFile(t, nbDir, challenger)

	out, err := execute("graph", "show", "--focus", ego.ID, "--depth", "1", "--direction", "both", "--zones", "--format", "text")
	if err != nil {
		t.Fatalf("graph show --zones --format text: %v", err)
	}

	// property T1: zone headers present and each node under the right header.
	// Split output into sections by header to check membership.
	section := func(header string) string {
		lines := strings.Split(out, "\n")
		var buf []string
		in := false
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if isZoneHeader(trimmed) {
				in = trimmed == header
				continue
			}
			if in {
				buf = append(buf, l)
			}
		}
		return strings.Join(buf, "\n")
	}

	if !strings.Contains(section("TOP"), up.ID) {
		t.Errorf("property T1: expected Upnote (%s) under TOP header, got:\n%s", up.ID, out)
	}
	if !strings.Contains(section("LEFT"), challenger.ID) {
		t.Errorf("property T1: expected Challengernote (%s) under LEFT header, got:\n%s", challenger.ID, out)
	}
}

// isZoneHeader reports whether a trimmed line is a graph-show category header.
func isZoneHeader(s string) bool {
	switch s {
	case "TOP", "BOTTOM", "LEFT", "RIGHT", "UNCLASSIFIED":
		return true
	}
	return false
}
