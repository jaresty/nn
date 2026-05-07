package cmd

import (
	"strings"
	"testing"
)

// Assertion: nn show --global contains the ## Protocols derivation block (once, at the end).
func TestShowGlobalDerivationBlockPresent(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "## Protocols") {
		t.Errorf("expected ## Protocols derivation block in --global output; got:\n%s", out)
	}
}

// Assertion: nn show --global contains the derivation block only once.
func TestShowGlobalDerivationBlockOnce(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	count := strings.Count(out, "Before responding to any message this session")
	if count != 1 {
		t.Errorf("expected derivation block exactly once in --global output; got %d occurrences:\n%s", count, out)
	}
}

// Assertion: nn show <individual-protocol-id> does NOT contain the ## Protocols derivation block.
func TestShowIndividualProtocolNoDerivationBlock(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("new", "--type", "protocol", "--title", "Test Individual Protocol", "--content", "Some protocol body.", "--no-edit")
	if err != nil {
		t.Fatalf("nn new protocol: %v", err)
	}
	id := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(out), "created "))
	out, err = execute("show", id)
	if err != nil {
		t.Fatalf("nn show %s: %v", id, err)
	}
	if strings.Contains(out, "Before responding to any message this session") {
		t.Errorf("expected no derivation block in individual nn show output; got:\n%s", out)
	}
}
