package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

type bridgeSearchWitnessEdge struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Annotation string `json:"annotation"`
}

type bridgeSearchRegionSummary struct {
	Representative clusterSearchTestNote `json:"representative"`
	Size           int                   `json:"size"`
}

type bridgeSearchWitnessRegions struct {
	Incoming   *bridgeSearchRegionSummary `json:"incoming"`
	Outgoing   *bridgeSearchRegionSummary `json:"outgoing"`
	SameRegion bool                       `json:"same_region"`
}

type bridgeSearchWitness struct {
	Incoming bridgeSearchWitnessEdge     `json:"incoming"`
	Outgoing bridgeSearchWitnessEdge     `json:"outgoing"`
	Regions  *bridgeSearchWitnessRegions `json:"regions"`
}

type bridgeSearchResult struct {
	ID             string                `json:"id"`
	Title          string                `json:"title"`
	Score          int                   `json:"score"`
	RelevanceScore *float64              `json:"relevance_score"`
	Witnesses      []bridgeSearchWitness `json:"witnesses"`
}

type orderedBridgesBackend struct {
	backend.Backend
	notes []*note.Note
}

func (b *orderedBridgesBackend) List() ([]*note.Note, error) { return b.notes, nil }

func executeBridgesWithNotes(t *testing.T, notes []*note.Note, args ...string) string {
	t.Helper()
	state := &rootState{backend: &orderedBridgesBackend{notes: notes}}
	cmd := newGraphBridgesCmd(state)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("graph bridges %v: %v", args, err)
	}
	return stdout.String()
}

func TestGraphBridgesSearchProjectsOntoFullGraphBridges(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	left := newTestNoteForCLI("20260101000000-0001", "Left context", note.TypeConcept)
	matchingBridge := newTestNoteForCLI("20260101000000-0002", "Quasarbridge crossing", note.TypeConcept)
	right := newTestNoteForCLI("20260101000000-0003", "Right context", note.TypeConcept)
	// Put the lexicographically later edge first so witness selection cannot
	// depend on stored edge order.
	left.Links = []note.Link{
		{TargetID: matchingBridge.ID, Type: "supports", Annotation: "later inbound"},
		{TargetID: matchingBridge.ID, Type: "extends", Annotation: "chosen inbound"},
	}
	matchingBridge.Links = []note.Link{
		{TargetID: right.ID, Type: "supports", Annotation: "later outgoing"},
		{TargetID: right.ID, Type: "grounded-by", Annotation: "chosen outgoing"},
	}

	matchingNonBridge := newTestNoteForCLI("20260101000000-0004", "Quasarbridge isolated", note.TypeConcept)
	otherLeft := newTestNoteForCLI("20260101000000-0005", "Other left", note.TypeConcept)
	unrelatedBridge := newTestNoteForCLI("20260101000000-0006", "Unrelated crossing", note.TypeConcept)
	otherRight := newTestNoteForCLI("20260101000000-0007", "Other right", note.TypeConcept)
	otherLeft.Links = []note.Link{{TargetID: unrelatedBridge.ID}}
	unrelatedBridge.Links = []note.Link{{TargetID: otherRight.ID}}

	for _, n := range []*note.Note{left, matchingBridge, right, matchingNonBridge, otherLeft, unrelatedBridge, otherRight} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "bridges", "--search", "quasarbridge", "--format", "json")
	if err != nil {
		t.Fatalf("graph bridges --search: %v", err)
	}
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("graph bridges --search JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].ID != matchingBridge.ID {
		t.Fatalf("projected bridges = %#v, want only full-graph bridge %s", got, matchingBridge.ID)
	}
	if got[0].Score <= 0 || got[0].RelevanceScore == nil || *got[0].RelevanceScore <= 0 {
		t.Fatalf("bridge result lacks structural and relevance scores: %#v", got[0])
	}
	if len(got[0].Witnesses) != 1 || got[0].Witnesses[0].Regions == nil {
		t.Fatalf("search bridge result lacks one crossing with default region context: %#v", got[0])
	}
	if want := (bridgeSearchWitnessEdge{ID: left.ID, Title: left.Title, Type: "extends", Annotation: "chosen inbound"}); got[0].Witnesses[0].Incoming != want {
		t.Errorf("incoming witness = %#v, want %#v", got[0].Witnesses[0].Incoming, want)
	}
	if want := (bridgeSearchWitnessEdge{ID: right.ID, Title: right.Title, Type: "grounded-by", Annotation: "chosen outgoing"}); got[0].Witnesses[0].Outgoing != want {
		t.Errorf("outgoing witness = %#v, want %#v", got[0].Witnesses[0].Outgoing, want)
	}
}

