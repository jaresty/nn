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
func TestShowGlobalVirtualFullBodyPresent(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	// Virtual notes have no notebook file — their full body must appear in --global output
	// so consumers don't need a follow-up nn show <id> to access the content.
	if !strings.Contains(out, "search window do not reset it") {
		t.Errorf("expected virtual protocol full body in --global output; got:\n%s", out)
	}
}
