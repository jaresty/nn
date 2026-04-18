package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newShowCmd(state *rootState) *cobra.Command {
	var linkedFrom string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show <id-or-title> [<id-or-title>...]",
		Short: "Print note content to stdout (accepts ID or title substring)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := outWriter(cmd)

			if linkedFrom != "" {
				src, err := resolveNote(state, linkedFrom)
				if err != nil {
					return fmt.Errorf("show --linked-from: %w", err)
				}
				for i, lnk := range src.Links {
					n, err := state.backend.Read(lnk.TargetID)
					if err != nil {
						continue // skip broken links silently
					}
					if i > 0 {
						fmt.Fprintln(w, "---")
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
