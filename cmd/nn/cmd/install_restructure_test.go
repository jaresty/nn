package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInstallPiCommandRemoved asserts D1: install-pi is no longer registered on root.
func TestInstallPiCommandRemoved(t *testing.T) {
	root := NewRootCmdForTest("")
	for _, sub := range root.Commands() {
		if sub.Name() == "install-pi" {
			t.Fatalf("install-pi command must not be registered on root; found it")
		}
	}
}

// TestInstallExtensionsCommandExists asserts D2: install-extensions is registered on root.
func TestInstallExtensionsCommandExists(t *testing.T) {
	root := NewRootCmdForTest("")
	for _, sub := range root.Commands() {
		if sub.Name() == "install-extensions" {
			return
		}
	}
	t.Fatalf("install-extensions command must be registered on root; not found")
}

// TestInstallExtensionsCopiesFiles asserts D2: install-extensions copies Pi extension files.
func TestInstallExtensionsCopiesFiles(t *testing.T) {
	dest := t.TempDir()
	root := NewRootCmdForTest("")
	root.SetArgs([]string{"install-extensions", "--extensions-dest", dest})
	if err := root.Execute(); err != nil {
		t.Fatalf("install-extensions: %v", err)
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("read extensions dest: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("install-extensions: no files copied to %s", dest)
	}
	found := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".ts" {
			found = true
		}
	}
	if !found {
		t.Fatalf("install-extensions: no .ts files found in %s", dest)
	}
}

// TestInstallForPiRoutesToSkillsAndExtensions asserts D3: nn install --for pi
// invokes both install-skills --for pi and install-extensions without error.
func TestInstallForPiRoutesToSkillsAndExtensions(t *testing.T) {
	skillsDest := t.TempDir()
	extDest := t.TempDir()
	root := NewRootCmdForTest("")
	root.SetArgs([]string{
		"install",
		"--for", "pi",
		"--skills-dest", skillsDest,
		"--extensions-dest", extDest,
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("install --for pi: %v", err)
	}
	skillsEntries, err := os.ReadDir(skillsDest)
	if err != nil {
		t.Fatalf("read skills dest: %v", err)
	}
	if len(skillsEntries) == 0 {
		t.Fatalf("install --for pi: no skills copied to %s", skillsDest)
	}
	extEntries, err := os.ReadDir(extDest)
	if err != nil {
		t.Fatalf("read extensions dest: %v", err)
	}
	if len(extEntries) == 0 {
		t.Fatalf("install --for pi: no extensions copied to %s", extDest)
	}
}
