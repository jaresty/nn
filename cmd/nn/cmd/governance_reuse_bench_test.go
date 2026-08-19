package cmd

import (
	"fmt"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// benchGovNotes builds N representation-subgraph notes (ontology/taxonomy/axiom
// chains) so the recursive rep_reach closure does real fixpoint work — the
// workload that dominated the target notebook's nn show.
func benchGovNotes(n int) []*note.Note {
	reps := []string{"ontology", "taxonomy", "axiom"}
	all := make([]*note.Note, 0, n)
	for i := range n {
		id := fmt.Sprintf("n%05d", i)
		nt := &note.Note{ID: id, Title: fmt.Sprintf("Note %d", i), Type: note.TypeConcept, Status: "reviewed", Representation: reps[(i/25)%3]}
		if i%25 == 0 {
			nt.Type = note.TypeModel
		} else {
			lt := "refines"
			if i%2 == 0 {
				lt = "extends"
			}
			nt.Links = []note.Link{{TargetID: fmt.Sprintf("n%05d", i-1), Type: lt, Annotation: "same-rep"}}
		}
		all = append(all, nt)
	}
	return all
}

// BenchmarkGovernancePerNoteEval is the OLD pattern: rebuild the engine and
// re-run the fixpoint for every shown note (one-shot governingProtocolsFromEngine
// per note). Cost = notesShown × full-fixpoint.
func BenchmarkGovernancePerNoteEval(b *testing.B) {
	all := benchGovNotes(600)
	targets := []string{"n00003", "n00007", "n00030", "n00120", "n00450"} // 5 shown notes
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, id := range targets {
			_ = governingProtocolsFromEngine(id, all)
		}
	}
}

// BenchmarkGovernanceEngineReuse is the NEW pattern (Layer 1): build+Eval once,
// query per note. Cost = one fixpoint + N cheap queries.
func BenchmarkGovernanceEngineReuse(b *testing.B) {
	all := benchGovNotes(600)
	targets := []string{"n00003", "n00007", "n00030", "n00120", "n00450"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ge := buildGovernanceEngine(all)
		for _, id := range targets {
			_ = governedBy(ge, id)
		}
	}
}
