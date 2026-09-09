package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestReviewBundleReadable(t *testing.T) {
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"review", reviewFixture(t), "--limit", "1", "--last", "2", "--order", "canonical", "--format", "text"})
	if e := c.Execute(); e != nil {
		t.Fatalf("native review text must execute: %v", e)
	}
	for _, s := range []string{"Task B", "failed", "omitted rooms: 2", "snapshot:"} {
		if !strings.Contains(out.String(), s) {
			t.Fatalf("native review text missing %q: %s", s, out.String())
		}
	}
}

func TestContextBundleReadable(t *testing.T) {
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"context", handoffFixture(t), "AAA", "--format", "text"})
	if e := c.Execute(); e != nil {
		t.Fatalf("native context text must execute: %v", e)
	}
	for _, s := range []string{"EXACT ASSIGNMENT", "SECOND ASSIGNMENT", "steering: unavailable", "LAUNCH"} {
		if !strings.Contains(out.String(), s) {
			t.Fatalf("native context text missing %q: %s", s, out.String())
		}
	}
}

func TestBundleTextConsumesPagesAndBoundsOutput(t *testing.T) {
	path := handoffFixture(t)
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	b = bytes.ReplaceAll(b, []byte("EXACT ASSIGNMENT"), []byte(strings.Repeat("assignment Ω ", 10000)))
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	first, e := buildTranscriptContext(path, "AAA", 5, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	if first.Pages < 2 {
		t.Fatal("fixture must span pages")
	}
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"context", path, "AAA", "--format", "text", "--max-output-chars", "2048", "--max-text-chars", "1000", "--snapshot", first.Snapshot})
	if e = c.Execute(); e != nil {
		t.Fatal(e)
	}
	if utf8.RuneCountInString(out.String()) > 2048 {
		t.Fatal("text must respect total character budget")
	}
	for _, s := range []string{fmt.Sprintf("Transport: %d/%d pages complete", first.Pages, first.Pages), "[truncated", "omitted from display:"} {
		if !strings.Contains(out.String(), s) {
			t.Fatalf("bounded multi-page output missing %q: %s", s, out.String())
		}
	}
}

func TestBundleTextDoesNotPublishPartialTransport(t *testing.T) {
	p := ledgerPage{Snapshot: "test", Page: 1, Pages: 2, Events: []json.RawMessage{json.RawMessage(`{"kind":"message","event_id":"x","payload":{"role":"assistant","content":"visible"}}`)}}
	var out bytes.Buffer
	e := renderBundleText(&out, p, bundleTextOptions{PerEvent: 100, Total: 2048}, func(int) (ledgerPage, error) { return ledgerPage{}, fmt.Errorf("damaged page") })
	if e == nil || out.Len() != 0 {
		t.Fatal("failed later page must produce no partial text")
	}
	p.Pages = 1
	p.Events = []json.RawMessage{json.RawMessage(`{"event_id":"x","segment":2,"segments":2,"text":"bad"}`)}
	if e = renderBundleText(&out, p, bundleTextOptions{PerEvent: 100, Total: 2048}, nil); e == nil || out.Len() != 0 {
		t.Fatal("invalid segment order must reject without output")
	}
}

func TestBundleTextVisibleFailuresNotThinking(t *testing.T) {
	p := ledgerPage{Snapshot: "test", Page: 1, Pages: 1, Events: []json.RawMessage{json.RawMessage(`{"kind":"message","event_id":"x","message":{"role":"assistant","stop_reason":"error","error_message":"fetch failed"},"payload":{"role":"assistant","content":[{"type":"thinking","thinking":"HIDDEN"},{"type":"text","text":"visible\u001b[31m"}]}}`)}}
	var out bytes.Buffer
	if e := renderBundleText(&out, p, bundleTextOptions{PerEvent: 1000, Total: 2048}, nil); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(out.String(), "HIDDEN") || strings.Contains(out.String(), "\x1b") || !strings.Contains(out.String(), "fetch failed") {
		t.Fatalf("failure/control/thinking rendering: %s", out.String())
	}
}

func TestBundleTextOptionValidation(t *testing.T) {
	path := handoffFixture(t)
	for _, flags := range [][]string{{"--format", "bad"}, {"--format", "text", "--json"}, {"--format", "text", "--page", "1"}, {"--format", "text", "--max-output-chars", "10"}, {"--format", "text", "--max-text-chars", "0"}, {"--json", "--max-text-chars", "100"}} {
		c := newTranscriptCmd(nil)
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(append([]string{"context", path, "AAA"}, flags...))
		if e := c.Execute(); e == nil {
			t.Fatalf("invalid text flags accepted: %v", flags)
		}
	}
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"review", path, "--format", "text"})
	if e := c.Execute(); e == nil {
		t.Fatal("review text must require explicit last bound")
	}
}
