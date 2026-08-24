package cmd

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type typedWitnessTestNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type typedWitnessTestEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Annotation string `json:"annotation"`
}

type typedWitnessTest struct {
	Nodes []typedWitnessTestNode `json:"nodes"`
	Edges []typedWitnessTestEdge `json:"edges"`
}

func testTypedTraversal(from, to, linkType, annotation string) typedTraversalEdge {
	return typedTraversalEdge{
		next: to,
		edge: typedWitnessEdge{From: from, To: to, Type: linkType, Annotation: annotation},
	}
}

func typedWitnessNodeIDs(w typedWitness) []string {
	ids := make([]string, len(w.Nodes))
	for i, node := range w.Nodes {
		ids[i] = node.ID
	}
	return ids
}

func typedWitnessTypes(w typedWitness) []string {
	types := make([]string, len(w.Edges))
	for i, edge := range w.Edges {
		types[i] = edge.Type
	}
	return types
}

func TestTypedWitnessesRetainEqualShortestPredecessorsAndPreferFirstHopDiversity(t *testing.T) {
	titles := map[string]string{"s": "Source", "a": "A", "b": "B", "c": "C", "d": "D", "t": "Target"}
	adj := map[string][]typedTraversalEdge{
		"s": {
			testTypedTraversal("s", "d", "requires", "fourth"),
			testTypedTraversal("s", "b", "extends", "second"),
			testTypedTraversal("s", "c", "requires", "third"),
			testTypedTraversal("s", "a", "extends", "first"),
		},
		"a": {testTypedTraversal("a", "t", "supports", "via a")},
		"b": {testTypedTraversal("b", "t", "supports", "via b")},
		"c": {testTypedTraversal("c", "t", "supports", "via c")},
		"d": {testTypedTraversal("d", "t", "supports", "via d")},
		"t": {testTypedTraversal("t", "s", "supports", "cycle")},
	}

	search := findShortestTypedWitnesses("s", titles, adj, 0)
	if got := len(search.predecessors["t"]); got != 4 {
		t.Fatalf("equal-shortest predecessors = %d, want 4", got)
	}
	got := search.witnessesTo("t")
	if len(got) != typedWitnessOutputLimit {
		t.Fatalf("witnesses = %d, want cap %d", len(got), typedWitnessOutputLimit)
	}
	wantPaths := [][]string{{"s", "a", "t"}, {"s", "b", "t"}, {"s", "c", "t"}}
	wantTypes := [][]string{{"extends", "supports"}, {"extends", "supports"}, {"requires", "supports"}}
	for i := range got {
		if path := typedWitnessNodeIDs(got[i]); !reflect.DeepEqual(path, wantPaths[i]) {
			t.Errorf("witness %d path = %v, want %v", i, path, wantPaths[i])
		}
		if types := typedWitnessTypes(got[i]); !reflect.DeepEqual(types, wantTypes[i]) {
			t.Errorf("witness %d types = %v, want %v", i, types, wantTypes[i])
		}
	}
}

func TestTypedWitnessesReportOutputCapTruncationWithoutChangingWitnesses(t *testing.T) {
	titles := map[string]string{"s": "Source", "a": "A", "b": "B", "c": "C", "d": "D", "t": "Target"}
	adj := map[string][]typedTraversalEdge{
		"s": {
			testTypedTraversal("s", "a", "supports", "a"),
			testTypedTraversal("s", "b", "supports", "b"),
			testTypedTraversal("s", "c", "supports", "c"),
			testTypedTraversal("s", "d", "supports", "d"),
		},
		"a": {testTypedTraversal("a", "t", "supports", "a")},
		"b": {testTypedTraversal("b", "t", "supports", "b")},
		"c": {testTypedTraversal("c", "t", "supports", "c")},
		"d": {testTypedTraversal("d", "t", "supports", "d")},
	}

	search := findShortestTypedWitnesses("s", titles, adj, 0)
	before := search.witnessesTo("t")
	got, truncated := search.witnessesToWithTruncation("t")
	if !reflect.DeepEqual(got, before) {
		t.Fatalf("truncation-aware witnesses changed output:\nbefore=%#v\nafter=%#v", before, got)
	}
	if !truncated {
		t.Fatal("four output candidates must report witnesses_truncated")
	}
}

func TestTypedWitnessesPropagateInternalPortfolioTruncation(t *testing.T) {
	titles := map[string]string{"s": "Source", "m": "Middle", "t": "Target"}
	adj := map[string][]typedTraversalEdge{"m": {testTypedTraversal("m", "t", "supports", "finish")}}
	for i := 0; i < typedWitnessPortfolioLimit+1; i++ {
		adj["s"] = append(adj["s"], testTypedTraversal("s", "m", "supports", "route-"+strconv.Itoa(i)))
	}

	search := findShortestTypedWitnesses("s", titles, adj, 0)
	if !search.portfolioTruncated["m"] {
		t.Fatal("middle node must record that its candidate portfolio was trimmed")
	}
	if !search.portfolioTruncated["t"] {
		t.Fatal("descendant must inherit a predecessor's internal portfolio loss")
	}
	_, truncated := search.witnessesToWithTruncation("t")
	if !truncated {
		t.Fatal("destination inheriting internal portfolio loss must report truncation")
	}
}

