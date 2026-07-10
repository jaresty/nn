package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend/gitlocal"
)

func newDrainCmd(state *rootState) *cobra.Command {
	var statusOnly bool

	cmd := &cobra.Command{
		Use:   "drain",
		Short: "Drain the commit queue, committing any pending git operations",
		Long: `Drain forces all pending commit queue items to be committed to git.

Normally the commit queue is drained automatically by whichever nn process
wins the drain lock. Use this command to force a drain after a suspected
crash, or before a git push to ensure all commits are present.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := nnConfigDir()
			queueDir := filepath.Join(configDir, "commit-queue")
			lockFile := filepath.Join(configDir, "commit-queue.lock")

			if statusOnly {
				return drainStatus(cmd, queueDir, lockFile)
			}

			if state.backend == nil {
				return fmt.Errorf("drain: no notebook loaded")
			}
			nb := state.notebookDir
			return gitlocal.DrainQueue(configDir, nb)
		},
	}
	cmd.Flags().BoolVar(&statusOnly, "status", false, "Print queue depth and lock holder without draining")
	return cmd
}

func drainStatus(cmd *cobra.Command, queueDir, lockFile string) error {
	entries, err := os.ReadDir(queueDir)
	if os.IsNotExist(err) {
		entries = nil
	} else if err != nil {
		return fmt.Errorf("drain status: %w", err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "queue depth: %d\n", count)

	data, err := os.ReadFile(lockFile)
	if os.IsNotExist(err) {
		fmt.Fprintln(cmd.OutOrStdout(), "lock: not held")
		return nil
	}
	if err != nil {
		return fmt.Errorf("drain status: read lock: %w", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	alive := pid > 0 && func() bool {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		return proc.Signal(syscall.Signal(0)) == nil
	}()
	if alive {
		fmt.Fprintf(cmd.OutOrStdout(), "lock: held by pid %d (alive)\n", pid)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "lock: stale (pid %d dead)\n", pid)
	}
	return nil
}

