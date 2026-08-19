package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedNotebookFiles writes n plain .md notes into dir plus a daily note and,
// when withGoverns is true, protocol notes with governs edges + nn-rule blocks
// so the --global path exercises the rules engine (governingProtocolsFromEngine)
// rather than skipping it.
// seedNotebookFiles writes n notes. When withGoverns is true it models a
// notebook whose --global path exercises the datalog engine the way profiling a
// real notebook showed: many notes carry ```nn-rule``` blocks with multi-literal
// rule bodies, and a chain of protocol notes governs (and refines) so the
// builtin recursive rules (transitively_governs / reachable) do real fixpoint
// work. Files use the real <id>-slug.md naming so findByID/Read resolve.
func seedNotebookFiles(b *testing.B, dir string, n int, withGoverns bool) {
	b.Helper()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			b.Fatalf("seed write %s: %v", name, err)
		}
	}
	id := func(i int) string { return fmt.Sprintf("20260101%06d-%04d", i, i%10000) }
	for i := range n {
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("id: " + id(i) + "\n")
		fmt.Fprintf(&sb, "title: 'Seed note %d'\n", i)
		switch {
		case withGoverns && i%10 == 0:
			// Protocol note: governs the next note and refines the previous
			// protocol, building refines chains the recursive builtins close.
			sb.WriteString("type: protocol\nstatus: permanent\n")
			sb.WriteString("links:\n")
			fmt.Fprintf(&sb, "    - to: %s\n      type: governs\n      annotation: g\n", id((i+1)%n))
			if i >= 10 {
				fmt.Fprintf(&sb, "    - to: %s\n      type: refines\n      annotation: r\n", id(i-10))
			}
			sb.WriteString("---\n\nProtocol body.\n\n```nn-rule\n")
			// Multi-literal rule body so evalBody/unify do real join work.
			fmt.Fprintf(&sb, "flagged%d(X) :- note(X, \"observation\", \"draft\"), link(X, _, \"refines\"), tag(X, \"alpha\").\n```\n", i)
		case withGoverns && i%3 == 0:
			sb.WriteString("type: observation\nstatus: draft\ntags:\n    - alpha\n---\n\n")
			sb.WriteString("Body.\n\n```nn-rule\n")
			fmt.Fprintf(&sb, "derived%d(X) :- note(X, _, _), tag(X, \"alpha\").\n```\n", i)
		default:
			sb.WriteString("type: observation\nstatus: draft\n---\n\nBody paragraph with realistic prose.\n")
		}
		write(id(i)+"-seed-note.md", sb.String())
	}
	// A daily note governed by protocol 0, so --global's daily render calls
	// governingProtocolsFromEngine and triggers the fixpoint over all facts.
	if withGoverns {
		daily := "20260101999999-0001"
		var db strings.Builder
		db.WriteString("---\nid: " + daily + "\n")
		db.WriteString("title: 'Daily: 2026-01-01'\ntype: observation\nstatus: draft\ntags:\n    - daily\n---\n\n## Relay\n\nseeded.\n")
		write(daily+"-daily.md", db.String())
	}
}

