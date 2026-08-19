package feedback

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFeedbackRequestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	q := FeedbackRequest{
		ID:           "q1",
		Surface:      "canvas",
		Mode:         "create",
		Instructions: "draw the thing",
		Context:      []string{"note-a", "file-b.png"},
		Workspace:    "workspace.excalidraw",
		Output:       "result.excalidraw",
	}
	if err := WriteRequest(dir, q); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	got, err := ReadRequest(dir)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if !reflect.DeepEqual(got, q) {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, q)
	}
}

func TestFeedbackResultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r := FeedbackResult{
		ID:      "r1",
		Surface: "canvas",
		Status:  "submitted",
		Artifacts: []Artifact{
			{Format: "excalidraw", Path: "result.excalidraw"},
			{Format: "png", Path: "result.png"},
		},
	}
	if err := WriteResult(dir, r); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	got, err := ReadResult(dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	if !reflect.DeepEqual(got, r) {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, r)
	}
}

func TestFeedbackResultEnvelopeShape(t *testing.T) {
	r := FeedbackResult{
		Surface:   "canvas",
		Status:    "submitted",
		Artifacts: []Artifact{{Format: "png", Path: "result.png"}},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"surface", "status", "artifacts"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("envelope missing key %q in %s", k, b)
		}
	}
	var arts []map[string]json.RawMessage
	if err := json.Unmarshal(m["artifacts"], &arts); err != nil {
		t.Fatalf("artifacts unmarshal: %v", err)
	}
	for _, a := range arts {
		if _, ok := a["format"]; !ok {
			t.Fatalf("artifact missing key %q in %s", "format", b)
		}
		if _, ok := a["path"]; !ok {
			t.Fatalf("artifact missing key %q in %s", "path", b)
		}
	}
}

func TestSessionDirUsesNNConfigDir(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", "/tmp/nncfg")
	got := SessionDir("abc123")
	want := filepath.Join("/tmp/nncfg", "feedback", "abc123")
	if got != want {
		t.Fatalf("SessionDir = %q, want %q", got, want)
	}
}

func TestSessionDirFallsBackToXDG(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	got := SessionDir("id1")
	want := filepath.Join("/tmp/xdg", "nn", "feedback", "id1")
	if got != want {
		t.Fatalf("SessionDir = %q, want %q", got, want)
	}
}
