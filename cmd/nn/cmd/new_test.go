package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestNewNoteCreatesFile(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	_, err := execute("new", "--title", "My First Note", "--type", "concept", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	entries, _ := os.ReadDir(nbDir)
	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) != 1 {
		t.Fatalf("expected 1 .md file, got %d: %v", len(mdFiles), mdFiles)
	}
	if !strings.Contains(mdFiles[0], "my-first-note") {
		t.Errorf("filename %q does not contain slug 'my-first-note'", mdFiles[0])
	}
}

func TestNewNoteOutputsID(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("new", "--title", "Output Test", "--type", "argument", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	if !strings.Contains(out, "created") {
		t.Errorf("output %q does not contain 'created'", out)
	}
}

func TestNewNoteWithContent(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	_, err := execute("new", "--title", "Content Note", "--type", "model",
		"--content", "This is the body text.", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	entries, _ := os.ReadDir(nbDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			data, _ := os.ReadFile(filepath.Join(nbDir, e.Name()))
			if !strings.Contains(string(data), "This is the body text.") {
				t.Errorf("file does not contain body text")
			}
		}
	}
}

func TestNewNoteWithTags(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	_, err := execute("new", "--title", "Tagged Note", "--type", "concept",
		"--tags", "alpha,beta", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	entries, _ := os.ReadDir(nbDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			data, _ := os.ReadFile(filepath.Join(nbDir, e.Name()))
			content := string(data)
			if !strings.Contains(content, "alpha") || !strings.Contains(content, "beta") {
				t.Errorf("file does not contain tags: %s", content)
			}
		}
	}
}

func TestNewNoteRequiresType(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("new", "--title", "No Type", "--no-edit")
	if err == nil {
		t.Fatal("nn new without --type: want error, got nil")
	}
}

func TestNewNoteInvalidType(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("new", "--title", "Bad Type", "--type", "invalid", "--no-edit")
	if err == nil {
		t.Fatal("nn new --type invalid: want error, got nil")
	}
}

func TestNewMultipleLinkTo(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "Note A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Note B", note.TypeConcept)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	_, err := execute("new", "--title", "Linked Note", "--type", "argument", "--no-edit",
		"--link-to", a.ID, "--link-type", "grounded-by", "--annotation", "first link",
		"--link-to", b.ID, "--link-type", "extends", "--annotation", "second link")
	if err != nil {
		t.Fatalf("nn new with multiple --link-to: %v", err)
	}

	out, err := execute("list", "--search", "Linked Note", "--json", "--fields", "id,title")
	if err != nil {
		t.Fatalf("nn list: %v", err)
	}
	// Find the created note ID from output then check its links via show.
	entries, _ := os.ReadDir(nbDir)
	var newFile string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && strings.Contains(e.Name(), "linked-note") {
			newFile = filepath.Join(nbDir, e.Name())
		}
	}
	_ = out
	if newFile == "" {
		t.Fatal("linked-note file not found")
	}
	data, _ := os.ReadFile(newFile)
	content := string(data)
	if !strings.Contains(content, a.ID) {
		t.Errorf("note body does not contain first link target %s:\n%s", a.ID, content)
	}
	if !strings.Contains(content, b.ID) {
		t.Errorf("note body does not contain second link target %s:\n%s", b.ID, content)
	}
	if !strings.Contains(content, "[grounded-by]") || !strings.Contains(content, "[extends]") {
		t.Errorf("note body does not contain paired link types:\n%s", content)
	}
}

func TestNewLinkToRequiresPairedKnownLinkType(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing", []string{"--link-to", "someid", "--annotation", "context"}},
		{"unknown", []string{"--link-to", "someid", "--link-type", "invented", "--annotation", "context"}},
		{"mismatched", []string{"--link-to", "a", "--link-type", "supports", "--annotation", "a", "--link-to", "b", "--annotation", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"new", "--title", "Bad Links", "--type", "concept", "--no-edit"}
			_, err := execute(append(args, tc.args...)...)
			if err == nil {
				t.Fatalf("nn new with %s link type: want error, got nil", tc.name)
			}
		})
	}
}

func TestNewLinkToMismatchedAnnotation(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("new", "--title", "Bad Links", "--type", "concept", "--no-edit",
		"--link-to", "someid", "--link-to", "otherid", "--annotation", "only one annotation")
	if err == nil {
		t.Fatal("nn new with mismatched --link-to/--annotation: want error, got nil")
	}
}

func TestNewCheckFlag(t *testing.T) {
	_, execute := setupNotebook(t)
	// A note with no representation: --check should be a no-op (no error).
	_, err := execute("new", "--title", "No Rep Note", "--type", "concept", "--no-edit", "--check")
	if err != nil {
		t.Fatalf("nn new --check with no representation: %v", err)
	}
}

func TestNewQuickFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	_, err := execute("new", "--quick", "--title", "Quick Note", "--no-edit")
	if err != nil {
		t.Fatalf("nn new --quick: %v", err)
	}
	entries, _ := os.ReadDir(nbDir)
	var noteFile string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			noteFile = filepath.Join(nbDir, e.Name())
		}
	}
	if noteFile == "" {
		t.Fatal("no .md file created")
	}
	data, err := os.ReadFile(noteFile)
	if err != nil {
		t.Fatalf("read note file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "type: observation") {
		t.Errorf("--quick did not set type=observation; got:\n%s", content)
	}
	if !strings.Contains(content, "status: draft") {
		t.Errorf("--quick did not set status=draft; got:\n%s", content)
	}
}
