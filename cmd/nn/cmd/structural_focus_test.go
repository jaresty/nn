package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

type clusterFocusEnvelope struct {
	Focus   clusterSearchTestNote `json:"focus"`
	Cluster *clusterFocusRecord   `json:"cluster"`
}

type clusterFocusRecord struct {
	Size           int                     `json:"size"`
	Representative clusterSearchTestNote   `json:"representative"`
	Notes          []clusterSearchTestNote `json:"notes"`
}

type bridgeFocusEnvelope struct {
	Focus  clusterSearchTestNote `json:"focus"`
	Bridge *bridgeSearchResult   `json:"bridge"`
}

// installFocusFlagForRedPhase keeps the pre-implementation test run on the
// command's present behavior instead of stopping at Cobra's unknown-flag path.
// Once production registers --focus this is a no-op.
func installFocusFlagForRedPhase(cmd *cobra.Command) {
	if cmd.Flags().Lookup("focus") == nil {
		cmd.Flags().String("focus", "", "Exact note ID (ADR-0036 contract test shim)")
	}
}

func executeClustersFocusContract(notes []*note.Note, args ...string) (string, error) {
	state := &rootState{backend: &orderedClusterBackend{notes: notes}}
	cmd := newClustersCmd(state)
	installFocusFlagForRedPhase(cmd)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

func executeBridgesFocusContract(notes []*note.Note, args ...string) (string, error) {
	state := &rootState{backend: &orderedBridgesBackend{notes: notes}}
	cmd := newGraphBridgesCmd(state)
	installFocusFlagForRedPhase(cmd)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

func decodeClusterFocusEnvelope(t *testing.T, out string) clusterFocusEnvelope {
	t.Helper()
	var shape any
	if err := json.Unmarshal([]byte(out), &shape); err != nil {
		t.Fatalf("clusters --focus returned invalid JSON: %v\n%s", err, out)
	}
	if _, ok := shape.(map[string]any); !ok {
		t.Fatalf("clusters --focus top-level JSON = %T, want object envelope: %s", shape, out)
	}
	var envelope clusterFocusEnvelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("clusters --focus envelope: %v\n%s", err, out)
	}
	return envelope
}

func decodeBridgeFocusEnvelope(t *testing.T, out string) bridgeFocusEnvelope {
	t.Helper()
	var shape any
	if err := json.Unmarshal([]byte(out), &shape); err != nil {
		t.Fatalf("graph bridges --focus returned invalid JSON: %v\n%s", err, out)
	}
	if _, ok := shape.(map[string]any); !ok {
		t.Fatalf("graph bridges --focus top-level JSON = %T, want object envelope: %s", shape, out)
	}
	var envelope bridgeFocusEnvelope
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("graph bridges --focus envelope: %v\n%s", err, out)
	}
	return envelope
}

