package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newListCmd(state *rootState) *cobra.Command {
	var (
		filterTag            string
		filterType           string
		filterStatus         string
		filterRepresentation string
		linkedFrom           string
		linkedTo             string
		orphan               bool
		global               bool
		long                 bool
		unactioned           bool
		olderThan            int
		noInbound            bool
		expired              bool
		hasExpires           bool
		hasURL               bool
		urlContains          string
		hasOpenItems         bool
		unblocked            bool
		limit                int
		jsonOut              bool
		rich                 bool
		fullOutput           bool
		showFirst            bool
		search               string
		sortBy               string
		since                string
		before               string
		similarTo            string
		envelope             bool
		boostRecent          bool
		fields               string
		countOnly            bool
		page                 int
		snapshot             string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List and filter notes; --similar <id> ranks by BM25 similarity",
		RunE: func(cmd *cobra.Command, args []string) error {
			pagination := cmd.Flags().Changed("page") || snapshot != ""
			if pagination {
				if !cmd.Flags().Changed("page") {
					return fmt.Errorf("list: --snapshot requires --page")
				}
				if page < 1 {
					return fmt.Errorf("list: --page must be at least 1")
				}
				if !jsonOut || !envelope {
					return fmt.Errorf("list: --page requires --json --envelope")
				}
				if limit > 0 || countOnly || showFirst {
					return fmt.Errorf("list: --page cannot be combined with --limit, --count, or --show-first")
				}
				if page > 1 && snapshot == "" {
					return fmt.Errorf("list: --snapshot is required for page %d", page)
				}
			}
			if countOnly && (jsonOut || envelope || rich || fullOutput || showFirst || fields != "") {
				return fmt.Errorf("list: --count cannot be combined with --json, --envelope, --rich, --full, --show-first, or --fields")
			}
			if envelope && !jsonOut {
				return fmt.Errorf("list: --envelope requires --json")
			}
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

			notesByID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				notesByID[n.ID] = n
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
				if filterRepresentation != "" && n.Representation != filterRepresentation {
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
				if noInbound && len(allInbound[n.ID]) > 0 {
					continue
				}
				if unactioned && !isUnactionedNote(n, state.notebookDir) {
					continue
				}
				if olderThan > 0 {
					cutoff := time.Now().UTC().Add(-time.Duration(olderThan) * 24 * time.Hour)
					if !n.Modified.Before(cutoff) {
						continue
					}
				}
				if expired && (n.Expires == nil || !n.Expires.Before(time.Now().UTC())) {
					continue
				}
				if hasExpires && n.Expires == nil {
					continue
				}
				if hasOpenItems && note.IsDone(n.Body) {
					continue
				}
				if unblocked {
					if !isUnblocked(n, notesByID) {
						continue
					}
				}
				if long && len(n.Body) <= atomicityThreshold {
					continue
				}
				if hasURL && !reURL.MatchString(n.Body) {
					continue
				}
				if urlContains != "" && !containsURL(n.Body, urlContains) {
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
				similarIDF := note.BM25IDF(notes, note.Tokenize(target.Title+" "+target.Body))
				scores := note.BM25ScoresWithIDF(filtered, similarIDF, target.Title+" "+target.Body, allInbound)
				sort.SliceStable(filtered, func(i, j int) bool {
					return scores[filtered[i].ID] > scores[filtered[j].ID]
				})
			}

			var searchScores map[string]float64
			var normalizedSearchScores map[string]float64
			if search != "" {
				searchScores = RankedByQuery(notes, filtered, search, state.notebookDir)
				positiveResults := make([]*note.Note, 0, len(filtered))
				for _, n := range filtered {
					if searchScores[n.ID] > 0 {
						positiveResults = append(positiveResults, n)
					}
				}
				filtered = positiveResults
				if boostRecent && searchScores != nil {
					now := time.Now()
					for _, n := range filtered {
						ageDays := now.Sub(n.Modified).Hours() / 24
						// half-life ~30 days; multiplier in (1.0, 1.5]
						searchScores[n.ID] *= 1.0 + 0.5*math.Exp(-ageDays/30.0)
					}
				}
				maxScore := 0.0
				for _, s := range searchScores {
					if s > maxScore {
						maxScore = s
					}
				}
				normalizedSearchScores = make(map[string]float64, len(filtered))
				for _, n := range filtered {
					if maxScore > 0 {
						normalizedSearchScores[n.ID] = searchScores[n.ID] / maxScore
					}
				}
				sort.SliceStable(filtered, func(i, j int) bool {
					return normalizedSearchScores[filtered[i].ID] > normalizedSearchScores[filtered[j].ID]
				})
			}

			// Only apply sort-by when not using --similar or --search (relevance ranking takes precedence).
			if similarTo == "" && (search == "" || sortBy != "") {
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

			if search != "" && limit == 0 && !pagination {
				limit = 5
			}
			totalMatching := len(filtered)
			if countOnly {
				fmt.Fprintln(outWriter(cmd), totalMatching)
				return nil
			}
			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			if fields != "" && !jsonOut {
				return fmt.Errorf("--fields requires --json")
			}
			if jsonOut {
				var requestedFields []string
				if fields != "" {
					requestedFields, err = parseFields(fields)
					if err != nil {
						return err
					}
				}
				if pagination {
					key := fmt.Sprintf("tag=%q|type=%q|status=%q|representation=%q|linked-from=%q|linked-to=%q|orphan=%t|global=%t|long=%t|unactioned=%t|older-than=%d|no-inbound=%t|expired=%t|has-expires=%t|has-url=%t|url-contains=%q|has-open-items=%t|unblocked=%t|search=%q|sort=%q|since=%q|before=%q|similar=%q|boost-recent=%t|rich=%t|full=%t|fields=%q", filterTag, filterType, filterStatus, filterRepresentation, linkedFrom, linkedTo, orphan, global, long, unactioned, olderThan, noInbound, expired, hasExpires, hasURL, urlContains, hasOpenItems, unblocked, search, sortBy, since, before, similarTo, boostRecent, rich, fullOutput, fields)
					return printPaginatedListJSON(cmd, filtered, notes, search, normalizedSearchScores, allInbound, requestedFields, fullOutput, rich, totalMatching, page, snapshot, key)
				}
				if rich {
					if envelope {
						return printListEnvelopeJSON(cmd, filtered, requestedFields, totalMatching, true)
					}
					return printNotesRichJSON(cmd, filtered, requestedFields)
				}
				if search != "" {
					if envelope {
						return printSearchEnvelopeJSON(cmd, filtered, search, normalizedSearchScores, allInbound, totalMatching)
					}
					if err := printSearchJSON(cmd, filtered, notes, search, normalizedSearchScores, allInbound, requestedFields, fullOutput); err != nil {
						return err
					}
				} else {
					if envelope {
						return printListEnvelopeJSON(cmd, filtered, requestedFields, totalMatching, false)
					}
					if err := printNotesJSON(cmd, filtered, requestedFields); err != nil {
						return err
					}
				}
				if showFirst && len(filtered) > 0 {
					fmt.Fprintf(outWriter(cmd), "\n---\n\n")
					fmt.Fprint(outWriter(cmd), formatNoteBody(filtered[0]))
				}
				return nil
			}
			w := outWriter(cmd)
			sessionReads := loadSessionReads(resolveCfgDir())
			hasUnread := false
			for _, n := range filtered {
				readMarker := ""
				if sessionReads[n.ID] {
					readMarker = " [read]"
				} else if search != "" {
					hasUnread = true
				}
				fmt.Fprintf(w, "%s  %s%s\n", n.ID, n.Title, readMarker)
				if search != "" {
					if ex := extractExcerpt(n.Body, search); ex != "" {
						fmt.Fprintf(w, "  %s\n", ex)
					}
					for _, nb := range buildNeighbors(n, notes) {
						if nb.Direction == "outgoing" {
							fmt.Fprintf(w, "  → [[%s|%s]] [%s] ↑%d ↓%d — %s\n", nb.ID, nb.Title, nb.Type, nb.OutDegree, nb.InDegree, nb.Annotation)
						} else {
							fmt.Fprintf(w, "  ← [[%s|%s]] [%s] ↑%d ↓%d — %s\n", nb.ID, nb.Title, nb.Type, nb.OutDegree, nb.InDegree, nb.Annotation)
						}
					}
				}
			}
			if search != "" && totalMatching > len(filtered) {
				fmt.Fprintf(w, "(%d shown, %d more — use --limit 0 to show all)\n", len(filtered), totalMatching-len(filtered))
			}
			if search != "" {
				printResolveInstruction(w, hasUnread)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&filterRepresentation, "representation", "", "Filter by representation type (ontology|taxonomy|axiom)")
	cmd.Flags().StringVar(&filterTag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&filterType, "type", "", "Filter by type")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by status")
	cmd.Flags().StringVar(&linkedFrom, "linked-from", "", "Notes that link to this ID")
	cmd.Flags().StringVar(&linkedTo, "linked-to", "", "Notes this ID links to")
	cmd.Flags().BoolVar(&orphan, "orphan", false, "Notes with no links (inbound or outbound)")
	cmd.Flags().BoolVar(&noInbound, "no-inbound", false, "Notes with no inbound links (may have outbound); stricter than --orphan")
	cmd.Flags().BoolVar(&unactioned, "unactioned", false, "Notes accessed via nn show but not committed since (advisory; requires access.log)")
	cmd.Flags().IntVar(&olderThan, "older-than", 0, "Notes not modified in the last N days (age-based; 3=aging, 14=stale)")
	cmd.Flags().BoolVar(&expired, "expired", false, "Notes with an expires date in the past")
	cmd.Flags().BoolVar(&hasExpires, "has-expires", false, "Notes with an expires date set (any date)")
	cmd.Flags().BoolVar(&global, "global", false, "Protocol notes with no outgoing governs links (applies universally)")
	cmd.Flags().BoolVar(&long, "long", false, "Filter to notes exceeding the atomicity threshold")
	cmd.Flags().BoolVar(&hasOpenItems, "has-open-items", false, "Notes with at least one unchecked checkbox (- [ ])")
	cmd.Flags().BoolVar(&unblocked, "unblocked", false, "Notes with requires links whose targets are all done (no unchecked checkboxes)")
	cmd.Flags().BoolVar(&hasURL, "has-url", false, "Filter to notes containing an http/https URL")
	cmd.Flags().StringVar(&urlContains, "url-contains", "", "Filter to notes containing a URL that includes this string")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Machine-readable JSON output")
	cmd.Flags().StringVar(&search, "search", "", "Full-text search across title and body")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort by field: title, modified, created (default: created desc)")
	cmd.Flags().StringVar(&since, "since", "", "Notes modified after this date (ISO 8601: 2006-01-02 or 2006-01-02T15:04:05Z)")
	cmd.Flags().StringVar(&before, "before", "", "Notes modified before this date (ISO 8601)")
	cmd.Flags().BoolVar(&boostRecent, "boost-recent", false, "Boost recently-modified notes in search results (requires --search)")
	cmd.Flags().BoolVar(&rich, "rich", false, "Include modified, link_count, body_preview in JSON output (requires --json)")
	cmd.Flags().BoolVar(&fullOutput, "full", false, "Disable truncation of excerpt and annotation in JSON output")
	cmd.Flags().BoolVar(&showFirst, "show-first", false, "Append full body of top search result after JSON output (requires --json)")
	cmd.Flags().StringVar(&similarTo, "similar", "", "Rank notes by BM25 similarity to this note ID (excludes the note itself)")
	cmd.Flags().BoolVar(&envelope, "envelope", false, "Wrap JSON in envelope with result_count, total_matching, and results (search also includes query)")
	cmd.Flags().BoolVar(&countOnly, "count", false, "Print only the number of notes matching all active filters")
	cmd.Flags().IntVar(&page, "page", 0, "One-based stable result page (requires --json --envelope)")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Snapshot SHA-256 returned by page 1 (required for later pages)")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to project in --json output (id,title,type,status,tags,applies_when,excerpt,score,modified,match_reason,is_protocol,link_count,backlink_count,created,body_preview,representation)")
	return cmd
}

// formatNoteBody returns the full marshaled note content (frontmatter + body).
func formatNoteBody(n *note.Note) string {
	data, err := n.Marshal()
	if err != nil {
		return fmt.Sprintf("id: %s\ntitle: %s\n\n(marshal error: %v)\n", n.ID, n.Title, err)
	}
	return string(data)
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

var reURL = regexp.MustCompile(`https?://\S+`)

// containsURL returns true if body contains an http/https URL that includes substr.
func containsURL(body, substr string) bool {
	for _, url := range reURL.FindAllString(body, -1) {
		if strings.Contains(url, substr) {
			return true
		}
	}
	return false
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
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	Tags           []string `json:"tags"`
	Created        string   `json:"created"`
	Modified       string   `json:"modified"`
	AppliesWhen    string   `json:"applies_when,omitempty"`
	Representation string   `json:"representation,omitempty"`
}

type neighborJSON struct {
	Direction  string `json:"direction"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Annotation string `json:"annotation"`
	OutDegree  int    `json:"out_degree"`
	InDegree   int    `json:"in_degree"`
}

type noteSearchJSON struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	Tags           []string       `json:"tags"`
	Excerpt        string         `json:"excerpt"`
	Score          float64        `json:"score"`
	Created        string         `json:"created"`
	Modified       string         `json:"modified"`
	MatchReason    string         `json:"match_reason"`
	IsProtocol     bool           `json:"is_protocol"`
	LinkCount      int            `json:"link_count"`
	BacklinkCount  int            `json:"backlink_count"`
	Neighbors      []neighborJSON `json:"neighbors"`
	AppliesWhen    string         `json:"applies_when,omitempty"`
	Representation string         `json:"representation,omitempty"`
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func buildNeighbors(n *note.Note, allNotes []*note.Note) []neighborJSON {
	byID := make(map[string]*note.Note, len(allNotes))
	inDegree := make(map[string]int, len(allNotes))
	for _, a := range allNotes {
		byID[a.ID] = a
		for _, lnk := range a.Links {
			inDegree[lnk.TargetID]++
		}
	}
	degreeOf := func(id string) (out, in int) {
		if t, ok := byID[id]; ok {
			out = len(t.Links)
		}
		return out, inDegree[id]
	}
	var neighbors []neighborJSON
	for _, lnk := range n.Links {
		title := lnk.TargetID
		if t, ok := byID[lnk.TargetID]; ok {
			title = t.Title
		}
		out, in := degreeOf(lnk.TargetID)
		neighbors = append(neighbors, neighborJSON{
			Direction:  "outgoing",
			ID:         lnk.TargetID,
			Title:      title,
			Type:       lnk.Type,
			Annotation: lnk.Annotation,
			OutDegree:  out,
			InDegree:   in,
		})
	}
	for _, a := range allNotes {
		for _, lnk := range a.Links {
			if lnk.TargetID == n.ID {
				out, in := degreeOf(a.ID)
				neighbors = append(neighbors, neighborJSON{
					Direction:  "incoming",
					ID:         a.ID,
					Title:      a.Title,
					Type:       lnk.Type,
					Annotation: lnk.Annotation,
					OutDegree:  out,
					InDegree:   in,
				})
				break
			}
		}
	}
	if neighbors == nil {
		neighbors = []neighborJSON{}
	}
	return neighbors
}

func printSearchJSON(cmd *cobra.Command, notes []*note.Note, allNotes []*note.Note, query string, scores map[string]float64, inbound map[string][]string, requestedFields []string, fullOutput bool) error {
	out := make([]noteSearchJSON, len(notes))
	for i, n := range notes {
		tags := n.Tags
		if tags == nil {
			tags = []string{}
		}
		normalizedScore := scores[n.ID]
		isProtocol := n.Type == note.TypeProtocol
		if !isProtocol {
			for _, lnk := range n.Links {
				if lnk.Type == "governs" {
					isProtocol = true
					break
				}
			}
		}
		ex := extractExcerpt(n.Body, query)
		if fullOutput {
			ex = n.Body
		}
		neighbors := buildNeighbors(n, allNotes)
		if !fullOutput {
			for j := range neighbors {
				neighbors[j].Annotation = truncateStr(neighbors[j].Annotation, 80)
			}
		}
		out[i] = noteSearchJSON{
			ID:             n.ID,
			Title:          n.Title,
			Type:           string(n.Type),
			Status:         string(n.Status),
			Tags:           tags,
			Excerpt:        ex,
			Score:          normalizedScore,
			Created:        n.Created.UTC().Format(time.RFC3339Nano),
			Modified:       n.Modified.UTC().Format(time.RFC3339Nano),
			MatchReason:    computeMatchReason(n, query),
			IsProtocol:     isProtocol,
			LinkCount:      len(n.Links),
			BacklinkCount:  len(inbound[n.ID]),
			Neighbors:      neighbors,
			AppliesWhen:    n.AppliesWhen,
			Representation: n.Representation,
		}
	}
	enc := json.NewEncoder(outWriter(cmd))
	if len(requestedFields) == 0 {
		return enc.Encode(out)
	}
	projected := make([]map[string]any, len(out))
	for i, v := range out {
		projected[i] = projectFields(v, requestedFields)
	}
	return enc.Encode(projected)
}

const listPageMaxBytes = 48_000

type paginatedListEnvelope struct {
	Snapshot      string            `json:"snapshot"`
	Page          int               `json:"page"`
	Pages         int               `json:"pages"`
	NextPage      int               `json:"next_page"`
	ResultCount   int               `json:"result_count"`
	TotalMatching int               `json:"total_matching"`
	Query         string            `json:"query,omitempty"`
	Results       []json.RawMessage `json:"results"`
}

func printPaginatedListJSON(cmd *cobra.Command, selected, allNotes []*note.Note, query string, scores map[string]float64, inbound map[string][]string, requestedFields []string, fullOutput, rich bool, totalMatching, page int, suppliedSnapshot, requestKey string) error {
	var rendered bytes.Buffer
	shadow := *cmd
	shadow.SetOut(&rendered)
	var err error
	switch {
	case rich:
		err = printNotesRichJSON(&shadow, selected, requestedFields)
	case query != "":
		err = printSearchJSON(&shadow, selected, allNotes, query, scores, inbound, requestedFields, fullOutput)
	default:
		err = printNotesJSON(&shadow, selected, requestedFields)
	}
	if err != nil {
		return err
	}
	var records []json.RawMessage
	if err := json.Unmarshal(rendered.Bytes(), &records); err != nil {
		return fmt.Errorf("list: paginate encoded results: %w", err)
	}

	h := sha256.New()
	writeSnapshotPart(h, []byte("nn list page snapshot v1\x00"))
	writeSnapshotPart(h, []byte(requestKey))
	writeSnapshotPart(h, rendered.Bytes())
	expectedSnapshot := hex.EncodeToString(h.Sum(nil))
	if suppliedSnapshot != "" && suppliedSnapshot != expectedSnapshot {
		return fmt.Errorf("list: stale or mismatched --snapshot; notebook, query, or result projection changed")
	}

	pages := [][]json.RawMessage{{}}
	for _, record := range records {
		candidate := append(append([]json.RawMessage(nil), pages[len(pages)-1]...), record)
		if listPageFits(expectedSnapshot, query, totalMatching, candidate) {
			pages[len(pages)-1] = candidate
			continue
		}
		if !listPageFits(expectedSnapshot, query, totalMatching, []json.RawMessage{record}) {
			return fmt.Errorf("list: one result cannot fit in a %d-byte page; narrow fields or disable --full", listPageMaxBytes)
		}
		pages = append(pages, []json.RawMessage{record})
	}
	if page > len(pages) {
		return fmt.Errorf("list: --page %d out of range (pages: %d)", page, len(pages))
	}
	response := paginatedListEnvelope{Snapshot: expectedSnapshot, Page: page, Pages: len(pages), ResultCount: len(pages[page-1]), TotalMatching: totalMatching, Query: query, Results: pages[page-1]}
	if page < len(pages) {
		response.NextPage = page + 1
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("list: encode page: %w", err)
	}
	if len(encoded)+1 > listPageMaxBytes {
		return fmt.Errorf("list: encoded page is %d bytes (limit %d)", len(encoded)+1, listPageMaxBytes)
	}
	_, err = outWriter(cmd).Write(append(encoded, '\n'))
	return err
}

func listPageFits(snapshot, query string, totalMatching int, records []json.RawMessage) bool {
	candidate := paginatedListEnvelope{Snapshot: snapshot, Page: math.MaxInt64, Pages: math.MaxInt64, NextPage: math.MaxInt64, ResultCount: len(records), TotalMatching: totalMatching, Query: query, Results: records}
	encoded, err := json.Marshal(candidate)
	return err == nil && len(encoded)+1 <= listPageMaxBytes
}

type listEnvelope struct {
	ResultCount   int `json:"result_count"`
	TotalMatching int `json:"total_matching"`
	Results       any `json:"results"`
}

func printListEnvelopeJSON(cmd *cobra.Command, notes []*note.Note, requestedFields []string, totalMatching int, rich bool) error {
	var results any
	if rich {
		values := make([]noteRichJSON, len(notes))
		for i, n := range notes {
			tags := n.Tags
			if tags == nil {
				tags = []string{}
			}
			preview := n.Body
			if len(preview) > bodyPreviewLen {
				preview = preview[:bodyPreviewLen]
			}
			values[i] = noteRichJSON{ID: n.ID, Title: n.Title, Type: string(n.Type), Status: string(n.Status), Tags: tags, Created: n.Created.UTC().Format(time.RFC3339Nano), Modified: n.Modified.UTC().Format(time.RFC3339Nano), LinkCount: len(n.Links), BodyPreview: preview}
		}
		if len(requestedFields) == 0 {
			results = values
		} else {
			projected := make([]map[string]any, len(values))
			for i, value := range values {
				projected[i] = projectFields(value, requestedFields)
			}
			results = projected
		}
	} else {
		values := make([]noteJSON, len(notes))
		for i, n := range notes {
			tags := n.Tags
			if tags == nil {
				tags = []string{}
			}
			values[i] = noteJSON{ID: n.ID, Title: n.Title, Type: string(n.Type), Status: string(n.Status), Tags: tags, Created: n.Created.UTC().Format(time.RFC3339Nano), Modified: n.Modified.UTC().Format(time.RFC3339Nano), AppliesWhen: n.AppliesWhen, Representation: n.Representation}
		}
		if len(requestedFields) == 0 {
			results = values
		} else {
			projected := make([]map[string]any, len(values))
			for i, value := range values {
				projected[i] = projectFields(value, requestedFields)
			}
			results = projected
		}
	}
	return json.NewEncoder(outWriter(cmd)).Encode(listEnvelope{ResultCount: len(notes), TotalMatching: totalMatching, Results: results})
}

type searchEnvelope struct {
	Query         string           `json:"query"`
	ResultCount   int              `json:"result_count"`
	TotalMatching int              `json:"total_matching"`
	Results       []noteSearchJSON `json:"results"`
}

func printSearchEnvelopeJSON(cmd *cobra.Command, notes []*note.Note, query string, scores map[string]float64, inbound map[string][]string, totalMatching int) error {
	results := make([]noteSearchJSON, len(notes))
	for i, n := range notes {
		tags := n.Tags
		if tags == nil {
			tags = []string{}
		}
		normalizedScore := scores[n.ID]
		isProtocol := n.Type == note.TypeProtocol
		if !isProtocol {
			for _, lnk := range n.Links {
				if lnk.Type == "governs" {
					isProtocol = true
					break
				}
			}
		}
		results[i] = noteSearchJSON{
			ID:             n.ID,
			Title:          n.Title,
			Type:           string(n.Type),
			Status:         string(n.Status),
			Tags:           tags,
			AppliesWhen:    n.AppliesWhen,
			Excerpt:        extractExcerpt(n.Body, query),
			Score:          normalizedScore,
			Created:        n.Created.UTC().Format(time.RFC3339Nano),
			Modified:       n.Modified.UTC().Format(time.RFC3339Nano),
			MatchReason:    computeMatchReason(n, query),
			IsProtocol:     isProtocol,
			LinkCount:      len(n.Links),
			BacklinkCount:  len(inbound[n.ID]),
			Representation: n.Representation,
		}
	}
	env := searchEnvelope{
		Query:         query,
		ResultCount:   len(notes),
		TotalMatching: totalMatching,
		Results:       results,
	}
	enc := json.NewEncoder(outWriter(cmd))

	return enc.Encode(env)
}

func computeMatchReason(n *note.Note, query string) string {
	terms := note.Tokenize(query)
	titleTokens := note.Tokenize(n.Title)
	bodyTokens := note.Tokenize(n.Body)

	titleSet := make(map[string]bool, len(titleTokens))
	for _, t := range titleTokens {
		titleSet[t] = true
	}
	bodySet := make(map[string]bool, len(bodyTokens))
	for _, t := range bodyTokens {
		bodySet[t] = true
	}
	tagSet := make(map[string]bool)
	for _, tag := range n.Tags {
		for _, t := range note.Tokenize(tag) {
			tagSet[t] = true
		}
	}
	outboundSet := make(map[string]bool)
	for _, lnk := range n.Links {
		for _, t := range note.Tokenize(lnk.Annotation) {
			outboundSet[t] = true
		}
	}

	var fields []string
	seenTitle, seenBody, seenTag, seenOutbound := false, false, false, false
	for _, term := range terms {
		if !seenTitle && titleSet[term] {
			fields = append(fields, "title")
			seenTitle = true
		}
		if !seenBody && bodySet[term] {
			fields = append(fields, "body")
			seenBody = true
		}
		if !seenTag && tagSet[term] {
			fields = append(fields, "tag")
			seenTag = true
		}
		if !seenOutbound && outboundSet[term] {
			fields = append(fields, "outbound")
			seenOutbound = true
		}
	}
	return strings.Join(fields, ", ")
}

var validFields = map[string]bool{
	"id": true, "title": true, "type": true, "status": true, "tags": true,
	"applies_when": true, "excerpt": true, "score": true, "modified": true,
	"match_reason": true, "is_protocol": true, "link_count": true,
	"backlink_count": true, "created": true, "body_preview": true,
	"representation": true,
}

func parseFields(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !validFields[p] {
			return nil, fmt.Errorf("unknown field %q; valid fields: id,title,type,status,tags,applies_when,excerpt,score,modified,match_reason,is_protocol,link_count,backlink_count,created,body_preview,representation", p)
		}
		out = append(out, p)
	}
	return out, nil
}

func projectFields(v any, requested []string) map[string]any {
	if len(requested) == 0 {
		return nil
	}
	data, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	out := make(map[string]any, len(requested))
	for _, f := range requested {
		if val, ok := m[f]; ok {
			out[f] = val
		}
	}
	return out
}

func printNotesJSON(cmd *cobra.Command, notes []*note.Note, requestedFields []string) error {
	out := make([]noteJSON, len(notes))
	for i, n := range notes {
		tags := n.Tags
		if tags == nil {
			tags = []string{}
		}
		out[i] = noteJSON{
			ID:             n.ID,
			Title:          n.Title,
			Type:           string(n.Type),
			Status:         string(n.Status),
			Tags:           tags,
			Created:        n.Created.UTC().Format(time.RFC3339Nano),
			Modified:       n.Modified.UTC().Format(time.RFC3339Nano),
			AppliesWhen:    n.AppliesWhen,
			Representation: n.Representation,
		}
	}
	enc := json.NewEncoder(outWriter(cmd))
	if len(requestedFields) == 0 {
		return enc.Encode(out)
	}
	projected := make([]map[string]any, len(out))
	for i, v := range out {
		projected[i] = projectFields(v, requestedFields)
	}
	return enc.Encode(projected)
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

func printNotesRichJSON(cmd *cobra.Command, notes []*note.Note, requestedFields []string) error {
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
			Created:     n.Created.UTC().Format(time.RFC3339Nano),
			Modified:    n.Modified.UTC().Format(time.RFC3339Nano),
			LinkCount:   len(n.Links),
			BodyPreview: preview,
		}
	}
	enc := json.NewEncoder(outWriter(cmd))
	if len(requestedFields) == 0 {
		return enc.Encode(out)
	}
	projected := make([]map[string]any, len(out))
	for i, v := range out {
		projected[i] = projectFields(v, requestedFields)
	}
	return enc.Encode(projected)
}

// tagsString is used by status command.
func tagsString(tags []string) string {
	return strings.Join(tags, ", ")
}

// suppress unused warning
var _ = tagsString

// isUnactionedNote returns true when a note appears in the access.log but has had
// no git commit touching its file since the last access timestamp.
// Returns false (not unactioned) on any error so failures are silent.
func isUnactionedNote(n *note.Note, repoDir string) bool {
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