func TestGraphBridgesRegionsUseFullGraphClusters(t *testing.T) {
	left := []*note.Note{
		newTestNoteForCLI("20260101000000-0001", "Left one", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0002", "Left two", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0003", "Left three", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0004", "Left gateway", note.TypeConcept),
	}
	bridge := newTestNoteForCLI("20260101000000-0005", "Quasarregion bridge", note.TypeConcept)
	right := []*note.Note{
		newTestNoteForCLI("20260101000000-0006", "Right one", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0007", "Right two", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0008", "Right three", note.TypeConcept),
		newTestNoteForCLI("20260101000000-0009", "Right gateway", note.TypeConcept),
	}
	addClique := func(nodes []*note.Note) {
		for i := range nodes {
			for j := i + 1; j < len(nodes); j++ {
				nodes[i].Links = append(nodes[i].Links, note.Link{TargetID: nodes[j].ID, Type: "extends"})
			}
		}
	}
	addClique(left)
	addClique(right)
	left[3].Links = append(left[3].Links, note.Link{TargetID: bridge.ID, Type: "supports"})
	bridge.Links = []note.Link{{TargetID: right[3].ID, Type: "extends"}}
	notes := append(append(append([]*note.Note{}, left...), bridge), right...)

	out := executeBridgesWithNotes(t, notes, "--search", "quasarregion", "--format", "json")
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("region-aware bridges JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || len(got[0].Witnesses) != 1 || got[0].Witnesses[0].Regions == nil {
		t.Fatalf("region-aware bridge result = %#v", got)
	}
	regions := got[0].Witnesses[0].Regions
	if regions.SameRegion {
		t.Fatalf("different full-graph regions reported as same: %#v", regions)
	}
	if regions.Incoming == nil || regions.Incoming.Size != 5 || regions.Incoming.Representative.ID != left[3].ID {
		t.Errorf("incoming region = %#v, want full size 5 represented by %s", regions.Incoming, left[3].ID)
	}
	if regions.Outgoing == nil || regions.Outgoing.Size != 4 || regions.Outgoing.Representative.ID != right[3].ID {
		t.Errorf("outgoing region = %#v, want full size 4 represented by %s", regions.Outgoing, right[3].ID)
	}
	plainOut := executeBridgesWithNotes(t, notes, "--format", "json")
	var plain []bridgeSearchResult
	if err := json.Unmarshal([]byte(plainOut), &plain); err != nil {
		t.Fatal(err)
	}
	var plainBridge *bridgeSearchResult
	for i := range plain {
		if plain[i].ID == bridge.ID {
			plainBridge = &plain[i]
			break
		}
	}
	if plainBridge == nil || !reflect.DeepEqual(plainBridge.Witnesses, got[0].Witnesses) {
		t.Fatalf("non-search did not use the same full-topology witnesses: %#v", plainBridge)
	}

	var raw []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	var rawWitnesses []map[string]json.RawMessage
	if err := json.Unmarshal(raw[0]["witnesses"], &rawWitnesses); err != nil {
		t.Fatal(err)
	}
	if len(rawWitnesses) != 1 {
		t.Fatalf("witnesses = %s, want one", raw[0]["witnesses"])
	}
	var rawRegions map[string]json.RawMessage
	if err := json.Unmarshal(rawWitnesses[0]["regions"], &rawRegions); err != nil {
		t.Fatal(err)
	}
	if len(rawRegions) != 3 || rawRegions["incoming"] == nil || rawRegions["outgoing"] == nil || rawRegions["same_region"] == nil {
		t.Fatalf("regions metadata is not bounded to endpoint summaries and same_region: %s", out)
	}
	for _, endpoint := range []string{"incoming", "outgoing"} {
		var summary map[string]json.RawMessage
		if err := json.Unmarshal(rawRegions[endpoint], &summary); err != nil {
			t.Fatal(err)
		}
		if len(summary) != 2 || summary["representative"] == nil || summary["size"] == nil {
			t.Errorf("%s region exposes unexpected fields: %s", endpoint, rawRegions[endpoint])
		}
		var representative map[string]json.RawMessage
		if err := json.Unmarshal(summary["representative"], &representative); err != nil {
			t.Fatal(err)
		}
		if len(representative) != 2 || representative["id"] == nil || representative["title"] == nil {
			t.Errorf("%s representative exposes unexpected fields: %s", endpoint, summary["representative"])
		}
	}
}

func TestGraphBridgesRegionsCanReportSameRegion(t *testing.T) {
	incoming := newTestNoteForCLI("20260101000000-0001", "Incoming", note.TypeConcept)
	bridge := newTestNoteForCLI("20260101000000-0002", "Sameregionneedle bridge", note.TypeConcept)
	outgoing := newTestNoteForCLI("20260101000000-0003", "Outgoing", note.TypeConcept)
	incoming.Links = []note.Link{{TargetID: bridge.ID, Type: "supports"}}
	bridge.Links = []note.Link{{TargetID: outgoing.ID, Type: "extends"}}

	out := executeBridgesWithNotes(t, []*note.Note{incoming, bridge, outgoing}, "--format", "json")
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Witnesses) != 1 || got[0].Witnesses[0].Regions == nil || !got[0].Witnesses[0].Regions.SameRegion {
		t.Fatalf("same-region witness metadata = %#v, want same_region true", got)
	}
	regions := got[0].Witnesses[0].Regions
	if regions.Incoming == nil || regions.Outgoing == nil || regions.Incoming.Size != 3 || !reflect.DeepEqual(regions.Incoming, regions.Outgoing) {
		t.Fatalf("same-region summaries = %#v", regions)
	}
	if regions.Incoming.Representative.ID != bridge.ID {
		t.Errorf("representative = %s, want highest-degree bridge %s", regions.Incoming.Representative.ID, bridge.ID)
	}
}

