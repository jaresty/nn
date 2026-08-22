package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/note"
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
// BenchmarkShowTargetPhases diagnoses the default nn show path for the note ID
// whose real-notebook invocation exposed an ~8s latency. The fixture uses the
// same ID so profiles and benchmark output remain directly searchable, while
// keeping the benchmark hermetic. Compare end_to_end with governance and render
// to identify whether rules fixpoint evaluation or output work dominates.
func BenchmarkShowTargetPhases(b *testing.B) {
	const targetID = "20260724233043-3087"

	nbDir, execute := benchNotebook(b, 600)
	targetPath := filepath.Join(nbDir, targetID+"-slow-show-target.md")
	target := fmt.Sprintf("---\nid: %s\ntitle: Slow show target\ntype: concept\nstatus: reviewed\n---\nTarget body.\n", targetID)
	if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
		b.Fatal(err)
	}

	gl, err := gitlocal.New(nbDir)
	if err != nil {
		b.Fatal(err)
	}
	state := &rootState{notebookDir: nbDir, backend: gl}
	all, err := state.backend.List()
	if err != nil {
		b.Fatal(err)
	}
	shown, err := state.backend.Read(targetID)
	if err != nil {
		b.Fatal(err)
	}
	byID := make(map[string]*note.Note, len(all))
	for _, n := range all {
		byID[n.ID] = n
	}
	data, err := shown.Marshal()
	if err != nil {
		b.Fatal(err)
	}
	backlinkers := findBacklinkers(targetID, all)

	b.Run("end_to_end", func(b *testing.B) {
		for b.Loop() {
			if _, err := execute("show", targetID); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("governance", func(b *testing.B) {
		for b.Loop() {
			if ge := buildGovernanceEngine(all); ge == nil {
				b.Fatal("governance engine build failed")
			}
		}
	})
	b.Run("render", func(b *testing.B) {
		raw := string(data)
		b.ResetTimer()
		for b.Loop() {
			_ = renderWithResolvedLinks(raw, shown, byID, backlinkers, false)
		}
	})
}

// TestBuildGovernanceEngineUsesPredicateDirectedEval prevents default show from
// regressing to complete-ruleset evaluation when it only queries governs_note.
func TestBuildGovernanceEngineUsesPredicateDirectedEval(t *testing.T) {
	source, err := os.ReadFile("show.go")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "func buildGovernanceEngine")
	end := strings.Index(string(source[start:]), "\n}\n")
	if start < 0 || end < 0 {
		t.Fatal("func buildGovernanceEngine not found")
	}
	body := string(source[start : start+end])
	if !strings.Contains(body, `EvalFor("governs_note")`) {
		t.Fatal(`buildGovernanceEngine must use EvalFor("governs_note")`)
	}
	if strings.Contains(body, ".Eval()") {
		t.Fatal("buildGovernanceEngine must not evaluate the complete ruleset")
	}
}

// BenchmarkShowLiveTargetPhases isolates the real notebook's default show phases.
// It is opt-in because it reads mutable local data:
//
//	NN_BENCH_NOTEBOOK=/path/to/notebook go test ./cmd/nn/cmd -run '^$' \
//	  -bench '^BenchmarkShowLiveTargetPhases$' -benchtime=1x
func BenchmarkShowLiveTargetPhases(b *testing.B) {
	const targetID = "20260724233043-3087"
	nbDir := os.Getenv("NN_BENCH_NOTEBOOK")
	if nbDir == "" {
		b.Skip("set NN_BENCH_NOTEBOOK to run the live-corpus show benchmark")
	}
	gl, err := gitlocal.New(nbDir)
	if err != nil {
		b.Fatal(err)
	}

	var all []*note.Note
	b.Run("list", func(b *testing.B) {
		for b.Loop() {
			all, err = gl.List()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	if all == nil {
		all, err = gl.List()
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Run("governance", func(b *testing.B) {
		for b.Loop() {
			if ge := buildGovernanceEngine(all); ge == nil {
				b.Fatal("governance engine build failed")
			}
		}
	})

	shown, err := gl.Read(targetID)
	if err != nil {
		b.Fatal(err)
	}
	byID := make(map[string]*note.Note, len(all))
	for _, n := range all {
		byID[n.ID] = n
	}
	data, err := shown.Marshal()
	if err != nil {
		b.Fatal(err)
	}
	backlinkers := findBacklinkers(targetID, all)
	b.Run("render", func(b *testing.B) {
		raw := string(data)
		b.ResetTimer()
		for b.Loop() {
			_ = renderWithResolvedLinks(raw, shown, byID, backlinkers, false)
		}
	})
}

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

// benchTraceCodeFile writes a Go source file with matchCount callable functions
// that match "handleAuth" and call a shared helper, so grep --trace has real
// callable matches to trace.
func benchTraceCodeFile(b *testing.B, matchCount int) string {
	b.Helper()
	dir := b.TempDir()
	var sb strings.Builder
	sb.WriteString("package server\n\n")
	sb.WriteString("func validateToken() {}\n\n")
	for i := 0; i < matchCount; i++ {
		fmt.Fprintf(&sb, "func handleAuth%d() { validateToken() }\n\n", i)
	}
	f := filepath.Join(dir, "server.go")
	if err := os.WriteFile(f, []byte(sb.String()), 0o644); err != nil {
		b.Fatal(err)
	}
	return f
}

// BenchmarkCmdPerfGrepTrace measures nn grep --trace end to end. Per unique
// matched file it builds a trace index and runs the annotated call graph, so
// this captures the in-loop BuildIndex + per-node annotation cost.
func BenchmarkCmdPerfGrepTrace(b *testing.B) {
	for _, n := range notebookSizes {
		b.Run(fmt.Sprintf("notes=%d", n), func(b *testing.B) {
			_, execute := benchNotebook(b, n)
			f := benchTraceCodeFile(b, 10)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := execute("grep", "handleAuth", f, "--trace", "--max-matches", "0"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCmdPerfGrepTraceMultiFile measures nn grep --trace when matches span
// many files in one directory. grep builds a trace index once per unique matched
// file, so this exposes the cost of repeated BuildIndex over the same tree.
func BenchmarkCmdPerfGrepTraceMultiFile(b *testing.B) {
	_, execute := benchNotebook(b, 200)
	dir := b.TempDir()
	const files = 10
	for j := 0; j < files; j++ {
		var sb strings.Builder
		sb.WriteString("package server\n\n")
		fmt.Fprintf(&sb, "func handleAuth%d() { validateToken%d() }\n", j, j)
		fmt.Fprintf(&sb, "func validateToken%d() {}\n", j)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", j)), []byte(sb.String()), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := execute("grep", "handleAuth", dir, "--trace", "--max-matches", "0"); err != nil {
			b.Fatal(err)
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
