package cmd

import (
	"strings"
	"testing"
)

// Publication only: ordinary observation must not inherit worker/task restrictions.
func TestObserveWidePublishedRecipe(t *testing.T) {
	_, execute := setupNotebook(t)
	body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--task", "--agent-task", "--attention-agent"} {
		if strings.Contains(body, flag) {
			t.Fatalf("WIDE_RECIPE FAIL: normal observation advertises %s", flag)
		}
	}
	for _, clause := range []string{"Signal: <signal_id> · policy v<version> · metric v<metric_version>", "--refresh <observation-snapshot>", "No first-20 evaluation cap", "never narrows normal observation"} {
		if !strings.Contains(body, clause) {
			t.Fatalf("WIDE_RECIPE FAIL: missing %s", clause)
		}
	}
	rooms, err := execute("skills", "get", "nn-transcript", "--reference", "rooms")
	if err != nil || !strings.Contains(rooms, "events <session> --agent <agent-id> --last 5 --include-assignment") {
		t.Fatal("WORKER_RECIPE FAIL", err)
	}
	t.Log("WIDE_RECIPE PASS; WORKER_RECIPE PASS")
}