// Retained property 1a: focus selects exact membership from the same complete
// graph clustering and uses the established representative rule.
func TestClustersFocusReturnsExactFullGraphCluster(t *testing.T) {
	hub := newTestNoteForCLI("20260101000000-0001", "Hub", note.TypeConcept)
	focus := newTestNoteForCLI("20260101000000-0002", "Focus", note.TypeConcept)
	context := newTestNoteForCLI("20260101000000-0003", "Context", note.TypeObservation)
	hub.Links = []note.Link{
		{TargetID: focus.ID, Type: "supports"},
		{TargetID: context.ID, Type: "extends"},
	}
	notes := []*note.Note{context, focus, hub}

	legacyOut, err := executeClustersFocusContract(notes, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var legacy []struct {
		Notes []clusterSearchTestNote `json:"notes"`
	}
	if err := json.Unmarshal([]byte(legacyOut), &legacy); err != nil || len(legacy) != 1 {
		t.Fatalf("legacy cluster fixture: err=%v output=%s", err, legacyOut)
	}

	out, err := executeClustersFocusContract(notes, "--focus", focus.ID, "--json")
	if err != nil {
		t.Fatalf("clusters --focus: %v", err)
	}
	got := decodeClusterFocusEnvelope(t, out)
	if got.Focus.ID != focus.ID || got.Focus.Title != focus.Title {
		t.Errorf("focus = %#v, want exact note %s/%q", got.Focus, focus.ID, focus.Title)
	}
	if got.Cluster == nil {
		t.Fatalf("cluster = null, want visible cluster for %s", focus.ID)
	}
	if got.Cluster.Size != 3 || !reflect.DeepEqual(got.Cluster.Notes, legacy[0].Notes) {
		t.Errorf("focus cluster membership = %#v, want exact legacy membership %#v", got.Cluster, legacy[0].Notes)
	}
	if got.Cluster.Representative.ID != hub.ID || got.Cluster.Representative.Title != hub.Title {
		t.Errorf("representative = %#v, want highest-total-degree note %s/%q", got.Cluster.Representative, hub.ID, hub.Title)
	}
}

// Retained property 1b: the active --min/--singletons visibility policy turns
// a known focus into cluster:null rather than changing or failing lookup.
func TestClustersFocusHonorsMinAndSingletonPolicy(t *testing.T) {
	a := newTestNoteForCLI("20260101000000-0011", "Pair A", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0012", "Pair B", note.TypeConcept)
	isolated := newTestNoteForCLI("20260101000000-0013", "Isolated", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Type: "extends"}}
	notes := []*note.Note{a, b, isolated}

	for _, tc := range []struct {
		name        string
		focus       *note.Note
		args        []string
		wantCluster bool
		wantSize    int
	}{
		{name: "min filters known pair", focus: a, args: []string{"--min", "3"}},
		{name: "singleton omitted by default", focus: isolated},
		{name: "singletons includes isolated focus", focus: isolated, args: []string{"--singletons"}, wantCluster: true, wantSize: 1},
		{name: "explicit min still filters singleton", focus: isolated, args: []string{"--singletons", "--min", "2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--focus", tc.focus.ID, "--json"}, tc.args...)
			out, err := executeClustersFocusContract(notes, args...)
			if err != nil {
				t.Fatalf("clusters --focus: %v", err)
			}
			got := decodeClusterFocusEnvelope(t, out)
			if got.Focus.ID != tc.focus.ID || got.Focus.Title != tc.focus.Title {
				t.Errorf("focus = %#v, want known note %s/%q", got.Focus, tc.focus.ID, tc.focus.Title)
			}
			if (got.Cluster != nil) != tc.wantCluster {
				t.Fatalf("cluster = %#v, want present=%t", got.Cluster, tc.wantCluster)
			}
			if got.Cluster != nil && got.Cluster.Size != tc.wantSize {
				t.Errorf("cluster size = %d, want %d", got.Cluster.Size, tc.wantSize)
			}
		})
	}
}

func bridgeFocusBeyondDefaultLimitFixture() ([]*note.Note, *note.Note) {
	var notes []*note.Note
	var focus *note.Note
	for i := 0; i < 11; i++ {
		prefix := "20260101000000-00" + string(rune('A'+i))
		left := newTestNoteForCLI(prefix+"1", "Left "+string(rune('A'+i)), note.TypeConcept)
		bridge := newTestNoteForCLI(prefix+"2", "Bridge "+string(rune('A'+i)), note.TypeConcept)
		right := newTestNoteForCLI(prefix+"3", "Right "+string(rune('A'+i)), note.TypeConcept)
		left.Links = []note.Link{{TargetID: bridge.ID, Type: "supports", Annotation: "inbound"}}
		bridge.Links = []note.Link{{TargetID: right.ID, Type: "extends", Annotation: "outbound"}}
		notes = append(notes, left, bridge, right)
		if i == 10 {
			focus = bridge
		}
	}
	return notes, focus
}

