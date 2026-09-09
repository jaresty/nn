package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func bundleEvents(t *testing.T, p ledgerPage) []ledgerEvent {
	t.Helper()
	out := []ledgerEvent{}
	for _, raw := range p.Events {
		var e ledgerEvent
		if err := json.Unmarshal(raw, &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

func TestReviewBundleScope(t *testing.T) {
	path := reviewFixture(t)
	p, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	ev := bundleEvents(t, p)
	receipt := ev[0]
	if receipt["eligible"] != float64(3) || receipt["retrieved_rooms"] != float64(1) || receipt["omitted_rooms"] != float64(2) || receipt["selected_events"] != float64(2) || receipt["inspection_status"] != "not_inferred" {
		t.Fatalf("bundle scope/accounting: %+v", receipt)
	}
	recent := 0
	for _, v := range ev[1:] {
		if v["agent_id"] != "B" {
			t.Fatalf("bundle must not widen room scope: %+v", v)
		}
		if v["section"] == "recent" {
			recent++
		}
	}
	if recent != 2 {
		t.Fatalf("bounded selected tail: %d", recent)
	}
	cursor := receipt["next_room_cursor"].(string)
	next, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, cursor, 2, true, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	if bundleEvents(t, next)[1]["agent_id"] != "C" {
		t.Fatal("next room cursor must advance room selection")
	}
	for _, change := range []struct {
		last    int
		payload bool
	}{{1, true}, {2, false}} {
		if _, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", change.last, change.payload, 1, p.Snapshot); e == nil {
			t.Fatal("bundle snapshot must bind last and payload")
		}
	}
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.WriteString("{\"type\":\"message\",\"agentId\":\"B\",\"message\":{\"role\":\"assistant\",\"content\":\"new\"}}\n")
	_ = f.Close()
	if _, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, p.Snapshot); e == nil {
		t.Fatal("changed evidence must reject bundle snapshot")
	}
}

func TestContextLosslessPages(t *testing.T) {
	path := handoffFixture(t)
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	assignment := strings.Repeat("long assignment Ω ", 6000)
	raw = bytes.ReplaceAll(raw, []byte("EXACT ASSIGNMENT"), []byte(assignment))
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	first, e := buildTranscriptContext(path, "AAA", 2, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	if first.Pages < 2 {
		t.Fatal("oversized assignment must paginate")
	}
	fragments := map[string][]string{}
	totals := map[string]int{}
	complete := []ledgerEvent{}
	for page := 1; page <= first.Pages; page++ {
		p, e := buildTranscriptContext(path, "AAA", 2, page, first.Snapshot)
		if e != nil {
			t.Fatal(e)
		}
		encoded, _ := json.Marshal(p)
		if len(encoded)+1 > 48000 {
			t.Fatalf("transport exceeds 48000 bytes: %d", len(encoded)+1)
		}
		for _, v := range bundleEvents(t, p) {
			if segment, ok := v["segment"].(float64); ok {
				id := v["event_id"].(string)
				if int(segment) != len(fragments[id])+1 {
					t.Fatal("segments must be ordered")
				}
				fragments[id] = append(fragments[id], v["text"].(string))
				totals[id] = int(v["segments"].(float64))
			} else {
				complete = append(complete, v)
			}
		}
	}
	for id, parts := range fragments {
		if len(parts) != totals[id] {
			t.Fatal("missing segment")
		}
		var v ledgerEvent
		if e = json.Unmarshal([]byte(strings.Join(parts, "")), &v); e != nil {
			t.Fatal(e)
		}
		complete = append(complete, v)
	}
	launches := 0
	recent := 0
	found := false
	for _, v := range complete {
		switch v["section"] {
		case "launch":
			launches++
			b, _ := json.Marshal(v)
			if bytes.Contains(b, []byte(assignment)) {
				found = true
			}
		case "recent":
			recent++
		}
		if v["kind"] == "context_receipt" && (v["steering_status"] != "unavailable" || v["governing_attempt"] != "not_inferred") {
			t.Fatal("context must not invent steering or attempt authority")
		}
	}
	if launches != 2 || recent != 2 || !found {
		t.Fatalf("lossless assignments and bounded recent events: launches=%d recent=%d found=%v", launches, recent, found)
	}
}

func TestContextValidation(t *testing.T) {
	path := handoffFixture(t)
	for _, id := range []string{"", "UNKNOWN"} {
		if _, e := buildTranscriptContext(path, id, 5, 1, ""); e == nil {
			t.Fatal("unknown room must fail")
		}
	}
	for _, c := range []struct {
		last, page int
		snapshot   string
	}{{0, 1, ""}, {201, 1, ""}, {5, 0, ""}, {5, 2, ""}, {5, 1, "bad"}} {
		if _, e := buildTranscriptContext(path, "AAA", c.last, c.page, c.snapshot); e == nil {
			t.Fatalf("invalid context options: %+v", c)
		}
	}
	for _, flags := range [][]string{{"--payload"}, {"--page", "1"}, {"--snapshot", "bad"}} {
		c := newTranscriptCmd(nil)
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(append([]string{"review", path, "--json"}, flags...))
		if e := c.Execute(); e == nil {
			t.Fatalf("review evidence flags require last: %v", flags)
		}
	}
}
