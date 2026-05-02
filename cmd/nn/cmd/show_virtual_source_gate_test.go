package cmd

import (
	"strings"
	"testing"
)

// Assertion: nn show --global virtual protocol includes reading source files in gate condition.
func TestShowGlobalVirtualIncludesSourceFileGate(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "reading source files not authored this session") {
		t.Errorf("expected 'reading source files not authored this session' in virtual protocol output:\n%s", out)
	}
}