func TestGraphBridgesRegionsIgnoreBackendAndLinkOrder(t *testing.T) {
	incoming := newTestNoteForCLI("20260101000000-0001", "Incoming", note.TypeConcept)
	bridge := newTestNoteForCLI("20260101000000-0002", "Orderneedle bridge", note.TypeConcept)
	outgoing := newTestNoteForCLI("20260101000000-0003", "Outgoing", note.TypeConcept)
	incoming.Links = []note.Link{
		{TargetID: bridge.ID, Type: "supports", Annotation: "later"},
		{TargetID: bridge.ID, Type: "extends", Annotation: "chosen"},
	}
	bridge.Links = []note.Link{
		{TargetID: outgoing.ID, Type: "supports", Annotation: "later"},
		{TargetID: outgoing.ID, Type: "extends", Annotation: "chosen"},
	}
	forward := []*note.Note{incoming, bridge, outgoing}
	for _, args := range [][]string{
		{"--format", "json"},
		{"--search", "orderneedle", "--format", "json"},
	} {
		want := executeBridgesWithNotes(t, forward, args...)
		for _, n := range forward {
			for i, j := 0, len(n.Links)-1; i < j; i, j = i+1, j-1 {
				n.Links[i], n.Links[j] = n.Links[j], n.Links[i]
			}
		}
		reverse := []*note.Note{outgoing, bridge, incoming}
		got := executeBridgesWithNotes(t, reverse, args...)
		if got != want {
			t.Fatalf("bridge output %v depends on backend/link order:\nforward=%s\nreverse=%s", args, want, got)
		}
	}
}