// seedRepresentationNotebook models the recursive-representation cost the target
// notebook has: notes carry a representation: ontology|axiom|taxonomy field and
// form same-representation refines/extends chains, so the builtin recursive
// rules (rep_link -> rep_reach transitive closure, run over all facts on every
// nn show --global) do the deep fixpoint work that profiling showed dominates.
func seedRepresentationNotebook(b *testing.B, dir string, n int) {
	b.Helper()
	reps := []string{"ontology", "taxonomy", "axiom"}
	id := func(i int) string { return fmt.Sprintf("20260101%06d-%04d", i, i%10000) }
	// Build several representation subgraphs; within each, notes chain via
	// refines/extends to the next, forming a path the transitive closure walks.
	subgraphSize := 25
	for i := range n {
		rep := reps[(i/subgraphSize)%len(reps)]
		posInSub := i % subgraphSize
		var sb strings.Builder
		sb.WriteString("---\nid: " + id(i) + "\n")
		fmt.Fprintf(&sb, "title: 'Rep note %d'\n", i)
		// Root of each subgraph is a model; others are concepts.
		if posInSub == 0 {
			sb.WriteString("type: model\n")
		} else {
			sb.WriteString("type: concept\n")
		}
		sb.WriteString("status: reviewed\n")
		fmt.Fprintf(&sb, "representation: %s\n", rep)
		// Same-representation refines link to the previous note in the subgraph
		// (roots have none), so rep_link fires and rep_reach closes the chain.
		if posInSub != 0 {
			sb.WriteString("links:\n")
			linkType := "refines"
			if posInSub%2 == 0 {
				linkType = "extends"
			}
			fmt.Fprintf(&sb, "    - to: %s\n      type: %s\n      annotation: same-rep\n", id(i-1), linkType)
		}
		sb.WriteString("---\n\nRepresentation subgraph node body.\n")
		if err := os.WriteFile(filepath.Join(dir, id(i)+"-rep-note.md"), []byte(sb.String()), 0o644); err != nil {
			b.Fatalf("seed rep write: %v", err)
		}
	}
	// Daily note so --global renders and runs the engine over the rep facts.
	daily := "20260101999999-0002"
	os.WriteFile(filepath.Join(dir, daily+"-daily.md"),
		[]byte("---\nid: "+daily+"\ntitle: 'Daily: 2026-01-01'\ntype: observation\nstatus: draft\ntags:\n    - daily\n---\n\n## Relay\n\nseeded.\n"), 0o644)
}

// BenchmarkShowGlobal measures the full nn show --global command over a seeded
// notebook. It captures both the backend.List() parse cost and, when governing
// edges exist, the rules-engine (datalog) cost of governingProtocolsFromEngine.
// Run: go test ./cmd/nn/cmd -bench=ShowGlobal -benchmem -run '^$'
func BenchmarkShowGlobal(b *testing.B) {
	// mode: "plain" (parse-only), "governs" (nn-rule blocks + governs chains),
	// "reps" (ontology/axiom/taxonomy representation subgraphs — the recursive
	// rep_reach transitive-closure workload the target notebook has).
	for _, tc := range []struct {
		n    int
		mode string
	}{
		{100, "plain"}, {600, "plain"}, {2000, "plain"},
		{600, "governs"}, {2000, "governs"},
		{600, "reps"}, {2000, "reps"},
	} {
		name := fmt.Sprintf("n=%d/mode=%s", tc.n, tc.mode)
		b.Run(name, func(b *testing.B) {
			nbDir := b.TempDir()
			cfgDir := b.TempDir()
			b.Setenv("NN_CONFIG_DIR", cfgDir)
			cfgFile := filepath.Join(cfgDir, "config.toml")
			os.WriteFile(cfgFile, []byte(fmt.Sprintf(`
[notebooks]
default = "test"

[notebooks.test]
path = %q
backend = "gitlocal"
`, nbDir)), 0o644)
			switch tc.mode {
			case "reps":
				seedRepresentationNotebook(b, nbDir, tc.n)
			default:
				seedNotebookFiles(b, nbDir, tc.n, tc.mode == "governs")
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var out bytes.Buffer
				root := NewRootCmdForTest(cfgFile)
				root.SetOut(&out)
				root.SetErr(io.Discard)
				root.SetArgs([]string{"show", "--global"})
				if err := root.Execute(); err != nil {
					b.Fatalf("show --global: %v", err)
				}
			}
		})
	}
}

// BenchmarkShowGlobalLive measures the production command path against the
// notebook selected by the caller's normal nn configuration. It is opt-in to
// keep tests hermetic and to avoid reading a developer's notebook in CI.
//
// Run benchmark + allocations:
//
//	NN_BENCH_LIVE=1 go test ./cmd/nn/cmd -run '^$' -bench '^BenchmarkShowGlobalLive$' -benchmem
//
// Run CPU + heap profiles:
//
//	NN_BENCH_LIVE=1 go test ./cmd/nn/cmd -run '^$' -bench '^BenchmarkShowGlobalLive$' -benchtime=1x -cpuprofile cpu.out -memprofile mem.out
func BenchmarkShowGlobalLive(b *testing.B) {
	if os.Getenv("NN_BENCH_LIVE") != "1" {
		b.Skip("set NN_BENCH_LIVE=1 to benchmark the configured live notebook")
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		root := NewRootCmdForTest("")
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		root.SetArgs([]string{"show", "--global"})
		if err := root.Execute(); err != nil {
			b.Fatalf("show --global against live notebook: %v", err)
		}
	}
}
