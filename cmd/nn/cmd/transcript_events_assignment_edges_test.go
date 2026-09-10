package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func assignmentLargeFixture(t *testing.T) (string, []string, ledgerPage) {
	t.Helper()
	p := handoffFixture(t)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.ReplaceAll(b, []byte("EXACT ASSIGNMENT"), []byte(strings.Repeat("assignment evidence ", 6000)))
	if err = os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"events", p, "--agent", "AAA", "--last", "3", "--include-assignment", "--payload"}
	out, err := contextCommand(t, args...)
	if err != nil {
		t.Fatal(err)
	}
	var first ledgerPage
	if err = json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatal(err)
	}
	if first.Pages < 2 {
		t.Fatal("ASSIGNMENT_PAGES FAIL: fixture did not span pages")
	}
	return p, args, first
}
func TestAssignmentMultiPageReplay(t *testing.T) {
	p, args, first := assignmentLargeFixture(t)
	textArgs := []string{"events", p, "--agent", "AAA", "--last", "3", "--include-assignment", "--format", "text"}
	before, err := contextCommand(t, textArgs...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(before, "truncated") || !strings.Contains(before, "## Selected events") {
		t.Fatal("ASSIGNMENT_BUDGET FAIL: selection lost or clipping undisclosed")
	}
	pages := make([]string, first.Pages)
	for n := 1; n <= first.Pages; n++ {
		pageArgs := append(append([]string{}, args...), "--snapshot", first.Snapshot, "--page", fmt.Sprint(n))
		pages[n-1], err = contextCommand(t, pageArgs...)
		if err != nil {
			t.Fatal(err)
		}
		if len(pages[n-1]) > 48000 {
			t.Fatal("ASSIGNMENT_PAGES FAIL: page exceeds bound")
		}
	}
	if err = os.Remove(p); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= first.Pages; n++ {
		replay, err := contextCommand(t, append(append([]string{}, args...), "--snapshot", first.Snapshot, "--page", fmt.Sprint(n))...)
		if err != nil || replay != pages[n-1] {
			t.Fatal("ASSIGNMENT_PAGES FAIL: replay differs", n, err)
		}
	}
	textSnapshot := strings.Fields(strings.SplitN(before, "\n", 2)[0])[1]
	replay, err := contextCommand(t, append(textArgs, "--snapshot", textSnapshot)...)
	if err != nil || replay != before {
		t.Fatal("ASSIGNMENT_PAGES FAIL: readable replay differs", err)
	}
	if _, err = contextCommand(t, append(textArgs, "--snapshot", first.Snapshot)...); err == nil {
		t.Fatal("ASSIGNMENT_BINDING FAIL: distinct selections accepted")
	}
	t.Log("ASSIGNMENT_PAGES PASS; ASSIGNMENT_BUDGET PASS; ASSIGNMENT_BINDING PASS")
}
func TestAssignmentOutputCeiling(t *testing.T) {
	p := handoffFixture(t)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		if _, err = fmt.Fprintf(f, "\n{\"type\":\"message\",\"agentId\":\"AAA\",\"message\":{\"role\":\"assistant\",\"content\":%q}}\n", strings.Repeat("x", 10000)); err != nil {
			f.Close()
			t.Fatal(err)
		}
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := contextCommand(t, "events", p, "--agent", "AAA", "--last", "30", "--include-assignment", "--format", "text", "--max-text-chars", "10000")
	if err == nil || !strings.Contains(err.Error(), "exceeds 200000 bytes") || strings.Contains(out, "## Launch assignments") || strings.Contains(out, "## Selected events") || strings.Contains(out, strings.Repeat("x", 100)) {
		t.Fatal("ASSIGNMENT_CEILING FAIL", err, len(out))
	}
	t.Log("ASSIGNMENT_CEILING PASS")
}

func TestAssignmentTransportAtomicity(t *testing.T) {
	p, _, first := assignmentLargeFixture(t)
	// Obtain native pages through the command to avoid manufacturing a different
	// request binding; deliberately damage only the transport presented to render.
	load := func(n int) (ledgerPage, error) {
		out, e := contextCommand(t, "events", p, "--agent", "AAA", "--last", "3", "--include-assignment", "--payload", "--snapshot", first.Snapshot, "--page", fmt.Sprint(n))
		var page ledgerPage
		if e == nil {
			e = json.Unmarshal([]byte(out), &page)
		}
		return page, e
	}
	for _, kind := range []string{"load", "snapshot", "page", "segments"} {
		t.Run(kind, func(t *testing.T) {
			var out bytes.Buffer
			err := renderAssignedEvents(&out, first, func(n int) (ledgerPage, error) {
				page, e := load(n)
				if e != nil {
					return page, e
				}
				switch kind {
				case "load":
					return page, fmt.Errorf("injected page failure")
				case "snapshot":
					page.Snapshot = "different"
				case "page":
					page.Page++
				case "segments":
					page.Events = nil
				}
				return page, nil
			}, 1000, 8000)
			if err == nil || out.Len() != 0 {
				t.Fatal("ASSIGNMENT_ATOMIC FAIL", kind, err, out.Len())
			}
		})
	}
	t.Log("ASSIGNMENT_ATOMIC PASS")
}
