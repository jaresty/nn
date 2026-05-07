package cmd

import (
	"strings"
	"testing"
)

// Assertion: C3 uses word-overlap against stated search rationale (not tool call argument string).
func TestVirtualCaptureDisciplineC3ArgumentString(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "stated search rationale") {
		t.Errorf("expected rationale-overlap clause in C3; got:\n%s", out)
	}
}

// Assertion: C8 skip uses verbatim excerpt + runtime-value path only (no intent-assessed durability reason).
func TestVirtualCaptureDisciplineC8SkipStructural(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "verbatim excerpt from the result") {
		t.Errorf("expected verbatim excerpt requirement in C8 skip; got:\n%s", out)
	}
	if !strings.Contains(out, "result is a runtime value") {
		t.Errorf("expected runtime-value skip path in C8; got:\n%s", out)
	}
	if strings.Contains(out, "durability reason stating why it would not change behavior") {
		t.Errorf("C8 skip should not contain intent-assessed durability reason; got:\n%s", out)
	}
}
