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

// Assertion: post-gate capture prompt instructs nn new --quick as default.
func TestCaptureDisciplinePostGatePrompt(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "nn new --quick") {
		t.Errorf("expected post-gate capture prompt 'nn new --quick' in protocol body; got:\n%s", out)
	}
}

// Assertion: capture is the default after gated action (opt-out model, not opt-in).
func TestCaptureDisciplineDefaultCapture(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if strings.Contains(out, "reusable across sessions") {
		t.Errorf("capture-discipline must not contain opt-in judgment 'reusable across sessions'; got:\n%s", out)
	}
}

// Assertion: skip-capture: prefix is the required opt-out string.
func TestCaptureDisciplineOptOutSkip(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "skip-capture:") {
		t.Errorf("capture-discipline must contain 'skip-capture:' opt-out prefix; got:\n%s", out)
	}
}

// Assertion: skip-capture requires naming a specific execution artifact.
func TestCaptureDisciplineSkipRequiresArtifact(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "runtime-only") {
		t.Errorf("capture-discipline must contain 'runtime-only' artifact class in skip condition; got:\n%s", out)
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

// Assertion: gate requires judgment-based selection with match_reason citation, not word-sharing criterion.
func TestVirtualCaptureDisciplineC3RationaleOverlap(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Selected because:") {
		t.Errorf("expected 'Selected because:' judgment requirement for nn show trigger; got:\n%s", out)
	}
	if !strings.Contains(out, "match_reason") {
		t.Errorf("expected match_reason citation requirement; got:\n%s", out)
	}
	if strings.Contains(out, "tool call argument string") {
		t.Errorf("gate should not reference 'tool call argument string'; got:\n%s", out)
	}
}
