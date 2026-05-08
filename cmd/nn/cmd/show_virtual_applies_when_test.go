package cmd

import (
	"strings"
	"testing"
)

// Assertion: nn show --global virtual protocol contains applies_when field.
func TestShowGlobalVirtualAppliesWhen(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "applies_when:") {
		t.Errorf("expected applies_when in virtual protocol --global output; got:\n%s", out)
	}
}

// Assertion: nn show --global virtual protocol body is NOT in compact output.
func TestShowGlobalVirtualNoFullBody(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	// The full protocol body contains this sentence — it should not appear in compact output.
	if strings.Contains(out, "Every action requires a preceding") {
		t.Errorf("expected virtual protocol full body to be absent from --global output; got:\n%s", out)
	}
}
