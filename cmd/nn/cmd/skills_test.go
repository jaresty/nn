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
	for _, skill := range []string{"nn-workflow", "nn-guide", "nn-navigate", "nn-capture-discipline", "nn-session-debrief"} {
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

func TestSkillDescriptionsLeadWithUseWhen(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "list")
	if err != nil {
		t.Fatalf("nn skills list: %v", err)
	}
	for _, skill := range []string{"nn-workflow", "nn-guide", "nn-navigate", "nn-capture-discipline", "nn-session-debrief", "nn-link-suggester", "nn-refine", "nn-refine-workflow"} {
		content, readErr := execute("skills", "get", skill)
		if readErr != nil {
			t.Fatalf("nn skills get %s: %v", skill, readErr)
		}
		// description field must start with "Use when"
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "description:") {
				if !strings.Contains(line, "Use when") {
					t.Errorf("skill %s: description does not lead with 'Use when': %q", skill, line)
				}
				break
			}
		}
	}
	// stub must gate on nn skills list, not list skills statically
	if !strings.Contains(out, "Use when") {
		t.Errorf("nn skills list output does not contain 'Use when' routing triggers: %q", out)
	}
	// descriptions must not tell Claude to use slash commands — those no longer exist as installed skills
	if strings.Contains(out, "Invoke with /nn-") {
		t.Errorf("nn skills list output contains stale slash-command invocation 'Invoke with /nn-': %q", out)
	}
}

func TestStubGatesOnSkillsList(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()
	if _, err := execute("install-skills", "--dest", destDir); err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}
	data, err := os.ReadFile(destDir + "/nn/SKILL.md")
	if err != nil {
		t.Fatalf("stub not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "nn skills list") {
		t.Errorf("stub does not instruct Claude to run 'nn skills list': %q", content)
	}
	if strings.Contains(content, "nn skills get nn-workflow") {
		t.Errorf("stub contains hardcoded 'nn skills get nn-workflow' — should use live output instead")
	}
}
