package cmd

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jaresty/nn/internal/trace"
)

// property [1]: grep --trace builds the trace index at most once per invocation,
// even when matches span multiple files in the same directory. This is the
// falsifiable perf artifact for hoisting BuildIndex out of the per-match loop;
// output equivalence is guarded separately by the grep --trace behavior tests.
func TestGrepTraceBuildsIndexOncePerInvocation(t *testing.T) {
	_, execute := setupNotebook(t)

	dir := t.TempDir()
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		src := "package server\n\nfunc handleAuth_" + name[:1] + "() { helper_" + name[:1] + "() }\nfunc helper_" + name[:1] + "() {}\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var calls int64
	orig := traceBuildIndex
	traceBuildIndex = func(root string) (*trace.Index, error) {
		atomic.AddInt64(&calls, 1)
		return orig(root)
	}
	defer func() { traceBuildIndex = orig }()

	if _, err := execute("grep", "handleAuth", dir, "--trace", "--max-matches", "0"); err != nil {
		t.Fatalf("nn grep --trace: %v", err)
	}

	if got := atomic.LoadInt64(&calls); got != 1 {
		t.Fatalf("BuildIndex called %d times for a multi-file grep --trace; want exactly 1", got)
	}
}
