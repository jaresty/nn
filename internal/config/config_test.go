package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaresty/nn/internal/config"
)

func TestLoadDefault(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgFile, []byte(`
[notebooks]
default = "personal"

[notebooks.personal]
path = "/tmp/notes"
backend = "gitlocal"
`), 0o644)

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	nb, err := cfg.Notebook("")
	if err != nil {
		t.Fatalf("Notebook: %v", err)
	}
	if nb.Path != "/tmp/notes" {
		t.Errorf("Path = %q, want /tmp/notes", nb.Path)
	}
	if nb.Backend != "gitlocal" {
		t.Errorf("Backend = %q, want gitlocal", nb.Backend)
	}
}

func TestLoadNamedNotebook(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgFile, []byte(`
[notebooks]
default = "personal"

[notebooks.personal]
path = "/tmp/personal"
backend = "gitlocal"

[notebooks.work]
path = "/tmp/work"
backend = "gitlocal"
`), 0o644)

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	nb, err := cfg.Notebook("work")
	if err != nil {
		t.Fatalf("Notebook(work): %v", err)
	}
	if nb.Path != "/tmp/work" {
		t.Errorf("Path = %q, want /tmp/work", nb.Path)
	}
}

func TestLoadMissingNotebook(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgFile, []byte(`
[notebooks]
default = "personal"

[notebooks.personal]
path = "/tmp/personal"
backend = "gitlocal"
`), 0o644)

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	_, err = cfg.Notebook("nonexistent")
	if err == nil {
		t.Fatal("Notebook(nonexistent): want error, got nil")
	}
}
