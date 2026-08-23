package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

type clusterSearchTestNote struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Score float64 `json:"score,omitempty"`
}

type clusterSearchTestRegion struct {
	Size           int                     `json:"size"`
	MatchCount     int                     `json:"match_count"`
	Score          float64                 `json:"score"`
	Representative clusterSearchTestNote   `json:"representative"`
	Matches        []clusterSearchTestNote `json:"matches"`
	Notes          []clusterSearchTestNote `json:"notes"`
}

type orderedClusterBackend struct {
	backend.Backend
	notes []*note.Note
}

func (b *orderedClusterBackend) List() ([]*note.Note, error) { return b.notes, nil }

func executeClustersWithNotes(t *testing.T, notes []*note.Note, args ...string) string {
	t.Helper()
	state := &rootState{backend: &orderedClusterBackend{notes: notes}}
	cmd := newClustersCmd(state)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("clusters %v: %v", args, err)
	}
	return stdout.String()
}

func TestClustersSearchProjectsHitsOntoFullGraphClusters(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	aHub := newTestNoteForCLI(note.GenerateID(), "Quasarfriction hub", note.TypeConcept)
	aMatch := newTestNoteForCLI(note.GenerateID(), "Quasarfriction evidence", note.TypeObservation)
	aContext := newTestNoteForCLI(note.GenerateID(), "Surrounding context", note.TypeConcept)
	aHub.Links = []note.Link{
		{TargetID: aMatch.ID, Type: "supports", Annotation: "first"},
		{TargetID: aContext.ID, Type: "extends", Annotation: "second"},
	}

	bHub := newTestNoteForCLI(note.GenerateID(), "Quasarfriction satellite", note.TypeConcept)
	bContext1 := newTestNoteForCLI(note.GenerateID(), "Satellite context one", note.TypeConcept)
	bContext2 := newTestNoteForCLI(note.GenerateID(), "Satellite context two", note.TypeConcept)
	bHub.Links = []note.Link{
		{TargetID: bContext1.ID, Type: "supports", Annotation: "first"},
		{TargetID: bContext2.ID, Type: "extends", Annotation: "second"},
	}

	unrelated1 := newTestNoteForCLI(note.GenerateID(), "Unrelated alpha", note.TypeConcept)
	unrelated2 := newTestNoteForCLI(note.GenerateID(), "Unrelated beta", note.TypeConcept)
	unrelated1.Links = []note.Link{{TargetID: unrelated2.ID, Type: "supports", Annotation: "unrelated"}}

	for _, n := range []*note.Note{aHub, aMatch, aContext, bHub, bContext1, bContext2, unrelated1, unrelated2} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("clusters", "--search", "quasarfriction", "--json")
	if err != nil {
		t.Fatalf("clusters --search: %v", err)
	}
	var regions []clusterSearchTestRegion
	if err := json.Unmarshal([]byte(out), &regions); err != nil {
		t.Fatalf("clusters --search JSON: %v\n%s", err, out)
	}
	if len(regions) != 2 {
		t.Fatalf("clusters --search regions = %d, want 2: %s", len(regions), out)
	}
	if regions[0].Size != 3 || regions[0].MatchCount != 2 || len(regions[0].Notes) != 3 || len(regions[0].Matches) != 2 {
		t.Fatalf("first projected region does not preserve full membership: %#v", regions[0])
	}
	if regions[0].Representative.ID != aHub.ID {
		t.Fatalf("representative = %s, want highest-degree %s", regions[0].Representative.ID, aHub.ID)
	}
	if regions[0].Score <= regions[1].Score {
		t.Fatalf("regions not ranked by aggregate search evidence: %#v", regions)
	}
	for _, region := range regions {
		for _, n := range region.Notes {
			if n.ID == unrelated1.ID || n.ID == unrelated2.ID {
				t.Fatalf("unrelated cluster was returned: %s", out)
			}
		}
	}
}

func TestClustersSearchIncludesMatchingSingletonWhenRequested(t *testing.T) {
	isolated := newTestNoteForCLI("20260101000000-0001", "Needle singleton", note.TypeConcept)
	unrelated1 := newTestNoteForCLI("20260101000000-0002", "Other alpha", note.TypeConcept)
	unrelated2 := newTestNoteForCLI("20260101000000-0003", "Other beta", note.TypeConcept)
	unrelated1.Links = []note.Link{{TargetID: unrelated2.ID, Type: "supports"}}
	notes := []*note.Note{isolated, unrelated1, unrelated2}
	if score := RankedByQuery(notes, notes, "needle", "")[isolated.ID]; score <= 0 {
		t.Fatalf("singleton fixture is not a positive search hit: score=%v", score)
	}
	out := executeClustersWithNotes(t, notes, "--search", "needle", "--json", "--singletons")
	var regions []clusterSearchTestRegion
	if err := json.Unmarshal([]byte(out), &regions); err != nil {
		t.Fatalf("clusters singleton JSON: %v\n%s", err, out)
	}
	if len(regions) != 1 || regions[0].Size != 1 || regions[0].Representative.ID != isolated.ID {
		t.Fatalf("matching singleton region missing: %s", out)
	}
}

