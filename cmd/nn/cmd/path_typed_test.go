package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type typedPathResult struct {
	Witnesses []typedWitnessTest `json:"witnesses"`
}

func setupTypedPathGraph(t *testing.T) (func(...string) (string, error), *note.Note, *note.Note, *note.Note) {
	t.Helper()
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI("20260101000000-0001", "Claim", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0002", "Evidence bridge", note.TypeConcept)
	c := newTestNoteForCLI("20260101000000-0003", "Observation", note.TypeObservation)
	a.Links = []note.Link{
		{TargetID: c.ID, Type: "extends", Annotation: "short but disallowed"},
		{TargetID: b.ID, Type: "supports", Annotation: "corroborated by bridge"},
	}
	b.Links = []note.Link{{TargetID: c.ID, Type: "grounded-by", Annotation: "based on observation"}}
	for _, n := range []*note.Note{a, b, c} {
		writeNoteFile(t, nbDir, n)
	}
	return execute, a, b, c
}

func TestPathLinksFiltersDirectedShortestWitness(t *testing.T) {
	execute, a, b, c := setupTypedPathGraph(t)
	out, err := execute("path", a.ID, c.ID, "--links", "supports,grounded-by", "--json")
	if err != nil {
		t.Fatalf("typed path: %v", err)
	}
	var got typedPathResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("typed path JSON: %v\n%s", err, out)
	}
	if len(got.Witnesses) != 1 {
		t.Fatalf("typed witnesses = %#v, want one", got.Witnesses)
	}
	witness := got.Witnesses[0]
	if len(witness.Nodes) != 3 || witness.Nodes[0].ID != a.ID || witness.Nodes[1].ID != b.ID || witness.Nodes[2].ID != c.ID {
		t.Fatalf("typed nodes = %#v, want A→B→C", witness.Nodes)
	}
	if len(witness.Edges) != 2 {
		t.Fatalf("typed edges = %#v, want 2", witness.Edges)
	}
	if witness.Edges[0].From != a.ID || witness.Edges[0].To != b.ID || witness.Edges[0].Type != "supports" || witness.Edges[0].Annotation != "corroborated by bridge" {
		t.Fatalf("first typed edge = %#v", witness.Edges[0])
	}
	if witness.Edges[1].From != b.ID || witness.Edges[1].To != c.ID || witness.Edges[1].Type != "grounded-by" {
		t.Fatalf("second typed edge = %#v", witness.Edges[1])
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &shape); err != nil || len(shape) != 1 || shape["witnesses"] == nil {
		t.Fatalf("typed path shape = %v, err=%v; want only witnesses", shape, err)
	}
}

func TestPathLinksFollowsStoredDirection(t *testing.T) {
	execute, a, _, c := setupTypedPathGraph(t)
	_, err := execute("path", c.ID, a.ID, "--links", "supports,grounded-by")
	if err == nil || !strings.Contains(err.Error(), "no path found") {
		t.Fatalf("reverse typed path error = %v", err)
	}
}

func TestPathLinksValidatesFilter(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, tc := range []struct {
		value string
		want  string
	}{
		{value: "", want: "--links requires at least one link type"},
		{value: "supports,invented", want: "unknown link type"},
	} {
		_, err := execute("path", "a", "b", "--links", tc.value)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("--links %q error = %v, want %q", tc.value, err, tc.want)
		}
	}
}

func TestPathLinksTextRendersEveryTypedWitness(t *testing.T) {
	execute, fixture := setupDiverseTypedWitnessGraph(t)
	out, err := execute("path", fixture.focus.ID, fixture.destination.ID, "--links", "extends,requires,supports")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "Witness "); got != 3 {
		t.Fatalf("typed text witness headings = %d, want 3:\n%s", got, out)
	}
	for _, n := range fixture.middles[:3] {
		if !strings.Contains(out, n.ID) {
			t.Errorf("typed text path missing selected first hop %s: %s", n.ID, out)
		}
	}
	if strings.Contains(out, fixture.middles[3].ID) {
		t.Errorf("typed text path exceeded cap with fourth first hop %s: %s", fixture.middles[3].ID, out)
	}
	for _, edgeType := range []string{"[extends]", "[requires]", "[supports]"} {
		if !strings.Contains(out, edgeType) {
			t.Errorf("typed text path missing %s edge labels: %s", edgeType, out)
		}
	}
}

func TestPathLinksReturnsThreeDiverseShortestWitnesses(t *testing.T) {
	execute, fixture := setupDiverseTypedWitnessGraph(t)
	out, err := execute("path", fixture.focus.ID, fixture.destination.ID, "--links", "supports,requires,extends", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got typedPathResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("typed path JSON: %v\n%s", err, out)
	}
	if len(got.Witnesses) != 3 {
		t.Fatalf("witnesses = %d, want 3: %s", len(got.Witnesses), out)
	}
	for i, witness := range got.Witnesses {
		if len(witness.Nodes) != 3 || witness.Nodes[1].ID != fixture.middles[i].ID {
			t.Errorf("witness %d nodes = %#v, want middle %s", i, witness.Nodes, fixture.middles[i].ID)
		}
	}
}

func TestPathLegacyTextAndJSONRemainByteExact(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI("20260101000000-0001", "Legacy A", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0002", "Legacy B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Type: "extends", Annotation: "legacy"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	text, err := execute("path", a.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantText := a.ID + "  " + a.Title + "\n  →\n" + b.ID + "  " + b.Title + "\n"
	if text != wantText {
		t.Fatalf("legacy text changed:\ngot  %q\nwant %q", text, wantText)
	}

	jsonOut, err := execute("path", a.ID, b.ID, "--json")
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := "[\n" +
		"  {\n" +
		"    \"id\": \"" + a.ID + "\",\n" +
		"    \"title\": \"" + a.Title + "\"\n" +
		"  },\n" +
		"  {\n" +
		"    \"id\": \"" + b.ID + "\",\n" +
		"    \"title\": \"" + b.Title + "\"\n" +
		"  }\n" +
		"]\n"
	if jsonOut != wantJSON {
		t.Fatalf("legacy JSON changed:\ngot:\n%s\nwant:\n%s", jsonOut, wantJSON)
	}
}

func TestPathLinksIsIntegratedWithNavigation(t *testing.T) {
	_, execute := setupNotebook(t)
	help, err := execute("path", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help, "--links") {
		t.Fatalf("path help missing --links:\n%s", help)
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"nn path <a> <b> --links <types> --json",
		"Orient", "Teleport", "Scan", "Peek", "Recenter", "Arrive",
		"witnesses", "at most 3", "first-hop", "type-sequence", "nodes[1]", "Datalog closure",
	} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing typed-path navigation guidance %q", required)
		}
	}
	show, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"{witnesses:[{nodes,edges}]}", "at most 3", "first-hop", "type-sequence", "legacy text and JSON unchanged"} {
		if !strings.Contains(string(show), required) {
			t.Errorf("embedded CLI reference missing typed-path witness guidance %q", required)
		}
	}
}
