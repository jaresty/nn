package cmd

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Property: `nn rules check` reports a violation and exits non-zero when a note
// graph breaks a built-in invariant; it reports "ok" and exits zero otherwise.

func TestRulesCheckReportsViolationAndErrors(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// A non-model root of an ontology subgraph is a violation.
	root := newTestNoteForCLI(note.GenerateID(), "Bad Root", note.TypeConcept)
	root.Representation = "ontology"
	child := newTestNoteForCLI(note.GenerateID(), "Child", note.TypeConcept)
	child.Representation = "ontology"
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "x", Type: "refines"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("rules", "check")
	if err == nil {
		t.Fatalf("expected non-zero exit on violation, got nil; out: %s", out)
	}
	if !strings.Contains(out, root.ID) {
		t.Errorf("expected violation output to name %s, got: %s", root.ID, out)
	}
	if !strings.Contains(out, "model") {
		t.Errorf("expected violation reason about model root, got: %s", out)
	}
}

func TestRulesCheckOkWhenClean(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := newTestNoteForCLI(note.GenerateID(), "Good Root", note.TypeModel)
	root.Representation = "ontology"
	child := newTestNoteForCLI(note.GenerateID(), "Child", note.TypeConcept)
	child.Representation = "ontology"
	root.Links = []note.Link{{TargetID: child.ID, Annotation: "x", Type: "refines"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("rules", "check")
	if err != nil {
		t.Fatalf("expected clean pass, got error: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("expected 'ok' output, got: %s", out)
	}
}

// Property: a ```nn-rule fence in a note body contributes a derivable fact
// that `nn rules query` can surface.
func TestRulesQuerySurfacesUserRule(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	a.Body = strings.Join([]string{
		"```nn-rule",
		`tagged_daily(N) :- tag(N, "daily").`,
		"```",
	}, "\n")
	a.Tags = []string{"daily"}
	writeNoteFile(t, nbDir, a)

	out, err := execute("rules", "query", "tagged_daily")
	if err != nil {
		t.Fatalf("rules query: %v\nout: %s", err, out)
	}
	if !strings.Contains(out, "tagged_daily("+a.ID+")") {
		t.Errorf("expected derived fact for %s, got: %s", a.ID, out)
	}
}
