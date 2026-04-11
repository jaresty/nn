package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
