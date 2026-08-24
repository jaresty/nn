package cmd

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type graphImpactTestNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type graphImpactTestEntry struct {
	Node      graphImpactTestNode `json:"node"`
	Depth     int                 `json:"depth"`
	Witnesses []typedWitnessTest  `json:"witnesses"`
}

type graphImpactTestDepthCount struct {
	Depth int `json:"depth"`
	Count int `json:"count"`
}

type graphImpactTestFirstHopCount struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count int    `json:"count"`
}

type graphImpactTestSummary struct {
	TotalImpacts       int                            `json:"total_impacts"`
	CountsByDepth      []graphImpactTestDepthCount    `json:"counts_by_depth"`
	CountsByFirstHop   []graphImpactTestFirstHopCount `json:"counts_by_first_hop"`
	WitnessesTruncated bool                           `json:"witnesses_truncated"`
}

type graphImpactTestResult struct {
	Focus     graphImpactTestNode    `json:"focus"`
	Direction string                 `json:"direction"`
	Links     []string               `json:"links"`
	Depth     int                    `json:"depth"`
	Summary   graphImpactTestSummary `json:"summary"`
	Impacts   []graphImpactTestEntry `json:"impacts"`
}

type graphImpactOutputShape struct {
	Impacts []map[string]json.RawMessage `json:"impacts"`
}

func TestGraphImpactCommandIsRegistered(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("graph", "impact", "--help")
	if err != nil {
		t.Fatalf("graph impact --help: %v", err)
	}
	for _, flag := range []string{"--focus", "--links", "--direction", "--depth", "--json"} {
		if !strings.Contains(out, flag) {
			t.Errorf("graph impact help missing %s:\n%s", flag, out)
		}
	}
}

