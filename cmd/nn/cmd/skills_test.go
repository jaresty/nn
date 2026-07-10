package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestSkillsList(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "list")
	if err != nil {
		t.Fatalf("nn skills list: %v", err)
	}
	for _, skill := range []string{"nn-workflow", "nn-guide", "nn-capture-discipline", "nn-session-debrief"} {
		if !strings.Contains(out, skill) {
			t.Errorf("skills list: missing %q in output: %q", skill, out)
		}
	}
}

func TestSkillsGet(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "get", "nn-workflow")
	if err != nil {
		t.Fatalf("nn skills get nn-workflow: %v", err)
	}
	if !strings.Contains(out, "nn-workflow") {
		t.Errorf("skills get nn-workflow: expected skill content, got: %q", out)
	}
}

func TestSkillsGetUnknownErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("skills", "get", "no-such-skill")
	if err == nil {
		t.Fatal("skills get no-such-skill: want error, got nil")
	}
}

func TestSkillsGetNoNameErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("skills", "get")
	if err == nil {
		t.Fatal("skills get (no name): want error, got nil")
	}
}

func TestInstallSkillsOutputShowsStubPath(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	out, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}
	if strings.Contains(out, "nn-workflow") {
		t.Errorf("install-skills output should not list per-skill names, got: %q", out)
	}
	if !strings.Contains(out, "nn/SKILL.md") {
		t.Errorf("install-skills output should show stub path containing 'nn/SKILL.md', got: %q", out)
	}
}

func TestInstallSkillsStub(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	_, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}

	stubPath := destDir + "/nn/SKILL.md"
	data, readErr := os.ReadFile(stubPath)
	if readErr != nil {
		t.Fatalf("stub not created at %s: %v", stubPath, readErr)
	}
	if !strings.Contains(string(data), "nn skills get") {
		t.Errorf("stub at %s does not instruct Claude to call 'nn skills get': %q", stubPath, string(data))
	}
}

func TestInstallSkillsRemovesDeprecated(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	// Pre-create a deprecated per-skill dir to simulate old install.
	oldSkillDir := destDir + "/nn-workflow"
	if err := os.MkdirAll(oldSkillDir+"/", 0o755); err != nil {
		t.Fatalf("setup deprecated dir: %v", err)
	}
	oldFile := oldSkillDir + "/SKILL.md"
	if err := os.WriteFile(oldFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("setup deprecated file: %v", err)
	}

	_, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}

	if _, statErr := os.Stat(oldSkillDir); !os.IsNotExist(statErr) {
		t.Errorf("deprecated skill dir %s was not removed", oldSkillDir)
	}
}
