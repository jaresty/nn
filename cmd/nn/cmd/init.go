package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

func newInitCmd(cfgFile string) *cobra.Command {
	var (
		nbPath string
		nbName string
		force  bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create nn configuration and initialise a notebook directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if nbPath == "" {
				return fmt.Errorf("--path is required")
			}
			if nbName == "" {
				nbName = "personal"
			}

			// Expand ~ in path
			if len(nbPath) >= 2 && nbPath[:2] == "~/" {
				home, _ := os.UserHomeDir()
				nbPath = filepath.Join(home, nbPath[2:])
			}

			// Check if config already exists
			if _, err := os.Stat(cfgFile); err == nil && !force {
				return fmt.Errorf("config already exists at %s (use --force to overwrite)", cfgFile)
			}

			// Create notebook directory
			if err := os.MkdirAll(nbPath, 0o755); err != nil {
				return fmt.Errorf("init: create notebook dir: %w", err)
			}

			// Init git repo if not already one
			if _, err := os.Stat(filepath.Join(nbPath, ".git")); os.IsNotExist(err) {
				gitCmd := exec.Command("git", "init", "-q")
				gitCmd.Dir = nbPath
				if out, err := gitCmd.CombinedOutput(); err != nil {
					return fmt.Errorf("init: git init: %w\n%s", err, out)
				}
			}

			// Write config
			if err := os.MkdirAll(filepath.Dir(cfgFile), 0o755); err != nil {
				return fmt.Errorf("init: create config dir: %w", err)
			}

			cfg := map[string]any{
				"notebooks": map[string]any{
					"default": nbName,
					nbName: map[string]any{
						"path":    nbPath,
						"backend": "gitlocal",
					},
				},
			}

			f, err := os.Create(cfgFile)
			if err != nil {
				return fmt.Errorf("init: create config: %w", err)
			}
			defer f.Close()
			if err := toml.NewEncoder(f).Encode(cfg); err != nil {
				return fmt.Errorf("init: write config: %w", err)
			}

			fmt.Fprintf(outWriter(cmd), "initialised notebook %q at %s\n", nbName, nbPath)
			fmt.Fprintf(outWriter(cmd), "config written to %s\n", cfgFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&nbPath, "path", "", "Path to the notebook directory (required)")
	cmd.Flags().StringVar(&nbName, "name", "personal", "Notebook name")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config")
	return cmd
}
