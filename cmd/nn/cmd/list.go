package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newListCmd(state *rootState) *cobra.Command {
	var (
		filterTag    string
		filterType   string
		filterStatus string
		linkedFrom   string
		linkedTo     string
		orphan       bool
		limit        int
		jsonOut      bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List and filter notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			// Build a set of all IDs that are link targets (for orphan detection).
			targetIDs := make(map[string]bool)
			for _, n := range notes {
				for _, lnk := range n.Links {
					targetIDs[lnk.TargetID] = true
				}
			}
			// Build a set of IDs with outbound links.
			hasOutbound := make(map[string]bool)
			for _, n := range notes {
				if len(n.Links) > 0 {
					hasOutbound[n.ID] = true
				}
			}

			var filtered []*note.Note
			for _, n := range notes {
				if filterTag != "" && !hasTag(n, filterTag) {
					continue
				}
				if filterType != "" && string(n.Type) != filterType {
					continue
				}
				if filterStatus != "" && string(n.Status) != filterStatus {
					continue
				}
				if linkedFrom != "" && !linksTo(n, linkedFrom) {
					continue
				}
				if linkedTo != "" {
					if !linkedToNote(notes, linkedTo, n.ID) {
						continue
					}
				}
				if orphan && (hasOutbound[n.ID] || targetIDs[n.ID]) {
					continue
				}
				filtered = append(filtered, n)
			}

			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			if jsonOut {
				return printNotesJSON(cmd, filtered)
			}
			for _, n := range filtered {
				fmt.Fprintf(outWriter(cmd), "%s  %s\n", n.ID, n.Title)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&filterTag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&filterType, "type", "", "Filter by type")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by status")
	cmd.Flags().StringVar(&linkedFrom, "linked-from", "", "Notes that link to this ID")
	cmd.Flags().StringVar(&linkedTo, "linked-to", "", "Notes this ID links to")
	cmd.Flags().BoolVar(&orphan, "orphan", false, "Notes with no links (inbound or outbound)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	return cmd
}

func hasTag(n *note.Note, tag string) bool {
	for _, t := range n.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func linksTo(n *note.Note, targetID string) bool {
	for _, lnk := range n.Links {
		if lnk.TargetID == targetID {
			return true
		}
	}
	return false
}

func linkedToNote(notes []*note.Note, fromID, targetID string) bool {
	for _, n := range notes {
		if n.ID == fromID {
			return linksTo(n, targetID)
		}
	}
	return false
}

type noteJSON struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Type   string   `json:"type"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
}

func printNotesJSON(cmd *cobra.Command, notes []*note.Note) error {
	out := make([]noteJSON, len(notes))
	for i, n := range notes {
		tags := n.Tags
		if tags == nil {
			tags = []string{}
		}
		out[i] = noteJSON{
			ID:     n.ID,
			Title:  n.Title,
			Type:   string(n.Type),
			Status: string(n.Status),
			Tags:   tags,
		}
	}
	enc := json.NewEncoder(outWriter(cmd))
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// tagsString is used by status command.
func tagsString(tags []string) string {
	return strings.Join(tags, ", ")
}

// suppress unused warning
var _ = tagsString