func TestGraphImpactOutgoingIsCycleSafeDeterministicAndDepthBounded(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Focus", note.TypeConcept)
	viaA := newTestNoteForCLI("20260101000000-0002", "Claim A", note.TypeConcept)
	viaB := newTestNoteForCLI("20260101000000-0003", "Claim B", note.TypeConcept)
	impact := newTestNoteForCLI("20260101000000-0004", "Conclusion", note.TypeArgument)
	tooDeep := newTestNoteForCLI("20260101000000-0005", "Beyond depth", note.TypeConcept)
	disallowed := newTestNoteForCLI("20260101000000-0006", "Wrong relationship", note.TypeConcept)

	// Stored order intentionally conflicts with endpoint/type/annotation order.
	focus.Links = []note.Link{
		{TargetID: viaB.ID, Type: "supports", Annotation: "later endpoint"},
		{TargetID: viaA.ID, Type: "supports", Annotation: "later type"},
		{TargetID: viaA.ID, Type: "grounded-by", Annotation: "z annotation"},
		{TargetID: viaA.ID, Type: "grounded-by", Annotation: "a annotation"},
		{TargetID: disallowed.ID, Type: "extends", Annotation: "filtered out"},
	}
	viaA.Links = []note.Link{{TargetID: impact.ID, Type: "supports", Annotation: "shortest predecessor A"}}
	viaB.Links = []note.Link{{TargetID: impact.ID, Type: "supports", Annotation: "equal route B"}}
	impact.Links = []note.Link{
		{TargetID: focus.ID, Type: "supports", Annotation: "cycle"},
		{TargetID: tooDeep.ID, Type: "supports", Annotation: "third hop"},
	}
	for _, n := range []*note.Note{focus, viaA, viaB, impact, tooDeep, disallowed} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "impact", "--focus", focus.ID, "--links", " supports,grounded-by,supports ", "--direction", "outgoing", "--depth", "2", "--json")
	if err != nil {
		t.Fatalf("graph impact outgoing: %v", err)
	}
	var got graphImpactTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("graph impact JSON: %v\n%s", err, out)
	}
	if got.Focus != (graphImpactTestNode{ID: focus.ID, Title: focus.Title}) || got.Direction != "outgoing" || got.Depth != 2 {
		t.Fatalf("metadata = %#v", got)
	}
	if want := []string{"grounded-by", "supports"}; !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("normalized links = %v, want %v", got.Links, want)
	}
	if len(got.Impacts) != 3 {
		t.Fatalf("impacts = %d, want three depth-bounded non-focus nodes: %s", len(got.Impacts), out)
	}
	if ids := []string{got.Impacts[0].Node.ID, got.Impacts[1].Node.ID, got.Impacts[2].Node.ID}; !reflect.DeepEqual(ids, []string{viaA.ID, viaB.ID, impact.ID}) {
		t.Fatalf("impact order = %v", ids)
	}
	if depths := []int{got.Impacts[0].Depth, got.Impacts[1].Depth, got.Impacts[2].Depth}; !reflect.DeepEqual(depths, []int{1, 1, 2}) {
		t.Fatalf("impact depths = %v", depths)
	}
	wantSummary := graphImpactTestSummary{
		TotalImpacts:       3,
		CountsByDepth:      []graphImpactTestDepthCount{{Depth: 1, Count: 2}, {Depth: 2, Count: 1}},
		CountsByFirstHop:   []graphImpactTestFirstHopCount{{ID: viaA.ID, Title: viaA.Title, Count: 2}, {ID: viaB.ID, Title: viaB.Title, Count: 2}},
		WitnessesTruncated: true,
	}
	if !reflect.DeepEqual(got.Summary, wantSummary) {
		t.Fatalf("summary = %#v, want %#v", got.Summary, wantSummary)
	}
	if len(got.Impacts[0].Witnesses) != 3 || got.Impacts[0].Witnesses[0].Edges[0] != (typedWitnessTestEdge{From: focus.ID, To: viaA.ID, Type: "grounded-by", Annotation: "a annotation"}) {
		t.Fatalf("deterministic direct witnesses = %#v", got.Impacts[0].Witnesses)
	}
	if len(got.Impacts[2].Witnesses) != 3 {
		t.Fatalf("deep witnesses = %#v, want bounded diverse cap", got.Impacts[2].Witnesses)
	}
	witness := got.Impacts[2].Witnesses[0]
	wantNodes := []typedWitnessTestNode{
		{ID: focus.ID, Title: focus.Title},
		{ID: viaA.ID, Title: viaA.Title},
		{ID: impact.ID, Title: impact.Title},
	}
	if !reflect.DeepEqual(witness.Nodes, wantNodes) {
		t.Fatalf("shortest witness nodes = %#v, want %#v", witness.Nodes, wantNodes)
	}
	wantEdges := []typedWitnessTestEdge{
		{From: focus.ID, To: viaA.ID, Type: "grounded-by", Annotation: "a annotation"},
		{From: viaA.ID, To: impact.ID, Type: "supports", Annotation: "shortest predecessor A"},
	}
	if !reflect.DeepEqual(witness.Edges, wantEdges) {
		t.Fatalf("shortest witness edges = %#v, want %#v", witness.Edges, wantEdges)
	}
	if got.Impacts[2].Witnesses[2].Nodes[1].ID != viaB.ID {
		t.Fatalf("first-hop diversity lost equal route B: %#v", got.Impacts[2].Witnesses)
	}
	var raw graphImpactOutputShape
	if err := json.Unmarshal([]byte(out), &raw); err != nil || len(raw.Impacts) != 3 {
		t.Fatalf("raw impact shape: %v / %v", raw, err)
	}
	if len(raw.Impacts[2]) != 3 || raw.Impacts[2]["node"] == nil || raw.Impacts[2]["depth"] == nil || raw.Impacts[2]["witnesses"] == nil {
		t.Fatalf("impact entry shape = %v, want node+depth+witnesses only", raw.Impacts[2])
	}
}

