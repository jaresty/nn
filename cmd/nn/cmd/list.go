package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		global       bool
		long         bool
		stale        bool
		limit        int
		jsonOut      bool
		rich         bool
		search       string
		sortBy       string
		since        string
		before       string
		similarTo    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List and filter notes; --similar <id> ranks by BM25 similarity",
		RunE: func(cmd *cobra.Command, args []string) error {
			if global && filterType != "" && filterType != "protocol" {
				return fmt.Errorf("list: --global only applies to protocol notes; --type %q is incompatible", filterType)
			}
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
			// Build inbound annotation map from all notes before filtering so
			// the pre-filter and scorer both see inbound text.
			allInbound := make(map[string][]string)
			for _, n := range notes {
				if len(n.Links) > 0 {
					hasOutbound[n.ID] = true
				}
				for _, lnk := range n.Links {
					allInbound[lnk.TargetID] = append(allInbound[lnk.TargetID], lnk.Annotation)
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
				if global {
					if n.Type != note.TypeProtocol {
						continue
					}
					hasGoverns := false
					for _, lnk := range n.Links {
						if lnk.Type == "governs" {
							hasGoverns = true
							break
						}
					}
					if hasGoverns {
						continue
					}
				}
				if stale && !isStaleNote(n, state.notebookDir) {
					continue
				}
				if search != "" && note.BM25Scores([]*note.Note{n}, search, allInbound)[n.ID] == 0 {
					continue
				}
				if long && len(n.Body) <= atomicityThreshold {
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

			if similarTo != "" {
				target, err := state.backend.Read(similarTo)
				if err != nil {
					return fmt.Errorf("list --similar: %w", err)
				}
				// Exclude the target note itself from results.
				var withoutTarget []*note.Note
				for _, n := range filtered {
					if n.ID != target.ID {
						withoutTarget = append(withoutTarget, n)
					}
				}
				filtered = withoutTarget
				scores := note.BM25Scores(filtered, target.Title+" "+target.Body, allInbound)
				sort.SliceStable(filtered, func(i, j int) bool {
					return scores[filtered[i].ID] > scores[filtered[j].ID]
				})
			}

			if search != "" {
				scores := note.BM25Scores(filtered, search, allInbound)
				sort.SliceStable(filtered, func(i, j int) bool {
					return scores[filtered[i].ID] > scores[filtered[j].ID]
				})
			}

			// Only apply sort-by when not using --similar (similarity ranking takes precedence).
			if similarTo == "" {
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
	cmd.Flags().BoolVar(&stale, "stale", false, "Notes accessed via nn show but not committed since (advisory; requires access.log)")
	cmd.Flags().BoolVar(&global, "global", false, "Protocol notes with no outgoing governs links (applies universally)")
	cmd.Flags().BoolVar(&long, "long", false, "Filter to notes exceeding the atomicity threshold")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().StringVar(&search, "search", "", "Full-text search across title and body")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by field: title, modified, created (default: created desc)")
	cmd.Flags().StringVar(&since, "since", "", "Notes modified after this date (ISO 8601: 2006-01-02 or 2006-01-02T15:04:05Z)")
	cmd.Flags().StringVar(&before, "before", "", "Notes modified before this date (ISO 8601)")
	cmd.Flags().BoolVar(&rich, "rich", false, "Include modified, link_count, body_preview in JSON output (requires --json)")
	cmd.Flags().StringVar(&similarTo, "similar", "", "Rank notes by BM25 similarity to this note ID (excludes the note itself)")
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

// isStaleNote returns true when a note appears in the access.log but has had
// no git commit touching its file since the last access timestamp.
// Returns false (not stale) on any error so failures are silent.
func isStaleNote(n *note.Note, repoDir string) bool {
	cfgDir := os.Getenv("NN_CONFIG_DIR")
	if cfgDir == "" {
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			home, _ := os.UserHomeDir()
			xdg = filepath.Join(home, ".config")
		}
		cfgDir = filepath.Join(xdg, "nn")
	}
	logPath := filepath.Join(cfgDir, "access.log")
	f, err := os.Open(logPath)
	if err != nil {
		return false
	}
	defer f.Close()

	// Find the most recent access timestamp for this note.
	var lastAccess time.Time
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 3 || parts[1] != "show" || parts[2] != n.ID {
			continue
		}
		t, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue
		}
		if t.After(lastAccess) {
			lastAccess = t
		}
	}
	if lastAccess.IsZero() {
		return false // not in access log
	}

	// Check git log for commits touching this note's file since lastAccess.
	since := lastAccess.UTC().Format(time.RFC3339)
	notePath := filepath.Join(repoDir, n.Filename())
	out, err := exec.Command("git", "-C", repoDir, "log",
		"--after="+since, "--format=%H", "--", notePath).Output()
	if err != nil {
		// Exit 128 = empty repo or no HEAD; no commits exist, so note is stale.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return true
		}
		return false
	}
	// If no commits since access, the note is stale.
	return strings.TrimSpace(string(out)) == ""
}
