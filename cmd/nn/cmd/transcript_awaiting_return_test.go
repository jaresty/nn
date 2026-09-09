package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAwaitingReturnOfficeContract(t *testing.T) {
	_, execute := setupNotebook(t)
	for reference, phrases := range map[string][]string{
		"navigate": {"Awaiting return — default Pi Office view", "--queue awaiting-return --order observed-recent --limit 20 --json", "Show every returned row", "More → All rooms"},
		"review":   {"regardless of terminal count", "Terminal recorded; return missing", "No terminal recorded", "Show every returned row", "expose Next", "not only ROOT children"},
	} {
		out, e := execute("skills", "get", "nn-transcript", "--reference", reference)
		if e != nil {
			t.Fatal(e)
		}
		for _, phrase := range phrases {
			if !strings.Contains(out, phrase) {
				t.Fatalf("ASSERT_AWAITING_OFFICE: %s missing %q", reference, phrase)
			}
		}
	}
}

func TestAwaitingReturnEligibility(t *testing.T) {
	for _, tc := range []struct {
		r    reviewRow
		want bool
	}{
		{reviewRow{AuthenticatedLaunches: 1}, true},
		{reviewRow{AuthenticatedLaunches: 1, Terminals: 1}, true},
		{reviewRow{AuthenticatedLaunches: 1, Returns: 1}, false},
		{reviewRow{Launches: 1}, false},
		{reviewRow{}, false},
		{reviewRow{AuthenticatedLaunches: 2, Returns: 1}, false},
	} {
		if got := reviewEligible(tc.r, "awaiting-return"); got != tc.want {
			t.Fatalf("ASSERT_AWAITING_RETURN: %+v got %v want %v", tc.r, got, tc.want)
		}
	}
	if reviewEligible(reviewRow{AuthenticatedLaunches: 1, Terminals: 1}, "open-handoff") {
		t.Fatal("legacy open-handoff changed")
	}
}

func TestAwaitingReturnCLI(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	args := []string{"transcript", "review", path, "--queue", "awaiting-return", "--order", "observed-recent", "--limit", "2", "--json"}
	out, e := execute(args...)
	if e != nil {
		t.Fatalf("ASSERT_AWAITING_CLI: %v", e)
	}
	var p reviewPage
	if e = json.Unmarshal([]byte(out), &p); e != nil {
		t.Fatal(e)
	}
	if p.Eligible != 3 || p.Returned != 2 || p.Omitted != 1 || p.Rows[0].ID != "B" || p.Rows[1].ID != "C" {
		t.Fatalf("ASSERT_AWAITING_PAGE: %s", out)
	}
	out, e = execute(append(args, "--cursor", p.NextCursor)...)
	if e != nil {
		t.Fatal(e)
	}
	var next reviewPage
	_ = json.Unmarshal([]byte(out), &next)
	if next.Rows[0].ID != "D" || next.Offset+next.Returned+next.Omitted != next.Eligible {
		t.Fatalf("ASSERT_AWAITING_CONTINUATION: %s", out)
	}
	if _, e = execute(append(args, "--queue", "open-handoff", "--cursor", p.NextCursor)...); e == nil {
		t.Fatal("ASSERT_AWAITING_BINDING: queue change accepted")
	}
}
