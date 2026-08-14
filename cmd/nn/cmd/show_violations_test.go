package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Property: nn show surfaces rule violations whose subject is the shown note,
// closing the protocol-enforcement loop — a type:protocol note's nn-rule can
// flag the notes it governs, and that shows up when you look at them.

func TestRuleViolationsSection_FlagsShownNote(t *testing.T) {
	// A protocol note carries an nn-rule that flags any draft note it governs.
	protocol := &note.Note{
		ID: "prot", Type: note.TypeProtocol, Status: note.StatusPermanent,
		Body: strings.Join([]string{
			"```nn-rule",
			`violation(N, "governed note must not be draft") :- link(P, N, "governs"), note(N, _, "draft").`,
			"```",
		}, "\n"),
		Links: []note.Link{{TargetID: "target", Type: "governs"}},
	}
	target := &note.Note{ID: "target", Type: note.TypeConcept, Status: note.StatusDraft}

	byID := map[string]*note.Note{"prot": protocol, "target": target}

	// Showing the governed (draft) target must surface the violation.
	sec := ruleViolationsSection(target, byID)
	if !strings.Contains(sec, "Rule violations") {
		t.Fatalf("expected a Rule violations section, got: %q", sec)
	}
	if !strings.Contains(sec, "governed note must not be draft") {
		t.Fatalf("expected the violation reason, got: %q", sec)
	}
}

func TestRuleViolationsSection_CleanNoteEmpty(t *testing.T) {
	// A note that violates nothing yields no section.
	clean := &note.Note{ID: "ok", Type: note.TypeConcept, Status: note.StatusReviewed}
	byID := map[string]*note.Note{"ok": clean}
	if sec := ruleViolationsSection(clean, byID); sec != "" {
		t.Fatalf("expected empty section for clean note, got: %q", sec)
	}
}