func TestGraphImpactSummaryCountsOverlappingFirstHopsOncePerImpactAndSorts(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Focus", note.TypeConcept)
	a := newTestNoteForCLI("20260101000000-0002", "Branch A", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0003", "Branch B", note.TypeConcept)
	c := newTestNoteForCLI("20260101000000-0004", "Branch C", note.TypeConcept)
	x := newTestNoteForCLI("20260101000000-0005", "Only A", note.TypeConcept)
	y := newTestNoteForCLI("20260101000000-0006", "A and B", note.TypeConcept)
	z := newTestNoteForCLI("20260101000000-0007", "B and C", note.TypeConcept)

	focus.Links = []note.Link{
		{TargetID: a.ID, Type: "supports", Annotation: "A duplicate 3"},
		{TargetID: a.ID, Type: "supports", Annotation: "A duplicate 2"},
		{TargetID: a.ID, Type: "supports", Annotation: "A duplicate 1"},
		{TargetID: b.ID, Type: "supports", Annotation: "B"},
		{TargetID: c.ID, Type: "supports", Annotation: "C"},
	}
	a.Links = []note.Link{
		{TargetID: x.ID, Type: "supports", Annotation: "A to X"},
		{TargetID: y.ID, Type: "supports", Annotation: "A to Y"},
	}
	b.Links = []note.Link{
		{TargetID: y.ID, Type: "supports", Annotation: "B to Y"},
		{TargetID: z.ID, Type: "supports", Annotation: "B to Z"},
	}
	c.Links = []note.Link{{TargetID: z.ID, Type: "supports", Annotation: "C to Z"}}
	for _, n := range []*note.Note{focus, a, b, c, x, y, z} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "outgoing", "--depth", "2", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got graphImpactTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	wantDepths := []graphImpactTestDepthCount{{Depth: 1, Count: 3}, {Depth: 2, Count: 3}}
	if !reflect.DeepEqual(got.Summary.CountsByDepth, wantDepths) {
		t.Fatalf("counts_by_depth = %#v, want %#v", got.Summary.CountsByDepth, wantDepths)
	}
	wantBranches := []graphImpactTestFirstHopCount{
		{ID: a.ID, Title: a.Title, Count: 3},
		{ID: b.ID, Title: b.Title, Count: 3},
		{ID: c.ID, Title: c.Title, Count: 2},
	}
	if got.Summary.TotalImpacts != 6 || !reflect.DeepEqual(got.Summary.CountsByFirstHop, wantBranches) {
		t.Fatalf("summary = %#v, want total 6 and branches %#v", got.Summary, wantBranches)
	}
	if branchTotal := got.Summary.CountsByFirstHop[0].Count + got.Summary.CountsByFirstHop[1].Count + got.Summary.CountsByFirstHop[2].Count; branchTotal != 8 {
		t.Fatalf("overlapping branch total = %d, want 8 > total impacts", branchTotal)
	}
}

func TestGraphImpactIncomingKeepsStoredWitnessOrientation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	evidence := newTestNoteForCLI("20260101000000-0001", "Observation", note.TypeObservation)
	claimA := newTestNoteForCLI("20260101000000-0002", "Grounded claim A", note.TypeConcept)
	claimB := newTestNoteForCLI("20260101000000-0003", "Grounded claim B", note.TypeConcept)
	higher := newTestNoteForCLI("20260101000000-0004", "Higher claim", note.TypeArgument)
	disallowed := newTestNoteForCLI("20260101000000-0005", "Supporting note", note.TypeObservation)

	// grounded-by is stored claim→evidence. Incoming traversal therefore walks
	// evidence→claim while every witness edge must remain claim→evidence.
	claimA.Links = []note.Link{{TargetID: evidence.ID, Type: "grounded-by", Annotation: "A depends on observation"}}
	claimB.Links = []note.Link{{TargetID: evidence.ID, Type: "grounded-by", Annotation: "B depends on observation"}}
	higher.Links = []note.Link{{TargetID: claimA.ID, Type: "grounded-by", Annotation: "higher depends on A"}}
	evidence.Links = []note.Link{{TargetID: higher.ID, Type: "grounded-by", Annotation: "closes cycle"}}
	disallowed.Links = []note.Link{{TargetID: evidence.ID, Type: "supports", Annotation: "wrong type"}}
	for _, n := range []*note.Note{evidence, claimA, claimB, higher, disallowed} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "impact", "--focus", evidence.ID, "--links", "grounded-by", "--direction", "incoming", "--depth", "2", "--json")
	if err != nil {
		t.Fatalf("graph impact incoming: %v", err)
	}
	var got graphImpactTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("graph impact incoming JSON: %v\n%s", err, out)
	}
	if len(got.Impacts) != 3 {
		t.Fatalf("incoming impacts = %d, want 3: %s", len(got.Impacts), out)
	}
	deep := got.Impacts[2]
	if deep.Node.ID != higher.ID || deep.Depth != 2 {
		t.Fatalf("deep incoming impact = %#v", deep)
	}
	if len(deep.Witnesses) != 1 {
		t.Fatalf("deep incoming witnesses = %#v, want one", deep.Witnesses)
	}
	witness := deep.Witnesses[0]
	wantNodes := []typedWitnessTestNode{
		{ID: evidence.ID, Title: evidence.Title},
		{ID: claimA.ID, Title: claimA.Title},
		{ID: higher.ID, Title: higher.Title},
	}
	wantEdges := []typedWitnessTestEdge{
		{From: claimA.ID, To: evidence.ID, Type: "grounded-by", Annotation: "A depends on observation"},
		{From: higher.ID, To: claimA.ID, Type: "grounded-by", Annotation: "higher depends on A"},
	}
	if !reflect.DeepEqual(witness.Nodes, wantNodes) || !reflect.DeepEqual(witness.Edges, wantEdges) {
		t.Fatalf("incoming witness = nodes %#v edges %#v; want nodes %#v edges %#v", witness.Nodes, witness.Edges, wantNodes, wantEdges)
	}
	for i, edge := range witness.Edges {
		if edge.From != witness.Nodes[i+1].ID || edge.To != witness.Nodes[i].ID {
			t.Errorf("incoming edge %d lost stored orientation: nodes=%#v edge=%#v", i, witness.Nodes, edge)
		}
	}
}

