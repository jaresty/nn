//go:build darwin || linux

package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestTranscriptArtifactReadRejectsFIFO(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "pipe")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	records := artifactReadTestRecords(path)
	event := artifactReadEventID(t, records)
	var out bytes.Buffer
	c := newTranscriptArtifactReadCmdUsing(func(string, string) ([]ledgerRecord, string, string, error) { return records, "pi", "available", nil }, openTranscriptArtifact)
	c.SetOut(&out)
	c.SilenceErrors = true
	c.SilenceUsage = true
	c.SetArgs([]string{"session", "agent", "--event", event, "--snapshot", artifactReadTestSnapshot(t, records, event), "--finding", "1", "--allow-root", root})
	if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "nonregular") || out.Len() != 0 {
		t.Fatalf("ASSERT_ARTIFACT_FIFO_NONBLOCK_DENIAL: err=%v output=%q", err, out.String())
	}
}
