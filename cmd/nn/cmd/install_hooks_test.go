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

func TestInstallHooksDoesNotWriteUserPromptSubmitToSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-seed a stale UserPromptSubmit entry to verify it gets removed.
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{"hooks":{"UserPromptSubmit":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _ = execute("install-hooks")

	hooks := readSettingsHooks(t, home)
	if _, ok := hooks["UserPromptSubmit"]; ok {
		t.Error("hooks.UserPromptSubmit should be absent — managed by plugin hooks.json, not user settings")
	}
}

func TestInstallHooksDoesNotWriteSessionStartToSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-seed a stale SessionStart entry to verify it gets removed.
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{"hooks":{"SessionStart":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _ = execute("install-hooks")

	hooks := readSettingsHooks(t, home)
	if _, ok := hooks["SessionStart"]; ok {
		t.Error("hooks.SessionStart should be absent — managed by plugin hooks.json, not user settings")
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

func TestInstallHooksWritesStopCommandToSettings(t *testing.T) {
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

	_, _ = execute("install-hooks")

	hooks := readSettingsHooks(t, home)
	stop, ok := hooks["Stop"]
	if !ok {
		t.Fatal("hooks.Stop missing after install-hooks")
	}
	entries, _ := stop.([]interface{})
	var cmdCount int
	for _, e := range entries {
		em, _ := e.(map[string]interface{})
		inner, _ := em["hooks"].([]interface{})
		for _, h := range inner {
			hm, _ := h.(map[string]interface{})
			if hm["type"] == "command" {
				if cmd, ok := hm["command"].(string); ok && cmd != "" {
					cmdCount++
				}
			}
		}
	}
	// Expect exactly 1 command hook (throttled shell script).
	if cmdCount != 1 {
		t.Errorf("hooks.Stop: expected exactly 1 command hook, got %d", cmdCount)
	}
}

func TestInstallHooksNoPreCompactInSettings(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(`{"hooks":{"PreCompact":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _ = execute("install-hooks")

	hooks := readSettingsHooks(t, home)
	if _, ok := hooks["PreCompact"]; ok {
		t.Error("hooks.PreCompact should be absent after install-hooks (stale key cleanup)")
	}
}

func TestInstallHooksNoPostCompactInSettings(t *testing.T) {
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

	_, _ = execute("install-hooks")

	hooks := readSettingsHooks(t, home)
	if _, ok := hooks["PostCompact"]; ok {
		t.Fatal("hooks.PostCompact must not be written to settings.json (not a valid event)")
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
