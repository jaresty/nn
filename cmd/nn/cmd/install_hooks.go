package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	nnPlugins "github.com/jaresty/nn/plugins"
)

func newInstallHooksCmd() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "Install the nn Claude Code plugin (pre-compaction note capture)",
		Long: `Install the nn-hooks Claude Code plugin.

The plugin adds a pre-compaction hook that spawns an agent to review the
session and decide what knowledge is worth capturing as notes before context
is compacted.

Scopes:
  user     ~/.claude/settings.json (default, global)
  project  .claude/settings.json (shared with team)
  local    .claude/settings.local.json (gitignored)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Write embedded plugin to ~/.local/share/nn/plugins/
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("install-hooks: resolve home: %w", err)
			}
			pluginsDir := filepath.Join(home, ".local", "share", "nn", "plugins")
			if err := copyPlugins(pluginsDir); err != nil {
				return fmt.Errorf("install-hooks: copy plugin: %w", err)
			}

			// Register the marketplace.
			addArgs := []string{"plugin", "marketplace", "add", pluginsDir}
			if out, err := exec.Command("claude", addArgs...).CombinedOutput(); err != nil {
				// Ignore "already exists" errors.
				if !isAlreadyExists(string(out)) {
					return fmt.Errorf("install-hooks: marketplace add: %s: %w", out, err)
				}
			}

			// Install the plugin.
			installArgs := []string{"plugin", "install", "nn-hooks@nn-marketplace", "--scope", scope}
			if out, err := exec.Command("claude", installArgs...).CombinedOutput(); err != nil {
				if !isAlreadyInstalled(string(out)) {
					return fmt.Errorf("install-hooks: plugin install: %s: %w", out, err)
				}
			}

			fmt.Fprintf(outWriter(cmd), "nn-hooks installed (scope: %s)\nRestart Claude Code to activate the pre-compaction hook.\n", scope)
			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Installation scope: user, project, or local")
	return cmd
}

// copyPlugins writes the embedded plugins directory to destDir.
func copyPlugins(destDir string) error {
	return fs.WalkDir(nnPlugins.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(destDir, path)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := nnPlugins.FS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

func isAlreadyExists(out string) bool {
	return strings.Contains(out, "already exists") || strings.Contains(out, "already registered")
}

func isAlreadyInstalled(out string) bool {
	return strings.Contains(out, "already installed")
}
