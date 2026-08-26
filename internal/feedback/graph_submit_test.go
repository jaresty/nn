package feedback

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var canonicalGraphSelectionKeys = map[string]bool{
	"groups":          true,
	"overall_comment": true,
	"handoff":         true,
}

func assertCanonicalGraphSelectionShape(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("graph-selection is not valid JSON: %v\n%s", err, body)
	}
	for _, key := range []string{"selected", "annotations", "edges", "answer", "handoffs", "explain_on_canvas"} {
		if _, exists := object[key]; exists {
			t.Errorf("obsolete top-level graph-selection key %q is present", key)
		}
	}
	for key := range canonicalGraphSelectionKeys {
		if _, exists := object[key]; !exists {
			t.Errorf("canonical top-level graph-selection key %q is absent", key)
		}
	}
	for key := range object {
		if !canonicalGraphSelectionKeys[key] {
			t.Errorf("unexpected top-level graph-selection key %q is present", key)
		}
	}
	return object
}

// retained property [19]: Graph Ask has one canonical singular-handoff
// schema, with no compatibility keys at top level.
func TestGraphSelectionJSONUsesOnlyCanonicalTopLevelKeys(t *testing.T) {
	selection, err := DecodeGraphSelection([]byte(`{
		"groups":[{"id":"group-1","nodes":[{"id":"ego","selection":"explicit"}],"edges":[],"comment":"core claim"}],
		"overall_comment":"overall answer",
		"handoff":"canvas"
	}`))
	if err != nil {
		t.Fatalf("DecodeGraphSelection: %v", err)
	}
	body, err := json.Marshal(selection)
	if err != nil {
		t.Fatalf("Marshal GraphSelection: %v", err)
	}
	assertCanonicalGraphSelectionShape(t, body)
}

