package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestTranscriptContextCommand(t *testing.T) {
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"context", handoffFixture(t), "AAA", "--last", "5", "--json"})
	err := c.Execute()
	if err != nil || !strings.Contains(out.String(), "EXACT ASSIGNMENT") || !strings.Contains(out.String(), "SECOND ASSIGNMENT") {
		t.Fatalf("context must return both independent assignments: %v %s", err, out.String())
	}
}

func TestTranscriptReviewTailsCommand(t *testing.T) {
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"review", reviewFixture(t), "--last", "2", "--payload", "--limit", "1", "--order", "canonical", "--json"})
	err := c.Execute()
	if err != nil || !strings.Contains(out.String(), "recent") || !strings.Contains(out.String(), "failed") {
		t.Fatalf("review must return selected room tail: %v %s", err, out.String())
	}
}
