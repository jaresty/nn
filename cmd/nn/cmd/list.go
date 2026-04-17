package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

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
		rich         bool
		search       string
		sortBy       string
		since        string
		before       string
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
				if search != "" && !containsFold(n.Title, search) && !containsFold(n.Body, search) {
					continue
				}
				if since != "" {
					t, err := parseDateTime(since)
					if err != nil {
						return fmt.Errorf("--since: %w", err)
					}
					if !n.Modified.After(t) {
						continue
					}
				}
				if before != "" {
					t, err := parseDateTime(before)
					if err != nil {
						return fmt.Errorf("--before: %w", err)
					}
					if !n.Modified.Before(t) {
						continue
					}
				}
				filtered = append(filtered, n)
			}

			if search != "" {
				scores := make(map[string]int, len(filtered))
				for _, n := range filtered {
					if containsFold(n.Title, search) {
						scores[n.ID] += 10
					}
					if containsFold(n.Body, search) {
						scores[n.ID] += 1
					}
				}
				sort.SliceStable(filtered, func(i, j int) bool {
					return scores[filtered[i].ID] > scores[filtered[j].ID]
				})
			}

			switch sortBy {
			case "modified":
				sort.Slice(filtered, func(i, j int) bool {
					return filtered[i].Modified.After(filtered[j].Modified)
				})
			case "title":
				sort.Slice(filtered, func(i, j int) bool {
					return filtered[i].Title < filtered[j].Title
				})
			case "created", "":
				sort.Slice(filtered, func(i, j int) bool {
					return filtered[i].Created.After(filtered[j].Created)
				})
			}

			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			if jsonOut {
				if rich {
					return printNotesRichJSON(cmd, filtered)
				}
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
	cmd.Flags().StringVar(&search, "search", "", "Full-text search across title and body")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by field: title, modified, created (default: created desc)")
	cmd.Flags().StringVar(&since, "since", "", "Notes modified after this date (ISO 8601: 2006-01-02 or 2006-01-02T15:04:05Z)")
	cmd.Flags().StringVar(&before, "before", "", "Notes modified before this date (ISO 8601)")
	cmd.Flags().BoolVar(&rich, "rich", false, "Include modified, link_count, body_preview in JSON output (requires --json)")
	return cmd
}

// parseDateTime parses an ISO 8601 date or datetime string.
func parseDateTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q: use 2006-01-02 or 2006-01-02T15:04:05Z", s)
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
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

const bodyPreviewLen = 200

type noteRichJSON struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Created     string   `json:"created"`
	Modified    string   `json:"modified"`
	LinkCount   int      `json:"link_count"`
	BodyPreview string   `json:"body_preview"`
}

func printNotesRichJSON(cmd *cobra.Command, notes []*note.Note) error {
	out := make([]noteRichJSON, len(notes))
	for i, n := range notes {
		tags := n.Tags
		if tags == nil {
			tags = []string{}
		}
		preview := n.Body
		if len(preview) > bodyPreviewLen {
			preview = preview[:bodyPreviewLen]
		}
		out[i] = noteRichJSON{
			ID:          n.ID,
			Title:       n.Title,
			Type:        string(n.Type),
			Status:      string(n.Status),
			Tags:        tags,
			Created:     n.Created.UTC().Format(time.RFC3339),
			Modified:    n.Modified.UTC().Format(time.RFC3339),
			LinkCount:   len(n.Links),
			BodyPreview: preview,
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
