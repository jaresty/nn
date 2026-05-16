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

// Assertion: virtual-nn-error-handling appears in nn show --global output.
func TestShowVirtualErrorHandlingGlobal(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-error-handling") {
		t.Errorf("expected virtual-nn-error-handling id in output:\n%s", out)
	}
}

// Assertion: nn show virtual-nn-error-handling returns body text.
func TestShowVirtualErrorHandlingBody(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-error-handling")
	if err != nil {
		t.Fatalf("nn show virtual-nn-error-handling: %v", err)
	}
	if !strings.Contains(out, "non-obvious error message") {
		t.Errorf("expected virtual error-handling body text in output:\n%s", out)
	}
}

// Assertion: error-handling skip clause is present and requires verbatim citation.
func TestShowVirtualErrorHandlingSkipClause(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "virtual-nn-error-handling")
	if err != nil {
		t.Fatalf("nn show virtual-nn-error-handling: %v", err)
	}
	if !strings.Contains(out, "Expected FAIL:") {
		t.Errorf("expected skip condition referencing 'Expected FAIL:' in output:\n%s", out)
	}
	if !strings.Contains(out, "verbatim") {
		t.Errorf("expected skip condition to require verbatim citation:\n%s", out)
	}
}

// Assertion: capture-discipline skip clause requires quoting a verbatim excerpt from the tool result,
// with [] mapping to "zero results returned" and non-empty results requiring a title citation.
func TestCaptureDisciplineSkipClauseRequiresVerbatimExcerpt(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-capture-discipline")
	if err != nil {
		t.Fatalf("nn show virtual-nn-capture-discipline: %v", err)
	}
	if !strings.Contains(out, "verbatim excerpt") {
		t.Errorf("expected skip clause to require 'verbatim excerpt' from tool result:\n%s", out)
	}
	if !strings.Contains(out, `[]`) {
		t.Errorf("expected skip clause to name '[]' as the zero-results signal:\n%s", out)
	}
	if !strings.Contains(out, "zero results returned") {
		t.Errorf("expected skip clause to name 'zero results returned' as the zero-results declaration:\n%s", out)
	}
}
