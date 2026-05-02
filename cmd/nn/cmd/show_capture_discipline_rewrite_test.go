package cmd

import (
	"strings"
	"testing"
)

// Assertion: virtual capture discipline body uses explicit allow-list, not self-assessed trigger.
func TestVirtualCaptureDisciplineNoSelfAssessedTrigger(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if strings.Contains(out, "introduces new information not already present in the conversation") {
		t.Errorf("virtual protocol should not contain self-assessed trigger phrase; got:\n%s", out)
	}
}

// Assertion: virtual capture discipline requires search result immediately above the action.
func TestVirtualCaptureDisciplineRequiresProximateSearch(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "immediately above it") {
		t.Errorf("expected 'immediately above it' in virtual protocol body; got:\n%s", out)
	}
}