// Retained property 2a: focus bypasses ranking/default limit and nests the
// exact existing rich bridge record, including witnesses and null relevance.
func TestGraphBridgesFocusReturnsExistingRichRecordBeyondDefaultLimit(t *testing.T) {
	notes, focus := bridgeFocusBeyondDefaultLimitFixture()
	baselineOut, err := executeBridgesFocusContract(notes, "--format", "json", "--limit", "0")
	if err != nil {
		t.Fatal(err)
	}
	var baseline []bridgeSearchResult
	if err := json.Unmarshal([]byte(baselineOut), &baseline); err != nil {
		t.Fatal(err)
	}
	var want *bridgeSearchResult
	for i := range baseline {
		if baseline[i].ID == focus.ID {
			want = &baseline[i]
			break
		}
	}
	if want == nil {
		t.Fatalf("fixture focus bridge %s missing from unlimited baseline", focus.ID)
	}
	defaultOut, err := executeBridgesFocusContract(notes, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var defaultRecords []bridgeSearchResult
	if err := json.Unmarshal([]byte(defaultOut), &defaultRecords); err != nil {
		t.Fatal(err)
	}
	for _, record := range defaultRecords {
		if record.ID == focus.ID {
			t.Fatalf("fixture focus %s unexpectedly appears inside default limit", focus.ID)
		}
	}

	out, err := executeBridgesFocusContract(notes, "--focus", focus.ID, "--format", "json")
	if err != nil {
		t.Fatalf("graph bridges --focus: %v", err)
	}
	got := decodeBridgeFocusEnvelope(t, out)
	if got.Focus.ID != focus.ID || got.Focus.Title != focus.Title {
		t.Errorf("focus = %#v, want exact note %s/%q", got.Focus, focus.ID, focus.Title)
	}
	if got.Bridge == nil {
		t.Fatalf("bridge = null, want bridge %s beyond the default ranked limit", focus.ID)
	}
	if !reflect.DeepEqual(*got.Bridge, *want) {
		t.Errorf("focused bridge differs from existing rich record:\ngot=%#v\nwant=%#v", *got.Bridge, *want)
	}
	if got.Bridge.RelevanceScore != nil || len(got.Bridge.Witnesses) == 0 {
		t.Errorf("focused rich evidence = %#v, want null relevance and deterministic witnesses", got.Bridge)
	}
}

// Retained property 2b: a known note that does not satisfy the established
// bridge heuristic is represented by bridge:null, not a lookup error.
func TestGraphBridgesFocusReturnsNullForKnownNonBridge(t *testing.T) {
	focus := newTestNoteForCLI("20260101000000-0101", "Known leaf", note.TypeConcept)
	other := newTestNoteForCLI("20260101000000-0102", "Other", note.TypeConcept)
	focus.Links = []note.Link{{TargetID: other.ID, Type: "extends"}}

	out, err := executeBridgesFocusContract([]*note.Note{focus, other}, "--focus", focus.ID, "--format", "json")
	if err != nil {
		t.Fatalf("graph bridges --focus: %v", err)
	}
	got := decodeBridgeFocusEnvelope(t, out)
	if got.Focus.ID != focus.ID || got.Focus.Title != focus.Title {
		t.Errorf("focus = %#v, want known note %s/%q", got.Focus, focus.ID, focus.Title)
	}
	if got.Bridge != nil {
		t.Errorf("bridge = %#v, want null for known non-bridge", got.Bridge)
	}
}

// Documentation and command help route Local territory to exact focus modes.
func TestStructuralFocusModesAreDocumentedForLocalScan(t *testing.T) {
	a := newTestNoteForCLI("20260101000000-0301", "A", note.TypeConcept)
	for name, run := range map[string]func() (string, error){
		"clusters": func() (string, error) { return executeClustersFocusContract([]*note.Note{a}, "--help") },
		"bridges":  func() (string, error) { return executeBridgesFocusContract([]*note.Note{a}, "--help") },
	} {
		help, err := run()
		if err != nil {
			t.Fatalf("%s help: %v", name, err)
		}
		if !strings.Contains(help, "--focus") || !strings.Contains(help, "exact note") {
			t.Errorf("%s help missing exact --focus mode:\n%s", name, help)
		}
	}

	files := map[string][]string{
		"../../../README.md": {
			"nn clusters --focus <id> --json",
			"nn graph bridges --focus <id> --format json",
		},
		"../../../skills/nn-guide/SKILL.md": {
			"nn clusters --focus ID --json",
			"known note omitted by the active",
			"nn graph bridges --focus ID --format json",
			"known non-bridge",
			"bypasses ranking and the default limit",
			"Without `--focus`, existing",
		},
		"../../../skills/nn-navigate/references/scan-and-routes.md": {
			"nn clusters --focus <id> --json",
			"nn graph bridges --focus <id> --format json",
			"exact local structural context",
			"Global landscape",
			"--search",
		},
		"show.go": {
			"nn clusters --focus ID --json",
			"nn graph bridges --focus ID --format json",
			"known non-bridge",
			"existing non-focus",
		},
	}
	for path, required := range files {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, snippet := range required {
			if !strings.Contains(string(contents), snippet) {
				t.Errorf("%s missing %q", path, snippet)
			}
		}
	}
}

// Retained property 3: focus is additive. Unknown IDs fail, focus-only output
// requirements and incompatible ranking flags are explicit, and omitted focus
// retains the established top-level array schemas.
func TestStructuralFocusModesAreAdditive(t *testing.T) {
	a := newTestNoteForCLI("20260101000000-0201", "A", note.TypeConcept)
	b := newTestNoteForCLI("20260101000000-0202", "B", note.TypeConcept)
	c := newTestNoteForCLI("20260101000000-0203", "C", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Type: "supports"}}
	b.Links = []note.Link{{TargetID: c.ID, Type: "extends"}}
	notes := []*note.Note{a, b, c}

	clusterLegacy, err := executeClustersFocusContract(notes, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var clusterShape any
	if err := json.Unmarshal([]byte(clusterLegacy), &clusterShape); err != nil {
		t.Fatal(err)
	}
	if _, ok := clusterShape.([]any); !ok {
		t.Errorf("clusters without --focus top-level JSON = %T, want retained array", clusterShape)
	}
	bridgeLegacy, err := executeBridgesFocusContract(notes, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var bridgeShape any
	if err := json.Unmarshal([]byte(bridgeLegacy), &bridgeShape); err != nil {
		t.Fatal(err)
	}
	if _, ok := bridgeShape.([]any); !ok {
		t.Errorf("graph bridges without --focus top-level JSON = %T, want retained array", bridgeShape)
	}

	for _, tc := range []struct {
		name string
		run  func() error
		want string
	}{
		{name: "clusters unknown", run: func() error {
			_, err := executeClustersFocusContract(notes, "--focus", "missing", "--json")
			return err
		}, want: `note "missing" not found`},
		{name: "clusters blank focus", run: func() error { _, err := executeClustersFocusContract(notes, "--focus", " \t", "--json"); return err }, want: "--focus requires a non-blank ID"},
		{name: "clusters JSON required", run: func() error { _, err := executeClustersFocusContract(notes, "--focus", b.ID); return err }, want: "--focus requires --json"},
		{name: "clusters rejects search", run: func() error {
			_, err := executeClustersFocusContract(notes, "--focus", b.ID, "--search", "needle", "--json")
			return err
		}, want: "--focus cannot be combined with --search"},
		{name: "clusters rejects summary", run: func() error {
			_, err := executeClustersFocusContract(notes, "--focus", b.ID, "--summary", "--json")
			return err
		}, want: "--focus cannot be combined with --summary"},
		{name: "clusters rejects match limit", run: func() error {
			_, err := executeClustersFocusContract(notes, "--focus", b.ID, "--match-limit", "0", "--json")
			return err
		}, want: "--focus cannot be combined with --match-limit"},
		{name: "bridges unknown", run: func() error {
			_, err := executeBridgesFocusContract(notes, "--focus", "missing", "--format", "json")
			return err
		}, want: `note "missing" not found`},
		{name: "bridges blank focus", run: func() error {
			_, err := executeBridgesFocusContract(notes, "--focus", " \t", "--format", "json")
			return err
		}, want: "--focus requires a non-blank ID"},
		{name: "bridges JSON required", run: func() error { _, err := executeBridgesFocusContract(notes, "--focus", b.ID); return err }, want: "--focus requires --format json"},
		{name: "bridges rejects search", run: func() error {
			_, err := executeBridgesFocusContract(notes, "--focus", b.ID, "--search", "needle", "--format", "json")
			return err
		}, want: "--focus cannot be combined with --search"},
		{name: "bridges rejects exclude", run: func() error {
			_, err := executeBridgesFocusContract(notes, "--focus", b.ID, "--exclude", a.ID, "--format", "json")
			return err
		}, want: "--focus cannot be combined with --exclude"},
		{name: "bridges rejects explicit limit", run: func() error {
			_, err := executeBridgesFocusContract(notes, "--focus", b.ID, "--limit", "0", "--format", "json")
			return err
		}, want: "--focus cannot be combined with --limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want %q", err, tc.want)
			}
		})
	}
}
