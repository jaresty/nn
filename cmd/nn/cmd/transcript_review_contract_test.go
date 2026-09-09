package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reviewFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "session.jsonl")
	lines := []string{`{"type":"session","version":3}`}
	for _, id := range []string{"A", "B", "C", "D"} {
		lines = append(lines, fmt.Sprintf(`{"type":"message","id":"invoke%s","message":{"role":"assistant","content":[{"type":"toolCall","id":"call%s","name":"Agent","arguments":{"description":"Task %s"}}]}}`, id, id, id))
		lines = append(lines, fmt.Sprintf(`{"type":"message","parentId":"invoke%s","message":{"role":"toolResult","toolName":"Agent","toolCallId":"call%s","details":{"status":"background","agentId":"%s"}}}`, id, id, id))
	}
	lines = append(lines,
		`{"type":"custom","customType":"subagents:record","timestamp":"2099-01-01T00:00:00Z","data":{"id":"A","status":"steered"}}`,
		`{"type":"message","agentId":"B","timestamp":"2026-01-01T00:00:00.5Z","message":{"role":"assistant","content":[{"type":"toolCall","name":"bash","arguments":{"command":"echo x"}}]}}`,
		`{"type":"message","agentId":"B","timestamp":"2026-01-01T00:05:00Z","message":{"role":"assistant","content":[{"type":"toolCall","name":"bash","arguments":{"command":"echo x"}}]}}`,
		`{"type":"message","agentId":"B","timestamp":"2026-01-01T00:06:00Z","message":{"role":"toolResult","isError":true,"content":"failed"}}`,
		`{"type":"message","agentId":"C","timestamp":"2026-01-01T01:00:00+02:00","message":{"role":"assistant","stopReason":"aborted","content":"old"}}`,
		`{"type":"message","agentId":"C","timestamp":"2026-01-01T00:00:00Z","message":{"role":"assistant","content":"new"}}`,
		`{"type":"message","message":{"role":"toolResult","toolName":"Agent","toolCallId":"missing","details":{"status":"background","agentId":"U"}}}`)
	writeTranscriptFile(t, p, strings.Join(lines, "\n")+"\n")
	return p
}

func TestReviewMembership(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    reviewRow
		want bool
	}{
		{"authenticated", reviewRow{AuthenticatedLaunches: 1}, true},
		{"unauthenticated", reviewRow{Launches: 1}, false},
		{"terminal", reviewRow{AuthenticatedLaunches: 1, Terminals: 1}, false},
		{"return", reviewRow{AuthenticatedLaunches: 1, Returns: 1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := reviewEligible(tc.r, "open-handoff"); got != tc.want {
				t.Fatalf("open eligibility: got %v want %v", got, tc.want)
			}
		})
	}
	if !reviewEligible(reviewRow{Launches: 1, Returns: 1}, "ambiguous-handoff") || reviewEligible(reviewRow{Launches: 1}, "ambiguous-handoff") {
		t.Fatal("ambiguous queue requires retained launches and returns")
	}
}

func TestReviewPaging(t *testing.T) {
	path := reviewFixture(t)
	p, e := buildReviewPage(path, "open-handoff", "canonical", "", 1, "")
	if e != nil {
		t.Fatal(e)
	}
	if p.Eligible != 3 || p.Rows[0].ID != "B" || p.Returned != 1 || p.Omitted != 2 || p.Offset != 0 || p.Population != 5 || p.Unknown != 3 {
		t.Fatalf("filter-before-page accounting: %+v", p)
	}
	next, e := buildReviewPage(path, "open-handoff", "canonical", "", 1, p.NextCursor)
	if e != nil {
		t.Fatal(e)
	}
	if next.Rows[0].ID != "C" || next.Offset != 1 || next.Eligible != next.Offset+next.Returned+next.Omitted {
		t.Fatalf("continuation accounting: %+v", next)
	}
	for _, opts := range [][3]string{{"archive", "canonical", ""}, {"open-handoff", "observed-recent", ""}, {"open-handoff", "canonical", "errors"}} {
		if _, e := buildReviewPage(path, opts[0], opts[1], opts[2], 1, p.NextCursor); e == nil {
			t.Fatalf("cursor must bind options: %v", opts)
		}
	}
	if _, e := buildReviewPage(path, "open-handoff", "canonical", "", 2, p.NextCursor); e == nil {
		t.Fatal("cursor must bind limit")
	}
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	_, _ = f.WriteString("{\"type\":\"message\",\"agentId\":\"B\",\"message\":{\"role\":\"assistant\",\"content\":\"changed\"}}\n")
	_ = f.Close()
	if retained, e := buildReviewPage(path, "open-handoff", "canonical", "", 1, p.NextCursor); e != nil || retained.Snapshot != p.Snapshot {
		t.Fatalf("cursor must retain captured evidence after append: %v", e)
	}
}

