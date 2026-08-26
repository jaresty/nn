package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestZoneOf pins the link-type -> screen-zone mapping for the Zoned ego layout.
// See design note 20260820154156-6576.
func TestZoneOf(t *testing.T) {
	cases := []struct {
		name     string
		linkType string
		dir      linkDir
		want     zone
	}{
		// TOP — what the ego answers to / depends on / is governed by
		{"governs into ego", "governs", dirIn, zoneTop}, // a note that governs the ego -> ego answers to it
		{"refines out of ego", "refines", dirOut, zoneTop},
		{"extends out of ego", "extends", dirOut, zoneTop},
		{"grounded-by out of ego", "grounded-by", dirOut, zoneTop},

		// BOTTOM — what builds on / is subordinate to the ego
		{"governs out of ego", "governs", dirOut, zoneBottom}, // ego governs it -> it answers to the ego
		{"refines into ego", "refines", dirIn, zoneBottom},
		{"extends into ego", "extends", dirIn, zoneBottom},
		{"grounded-by into ego", "grounded-by", dirIn, zoneBottom},
		{"supports out of ego", "supports", dirOut, zoneBottom},

		// LEFT — tension (either direction)
		{"contradicts out", "contradicts", dirOut, zoneLeft},
		{"contradicts in", "contradicts", dirIn, zoneLeft},
		{"questions out", "questions", dirOut, zoneLeft},
		{"questions in", "questions", dirIn, zoneLeft},

		// RIGHT — lateral: provenance / task / generic association (either direction)
		{"source-of out", "source-of", dirOut, zoneRight},
		{"source-of in", "source-of", dirIn, zoneRight},
		{"requires out", "requires", dirOut, zoneRight},
		{"requires in", "requires", dirIn, zoneRight},
		{"follows out", "follows", dirOut, zoneRight},
		{"follows in", "follows", dirIn, zoneRight},
		// 'related' is a legacy/generic association (not a core nn type) -> lateral.
		{"related out", "related", dirOut, zoneRight},
		{"related in", "related", dirIn, zoneRight},

		// Evidence that supports the ego is something the ego answers to.
		{"supports into ego", "supports", dirIn, zoneTop},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := zoneOf(tc.linkType, tc.dir)
			if got != tc.want {
				t.Errorf("zoneOf(%q, %q) = %q, want %q", tc.linkType, tc.dir, got, tc.want)
			}
		})
	}
}

func TestGraphTemplateDocumentsFollowsAsLateral(t *testing.T) {
	template, err := os.ReadFile("templates/graph.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(template)
	for _, required := range []string{"source-of, requires, follows", `case "follows": return "#f7c66b"`} {
		if !strings.Contains(text, required) {
			t.Errorf("graph template missing follows lateral presentation %q", required)
		}
	}
}

func TestNavigationGuideDocumentsEvidenceZoneDirections(t *testing.T) {
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"grounded-by OUT → TOP", "supports OUT → BOTTOM", "supports IN → TOP"} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing evidence-zone rule %q", required)
		}
	}
}
