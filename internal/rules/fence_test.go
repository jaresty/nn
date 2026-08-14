package rules

import (
	"strings"
	"testing"
)

func TestExtractFenceRules_WellFormed(t *testing.T) {
	body := strings.Join([]string{
		"Some prose.",
		"",
		"```nn-rule",
		`tgov(X, Y) :- link(X, Y, "governs").`,
		`violation(N, "bad") :- link(N, T, "contradicts"), note(T, _, "permanent").`,
		"```",
		"",
		"More prose.",
	}, "\n")

	rules, warns := ExtractFenceRules("id-1", body)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(rules))
	}
	if rules[0].Head.Pred != "tgov" || rules[1].Head.Pred != "violation" {
		t.Fatalf("unexpected heads: %q, %q", rules[0].Head.Pred, rules[1].Head.Pred)
	}
}

func TestExtractFenceRules_MalformedWarnsWithProvenanceAndKeepsGood(t *testing.T) {
	body := strings.Join([]string{
		"```nn-rule",
		`good(X) :- note(X, _, _).`,
		`this is not a valid clause at all`,
		"```",
	}, "\n")

	rules, warns := ExtractFenceRules("note-42", body)

	// The good rule must still be returned — a bad clause never suppresses loading.
	if len(rules) != 1 || rules[0].Head.Pred != "good" {
		t.Fatalf("good rule not preserved: %+v", rules)
	}
	// There must be a warning, and it must name the note ID for provenance.
	if len(warns) == 0 {
		t.Fatal("expected a warning for the malformed clause, got none")
	}
	if !strings.Contains(warns[0], "note-42") {
		t.Fatalf("warning lacks note-ID provenance: %q", warns[0])
	}
}

func TestExtractFenceRules_NoFence(t *testing.T) {
	rules, warns := ExtractFenceRules("id", "just prose, no fences here")
	if len(rules) != 0 || len(warns) != 0 {
		t.Fatalf("expected nothing, got rules=%v warns=%v", rules, warns)
	}
}
