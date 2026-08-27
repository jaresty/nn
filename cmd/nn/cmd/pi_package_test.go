package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPiPackageManifestExposesGlobalContextExtension(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	manifestPath := filepath.Join(repoRoot, "package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read Pi package manifest: %v", err)
	}

	var manifest struct {
		Keywords []string `json:"keywords"`
		Files    []string `json:"files"`
		Pi       struct {
			Extensions []string `json:"extensions"`
			Skills     []string `json:"skills"`
		} `json:"pi"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse Pi package manifest: %v", err)
	}

	if !containsString(manifest.Keywords, "pi-package") {
		t.Fatalf("Pi package manifest missing pi-package keyword: %v", manifest.Keywords)
	}
	if !containsString(manifest.Pi.Extensions, "./pi/extensions") {
		t.Fatalf("Pi package manifest missing ./pi/extensions: %v", manifest.Pi.Extensions)
	}
	if !containsString(manifest.Pi.Skills, "./skills") {
		t.Fatalf("Pi package manifest missing ./skills: %v", manifest.Pi.Skills)
	}
	for _, packagedRoot := range []string{"pi/extensions", "skills"} {
		if !containsString(manifest.Files, packagedRoot) {
			t.Fatalf("Pi install package files must recursively include %q: %v", packagedRoot, manifest.Files)
		}
	}

	navigatePath := filepath.Join(repoRoot, "skills", "nn-navigate", "SKILL.md")
	navigate, err := os.ReadFile(navigatePath)
	if err != nil {
		t.Fatalf("Pi package does not ship nn-navigate at %s: %v", navigatePath, err)
	}
	if !strings.Contains(string(navigate), "name: nn-navigate") {
		t.Fatalf("packaged nn-navigate has invalid frontmatter: %s", navigate)
	}
	for _, name := range []string{"ask", "integrate", "lenses", "movement", "presentation", "scan-and-routes", "state"} {
		path := filepath.Join(repoRoot, "skills", "nn-navigate", "references", name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Pi install package does not recursively ship %s: %v", path, err)
		}
		if !strings.Contains(string(data), "applies_when:") {
			t.Fatalf("packaged reference %s lacks applicability metadata", path)
		}
	}

	extensionPath := filepath.Join(repoRoot, "pi", "extensions", "nn_global_context.ts")
	extension, err := os.ReadFile(extensionPath)
	if err != nil {
		t.Fatalf("read Pi global context extension: %v", err)
	}
	extensionText := string(extension)
	for _, expected := range []string{"session_start", "before_agent_start", "nn show --global"} {
		if !strings.Contains(extensionText, expected) {
			t.Fatalf("Pi global context extension missing %q", expected)
		}
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
