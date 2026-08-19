package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestLiveShowGlobalBenchmarkRegistered(t *testing.T) {
	source, err := os.ReadFile("show_global_bench_test.go")
	if err != nil {
		t.Fatalf("read show_global_bench_test.go: %v", err)
	}
	if !strings.Contains(string(source), "func BenchmarkShowGlobalLive(") {
		t.Fatal("BenchmarkShowGlobalLive must be registered for opt-in live-notebook profiling")
	}
}
