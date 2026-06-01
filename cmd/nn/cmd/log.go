package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func newLogCmd(state *rootState) *cobra.Command {
	var since string

	cmd := &cobra.Command{
		Use:   "log <id-or-title>",
		Short: "Show git history for a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("log: %w", err)
			}

			gitArgs := []string{"log", "-p", "--follow"}
			if since != "" {
				gitArgs = append(gitArgs, "--since="+since)
			}
			gitArgs = append(gitArgs, "--", n.Filename())

			c := exec.Command("git", gitArgs...)
			c.Dir = state.notebookDir
			out, err := c.Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return fmt.Errorf("git log: %s", exitErr.Stderr)
				}
				return fmt.Errorf("git log: %w", err)
			}
			fmt.Fprint(outWriter(cmd), string(out))
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "Limit history to commits after this date (e.g. 2025-01-01)")
	return cmd
}
