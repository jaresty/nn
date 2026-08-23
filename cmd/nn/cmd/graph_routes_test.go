package cmd

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

type graphRouteTestResult struct {
	Destination struct {
		ID             string  `json:"id"`
		Title          string  `json:"title"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"destination"`
	Nodes []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"nodes"`
	Edges []pathWitnessEdge `json:"edges"`
}

func TestGraphRoutesCommandIsRegistered(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("graph", "routes", "--help")
	if err != nil {
		t.Fatalf("graph routes --help: %v", err)
	}
	for _, flag := range []string{"--focus", "--links", "--search", "--limit", "--json"} {
		if !strings.Contains(out, flag) {
			t.Errorf("graph routes help missing %s:\n%s", flag, out)
		}
	}
}

func TestGraphRoutesReturnsRankedDirectedTypedWitnesses(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	via := newTestNoteForCLI("20260101000000-0002", "Transit", note.TypeConcept)
	twoHop := newTestNoteForCLI("20260101000000-0003", "Needle destination", note.TypeConcept)
	direct := newTestNoteForCLI("20260101000000-0004", "Needle destination", note.TypeConcept)
	unreachable := newTestNoteForCLI("20260101000000-0005", "Needle needle unreachable", note.TypeConcept)
	unreachable.Body = "needle needle"
	disallowed := newTestNoteForCLI("20260101000000-0006", "Needle disallowed", note.TypeConcept)
	nonmatch := newTestNoteForCLI("20260101000000-0007", "Unrelated reachable", note.TypeConcept)

	// Stored order is intentionally different from the required target/type/
	// annotation order. The grounded-by edge must become via's predecessor.
	focus.Links = []note.Link{
		{TargetID: disallowed.ID, Type: "extends", Annotation: "wrong relationship"},
		{TargetID: direct.ID, Type: "supports", Annotation: "direct route"},
		{TargetID: via.ID, Type: "supports", Annotation: "later type"},
		{TargetID: via.ID, Type: "grounded-by", Annotation: "chosen predecessor"},
		{TargetID: nonmatch.ID, Type: "supports", Annotation: "reachable without a match"},
	}
	via.Links = []note.Link{{TargetID: twoHop.ID, Type: "supports", Annotation: "second hop"}}
	for _, n := range []*note.Note{focus, via, twoHop, direct, unreachable, disallowed, nonmatch} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports,grounded-by", "--search", "needle", "--limit", "10", "--json")
	if err != nil {
		t.Fatalf("graph routes: %v", err)
	}
	var got []graphRouteTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("graph routes JSON: %v\n%s", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("routes = %d, want reachable positive destinations only: %s", len(got), out)
	}
	ids := map[string]bool{}
	for _, route := range got {
		ids[route.Destination.ID] = true
	}
	if !ids[direct.ID] || !ids[twoHop.ID] || ids[focus.ID] || ids[unreachable.ID] || ids[disallowed.ID] || ids[nonmatch.ID] {
		t.Fatalf("route destinations = %#v", ids)
	}
	if got[0].Destination.ID != direct.ID || got[1].Destination.ID != twoHop.ID {
		t.Fatalf("routes not relevance-first: %#v", got)
	}
	if got[0].Destination.RelevanceScore <= 0 || got[0].Destination.RelevanceScore >= 1 {
		t.Fatalf("reachable relevance = %v, want normalized against stronger unreachable full-corpus hit", got[0].Destination.RelevanceScore)
	}
	if len(got[1].Nodes) != 3 || got[1].Nodes[0].ID != focus.ID || got[1].Nodes[1].ID != via.ID || got[1].Nodes[2].ID != twoHop.ID {
		t.Fatalf("two-hop nodes = %#v", got[1].Nodes)
	}
	if len(got[1].Edges) != 2 || got[1].Edges[0] != (pathWitnessEdge{From: focus.ID, To: via.ID, Type: "grounded-by", Annotation: "chosen predecessor"}) || got[1].Edges[1].To != twoHop.ID {
		t.Fatalf("two-hop edges = %#v", got[1].Edges)
	}
}

func TestGraphRoutesLimitAndEmptyArray(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	a := newTestNoteForCLI("20260101000000-0002", "Needle alpha", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0003", "Needle beta", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: a.ID, Type: "supports"}, {TargetID: b.ID, Type: "supports"}}
	for _, n := range []*note.Note{focus, a, b} {
		writeNoteFile(t, nbDir, n)
	}
	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "needle", "--limit", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var limited []graphRouteTestResult
	if err := json.Unmarshal([]byte(out), &limited); err != nil || len(limited) != 1 {
		t.Fatalf("limited routes = %q, err=%v", out, err)
	}
	out, err = execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "absentterm", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty routes = %q, want []", out)
	}
}

func TestGraphRoutesValidatesRequiredInputs(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	writeNoteFile(t, nbDir, focus)
	base := []string{"graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "needle", "--limit", "5", "--json"}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing focus", args: []string{"graph", "routes", "--links", "supports", "--search", "needle", "--json"}, want: "--focus requires a non-blank ID"},
		{name: "blank focus", args: []string{"graph", "routes", "--focus", " \t ", "--links", "supports", "--search", "needle", "--json"}, want: "--focus requires a non-blank ID"},
		{name: "unknown focus", args: []string{"graph", "routes", "--focus", "missing", "--links", "supports", "--search", "needle", "--json"}, want: "note \"missing\" not found"},
		{name: "blank links", args: []string{"graph", "routes", "--focus", focus.ID, "--links", " ", "--search", "needle", "--json"}, want: "--links requires at least one link type"},
		{name: "empty link member", args: []string{"graph", "routes", "--focus", focus.ID, "--links", "supports,,extends", "--search", "needle", "--json"}, want: "--links contains an empty value"},
		{name: "unknown link", args: []string{"graph", "routes", "--focus", focus.ID, "--links", "supports,invented", "--search", "needle", "--json"}, want: "unknown link type"},
		{name: "blank search", args: []string{"graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", " \t ", "--json"}, want: "--search requires a non-blank query"},
		{name: "json absent", args: base[:len(base)-1], want: "requires --json"},
		{name: "json false", args: append(append([]string{}, base[:len(base)-1]...), "--json=false"), want: "requires --json"},
		{name: "zero limit", args: []string{"graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "needle", "--limit", "0", "--json"}, want: "--limit must be greater than zero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

type orderedRoutesBackend struct {
	backend.Backend
	notes []*note.Note
}

func (b *orderedRoutesBackend) List() ([]*note.Note, error) { return b.notes, nil }

func executeRoutesWithNotes(t *testing.T, notes []*note.Note) string {
	t.Helper()
	state := &rootState{backend: &orderedRoutesBackend{notes: notes}}
	cmd := newGraphRoutesCmd(state)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--focus", notes[0].ID, "--links", "supports", "--search", "needle", "--limit", "10", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("graph routes: %v", err)
	}
	return stdout.String()
}

func TestGraphRoutesScoringAndOutputIgnoreBackendOrder(t *testing.T) {
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	a := newTestNoteForCLI("20260101000000-0002", "Needle", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0003", "Needle", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: b.ID, Type: "supports"}, {TargetID: a.ID, Type: "supports"}}
	forward := []*note.Note{focus, a, b}
	reverse := []*note.Note{focus, b, a}
	var gotForward, gotReverse []graphRouteTestResult
	if err := json.Unmarshal([]byte(executeRoutesWithNotes(t, forward)), &gotForward); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(executeRoutesWithNotes(t, reverse)), &gotReverse); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotForward, gotReverse) {
		t.Fatalf("routes depend on backend order:\nforward=%#v\nreverse=%#v", gotForward, gotReverse)
	}
}

func TestGraphRouteCandidateTieBreaksByHopsThenID(t *testing.T) {
	candidates := []graphRouteCandidate{
		{id: "b", relevance: 0.5, hops: 2},
		{id: "c", relevance: 0.5, hops: 1},
		{id: "a", relevance: 0.5, hops: 1},
		{id: "z", relevance: math.Nextafter(0.5, 1), hops: 9},
	}
	sortGraphRouteCandidates(candidates)
	got := []string{candidates[0].id, candidates[1].id, candidates[2].id, candidates[3].id}
	want := []string{"z", "a", "c", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate order = %v, want %v", got, want)
	}
}

func TestGraphRoutesIsDocumentedForNavigation(t *testing.T) {
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"nn graph routes --focus ID --links TYPES --search QUERY --limit N --json",
		"Orient", "Scan", "Peek", "Recenter", "Arrive", "relevance_score", "nodes[1]",
	} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing typed destination discovery guidance %q", required)
		}
	}
	show, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(show), "nn graph routes --focus ID --links TYPES --search QUERY --limit N --json") {
		t.Error("embedded CLI reference missing graph routes command")
	}
}