func TestReviewEvidence(t *testing.T) {
	path := reviewFixture(t)
	p, e := buildReviewPage(path, "archive", "observed-recent", "", 20, "")
	if e != nil {
		t.Fatal(e)
	}
	if p.Rows[0].ID != "B" || p.Rows[1].ID != "C" || *p.Rows[1].LastObservedAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("work recency uses absolute times, excludes lifecycle: %+v", p.Rows)
	}
	var b reviewRow
	for _, r := range p.Rows {
		if r.ID == "B" {
			b = r
		}
		if r.Liveness != "not_inferred" || r.Pairing != "not_inferred" {
			t.Fatal("must not infer runtime state or attempts")
		}
		if r.ID == "A" && (r.Terminals != 1 || r.Returns != 1) {
			t.Fatal("every producer terminal record counts")
		}
	}
	if b.Label != "Task B" || b.LabelProvenance != "recorded" || b.LabelEventID == "unavailable" || b.LaunchOccurrence != 1 {
		t.Fatalf("recorded label provenance: %+v", b)
	}
	if b.RepeatedTools != 1 || b.RepeatedCommands != 1 || b.Errors != 1 || b.MaxGapSeconds != 299.5 {
		t.Fatalf("exact deterministic reductions: %+v", b)
	}
	recs, _, _, e := ledgerRecords(path, "B")
	if e != nil {
		t.Fatal(e)
	}
	events, e := projectLedger(recs, "B", []string{"message"}, false)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, event := range events {
		if event["event_id"] == b.LastObservedEventID {
			found = true
		}
	}
	if !found {
		t.Fatal("recency provenance must resolve in events ledger")
	}
	for _, pattern := range []string{"errors", "repeated-tools", "repeated-commands", "interruptions", "missing-evidence", "timing-gaps"} {
		p, e := buildReviewPage(path, "archive", "canonical", pattern, 20, "")
		if e != nil || p.Algorithm == "" {
			t.Fatalf("pattern %s: %v", pattern, e)
		}
		want := 1
		if pattern == "missing-evidence" {
			want = 3
		}
		if pattern == "timing-gaps" {
			want = 1
		}
		if p.Eligible != want {
			t.Fatalf("pattern %s eligible=%d want=%d", pattern, p.Eligible, want)
		}
	}
}

func TestReviewValidation(t *testing.T) {
	path := reviewFixture(t)
	for _, tc := range []struct {
		q, o, p string
		l       int
	}{{"bad", "canonical", "", 20}, {"archive", "bad", "", 20}, {"archive", "canonical", "semantic", 20}, {"archive", "canonical", "", 0}, {"archive", "canonical", "", 201}} {
		if _, e := buildReviewPage(path, tc.q, tc.o, tc.p, tc.l, ""); e == nil {
			t.Fatalf("invalid options accepted: %+v", tc)
		}
	}
	c := newTranscriptCmd(nil)
	c.SetArgs([]string{"review", path})
	if e := c.Execute(); e == nil {
		t.Fatal("JSON must be required")
	}
	p, e := buildReviewPage(path, "open-handoff", "canonical", "timing-gaps", 20, "")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), `"rows":null`) {
		t.Fatal("empty rows must be array")
	}
}

func TestReviewIndependentOccurrences(t *testing.T) {
	p, e := buildReviewPage(handoffFixture(t), "ambiguous-handoff", "canonical", "", 20, "")
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Rows) != 1 || p.Rows[0].Launches != 2 || p.Rows[0].Returns != 2 || p.Rows[0].LaunchOccurrence != 2 || p.Rows[0].Label != "Resume trial" {
		t.Fatalf("independent retained occurrences: %+v", p)
	}
}
