package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEventsAgentAlias(t *testing.T) {
	p := handoffFixture(t)
	a, e := contextCommand(t, "events", p, "AAA", "--last", "20")
	if e != nil {
		t.Fatal(e)
	}
	b, e := contextCommand(t, "events", p, "--agent", "AAA", "--last", "20")
	if e != nil || a != b {
		t.Fatal("ALIAS FAIL", e)
	}
	for _, args := range [][]string{{p}, {p, "AAA", "--agent", "AAA"}, {p, "--agent", ""}} {
		if _, e = contextCommand(t, append([]string{"events"}, args...)...); e == nil {
			t.Fatal("ALIAS_CONFLICT FAIL")
		}
	}
	t.Log("ALIAS PASS")
}
func TestEventsAssignmentBundle(t *testing.T) {
	p := handoffFixture(t)
	args := []string{"events", p, "--agent", "AAA", "--last", "3", "--include-assignment", "--payload"}
	out, e := contextCommand(t, args...)
	if e != nil {
		t.Fatal(e)
	}
	var first ledgerPage
	if e = json.Unmarshal([]byte(out), &first); e != nil {
		t.Fatal(e)
	}
	var selected []string
	launches := 0
	for _, raw := range first.Events {
		var row map[string]json.RawMessage
		_ = json.Unmarshal(raw, &row)
		switch ledgerString(row, "section") {
		case "selected":
			selected = append(selected, ledgerString(row, "event_id"))
		case "launch":
			launches++
		}
	}
	plain, e := contextCommand(t, "events", p, "AAA", "--last", "3", "--payload")
	if e != nil {
		t.Fatal(e)
	}
	var regular ledgerPage
	_ = json.Unmarshal([]byte(plain), &regular)
	ids := []string{}
	for _, raw := range regular.Events {
		var row map[string]json.RawMessage
		_ = json.Unmarshal(raw, &row)
		ids = append(ids, ledgerString(row, "event_id"))
	}
	if launches < 2 || !reflect.DeepEqual(selected, ids) {
		t.Fatal("ASSIGNMENT_SELECTION FAIL", launches, selected, ids)
	}
	text, e := contextCommand(t, "events", p, "--agent", "AAA", "--last", "3", "--include-errors", "2", "--include-assignment", "--format", "text")
	if e != nil || !strings.Contains(text, "EXACT ASSIGNMENT") || !strings.Contains(text, "SECOND ASSIGNMENT") {
		t.Fatal("ASSIGNMENT_TEXT FAIL", e, text)
	}
	if e = os.Remove(p); e != nil {
		t.Fatal(e)
	}
	replay, e := contextCommand(t, append(args, "--snapshot", first.Snapshot)...)
	if e != nil || replay != out {
		t.Fatal("ASSIGNMENT_REPLAY FAIL", e)
	}
	t.Log("ASSIGNMENT_SELECTION PASS; ASSIGNMENT_TEXT PASS; ASSIGNMENT_REPLAY PASS")
}
func TestObserveEligibilityContract(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	cutoff := now.Add(-time.Hour)
	prior := &observeLiveAgent{Fingerprint: "same", State: "evaluated"}
	cases := []struct {
		p                      *observeLiveAgent
		fp                     string
		latest                 time.Time
		unknown, refresh, want bool
	}{{nil, "same", old, false, false, false}, {nil, "same", cutoff, false, false, true}, {nil, "same", old, true, false, true}, {prior, "same", old, false, true, false}, {prior, "changed", old, false, true, true}, {nil, "same", old, false, true, true}}
	for _, c := range cases {
		if got := observeEligible(c.p, c.fp, c.latest, c.unknown, cutoff, c.refresh); got != c.want {
			t.Fatalf("ELIGIBILITY FAIL: %+v got %v", c, got)
		}
	}
	t.Log("ELIGIBILITY PASS")
}
func TestObserveWideRefreshReplay(t *testing.T) {
	p := observeAttentionFixture(t)
	f, e := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 24; i++ {
		fmt.Fprintf(f, "{\"type\":\"custom\",\"customType\":\"subagents:record\",\"data\":{\"id\":\"extra%d\",\"status\":\"completed\"}}\n", i)
	}
	f.Close()
	out, e := contextCommand(t, "observe", p)
	if e != nil {
		t.Fatal(e)
	}
	extract := func(out string) string {
		t.Helper()
		pieces := strings.SplitN(out, "Observation snapshot: ", 2)
		if len(pieces) != 2 {
			t.Fatal(out)
		}
		return strings.SplitN(pieces[1], "\n", 2)[0]
	}
	snap := extract(out)
	s, e := loadObserveState(snap, p)
	if e != nil || len(s.Agents) != 28 {
		t.Fatal("WIDE_COVERAGE FAIL", e, len(s.Agents))
	}
	for _, a := range s.Agents {
		if a.Room == nil {
			t.Fatal("WIDE_COVERAGE FAIL: silently excluded", a.ID)
		}
	}
	refreshed, e := contextCommand(t, "observe", p, "--refresh", snap)
	if e != nil {
		t.Fatal(e)
	}
	s2, e := loadObserveState(extract(refreshed), p)
	if e != nil {
		t.Fatal(e)
	}
	for i, a := range s2.Agents {
		if a.State != "unchanged" || a.Snapshot != s.Agents[i].Snapshot || !reflect.DeepEqual(a.Room, s.Agents[i].Room) {
			t.Fatal("UNCHANGED FAIL", a.ID)
		}
	}
	coverage, e := contextCommand(t, "observe", p, "--snapshot", snap, "--coverage-page", "2")
	if e != nil || !strings.Contains(coverage, "page 2/2") {
		t.Fatal("COVERAGE_PAGE FAIL", e, coverage)
	}
	if e = os.Remove(p); e != nil {
		t.Fatal(e)
	}
	replay, e := contextCommand(t, "observe", p, "--snapshot", snap)
	if e != nil || replay != out {
		t.Fatal("OBSERVE_REPLAY FAIL", e)
	}
	t.Log("WIDE_COVERAGE PASS; UNCHANGED PASS; COVERAGE_PAGE PASS; OBSERVE_REPLAY PASS")
}
