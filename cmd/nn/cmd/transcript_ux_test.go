package cmd

import (
	"strings"
	"testing"
)

func TestTranscriptRecentAliasesLsCommand(t *testing.T) {
	transcript := newTranscriptCmd(&rootState{})
	lsCmd, _, err := transcript.Find([]string{"ls"})
	if err != nil {
		t.Fatalf("find ls: %v", err)
	}
	recentCmd, _, err := transcript.Find([]string{"recent"})
	if err != nil {
		t.Fatalf("find recent: %v", err)
	}
	if recentCmd.Name() != "ls" {
		t.Fatalf("recent resolved to %q, want ls", recentCmd.Name())
	}
	if recentCmd != lsCmd {
		t.Fatal("recent and ls resolved to different command implementations")
	}
}

func TestTranscriptEventsMissingAgentExplainsRecoveryWithoutAcquiring(t *testing.T) {
	acquired := false
	cmd := newTranscriptEventsCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) {
		acquired = true
		return nil, "", "", nil
	})
	cmd.Args = nil // Unit-test RunE directly; outer session discovery is covered separately.
	cmd.SetArgs([]string{"session-123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("missing selector: expected an error")
	}
	if acquired {
		t.Fatal("missing selector: acquisition ran before explicit agent selection")
	}
	for _, want := range []string{"ROOT", "--agent", "--description", "nn transcript tree session-123"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing selector error %q does not contain %q", err, want)
		}
	}
}
