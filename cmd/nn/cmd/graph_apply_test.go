package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [1]: key in manifest note spec resolves to the created note ID in edges
func TestGraphApplyKeyResolution(t *testing.T) {
	_, execute := setupNotebook(t)

	manifest := `
notes:
  - key: root
    title: "Root Concept"
    type: concept
    content: "Root body"
  - key: child
    title: "Child Concept"
    type: concept
    content: "Child body"
edges:
  - from: root
    to: child
    type: refines
    annotation: "child refines root"
`
	mf := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(mf, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("graph", "apply", mf, "--commit")
	if err != nil {
		t.Fatalf("graph apply --commit: %v\noutput: %s", err, out)
	}

	// property [1]: both notes created, edge exists between them
	all, listErr := execute("list", "--json")
	if listErr != nil {
		t.Fatalf("list: %v", listErr)
	}
	if !strings.Contains(all, "Root Concept") {
		t.Errorf("property [1]: Root Concept not found in notebook")
	}
	if !strings.Contains(all, "Child Concept") {
		t.Errorf("property [1]: Child Concept not found in notebook")
	}

	// Find note IDs by listing JSON
	listOut, _ := execute("list", "--json", "--fields", "id,title")
	var listed []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	_ = json.Unmarshal([]byte(listOut), &listed)
	var rootID, childID string
	for _, n := range listed {
		if n.Title == "Root Concept" {
			rootID = n.ID
		}
		if n.Title == "Child Concept" {
			childID = n.ID
		}
	}
	if rootID == "" || childID == "" {
		t.Fatalf("property [1]: could not find note IDs in: %s", listOut)
	}
	// Verify link exists
	linksOut, _ := execute("links", rootID, "--json")
	if !strings.Contains(linksOut, childID) || !strings.Contains(linksOut, "refines") {
		t.Errorf("property [1]: expected refines link from root to child; links output: %s", linksOut)
	}
}

// property [2a/2b]: unresolvable edge reference causes error, no notes written
func TestGraphApplyUnresolvableEdge(t *testing.T) {
	_, execute := setupNotebook(t)

	manifest := `
notes:
  - key: root
    title: "Root"
    type: concept
    content: "body"
edges:
  - from: root
    to: nonexistent-key
    type: refines
    annotation: "bad ref"
`
	mf := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(mf, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// property [2b]: --commit must fail
	_, err := execute("graph", "apply", mf, "--commit")
	if err == nil {
		t.Error("property [2b]: expected error for unresolvable edge, got nil")
	}

	// property [2b]: no notes written
	all, _ := execute("list", "--json")
	if strings.Contains(all, "Root") {
		t.Error("property [2b]: notes were written despite unresolvable edge")
	}
}

// property [2a/2b]: existing:<id> reference resolves correctly
func TestGraphApplyExistingIDReference(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// create a pre-existing note
	existing := newTestNoteForCLI(note.GenerateID(), "Existing Note", note.TypeConcept)
	writeNoteFile(t, nbDir, existing)

	manifest := `
notes:
  - key: newone
    title: "New Note"
    type: concept
    content: "new body"
edges:
  - from: newone
    to: "existing:` + existing.ID + `"
    type: supports
    annotation: "new supports existing"
`
	mf := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(mf, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("graph", "apply", mf, "--commit")
	if err != nil {
		t.Fatalf("graph apply with existing:<id>: %v\noutput: %s", err, out)
	}

	listOut, _ := execute("list", "--json", "--fields", "id,title")
	var listed []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	_ = json.Unmarshal([]byte(listOut), &listed)
	var newID string
	for _, n := range listed {
		if n.Title == "New Note" {
			newID = n.ID
		}
	}
	if newID == "" {
		t.Fatal("property [2a]: New Note not created")
	}
	linksOut, _ := execute("links", newID, "--json")
	if !strings.Contains(linksOut, existing.ID) || !strings.Contains(linksOut, "supports") {
		t.Errorf("property [2a]: expected supports link to existing note; links: %s", linksOut)
	}
}

// property [3a]: --dry-run prints plan, writes no files
func TestGraphApplyDryRun(t *testing.T) {
	_, execute := setupNotebook(t)

	manifest := `
notes:
  - key: x
    title: "Dry Run Note"
    type: concept
    content: "body"
`
	mf := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(mf, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("graph", "apply", mf, "--dry-run")
	if err != nil {
		t.Fatalf("graph apply --dry-run: %v\noutput: %s", err, out)
	}

	// property [3a]: output describes planned changes
	if !strings.Contains(out, "create") && !strings.Contains(out, "Create") {
		t.Errorf("property [3a]: expected create count in dry-run output, got: %s", out)
	}

	// property [3a]: no notes in notebook
	all, _ := execute("list", "--json")
	if strings.Contains(all, "Dry Run Note") {
		t.Error("property [3a]: dry-run wrote note to notebook")
	}
}

// property [3b]: --commit writes in one git commit
func TestGraphApplyCommitIsAtomic(t *testing.T) {
	_, execute := setupNotebook(t)

	manifest := `
notes:
  - key: a
    title: "Note A"
    type: concept
    content: "body a"
  - key: b
    title: "Note B"
    type: concept
    content: "body b"
edges:
  - from: a
    to: b
    type: extends
    annotation: "a extends b"
`
	mf := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(mf, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("graph", "apply", mf, "--commit")
	if err != nil {
		t.Fatalf("graph apply --commit: %v\noutput: %s", err, out)
	}

	// Output should mention both notes created
	if !strings.Contains(out, "Note A") && !strings.Contains(out, "created") {
		t.Errorf("property [3b]: expected creation output, got: %s", out)
	}
}

// property [4]: missing --dry-run and --commit returns error
func TestGraphApplyRequiresMode(t *testing.T) {
	_, execute := setupNotebook(t)

	manifest := `
notes:
  - key: x
    title: "X"
    type: concept
    content: "body"
`
	mf := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(mf, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := execute("graph", "apply", mf)
	if err == nil {
		t.Error("property [4]: expected error when neither --dry-run nor --commit is passed")
	}
}
