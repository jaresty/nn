package cmd

import (
	"strings"
	"testing"
)

// Assertion: protocol requires quoting a verbatim excerpt after the gated action.
func TestVirtualCaptureDisciplineTitleCitationDismiss(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "quote a verbatim excerpt from the result") {
		t.Errorf("expected verbatim-excerpt requirement after gated action; got:\n%s", out)
	}
}
