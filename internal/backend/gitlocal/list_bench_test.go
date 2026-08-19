package gitlocal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedNotebook writes n plain .md notes into dir. When withRules is true, a
// fraction of the notes carry a governs link and an ```nn-rule``` fenced block,
// so benchmarks can exercise the rules/datalog path that nn show --global takes
// when governing edges exist (the branch that otherwise skips Eval()).
func seedNotebook(b *testing.B, dir string, n int, withRules bool) {
	b.Helper()
	for i := range n {
		id := fmt.Sprintf("20260101%06d-%04d", i, i%10000)
		typ := "observation"
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("id: " + id + "\n")
		fmt.Fprintf(&sb, "title: 'Seed note %d with a moderately long title for realistic parse cost'\n", i)
		sb.WriteString("type: " + typ + "\n")
		sb.WriteString("status: draft\n")
		if withRules && i%20 == 0 {
			// A protocol note with a governs link + a datalog rule block.
			sb.Reset()
			sb.WriteString("---\n")
			sb.WriteString("id: " + id + "\n")
			fmt.Fprintf(&sb, "title: 'Protocol seed %d'\n", i)
			sb.WriteString("type: protocol\n")
			sb.WriteString("status: permanent\n")
			sb.WriteString("links:\n")
			fmt.Fprintf(&sb, "    - to: 20260101%06d-%04d\n", (i+1)%n, (i+1)%10000)
			sb.WriteString("      type: governs\n")
			sb.WriteString("      annotation: seeded\n")
			sb.WriteString("---\n\n")
			sb.WriteString("Body paragraph.\n\n")
			sb.WriteString("```nn-rule\n")
			fmt.Fprintf(&sb, "flagged%d(X) :- note(X, \"observation\", \"draft\").\n", i)
			sb.WriteString("```\n")
		} else {
			sb.WriteString("---\n\n")
			sb.WriteString("Body paragraph one with some realistic prose so the parser does real work.\n\n")
			sb.WriteString("Body paragraph two referencing seed material and more tokens for BM25.\n")
		}
		path := filepath.Join(dir, id+".md")
		if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
			b.Fatalf("seed write: %v", err)
		}
	}
}

// BenchmarkBackendList measures the cost of loading and parsing the whole
// notebook — the dominant cost of nn show --global, which calls List() and then
// filters. Run scaled: go test -bench=BackendList -benchmem
func BenchmarkBackendList(b *testing.B) {
	for _, n := range []int{100, 600, 2000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			dir := b.TempDir()
			b.Setenv("NN_CONFIG_DIR", b.TempDir())
			seedNotebook(b, dir, n, false)
			backend, err := New(dir)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				notes, err := backend.List()
				if err != nil {
					b.Fatalf("List: %v", err)
				}
				if len(notes) == 0 {
					b.Fatalf("List returned 0 notes")
				}
			}
		})
	}
}

// BenchmarkBackendListWithRules seeds a notebook that carries governs edges and
// datalog rule blocks, so the number reflects a notebook where nn show --global
// does NOT skip the rules engine. Compare against BenchmarkBackendList to
// isolate the parse cost from the datalog/Eval cost.
func BenchmarkBackendListWithRules(b *testing.B) {
	for _, n := range []int{100, 600, 2000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			dir := b.TempDir()
			b.Setenv("NN_CONFIG_DIR", b.TempDir())
			seedNotebook(b, dir, n, true)
			backend, err := New(dir)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				notes, err := backend.List()
				if err != nil {
					b.Fatalf("List: %v", err)
				}
				if len(notes) == 0 {
					b.Fatalf("List returned 0 notes")
				}
			}
		})
	}
}