func TestGraphBridgesWitnessesPreferDistinctRegionPairsAndCap(t *testing.T) {
	community := func(base int, title string) []*note.Note {
		nodes := make([]*note.Note, 4)
		for i := range nodes {
			nodes[i] = newTestNoteForCLI(fmt.Sprintf("20260101000000-%04d", base+i), fmt.Sprintf("%s %d", title, i+1), note.TypeConcept)
		}
		for i := range nodes {
			for j := i + 1; j < len(nodes); j++ {
				nodes[i].Links = append(nodes[i].Links, note.Link{TargetID: nodes[j].ID, Type: "extends"})
			}
		}
		return nodes
	}

	incomingA := community(100, "Incoming A")
	incomingB := community(200, "Incoming B")
	bridge := newTestNoteForCLI("20260101000000-0050", "Diversityneedle bridge", note.TypeConcept)
	outgoingC := community(600, "Outgoing C")
	outgoingD := community(700, "Outgoing D")

	incomingA[2].Links = append(incomingA[2].Links,
		note.Link{TargetID: bridge.ID, Type: "supports", Annotation: "later inbound A"},
		note.Link{TargetID: bridge.ID, Type: "extends", Annotation: "chosen inbound A"},
	)
	incomingA[3].Links = append(incomingA[3].Links, note.Link{TargetID: bridge.ID, Type: "extends", Annotation: "duplicate region A"})
	incomingB[3].Links = append(incomingB[3].Links, note.Link{TargetID: bridge.ID, Type: "supports", Annotation: "inbound B"})
	bridge.Links = []note.Link{
		{TargetID: outgoingD[2].ID, Type: "supports", Annotation: "later outgoing D"},
		{TargetID: incomingA[1].ID, Type: "extends", Annotation: "duplicate same region"},
		{TargetID: outgoingC[2].ID, Type: "supports", Annotation: "later outgoing C"},
		{TargetID: incomingA[0].ID, Type: "supports", Annotation: "later same region"},
		{TargetID: outgoingD[2].ID, Type: "extends", Annotation: "chosen outgoing D"},
		{TargetID: outgoingC[3].ID, Type: "extends", Annotation: "duplicate region C"},
		{TargetID: incomingA[0].ID, Type: "extends", Annotation: "chosen same region"},
		{TargetID: outgoingC[2].ID, Type: "extends", Annotation: "chosen outgoing C"},
	}

	notes := append(append(append(append([]*note.Note{}, incomingA...), incomingB...), bridge), outgoingC...)
	notes = append(notes, outgoingD...)
	findBridge := func(out string) bridgeSearchResult {
		t.Helper()
		var got []bridgeSearchResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("bridge JSON: %v\n%s", err, out)
		}
		for _, result := range got {
			if result.ID == bridge.ID {
				return result
			}
		}
		t.Fatalf("bridge %s missing: %s", bridge.ID, out)
		return bridgeSearchResult{}
	}

	for _, args := range [][]string{
		{"--format", "json"},
		{"--search", "diversityneedle", "--format", "json"},
	} {
		out := executeBridgesWithNotes(t, notes, args...)
		got := findBridge(out)
		if len(got.Witnesses) != 3 {
			t.Fatalf("witness count %v = %d, want cap 3: %#v", args, len(got.Witnesses), got.Witnesses)
		}
		wantIncoming := map[string]bridgeSearchWitnessEdge{
			incomingA[2].ID: {ID: incomingA[2].ID, Title: incomingA[2].Title, Type: "extends", Annotation: "chosen inbound A"},
			incomingB[3].ID: {ID: incomingB[3].ID, Title: incomingB[3].Title, Type: "supports", Annotation: "inbound B"},
		}
		wantOutgoing := map[string]bridgeSearchWitnessEdge{
			incomingA[0].ID: {ID: incomingA[0].ID, Title: incomingA[0].Title, Type: "extends", Annotation: "chosen same region"},
			outgoingC[2].ID: {ID: outgoingC[2].ID, Title: outgoingC[2].Title, Type: "extends", Annotation: "chosen outgoing C"},
			outgoingD[2].ID: {ID: outgoingD[2].ID, Title: outgoingD[2].Title, Type: "extends", Annotation: "chosen outgoing D"},
		}
		for i, witness := range got.Witnesses {
			if want, ok := wantIncoming[witness.Incoming.ID]; !ok || witness.Incoming != want {
				t.Errorf("crossing %d incoming = %#v, want earliest edge for its region", i+1, witness.Incoming)
			}
			if want, ok := wantOutgoing[witness.Outgoing.ID]; !ok || witness.Outgoing != want {
				t.Errorf("crossing %d outgoing = %#v, want earliest edge for its region", i+1, witness.Outgoing)
			}
			if witness.Regions == nil {
				t.Fatalf("crossing %d lacks regions", i+1)
			}
		}
		if !got.Witnesses[0].Regions.SameRegion || got.Witnesses[1].Regions.SameRegion || got.Witnesses[2].Regions.SameRegion {
			t.Fatalf("same-region diversity = %#v, want only crossing 1 same-region", got.Witnesses)
		}
		pairKeys := make(map[string]bool)
		previousKey := ""
		for _, witness := range got.Witnesses {
			incomingRep, outgoingRep := "\x00unclustered:"+witness.Incoming.ID, "\x00unclustered:"+witness.Outgoing.ID
			if witness.Regions.Incoming != nil {
				incomingRep = witness.Regions.Incoming.Representative.ID
			}
			if witness.Regions.Outgoing != nil {
				outgoingRep = witness.Regions.Outgoing.Representative.ID
			}
			key := incomingRep + "->" + outgoingRep
			if pairKeys[key] {
				t.Fatalf("duplicate region-pair %s crowded witnesses: %#v", key, got.Witnesses)
			}
			if previousKey != "" && key < previousKey {
				t.Fatalf("region-pair ordering = %q before %q", previousKey, key)
			}
			pairKeys[key] = true
			previousKey = key
		}
	}

	text := executeBridgesWithNotes(t, notes, "--search", "diversityneedle")
	for i := 1; i <= 3; i++ {
		if !strings.Contains(text, fmt.Sprintf("crossing %d:", i)) {
			t.Errorf("diverse bridge text missing crossing %d:\n%s", i, text)
		}
	}
	if strings.Contains(text, "crossing 4:") || strings.Count(text, "inbound edge:") != 3 || strings.Count(text, "same region:") != 3 {
		t.Errorf("diverse bridge text is not capped with complete per-crossing evidence:\n%s", text)
	}

	for _, args := range [][]string{
		{"--format", "json"},
		{"--search", "diversityneedle", "--format", "json"},
	} {
		want := executeBridgesWithNotes(t, notes, args...)
		for _, n := range notes {
			for i, j := 0, len(n.Links)-1; i < j; i, j = i+1, j-1 {
				n.Links[i], n.Links[j] = n.Links[j], n.Links[i]
			}
		}
		sort.Slice(notes, func(i, j int) bool { return notes[i].ID > notes[j].ID })
		got := executeBridgesWithNotes(t, notes, args...)
		if got != want {
			t.Fatalf("diverse bridge output %v depends on backend/link order:\nforward=%s\nreverse=%s", args, want, got)
		}
	}
}

