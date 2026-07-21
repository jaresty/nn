package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newTodoCmd(state *rootState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo",
		Short: "Manage todo checkboxes in notes",
	}
	cmd.AddCommand(newTodoListCmd(state), newTodoDoneCmd(state), newTodoReopenCmd(state))
	return cmd
}

func newTodoListCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List open checkboxes grouped by note",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("todo list: %w", err)
			}
			w := outWriter(cmd)
			first := true
			for _, n := range notes {
				var open []string
				for _, line := range strings.Split(n.Body, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "- [ ]") {
						open = append(open, line)
					}
				}
				if len(open) == 0 {
					continue
				}
				if !first {
					fmt.Fprintln(w)
				}
				first = false
				fmt.Fprintf(w, "%s  %s\n", n.ID, n.Title)
				for _, line := range open {
					fmt.Fprintf(w, "  %s\n", strings.TrimSpace(line))
				}
			}
			return nil
		},
	}
}

func newTodoDoneCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "done <id> <pattern>",
		Short: "Mark the first matching open checkbox as done",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flipCheckbox(cmd, state, args[0], args[1], "- [ ]", "- [x]", "open")
		},
	}
}

func newTodoReopenCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <id> <pattern>",
		Short: "Mark the first matching done checkbox as open",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flipCheckbox(cmd, state, args[0], args[1], "- [x]", "- [ ]", "done")
		},
	}
}

func flipCheckbox(cmd *cobra.Command, state *rootState, id, pattern, from, to, fromLabel string) error {
	n, err := resolveNote(state, id)
	if err != nil {
		return fmt.Errorf("todo: %w", err)
	}

	lines := strings.Split(n.Body, "\n")
	lowerPattern := strings.ToLower(pattern)
	matched := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, from) && strings.Contains(strings.ToLower(line), lowerPattern) {
			matched = i
			break
		}
	}
	if matched == -1 {
		return fmt.Errorf("no %s checkbox matching %q found in note %s", fromLabel, pattern, n.ID)
	}

	lines[matched] = strings.Replace(lines[matched], from, to, 1)
	n.Body = strings.Join(lines, "\n")
	n.Modified = time.Now().In(time.Local)

	if err := state.backend.Update(n); err != nil {
		return fmt.Errorf("todo: %w", err)
	}
	fmt.Fprintf(outWriter(cmd), "updated %s\nmodified: %s\n", n.ID, n.Modified.Format(time.RFC3339))
	return nil
}
