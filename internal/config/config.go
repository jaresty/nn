// Package config handles loading and resolving nn configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// NotebookEntry holds the configuration for a single named notebook.
type NotebookEntry struct {
	Path    string `toml:"path"`
	Backend string `toml:"backend"`
}

// notebooks is the intermediate TOML structure for [notebooks.*] sections.
type notebooksSection struct {
	Default string                   `toml:"default"`
	Named   map[string]NotebookEntry `toml:"-"`
}

// Config holds the full parsed configuration.
type Config struct {
	defaultName string
	notebooks   map[string]NotebookEntry
}

type rawConfig struct {
	Notebooks map[string]any `toml:"notebooks"`
}

// Load parses the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}

	// Use a generic map to handle the mixed [notebooks] structure.
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("config.Load: toml: %w", err)
	}

	cfg := &Config{notebooks: make(map[string]NotebookEntry)}

	nbSection, ok := raw["notebooks"].(map[string]any)
	if !ok {
		return cfg, nil
	}

	if def, ok := nbSection["default"].(string); ok {
		cfg.defaultName = def
	}

	for k, v := range nbSection {
		if k == "default" {
			continue
		}
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		nb := NotebookEntry{}
		if p, ok := entry["path"].(string); ok {
			nb.Path = expandHome(p)
		}
		if b, ok := entry["backend"].(string); ok {
			nb.Backend = b
		}
		cfg.notebooks[k] = nb
	}
	return cfg, nil
}

// Notebook returns the notebook entry for the given name.
// If name is empty, it returns the default notebook.
func (c *Config) Notebook(name string) (NotebookEntry, error) {
	if name == "" {
		name = c.defaultName
	}
	nb, ok := c.notebooks[name]
	if !ok {
		return NotebookEntry{}, fmt.Errorf("config: notebook %q not found", name)
	}
	return nb, nil
}

// DefaultConfigPath returns the default path to the nn config file.
// Respects NN_CONFIG_DIR environment variable for testability.
func DefaultConfigPath() string {
	if dir := os.Getenv("NN_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.toml")
	}
	// Use XDG_CONFIG_HOME if set, otherwise ~/.config
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "nn", "config.toml")
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
