package cmd

import (
	"strings"
	"testing"
)

func TestDelegatedProcessVirtualProtocol(t *testing.T) {
	_, execute := setupNotebook(t)
	body, err := execute("show", "virtual-nn-delegated-process")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"decision depends on what the agent actually established", "Keep the read proportional", "planned RED tests from unexpected failures", "neither substitutes for transcript evidence"} {
		if !strings.Contains(body, want) {
			t.Fatalf("DELEGATED_PROTOCOL FAIL: missing %q", want)
		}
	}
	if strings.Contains(body, "20260910035821-0195") {
		t.Fatal("DELEGATED_PROTOCOL FAIL: old note ID preserved")
	}
	global, err := execute("show", "--global")
	if err != nil || strings.Count(global, "id: virtual-nn-delegated-process") != 1 {
		t.Fatal("DELEGATED_PROTOCOL FAIL: global registration", err)
	}
	t.Log("DELEGATED_PROTOCOL PASS")
}
