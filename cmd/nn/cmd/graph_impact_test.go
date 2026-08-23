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
	Node  graphImpactTestNode   `json:"node"`
	Depth int                   `json:"depth"`
	Nodes []graphImpactTestNode `json:"nodes"`
	Edges []pathWitnessEdge     `json:"edges"`
}

type graphImpactTestResult struct {
	Focus     graphImpactTestNode    `json:"focus"`
	Direction string                 `json:"direction"`
	Links     []string               `json:"links"`
	Depth     int                    `json:"depth"`
	Impacts   []graphImpactTestEntry `json:"impacts"`
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
	if got.Impacts[0].Edges[0] != (pathWitnessEdge{From: focus.ID, To: viaA.ID, Type: "grounded-by", Annotation: "a annotation"}) {
		t.Fatalf("deterministic first edge = %#v", got.Impacts[0].Edges[0])
	}
	wantNodes := []graphImpactTestNode{
		{ID: focus.ID, Title: focus.Title},
		{ID: viaA.ID, Title: viaA.Title},
		{ID: impact.ID, Title: impact.Title},
	}
	if !reflect.DeepEqual(got.Impacts[2].Nodes, wantNodes) {
		t.Fatalf("shortest witness nodes = %#v, want %#v", got.Impacts[2].Nodes, wantNodes)
	}
	wantEdges := []pathWitnessEdge{
		{From: focus.ID, To: viaA.ID, Type: "grounded-by", Annotation: "a annotation"},
		{From: viaA.ID, To: impact.ID, Type: "supports", Annotation: "shortest predecessor A"},
	}
	if !reflect.DeepEqual(got.Impacts[2].Edges, wantEdges) {
		t.Fatalf("shortest witness edges = %#v, want %#v", got.Impacts[2].Edges, wantEdges)
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
	wantNodes := []graphImpactTestNode{
		{ID: evidence.ID, Title: evidence.Title},
		{ID: claimA.ID, Title: claimA.Title},
		{ID: higher.ID, Title: higher.Title},
	}
	wantEdges := []pathWitnessEdge{
		{From: claimA.ID, To: evidence.ID, Type: "grounded-by", Annotation: "A depends on observation"},
		{From: higher.ID, To: claimA.ID, Type: "grounded-by", Annotation: "higher depends on A"},
	}
	if !reflect.DeepEqual(deep.Nodes, wantNodes) || !reflect.DeepEqual(deep.Edges, wantEdges) {
		t.Fatalf("incoming witness = nodes %#v edges %#v; want nodes %#v edges %#v", deep.Nodes, deep.Edges, wantNodes, wantEdges)
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
	if len(got.Impacts) != 2 || got.Impacts[1].Node.ID != conclusion.ID || got.Impacts[1].Edges[0].From != evidence.ID || got.Impacts[1].Edges[1].To != conclusion.ID {
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
}

func TestGraphImpactIsDocumentedForNavigation(t *testing.T) {
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	guideText := string(guide)
	for _, required := range []string{
		"nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json",
		"Scan", "Peek", "Recenter", "Arrive",
		"grounded-by", "incoming", "supports", "outgoing",
		"stored source", "target", "opposite",
	} {
		if !strings.Contains(guideText, required) {
			t.Errorf("nn-guide missing impact guidance %q", required)
		}
	}
	show, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(show), "nn graph impact --focus ID --links TYPES --direction incoming|outgoing --depth N --json") {
		t.Error("embedded CLI reference missing graph impact command")
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
