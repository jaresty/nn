package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// setup helper: a protocol note whose nn-rule flags any draft it governs, and a
// governed draft target. Returns (execute, targetID).
func setupGovernedDraft(t *testing.T) (func(...string) (string, error), string) {
	t.Helper()
	nbDir, execute := setupNotebook(t)

	protocol := newTestNoteForCLI(note.GenerateID(), "Governing Protocol", note.TypeProtocol)
	protocol.Body = strings.Join([]string{
		"```nn-rule",
		`violation(N, "governed note must not be draft") :- link(P, N, "governs"), note(N, _, "draft").`,
		"```",
	}, "\n")
	target := newTestNoteForCLI(note.GenerateID(), "Governed Target", note.TypeConcept)
	protocol.Links = []note.Link{{TargetID: target.ID, Annotation: "governs it", Type: "governs"}}

	writeNoteFile(t, nbDir, protocol)
	writeNoteFile(t, nbDir, target)
	return execute, target.ID
}

// P1: default nn show must NOT include the rule-violations section (no engine run).
func TestShow_DefaultOmitsRuleViolations(t *testing.T) {
	execute, targetID := setupGovernedDraft(t)

	out, err := execute("show", targetID)
	if err != nil {
		t.Fatalf("show: %v\nout: %s", err, out)
	}
	if strings.Contains(out, "Rule violations") {
		t.Fatalf("default nn show must not include the Rule violations section, got:\n%s", out)
	}
}

// P2: nn show --rules DOES include the rule-violations section for a flagged note.
func TestShow_RulesFlagIncludesViolations(t *testing.T) {
	execute, targetID := setupGovernedDraft(t)

	out, err := execute("show", targetID, "--rules")
	if err != nil {
		t.Fatalf("show --rules: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "Rule violations") {
		t.Fatalf("nn show --rules must include the Rule violations section, got:\n%s", out)
	}
	if !strings.Contains(out, "governed note must not be draft") {
		t.Fatalf("nn show --rules must show the violation reason, got:\n%s", out)
	}
}