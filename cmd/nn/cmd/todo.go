package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func listTodosText(notes []*note.Note, byID map[string]*note.Note) string {
	var sb strings.Builder
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
		if isBlocked(n, byID) {
			continue
		}
		if !first {
			fmt.Fprintln(&sb)
		}
		first = false
		fmt.Fprintf(&sb, "%s  %s\n", n.ID, n.Title)
		for _, line := range open {
			fmt.Fprintf(&sb, "  %s\n", strings.TrimSpace(line))
		}
	}
	return sb.String()
}

// isWaiting reports whether a checkbox line contains a [waiting: ...] tag.
func isWaiting(line string) (bool, string) {
	trimmed := strings.TrimSpace(line)
	const prefix = "[waiting:"
	idx := strings.Index(strings.ToLower(trimmed), prefix)
	if idx == -1 {
		return false, ""
	}
	rest := trimmed[idx+len(prefix):]
	end := strings.Index(rest, "]")
	if end == -1 {
		return false, ""
	}
	return true, strings.TrimSpace(rest[:end])
}

// todoContext returns the @context value from a checkbox line, or "" if absent.
func todoContext(line string) string {
	for _, word := range strings.Fields(line) {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			return strings.ToLower(word[1:])
		}
	}
	return ""
}

func newTodoCmd(state *rootState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo",
		Short: "Manage todo checkboxes in notes",
	}
	cmd.AddCommand(newTodoListCmd(state), newTodoDoneCmd(state), newTodoReopenCmd(state), newTodoSetCmd(state))
	return cmd
}

// isBlocked reports whether n has at least one requires link pointing to a note
// that is not done (has unchecked checkboxes or has checkboxes and not all checked).
func isBlocked(n *note.Note, byID map[string]*note.Note) bool {
	for _, lnk := range n.Links {
		if lnk.Type != "requires" {
			continue
		}
		target, ok := byID[lnk.TargetID]
		if !ok {
			continue
		}
		if !note.IsDone(target.Body) {
			return true
		}
	}
	return false
}

func newTodoListCmd(state *rootState) *cobra.Command {
	var showAll bool
	var showWaiting bool
	var contextFilter string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List open checkboxes grouped by note",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("todo list: %w", err)
			}
			byID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
			}
			w := outWriter(cmd)
			first := true
			for _, n := range notes {
				var open []string
				for _, line := range strings.Split(n.Body, "\n") {
					if !strings.HasPrefix(strings.TrimSpace(line), "- [ ]") {
						continue
					}
					waiting, _ := isWaiting(line)
					if showWaiting {
						if !waiting {
							continue
						}
					} else {
						if waiting {
							continue
						}
					}
					if contextFilter != "" {
						if todoContext(line) != strings.ToLower(contextFilter) {
							continue
						}
					}
					open = append(open, line)
				}
				if len(open) == 0 {
					continue
				}
				if !showAll && !showWaiting && isBlocked(n, byID) {
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
	cmd.Flags().BoolVar(&showAll, "all", false, "Show all notes with open items, including blocked ones")
	cmd.Flags().BoolVar(&showWaiting, "waiting", false, "Show only items tagged [waiting: reason]")
	cmd.Flags().StringVar(&contextFilter, "context", "", "Show only items tagged with @context")
	return cmd
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

func newTodoSetCmd(state *rootState) *cobra.Command {
	var waitingReason string
	var clearWaiting bool
	var contextName string
	var clearContext bool
	cmd := &cobra.Command{
		Use:   "set <id> <pattern>",
		Short: "Set or clear inline metadata ([waiting:], @context) on a matching open checkbox",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, pattern := args[0], args[1]
			n, err := resolveNote(state, id)
			if err != nil {
				return fmt.Errorf("todo set: %w", err)
			}
			lines := strings.Split(n.Body, "\n")
			lowerPattern := strings.ToLower(pattern)
			matched := -1
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "- [ ]") && strings.Contains(strings.ToLower(line), lowerPattern) {
					matched = i
					break
				}
			}
			if matched == -1 {
				return fmt.Errorf("no open checkbox matching %q found in note %s", pattern, n.ID)
			}
			line := lines[matched]
			if waitingReason != "" {
				// replace existing [waiting: ...] or prepend after "- [ ] "
				if ok, _ := isWaiting(line); ok {
					line = replaceWaitingTag(line, waitingReason)
				} else {
					line = insertAfterCheckbox(line, "[waiting: "+waitingReason+"] ")
				}
			}
			if clearWaiting {
				line = removeWaitingTag(line)
			}
			if contextName != "" {
				if todoContext(line) != "" {
					line = replaceContextTag(line, contextName)
				} else {
					line = insertAfterCheckbox(line, "@"+contextName+" ")
				}
			}
			if clearContext {
				line = removeContextTag(line)
			}
			lines[matched] = line
			n.Body = strings.Join(lines, "\n")
			n.Modified = time.Now().In(time.Local)
			if err := state.backend.Update(n); err != nil {
				return fmt.Errorf("todo set: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "updated %s\nmodified: %s\n", n.ID, n.Modified.Format(time.RFC3339))
			return nil
		},
	}
	cmd.Flags().StringVar(&waitingReason, "waiting", "", "Add or replace [waiting: reason] tag")
	cmd.Flags().BoolVar(&clearWaiting, "clear-waiting", false, "Remove [waiting: ...] tag")
	cmd.Flags().StringVar(&contextName, "context", "", "Add or replace @context tag")
	cmd.Flags().BoolVar(&clearContext, "clear-context", false, "Remove @context tag")
	return cmd
}

// insertAfterCheckbox inserts text immediately after the "- [ ] " prefix.
func insertAfterCheckbox(line, insert string) string {
	const prefix = "- [ ] "
	idx := strings.Index(line, prefix)
	if idx == -1 {
		return line
	}
	return line[:idx+len(prefix)] + insert + line[idx+len(prefix):]
}

// replaceWaitingTag replaces an existing [waiting: ...] tag with a new reason.
func replaceWaitingTag(line, reason string) string {
	lower := strings.ToLower(line)
	start := strings.Index(lower, "[waiting:")
	if start == -1 {
		return line
	}
	end := strings.Index(line[start:], "]")
	if end == -1 {
		return line
	}
	return line[:start] + "[waiting: " + reason + "]" + line[start+end+1:]
}

// removeWaitingTag removes a [waiting: ...] tag and any trailing space from the line.
func removeWaitingTag(line string) string {
	lower := strings.ToLower(line)
	start := strings.Index(lower, "[waiting:")
	if start == -1 {
		return line
	}
	end := strings.Index(line[start:], "]")
	if end == -1 {
		return line
	}
	removed := line[:start] + line[start+end+1:]
	return strings.ReplaceAll(removed, "  ", " ")
}

// replaceContextTag replaces an existing @context tag with a new one.
func replaceContextTag(line, name string) string {
	words := strings.Fields(line)
	for i, w := range words {
		if strings.HasPrefix(w, "@") && len(w) > 1 {
			words[i] = "@" + name
			break
		}
	}
	// preserve leading whitespace
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	return indent + strings.Join(words, " ")
}

// removeContextTag removes the first @context word from the line.
func removeContextTag(line string) string {
	words := strings.Fields(line)
	result := words[:0]
	removed := false
	for _, w := range words {
		if !removed && strings.HasPrefix(w, "@") && len(w) > 1 {
			removed = true
			continue
		}
		result = append(result, w)
	}
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	return indent + strings.Join(result, " ")
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
