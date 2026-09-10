package cmd

import (
	"io"
	"os"
	"testing"
)

// Opt-in: reads the named recording and writes only ordinary private captures.
func BenchmarkObserveRecording(b *testing.B) {
	source := os.Getenv("NN_OBSERVE_BENCH_SOURCE")
	if source == "" {
		b.Skip("set NN_OBSERVE_BENCH_SOURCE to an explicit recording")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c := newTranscriptObserveCmd()
		c.SetArgs([]string{source})
		c.SetOut(io.Discard)
		c.SetErr(io.Discard)
		if err := c.Execute(); err != nil {
			b.Fatal(err)
		}
	}
}