func TestTypedWitnessesPreferTypeSequenceDiversityAfterFirstHop(t *testing.T) {
	titles := map[string]string{"s": "Source", "m": "Middle", "t": "Target"}
	adj := map[string][]typedTraversalEdge{
		"s": {
			testTypedTraversal("s", "m", "extends", "c"),
			testTypedTraversal("s", "m", "requires", "d"),
			testTypedTraversal("s", "m", "extends", "b"),
			testTypedTraversal("s", "m", "extends", "a"),
		},
		"m": {testTypedTraversal("m", "t", "supports", "finish")},
	}

	got := findShortestTypedWitnesses("s", titles, adj, 0).witnessesTo("t")
	if len(got) != 3 {
		t.Fatalf("witnesses = %d, want 3", len(got))
	}
	wantAnnotations := []string{"a", "b", "d"}
	wantTypes := [][]string{{"extends", "supports"}, {"extends", "supports"}, {"requires", "supports"}}
	for i := range got {
		if got[i].Edges[0].Annotation != wantAnnotations[i] {
			t.Errorf("witness %d first annotation = %q, want %q", i, got[i].Edges[0].Annotation, wantAnnotations[i])
		}
		if types := typedWitnessTypes(got[i]); !reflect.DeepEqual(types, wantTypes[i]) {
			t.Errorf("witness %d types = %v, want %v", i, types, wantTypes[i])
		}
	}
}

func TestTypedWitnessesContainOnlyShortestPathsAndAreCycleSafe(t *testing.T) {
	titles := map[string]string{"s": "Source", "m": "Middle", "t": "Target"}
	adj := map[string][]typedTraversalEdge{
		"s": {
			testTypedTraversal("s", "m", "extends", "long route"),
			testTypedTraversal("s", "t", "supports", "direct"),
		},
		"m": {testTypedTraversal("m", "t", "supports", "second hop")},
		"t": {testTypedTraversal("t", "s", "supports", "cycle")},
	}

	search := findShortestTypedWitnesses("s", titles, adj, 0)
	got := search.witnessesTo("t")
	withTruncation, truncated := search.witnessesToWithTruncation("t")
	if !reflect.DeepEqual(withTruncation, got) || truncated {
		t.Fatalf("complete witnesses changed or reported truncation: witnesses=%#v truncated=%v", withTruncation, truncated)
	}
	if search.depthByID["t"] != 1 || len(got) != 1 {
		t.Fatalf("target depth/witnesses = %d/%d, want 1/1", search.depthByID["t"], len(got))
	}
	if path := typedWitnessNodeIDs(got[0]); !reflect.DeepEqual(path, []string{"s", "t"}) {
		t.Fatalf("path = %v, want direct shortest path", path)
	}
}

func TestTypedWitnessesBoundLongCombinatorialDAG(t *testing.T) {
	const layers = 36 // 2^36 shortest paths without bounded reconstruction.
	titles := map[string]string{"s": "Source", "t": "Target"}
	adj := make(map[string][]typedTraversalEdge)
	previous := []string{"s"}
	for layer := 0; layer < layers; layer++ {
		current := []string{
			"layer-" + strconv.Itoa(layer) + "-left",
			"layer-" + strconv.Itoa(layer) + "-right",
		}
		for _, id := range current {
			titles[id] = id
		}
		for _, from := range previous {
			for i, to := range current {
				linkType := "extends"
				if i == 1 {
					linkType = "requires"
				}
				adj[from] = append(adj[from], testTypedTraversal(from, to, linkType, from+"->"+to))
			}
		}
		if layer > 0 {
			adj[current[1]] = append(adj[current[1]], testTypedTraversal(current[1], previous[0], "supports", "back edge cycle"))
		}
		previous = current
	}
	for _, from := range previous {
		adj[from] = append(adj[from], testTypedTraversal(from, "t", "supports", "finish"))
	}

	search := findShortestTypedWitnesses("s", titles, adj, 0)
	for id, portfolio := range search.portfolios {
		if len(portfolio) > typedWitnessPortfolioLimit {
			t.Fatalf("portfolio %q = %d, exceeds internal bound %d", id, len(portfolio), typedWitnessPortfolioLimit)
		}
	}
	got := search.witnessesTo("t")
	if len(got) != typedWitnessOutputLimit {
		t.Fatalf("target witnesses = %d, want %d", len(got), typedWitnessOutputLimit)
	}
	firstHops := make(map[string]bool)
	typeSequences := make(map[string]bool)
	for i, witness := range got {
		if len(witness.Edges) != layers+1 {
			t.Errorf("witness %d edges = %d, want shortest depth %d", i, len(witness.Edges), layers+1)
		}
		firstHops[witness.Nodes[1].ID] = true
		typeSequences[typedWitnessTypeSequenceKey(witness)] = true
	}
	if len(firstHops) != 2 || len(typeSequences) != 3 {
		t.Fatalf("convergent DAG diversity = %d first hops/%d type sequences, want 2/3", len(firstHops), len(typeSequences))
	}
}