func TestGraphBridgesUsesUnifiedJSONSchemaWithNullableRelevance(t *testing.T) {
	left := newTestNoteForCLI("20260101000000-0001", "Left", note.TypeConcept)
	bridge := newTestNoteForCLI("20260101000000-0002", "Needle bridge", note.TypeConcept)
	right := newTestNoteForCLI("20260101000000-0003", "Right", note.TypeConcept)
	left.Links = []note.Link{{TargetID: bridge.ID, Type: "supports", Annotation: "inbound evidence"}}
	bridge.Links = []note.Link{{TargetID: right.ID, Type: "extends", Annotation: "outgoing evidence"}}
	notes := []*note.Note{left, bridge, right}

	outputs := []struct {
		name       string
		args       []string
		wantNilRel bool
	}{
		{name: "without search", args: []string{"--format", "json"}, wantNilRel: true},
		{name: "with search", args: []string{"--search", "needle", "--format", "json"}},
	}
	var witnessSets [][]bridgeSearchWitness
	for _, test := range outputs {
		t.Run(test.name, func(t *testing.T) {
			out := executeBridgesWithNotes(t, notes, test.args...)
			var raw []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(out), &raw); err != nil {
				t.Fatalf("bridge JSON: %v\n%s", err, out)
			}
			if len(raw) != 1 {
				t.Fatalf("bridge JSON result = %s", out)
			}
			wantKeys := []string{"id", "relevance_score", "score", "title", "witnesses"}
			gotKeys := make([]string, 0, len(raw[0]))
			for key := range raw[0] {
				gotKeys = append(gotKeys, key)
			}
			sort.Strings(gotKeys)
			if !reflect.DeepEqual(gotKeys, wantKeys) {
				t.Fatalf("top-level JSON keys = %v, want %v: %s", gotKeys, wantKeys, out)
			}
			var got []bridgeSearchResult
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatal(err)
			}
			if (got[0].RelevanceScore == nil) != test.wantNilRel {
				t.Errorf("relevance_score = %#v, want nil=%t", got[0].RelevanceScore, test.wantNilRel)
			}
			if !test.wantNilRel && *got[0].RelevanceScore <= 0 {
				t.Errorf("normalized relevance_score = %v, want positive", *got[0].RelevanceScore)
			}
			if _, singularPresent := raw[0]["witness"]; singularPresent {
				t.Fatalf("legacy singular witness key is present: %s", out)
			}
			if len(got[0].Witnesses) != 1 {
				t.Fatalf("witnesses = %#v, want one", got[0].Witnesses)
			}
			witnessSets = append(witnessSets, got[0].Witnesses)
		})
	}
	if len(witnessSets) != 2 || !reflect.DeepEqual(witnessSets[0], witnessSets[1]) {
		t.Fatalf("search and non-search witnesses differ: %#v", witnessSets)
	}
}

