package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestTranscriptEventContextTransport(t *testing.T) {
	p := contextFixture(t)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	large := strings.Repeat("x", 90000)
	writeTranscriptFile(t, p, strings.Replace(string(data), "Continue after result", large, 1))
	page := contextPage(t, p, "--search", large[:50], "--payload")
	if page.Pages < 2 {
		t.Fatal("expected segmented transport")
	}
	snapshot := page.Snapshot
	var reconstructed strings.Builder
	for n := 1; n <= page.Pages; n++ {
		current := page
		if n > 1 {
			current = contextPage(t, p, "--search", large[:50], "--payload", "--page", fmt.Sprint(n), "--snapshot", snapshot)
		}
		if current.Snapshot != snapshot {
			t.Fatal("snapshot drift")
		}
		b, _ := json.Marshal(current)
		if len(b)+1 > 48000 {
			t.Fatal("page bound")
		}
		for _, raw := range current.Events {
			var fragment struct {
				Text string `json:"text"`
			}
			if err = json.Unmarshal(raw, &fragment); err != nil {
				t.Fatal(err)
			}
			reconstructed.WriteString(fragment.Text)
		}
	}
	var e ledgerEvent
	if err = json.Unmarshal([]byte(reconstructed.String()), &e); err != nil {
		t.Fatal(err)
	}
	if e["window_match"] != true || ledgerSearchText(e) != large {
		t.Fatal("lost anchor or payload")
	}
}

func TestTranscriptSearchContextChangedAnchor(t *testing.T) {
	p := contextFixture(t)
	r, err := searchTranscriptFiles([]string{p}, "First plan", "", "", false, 5)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	writeTranscriptFile(t, p, strings.Replace(string(data), `"id":"call"`, `"id":"replacement"`, 1))
	err = addTranscriptSearchContext(&r, ledgerWindowOptions{Before: 1, After: 1})
	if err == nil || !strings.Contains(err.Error(), "source changed") {
		t.Fatal("changed identity accepted", err)
	}
}

func TestTranscriptContextOwnedChild(t *testing.T) {
	p := contextFixture(t)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	child := strings.ReplaceAll(string(data), `"type":"message",`, `"type":"message","agentId":"child",`)
	child = strings.ReplaceAll(child, "Continue", "CHILD Continue")
	writeTranscriptFile(t, p, string(data)+child)
	text, err := contextCommand(t, "events", p, "child", "--search", "continue", "-C", "1", "--format", "text")
	if err != nil || !strings.Contains(text, "CHILD Continue") || strings.Contains(text, ": Continue") {
		t.Fatal(err, text)
	}
	text, err = contextCommand(t, "search", "continue", p, "--agent", "child", "-C", "1")
	if err != nil || !strings.Contains(text, "CHILD Continue") || strings.Contains(text, ": Continue") {
		t.Fatal(err, text)
	}
}

func TestTranscriptEventContextTimeAndErrorAnchorOnly(t *testing.T) {
	p := contextFixture(t)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(data), `"id":"result",`, `"id":"result","timestamp":"2026-09-09T12:00:00Z",`, 1)
	s = strings.Replace(s, `"role":"toolResult",`, `"role":"toolResult","isError":true,`, 1)
	writeTranscriptFile(t, p, s)
	page := contextPage(t, p, "--errors-only", "--since", "2026-09-09T12:00:00Z", "--until", "2026-09-09T12:00:00Z", "-C", "1")
	if len(page.Events) != 3 || page.Query.Window.SelectedMatches != 1 {
		t.Fatal(page)
	}
	events := contextEvents(t, page)
	if events[1]["kind"] != "tool_result" || events[2]["kind"] != "message" {
		t.Fatal(events)
	}
}
