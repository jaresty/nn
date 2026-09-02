package cmd

import (
	"strings"
	"testing"
)

// The capture-discipline staleness check keys expiry on claim volatility, not
// concreteness. This guards the rewritten language rendered by
// nn show virtual-nn-capture-discipline.
func TestStalenessChecksVolatilityNotConcreteness(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}

	// property [3]: concreteness is no longer the expiry trigger.
	if strings.Contains(out, "concrete and time-bound") {
		t.Errorf("staleness check still triggers on concreteness ('concrete and time-bound'); got:\n%s", out)
	}
	if strings.Contains(out, "concrete+old") {
		t.Errorf("staleness check still emits the concreteness sentinel 'concrete+old'; got:\n%s", out)
	}

	// property [1]: source/date-bounded historical claims are not expired.
	if !strings.Contains(out, "bounded to a past source") {
		t.Errorf("staleness check does not carve out source/date-bounded historical claims; got:\n%s", out)
	}

	// property [2]: claims whose truth depends on mutable present/future
	// conditions get an expiry or expires_when.
	if !strings.Contains(out, "mutable present or future") {
		t.Errorf("staleness check does not key expiry on mutable present/future conditions; got:\n%s", out)
	}
	if !strings.Contains(out, "expires_when") {
		t.Errorf("staleness check does not mention expires_when for forecasts/unresolved assumptions; got:\n%s", out)
	}
}

// TestStalenessClassificationResistsContextualClaimSubstitution guards the
// observed failure mode: a present-tense title, stale freshness label, or
// conversation about current/future use must not replace a commit-bounded body
// claim with a stronger current-state proposition.
func TestStalenessClassificationResistsContextualClaimSubstitution(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}

	assertions := []string{
		"Classify the stored claim exactly as written in the note body",
		"Do not strengthen, reinterpret, or replace it using the title, freshness display, neighboring notes, search context, or the current conversation",
		"If the body explicitly anchors the claim to a past source, commit, revision, event, or date, stop immediately",
		"classify it as historical and never add expiry",
		"Freshness may limit using that observation as evidence about the present, but it does not make the historical observation expire",
		"classified proposition: \"<verbatim sentence from the note body>\"",
	}
	for _, assertion := range assertions {
		if !strings.Contains(out, assertion) {
			t.Errorf("missing staleness safeguard %q; got:\n%s", assertion, out)
		}
	}
}
