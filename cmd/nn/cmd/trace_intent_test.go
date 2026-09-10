package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/trace"
)

func TestGrepIntentRequiresTrace(t *testing.T) {
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "subject.go")
	if err := os.WriteFile(file, []byte("package subject\nfunc Subject() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := execute("grep", "Subject", file, "--intent", "investigate tracing")
	if err == nil || !strings.Contains(err.Error(), "--intent requires --trace") {
		t.Fatalf("error = %v, want --intent requires --trace", err)
	}
}

func TestTraceIntentChangesEnrichmentNotGraph(t *testing.T) {
	_, execute := setupNotebook(t)
	if _, err := execute("new", "--title", "irrelevanttoken implementation", "--type", "observation", "--no-edit", "--no-suggest"); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("new", "--title", "ambiguous call resolution tracing", "--type", "observation", "--no-edit", "--no-suggest"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "subject.go"), []byte("package subject\nfunc Subject() { irrelevanttoken() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	legacyJSON, err := execute("trace", dir, "--symbol", "Subject", "--depth", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	intentJSON, err := execute("trace", dir, "--symbol", "Subject", "--depth", "1", "--json", "--intent", "ambiguous call resolution tracing")
	if err != nil {
		t.Fatal(err)
	}
	var legacy, withIntent trace.Result
	if err := json.Unmarshal([]byte(legacyJSON), &legacy); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(intentJSON), &withIntent); err != nil {
		t.Fatal(err)
	}
	legacyNotes := legacy.Nodes[0].NNNotes
	intentNotes := withIntent.Nodes[0].NNNotes
	for i := range legacy.Nodes {
		legacy.Nodes[i].NNNotes = nil
	}
	for i := range withIntent.Nodes {
		withIntent.Nodes[i].NNNotes = nil
	}
	withIntent.Intent = ""
	if !reflect.DeepEqual(legacy, withIntent) {
		t.Fatalf("structural graph changed with intent\nlegacy=%#v\nintent=%#v", legacy, withIntent)
	}
	if len(legacyNotes) == 0 || len(intentNotes) == 0 {
		t.Fatalf("missing note results: legacy=%#v intent=%#v", legacyNotes, intentNotes)
	}
	if intentNotes[0].Title != "ambiguous call resolution tracing" {
		t.Fatalf("intent first note = %q", intentNotes[0].Title)
	}
	if len(intentNotes[0].Provenance) == 0 {
		t.Fatal("intent result missing ranking provenance")
	}
}

func TestGrepTraceIntentKeepsNodeAnnotatorNil(t *testing.T) {
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "subject.go")
	if err := os.WriteFile(file, []byte("package subject\nfunc Subject() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := traceRun
	defer func() { traceRun = orig }()
	called := false
	traceRun = func(idx *trace.Index, symbols []string, depth int, annotate trace.Annotator) *trace.Result {
		called = true
		if annotate != nil {
			t.Fatal("grep --trace --intent passed a node annotator")
		}
		return orig(idx, symbols, depth, annotate)
	}
	if _, err := execute("grep", "Subject", file, "--trace", "--intent", "investigate tracing"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("trace runner was not called")
	}
}
