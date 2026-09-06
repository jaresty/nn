package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// sessionRow is one entry of the recent-sessions menu (the navigator front door).
type sessionRow struct {
	Session     string `json:"session"`
	Path        string `json:"path"`
	Modified    string `json:"modified"`
	Schema      string `json:"schema"`
	AgentCount  int    `json:"agent_count"`
	TotalCost   int    `json:"total_cost"`
	TreePreview string `json:"tree_preview"`
}

func newTranscriptLsCmd() *cobra.Command {
	var (
		limit   int
		before  string
		asJSON  bool
	)
	cmd := &cobra.Command{
		Use:   "ls [dir]",
		Short: "List recent sessions most-recent-first, each with a compact subagent-tree preview",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			var beforeTime time.Time
			if before != "" {
				t, err := time.Parse(time.RFC3339, before)
				if err != nil {
					return fmt.Errorf("--before must be RFC3339: %w", err)
				}
				beforeTime = t
			}
			rows, err := listSessions(dir, limit, beforeTime)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}
			out := cmd.OutOrStdout()
			for _, r := range rows {
				fmt.Fprintf(out, "%s  %-11s  %2d agents  cost=%-7d  %s  %s\n",
					r.Modified[:10], r.Schema, r.AgentCount, r.TotalCost, r.Session, r.TreePreview)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "list at most N sessions (0 = all)")
	cmd.Flags().StringVar(&before, "before", "", "only sessions modified before this RFC3339 timestamp (page further back)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit structured session rows as JSON")
	return cmd
}

// listSessions discovers top-level session .jsonl files, classifies each, and
// returns them most-recent-first (by mtime), applying --before and --limit.
func listSessions(dir string, limit int, before time.Time) ([]sessionRow, error) {
	type disc struct {
		path string
		mod  time.Time
	}
	var found []disc
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		// skip subagent child files (classified with their parent session)
		if filepath.Base(filepath.Dir(path)) == "subagents" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		found = append(found, disc{path: path, mod: info.ModTime()})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// most-recent-first
	sort.Slice(found, func(i, j int) bool { return found[i].mod.After(found[j].mod) })

	rows := make([]sessionRow, 0, len(found))
	for _, f := range found {
		if !before.IsZero() && !f.mod.Before(before) {
			continue
		}
		row := sessionRow{
			Session:  strings.TrimSuffix(filepath.Base(f.path), ".jsonl"),
			Path:     f.path,
			Modified: f.mod.UTC().Format(time.RFC3339),
			Schema:   classifyTranscript(f.path),
		}
		// agent count, cost, and mini-tree from the spine (best-effort; a
		// malformed session still lists with a note rather than aborting ls).
		if agents, terr := buildTree(f.path); terr == nil {
			row.AgentCount = len(agents)
			for _, a := range agents {
				row.TotalCost += a.Cost
			}
			row.TreePreview = miniTree(agents)
		} else {
			row.TreePreview = "(tree unavailable)"
		}
		rows = append(rows, row)
		if limit > 0 && len(rows) >= limit {
			break
		}
	}
	return rows, nil
}

// miniTree renders a compact one-line preview of the spawn structure, e.g.
// "ROOT─┬─aaa─bbb" for a small chain, capped so the row stays scannable.
func miniTree(agents []agent) string {
	children := map[string][]agent{}
	var roots []agent
	for _, a := range agents {
		if a.ParentID == "" {
			roots = append(roots, a)
		} else {
			children[a.ParentID] = append(children[a.ParentID], a)
		}
	}
	for k := range children {
		sort.Slice(children[k], func(i, j int) bool { return children[k][i].ID < children[k][j].ID })
	}
	const maxNodes = 6
	shown := 0
	var render func(a agent) string
	render = func(a agent) string {
		if shown >= maxNodes {
			return ""
		}
		shown++
		label := shortID(a.ID)
		kids := children[a.ID]
		if len(kids) == 0 {
			return label
		}
		var parts []string
		for _, k := range kids {
			if shown >= maxNodes {
				parts = append(parts, "…")
				break
			}
			parts = append(parts, render(k))
		}
		if len(kids) > 1 {
			return label + "─┬─" + strings.Join(parts, "─")
		}
		return label + "─" + strings.Join(parts, "─")
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })
	var out []string
	for _, r := range roots {
		out = append(out, render(r))
	}
	return strings.Join(out, "  ")
}

func shortID(id string) string {
	if id == "ROOT" {
		return "ROOT"
	}
	if len(id) > 6 {
		return id[:6]
	}
	return id
}
