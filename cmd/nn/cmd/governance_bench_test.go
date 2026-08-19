package cmd

import (
	"fmt"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// BenchmarkGoverningProtocolsFromEngine isolates the datalog cost that nn show
// --global pays via governingProtocolsFromEngine: FactsFromNotes(all) + builtin
// rules + full Eval() fixpoint, over an in-memory notebook of N notes. This is
// the "builtin datalog rules over all facts" path — compare its numbers against
// BenchmarkBackendList to see whether Eval or parse dominates on a given corpus.
func BenchmarkGoverningProtocolsFromEngine(b *testing.B) {
	for _, n := range []int{100, 600, 2000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			all := make([]*note.Note, 0, n)
			for i := range n {
				id := fmt.Sprintf("20260101%06d-%04d", i, i%10000)
				nt := &note.Note{ID: id, Title: fmt.Sprintf("Seed %d", i), Type: note.TypeObservation, Status: "draft"}
				if i%20 == 0 {
					nt.Type = note.TypeProtocol
					nt.Status = "permanent"
					nt.Links = []note.Link{{TargetID: fmt.Sprintf("20260101%06d-%04d", (i+1)%n, (i+1)%10000), Type: "governs", Annotation: "seeded"}}
				}
				all = append(all, nt)
			}
			target := all[len(all)-1].ID
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = governingProtocolsFromEngine(target, all)
			}
		})
	}
}
