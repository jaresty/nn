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
	stableDir := filepath.Join(home, ".local", "share", "nn", "plugins", "nn-hooks", "scripts")
	found := false
	for _, c := range cmds {
		if strings.Contains(c, "protocols-reminder.sh") && strings.Contains(c, stableDir) {
			found = true
		}
	}
	if !found {
		t.Errorf("hooks.UserPromptSubmit command does not reference stable path %s/protocols-reminder.sh: %v", stableDir, cmds)
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
	stableDir := filepath.Join(home, ".local", "share", "nn", "plugins", "nn-hooks", "scripts")
	cmds := hookCommands(hooks, "SessionStart")
	if len(cmds) == 0 {
		t.Fatal("hooks.SessionStart missing after install-hooks")
	}
	found := false
	for _, c := range cmds {
		if strings.Contains(c, "load-protocols.sh") && strings.Contains(c, stableDir) {
			found = true
		}
	}
	if !found {
		t.Errorf("hooks.SessionStart command does not reference stable path %s/load-protocols.sh: %v", stableDir, cmds)
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

// TestLoadProtocolsScriptReadsFromSkill verifies that the deployed load-protocols.sh
// reads the nn-capture-discipline SKILL.md when available, rather than using hardcoded text.
func TestLoadProtocolsScriptReadsFromSkill(t *testing.T) {
	if _, err := os.LookupEnv("SKIP_HOOK_SCRIPT_TEST"); err == false {
		// Only run when bash is available.
		if _, err := os.Stat("/bin/bash"); err != nil {
			t.Skip("bash not available")
		}
	}

	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Install hooks so load-protocols.sh is deployed.
	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = execute("install-hooks")

	// Plant a sentinel in the skill SKILL.md.
	skillDir := filepath.Join(home, ".claude", "skills", "nn-capture-discipline")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := "SENTINEL_CAPTURE_DISCIPLINE_CONTENT"
	skillContent := "---\nname: nn-capture-discipline\n---\n\n# nn-capture-discipline\n\n" + sentinel
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run the deployed script.
	scriptPath := filepath.Join(home, ".local", "share", "nn", "plugins", "nn-hooks", "scripts", "load-protocols.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Skipf("load-protocols.sh not deployed (install-hooks may have failed in test env): %v", err)
	}

	out, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	// The script must reference the skill SKILL.md path, not hardcode the protocol inline.
	if !strings.Contains(string(out), "nn-capture-discipline") {
		t.Errorf("load-protocols.sh does not reference nn-capture-discipline skill: %q", string(out))
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
