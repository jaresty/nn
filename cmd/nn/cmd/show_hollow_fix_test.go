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

// Assertion: C8 skip uses verbatim excerpt + skip-capture: runtime-only prefix (no intent-assessed durability reason).
func TestVirtualCaptureDisciplineC8SkipStructural(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "verbatim excerpt from the result") {
		t.Errorf("expected verbatim excerpt requirement in C8 skip; got:\n%s", out)
	}
	if !strings.Contains(out, "skip-capture: runtime-only") {
		t.Errorf("expected 'skip-capture: runtime-only' skip path in C8; got:\n%s", out)
	}
	if strings.Contains(out, "durability reason stating why it would not change behavior") {
		t.Errorf("C8 skip should not contain intent-assessed durability reason; got:\n%s", out)
	}
}
