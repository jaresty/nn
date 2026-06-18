package cmd

import (
	"strings"
	"testing"
)

// Assertion: nn show --global virtual protocol includes reading source files in gate condition.
func TestShowGlobalVirtualIncludesSourceFileGate(t *testing.T) {
	_, execute := setupNotebook(t)

	// Body is in nn show <id>, not --global compact output.
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "Allow-list (no gate required)") {
		t.Errorf("expected allow-list section in virtual protocol output:\n%s", out)
	}
}
