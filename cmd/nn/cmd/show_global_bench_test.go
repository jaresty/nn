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
func seedNotebookFiles(b *testing.B, dir string, n int, withGoverns bool) {
	b.Helper()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			b.Fatalf("seed write %s: %v", name, err)
		}
	}
	for i := range n {
		id := fmt.Sprintf("20260101%06d-%04d", i, i%10000)
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("id: " + id + "\n")
		fmt.Fprintf(&sb, "title: 'Seed note %d'\n", i)
		if withGoverns && i%20 == 0 {
			sb.WriteString("type: protocol\nstatus: permanent\n")
			sb.WriteString("links:\n")
			fmt.Fprintf(&sb, "    - to: 20260101%06d-%04d\n", (i+1)%n, (i+1)%10000)
			sb.WriteString("      type: governs\n      annotation: seeded\n---\n\n")
			sb.WriteString("Body.\n\n```nn-rule\n")
			fmt.Fprintf(&sb, "flagged%d(X) :- note(X, \"observation\", \"draft\").\n```\n", i)
		} else {
			sb.WriteString("type: observation\nstatus: draft\n---\n\nBody paragraph with realistic prose.\n")
		}
		write(id+".md", sb.String())
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
