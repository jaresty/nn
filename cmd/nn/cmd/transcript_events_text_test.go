package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestTranscriptEventsReadableTail(t *testing.T) {
	path := reviewFixture(t)
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetArgs([]string{"events", path, "B", "--format", "text", "--last", "2", "--max-text-chars", "5"})
	if e := c.Execute(); e != nil {
		t.Fatalf("readable tail must execute: %v", e)
	}
	for _, s := range []string{"snapshot:", "returned: 2", "RESULT", "[truncated", "omitted:"} {
		if !strings.Contains(out.String(), s) {
			t.Fatalf("readable tail missing %q: %s", s, out.String())
		}
	}
	if strings.Contains(out.String(), "CALL") {
		t.Fatal("tail must select before rendering")
	}
}

func TestTranscriptEventsTextBounds(t *testing.T) {
	path := reviewFixture(t)
	for _, flags := range [][]string{{"--format", "text"}, {"--format", "text", "--last", "201"}, {"--format", "text", "--last", "1", "--max-text-chars", "0"}, {"--format", "text", "--last", "1", "--json"}, {"--format", "text", "--last", "1", "--all"}, {"--max-text-chars", "10"}, {"--format", "bad"}} {
		c := newTranscriptCmd(nil)
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(append([]string{"events", path, "B"}, flags...))
		if e := c.Execute(); e == nil {
			t.Fatalf("invalid text options accepted: %v", flags)
		}
	}
}
