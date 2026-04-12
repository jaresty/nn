package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestInitCreatesConfig(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	nbDir := t.TempDir()

	root := NewRootCmd(cfgFile)
	root.SetArgs([]string{"init", "--path", nbDir, "--name", "personal"})
	if err := root.Execute(); err != nil {
		t.Fatalf("nn init: %v", err)
	}

	if _, err := os.Stat(cfgFile); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	var raw map[string]any
	if _, err := toml.DecodeFile(cfgFile, &raw); err != nil {
		t.Fatalf("config not valid TOML: %v", err)
	}
}

func TestInitCreatesNotebookDir(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	nbDir := filepath.Join(t.TempDir(), "notes")

	root := NewRootCmd(cfgFile)
	root.SetArgs([]string{"init", "--path", nbDir, "--name", "personal"})
	if err := root.Execute(); err != nil {
		t.Fatalf("nn init: %v", err)
	}

	if _, err := os.Stat(nbDir); err != nil {
		t.Fatalf("notebook dir not created: %v", err)
	}
}

func TestInitInitsGitRepo(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	nbDir := filepath.Join(t.TempDir(), "notes")

	root := NewRootCmd(cfgFile)
	root.SetArgs([]string{"init", "--path", nbDir, "--name", "personal"})
	if err := root.Execute(); err != nil {
		t.Fatalf("nn init: %v", err)
	}

	if _, err := os.Stat(filepath.Join(nbDir, ".git")); err != nil {
		t.Fatalf("git repo not initialised: %v", err)
	}
}

func TestInitDefaultName(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	nbDir := t.TempDir()

	root := NewRootCmd(cfgFile)
	root.SetArgs([]string{"init", "--path", nbDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("nn init without --name: %v", err)
	}

	data, _ := os.ReadFile(cfgFile)
	if !strings.Contains(string(data), "personal") {
		t.Errorf("default name 'personal' not in config: %s", data)
	}
}

func TestInitErrorIfExists(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	nbDir := t.TempDir()

	root := NewRootCmd(cfgFile)
	root.SetArgs([]string{"init", "--path", nbDir})
	root.Execute()

	// Second init without --force should error
	root2 := NewRootCmd(cfgFile)
	root2.SetArgs([]string{"init", "--path", nbDir})
	if err := root2.Execute(); err == nil {
		t.Fatal("second nn init without --force: want error, got nil")
	}
}

func TestInitForceOverwrites(t *testing.T) {
	cfgDir := t.TempDir()
	cfgFile := filepath.Join(cfgDir, "config.toml")
	nbDir := t.TempDir()

	root := NewRootCmd(cfgFile)
	root.SetArgs([]string{"init", "--path", nbDir, "--name", "personal"})
	root.Execute()

	nbDir2 := t.TempDir()
	root2 := NewRootCmd(cfgFile)
	root2.SetArgs([]string{"init", "--path", nbDir2, "--name", "work", "--force"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("nn init --force: %v", err)
	}

	data, _ := os.ReadFile(cfgFile)
	if !strings.Contains(string(data), "work") {
		t.Errorf("--force did not overwrite config: %s", data)
	}
}
