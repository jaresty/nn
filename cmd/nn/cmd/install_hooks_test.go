package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// settingsHooks reads ~/.claude/settings.json and returns the hooks subtree as a map.
func readSettingsHooks(t *testing.T, home string) map[string]interface{} {
	t.Helper()
	path := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readSettingsHooks: read %s: %v", path, err)
	}
	var full map[string]interface{}
	if err := json.Unmarshal(data, &full); err != nil {
		t.Fatalf("readSettingsHooks: parse: %v", err)
	}
	hooks, _ := full["hooks"].(map[string]interface{})
	return hooks
}

// hookCommands extracts the command strings from a hooks event entry.
func hookCommands(hooks map[string]interface{}, event string) []string {
	raw, ok := hooks[event]
	if !ok {
		return nil
	}
	entries, _ := raw.([]interface{})
	var cmds []string
	for _, e := range entries {
		em, _ := e.(map[string]interface{})
		inner, _ := em["hooks"].([]interface{})
		for _, h := range inner {
			hm, _ := h.(map[string]interface{})
			if cmd, ok := hm["command"].(string); ok {
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

func TestInstallHooksWritesUserPromptSubmitToSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	// pre-create settings.json with an unrelated key so we can verify merge
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := []byte(`{"model":"sonnet"}`)
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), existing, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("install-hooks")
	_ = out
	// claude plugin commands will fail in test env — that's expected; we only care about settings.json
	_ = err

	hooks := readSettingsHooks(t, home)
	if hooks == nil {
		t.Fatal("hooks key missing from settings.json after install-hooks")
	}
	cmds := hookCommands(hooks, "UserPromptSubmit")
	if len(cmds) == 0 {
		t.Fatal("hooks.UserPromptSubmit missing after install-hooks")
	}
	found := false
	for _, c := range cmds {
		if strings.Contains(c, "protocols-reminder.sh") {
			found = true
		}
	}
	if !found {
		t.Errorf("hooks.UserPromptSubmit command does not reference protocols-reminder.sh: %v", cmds)
	}
}

func TestInstallHooksWritesSessionStartToSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("install-hooks")
	_ = out
	_ = err

	hooks := readSettingsHooks(t, home)
	cmds := hookCommands(hooks, "SessionStart")
	if len(cmds) == 0 {
		t.Fatal("hooks.SessionStart missing after install-hooks")
	}
	found := false
	for _, c := range cmds {
		if strings.Contains(c, "load-protocols.sh") {
			found = true
		}
	}
	if !found {
		t.Errorf("hooks.SessionStart command does not reference load-protocols.sh: %v", cmds)
	}
}

func TestInstallHooksMergesExistingSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := []byte(`{"model":"opus","effortLevel":"low"}`)
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), existing, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := execute("install-hooks")
	_ = out
	_ = err

	data, err := os.ReadFile(filepath.Join(settingsDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var full map[string]interface{}
	if err := json.Unmarshal(data, &full); err != nil {
		t.Fatalf("parse settings.json: %v", err)
	}
	if full["model"] != "opus" {
		t.Errorf("existing model key clobbered: got %v", full["model"])
	}
	if full["effortLevel"] != "low" {
		t.Errorf("existing effortLevel key clobbered: got %v", full["effortLevel"])
	}
}

func TestInstallHooksSuccessMessageMentionsSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _ := execute("install-hooks")
	if !strings.Contains(out, "settings.json") {
		t.Errorf("success message does not mention settings.json: %q", out)
	}
}