func TestTypedWitnessesUseOrdinaryLexicalFullPathOrder(t *testing.T) {
	titles := map[string]string{"s": "Source", "aa": "AA", "b": "B", "c": "C", "t": "Target"}
	adj := map[string][]typedTraversalEdge{
		"s": {
			testTypedTraversal("s", "c", "supports", "c"),
			testTypedTraversal("s", "b", "supports", "b"),
			testTypedTraversal("s", "aa", "supports", "aa"),
		},
		"aa": {testTypedTraversal("aa", "t", "supports", "aa")},
		"b":  {testTypedTraversal("b", "t", "supports", "b")},
		"c":  {testTypedTraversal("c", "t", "supports", "c")},
	}

	got := findShortestTypedWitnesses("s", titles, adj, 0).witnessesTo("t")
	want := []string{"aa", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("witnesses = %d, want %d", len(got), len(want))
	}
	for i, witness := range got {
		if witness.Nodes[1].ID != want[i] {
			t.Errorf("witness %d first hop = %q, want ordinary lexical %q", i, witness.Nodes[1].ID, want[i])
		}
	}
}

func TestTypedWitnessesIgnoreBackendAndLinkOrder(t *testing.T) {
	titles := map[string]string{"s": "Source", "a": "A", "b": "B", "c": "C", "t": "Target"}
	adj := map[string][]typedTraversalEdge{
		"s": {
			testTypedTraversal("s", "c", "requires", "c"),
			testTypedTraversal("s", "a", "extends", "a"),
			testTypedTraversal("s", "b", "extends", "b"),
		},
		"a": {testTypedTraversal("a", "t", "supports", "a")},
		"b": {testTypedTraversal("b", "t", "supports", "b")},
		"c": {testTypedTraversal("c", "t", "supports", "c")},
	}
	reversed := make(map[string][]typedTraversalEdge, len(adj))
	for id, edges := range adj {
		for i := len(edges) - 1; i >= 0; i-- {
			reversed[id] = append(reversed[id], edges[i])
		}
	}

	forward := findShortestTypedWitnesses("s", titles, adj, 0).witnessesTo("t")
	backward := findShortestTypedWitnesses("s", titles, reversed, 0).witnessesTo("t")
	if !reflect.DeepEqual(forward, backward) {
		t.Fatalf("witnesses depend on adjacency order:\nforward=%#v\nreversed=%#v", forward, backward)
	}
}

type diverseTypedWitnessFixture struct {
	focus       *note.Note
	destination *note.Note
	middles     []*note.Note
}

func setupDiverseTypedWitnessGraph(t *testing.T) (func(...string) (string, error), diverseTypedWitnessFixture) {
	t.Helper()
	nbDir, execute := setupNotebook(t)
	focus := newTestNoteForCLI("20260101000000-0001", "Origin", note.TypeConcept)
	middles := []*note.Note{
		newTestNoteForCLI("20260101000000-0002", "Middle A", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0003", "Middle B", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0004", "Middle C", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0005", "Middle D", note.TypeConcept),
	}
	destination := newTestNoteForCLI("20260101000000-0006", "Needle destination", note.TypeArgument)
	focus.Links = []note.Link{
		{TargetID: middles[3].ID, Type: "requires", Annotation: "fourth first hop"},
		{TargetID: middles[1].ID, Type: "extends", Annotation: "second first hop"},
		{TargetID: middles[2].ID, Type: "requires", Annotation: "third first hop"},
		{TargetID: middles[0].ID, Type: "extends", Annotation: "first first hop"},
	}
	for i, middle := range middles {
		middle.Links = []note.Link{{TargetID: destination.ID, Type: "supports", Annotation: "finish " + strconv.Itoa(i)}}
	}
	destination.Links = []note.Link{{TargetID: focus.ID, Type: "supports", Annotation: "cycle"}}
	for _, n := range append([]*note.Note{focus, destination}, middles...) {
		writeNoteFile(t, nbDir, n)
	}
	return execute, diverseTypedWitnessFixture{focus: focus, destination: destination, middles: middles}
}