func TestClustersSearchRankingIgnoresBackendOrder(t *testing.T) {
	aMatch := newTestNoteForCLI("20260101000000-0001", "Needle", note.TypeConcept)
	aContext := newTestNoteForCLI("20260101000000-0002", "Alpha context", note.TypeConcept)
	aMatch.Links = []note.Link{{TargetID: aContext.ID, Type: "supports"}}
	bMatch := newTestNoteForCLI("20260101000000-0003", "Needle", note.TypeConcept)
	bContext := newTestNoteForCLI("20260101000000-0004", "Beta context", note.TypeConcept)
	bMatch.Links = []note.Link{{TargetID: bContext.ID, Type: "supports"}}

	forward := []*note.Note{aMatch, aContext, bMatch, bContext}
	reverse := []*note.Note{bContext, bMatch, aContext, aMatch}
	decode := func(out string) []clusterSearchTestRegion {
		var regions []clusterSearchTestRegion
		if err := json.Unmarshal([]byte(out), &regions); err != nil {
			t.Fatalf("clusters ranking JSON: %v\n%s", err, out)
		}
		return regions
	}
	gotForward := decode(executeClustersWithNotes(t, forward, "--search", "needle", "--json"))
	gotReverse := decode(executeClustersWithNotes(t, reverse, "--search", "needle", "--json"))
	type rankedRegion struct {
		representativeID string
		score            float64
	}
	projectRanking := func(regions []clusterSearchTestRegion) []rankedRegion {
		out := make([]rankedRegion, len(regions))
		for i, region := range regions {
			out[i] = rankedRegion{representativeID: region.Representative.ID, score: region.Score}
		}
		return out
	}
	if got, want := projectRanking(gotReverse), projectRanking(gotForward); !reflect.DeepEqual(got, want) {
		t.Fatalf("search ranking depends on backend order:\nforward=%#v\nreverse=%#v", want, got)
	}
}

func TestClustersSearchRequiresJSONAndNonBlankQuery(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("clusters", "--search", "friction")
	if err == nil || !strings.Contains(err.Error(), "--search requires --json") {
		t.Fatalf("clusters --search without JSON error = %v", err)
	}
	for _, query := range []string{"", " \t "} {
		_, err := execute("clusters", "--search", query, "--json")
		if err == nil || !strings.Contains(err.Error(), "--search requires a non-blank query") {
			t.Errorf("clusters --search %q error = %v", query, err)
		}
	}
}

func TestClustersSearchRepresentativeUsesMinimalJSONShape(t *testing.T) {
	match := newTestNoteForCLI("20260101000000-0001", "Needle hub", note.TypeConcept)
	context := newTestNoteForCLI("20260101000000-0002", "Context", note.TypeConcept)
	match.Links = []note.Link{{TargetID: context.ID, Type: "supports"}}
	out := executeClustersWithNotes(t, []*note.Note{match, context}, "--search", "needle", "--json")
	var regions []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &regions); err != nil {
		t.Fatalf("clusters shape JSON: %v\n%s", err, out)
	}
	var representative map[string]json.RawMessage
	if err := json.Unmarshal(regions[0]["representative"], &representative); err != nil {
		t.Fatal(err)
	}
	if len(representative) != 2 || representative["id"] == nil || representative["title"] == nil {
		t.Fatalf("representative keys = %v, want exactly id and title: %s", representative, out)
	}
}

func TestClustersSearchSummaryOmitsNotesAndPreservesLandingHandles(t *testing.T) {
	match := newTestNoteForCLI("20260101000000-0001", "Needle hub", note.TypeConcept)
	context := newTestNoteForCLI("20260101000000-0002", "Context", note.TypeConcept)
	match.Links = []note.Link{{TargetID: context.ID, Type: "supports"}}
	out := executeClustersWithNotes(t, []*note.Note{match, context}, "--search", "needle", "--json", "--summary")
	var regions []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &regions); err != nil {
		t.Fatalf("clusters summary JSON: %v\n%s", err, out)
	}
	if len(regions) != 1 {
		t.Fatalf("summary regions = %d, want 1: %s", len(regions), out)
	}
	wantKeys := []string{"size", "match_count", "score", "representative", "matches"}
	if len(regions[0]) != len(wantKeys) {
		t.Fatalf("summary keys = %v, want exactly %v", regions[0], wantKeys)
	}
	for _, key := range wantKeys {
		if regions[0][key] == nil {
			t.Errorf("summary missing %q: %s", key, out)
		}
	}
	var representative clusterSearchTestNote
	if err := json.Unmarshal(regions[0]["representative"], &representative); err != nil {
		t.Fatal(err)
	}
	var matches []clusterSearchTestNote
	if err := json.Unmarshal(regions[0]["matches"], &matches); err != nil {
		t.Fatal(err)
	}
	if representative.ID == "" || len(matches) == 0 || matches[0].ID == "" {
		t.Fatalf("summary lacks navigation handles: %s", out)
	}
}

func TestClustersSearchSummaryRequiresSearchAndJSON(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"clusters", "--summary", "--json"}, want: "--summary requires --search"},
		{args: []string{"clusters", "--summary", "--search", "needle"}, want: "--summary requires --json"},
	} {
		_, err := execute(tc.args...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("clusters %v error = %v, want %q", tc.args, err, tc.want)
		}
	}
}

func TestClustersSearchIsDocumentedForTeleport(t *testing.T) {
	_, execute := setupNotebook(t)
	help, err := execute("clusters", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--search", "--summary"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("clusters help missing %s:\n%s", flag, help)
		}
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"nn clusters --search \"<query>\" --json --summary", "default landing-zone source", "representative.id", "recenter"} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing %q", required)
		}
	}
}
