package cmd

import "testing"

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
		{"supports into ego", "supports", dirIn, zoneBottom},

		// LEFT — tension (either direction)
		{"contradicts out", "contradicts", dirOut, zoneLeft},
		{"contradicts in", "contradicts", dirIn, zoneLeft},
		{"questions out", "questions", dirOut, zoneLeft},
		{"questions in", "questions", dirIn, zoneLeft},

		// RIGHT — lateral: provenance / task edges (either direction)
		{"source-of out", "source-of", dirOut, zoneRight},
		{"source-of in", "source-of", dirIn, zoneRight},
		{"requires out", "requires", dirOut, zoneRight},
		{"requires in", "requires", dirIn, zoneRight},

		// supports OUT is not a top/bottom relation (ego corroborates target) -> lateral-none is acceptable;
		// pin it to none so the mapping stays total and intentional.
		{"supports out of ego", "supports", dirOut, zoneNone},
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