func TestDecodeGraphSelectionRejectsObsoleteTopLevelKeys(t *testing.T) {
	obsolete := map[string]string{
		"selected":          `[]`,
		"annotations":       `{}`,
		"edges":             `[]`,
		"answer":            `"old answer"`,
		"handoffs":          `[]`,
		"explain_on_canvas": `false`,
	}
	for key, value := range obsolete {
		t.Run(key, func(t *testing.T) {
			body := []byte(`{"groups":[],"overall_comment":"","handoff":null,"` + key + `":` + value + `}`)
			_, err := DecodeGraphSelection(body)
			if err == nil {
				t.Fatalf("obsolete top-level graph-selection key %q was accepted", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Fatalf("decoder error %q does not identify obsolete key %q", err, key)
			}
		})
	}
}

// retained property [20]: handoff accepts exactly null, canvas, or document.
func TestDecodeGraphSelectionAcceptsOnlyCanonicalHandoffValues(t *testing.T) {
	accepted := map[string]*string{
		"null":     nil,
		"canvas":   ptr("canvas"),
		"document": ptr("document"),
	}
	for value, want := range accepted {
		t.Run("accept-"+value, func(t *testing.T) {
			literal := value
			if value != "null" {
				literal = `"` + value + `"`
			}
			selection, err := DecodeGraphSelection([]byte(`{"groups":[],"overall_comment":"","handoff":` + literal + `}`))
			if err != nil {
				t.Fatalf("canonical handoff %s rejected: %v", value, err)
			}
			if (selection.Handoff == nil) != (want == nil) || selection.Handoff != nil && string(*selection.Handoff) != *want {
				t.Fatalf("handoff = %v, want %v", selection.Handoff, want)
			}
		})
	}

	rejected := map[string]string{
		"unknown":    `"slides"`,
		"wrong-case": `"Canvas"`,
		"array":      `["canvas"]`,
		"boolean":    `false`,
		"number":     `1`,
		"object":     `{}`,
	}
	for name, value := range rejected {
		t.Run("reject-"+name, func(t *testing.T) {
			body := []byte(`{"groups":[],"overall_comment":"","handoff":` + value + `}`)
			if _, err := DecodeGraphSelection(body); err == nil {
				t.Fatalf("invalid handoff accepted: %s", body)
			}
		})
	}
}

func ptr(value string) *string { return &value }

func TestDecodeGraphSelectionRejectsMissingCanonicalTopLevelKeys(t *testing.T) {
	tests := map[string]string{
		"groups":          `{"overall_comment":"","handoff":null}`,
		"overall_comment": `{"groups":[],"handoff":null}`,
		"handoff":         `{"groups":[],"overall_comment":""}`,
	}
	for missing, body := range tests {
		t.Run(missing, func(t *testing.T) {
			_, err := DecodeGraphSelection([]byte(body))
			if err == nil {
				t.Fatalf("missing canonical key %q was accepted", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Fatalf("decoder error %q does not identify missing key %q", err, missing)
			}
		})
	}
}

func TestDecodeGraphSelectionPreservesAnnotatedGroupMembership(t *testing.T) {
	body := []byte(`{
		"groups":[{
			"id":"group-1",
			"name":"Core argument",
			"classification":"belong-together",
			"nodes":[{"id":"ego","selection":"explicit","comment":"core claim"},{"id":"nbr","selection":"explicit"}],
			"edges":[{"source":"ego","target":"nbr","type":"supports","selection":"implicit","comment":"supporting relation"}],
			"comment":"Review these as one argument."
		}],
		"overall_comment":"overall answer",
		"handoff":"document"
	}`)

	selection, err := DecodeGraphSelection(body)
	if err != nil {
		t.Fatalf("DecodeGraphSelection: %v", err)
	}
	if len(selection.Groups) != 1 || selection.Groups[0].Name != "Core argument" || selection.Groups[0].Comment != "Review these as one argument." {
		t.Fatalf("annotated groups were not decoded: %+v", selection.Groups)
	}
	if selection.Groups[0].Nodes[0].Selection != SelectionExplicit || selection.Groups[0].Edges[0].Selection != SelectionImplicit {
		t.Fatalf("selection kinds were not distinguished: %+v", selection.Groups[0])
	}
	if selection.Groups[0].Edges[0].Comment != "supporting relation" {
		t.Fatalf("edge comment was not decoded: %+v", selection.Groups[0])
	}
	if selection.OverallComment != "overall answer" {
		t.Fatalf("overall comment was not decoded: %+v", selection)
	}
	canonical, err := json.Marshal(selection)
	if err != nil {
		t.Fatalf("Marshal GraphSelection: %v", err)
	}
	var object struct {
		Handoff *string `json:"handoff"`
	}
	if err := json.Unmarshal(canonical, &object); err != nil {
		t.Fatalf("decode canonical handoff: %v", err)
	}
	if object.Handoff == nil || *object.Handoff != "document" {
		t.Fatalf("handoff = %v, want document", object.Handoff)
	}
}

// property [4a]+[4b]: submitting the graph surface materializes a graph-native
// canonical result artifact and the thin envelope names that artifact by path.
func TestGraphSubmitWritesGraphSelectionArtifact(t *testing.T) {
	q := FeedbackRequest{ID: "s1", Surface: "graph", Focus: "ego", AllowedNodes: []string{"ego", "nbr"}}
	dir := t.TempDir()
	if err := WriteRequest(dir, q); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	srv, err := NewServer(q.ID, dir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	base := "http://" + srv.Addr()

	draft := []byte(`{"groups":[{"id":"group-1","nodes":[{"id":"ego","selection":"explicit","comment":"core claim"},{"id":"nbr","selection":"explicit","comment":"supports it"}],"edges":[],"comment":"review together"}],"overall_comment":"nbr is the strongest support","handoff":"document"}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s1/draft", bytes.NewReader(draft))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT draft: %v", err)
	}

	go func() { srv.Wait() }()
	resp, err := http.Post(base+"/session/s1/submit", "application/json", nil)
	if err != nil {
		t.Fatalf("POST submit: %v", err)
	}
	resp.Body.Close()

	result, err := ReadResult(dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}

	// property [4a]: envelope names a graph-selection artifact.
	var art *Artifact
	for i := range result.Artifacts {
		if result.Artifacts[i].Format == "graph-selection" {
			art = &result.Artifacts[i]
		}
	}
	if art == nil {
		t.Fatalf("no graph-selection artifact in envelope: %+v", result.Artifacts)
	}

	// property [4b]: the artifact has exactly the canonical top-level shape.
	body, err := os.ReadFile(filepath.Join(dir, art.Path))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	assertCanonicalGraphSelectionShape(t, body)
	payload, err := DecodeGraphSelection(body)
	if err != nil {
		t.Fatalf("decode graph-selection artifact: %v\n%s", err, body)
	}
	if len(payload.Groups) != 1 || len(payload.Groups[0].Nodes) != 2 {
		t.Fatalf("group membership not carried: %+v", payload.Groups)
	}
	if payload.Groups[0].Nodes[0].Comment != "core claim" {
		t.Fatalf("node comment not carried: %+v", payload.Groups[0].Nodes)
	}
	if payload.OverallComment != "nbr is the strongest support" {
		t.Fatalf("overall_comment not carried: %q", payload.OverallComment)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("document intent should remain in graph-selection without a seed artifact: %+v", result.Artifacts)
	}
	var object struct {
		Handoff *string `json:"handoff"`
	}
	if err := json.Unmarshal(body, &object); err != nil || object.Handoff == nil || *object.Handoff != "document" {
		t.Fatalf("document handoff intent not carried: handoff=%v err=%v", object.Handoff, err)
	}
}

func TestGraphSubmitMaterializesNonStoredCanvasSeedForAnnotatedGroups(t *testing.T) {
	q := FeedbackRequest{ID: "s2", Surface: "graph", Focus: "ego", AllowedNodes: []string{"ego", "nbr"}}
	dir := t.TempDir()
	if err := WriteRequest(dir, q); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	srv, err := NewServer(q.ID, dir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	base := "http://" + srv.Addr()

	draft := []byte(`{
		"groups":[{
			"id":"group-1",
			"name":"Core argument",
			"classification":"belong-together",
			"nodes":[{"id":"ego","selection":"explicit","comment":"existing note comment"},{"id":"nbr","selection":"explicit"}],
			"edges":[{"source":"ego","target":"nbr","type":"supports","selection":"implicit","comment":"existing edge comment"}],
			"comment":"Explain this structure."
		}],
		"overall_comment":"overall answer",
		"handoff":"canvas"
	}`)
	putReq, _ := http.NewRequest(http.MethodPut, base+"/session/s2/draft", bytes.NewReader(draft))
	if _, err := http.DefaultClient.Do(putReq); err != nil {
		t.Fatalf("PUT draft: %v", err)
	}

	go func() { srv.Wait() }()
	resp, err := http.Post(base+"/session/s2/submit", "application/json", nil)
	if err != nil {
		t.Fatalf("POST submit: %v", err)
	}
	resp.Body.Close()

	result, err := ReadResult(dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	var graphArtifact, canvasArtifact *Artifact
	for i := range result.Artifacts {
		switch result.Artifacts[i].Format {
		case "graph-selection":
			graphArtifact = &result.Artifacts[i]
		case "canvas-seed":
			canvasArtifact = &result.Artifacts[i]
		}
	}
	if graphArtifact == nil || canvasArtifact == nil {
		t.Fatalf("artifacts = %+v, want graph-selection and canvas-seed", result.Artifacts)
	}

	graphBody, err := os.ReadFile(filepath.Join(dir, graphArtifact.Path))
	if err != nil {
		t.Fatalf("read graph artifact: %v", err)
	}
	assertCanonicalGraphSelectionShape(t, graphBody)
	selection, err := DecodeGraphSelection(graphBody)
	if err != nil {
		t.Fatalf("decode graph artifact: %v", err)
	}
	if selection.Groups[0].Nodes[0].Comment != "existing note comment" ||
		selection.Groups[0].Edges[0].Comment != "existing edge comment" {
		t.Fatalf("member comments were not preserved in canonical groups: %+v", selection.Groups[0])
	}

	seedBody, err := os.ReadFile(filepath.Join(dir, canvasArtifact.Path))
	if err != nil {
		t.Fatalf("read canvas seed: %v", err)
	}
	var seed CanvasSeed
	if err := json.Unmarshal(seedBody, &seed); err != nil {
		t.Fatalf("decode canvas seed: %v\n%s", err, seedBody)
	}
	if seed.Storage != "NON_STORED" || len(seed.Groups) != 1 || seed.Groups[0].Name != "Core argument" {
		t.Fatalf("canvas seed does not preserve and label derived structure: %+v", seed)
	}
}
