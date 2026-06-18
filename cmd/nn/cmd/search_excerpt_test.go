package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestSearchExcerptPlain(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept)
	n.Body = "The quick brown fox jumps over the lazy dog near the riverbank."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "riverbank")
	if err != nil {
		t.Fatalf("nn list --search: %v", err)
	}
	if !strings.Contains(out, "riverbank") {
		t.Errorf("plain search output missing excerpt containing matched term: %q", out)
	}
	// Excerpt should appear on a second line indented under the result
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (result + excerpt), got %d: %q", len(lines), out)
	}
	excerptLine := lines[1]
	if !strings.Contains(excerptLine, "riverbank") {
		t.Errorf("second line does not contain matched term: %q", excerptLine)
	}
}

func TestExtractExcerptHeadingBreadcrumb(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		query    string
		wantPfx  string
	}{
		{
			name:    "match under single heading",
			body:    "## Section\n\nsome matched content here",
			query:   "matched",
			wantPfx: "## Section",
		},
		{
			name:    "match under nested headings",
			body:    "## Parent\n\n### Child\n\nsome matched content here",
			query:   "matched",
			wantPfx: "## Parent > ### Child",
		},
		{
			name:    "match before any heading",
			body:    "some matched content here\n\n## Section\n\nother stuff",
			query:   "matched",
			wantPfx: "",
		},
		{
			name:    "match under heading no subheading",
			body:    "## Only\n\nsome matched content here",
			query:   "matched",
			wantPfx: "## Only",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractExcerpt(tc.body, tc.query)
			if tc.wantPfx == "" {
				if strings.HasPrefix(got, "#") {
					t.Errorf("expected no heading prefix, got %q", got)
				}
			} else {
				if !strings.HasPrefix(got, tc.wantPfx+" | ") {
					t.Errorf("expected prefix %q, got %q", tc.wantPfx+" | ", got)
				}
			}
		})
	}
}

func TestSearchExcerptJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Some Note", note.TypeConcept)
	n.Body = "The quick brown fox jumps over the lazy dog near the riverbank."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "riverbank", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	excerpt, ok := results[0]["excerpt"].(string)
	if !ok {
		t.Errorf("'excerpt' field missing or not a string in JSON result: %v", results[0])
	}
	if !strings.Contains(excerpt, "riverbank") {
		t.Errorf("excerpt does not contain matched term: %q", excerpt)
	}
}
