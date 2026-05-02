package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	nnPlugins "github.com/jaresty/nn/plugins"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\[[0-9;]+m\]?`)

func newInstallHooksCmd() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "Install the nn Claude Code plugin (protocol reload + note capture hooks)",
		Long: `Install the nn-hooks Claude Code plugin.

The plugin installs four hooks:

  SessionStart      — reloads global protocol notes into context at the start
                      of every session and after /clear, so protocols remain
                      binding.

  UserPromptSubmit  — emits a per-turn system-reminder instructing the agent
                      to output a ## Protocols block before each response.

  PreCompact        — before context is compacted, spawns an agent to review
                      the session and capture durable knowledge as atomic notes.

  PostCompact       — after compaction, reloads global protocol notes so they
                      remain binding in the new context window.

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

			settingsPath := filepath.Join(home, ".claude", "settings.json")

			// Register the marketplace (best-effort; claude may not be installed).
			addArgs := []string{"plugin", "marketplace", "add", pluginsDir}
			if out, err := exec.Command("claude", addArgs...).CombinedOutput(); err != nil {
				if !isAlreadyExists(string(out)) && !isCommandNotFound(err) {
					return fmt.Errorf("install-hooks: marketplace add: %s: %w", out, err)
				}
			}

			// Install the plugin (best-effort).
			installArgs := []string{"plugin", "install", "nn-hooks@nn-marketplace", "--scope", scope}
			if out, err := exec.Command("claude", installArgs...).CombinedOutput(); err != nil {
				if !isAlreadyInstalled(string(out)) && !isCommandNotFound(err) {
					return fmt.Errorf("install-hooks: plugin install: %s: %w", out, err)
				}
			}

			// Write hooks directly to ~/.claude/settings.json after plugin install,
			// so our prompt-based agent hooks win over any hooks.json merge done by
			// the plugin installer. Plugin hooks (hooks/hooks.json) are broken upstream
			// and don't fire; settings.json hooks with "prompt" field do.
			if err := mergeHooksIntoSettings(settingsPath, home); err != nil {
				return fmt.Errorf("install-hooks: write settings.json hooks: %w", err)
			}

			fmt.Fprintf(outWriter(cmd), "nn-hooks installed (scope: %s)\nHooks written to %s.\nRestart Claude Code to activate the hooks.\n", scope, settingsPath)
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
		perm := fs.FileMode(0o644)
		if strings.HasSuffix(path, ".sh") {
			perm = 0o755
		}
		return os.WriteFile(dst, data, perm)
	})
}

// mergeHooksIntoSettings reads settingsPath (creating it if absent), merges the
// nn hook entries for UserPromptSubmit and SessionStart, and writes it back.
// The plugin cache path is derived from home.
// readAgentPrompt reads an agent definition file from the deployed plugin directory.
// Returns the file content, or a fallback string if the file cannot be read.
func readAgentPrompt(home, name string) string {
	path := filepath.Join(home, ".local", "share", "nn", "plugins", "nn-hooks", "agents", name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "# " + name + "\n\nAgent definition not found. Run nn install to deploy."
	}
	return string(data)
}

func mergeHooksIntoSettings(settingsPath, home string) error {
	// cacheScripts unused — SessionStart and UserPromptSubmit managed by plugin hooks.json

	// Read existing settings or start with empty object.
	data, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", settingsPath, err)
	}
	var settings map[string]interface{}
	if len(data) > 0 {
		clean := ansiEscape.ReplaceAll(data, nil)
		if err := json.Unmarshal(clean, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}
	if settings == nil {
		settings = map[string]interface{}{}
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}

	// Remove stale hook keys that are no longer used.
	delete(hooks, "PreCompact")
	delete(hooks, "PostCompact") // Not valid in current Claude Code — causes entire settings.json to be skipped.
	delete(hooks, "SessionStart") // Protocol loading merged into UserPromptSubmit conditional directive.

	pluginScripts := filepath.Join(home, ".local", "share", "nn", "plugins", "nn-hooks", "scripts")

	hooks["UserPromptSubmit"] = []interface{}{
		map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": `bash "` + filepath.Join(pluginScripts, "protocols-reminder.sh") + `"`,
					"timeout": 5,
				},
			},
		},
	}

	stopScript := filepath.Join(pluginScripts, "nn-stop-hook.sh")
	hooks["Stop"] = []interface{}{
		map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{
					"type":          "command",
					"command":       stopScript,
					"statusMessage": "Capturing and debriefing session...",
					"timeout":       180,
				},
			},
		},
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(settingsPath), err)
	}
	return os.WriteFile(settingsPath, out, 0o644)
}

func isAlreadyExists(out string) bool {
	return strings.Contains(out, "already exists") ||
		strings.Contains(out, "already registered") ||
		strings.Contains(out, "already installed")
}

func isAlreadyInstalled(out string) bool {
	return strings.Contains(out, "already installed")
}

func isCommandNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "executable file not found")
}
