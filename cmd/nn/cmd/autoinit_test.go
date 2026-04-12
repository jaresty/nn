package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutoInitHelpfulErrorWhenNoConfig(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.toml") // does not exist

	var stderr bytes.Buffer
	root := NewRootCmd(cfgFile)
	root.SetErr(&stderr)
	root.SetArgs([]string{"list"})
	err := root.Execute()

	if err == nil {
		t.Fatal("nn list with no config: want error, got nil")
	}
	msg := strings.ToLower(err.Error() + stderr.String())
	if !strings.Contains(msg, "nn init") {
		t.Errorf("error message should mention 'nn init', got: %q", msg)
	}
}
