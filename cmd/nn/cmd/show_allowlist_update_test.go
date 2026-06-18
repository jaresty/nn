package cmd

import (
	"strings"
	"testing"
)

// Assertion: bullet 2 covers commands producing output from session state.
func TestVirtualCaptureDisciplineBullet2LocalOnly(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "output solely from session state") {
		t.Errorf("expected bullet-2 session-state clause in virtual protocol; got:\n%s", out)
	}
}

// Assertion: bullet 3 covers live machine-generated output with resource identifier.
func TestVirtualCaptureDisciplineBullet3RemoteExecution(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "resource identifier for that") {
		t.Errorf("expected bullet-3 resource-identifier clause in virtual protocol; got:\n%s", out)
	}
}
