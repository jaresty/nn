package cmd

import (
	"strings"
	"testing"
)

// Assertion: virtual capture discipline body uses explicit allow-list, not self-assessed trigger.
func TestVirtualCaptureDisciplineNoSelfAssessedTrigger(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if strings.Contains(out, "introduces new information not already present in the conversation") {
		t.Errorf("virtual protocol should not contain self-assessed trigger phrase; got:\n%s", out)
	}
}

// Assertion: virtual capture discipline requires search result immediately above the action.
func TestVirtualCaptureDisciplineRequiresProximateSearch(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "result block occupies the position immediately before the gated tool call") {
		t.Errorf("expected proximate search requirement in virtual protocol body; got:\n%s", out)
	}
}

// Assertion D5: trigger is structural (Read tool call / file-reading Bash), not semantic session-provenance.
func TestVirtualCaptureDisciplineStructuralTrigger(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Read tool call or file-reading Bash tool call") {
		t.Errorf("expected structural trigger in body; got:\n%s", out)
	}
	if strings.Contains(out, "not authored this session") {
		t.Errorf("body must not contain semantic session-provenance trigger; got:\n%s", out)
	}
}

// Assertion D6: body requires explicit Gate: line before every gated tool call.
func TestVirtualCaptureDisciplineGateLine(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Gate: allow-listed") {
		t.Errorf("expected 'Gate: allow-listed' form in body; got:\n%s", out)
	}
	if !strings.Contains(out, "Gate: Search rationale:") {
		t.Errorf("expected 'Gate: Search rationale:' form in body; got:\n%s", out)
	}
}

// Assertion D7: body contains discovery framing — prior knowledge is not an exemption.
func TestVirtualCaptureDisciplineDiscoveryFraming(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Prior knowledge") {
		t.Errorf("expected 'Prior knowledge' discovery framing in body; got:\n%s", out)
	}
	if !strings.Contains(out, "not an exemption") {
		t.Errorf("expected 'not an exemption' in discovery framing; got:\n%s", out)
	}
	if !strings.Contains(out, "Skip resistance") {
		t.Errorf("expected 'Skip resistance' signal sentence in body; got:\n%s", out)
	}
}

// Assertion D8: body requires retry with rephrased query when search returns [].
func TestVirtualCaptureDisciplineZeroResultsRetry(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "rephrased") {
		t.Errorf("expected retry-with-rephrased requirement for zero results; got:\n%s", out)
	}
}
