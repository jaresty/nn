package trace_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/trace"
)

// ─────────────────────────────────────────────────────────────────────────────
// Performance benchmarks for the trace call-graph annotation path.
//
// trace.Trace annotates each visited node with BM25-ranked nn notes and, for
// each node, re-parses the node's source file to extract its calls. Both are
// per-node costs that grow with graph size × corpus size. This suite captures a
// baseline so an optimization (memoized scoring, cached parse trees) can be
// compared:
//
//	go test ./internal/trace/ -run '^$' -bench BenchmarkTrace -benchmem
// ─────────────────────────────────────────────────────────────────────────────

var traceNoteSizes = []int{200, 600}

// benchTraceTree writes a small Go package with a chain of calling functions so
// Trace performs real DFS work: f0 calls f1 calls f2 ... plus a shared helper
// each level calls, exercising cycle/dedup handling.
func benchTraceTree(b *testing.B, chain int) string {
	b.Helper()
	dir := b.TempDir()
	var body strings.Builder
	body.WriteString("package server\n\n")
	body.WriteString("func helper() { validateToken() }\n")
	body.WriteString("func validateToken() {}\n\n")
	for i := 0; i < chain; i++ {
		if i == chain-1 {
			fmt.Fprintf(&body, "func f%d() { helper() }\n", i)
		} else {
			fmt.Fprintf(&body, "func f%d() { f%d(); helper() }\n", i, i+1)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "server.go"), []byte(body.String()), 0o644); err != nil {
		b.Fatal(err)
	}
	return dir
}

// benchNotes synthesizes n notes whose vocabulary overlaps the traced source so
// the BM25 annotation path produces non-empty results.
func benchNotes(n int) []*note.Note {
	notes := make([]*note.Note, n)
	for i := 0; i < n; i++ {
		notes[i] = &note.Note{
			ID:    fmt.Sprintf("2026%08d-%04d", i, i%10000),
			Title: fmt.Sprintf("Note %d on auth and routing", i),
			Body:  fmt.Sprintf("Note %d discusses helper validateToken session middleware routing %d.", i, i%13),
		}
	}
	return notes
}

// BenchmarkTraceWithNotes measures trace.Trace including per-node BM25
// annotation and per-node source re-parse, across corpus size.
func BenchmarkTraceWithNotes(b *testing.B) {
	dir := benchTraceTree(b, 8)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		b.Fatal(err)
	}
	for _, n := range traceNoteSizes {
		notes := benchNotes(n)
		annotate := benchAnnotator(notes)
		b.Run(fmt.Sprintf("notes=%d", n), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = trace.Trace(idx, []string{"f0"}, 3, annotate)
			}
		})
	}
}

// benchAnnotator builds a trace.Annotator backed by the memoizing per-field
// scorer, mirroring the production nn grep/trace annotation path.
func benchAnnotator(notes []*note.Note) trace.Annotator {
	scorer := note.NewCorpusScorer(notes, note.BM25FieldIDF(notes, nil), nil, nil)
	return func(query string) []trace.NoteRef {
		scores := scorer.Score(notes, query)
		var refs []trace.NoteRef
		for _, nt := range notes {
			if scores[nt.ID] > 0 {
				refs = append(refs, trace.NoteRef{ID: nt.ID, Title: nt.Title})
				if len(refs) == 2 {
					break
				}
			}
		}
		return refs
	}
}

// BenchmarkTraceNoNotes isolates the DFS + per-node re-parse cost without the
// BM25 annotation path (notes=nil).
func BenchmarkTraceNoNotes(b *testing.B) {
	dir := benchTraceTree(b, 8)
	idx, err := trace.BuildIndex(dir)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = trace.Trace(idx, []string{"f0"}, 3, nil)
	}
}
