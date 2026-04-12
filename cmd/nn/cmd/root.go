// Package cmd contains all cobra subcommands for the nn CLI.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/config"
)

// rootState holds resolved runtime state shared by all subcommands.
type rootState struct {
	notebookDir string
	backend     backend.Backend
}

// NewRootCmd creates a fresh root command. cfgFile overrides the default config path.
func NewRootCmd(cfgFile string) *cobra.Command {
	state := &rootState{}

	root := &cobra.Command{
		Use:          "nn",
		Short:        "LLM-driven Zettelkasten CLI",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initState(cmd, state, cfgFile)
		},
	}

	root.AddCommand(
		newInitCmd(cfgFile),
		newNewCmd(state),
		newShowCmd(state),
		newListCmd(state),
		newLinkCmd(state),
		newUnlinkCmd(state),
		newGraphCmd(state),
		newStatusCmd(state),
		newPromoteCmd(state),
		newDeleteCmd(state),
		newInstallSkillsCmd(),
	)
	return root
}

// NewRootCmdForTest creates a root command wired to the given config file path.
// Exported for use in tests within the cmd_test package.
func NewRootCmdForTest(cfgFile string) *cobra.Command {
	return NewRootCmd(cfgFile)
}

// initState resolves the notebook directory and initialises the backend.
func initState(cmd *cobra.Command, state *rootState, cfgFile string) error {
	// These commands manage config/skills and don't need a notebook.
	if cmd.Name() == "install-skills" || cmd.Name() == "init" {
		return nil
	}

	nbName := os.Getenv("NN_NOTEBOOK")
	if cfgFile == "" {
		cfgFile = defaultConfigPath()
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		if os.IsNotExist(err) || isNotExistWrapped(err) {
			return fmt.Errorf("no config found at %s — run `nn init --path <notebook-dir>` to get started", cfgFile)
		}
		return fmt.Errorf("load config: %w", err)
	}
	nb, err := cfg.Notebook(nbName)
	if err != nil {
		return fmt.Errorf("resolve notebook: %w", err)
	}

	b, err := gitlocal.New(nb.Path)
	if err != nil {
		return fmt.Errorf("open notebook %q: %w", nb.Path, err)
	}

	state.notebookDir = nb.Path
	state.backend = b
	return nil
}

// defaultConfigPath returns the resolved path to the nn config file.
func defaultConfigPath() string {
	return config.DefaultConfigPath()
}

// outWriter returns the command's output writer.
func outWriter(cmd *cobra.Command) io.Writer {
	return cmd.OutOrStdout()
}

// isNotExistWrapped reports whether err wraps a not-exist error (e.g. from config.Load).
func isNotExistWrapped(err error) bool {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return os.IsNotExist(pathErr)
	}
	return false
}
