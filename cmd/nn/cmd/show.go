package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newShowCmd(state *rootState) *cobra.Command {
	var linkedFrom string
	var jsonOut bool
	var depth int

	cmd := &cobra.Command{
		Use:   "show <id-or-title> [<id-or-title>...]",
		Short: "Print note content to stdout (accepts ID or title substring; --depth N for graph traversal)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := outWriter(cmd)

			if depth > 0 {
				if len(args) != 1 {
					return fmt.Errorf("show --depth: provide exactly one ID")
				}
				root, err := resolveNote(state, args[0])
				if err != nil {
					return fmt.Errorf("show --depth: %w", err)
				}
				all, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("show --depth: list: %w", err)
				}
				// Build ID→note index for fast lookup.
				byID := make(map[string]*note.Note, len(all))
				for _, n := range all {
					byID[n.ID] = n
				}
				// BFS up to depth hops following outgoing links.
				type entry struct {
					n     *note.Note
					level int
				}
				visited := map[string]bool{root.ID: true}
				queue := []entry{{root, 0}}
				var ordered []entry
				for len(queue) > 0 {
					cur := queue[0]
					queue = queue[1:]
					ordered = append(ordered, cur)
					if cur.level >= depth {
						continue
					}
					for _, lnk := range cur.n.Links {
						if visited[lnk.TargetID] {
							continue
						}
						visited[lnk.TargetID] = true
						if target, ok := byID[lnk.TargetID]; ok {
							queue = append(queue, entry{target, cur.level + 1})
						}
					}
				}

				if jsonOut {
					type depthNoteJSON struct {
						ID       string   `json:"id"`
						Title    string   `json:"title"`
						Type     string   `json:"type"`
						Status   string   `json:"status"`
						Tags     []string `json:"tags"`
						Created  string   `json:"created"`
						Modified string   `json:"modified"`
						Body     string   `json:"body"`
						Depth    int      `json:"depth"`
					}
					out := make([]depthNoteJSON, len(ordered))
					for i, e := range ordered {
						tags := e.n.Tags
						if tags == nil {
							tags = []string{}
						}
						out[i] = depthNoteJSON{
							ID:       e.n.ID,
							Title:    e.n.Title,
							Type:     string(e.n.Type),
							Status:   string(e.n.Status),
							Tags:     tags,
							Created:  e.n.Created.UTC().Format(time.RFC3339),
							Modified: e.n.Modified.UTC().Format(time.RFC3339),
							Body:     e.n.Body,
							Depth:    e.level,
						}
					}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(out)
				}

				for i, e := range ordered {
					if i > 0 {
						fmt.Fprintln(w, "---")
					}
					data, err := e.n.Marshal()
					if err != nil {
						return fmt.Errorf("show --depth: marshal: %w", err)
					}
					fmt.Fprint(w, string(data))
				}
				return nil
			}

			if linkedFrom != "" {
				src, err := resolveNote(state, linkedFrom)
				if err != nil {
					return fmt.Errorf("show --linked-from: %w", err)
				}
				all, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("show --linked-from: list: %w", err)
				}
				for i, lnk := range src.Links {
					n, err := state.backend.Read(lnk.TargetID)
					if err != nil {
						continue // skip broken links silently
					}
					if i > 0 {
						fmt.Fprintln(w, "---")
					}
					protos := findGoverningProtocols(n.ID, all)
					if len(protos) > 0 {
						fmt.Fprintf(w, "governing protocols:\n")
						for _, p := range protos {
							fmt.Fprintf(w, "  - [%s] %s\n", p.ID, p.Title)
						}
						fmt.Fprintln(w)
					}
					data, err := n.Marshal()
					if err != nil {
						return fmt.Errorf("show: marshal: %w", err)
					}
					fmt.Fprint(w, string(data))
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("show: provide at least one ID or use --linked-from")
			}

			all, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("show: list: %w", err)
			}

			for i, query := range args {
				if i > 0 {
					fmt.Fprintln(w, "---")
				}
				n, err := resolveNote(state, query)
				if err != nil {
					return fmt.Errorf("show: %w", err)
				}
				protos := findGoverningProtocols(n.ID, all)

				if jsonOut {
					type protoRef struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					}
					type showJSON struct {
						ID                  string     `json:"id"`
						Title               string     `json:"title"`
						Type                string     `json:"type"`
						Status              string     `json:"status"`
						Tags                []string   `json:"tags"`
						Created             string     `json:"created"`
						Modified            string     `json:"modified"`
						Body                string     `json:"body"`
						GoverningProtocols  []protoRef `json:"governing_protocols"`
					}
					refs := make([]protoRef, len(protos))
					for j, p := range protos {
						refs[j] = protoRef{ID: p.ID, Title: p.Title}
					}
					if refs == nil {
						refs = []protoRef{}
					}
					out := showJSON{
						ID:                 n.ID,
						Title:              n.Title,
						Type:               string(n.Type),
						Status:             string(n.Status),
						Tags:               n.Tags,
						Created:            n.Created.UTC().Format("2006-01-02T15:04:05Z"),
						Modified:           n.Modified.UTC().Format("2006-01-02T15:04:05Z"),
						Body:               n.Body,
						GoverningProtocols: refs,
					}
					if out.Tags == nil {
						out.Tags = []string{}
					}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					if err := enc.Encode(out); err != nil {
						return fmt.Errorf("show: json: %w", err)
					}
					continue
				}

				if len(protos) > 0 {
					fmt.Fprintf(w, "governing protocols:\n")
					for _, p := range protos {
						fmt.Fprintf(w, "  - [%s] %s\n", p.ID, p.Title)
					}
					fmt.Fprintln(w)
				}
				data, err := n.Marshal()
				if err != nil {
					return fmt.Errorf("show: marshal: %w", err)
				}
				fmt.Fprint(w, string(data))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&linkedFrom, "linked-from", "", "Show all notes linked from this ID")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output note as JSON with governing_protocols")
	cmd.Flags().IntVar(&depth, "depth", 0, "Traverse outgoing links to this depth and print all reachable notes")
	return cmd
}

// findGoverningProtocols returns all notes that link to targetID with type "governs".
func findGoverningProtocols(targetID string, all []*note.Note) []*note.Note {
	var result []*note.Note
	for _, n := range all {
		for _, lnk := range n.Links {
			if lnk.TargetID == targetID && lnk.Type == "governs" {
				result = append(result, n)
				break
			}
		}
	}
	return result
}

// resolveNote finds a note by exact ID or case-insensitive title substring.
func resolveNote(state *rootState, query string) (*note.Note, error) {
	n, err := state.backend.Read(query)
	if err == nil {
		return n, nil
	}
	all, listErr := state.backend.List()
	if listErr != nil {
		return nil, fmt.Errorf("%w", err)
	}
	type match struct{ id, title string }
	var matches []match
	for _, candidate := range all {
		if strings.Contains(strings.ToLower(candidate.Title), strings.ToLower(query)) {
			matches = append(matches, match{candidate.ID, candidate.Title})
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no note found matching %q", query)
	case 1:
		return state.backend.Read(matches[0].id)
	default:
		return nil, fmt.Errorf("ambiguous query %q — %d matches; use full ID", query, len(matches))
	}
}
