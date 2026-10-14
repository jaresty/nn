package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func seedExpiredDailyListFixture(t *testing.T) func(...string) (string, error) {
	t.Helper()
	nbDir, execute := setupNotebook(t)
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)
	for _, tc := range []struct {
		title string
		tags  []string
		exp   time.Time
	}{
		{"expired daily one", []string{"daily"}, past},
		{"expired daily two", []string{"daily"}, past},
		{"expired receipt", []string{"transcript-receipt"}, past},
		{"current daily", []string{"daily"}, future},
	} {
		n := newTestNoteForCLI(note.GenerateID(), tc.title, note.TypeObservation)
		n.Tags = tc.tags
		n.Expires = &tc.exp
		writeNoteFile(t, nbDir, n)
	}
	return execute
}

func TestListCountPreservesAllFilters(t *testing.T) {
	execute := seedExpiredDailyListFixture(t)
	out, err := execute("list", "--expired", "--tag", "daily", "--count")
	if err != nil {
		t.Fatalf("filtered count: %v", err)
	}
	if out != "2\n" {
		t.Fatalf("filtered count = %q, want %q", out, "2\\n")
	}
}

func TestListEnvelopeWorksWithoutSearchAndKeepsTotalBeforeLimit(t *testing.T) {
	execute := seedExpiredDailyListFixture(t)
	out, err := execute("list", "--expired", "--tag", "daily", "--json", "--envelope", "--limit", "1", "--fields", "id,title")
	if err != nil {
		t.Fatalf("filtered envelope: %v", err)
	}
	var got struct {
		ResultCount   int              `json:"result_count"`
		TotalMatching int              `json:"total_matching"`
		Results       []map[string]any `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode envelope: %v\n%s", err, out)
	}
	if got.ResultCount != 1 || got.TotalMatching != 2 || len(got.Results) != 1 {
		t.Fatalf("envelope = %#v", got)
	}
	if len(got.Results[0]) != 2 {
		t.Fatalf("projected result fields = %#v", got.Results[0])
	}
}

func TestListCountRejectsOutputShaping(t *testing.T) {
	execute := seedExpiredDailyListFixture(t)
	for _, args := range [][]string{
		{"list", "--count", "--json"},
		{"list", "--count", "--envelope"},
		{"list", "--count", "--fields", "id"},
	} {
		_, err := execute(args...)
		if err == nil || !strings.Contains(err.Error(), "--count") {
			t.Fatalf("%v error = %v, want --count incompatibility", args, err)
		}
	}
}
