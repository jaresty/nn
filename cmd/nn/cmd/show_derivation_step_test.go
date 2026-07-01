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

// Assertion: post-gate capture prompt fires when result is reusable — instructs nn new --quick.
func TestCaptureDisciplinePostGatePrompt(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "nn new --quick") {
		t.Errorf("expected post-gate capture prompt 'nn new --quick' in protocol body; got:\n%s", out)
	}
	if !strings.Contains(out, "reusable across sessions") {
		t.Errorf("expected condition 'reusable across sessions' in protocol body; got:\n%s", out)
	}
}

// Assertion: re-discovery of a draft note triggers promotion instruction in capture-discipline.
func TestCaptureDisciplineRediscoveryPromotion(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "draft note") {
		t.Errorf("expected re-discovery promotion for 'draft note' in capture-discipline body; got:\n%s", out)
	}
	if !strings.Contains(out, "--status reviewed") {
		t.Errorf("expected '--status reviewed' promotion instruction in capture-discipline body; got:\n%s", out)
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
