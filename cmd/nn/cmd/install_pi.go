package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	nnPi "github.com/jaresty/nn/pi"
)

func newInstallPiCmd() *cobra.Command {
	var (
		extensionsDest string
		skillsDest     string
	)

	cmd := &cobra.Command{
		Use:   "install-pi",
		Short: "Install nn support for Pi",
		Long: `Install nn support for Pi.

This installs:
  - nn skills into ~/.pi/agent/skills/
  - the nn global context extension into ~/.pi/agent/extensions/

Restart Pi or run /reload after installing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if extensionsDest == "" || skillsDest == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("install-pi: resolve home: %w", err)
				}
				if extensionsDest == "" {
					extensionsDest = filepath.Join(home, ".pi", "agent", "extensions")
				}
				if skillsDest == "" {
					skillsDest = filepath.Join(home, ".pi", "agent", "skills")
				}
			}

			if err := copyPiExtensions(extensionsDest); err != nil {
				return fmt.Errorf("install-pi: copy extensions: %w", err)
			}
			if err := installSkillsToDest(skillsDest); err != nil {
				return fmt.Errorf("install-pi: copy skills: %w", err)
			}

			fmt.Fprintf(outWriter(cmd), "nn Pi support installed\nExtensions: %s\nSkills: %s\nRestart Pi or run /reload to activate.\n", extensionsDest, skillsDest)
			return nil
		},
	}

	cmd.Flags().StringVar(&extensionsDest, "extensions-dest", "", "Custom Pi extensions directory (default: ~/.pi/agent/extensions)")
	cmd.Flags().StringVar(&skillsDest, "skills-dest", "", "Custom Pi skills directory (default: ~/.pi/agent/skills)")
	return cmd
}

func copyPiExtensions(destDir string) error {
	return fs.WalkDir(nnPi.FS, "extensions", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("extensions", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(destDir, 0o755)
		}
		dst := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := nnPi.FS.ReadFile(path)
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
