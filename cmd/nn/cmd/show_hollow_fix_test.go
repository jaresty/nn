package cmd

import (
	"strings"
	"testing"
)

// Assertion: gate uses judgment+match_reason selection, not word-sharing criterion against topic.
func TestVirtualCaptureDisciplineC3ArgumentString(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Selected because:") {
		t.Errorf("expected 'Selected because:' judgment requirement in gate; got:\n%s", out)
	}
	if strings.Contains(out, "shares a word with") {
		t.Errorf("gate should not use word-sharing criterion (replaced by match_reason judgment); got:\n%s", out)
	}
}

// Assertion: post-gate clause uses two-branch opt-in model anchored to gate outcome strings.
func TestVirtualCaptureDisciplineC8SkipStructural(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "verbatim excerpt from the result") {
		t.Errorf("expected verbatim excerpt requirement in post-gate clause; got:\n%s", out)
	}
	if !strings.Contains(out, "not in note:") {
		t.Errorf("expected 'not in note:' capture branch string; got:\n%s", out)
	}
	if !strings.Contains(out, "represented by:") {
		t.Errorf("expected 'represented by:' no-capture string; got:\n%s", out)
	}
	if strings.Contains(out, "skip-capture: runtime-only") {
		t.Errorf("old runtime-only prefix must be removed; got:\n%s", out)
	}
	if strings.Contains(out, "skip-capture: execution-artifact") {
		t.Errorf("old execution-artifact prefix must be removed; got:\n%s", out)
	}
	if strings.Contains(out, "skip-capture: session-transient") {
		t.Errorf("old session-transient prefix must be removed; got:\n%s", out)
	}
}
