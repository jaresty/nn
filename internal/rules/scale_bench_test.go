package rules

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// ─────────────────────────────────────────────────────────────────────────────
// Reusable scale-benchmark suite for the rules pipeline.
//
// A single generator (genNotebook) synthesizes N realistic notes as Markdown
// documents, and four layered sub-benchmarks share it so the cost of each stage
// of `nn rules check` is separately attributable:
//
//	Parse           — note.Parse over the raw Markdown (frontmatter + link parse)
//	FactsFromNotes  — the parsed notes → Datalog facts bridge
//	Eval            — Engine.Eval over the facts with the builtin ruleset
//	EndToEnd        — Parse + FactsFromNotes + Eval, the inner path of rules check
//
// Run with:
//
//	go test ./internal/rules/ -run '^$' -bench BenchmarkRulesScale -benchmem
//
// Reuse this for any future rules-pipeline perf work: compare -benchmem output
// before/after a change, or add a new sub-benchmark for a new stage.
// ─────────────────────────────────────────────────────────────────────────────

// The parse and fact-bridge layers are near-linear, so they are benchmarked at
// large sizes where per-note cost is meaningful. The Eval and end-to-end layers
// run the recursive builtin ruleset, whose transitive closures are superlinear
// (reachable/rep_reach over a chain derive O(n^2) facts across many fixpoint
// rounds), so they are benchmarked at small sizes to stay within seconds. Adjust
// these lists when profiling a specific layer.
var (
	parseSizes = []int{500, 2000}
	evalSizes  = []int{30, 60}
)

// genNotebook returns n Markdown note documents forming a refines chain
// n0 → n1 → … → n(n-1), each with realistic frontmatter (id/type/status/tags/
// representation) and a couple of open items. The chain drives the builtin
// recursive rules (reachable, rep_reach, transitively_governs) so Eval does real
// transitive-closure work.
func genNotebook(n int) [][]byte {
	docs := make([][]byte, n)
	for i := range n {
		var b strings.Builder
		id := fmt.Sprintf("2026%08d-%04d", i, i%10000)
		b.WriteString("---\n")
		fmt.Fprintf(&b, "id: %s\n", id)
		fmt.Fprintf(&b, "title: Note %d\n", i)
		b.WriteString("type: concept\n")
		b.WriteString("status: reviewed\n")
		fmt.Fprintf(&b, "tags:\n  - t%d\n  - scale\n", i%7)
		b.WriteString("representation: taxonomy\n")
		b.WriteString("---\n")
		fmt.Fprintf(&b, "Body of note %d.\n\n", i)
		b.WriteString("- [ ] an open item\n")
		b.WriteString("- [x] a done item\n")
		if i+1 < n {
			next := fmt.Sprintf("2026%08d-%04d", i+1, (i+1)%10000)
			b.WriteString("\n## Links\n")
			fmt.Fprintf(&b, "- [[%s]] [refines] {reviewed} — chain link\n", next)
		}
		docs[i] = []byte(b.String())
	}
	return docs
}

// parseAll parses every generated doc, failing the benchmark if any note is
// dropped — a malformed generator would otherwise make every layer measure a
// degenerate workload (property [6]/[7]).
func parseAll(tb testing.TB, docs [][]byte) []*note.Note {
	notes := make([]*note.Note, 0, len(docs))
	for i, d := range docs {
		n, err := note.Parse(d)
		if err != nil {
			tb.Fatalf("parse doc %d: %v", i, err)
		}
		notes = append(notes, n)
	}
	if len(notes) != len(docs) {
		tb.Fatalf("parsed %d notes, want %d", len(notes), len(docs))
	}
	return notes
}

// evalEngine builds an engine over facts + the builtin rules and evaluates it.
func evalEngine(tb testing.TB, facts []Fact) *Engine {
	e := NewEngine()
	for _, f := range facts {
		e.AddFact(f)
	}
	rs, err := ParseProgram(BuiltinRules())
	if err != nil {
		tb.Fatalf("parse builtin rules: %v", err)
	}
	e.AddRules(rs)
	if err := e.Eval(); err != nil {
		tb.Fatalf("eval: %v", err)
	}
	return e
}

func BenchmarkRulesScale(b *testing.B) {
	// Near-linear layers: parsing Markdown and bridging notes → facts.
	for _, n := range parseSizes {
		docs := genNotebook(n)

		// Self-check once, outside the timed loops (property [6]/[7]): a
		// degenerate generator fails fast rather than reporting empty timings.
		notes := parseAll(b, docs)
		facts := FactsFromNotes(notes)
		if len(facts) < n {
			b.Fatalf("n=%d: derived %d base facts, want >= %d", n, len(facts), n)
		}

		b.Run(fmt.Sprintf("Parse/n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parseAll(b, docs)
			}
		})

		b.Run(fmt.Sprintf("FactsFromNotes/n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = FactsFromNotes(notes)
			}
		})
	}

	// Superlinear layers: the Datalog fixpoint and the full inner path.
	for _, n := range evalSizes {
		docs := genNotebook(n)
		notes := parseAll(b, docs)
		facts := FactsFromNotes(notes)

		// Self-check the eval workload derives real transitive facts.
		e := evalEngine(b, facts)
		if got := len(e.Query("reachable")); got < n {
			b.Fatalf("n=%d: %d reachable facts, want >= %d", n, got, n)
		}

		b.Run(fmt.Sprintf("Eval/n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				evalEngine(b, facts) // fresh engine; Eval mutates the fact base
			}
		})

		b.Run(fmt.Sprintf("EndToEnd/n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parsed := parseAll(b, docs)
				evalEngine(b, FactsFromNotes(parsed))
			}
		})
	}
}
