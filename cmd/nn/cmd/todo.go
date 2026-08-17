package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/rules"
)

// todoListOptions controls which open todo items collectOpenTodos includes.
// The zero value matches `nn todo list` default behavior: historical daily
// notes, blocked notes, and waiting items are all excluded.
type todoListOptions struct {
	showAll       bool   // include notes blocked by an incomplete requires target
	showWaiting   bool   // show only items tagged [waiting: ...] instead of hiding them
	contextFilter string // if non-empty, keep only items tagged @<contextFilter>
}

// collectOpenTodos is the single source of truth for rendering open checkboxes
// grouped by note. Both `nn show --global` (via listTodosText) and `nn todo list`
// call it so their filtering can never drift apart.
func collectOpenTodos(notes []*note.Note, byID map[string]*note.Note, opts todoListOptions) string {
	loc := time.Now().Location()
	today := time.Now().In(loc).Truncate(24 * time.Hour)

	// Compute the transitively-blocked set once via the engine (gated on the
	// notebook actually having requires edges, so the common no-dependency case
	// skips evaluation entirely).
	blocked := blockedSet(notes)

	var sb strings.Builder
	first := true
	for _, n := range notes {
		// Always exclude daily notes from previous days — their todos are carried forward.
		if hasDailyTag(n) {
			createdDay := n.Created.In(loc).Truncate(24 * time.Hour)
			if createdDay.Before(today) {
				continue
			}
		}
		var open []string
		for _, line := range strings.Split(n.Body, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "- [ ]") {
				continue
			}
			waiting, _ := isWaiting(line)
			if opts.showWaiting {
				if !waiting {
					continue
				}
			} else {
				if waiting {
					continue
				}
			}
			if opts.contextFilter != "" {
				if todoContext(line) != strings.ToLower(opts.contextFilter) {
					continue
				}
			}
			open = append(open, line)
		}
		if len(open) == 0 {
			continue
		}
		if !opts.showAll && !opts.showWaiting && blocked[n.ID] {
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

// listTodosText renders the default todo view for nn show --global, matching
// nn todo list's default filter.
func listTodosText(notes []*note.Note, byID map[string]*note.Note) string {
	return collectOpenTodos(notes, byID, todoListOptions{})
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

func hasDailyTag(n *note.Note) bool {
	for _, t := range n.Tags {
		if t == "daily" {
			return true
		}
	}
	return false
}

func newTodoCmd(state *rootState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todo",
		Short: "Manage todo checkboxes in notes",
	}
	cmd.AddCommand(newTodoListCmd(state), newTodoDoneCmd(state), newTodoReopenCmd(state), newTodoSetCmd(state))
	return cmd
}

// blockedSet returns the set of note IDs that are blocked by an unfinished task
// dependency, derived from the engine's transitive `blocked` predicate (X is
// blocked if it requires a not-done target, or a target that is itself blocked).
// It is gated: if the notebook has no `requires` edges, no engine evaluation runs
// and the empty set is returned, keeping the common (no-dependency) case fast.
func blockedSet(notes []*note.Note) map[string]bool {
	hasRequires := false
	for _, n := range notes {
		for _, lnk := range n.Links {
			if lnk.Type == "requires" {
				hasRequires = true
				break
			}
		}
		if hasRequires {
			break
		}
	}
	blocked := map[string]bool{}
	if !hasRequires {
		return blocked
	}

	e := rules.NewEngine()
	for _, f := range rules.FactsFromNotes(notes) {
		e.AddFact(f)
	}
	builtin, err := rules.ParseProgram(rules.BuiltinRules())
	if err != nil {
		return blocked
	}
	e.AddRules(builtin)
	if err := e.Eval(); err != nil {
		return blocked
	}
	for _, f := range e.Query("blocked") {
		if len(f.Args) >= 1 {
			blocked[f.Args[0]] = true
		}
	}
	return blocked
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

			out := collectOpenTodos(notes, byID, todoListOptions{
				showAll:       showAll,
				showWaiting:   showWaiting,
				contextFilter: contextFilter,
			})
			fmt.Fprint(outWriter(cmd), out)
			return nil
		},
	}
	cmd.Flags().BoolVar(&showAll, "all", false, "Show all notes with open items, including blocked ones")
	cmd.Flags().BoolVar(&showWaiting, "waiting", false, "Show only items tagged [waiting: reason]")
	cmd.Flags().StringVar(&contextFilter, "context", "", "Show only items tagged with @context")
	return cmd
}

func newTodoDoneCmd(state *rootState) *cobra.Command {
	var resolution string
	cmd := &cobra.Command{
		Use:   "done <id> <pattern> [<pattern>...]",
		Short: "Mark matching open checkboxes as done (multiple patterns flip in one write)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flipCheckboxes(cmd, state, args[0], args[1:], "- [ ]", "- [x]", "open", resolution)
		},
	}
	cmd.Flags().StringVar(&resolution, "resolution", "", "Append commentary explaining why the item(s) were completed")
	return cmd
}

func newTodoReopenCmd(state *rootState) *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <id> <pattern> [<pattern>...]",
		Short: "Mark matching done checkboxes as open (multiple patterns flip in one write)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flipCheckbox(cmd, state, args[0], args[1:], "- [x]", "- [ ]", "done")
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
			sinceTs := n.Modified
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
			if err := state.backend.Update(n, &sinceTs); err != nil {
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

func flipCheckbox(cmd *cobra.Command, state *rootState, id string, patterns []string, from, to, fromLabel string) error {
	return flipCheckboxes(cmd, state, id, patterns, from, to, fromLabel, "")
}

// flipCheckboxes flips every checkbox matching one of patterns from `from` to
// `to` in a single read-modify-write. Matching is all-or-nothing: every pattern
// must match a distinct not-yet-flipped line, and if any pattern matches none
// the note is left unchanged (no partial write). Doing all flips in one Update
// avoids the concurrency conflict that parallel single-pattern calls hit. When
// resolution is non-empty it is appended to each flipped line as commentary.
func flipCheckboxes(cmd *cobra.Command, state *rootState, id string, patterns []string, from, to, fromLabel, resolution string) error {
	n, err := resolveNote(state, id)
	if err != nil {
		return fmt.Errorf("todo: %w", err)
	}
	sinceTs := n.Modified

	lines := strings.Split(n.Body, "\n")
	// Resolve every pattern to a distinct matching line before mutating, so a
	// single unmatched pattern aborts without a partial write.
	matchedIdx := make([]int, 0, len(patterns))
	usedLine := make(map[int]bool)
	for _, pattern := range patterns {
		lowerPattern := strings.ToLower(pattern)
		found := -1
		for i, line := range lines {
			if usedLine[i] {
				continue
			}
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, from) && strings.Contains(strings.ToLower(line), lowerPattern) {
				found = i
				break
			}
		}
		if found == -1 {
			return fmt.Errorf("no %s checkbox matching %q found in note %s", fromLabel, pattern, n.ID)
		}
		usedLine[found] = true
		matchedIdx = append(matchedIdx, found)
	}

	for _, i := range matchedIdx {
		lines[i] = strings.Replace(lines[i], from, to, 1)
		if resolution != "" {
			lines[i] = strings.TrimRight(lines[i], " ") + " — " + resolution
		}
	}
	n.Body = strings.Join(lines, "\n")
	n.Modified = time.Now().In(time.Local)

	if err := state.backend.Update(n, &sinceTs); err != nil {
		return fmt.Errorf("todo: %w", err)
	}
	fmt.Fprintf(outWriter(cmd), "updated %s\nmodified: %s\n", n.ID, n.Modified.Format(time.RFC3339))
	return nil
}
