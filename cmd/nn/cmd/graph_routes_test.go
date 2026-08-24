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
	Witnesses []typedWitnessTest `json:"witnesses"`
}

type graphRoutesExplainTestOutput struct {
	Routes      []graphRouteTestResult `json:"routes"`
	Diagnostics struct {
		QueryTokens          []string `json:"query_tokens"`
		TotalNotes           int      `json:"total_notes"`
		DirectLexicalMatches int      `json:"direct_lexical_matches"`
		FocusExcluded        int      `json:"focus_excluded"`
		TypedReachable       int      `json:"typed_reachable"`
		EligibleDestinations int      `json:"eligible_destinations"`
		GraphScoredMatches   int      `json:"graph_scored_matches"`
		Returned             int      `json:"returned"`
		TruncatedByLimit     bool     `json:"truncated_by_limit"`
	} `json:"diagnostics"`
}

func TestGraphRoutesCommandIsRegistered(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("graph", "routes", "--help")
	if err != nil {
		t.Fatalf("graph routes --help: %v", err)
	}
	for _, flag := range []string{"--focus", "--links", "--search", "--limit", "--json", "--explain"} {
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
	if len(got[1].Witnesses) != 2 {
		t.Fatalf("two-hop witnesses = %#v, want both equal-shortest type sequences", got[1].Witnesses)
	}
	witness := got[1].Witnesses[0]
	if len(witness.Nodes) != 3 || witness.Nodes[0].ID != focus.ID || witness.Nodes[1].ID != via.ID || witness.Nodes[2].ID != twoHop.ID {
		t.Fatalf("two-hop nodes = %#v", witness.Nodes)
	}
	if len(witness.Edges) != 2 || witness.Edges[0] != (typedWitnessTestEdge{From: focus.ID, To: via.ID, Type: "grounded-by", Annotation: "chosen predecessor"}) || witness.Edges[1].To != twoHop.ID {
		t.Fatalf("two-hop edges = %#v", witness.Edges)
	}
	if got[1].Witnesses[1].Edges[0].Type != "supports" {
		t.Fatalf("second equal-shortest type sequence = %#v, want supports first edge", got[1].Witnesses[1])
	}
	var rawRoutes []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &rawRoutes); err != nil {
		t.Fatal(err)
	}
	if len(rawRoutes[1]) != 2 || rawRoutes[1]["destination"] == nil || rawRoutes[1]["witnesses"] == nil {
		t.Fatalf("route entry shape = %v, want destination+witnesses only", rawRoutes[1])
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

func TestGraphRoutesLegacyJSONRemainsTopLevelArray(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	destination := newTestNoteForCLI("20260101000000-0002", "Needle destination", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: destination.ID, Type: "supports"}}
	for _, n := range []*note.Note{focus, destination} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "needle", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if trimmed := strings.TrimSpace(out); !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		t.Fatalf("legacy graph routes JSON is not a top-level array: %s", out)
	}
	var got []graphRouteTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got) != 1 {
		t.Fatalf("legacy graph routes JSON = %q, err=%v", out, err)
	}
}

func TestGraphRoutesExplainHyphenatedQueryUsesNormalizedTokensAndHonestCounts(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "No origin", note.TypeConcept)
	reachable := newTestNoteForCLI("20260101000000-0002", "Such destination", note.TypeConcept)
	unreachable := newTestNoteForCLI("20260101000000-0003", "Destination elsewhere", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: reachable.ID, Type: "supports"}}
	for _, n := range []*note.Note{focus, reachable, unreachable} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "zzzz-no-such-destination-zzzz", "--json", "--explain")
	if err != nil {
		t.Fatal(err)
	}
	var got graphRoutesExplainTestOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("explained graph routes JSON: %v\n%s", err, out)
	}
	wantTokens := []string{"zzzz", "no", "such", "destin", "zzzz"}
	if !reflect.DeepEqual(got.Diagnostics.QueryTokens, wantTokens) {
		t.Fatalf("query tokens = %v, want %v", got.Diagnostics.QueryTokens, wantTokens)
	}
	if got.Diagnostics.TotalNotes != 3 || got.Diagnostics.DirectLexicalMatches != 3 || got.Diagnostics.FocusExcluded != 1 || got.Diagnostics.TypedReachable != 1 || got.Diagnostics.EligibleDestinations != 1 || got.Diagnostics.GraphScoredMatches != 3 || got.Diagnostics.Returned != 1 || got.Diagnostics.TruncatedByLimit {
		t.Fatalf("dishonest hyphenated-query diagnostics: %+v", got.Diagnostics)
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &shape); err != nil || len(shape) != 2 || shape["routes"] == nil || shape["diagnostics"] == nil {
		t.Fatalf("explained output shape = %v, err=%v", shape, err)
	}
	var diagnosticShape map[string]json.RawMessage
	if err := json.Unmarshal(shape["diagnostics"], &diagnosticShape); err != nil || len(diagnosticShape) != 9 {
		t.Fatalf("diagnostic shape = %v, err=%v", diagnosticShape, err)
	}
	for _, key := range []string{"query_tokens", "total_notes", "direct_lexical_matches", "focus_excluded", "typed_reachable", "eligible_destinations", "graph_scored_matches", "returned", "truncated_by_limit"} {
		if diagnosticShape[key] == nil {
			t.Errorf("diagnostics missing %q: %s", key, out)
		}
	}
	if len(got.Routes) != 1 || got.Routes[0].Destination.ID != reachable.ID {
		t.Fatalf("routes = %+v, want only reachable destination", got.Routes)
	}
}

