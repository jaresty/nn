package cmd

import (
	"strings"
	"testing"
)

// Assertion: Found path clause uses match_reason substring criterion, not numeric score threshold.
func TestCaptureDisciplineMatchReasonSubstringCriterion(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "match_reason` contains the literal") {
		t.Errorf("expected match_reason substring criterion in Found path clause; got:\n%s", out)
	}
	if strings.Contains(out, "score must be") || strings.Contains(out, ">= 0.3") || strings.Contains(out, "≥ 0.3") {
		t.Errorf("Found path clause must not use numeric score threshold; got:\n%s", out)
	}
}

// Assertion: Repeat search suppression paragraph present after Empty/truncated path block.
func TestCaptureDisciplineRepeatSearchSuppression(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Repeat search suppression") {
		t.Errorf("expected 'Repeat search suppression' paragraph; got:\n%s", out)
	}
	if !strings.Contains(out, "already searched this session") {
		t.Errorf("expected 'already searched this session' exit string; got:\n%s", out)
	}
	// Verify topic placeholder is the literal --search argument, not a free-form [topic]
	if !strings.Contains(out, "literal `--search` argument string") {
		t.Errorf("expected 'literal `--search` argument string' to define topic identity; got:\n%s", out)
	}
}
