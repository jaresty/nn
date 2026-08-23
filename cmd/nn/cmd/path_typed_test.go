package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type typedPathResult struct {
	Nodes []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"nodes"`
	Edges []struct {
		From       string `json:"from"`
		To         string `json:"to"`
		Type       string `json:"type"`
		Annotation string `json:"annotation"`
	} `json:"edges"`
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
	if len(got.Nodes) != 3 || got.Nodes[0].ID != a.ID || got.Nodes[1].ID != b.ID || got.Nodes[2].ID != c.ID {
		t.Fatalf("typed nodes = %#v, want A→B→C", got.Nodes)
	}
	if len(got.Edges) != 2 {
		t.Fatalf("typed edges = %#v, want 2", got.Edges)
	}
	if got.Edges[0].From != a.ID || got.Edges[0].To != b.ID || got.Edges[0].Type != "supports" || got.Edges[0].Annotation != "corroborated by bridge" {
		t.Fatalf("first typed edge = %#v", got.Edges[0])
	}
	if got.Edges[1].From != b.ID || got.Edges[1].To != c.ID || got.Edges[1].Type != "grounded-by" {
		t.Fatalf("second typed edge = %#v", got.Edges[1])
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

func TestPathLinksTextPreservesNodeRouteShape(t *testing.T) {
	execute, a, b, c := setupTypedPathGraph(t)
	out, err := execute("path", a.ID, c.ID, "--links", "supports,grounded-by")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a.ID, b.ID, c.ID} {
		if !strings.Contains(out, id) {
			t.Errorf("typed text path missing %s: %s", id, out)
		}
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
		"nodes[1]", "Datalog closure",
	} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing typed-path navigation guidance %q", required)
		}
	}
}
