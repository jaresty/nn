package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func combinedFixture() []ledgerEvent {
	events := []ledgerEvent{}
	for i := 1; i <= 6; i++ {
		stop := ""
		if i == 2 || i == 6 {
			stop = "error"
		}
		events = append(events, ledgerEvent{"event_id": fmt.Sprintf("e%d", i), "ordinal": i, "kind": "message", "timestamp": fmt.Sprintf("2026-09-01T00:00:0%dZ", i), "message": map[string]any{"role": "assistant", "stop_reason": stop}, "payload": map[string]any{"role": "assistant", "content": fmt.Sprintf("event-%d", i)}})
	}
	return events
}
func combinedOutput(t *testing.T, events []ledgerEvent, q ledgerQuery, n, max int) (string, error) {
	t.Helper()
	var b bytes.Buffer
	fields, _ := ledgerSelect("identity,message,tools,lifecycle")
	err := renderCombinedEventTails(&b, "/fixture", "A", "pi", "available", fields, events, q, n, max)
	return b.String(), err
}
func combinedSections(s string) (string, string) {
	parts := strings.SplitN(s, "## Latest explicit failures", 2)
	if len(parts) != 2 {
		return s, ""
	}
	return parts[0], parts[1]
}
func TestCombinedTailCraft(t *testing.T) {
	cases := []struct {
		name  string
		check func(*testing.T) bool
	}{
		{"recent", func(t *testing.T) bool {
			o, e := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 2}, 1, 100)
			r, _ := combinedSections(o)
			return e == nil && strings.Contains(r, "returned: 2") && strings.Contains(r, "event-5") && strings.Contains(r, "event-6") && !strings.Contains(r, "event-4")
		}},
		{"failures", func(t *testing.T) bool {
			es := combinedFixture()
			es[4]["payload"].(map[string]any)["content"] = "prose mentioning error"
			o, e := combinedOutput(t, es, ledgerQuery{Last: 1}, 2, 100)
			_, f := combinedSections(o)
			empty, ee := combinedOutput(t, es[:1], ledgerQuery{Last: 1}, 2, 100)
			_, ef := combinedSections(empty)
			return e == nil && ee == nil && strings.Contains(f, "returned: 2") && strings.Contains(f, "event-2") && strings.Contains(f, "event-6") && !strings.Contains(f, "prose mentioning error") && strings.Contains(ef, "returned: 0")
		}},
		{"acquisition", func(t *testing.T) bool {
			calls := 0
			c := newTranscriptEventsCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) {
				calls++
				return nil, "pi", "unavailable", nil
			})
			var b bytes.Buffer
			c.SetOut(&b)
			c.SetArgs([]string{"/fixture", "A", "--last", "2", "--include-errors", "3", "--format", "text"})
			return c.Execute() == nil && calls == 1
		}},
		{"snapshot", func(t *testing.T) bool {
			o, e := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 2}, 1, 100)
			var snaps []string
			for _, line := range strings.Split(o, "\n") {
				if strings.HasPrefix(line, "snapshot: ") {
					snaps = append(snaps, strings.TrimPrefix(line, "snapshot: "))
				}
			}
			changed, ce := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 2}, 2, 100)
			es := combinedFixture()
			es[0]["payload"].(map[string]any)["content"] = "changed outside either tail"
			content, pe := combinedOutput(t, es, ledgerQuery{Last: 2}, 1, 100)
			return e == nil && ce == nil && pe == nil && len(snaps) == 2 && snaps[0] == snaps[1] && !strings.Contains(changed, "snapshot: "+snaps[0]) && !strings.Contains(content, "snapshot: "+snaps[0])
		}},
		{"overlap", func(t *testing.T) bool {
			o, e := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 2}, 2, 100)
			return e == nil && strings.Contains(o, "Overlapping event IDs (1): [e6]")
		}},
		{"flags", func(t *testing.T) bool {
			for _, extra := range [][]string{{"--include-errors", "0"}, {"--include-errors", "201"}, {"--errors-only"}, {"--search", "x"}, {"--context", "1"}, {"--event", "e1"}, {"--format", "json"}, {"--summary", "tools"}, {"--at", "launch"}, {"--snapshot", "x"}} {
				c := newTranscriptEventsCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return nil, "pi", "available", nil })
				var b bytes.Buffer
				c.SetOut(&b)
				c.SetErr(&b)
				c.SetArgs(append([]string{"/fixture", "A", "--last", "2", "--include-errors", "2", "--format", "text"}, extra...))
				if c.Execute() == nil {
					return false
				}
			}
			return true
		}},
		{"publication", func(t *testing.T) bool {
			es := combinedFixture()
			for i := 0; i < 30; i++ {
				e := ledgerEvent{"event_id": fmt.Sprint(i), "ordinal": i, "kind": "message", "payload": map[string]any{"role": "assistant", "content": strings.Repeat("x", 10000)}}
				es = append(es, e)
			}
			o, e := combinedOutput(t, es, ledgerQuery{Last: 30}, 1, 10000)
			small, se := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 2}, 2, 5)
			return e != nil && o == "" && se == nil && strings.Contains(small, "[truncated")
		}},
		{"time", func(t *testing.T) bool {
			lo, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:02Z")
			hi, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:05Z")
			o, e := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 10, Since: &lo, Until: &hi}, 10, 100)
			r, f := combinedSections(o)
			return e == nil && strings.Contains(r, "returned: 4") && strings.Contains(r, "event-2") && strings.Contains(r, "event-5") && !strings.Contains(r, "event-6") && strings.Contains(f, "returned: 1") && strings.Contains(f, "event-2") && !strings.Contains(f, "event-6")
		}},
		{"identity", func(t *testing.T) bool {
			es := combinedFixture()
			es[1]["tools"] = map[string]any{"join": map[string]any{"status": "matched", "call_event_id": "outside"}}
			before, _ := json.Marshal(es)
			o, e := combinedOutput(t, es, ledgerQuery{Last: 2}, 2, 100)
			after, _ := json.Marshal(es)
			r, f := combinedSections(o)
			return e == nil && bytes.Equal(before, after) && strings.Contains(r, "[e5]") && strings.Contains(r, "[e6]") && strings.Contains(f, "[e2]") && strings.Contains(f, "[e6]")
		}},
		{"qualification", func(t *testing.T) bool {
			o, e := combinedOutput(t, combinedFixture(), ledgerQuery{Last: 1}, 2, 100)
			return e == nil && strings.Contains(o, "does not establish that they remain unresolved") && strings.Contains(o, "No failures is not a health verdict")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("procedure: TestCombinedTailCraft/%s; assertion: %s", tc.name, tc.name)
			if !tc.check(t) {
				t.Fatalf("%s FAIL", tc.name)
			}
			t.Logf("%s PASS", tc.name)
		})
	}
}