func TestGraphImpactSupportsOutgoingEvidenceExample(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	evidence := newTestNoteForCLI("20260101000000-0001", "Evidence", note.TypeObservation)
	claim := newTestNoteForCLI("20260101000000-0002", "Supported claim", note.TypeConcept)
	conclusion := newTestNoteForCLI("20260101000000-0003", "Supported conclusion", note.TypeArgument)
	evidence.Links = []note.Link{{TargetID: claim.ID, Type: "supports", Annotation: "corroborates claim"}}
	claim.Links = []note.Link{{TargetID: conclusion.ID, Type: "supports", Annotation: "corroborates conclusion"}}
	for _, n := range []*note.Note{evidence, claim, conclusion} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "impact", "--focus", evidence.ID, "--links", "supports", "--direction", "outgoing", "--depth", "2", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got graphImpactTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Impacts) != 2 || got.Impacts[1].Node.ID != conclusion.ID || len(got.Impacts[1].Witnesses) != 1 || got.Impacts[1].Witnesses[0].Edges[0].From != evidence.ID || got.Impacts[1].Witnesses[0].Edges[1].To != conclusion.ID {
		t.Fatalf("supports outgoing impacts = %s", out)
	}
}

func TestGraphImpactValidatesExplicitRequiredInputs(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Focus", note.TypeConcept)
	writeNoteFile(t, nbDir, focus)
	valid := []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "outgoing", "--depth", "2", "--json"}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing focus", args: []string{"graph", "impact", "--links", "supports", "--direction", "outgoing", "--depth", "2", "--json"}, want: "--focus"},
		{name: "blank focus", args: []string{"graph", "impact", "--focus", " \t ", "--links", "supports", "--direction", "outgoing", "--depth", "2", "--json"}, want: "--focus requires a non-blank ID"},
		{name: "unknown focus", args: []string{"graph", "impact", "--focus", "missing", "--links", "supports", "--direction", "outgoing", "--depth", "2", "--json"}, want: "note \"missing\" not found"},
		{name: "missing links", args: []string{"graph", "impact", "--focus", focus.ID, "--direction", "outgoing", "--depth", "2", "--json"}, want: "--links"},
		{name: "blank links", args: []string{"graph", "impact", "--focus", focus.ID, "--links", " ", "--direction", "outgoing", "--depth", "2", "--json"}, want: "--links requires at least one link type"},
		{name: "empty link member", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports,,extends", "--direction", "outgoing", "--depth", "2", "--json"}, want: "--links contains an empty value"},
		{name: "unknown link", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "invented", "--direction", "outgoing", "--depth", "2", "--json"}, want: "unknown link type"},
		{name: "missing direction", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--depth", "2", "--json"}, want: "--direction"},
		{name: "both direction", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "both", "--depth", "2", "--json"}, want: "--direction must be exactly incoming or outgoing"},
		{name: "padded direction", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", " incoming ", "--depth", "2", "--json"}, want: "--direction must be exactly incoming or outgoing"},
		{name: "missing depth", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "outgoing", "--json"}, want: "--depth"},
		{name: "zero depth", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "outgoing", "--depth", "0", "--json"}, want: "--depth must be greater than zero"},
		{name: "negative depth", args: []string{"graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "outgoing", "--depth", "-1", "--json"}, want: "--depth must be greater than zero"},
		{name: "missing json", args: valid[:len(valid)-1], want: "requires --json"},
		{name: "json false", args: append(append([]string{}, valid[:len(valid)-1]...), "--json=false"), want: "requires --json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestGraphImpactEmptyResultUsesJSONArray(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Focus", note.TypeConcept)
	writeNoteFile(t, nbDir, focus)
	out, err := execute("graph", "impact", "--focus", focus.ID, "--links", "supports", "--direction", "outgoing", "--depth", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got graphImpactTestResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Impacts == nil || len(got.Impacts) != 0 {
		t.Fatalf("impacts = %#v, want non-nil empty array; output=%s", got.Impacts, out)
	}
	wantSummary := graphImpactTestSummary{
		CountsByDepth:    []graphImpactTestDepthCount{},
		CountsByFirstHop: []graphImpactTestFirstHopCount{},
	}
	if !reflect.DeepEqual(got.Summary, wantSummary) {
		t.Fatalf("empty summary = %#v, want %#v", got.Summary, wantSummary)
	}
	depthAt := strings.Index(out, `"depth": 1`)
	summaryAt := strings.Index(out, `"summary":`)
	impactsAt := strings.Index(out, `"impacts":`)
	if depthAt < 0 || summaryAt < depthAt || impactsAt < summaryAt {
		t.Fatalf("top-level JSON field order must place summary between depth and impacts:\n%s", out)
	}
}

func TestGraphImpactIsDocumentedForNavigation(t *testing.T) {
	navigate, err := os.ReadFile("../../../skills/nn-navigate/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	navigateText := string(navigate)
	for _, required := range []string{
		"nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json",
		"Scan", "Peek", "Recenter", "Arrive", "summary", "total_impacts", "witnesses_truncated", "witnesses", "at most 3", "first-hop", "type-sequence",
		"grounded-by", "incoming", "supports", "outgoing",
		"stored source", "target", "opposite",
	} {
		if !strings.Contains(navigateText, required) {
			t.Errorf("nn-navigate missing impact guidance %q", required)
		}
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"counts_by_depth", "counts_by_first_hop"} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide command reference missing impact summary field %q", required)
		}
	}
	show, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json",
		"`summary`", "`total_impacts`", "`counts_by_depth`", "`counts_by_first_hop`", "`witnesses_truncated`", "`node`/`depth`", "`witnesses`", "at most 3", "nodes run focus→impact", "stored source→target orientation",
	} {
		if !strings.Contains(string(show), required) {
			t.Errorf("embedded CLI reference missing graph impact guidance %q", required)
		}
	}
	adr, err := os.ReadFile("../../../docs/adr/0033-explicit-impact-traversal.md")
	if err != nil {
		t.Fatalf("impact ADR: %v", err)
	}
	for _, required := range []string{"grounded-by", "incoming", "supports", "outgoing", "stored source", "opposite"} {
		if !strings.Contains(string(adr), required) {
			t.Errorf("impact ADR missing %q", required)
		}
	}
}
