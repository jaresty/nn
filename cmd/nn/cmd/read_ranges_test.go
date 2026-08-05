package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestReadMultipleRanges_Property1_ParsesCommaSeparated(t *testing.T) {
	// property [1]: parser splits on comma and produces multiple (start,end) pairs
	f, _ := os.CreateTemp("", "nn-read-test-*.txt")
	defer os.Remove(f.Name())
	f.WriteString("line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n")
	f.Close()

	state := &rootState{backend: nil}
	cmd := newReadCmd(state)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{f.Name(), "--lines", "1-2,5-6"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// Should contain lines 1, 2, 5, 6 — not just lines 1-2
	if !strings.Contains(out, "5\t") {
		t.Errorf("expected line 5 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "6\t") {
		t.Errorf("expected line 6 in output, got:\n%s", out)
	}
}

func TestReadMultipleRanges_Property3a_Deduplication(t *testing.T) {
	// property [3a]: overlapping ranges deduplicate
	f, _ := os.CreateTemp("", "nn-read-test-*.txt")
	defer os.Remove(f.Name())
	f.WriteString("line1\nline2\nline3\nline4\nline5\n")
	f.Close()

	state := &rootState{backend: nil}
	cmd := newReadCmd(state)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{f.Name(), "--lines", "1-3,2-4"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Filter out the Related notes section
	var numbered []string
	for _, l := range lines {
		if len(l) > 0 && l[0] >= '0' && l[0] <= '9' {
			numbered = append(numbered, l)
		}
	}
	if len(numbered) != 4 {
		t.Errorf("expected 4 unique lines (1-4), got %d:\n%s", len(numbered), out)
	}
}

func TestReadMultipleRanges_Property3b_AscendingOrder(t *testing.T) {
	// property [3b]: ranges given out of order still produce ascending output
	f, _ := os.CreateTemp("", "nn-read-test-*.txt")
	defer os.Remove(f.Name())
	f.WriteString("line1\nline2\nline3\nline4\nline5\nline6\n")
	f.Close()

	state := &rootState{backend: nil}
	cmd := newReadCmd(state)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{f.Name(), "--lines", "5-6,1-2"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var numbered []string
	for _, l := range lines {
		if len(l) > 0 && l[0] >= '0' && l[0] <= '9' {
			numbered = append(numbered, l)
		}
	}
	if len(numbered) < 2 {
		t.Fatalf("expected at least 2 numbered lines, got: %s", out)
	}
	// First numbered line should be line 1, not line 5
	if !strings.HasPrefix(numbered[0], "1\t") {
		t.Errorf("expected first output line to be line 1, got: %s", numbered[0])
	}
}

func TestReadMultipleRanges_Property4_OutOfBoundsSkipped(t *testing.T) {
	// property [4]: segment where start > |file| contributes zero lines
	f, _ := os.CreateTemp("", "nn-read-test-*.txt")
	defer os.Remove(f.Name())
	f.WriteString("line1\nline2\nline3\n")
	f.Close()

	state := &rootState{backend: nil}
	cmd := newReadCmd(state)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{f.Name(), "--lines", "1-2,100-200"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var numbered []string
	for _, l := range lines {
		if len(l) > 0 && l[0] >= '0' && l[0] <= '9' {
			numbered = append(numbered, l)
		}
	}
	if len(numbered) != 2 {
		t.Errorf("expected 2 lines (1-2 only), got %d:\n%s", len(numbered), out)
	}
}

func TestReadMultipleRanges_Property5_InvalidSegmentErrors(t *testing.T) {
	// property [5]: invalid segment causes non-zero exit with error identifying segment
	f, _ := os.CreateTemp("", "nn-read-test-*.txt")
	defer os.Remove(f.Name())
	f.WriteString("line1\nline2\nline3\n")
	f.Close()

	state := &rootState{backend: nil}
	cmd := newReadCmd(state)
	cmd.SetArgs([]string{f.Name(), "--lines", "1-2,abc"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid segment, got nil")
	}
}

func TestReadMultipleRanges_Property6_LimitCapsUnion(t *testing.T) {
	// property [6]: --limit N caps output to N lines from head of sorted union
	f, _ := os.CreateTemp("", "nn-read-test-*.txt")
	defer os.Remove(f.Name())
	f.WriteString("line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\n")
	f.Close()

	state := &rootState{backend: nil}
	cmd := newReadCmd(state)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{f.Name(), "--lines", "1-4,6-8", "--limit", "3"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var numbered []string
	for _, l := range lines {
		if len(l) > 0 && l[0] >= '0' && l[0] <= '9' {
			numbered = append(numbered, l)
		}
	}
	if len(numbered) != 3 {
		t.Errorf("expected 3 lines after --limit 3, got %d:\n%s", len(numbered), out)
	}
}
