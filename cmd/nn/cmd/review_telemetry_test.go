package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProtocolPresenceLog(t *testing.T, cfgDir string, entries []string) {
	t.Helper()
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(filepath.Join(cfgDir, "protocol-presence.log"))
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	defer f.Close()
	for _, e := range entries {
		fmt.Fprintln(f, e)
	}
}

func writeAccessLog(t *testing.T, cfgDir string, entries []string) {
	t.Helper()
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(filepath.Join(cfgDir, "access.log"))
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	defer f.Close()
	for _, e := range entries {
		fmt.Fprintln(f, e)
	}
}

// Assertion: review output contains Protocol telemetry section when log exists.
func TestReviewProtocolTelemetrySection(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	writeProtocolPresenceLog(t, cfgDir, []string{
		"2026-05-01T10:00:00Z virtual-nn-capture-discipline proto-abc",
		"2026-05-02T10:00:00Z virtual-nn-capture-discipline proto-abc",
		"2026-05-03T10:00:00Z virtual-nn-capture-discipline",
	})

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "## Protocol telemetry") {
		t.Errorf("expected '## Protocol telemetry' section; got:\n%s", out)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected protocol ID in telemetry section; got:\n%s", out)
	}
}

// Assertion: review JSON output contains protocol_telemetry key.
func TestReviewProtocolTelemetryJSON(t *testing.T) {
	_, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	writeProtocolPresenceLog(t, cfgDir, []string{
		"2026-05-01T10:00:00Z virtual-nn-capture-discipline",
		"2026-05-02T10:00:00Z virtual-nn-capture-discipline",
	})

	out, err := execute("review", "--format", "json")
	if err != nil {
		t.Fatalf("review --format json: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if _, ok := result["protocol_telemetry"]; !ok {
		t.Errorf("expected 'protocol_telemetry' key in JSON; got keys: %v", jsonKeys(result))
	}
}

// Assertion: review output contains Note access section when access.log exists.
func TestReviewNoteAccessSection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	_ = nbDir
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	writeAccessLog(t, cfgDir, []string{
		"2026-05-01T10:00:00Z show note-aaa",
		"2026-05-01T11:00:00Z show note-aaa",
		"2026-05-02T10:00:00Z show note-bbb",
	})

	out, err := execute("review")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !strings.Contains(out, "## Note access") {
		t.Errorf("expected '## Note access' section; got:\n%s", out)
	}
	if !strings.Contains(out, "note-aaa") {
		t.Errorf("expected note ID in access section; got:\n%s", out)
	}
}
