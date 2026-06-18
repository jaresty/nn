package cmd

import (
	"strings"
	"testing"
)

// Assertion: protocol requires Gate: Search rationale form and nn show word-match trigger.
func TestVirtualCaptureDisciplineSearchRationale(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Gate: Search rationale:") {
		t.Errorf("expected 'Gate: Search rationale:' form in virtual protocol; got:\n%s", out)
	}
	if strings.Contains(out, "Use X as the search topic, not the action") {
		t.Errorf("protocol should not contain unenforceable correctness claim; got:\n%s", out)
	}
}

// Assertion: C3 word-overlap check uses topic-sharing criterion, not tool call argument string.
func TestVirtualCaptureDisciplineC3RationaleOverlap(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "shares a word with") {
		t.Errorf("expected word-sharing criterion for nn show trigger; got:\n%s", out)
	}
	if strings.Contains(out, "tool call argument string") {
		t.Errorf("C3 should not reference 'tool call argument string'; got:\n%s", out)
	}
}
