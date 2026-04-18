package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// noteGoFile is the path to internal/note/note.go relative to the test package dir.
const noteGoFile = "../../../internal/note/note.go"

func TestAstGoFile(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", noteGoFile)
	if err != nil {
		t.Fatalf("nn ast: %v", err)
	}
	if !strings.Contains(out, "language: go") {
		t.Errorf("ast output missing 'language: go':\n%s", out)
	}
	// Should show the Note type or Parse function.
	if !strings.Contains(out, "Note") && !strings.Contains(out, "Parse") {
		t.Errorf("ast output missing expected symbols:\n%s", out)
	}
}

func TestAstJSON(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("ast", "--json", noteGoFile)
	if err != nil {
		t.Fatalf("nn ast --json: %v", err)
	}
	var symbols []struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Signature string `json:"signature"`
		Line      int    `json:"line"`
	}
	if err := json.Unmarshal([]byte(out), &symbols); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(symbols) == 0 {
		t.Error("expected at least one symbol in JSON output")
	}
}

func TestAstTrace(t *testing.T) {
	_, execute := setupNotebook(t)

	// --trace is a boolean; traces every symbol found in the AST.
	out, err := execute("ast", noteGoFile, "--trace", "--root", "../../../")
	if err != nil {
		t.Fatalf("nn ast --trace: %v", err)
	}
	// Should find references to symbols defined in note.go (e.g. GenerateID, Parse, Marshal).
	if !strings.Contains(out, "GenerateID") && !strings.Contains(out, "Parse") && !strings.Contains(out, "Marshal") {
		t.Errorf("ast --trace missing expected symbol references:\n%s", out)
	}
	if !strings.Contains(out, "name-match") {
		t.Errorf("ast --trace missing name-match disclaimer:\n%s", out)
	}
	// Should include multiple symbol sections (not just one name).
	if strings.Count(out, "references to") < 2 {
		t.Errorf("ast --trace should include multiple symbol sections (one per symbol), got:\n%s", out)
	}
}

func TestAstUnknownLanguage(t *testing.T) {
	_, execute := setupNotebook(t)

	// A .xyz file won't be recognized.
	_, err := execute("ast", "nonexistent.xyz")
	if err == nil {
		t.Error("nn ast on unknown language should return error")
	}
}

func TestNewFromFile(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)

	root := NewRootCmdForTest(cfgFile)
	root.SetArgs([]string{"new", "--title", "From File Note", "--type", "concept", "--from-file", noteGoFile})
	var stdout strings.Builder
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("nn new --from-file: %v", err)
	}
	if !strings.Contains(stdout.String(), "created") {
		t.Errorf("expected 'created' in output: %q", stdout.String())
	}
}