func TestGraphRoutesExplainSingleNonceHasNoMatches(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	destination := newTestNoteForCLI("20260101000000-0002", "Ordinary destination", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: destination.ID, Type: "supports"}}
	for _, n := range []*note.Note{focus, destination} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "singlenoncezzzz", "--json", "--explain")
	if err != nil {
		t.Fatal(err)
	}
	var got graphRoutesExplainTestOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Diagnostics.DirectLexicalMatches != 0 || got.Diagnostics.EligibleDestinations != 0 || got.Diagnostics.GraphScoredMatches != 0 || got.Diagnostics.Returned != 0 || len(got.Routes) != 0 {
		t.Fatalf("nonce diagnostics/routes = %+v / %+v", got.Diagnostics, got.Routes)
	}
}

func TestGraphRoutesExplainSeparatesAnnotationScoresFromDirectEligibility(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	destination := newTestNoteForCLI("20260101000000-0002", "Ordinary destination", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: destination.ID, Type: "supports", Annotation: "annotationonlynonce"}}
	for _, n := range []*note.Note{focus, destination} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "annotationonlynonce", "--json", "--explain")
	if err != nil {
		t.Fatal(err)
	}
	var got graphRoutesExplainTestOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Diagnostics.GraphScoredMatches == 0 || got.Diagnostics.DirectLexicalMatches != 0 || got.Diagnostics.EligibleDestinations != 0 || got.Diagnostics.Returned != 0 || len(got.Routes) != 0 {
		t.Fatalf("annotation-only diagnostics/routes = %+v / %+v", got.Diagnostics, got.Routes)
	}
}

func TestGraphRoutesExplainReportsReachabilityElimination(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	reachableNonmatch := newTestNoteForCLI("20260101000000-0002", "Ordinary waypoint", note.TypeConcept)
	unreachableMatch := newTestNoteForCLI("20260101000000-0003", "Quasarbloom destination", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: reachableNonmatch.ID, Type: "supports"}}
	for _, n := range []*note.Note{focus, reachableNonmatch, unreachableMatch} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "quasarbloom", "--json", "--explain")
	if err != nil {
		t.Fatal(err)
	}
	var got graphRoutesExplainTestOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Diagnostics.DirectLexicalMatches != 1 || got.Diagnostics.FocusExcluded != 0 || got.Diagnostics.TypedReachable != 1 || got.Diagnostics.EligibleDestinations != 0 || got.Diagnostics.Returned != 0 || len(got.Routes) != 0 {
		t.Fatalf("reachability diagnostics/routes = %+v / %+v", got.Diagnostics, got.Routes)
	}
}

func TestGraphRoutesExplainReportsLimitTruncation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	a := newTestNoteForCLI("20260101000000-0002", "Needle alpha", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0003", "Needle beta", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: a.ID, Type: "supports"}, {TargetID: b.ID, Type: "supports"}}
	for _, n := range []*note.Note{focus, a, b} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "needle", "--limit", "1", "--json", "--explain")
	if err != nil {
		t.Fatal(err)
	}
	var got graphRoutesExplainTestOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Diagnostics.EligibleDestinations != 2 || got.Diagnostics.Returned != 1 || !got.Diagnostics.TruncatedByLimit || len(got.Routes) != 1 {
		t.Fatalf("limit diagnostics/routes = %+v / %+v", got.Diagnostics, got.Routes)
	}
}