func TestGraphBridgesTextCarriesUnifiedEvidence(t *testing.T) {
	left := newTestNoteForCLI("20260101000000-0001", "Left", note.TypeConcept)
	bridge := newTestNoteForCLI("20260101000000-0002", "Needle bridge", note.TypeConcept)
	right := newTestNoteForCLI("20260101000000-0003", "Right", note.TypeConcept)
	left.Links = []note.Link{{TargetID: bridge.ID, Type: "supports", Annotation: "inbound evidence"}}
	bridge.Links = []note.Link{{TargetID: right.ID, Type: "extends", Annotation: "outgoing evidence"}}
	notes := []*note.Note{left, bridge, right}

	plain := executeBridgesWithNotes(t, notes)
	searched := executeBridgesWithNotes(t, notes, "--search", "needle")
	common := []string{
		bridge.ID + "  " + bridge.Title + "  (score 1)",
		"crossing 1:",
		"inbound edge: " + left.ID + "  " + left.Title + " -> " + bridge.ID,
		"type: \"supports\", annotation: \"inbound evidence\"",
		"outgoing edge: " + bridge.ID + "  " + bridge.Title + " -> " + right.ID,
		"type: \"extends\", annotation: \"outgoing evidence\"",
		"incoming region: representative " + bridge.ID + "  " + bridge.Title + "; size: 3",
		"outgoing region: representative " + bridge.ID + "  " + bridge.Title + "; size: 3",
		"same region: true",
	}
	for _, output := range []string{plain, searched} {
		for _, field := range common {
			if !strings.Contains(output, field) {
				t.Errorf("bridge text missing %q:\n%s", field, output)
			}
		}
	}
	if !strings.Contains(plain, "relevance: n/a") {
		t.Errorf("non-search bridge text lacks unavailable relevance:\n%s", plain)
	}
	if !strings.Contains(searched, "relevance: 1.000000") {
		t.Errorf("search bridge text lacks normalized relevance:\n%s", searched)
	}
	for _, output := range []string{plain, searched} {
		if strings.Contains(output, "crossing 2:") {
			t.Errorf("single-pair bridge text has an extra crossing:\n%s", output)
		}
	}
}

func TestGraphBridgesReportsUnclusteredEndpoint(t *testing.T) {
	left := newTestNoteForCLI("20260101000000-0001", "Left", note.TypeConcept)
	bridge := newTestNoteForCLI("20260101000000-0002", "Bridge", note.TypeConcept)
	left.Links = []note.Link{{TargetID: bridge.ID, Type: "supports"}}
	bridge.Links = []note.Link{
		{TargetID: "missing-z", Type: "extends"},
		{TargetID: "missing-a", Type: "supports"},
	}
	notes := []*note.Note{left, bridge}

	out := executeBridgesWithNotes(t, notes, "--format", "json")
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Witnesses) != 2 {
		t.Fatalf("unclustered endpoint witnesses = %#v, want two distinct region pairs", got)
	}
	for _, witness := range got[0].Witnesses {
		if witness.Regions == nil || witness.Regions.Outgoing != nil || witness.Regions.SameRegion {
			t.Fatalf("unclustered outgoing endpoint context = %#v", got)
		}
	}
	if got[0].Witnesses[0].Outgoing.ID != "missing-a" || got[0].Witnesses[1].Outgoing.ID != "missing-z" {
		t.Fatalf("unclustered sentinel ordering = %#v, want missing-a then missing-z", got[0].Witnesses)
	}
	text := executeBridgesWithNotes(t, notes)
	if strings.Count(text, "outgoing region: unclustered") != 2 || strings.Count(text, "same region: false") != 2 {
		t.Fatalf("unclustered endpoints missing from text:\n%s", text)
	}
}

