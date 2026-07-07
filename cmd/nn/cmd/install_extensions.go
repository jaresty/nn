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

func newInstallExtensionsCmd() *cobra.Command {
	var extensionsDest string

	cmd := &cobra.Command{
		Use:   "install-extensions",
		Short: "Install nn Pi extensions into the Pi extensions directory",
		Long: `Install nn Pi extensions into the Pi agent extensions directory.

Default destination: ~/.pi/agent/extensions

Restart Pi or run /reload after installing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if extensionsDest == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("install-extensions: resolve home: %w", err)
				}
				extensionsDest = filepath.Join(home, ".pi", "agent", "extensions")
			}
			if err := copyPiExtensions(extensionsDest); err != nil {
				return fmt.Errorf("install-extensions: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "nn Pi extensions installed\nExtensions: %s\nRestart Pi or run /reload to activate.\n", extensionsDest)
			return nil
		},
	}

	cmd.Flags().StringVar(&extensionsDest, "extensions-dest", "", "Custom Pi extensions directory (default: ~/.pi/agent/extensions)")
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
