package rules

import (
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// runBuiltin builds an engine from notes + the embedded builtin ruleset,
// evaluates it, and returns the violation facts as "id|reason" strings.
func runBuiltin(t *testing.T, notes []*note.Note) []string {
	t.Helper()
	e := NewEngine()
	for _, f := range FactsFromNotes(notes) {
		e.AddFact(f)
	}
	rules, err := ParseProgram(BuiltinRules())
	if err != nil {
		t.Fatalf("parse builtin: %v", err)
	}
	e.AddRules(rules)
	if err := e.Eval(); err != nil {
		t.Fatalf("eval: %v", err)
	}
	var out []string
	for _, f := range e.Query("violation") {
		out = append(out, f.Args[0]+"|"+f.Args[1])
	}
	return out
}

func n(id string, ty note.Type, rep string, links ...note.Link) *note.Note {
	return &note.Note{ID: id, Type: ty, Status: note.StatusReviewed, Representation: rep, Links: links}
}

// A well-formed ontology: model root -> concept children, no cycles. No violations.
func TestBuiltin_ValidOntologyPasses(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "ontology", note.Link{TargetID: "c1", Type: "refines"}),
		n("c1", note.TypeConcept, "ontology", note.Link{TargetID: "c2", Type: "supports"}),
		n("c2", note.TypeConcept, "ontology"),
	}
	if v := runBuiltin(t, notes); len(v) != 0 {
		t.Fatalf("expected no violations, got %v", v)
	}
}

// A non-model root must be flagged (mirrors check.go root type check).
func TestBuiltin_NonModelRootFlagged(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeConcept, "ontology", note.Link{TargetID: "c1", Type: "refines"}),
		n("c1", note.TypeConcept, "ontology"),
	}
	v := runBuiltin(t, notes)
	if !anyContains(v, "root", "model") {
		t.Fatalf("expected non-model-root violation, got %v", v)
	}
}

// A non-root that is neither concept nor argument must be flagged.
func TestBuiltin_BadChildTypeFlagged(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "ontology", note.Link{TargetID: "bad", Type: "refines"}),
		n("bad", note.TypeObservation, "ontology"),
	}
	v := runBuiltin(t, notes)
	if !anyContains(v, "bad", "concept") {
		t.Fatalf("expected bad-child-type violation, got %v", v)
	}
}

// Taxonomy disallows link types other than refines/extends.
func TestBuiltin_TaxonomyDisallowedLinkFlagged(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "taxonomy", note.Link{TargetID: "c1", Type: "supports"}),
		n("c1", note.TypeConcept, "taxonomy"),
	}
	v := runBuiltin(t, notes)
	if !anyContains(v, "root", "taxonomy") {
		t.Fatalf("expected taxonomy-link violation, got %v", v)
	}
}

// Axiom root must have at least one grounded-by link.
func TestBuiltin_AxiomMissingGroundedByFlagged(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "axiom", note.Link{TargetID: "c1", Type: "refines"}),
		n("c1", note.TypeConcept, "axiom"),
	}
	v := runBuiltin(t, notes)
	if !anyContains(v, "root", "grounded-by") {
		t.Fatalf("expected axiom grounded-by violation, got %v", v)
	}
}

// An axiom root WITH a grounded-by link passes the grounded-by check.
func TestBuiltin_AxiomWithGroundedByPasses(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "axiom", note.Link{TargetID: "c1", Type: "grounded-by"}),
		n("c1", note.TypeConcept, "axiom"),
	}
	v := runBuiltin(t, notes)
	if anyContains(v, "root", "grounded-by") {
		t.Fatalf("did not expect grounded-by violation, got %v", v)
	}
}

// A cycle within a representation subgraph must be flagged.
func TestBuiltin_CycleFlagged(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "ontology", note.Link{TargetID: "c1", Type: "refines"}),
		n("c1", note.TypeConcept, "ontology", note.Link{TargetID: "root", Type: "refines"}),
	}
	v := runBuiltin(t, notes)
	if !anyContains(v, "", "cycle") && !anyContainsReason(v, "cycle") {
		t.Fatalf("expected cycle violation, got %v", v)
	}
}

// Links to notes with a different representation are not traversed: the model
// root's own subgraph stays clean even though the target is a bad type. (The
// target, as its own singleton subgraph, is a separate global finding — that is
// correct global semantics, so we assert only that the cross-rep edge did not
// pull "other" into root's ontology subgraph as a bad child.)
func TestBuiltin_CrossRepLinkNotTraversed(t *testing.T) {
	notes := []*note.Note{
		n("root", note.TypeModel, "ontology", note.Link{TargetID: "other", Type: "supports"}),
		n("other", note.TypeObservation, "taxonomy"),
	}
	v := runBuiltin(t, notes)
	// "other" must NOT be flagged as a non-root bad child (that would mean the
	// ontology traversal crossed the representation boundary).
	if anyContains(v, "other", "non-root") {
		t.Fatalf("cross-rep link was traversed into ontology subgraph: %v", v)
	}
	// root itself is a valid model root — no violation for root.
	if anyContainsID(v, "root") {
		t.Fatalf("valid model root should not be flagged: %v", v)
	}
}

func anyContains(vs []string, idPart, reasonPart string) bool {
	for _, v := range vs {
		parts := strings.SplitN(v, "|", 2)
		if len(parts) == 2 && parts[0] == idPart && strings.Contains(parts[1], reasonPart) {
			return true
		}
	}
	return false
}

func anyContainsID(vs []string, idPart string) bool {
	for _, v := range vs {
		if strings.HasPrefix(v, idPart+"|") {
			return true
		}
	}
	return false
}

func anyContainsReason(vs []string, reasonPart string) bool {
	for _, v := range vs {
		if strings.Contains(v, reasonPart) {
			return true
		}
	}
	return false
}
