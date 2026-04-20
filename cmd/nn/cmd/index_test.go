package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// Assertion: TestIndexCommandExists — command is registered and runs without error.
func TestIndexCommandExists(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("index", "zettelkasten")
	if err != nil {
		t.Fatalf("index command failed: %v", err)
	}
}

// Assertion: TestIndexTopicSection — output contains topic notes matching the query.
func TestIndexTopicSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	match := writeSuggestNote(t, nbDir, "20240101000001", "Zettelkasten Overview", "zettelkasten is a method for linking notes")
	writeSuggestNote(t, nbDir, "20240101000002", "Cooking Recipes", "pasta carbonara recipe")

	out, err := execute("index", "zettelkasten")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if !strings.Contains(out, "## Topic") {
		t.Errorf("expected '## Topic' section; got:\n%s", out)
	}
	if !strings.Contains(out, match.ID) {
		t.Errorf("expected matching note %q in topic section; got:\n%s", match.ID, out)
	}
}

// Assertion: TestIndexClusterSection — output contains a clusters/groupings block.
func TestIndexClusterSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeSuggestNote(t, nbDir, "20240101000001", "Zettelkasten Overview", "zettelkasten is a method for linking notes atomically")
	writeSuggestNote(t, nbDir, "20240101000002", "Zettelkasten Principles", "zettelkasten atomic permanent notes linking")
	writeSuggestNote(t, nbDir, "20240101000003", "Zettelkasten Practice", "zettelkasten daily writing habits review")

	out, err := execute("index", "zettelkasten")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if !strings.Contains(out, "## Clusters") {
		t.Errorf("expected '## Clusters' section; got:\n%s", out)
	}
}

// Assertion: TestIndexFormatJSON — --format json produces valid JSON with required keys.
func TestIndexFormatJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeSuggestNote(t, nbDir, "20240101000001", "Zettelkasten Overview", "zettelkasten is a method for linking notes")

	out, err := execute("index", "zettelkasten", "--format", "json")
	if err != nil {
		t.Fatalf("index --format json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("expected valid JSON; got:\n%s\nerr: %v", out, err)
	}
	for _, key := range []string{"topic_notes", "clusters"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected %q key in JSON; got keys: %v", key, jsonKeys(result))
		}
	}
}
