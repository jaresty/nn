package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// Native capability test: the proposed ROOT + bounded direct-child recipe must
// execute on existing adapters. It does not evaluate LLM selection or explanation.
func TestTranscriptTracerNativeRecipe(t *testing.T) {
	for _, schema := range []string{"pi", "sdk", "inline"} {
		t.Run(schema, func(t *testing.T) {
			_, execute := setupNotebook(t)
			var path string
			switch schema {
			case "pi":
				path = reviewFixture(t)
			case "sdk":
				path = writeSDKCLIFixture(t, t.TempDir())
			default:
				path = claudeCodeFixture(t)
			}
			body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
			if err != nil {
				t.Fatal(err)
			}
			var commands []string
			for _, line := range strings.Split(body, "\n") {
				if strings.HasPrefix(line, "nn transcript ") {
					commands = append(commands, line)
				}
			}
			if len(commands) != 2 {
				t.Fatal("TRACER_RECIPE FAIL: expected discovery and observation commands")
			}
			run := func(index int, child string) string {
				t.Helper()
				// Split the published argument template before substituting paths, so
				// fixture directories with spaces remain single operands. No shell.
				args := strings.Fields(commands[index])[1:]
				if index == 0 {
					args = append(args, filepath.Dir(path))
				}
				for i, arg := range args {
					switch arg {
					case "<session-id>":
						args[i] = path
					case "<child-id>":
						args[i] = child
					}
				}
				out, e := execute(args...)
				if e != nil {
					t.Fatalf("TRACER_RECIPE FAIL: %s: %v", commands[index], e)
				}
				if strings.TrimSpace(out) == "" {
					t.Fatal("TRACER_RECIPE FAIL: empty receipt")
				}
				return out
			}
			var rows []sessionRow
			if err = json.Unmarshal([]byte(run(0, "")), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) == 0 || len(rows) > 5 {
				t.Fatal("TRACER_RECIPE FAIL: discovery selection")
			}
			found := false
			for _, row := range rows {
				if row.Path == path {
					found = true
					path = row.Path
					break
				}
			}
			if !found {
				t.Fatal("TRACER_RECIPE FAIL: fixture canonical path not discovered")
			}
			var page treeChildPage
			tree, err := execute("transcript", "tree", path, "--parent", "ROOT", "--limit", "2", "--json")
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal([]byte(tree), &page); err != nil {
				t.Fatal(err)
			}
			observed := run(1, "")
			if strings.Count(observed, "\n## ") != len(page.Children)+1 || !strings.Contains(observed, "## ROOT —") {
				t.Fatal("TRACER_RECIPE FAIL: wrong stream sample")
			}
			for _, child := range page.Children {
				if !strings.Contains(observed, "## "+child.ID+" —") {
					t.Fatal("TRACER_RECIPE FAIL: selected child omitted")
				}
			}
			t.Log("TRACER_RECIPE PASS: served discovery, ROOT and bounded direct children execute; no liveness or completeness inference")
		})
	}
}

// These are instruction publication guards, not semantic compliance tests.
func TestTranscriptTracerPublication(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := []struct{ name, owner, clause string }{
		{"DIRECT", "interaction", "Ordinary bounded read-only requests execute before optional choices."},
		{"SCOPE", "interaction", "Consulting evidence outside observation scope does not itself broaden that scope."},
		{"BACK", "interaction", "Back restores retained observations without new acquisition."},
		{"ROOT", "observe", "Always include ROOT in the initial selected-conversation sample."},
		{"COVERAGE", "observe", "The candidate page is not the entire scope."},
		{"LEARNING", "actions", "A snapshot identifier alone is not a durable learning."},
		{"CAPTURE", "actions", "Require explicit approval of the concrete proposal before creating/updating a note or link."},
		{"QUESTIONS", "investigate", "Let the question choose the evidence, not a mandatory tree descent."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("TRACER_%s PROCEDURE: fetch served %s reference and check its declared instruction", tc.name, tc.owner)
			body, err := execute("skills", "get", "nn-transcript", "--reference", tc.owner)
			if err != nil || !strings.Contains(strings.Join(strings.Fields(body), " "), tc.clause) {
				t.Fatalf("TRACER_%s FAIL: missing instruction %q: %v", tc.name, tc.clause, err)
			}
			t.Logf("TRACER_%s PASS", tc.name)
		})
	}
}