func TestGraphRoutesNonsenseQueryAgainstReachableNotesReturnsEmpty(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	via := newTestNoteForCLI("20260101000000-0002", "Ordinary waypoint", note.TypeConcept)
	destination := newTestNoteForCLI("20260101000000-0003", "Ordinary destination", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: via.ID, Type: "supports", Annotation: "ordinary route"}}
	via.Links = []note.Link{{TargetID: destination.ID, Type: "supports", Annotation: "nonsensequasar"}}
	for _, n := range []*note.Note{focus, via, destination} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "nonsensequasar", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("nonsense query routes = %q, want [] without direct note evidence", out)
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
		{name: "explain without json", args: []string{"graph", "routes", "--focus", focus.ID, "--links", "supports", "--search", "needle", "--explain"}, want: "--explain requires --json"},
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

func TestGraphRoutesScoringAndOutputIgnoreBackendAndLinkOrder(t *testing.T) {
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	a := newTestNoteForCLI("20260101000000-0002", "Needle", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0003", "Needle", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: b.ID, Type: "supports"}, {TargetID: a.ID, Type: "supports"}}
	reversedFocus := *focus
	reversedFocus.Links = []note.Link{{TargetID: a.ID, Type: "supports"}, {TargetID: b.ID, Type: "supports"}}
	forward := []*note.Note{focus, a, b}
	reverse := []*note.Note{&reversedFocus, b, a}
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

func TestGraphRoutesAndOutgoingImpactShareExactWitnesses(t *testing.T) {
	execute, fixture := setupDiverseTypedWitnessGraph(t)
	routesOut, err := execute("graph", "routes", "--focus", fixture.focus.ID, "--links", "extends,requires,supports", "--search", "needle", "--limit", "10", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var routes []graphRouteTestResult
	if err := json.Unmarshal([]byte(routesOut), &routes); err != nil {
		t.Fatalf("routes JSON: %v\n%s", err, routesOut)
	}
	if len(routes) != 1 || routes[0].Destination.ID != fixture.destination.ID {
		t.Fatalf("routes = %#v, want diverse fixture destination", routes)
	}

	impactOut, err := execute("graph", "impact", "--focus", fixture.focus.ID, "--links", "extends,requires,supports", "--direction", "outgoing", "--depth", "2", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var impact graphImpactTestResult
	if err := json.Unmarshal([]byte(impactOut), &impact); err != nil {
		t.Fatalf("impact JSON: %v\n%s", err, impactOut)
	}
	var impactWitnesses []typedWitnessTest
	for _, entry := range impact.Impacts {
		if entry.Node.ID == fixture.destination.ID {
			impactWitnesses = entry.Witnesses
			break
		}
	}
	if !reflect.DeepEqual(routes[0].Witnesses, impactWitnesses) {
		t.Fatalf("shared traversal produced different witnesses:\nroutes=%#v\nimpact=%#v", routes[0].Witnesses, impactWitnesses)
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
	navigate, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"nn graph routes --focus ID --links TYPES --search QUERY --limit N --json",
		"--explain", "direct_lexical_matches", "graph_scored_matches", "truncated_by_limit",
		"witnesses", "at most 3", "first-hop", "type-sequence",
		"Orient", "Scan", "Peek", "Recenter", "Arrive", "relevance_score", "nodes[1]",
	} {
		if !strings.Contains(string(navigate), required) {
			t.Errorf("nn-navigate missing typed destination discovery guidance %q", required)
		}
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), "positive direct lexical BM25 evidence") {
		t.Error("nn-guide command reference missing direct lexical route eligibility semantics")
	}
	show, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"nn graph routes --focus ID --links TYPES --search QUERY --limit N --json [--explain]",
		"direct_lexical_matches", "destination` plus `witnesses", "at most 3", "first-hop", "type-sequence", "ranking is unchanged",
	} {
		if !strings.Contains(string(show), required) {
			t.Errorf("embedded CLI reference missing graph routes guidance %q", required)
		}
	}
	adr, err := os.ReadFile("../../../docs/adr/0032-typed-destination-discovery.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"--explain", "{routes:", "query_tokens", "typed_reachable", "eligible_destinations", "graph_scored_matches", "truncated_by_limit"} {
		if !strings.Contains(string(adr), required) {
			t.Errorf("ADR 0032 missing explained routes decision %q", required)
		}
	}
}
