package cmd

import (
	"strings"
	"testing"
)

// Publication tripwires, not a semantic equivalence or model-compliance test.
// Exercise the embedded, CLI-served artifacts rather than only checkout files.
func TestTranscriptDocumentationOwners(t *testing.T) {
	_, execute := setupNotebook(t)
	core, err := execute("skills", "get", "nn-transcript")
	if err != nil {
		t.Fatal(err)
	}
	listing, err := execute("skills", "get", "nn-transcript", "--list-references")
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"search", "recovery"} {
		t.Run("route_"+owner, func(t *testing.T) {
			a := "DOC_ROUTE_" + owner
			t.Log(a + " PROCEDURE: core dispatch plus applicability listing plus served owner")
			_, err := execute("skills", "get", "nn-transcript", "--reference", owner)
			if !strings.Contains(core, "--reference "+owner) || !strings.Contains(listing, owner+"\t") || err != nil {
				t.Fatalf("%s FAIL: owner not reachable through core/listing/serve: %v", a, err)
			}
			t.Log(a + " PASS")
		})
	}
	for _, tc := range []struct {
		name, owner         string
		required, forbidden []string
	}{
		{"DOC_SEARCH_CONTENT", "search", []string{"--regex", "(?i)", "Go syntax", "case-insensitive literal", "--raw", "--limit", "canonical path", "skipped_files", "--session", "--reference patterns"}, nil},
		{"DOC_RECOVERY_CONTENT", "recovery", []string{"nn transcript doctor", "exactly one parent", "DAG, no cycles", "at or after its parent's start", "equals the count of spawn tool-calls", "Only emit the relation after all four pass"}, nil},
		{"DOC_PATTERNS_OWNERS", "patterns", []string{"--reference search", "--reference recovery", "--reference actions"}, []string{"--regex", "each spawn timestamp is at or after"}},
		{"DOC_SAMPLING_BOUNDARY", "patterns", []string{"Sampling is not retrieval coverage.", "uninspected", "complete", "claim"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.name + " PROCEDURE: served owner, whitespace-normalized required terms and excluded duplicate rules")
			text, err := execute("skills", "get", "nn-transcript", "--reference", tc.owner)
			if err != nil {
				t.Fatalf("%s FAIL: owner unavailable: %v", tc.name, err)
			}
			text = strings.Join(strings.Fields(text), " ")
			for _, s := range tc.required {
				if !strings.Contains(text, s) {
					t.Fatalf("%s FAIL: missing %q", tc.name, s)
				}
			}
			for _, s := range tc.forbidden {
				if strings.Contains(text, s) {
					t.Fatalf("%s FAIL: duplicated rule %q", tc.name, s)
				}
			}
			t.Log(tc.name + " PASS")
		})
	}
}
