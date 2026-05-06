package cmd

import (
	"strings"
	"testing"
)

// Assertion: bullet 2 uses 'solely from local code or state present in this session'.
func TestVirtualCaptureDisciplineBullet2LocalOnly(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "solely from local code or state present in this session") {
		t.Errorf("expected bullet-2 rewrite in virtual protocol; got:\n%s", out)
	}
}

// Assertion: bullet 3 covers fetching output from execution systems triggered this session.
func TestVirtualCaptureDisciplineBullet3RemoteExecution(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "triggered or are operating in this session") {
		t.Errorf("expected bullet-3 remote execution clause in virtual protocol; got:\n%s", out)
	}
}
