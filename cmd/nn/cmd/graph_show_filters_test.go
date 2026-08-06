package cmd

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

type graphShowFilterResult struct {
	Center string `json:"center"`
	Nodes  []struct {
		ID string `json:"id"`
	} `json:"nodes"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
		Type string `json:"type"`
	} `json:"edges"`
}

func mustGraphShowFilter(t *testing.T, label string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
}

func graphShowFilterNote(title string, status note.Status, representation string) *note.Note {
	now := time.Now().UTC().Truncate(time.Second)
	return &note.Note{
		ID:             note.GenerateID(),
		Title:          title,
		Type:           note.TypeConcept,
		Status:         status,
		Representation: representation,
		Created:        now,
		Modified:       now,
	}
}

func TestGraphShowConstrainedTraversal(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	reviewedTax := graphShowFilterNote("reviewed taxonomy", note.StatusReviewed, "taxonomy")
	draftTax := graphShowFilterNote("draft taxonomy", note.StatusDraft, "taxonomy")
	reviewedOnt := graphShowFilterNote("reviewed ontology", note.StatusReviewed, "ontology")
	deep := graphShowFilterNote("deep permanent taxonomy", note.StatusPermanent, "taxonomy")
	blockedDeep := graphShowFilterNote("blocked deep taxonomy", note.StatusReviewed, "taxonomy")
	incoming := graphShowFilterNote("incoming reviewed taxonomy", note.StatusReviewed, "taxonomy")
	incomingDeep := graphShowFilterNote("incoming deep taxonomy", note.StatusPermanent, "taxonomy")

	root.Links = []note.Link{
		{TargetID: reviewedTax.ID, Type: "supports"},
		{TargetID: draftTax.ID, Type: "refines"},
		{TargetID: reviewedOnt.ID, Type: "supports"},
	}
	reviewedTax.Links = []note.Link{{TargetID: deep.ID, Type: "supports"}}
	draftTax.Links = []note.Link{{TargetID: blockedDeep.ID, Type: "supports"}}
	incoming.Links = []note.Link{{TargetID: root.ID, Type: "governs"}}
	incomingDeep.Links = []note.Link{{TargetID: incoming.ID, Type: "supports"}}
	for _, n := range []*note.Note{root, reviewedTax, draftTax, reviewedOnt, deep, blockedDeep, incoming, incomingDeep} {
		writeNoteFile(t, nbDir, n)
	}

	edge := func(from, typ, to string) string { return from + "|" + typ + "|" + to }
	cases := []struct {
		name  string
		args  []string
		nodes []string
		edges []string
	}{
		{
			name:  "incoming traverses sources and continues incoming",
			args:  []string{"--direction", "incoming", "--depth", "2"},
			nodes: []string{root.ID, incoming.ID, incomingDeep.ID},
			edges: []string{edge(incoming.ID, "governs", root.ID), edge(incomingDeep.ID, "supports", incoming.ID)},
		},
		{
			name:  "both unions directions at bounded depth",
			args:  []string{"--direction", "both", "--depth", "1"},
			nodes: []string{root.ID, reviewedTax.ID, draftTax.ID, reviewedOnt.ID, incoming.ID},
			edges: []string{edge(root.ID, "supports", reviewedTax.ID), edge(root.ID, "refines", draftTax.ID), edge(root.ID, "supports", reviewedOnt.ID), edge(incoming.ID, "governs", root.ID)},
		},
		{
			name:  "link filter constrains expansion",
			args:  []string{"--links", "supports", "--depth", "2"},
			nodes: []string{root.ID, reviewedTax.ID, reviewedOnt.ID, deep.ID},
			edges: []string{edge(root.ID, "supports", reviewedTax.ID), edge(root.ID, "supports", reviewedOnt.ID), edge(reviewedTax.ID, "supports", deep.ID)},
		},
		{
			name:  "status filter blocks intermediate expansion",
			args:  []string{"--status", "reviewed", "--depth", "2"},
			nodes: []string{root.ID, reviewedTax.ID, reviewedOnt.ID},
			edges: []string{edge(root.ID, "supports", reviewedTax.ID), edge(root.ID, "supports", reviewedOnt.ID)},
		},
		{
			name:  "representation filter constrains expansion",
			args:  []string{"--representation", "taxonomy", "--depth", "2"},
			nodes: []string{root.ID, reviewedTax.ID, draftTax.ID, deep.ID, blockedDeep.ID},
			edges: []string{edge(root.ID, "supports", reviewedTax.ID), edge(root.ID, "refines", draftTax.ID), edge(reviewedTax.ID, "supports", deep.ID), edge(draftTax.ID, "supports", blockedDeep.ID)},
		},
		{
			name:  "combined filters intersect during traversal",
			args:  []string{"--direction", "outgoing", "--links", "supports", "--status", "reviewed,permanent", "--representation", "taxonomy", "--depth", "2"},
			nodes: []string{root.ID, reviewedTax.ID, deep.ID},
			edges: []string{edge(root.ID, "supports", reviewedTax.ID), edge(reviewedTax.ID, "supports", deep.ID)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"graph", "show", "--focus", root.ID, "--format", "json"}
			args = append(args, tc.args...)
			out, err := execute(args...)
			mustGraphShowFilter(t, tc.name, err)
			var got graphShowFilterResult
			mustGraphShowFilter(t, tc.name+" JSON", json.Unmarshal([]byte(out), &got))
			var gotNodes []string
			for _, n := range got.Nodes {
				gotNodes = append(gotNodes, n.ID)
			}
			var gotEdges []string
			for _, e := range got.Edges {
				gotEdges = append(gotEdges, edge(e.From, e.Type, e.To))
			}
			sort.Strings(gotNodes)
			sort.Strings(gotEdges)
			sort.Strings(tc.nodes)
			sort.Strings(tc.edges)
			if strings.Join(gotNodes, ",") != strings.Join(tc.nodes, ",") {
				t.Fatalf("%s nodes = %v, want %v", tc.name, gotNodes, tc.nodes)
			}
			if strings.Join(gotEdges, ",") != strings.Join(tc.edges, ",") {
				t.Fatalf("%s edges = %v, want %v", tc.name, gotEdges, tc.edges)
			}
			if got.Center != root.ID {
				t.Fatalf("%s center = %q, want %q", tc.name, got.Center, root.ID)
			}
		})
	}
}

func TestGraphShowIncomingTextDirection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("incoming text root", note.StatusDraft, "")
	source := graphShowFilterNote("incoming text source", note.StatusReviewed, "")
	source.Links = []note.Link{{TargetID: root.ID, Type: "governs", Annotation: "incoming annotation"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, source)
	out, err := execute("graph", "show", "--focus", root.ID, "--direction", "incoming", "--depth", "1")
	mustGraphShowFilter(t, "incoming text", err)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 || !strings.Contains(lines[0], root.ID) {
		t.Fatalf("incoming text root missing or unordered: %s", out)
	}
	want := "← [governs] " + source.ID + "  " + source.Title + " — incoming annotation"
	if !strings.Contains(lines[1], want) {
		t.Fatalf("incoming text edge = %q, want substring %q", lines[1], want)
	}
}

func TestGraphShowTraversalFiltersRequireFocus(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := [][]string{
		{"--direction", "incoming"},
		{"--links", "supports"},
		{"--status", "reviewed"},
		{"--representation", "taxonomy"},
	}
	for _, args := range cases {
		_, err := execute(append([]string{"graph", "show"}, args...)...)
		if err == nil {
			t.Fatalf("no-focus traversal filter accepted: %v", args)
		}
		if !strings.Contains(err.Error(), "requires --focus") {
			t.Fatalf("no-focus traversal filter error = %q, want requires --focus", err)
		}
	}
}

func TestGraphShowRejectsInvalidTraversalFilters(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := []struct {
		flag string
		args []string
		want string
	}{
		{"direction", []string{"--direction", "sideways"}, "invalid --direction"},
		{"links", []string{"--links", "supports,teleports"}, "invalid --links value"},
		{"status", []string{"--status", "reviewed,unknown"}, "invalid --status value"},
		{"empty links", []string{"--links", "supports,,governs"}, "--links contains an empty value"},
	}
	for _, tc := range cases {
		t.Run(tc.flag, func(t *testing.T) {
			_, err := execute(append([]string{"graph", "show"}, tc.args...)...)
			if err == nil {
				t.Fatalf("invalid %s accepted", tc.flag)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid %s error = %q, want substring %q", tc.flag, err, tc.want)
			}
		})
	}
}

func TestGraphShowDefaultEqualsExplicitOutgoing(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root, _, _ := makeLinkedNotes(t, nbDir)
	incomingOnly := graphShowFilterNote("incoming only", note.StatusReviewed, "")
	incomingOnly.Links = []note.Link{{TargetID: root.ID, Type: "supports"}}
	writeNoteFile(t, nbDir, incomingOnly)
	for _, format := range []string{"text", "json"} {
		implicit, err := execute("graph", "show", "--focus", root.ID, "--depth", "2", "--format", format)
		mustGraphShowFilter(t, "implicit outgoing", err)
		explicit, err := execute("graph", "show", "--focus", root.ID, "--depth", "2", "--format", format, "--direction", "outgoing")
		mustGraphShowFilter(t, "explicit outgoing", err)
		if implicit != explicit {
			t.Fatalf("default output differs from explicit outgoing for %s:\nimplicit=%s\nexplicit=%s", format, implicit, explicit)
		}
	}
}

func TestGraphShowFilteredOutputDeterministic(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("deterministic root", note.StatusDraft, "")
	root.ID = "20990101000000-5000"
	first := graphShowFilterNote("first incoming", note.StatusReviewed, "taxonomy")
	first.ID = "20990101000000-1000"
	second := graphShowFilterNote("second incoming", note.StatusReviewed, "taxonomy")
	second.ID = "20990101000000-2000"
	first.Links = []note.Link{{TargetID: root.ID, Type: "supports"}}
	second.Links = []note.Link{{TargetID: root.ID, Type: "supports"}}
	for _, n := range []*note.Note{root, second, first} {
		writeNoteFile(t, nbDir, n)
	}
	args := []string{"graph", "show", "--focus", root.ID, "--direction", "incoming", "--format", "json"}
	baseline, err := execute(args...)
	mustGraphShowFilter(t, "deterministic baseline", err)
	for i := 0; i < 10; i++ {
		got, err := execute(args...)
		mustGraphShowFilter(t, "deterministic repeat", err)
		if got != baseline {
			t.Fatalf("filtered graph output changed between runs:\nfirst=%s\nrepeat=%s", baseline, got)
		}
	}
	var result graphShowFilterResult
	mustGraphShowFilter(t, "deterministic JSON", json.Unmarshal([]byte(baseline), &result))
	if len(result.Edges) != 2 || result.Edges[0].From != first.ID || result.Edges[1].From != second.ID {
		t.Fatalf("incoming edge order = %v, want source IDs ascending", result.Edges)
	}
}
