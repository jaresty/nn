package cmd

import "sync/atomic"

// Process-local acquisition diagnostic. Counts decoded raw transcript records,
// including repeated reads, not retained observation-state JSON or message
// projection passes. Atomic so instrumentation is safe in concurrent tests.
var transcriptDecodeCount atomic.Uint64
