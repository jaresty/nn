package rules

import (
	"fmt"
	"testing"
)

// benchNotebook builds a representative fact base that mimics a real notebook:
// many notes across several predicates (note/link/tag/representation) plus a
// refines chain that the builtin recursive rules (reachable, rep_reach,
// transitively_governs) close transitively — the same predicate mix and rule
// set nn rules check runs.
func benchNotebook(n int) *Engine {
	e := NewEngine()
	for i := range n {
		id := fmt.Sprintf("n%d", i)
		e.AddFact(Fact{Pred: "note", Args: []string{id, "concept", "reviewed"}})
		e.AddFact(Fact{Pred: "tag", Args: []string{id, fmt.Sprintf("t%d", i%7)}})
		e.AddFact(Fact{Pred: "representation", Args: []string{id, "taxonomy"}})
		if i+1 < n {
			// refines chain: drives reachable + transitively_governs + rep_reach.
			e.AddFact(Fact{Pred: "link", Args: []string{id, fmt.Sprintf("n%d", i+1), "refines"}})
		}
	}
	rs, err := ParseProgram(BuiltinRules())
	if err != nil {
		panic(err)
	}
	e.AddRules(rs)
	return e
}

// BenchmarkEval measures ns/op and (with -benchmem) allocs/op for a full Eval
// over a representative notebook, running the complete builtin ruleset — the
// same work nn rules check performs. It is an allocation/regression baseline for
// the engine core, NOT an A/B for the predicate index: a synthetic engine-only
// workload at this scale does not reproduce the end-to-end nn rules check
// speedup (see note 20260814145701-2860). n is kept small because the recursive
// closures are superlinear — n=300 takes minutes per op.
func BenchmarkEval(b *testing.B) {
	const n = 60

	// Confirm the workload actually drives the recursive join path: a degenerate
	// fact base would make the benchmark measure an empty fixpoint.
	warm := benchNotebook(n)
	if err := warm.Eval(); err != nil {
		b.Fatalf("eval: %v", err)
	}
	if got := len(queryKeys(warm, "reachable")); got < n {
		b.Fatalf("benchmark workload too small: derived %d reachable facts, want >= %d", got, n)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		e := benchNotebook(n) // fresh engine per iteration; Eval mutates the fact base
		b.StartTimer()
		if err := e.Eval(); err != nil {
			b.Fatalf("eval: %v", err)
		}
	}
}
