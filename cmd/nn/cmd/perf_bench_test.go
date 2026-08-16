package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Cross-command performance benchmark suite for the note-annotating commands.
//
// Several commands annotate their output with BM25-ranked "related notes":
// nn grep, nn read, nn ast, nn tee, nn shuf, nn fetch, nn list --search, and
// others. They share one hot path — RankedByQuery (cmd/nn/cmd/search_helper.go)
// — which rebuilds the corpus link maps and calls index.GetOrComputeFieldIDFPath
// on every call. grep/ast/shuf call it once PER MATCH/SAMPLE, so the invariant
// corpus work (SQLite open, git rev-parse, whole-corpus SHA-256 cache key, link
// maps) is recomputed N times per invocation.
//
// This suite drives the real command path through the same harness cmd tests use
// (setupNotebook -> execute) over a synthetic notebook + synthetic code tree, so
// -benchmem attributes CPU and allocation cost to each command at scale. Use it
// to capture a baseline, then compare before/after an optimization:
//
//	go test ./cmd/nn/cmd/ -run '^$' -bench BenchmarkCmdPerf -benchmem
//
// Reuse this for any future annotate-path perf work: add a sub-benchmark for a
// new command, or widen the size/match lists below when profiling one layer.
// ─────────────────────────────────────────────────────────────────────────────

// notebookSizes controls how many synthetic notes the corpus contains. The
// per-call corpus work (link maps + fieldIDF cache key hashing) scales with this.
var notebookSizes = []int{200, 600}

// matchCounts controls how many matching lines the searched file contains, which
// is how many times grep/ast/shuf invoke the per-match ranking path.
var matchCounts = []int{10, 50}

// tbHarness is the subset of *testing.B used by the shared setup helpers, which
// were written against *testing.T. Both satisfy testing.TB.
//
// benchNotebook builds an isolated notebook of n notes plus the git repo and
// config the real command path expects, returning the execute closure.
func benchNotebook(b *testing.B, n int) (string, func(...string) (string, error)) {
	b.Helper()
	nbDir := b.TempDir()
	initGitRepoForBench(b, nbDir)

	cfgDir := b.TempDir()
	b.Setenv("NN_CONFIG_DIR", cfgDir)
	cfgFile := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte(fmt.Sprintf(`
[notebooks]
default = "test"

[notebooks.test]
path = %q
backend = "gitlocal"
`, nbDir)), 0o644); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < n; i++ {
		id := fmt.Sprintf("2026%08d-%04d", i, i%10000)
		body := fmt.Sprintf("Note %d discusses handleAuth token validation, session middleware, "+
			"and request routing for service %d.", i, i%13)
		doc := fmt.Sprintf("---\nid: %s\ntitle: Note %d on auth and routing\ntype: concept\nstatus: reviewed\ntags:\n  - t%d\n  - scale\n---\n%s\n",
			id, i, i%7, body)
		if err := os.WriteFile(filepath.Join(nbDir, id+".md"), []byte(doc), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	execute := func(args ...string) (string, error) {
		root := NewRootCmdForTest(cfgFile)
		var sb strings.Builder
		root.SetOut(&sb)
		root.SetErr(&sb)
		root.SetArgs(args)
		err := root.Execute()
		return sb.String(), err
	}
	return nbDir, execute
}

func initGitRepoForBench(b *testing.B, dir string) {
	b.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
}

// benchCodeFile writes a Go source file containing matchCount lines matching the
// pattern "handleAuth", each surrounded by realistic context.
func benchCodeFile(b *testing.B, matchCount int) string {
	b.Helper()
	dir := b.TempDir()
	var sb strings.Builder
	sb.WriteString("package server\n\n")
	for i := 0; i < matchCount; i++ {
		fmt.Fprintf(&sb, "// route %d wires the auth middleware into the request path\n", i)
		fmt.Fprintf(&sb, "func handleAuth%d(sess *Session) error {\n", i)
		fmt.Fprintf(&sb, "\treturn validateToken(sess.token) // handleAuth checks the session\n")
		fmt.Fprintf(&sb, "}\n\n")
	}
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte(sb.String()), 0o644); err != nil {
		b.Fatal(err)
	}
	return f
}

// BenchmarkCmdPerfGrep measures nn grep across notebook size × match count. The
// per-match ranking path means cost grows with matchCount for a fixed notebook.
func BenchmarkCmdPerfGrep(b *testing.B) {
	for _, n := range notebookSizes {
		for _, m := range matchCounts {
			b.Run(fmt.Sprintf("notes=%d/matches=%d", n, m), func(b *testing.B) {
				_, execute := benchNotebook(b, n)
				f := benchCodeFile(b, m)
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := execute("grep", "handleAuth", f, "--max-matches", "0"); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkCmdPerfRead measures nn read, which calls the ranking path once per
// invocation (Pattern B) — cost scales with notebook size, not match count.
func BenchmarkCmdPerfRead(b *testing.B) {
	for _, n := range notebookSizes {
		b.Run(fmt.Sprintf("notes=%d", n), func(b *testing.B) {
			_, execute := benchNotebook(b, n)
			f := benchCodeFile(b, 20)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := execute("read", f); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCmdPerfAst measures nn ast, which ranks per matched reference window.
func BenchmarkCmdPerfAst(b *testing.B) {
	for _, n := range notebookSizes {
		b.Run(fmt.Sprintf("notes=%d", n), func(b *testing.B) {
			_, execute := benchNotebook(b, n)
			f := benchCodeFile(b, 20)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := execute("ast", f); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCmdPerfShuf measures nn shuf, which ranks per sampled unit. --count
// controls how many samples invoke the ranking path per invocation.
func BenchmarkCmdPerfShuf(b *testing.B) {
	for _, n := range notebookSizes {
		b.Run(fmt.Sprintf("notes=%d", n), func(b *testing.B) {
			_, execute := benchNotebook(b, n)
			dir := b.TempDir()
			var body string
			for i := 0; i < 30; i++ {
				body += fmt.Sprintf("Paragraph %d about handleAuth session token routing middleware.\n\n", i)
			}
			f := filepath.Join(dir, "doc.txt")
			if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := execute("shuf", f, "--count", "30", "--unit", "paragraphs"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCmdPerfListSearch measures nn list --search, a single-call ranking
// path over the whole corpus as candidates.
func BenchmarkCmdPerfListSearch(b *testing.B) {
	for _, n := range notebookSizes {
		b.Run(fmt.Sprintf("notes=%d", n), func(b *testing.B) {
			_, execute := benchNotebook(b, n)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := execute("list", "--search", "handleAuth token validation"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
