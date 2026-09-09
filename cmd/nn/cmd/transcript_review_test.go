package cmd

import (
	"bytes"

	"strings"
	"testing"
)

func TestTranscriptReviewRegistered(t *testing.T) {
	c := newTranscriptCmd(nil)
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetArgs([]string{"review", "--help"})
	err := c.Execute()
	if err != nil || !strings.Contains(out.String(), "open-handoff") {
		t.Fatalf("review command must expose conservative queues: err=%v output=%s", err, out.String())
	}
}
