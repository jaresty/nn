package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func makeRepNote(id, title string, typ note.Type, rep string) *note.Note {
	n := newTestNoteForCLI(id, title, typ)
	n.Representation = rep
	return n
}

// property [3]: root must be type: model
func TestCheckGraphRootMustBeModel(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeConcept, "ontology")
	writeNoteFile(t, nbDir, root)

	_, err := execute("check", root.ID)
	if err == nil {
		t.Errorf("nn check: expected failure when root is not type:model, got nil")
	}
}

// property [3] passing: root type:model passes root type check
func TestCheckGraphRootModelPasses(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeModel, "ontology")
	writeNoteFile(t, nbDir, root)

	out, err := execute("check", root.ID)
	if err != nil {
		t.Errorf("nn check: expected pass when root is type:model, got: %v\n%s", err, out)
	}
}

// property [2]: non-root nodes must be type:concept or type:argument
func TestCheckGraphChildMustBeConceptOrArgument(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeModel, "ontology")
	child := makeRepNote(note.GenerateID(), "Bad Child", note.TypeModel, "ontology")
	root.Links = []note.Link{{TargetID: child.ID, Type: "extends", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	_, err := execute("check", root.ID)
	if err == nil {
		t.Errorf("nn check: expected failure when non-root child is type:model, got nil")
	}
}

// property [2] passing: children of type:concept pass node type check
func TestCheckGraphChildConceptPasses(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeModel, "ontology")
	child := makeRepNote(note.GenerateID(), "Concept Child", note.TypeConcept, "ontology")
	root.Links = []note.Link{{TargetID: child.ID, Type: "extends", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("check", root.ID)
	if err != nil {
		t.Errorf("nn check: expected pass with concept child, got: %v\n%s", err, out)
	}
}

// property [1]: cycle in same-representation subgraph fails
func TestCheckGraphCycleFails(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeModel, "ontology")
	child := makeRepNote(note.GenerateID(), "Concept Child", note.TypeConcept, "ontology")
	root.Links = []note.Link{{TargetID: child.ID, Type: "extends", Annotation: ""}}
	child.Links = []note.Link{{TargetID: root.ID, Type: "extends", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	_, err := execute("check", root.ID)
	if err == nil {
		t.Errorf("nn check: expected failure on cycle, got nil")
	}
}

// property [4]: taxonomy links must be refines or extends
func TestCheckGraphTaxonomyLinkTypeFails(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Taxonomy Root", note.TypeModel, "taxonomy")
	child := makeRepNote(note.GenerateID(), "Category", note.TypeConcept, "taxonomy")
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	_, err := execute("check", root.ID)
	if err == nil {
		t.Errorf("nn check taxonomy: expected failure when link type is not refines/extends, got nil")
	}
}

// property [4] passing: taxonomy with refines link passes
func TestCheckGraphTaxonomyRefinesPasses(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Taxonomy Root", note.TypeModel, "taxonomy")
	child := makeRepNote(note.GenerateID(), "Category", note.TypeConcept, "taxonomy")
	root.Links = []note.Link{{TargetID: child.ID, Type: "refines", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("check", root.ID)
	if err != nil {
		t.Errorf("nn check taxonomy: expected pass with refines link, got: %v\n%s", err, out)
	}
}

// property [5]: axiom root must have at least one grounded-by link within same-representation subgraph
func TestCheckGraphAxiomRequiresGroundedBy(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Axiom Root", note.TypeModel, "axiom")
	child := makeRepNote(note.GenerateID(), "Axiom Child", note.TypeConcept, "axiom")
	root.Links = []note.Link{{TargetID: child.ID, Type: "extends", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	_, err := execute("check", root.ID)
	if err == nil {
		t.Errorf("nn check axiom: expected failure when root has no grounded-by link, got nil")
	}
}

// property [5] passing: axiom root with grounded-by passes
func TestCheckGraphAxiomGroundedByPasses(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Axiom Root", note.TypeModel, "axiom")
	child := makeRepNote(note.GenerateID(), "Axiom Child", note.TypeConcept, "axiom")
	root.Links = []note.Link{{TargetID: child.ID, Type: "grounded-by", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("check", root.ID)
	if err != nil {
		t.Errorf("nn check axiom: expected pass with grounded-by link, got: %v\n%s", err, out)
	}
}

// property [6]: links to notes with different representation are not traversed
func TestCheckGraphSkipsDifferentRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeModel, "ontology")
	outsider := makeRepNote(note.GenerateID(), "Taxonomy Note", note.TypeModel, "taxonomy")
	root.Links = []note.Link{{TargetID: outsider.ID, Type: "extends", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, outsider)

	// outsider has different representation — should not be traversed, so root passes
	out, err := execute("check", root.ID)
	if err != nil {
		t.Errorf("nn check: expected pass when linked note has different representation (not traversed), got: %v\n%s", err, out)
	}
}

// property [6b]: links to notes with no representation are not traversed
func TestCheckGraphSkipsNoRepresentation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	root := makeRepNote(note.GenerateID(), "Ontology Root", note.TypeModel, "ontology")
	plain := newTestNoteForCLI(note.GenerateID(), "Plain Note", note.TypeConcept)
	root.Links = []note.Link{{TargetID: plain.ID, Type: "extends", Annotation: ""}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, plain)

	out, err := execute("check", root.ID)
	if err != nil {
		t.Errorf("nn check: expected pass when linked note has no representation (not traversed), got: %v\n%s", err, out)
	}
}
