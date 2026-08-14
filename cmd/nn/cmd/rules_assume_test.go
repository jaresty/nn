package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// --assume evaluates the ruleset as if a hypothetical fact were present. A
// type:concept note with NO representation on disk is clean; assuming it carries
// representation=ontology makes it a non-model ontology root, which the built-in
// "root must be type:model" invariant flags — a counterfactual that never
// touches disk.
func TestRulesCheckAssumeInjectsHypotheticalFact(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Rootless", note.TypeConcept)
	// No Representation set — clean under normal evaluation.
	writeNoteFile(t, nbDir, n)

	// Baseline: without --assume, no violation.
	out, err := execute("rules", "check")
	if err != nil {
		t.Fatalf("baseline should be clean, got error: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("baseline expected 'ok', got: %s", out)
	}

	// With the assumed representation fact, the same note becomes a non-model
	// ontology root and is flagged.
	out, err = execute("rules", "check", "--assume", "representation("+n.ID+", ontology)")
	if err == nil {
		t.Fatalf("expected a violation under --assume, got nil; out: %s", out)
	}
	if !strings.Contains(out, n.ID) || !strings.Contains(out, "model") {
		t.Fatalf("expected a model-root violation naming %s, got: %s", n.ID, out)
	}
}

// A malformed --assume value is a hard error (unlike malformed nn-rule fences,
// which warn and load the rest).
func TestRulesCheckAssumeMalformedErrors(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("rules", "check", "--assume", "not a fact")
	if err == nil {
		t.Fatalf("expected an error for malformed --assume, got nil; out: %s", out)
	}
}
