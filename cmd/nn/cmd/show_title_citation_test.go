package cmd

import (
	"strings"
	"testing"
)

// Assertion: protocol requires citing a result title when dismissing search results.
func TestVirtualCaptureDisciplineTitleCitationDismiss(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "cite") {
		t.Errorf("expected title-citation requirement in protocol; got:\n%s", out)
	}
	if !strings.Contains(out, "does not cover") {
		t.Errorf("expected 'does not cover' dismissal clause in protocol; got:\n%s", out)
	}
}