func TestGraphBridgesNonSearchOrdersByScoreThenID(t *testing.T) {
	lowID := newTestNoteForCLI("20260101000000-0010", "Low ID", note.TypeConcept)
	highID := newTestNoteForCLI("20260101000000-0020", "High ID", note.TypeConcept)
	highScore := newTestNoteForCLI("20260101000000-0030", "High score", note.TypeConcept)
	leftLow := newTestNoteForCLI("20260101000000-0101", "Left low", note.TypeConcept)
	leftHighID := newTestNoteForCLI("20260101000000-0102", "Left high ID", note.TypeConcept)
	leftHighScoreA := newTestNoteForCLI("20260101000000-0103", "Left high score A", note.TypeConcept)
	leftHighScoreB := newTestNoteForCLI("20260101000000-0104", "Left high score B", note.TypeConcept)
	right := newTestNoteForCLI("20260101000000-0200", "Right", note.TypeConcept)
	leftLow.Links = []note.Link{{TargetID: lowID.ID}}
	leftHighID.Links = []note.Link{{TargetID: highID.ID}}
	leftHighScoreA.Links = []note.Link{{TargetID: highScore.ID}}
	leftHighScoreB.Links = []note.Link{{TargetID: highScore.ID}}
	lowID.Links = []note.Link{{TargetID: right.ID}}
	highID.Links = []note.Link{{TargetID: right.ID}}
	highScore.Links = []note.Link{{TargetID: right.ID}}
	notes := []*note.Note{highID, leftHighScoreB, lowID, right, highScore, leftLow, leftHighID, leftHighScoreA}

	out := executeBridgesWithNotes(t, notes, "--format", "json")
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	want := []string{highScore.ID, lowID.ID, highID.ID}
	if len(got) != len(want) {
		t.Fatalf("ordered bridges = %#v", got)
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("bridge order = %#v, want %v", got, want)
		}
	}
}

func TestGraphBridgesSearchRanksByRelevanceBeforeBridgeScore(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	highRel := newTestNoteForCLI("20260101000000-0010", "Needle needle crossing", note.TypeConcept)
	lowRelHighBridge := newTestNoteForCLI("20260101000000-0020", "Needle crossing", note.TypeConcept)
	left1 := newTestNoteForCLI("20260101000000-0011", "L1", note.TypeConcept)
	right1 := newTestNoteForCLI("20260101000000-0012", "R1", note.TypeConcept)
	left1.Links = []note.Link{{TargetID: highRel.ID}}
	highRel.Links = []note.Link{{TargetID: right1.ID}}

	left2a := newTestNoteForCLI("20260101000000-0021", "L2a", note.TypeConcept)
	left2b := newTestNoteForCLI("20260101000000-0022", "L2b", note.TypeConcept)
	right2a := newTestNoteForCLI("20260101000000-0023", "R2a", note.TypeConcept)
	right2b := newTestNoteForCLI("20260101000000-0024", "R2b", note.TypeConcept)
	left2a.Links = []note.Link{{TargetID: lowRelHighBridge.ID}}
	left2b.Links = []note.Link{{TargetID: lowRelHighBridge.ID}}
	lowRelHighBridge.Links = []note.Link{{TargetID: right2a.ID}, {TargetID: right2b.ID}}

	for _, n := range []*note.Note{highRel, lowRelHighBridge, left1, right1, left2a, left2b, right2a, right2b} {
		writeNoteFile(t, nbDir, n)
	}
	out, err := execute("graph", "bridges", "--search", "needle", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("bridge count = %d, want 2: %s", len(got), out)
	}
	if got[0].ID != highRel.ID || got[0].RelevanceScore == nil || got[1].RelevanceScore == nil ||
		*got[0].RelevanceScore <= *got[1].RelevanceScore || got[0].Score >= got[1].Score {
		t.Fatalf("bridges not relevance-first: %#v", got)
	}
}

func TestGraphBridgesSearchExcludesBeforeLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	var notes []*note.Note
	for i := 1; i <= 3; i++ {
		left := newTestNoteForCLI(fmt.Sprintf("20260101000000-10%d1", i), fmt.Sprintf("Left %d", i), note.TypeConcept)
		bridge := newTestNoteForCLI(fmt.Sprintf("20260101000000-10%d2", i), strings.Repeat("Needle ", 4-i)+fmt.Sprintf("bridge %d", i), note.TypeConcept)
		right := newTestNoteForCLI(fmt.Sprintf("20260101000000-10%d3", i), fmt.Sprintf("Right %d", i), note.TypeConcept)
		left.Links = []note.Link{{TargetID: bridge.ID, Type: "supports"}}
		bridge.Links = []note.Link{{TargetID: right.ID, Type: "extends"}}
		notes = append(notes, left, bridge, right)
	}
	for _, n := range notes {
		writeNoteFile(t, nbDir, n)
	}

	decode := func(args ...string) []bridgeSearchResult {
		out, err := execute(args...)
		if err != nil {
			t.Fatalf("graph bridges %v: %v", args, err)
		}
		var got []bridgeSearchResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("graph bridges JSON: %v\n%s", err, out)
		}
		return got
	}
	baseline := decode("graph", "bridges", "--search", "needle", "--format", "json")
	if len(baseline) != 3 {
		t.Fatalf("baseline bridge count = %d, want 3", len(baseline))
	}
	got := decode("graph", "bridges", "--search", "needle", "--format", "json", "--exclude", baseline[0].ID, "--exclude", baseline[1].ID, "--limit", "1")
	if len(got) != 1 || got[0].ID != baseline[2].ID {
		t.Fatalf("excluded limited bridges = %#v, want baseline third %s", got, baseline[2].ID)
	}
	withUnknown := decode("graph", "bridges", "--search", "needle", "--format", "json", "--exclude", "unknown-note")
	if len(withUnknown) != len(baseline) {
		t.Fatalf("unknown exclusion changed results: got %d, want %d", len(withUnknown), len(baseline))
	}
}

func TestGraphBridgesSearchAcceptsTextAndValidatesArguments(t *testing.T) {
	_, execute := setupNotebook(t)
	if _, err := execute("graph", "bridges", "--search", "needle"); err != nil {
		t.Fatalf("text bridge search: %v", err)
	}
	for _, query := range []string{"", " \t "} {
		_, err := execute("graph", "bridges", "--search", query, "--format", "json")
		if err == nil || !strings.Contains(err.Error(), "--search requires a non-blank query") {
			t.Errorf("bridge search %q error = %v", query, err)
		}
	}
	for _, format := range []string{"yaml", "", "JSON"} {
		_, err := execute("graph", "bridges", "--format", format)
		if err == nil || !strings.Contains(err.Error(), "unsupported format") {
			t.Errorf("bridge format %q error = %v", format, err)
		}
	}
	_, err := execute("graph", "bridges", "--search", "needle", "--format", "json", "--exclude", " \t ")
	if err == nil || !strings.Contains(err.Error(), "--exclude requires a non-blank ID") {
		t.Errorf("blank bridge exclusion error = %v", err)
	}
	_, err = execute("graph", "bridges", "--exclude", "note-id")
	if err == nil || !strings.Contains(err.Error(), "--exclude requires --search") {
		t.Errorf("bridge exclusion without search error = %v", err)
	}
}

func TestGraphBridgesSearchIsDocumentedForScan(t *testing.T) {
	_, execute := setupNotebook(t)
	help, err := execute("graph", "bridges", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--search", "--exclude"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("bridges help missing %s:\n%s", flag, help)
		}
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"nn graph bridges --search \"<query>\" --format json", "--format text|json", "only the encoding", "--exclude <focus-id>", "before `--limit`", "Peek", "Recenter", "relevance_score", "JSON encodes `relevance_score` as `null`", "`witnesses`", "at most 3", "region pair", "full-graph label-propagation", "same_region", "not proof", "no durable region ID"} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing %q", required)
		}
	}
	virtualReference, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"unified rich bridge model", "--format text|json", "relevance_score", "JSON `null` / text `n/a`", "deterministic `witnesses`", "at most 3", "region pair", "full-graph label propagation", "same_region", "No durable region IDs"} {
		if !strings.Contains(string(virtualReference), required) {
			t.Errorf("virtual CLI reference missing %q", required)
		}
	}
}
