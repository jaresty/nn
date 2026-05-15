package cmd

import (
	"strings"
	"testing"
)

func TestFlagErrorUsageBlock(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)
	stdout, _, err := executeWithStderr(t, cfgFile, "new", "--status", "draft")
	if err == nil {
		t.Fatal("expected error for unknown flag --status, got nil")
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("expected usage block in stdout on flag error, got: %q", stdout)
	}
}
