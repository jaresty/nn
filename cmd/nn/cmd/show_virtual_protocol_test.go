package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: nn show --global always includes virtual protocols even when notebook is empty.
func TestShowGlobalVirtualAlwaysPresent(t *testing.T) {
	_, execute := setupNotebook(t)
	// Empty notebook — no notes written.

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected virtual-nn-capture-discipline id in output:\n%s", out)
	}
	// Compact format: body is not in --global; fetch via nn show <id>
	out2, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out2, "Every action requires a preceding") {
		t.Errorf("expected virtual protocol body text in nn show output:\n%s", out2)
	}
}

// Assertion: nn show --global virtual protocols appear alongside real notebook protocols.
func TestShowGlobalVirtualAppearsWithReal(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	real := newTestNoteForCLI(note.GenerateID(), "Real Protocol", note.TypeProtocol)
	writeNoteFile(t, nbDir, real)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Real Protocol") {
		t.Errorf("expected real protocol in output:\n%s", out)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected virtual protocol in output alongside real:\n%s", out)
	}
}
