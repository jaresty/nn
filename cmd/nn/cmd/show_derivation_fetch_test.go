package cmd

import (
	"strings"
	"testing"
)

// Assertion: derivation block instructs LLM to fetch full body when applies_when holds.
func TestDerivationBlockInstructsFetch(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "nn show") {
		t.Errorf("expected derivation block to instruct 'nn show <id>' fetch; got:\n%s", out)
	}
	if !strings.Contains(out, "applies_when") {
		t.Errorf("expected derivation block to reference applies_when evaluation; got:\n%s", out)
	}
}

// Assertion: derivation block instructs LLM to add applies_when if a protocol lacks it.
func TestDerivationBlockInstructsAddAppliesWhen(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "add an applies_when") {
		t.Errorf("expected derivation block to instruct adding applies_when when missing; got:\n%s", out)
	}
}
