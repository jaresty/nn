package cmd

import (
	"strings"
	"testing"
)

// Assertion: protocol requires Search rationale sentence with structural search query constraint.
func TestVirtualCaptureDisciplineSearchRationale(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Search rationale:") {
		t.Errorf("expected 'Search rationale:' derivation step in virtual protocol; got:\n%s", out)
	}
	if !strings.Contains(out, "The search query must contain at least one word from X") {
		t.Errorf("expected structural search query constraint in derivation sentence; got:\n%s", out)
	}
	if strings.Contains(out, "Use X as the search topic, not the action") {
		t.Errorf("derivation sentence should not contain unenorceable correctness claim; got:\n%s", out)
	}
}

// Assertion: C3 word-overlap check references stated search rationale, not tool call argument string.
func TestVirtualCaptureDisciplineC3RationaleOverlap(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "stated search rationale") {
		t.Errorf("expected C3 to reference 'stated search rationale'; got:\n%s", out)
	}
	if strings.Contains(out, "tool call argument string") {
		t.Errorf("C3 should not reference 'tool call argument string' after rationale update; got:\n%s", out)
	}
}
