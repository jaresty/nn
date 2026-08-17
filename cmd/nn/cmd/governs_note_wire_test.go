package cmd

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// property [4]: the engine-backed governance query returns a superset of the
// legacy findGoverningProtocols — every protocol the old function found is still
// present (whole-tree preserved), and specialization adds more.
func TestGovernsNoteSupersetsFindGoverning(t *testing.T) {
	// Whole-tree: protocol P governs model root R; child C is in R's ontology
	// subtree via a non-refines edge (supports). findGoverningProtocols catches
	// this via representation-root climbing.
	notes := []*note.Note{
		{ID: "p", Type: note.TypeProtocol, Status: note.StatusPermanent,
			Links: []note.Link{{TargetID: "r", Type: "governs"}}},
		{ID: "r", Type: note.TypeModel, Status: note.StatusReviewed, Representation: "ontology",
			Links: []note.Link{{TargetID: "c", Type: "supports"}}},
		{ID: "c", Type: note.TypeConcept, Status: note.StatusReviewed, Representation: "ontology"},
	}

	assertSuperset(t, notes, "r", "c")
}

// property [4] over the edge-case fixtures (ambiguous/cyclic representation
// ancestry) that findGoverningProtocols handles specially: the engine must not
// drop any protocol the legacy function returns. (It may return more — legacy
// silently drops inherited governance on malformed structure; the engine
// surfaces it, which still satisfies the superset property.)
func TestGovernsNoteSupersetsEdgeCases(t *testing.T) {
	proto := func(id, tgt, typ string) *note.Note {
		return &note.Note{ID: id, Type: note.TypeProtocol, Status: note.StatusReviewed,
			Links: []note.Link{{TargetID: tgt, Type: typ}}}
	}
	// ambiguous: concept with two model parents; direct + inherited governors.
	amb := &note.Note{ID: "ambiguous", Type: note.TypeConcept, Status: note.StatusReviewed, Representation: "ontology"}
	pA := &note.Note{ID: "parent-a", Type: note.TypeModel, Status: note.StatusReviewed, Representation: "ontology",
		Links: []note.Link{{TargetID: "ambiguous", Type: "extends"}}}
	pB := &note.Note{ID: "parent-b", Type: note.TypeModel, Status: note.StatusReviewed, Representation: "ontology",
		Links: []note.Link{{TargetID: "ambiguous", Type: "extends"}}}
	notes := []*note.Note{amb, pA, pB,
		proto("proto-f", "parent-a", "governs"), // inherited via parent-a
		proto("proto-g", "ambiguous", "governs"), // direct
	}
	assertSuperset(t, notes, "ambiguous")
}

func assertSuperset(t *testing.T, notes []*note.Note, targets ...string) {
	t.Helper()
	for _, target := range targets {
		legacy := findGoverningProtocols(target, notes)
		engine := governingProtocolsFromEngine(target, notes)
		engineSet := map[string]bool{}
		for _, n := range engine {
			engineSet[n.ID] = true
		}
		for _, n := range legacy {
			if !engineSet[n.ID] {
				t.Errorf("target %s: engine governance dropped protocol %s that findGoverningProtocols returned", target, n.ID)
			}
		}
	}
}
