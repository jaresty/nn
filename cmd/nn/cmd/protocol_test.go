package cmd

import (
	"strings"
	"testing"
)

// Assertion: nn new --type protocol creates a note successfully.
func TestNewProtocolType(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--type", "protocol", "--title", "Test Protocol", "--content", "When X, do Y.", "--no-edit")
	if err != nil {
		t.Fatalf("nn new --type protocol: %v", err)
	}
	if !strings.HasPrefix(out, "created ") {
		t.Errorf("expected 'created <id>', got %q", out)
	}
}

// Assertion: nn list --type protocol filters to only protocol notes.
func TestListFilterProtocol(t *testing.T) {
	_, execute := setupNotebook(t)

	out1, err := execute("new", "--type", "protocol", "--title", "My Protocol", "--content", "body", "--no-edit")
	if err != nil {
		t.Fatalf("create protocol note: %v", err)
	}
	protocolID := strings.TrimPrefix(strings.TrimSpace(out1), "created ")

	_, err = execute("new", "--type", "concept", "--title", "Not A Protocol", "--content", "body", "--no-edit")
	if err != nil {
		t.Fatalf("create concept note: %v", err)
	}

	out, err := execute("list", "--type", "protocol")
	if err != nil {
		t.Fatalf("nn list --type protocol: %v", err)
	}
	if !strings.Contains(out, protocolID) {
		t.Errorf("protocol ID %q not in output:\n%s", protocolID, out)
	}
	if strings.Contains(out, "Not A Protocol") {
		t.Errorf("concept note leaked into --type protocol output:\n%s", out)
	}
}
