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

// BenchmarkShowGlobal measures the full nn show --global command over a seeded
// notebook. It captures both the backend.List() parse cost and, when governing
// edges exist, the rules-engine (datalog) cost of governingProtocolsFromEngine.
// Run: go test ./cmd/nn/cmd -bench=ShowGlobal -benchmem -run '^$'
func BenchmarkShowGlobal(b *testing.B) {
	for _, tc := range []struct {
		n       int
		governs bool
	}{
		{100, false}, {600, false}, {2000, false},
		{600, true}, {2000, true},
	} {
		name := fmt.Sprintf("n=%d/governs=%v", tc.n, tc.governs)
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
			seedNotebookFiles(b, nbDir, tc.n, tc.governs)

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
