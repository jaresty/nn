package cmd

import (
	"math"
	"strings"
	"testing"
)

func TestTranscriptUsageSummaryOverflow(t *testing.T) {
	for _, values := range [][]int64{{math.MaxInt64, 1}, {-1}} {
		events := []ledgerEvent{}
		for _, value := range values {
			events = append(events, ledgerEvent{"kind": "message", "usage": map[string]any{"input_tokens": value}})
		}
		if _, err := summarizeLedgerUsage(events, 0); err == nil {
			t.Fatal("ASSERT_USAGE_OVERFLOW: fail")
		}
	}
	t.Log("ASSERT_USAGE_OVERFLOW: pass")
}
func TestTranscriptUsageSummaryLimit(t *testing.T) {
	fixture := summaryFixture(t)
	events := []ledgerEvent{}
	for i := 0; i < 100; i++ {
		events = append(events, fixture[0])
	}
	if _, err := buildUsageSummary("fixture", "A", "pi", "available", events, 1, ""); err == nil || !strings.Contains(err.Error(), "increase --bucket-size") {
		t.Fatalf("ASSERT_USAGE_BOUND: fail — %v", err)
	}
	if data, err := buildUsageSummary("fixture", "A", "pi", "available", events, 10, ""); err != nil || len(data) > 48000 {
		t.Fatalf("ASSERT_USAGE_BOUND: fail — %v", err)
	}
	t.Log("ASSERT_USAGE_BOUND: pass")
}
