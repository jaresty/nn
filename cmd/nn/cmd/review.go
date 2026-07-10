package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// nnConfigDir returns the nn config directory, respecting NN_CONFIG_DIR and XDG_CONFIG_HOME.
func nnConfigDir() string {
	if d := os.Getenv("NN_CONFIG_DIR"); d != "" {
		return d
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "nn")
}

// readProtocolPresenceCounts reads protocol-presence.log and returns a map of
// protocol ID → session count (number of lines containing that ID).
func readProtocolPresenceCounts() map[string]int {
	counts := make(map[string]int)
	path := filepath.Join(nnConfigDir(), "protocol-presence.log")
	f, err := os.Open(path)
	if err != nil {
		return counts
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// Format: <timestamp> <id1> <id2> ...
		for _, id := range fields[1:] {
			counts[id]++
		}
	}
	return counts
}

// readNoteAccessCounts reads access.log and returns a map of note ID → show count.
func readNoteAccessCounts() map[string]int {
	counts := make(map[string]int)
	path := filepath.Join(nnConfigDir(), "access.log")
	f, err := os.Open(path)
	if err != nil {
		return counts
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Format: <timestamp> show <id>
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 && fields[1] == "show" {
			counts[fields[2]]++
		}
	}
	return counts
}

func newReviewCmd(state *rootState) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Notebook health report formatted for LLM-driven analysis",
		Long: `Produces a structured Markdown health report covering growth, connectivity,
and structural gaps. Output is ready to paste into an LLM session for
interpretation and recommendations.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("review: %w", err)
			}

			now := time.Now().UTC()
			last7 := now.AddDate(0, 0, -7)
			last30 := now.AddDate(0, 0, -30)

			// ── Growth ────────────────────────────────────────────────────────
			totalNotes := len(notes)
			byType := make(map[string]int)
			var recent7, recent30 int
			for _, n := range notes {
				byType[string(n.Type)]++
				if n.Created.After(last7) {
					recent7++
				}
				if n.Created.After(last30) {
					recent30++
				}
			}

			// ── Connectivity ──────────────────────────────────────────────────
			// Build inbound link count.
			inbound := make(map[string]int)
			outbound := make(map[string]int)
			for _, n := range notes {
				for _, lnk := range n.Links {
					outbound[n.ID]++
					inbound[lnk.TargetID]++
				}
			}

			var totalLinks int
			for _, n := range notes {
				totalLinks += outbound[n.ID]
			}

			var avgLinks float64
			if totalNotes > 0 {
				avgLinks = float64(totalLinks) / float64(totalNotes)
			}

			// Orphans: no links in either direction.
			// Global notes (type=protocol, status=permanent) are excluded — their
			// connectivity is by design; they are referenced at session start, not via links.
			var orphans []*note.Note
			for _, n := range notes {
				if outbound[n.ID] == 0 && inbound[n.ID] == 0 &&
					!(n.Type == note.TypeProtocol && n.Status == note.StatusPermanent) {
					orphans = append(orphans, n)
				}
			}

			// Dead-ends: has outbound links but no inbound links.
			var deadEnds []*note.Note
			for _, n := range notes {
				if outbound[n.ID] > 0 && inbound[n.ID] == 0 {
					deadEnds = append(deadEnds, n)
				}
			}

			// Draft notes.
			var drafts []*note.Note
			for _, n := range notes {
				if n.Status == note.StatusDraft {
					drafts = append(drafts, n)
				}
			}

			// Sort by ID for stable output.
			sortByID := func(ns []*note.Note) {
				sort.Slice(ns, func(i, j int) bool { return ns[i].ID < ns[j].ID })
			}
			sortByID(orphans)
			sortByID(deadEnds)
			sortByID(drafts)

			// Long notes: body exceeds atomicityThreshold bytes.
			var longNotes []*note.Note
			for _, n := range notes {
				if len(n.Body) > atomicityThreshold {
					longNotes = append(longNotes, n)
				}
			}
			sortByID(longNotes)

			// Aging notes: split into aging (3–14 days) and stale (>14 days) by modified time.
			var agingNotes, staleNotes []*note.Note
			for _, n := range notes {
				age := now.Sub(n.Modified)
				switch {
				case age >= 14*24*time.Hour:
					staleNotes = append(staleNotes, n)
				case age >= 3*24*time.Hour:
					agingNotes = append(agingNotes, n)
				}
			}
			// Sort oldest-first within each bucket.
			sortOldestFirst := func(ns []*note.Note) {
				sort.Slice(ns, func(i, j int) bool { return ns[i].Modified.Before(ns[j].Modified) })
			}
			sortOldestFirst(agingNotes)
			sortOldestFirst(staleNotes)

			// Expired notes: notes with an expires date in the past.
			var expiredNotes []*note.Note
			for _, n := range notes {
				if n.Expires != nil && n.Expires.Before(now) {
					expiredNotes = append(expiredNotes, n)
				}
			}
			sortByID(expiredNotes)

			// Pending conditions: notes with expires_when set.
			var pendingConditions []*note.Note
			for _, n := range notes {
				if n.ExpiresWhen != "" {
					pendingConditions = append(pendingConditions, n)
				}
			}
			sortByID(pendingConditions)

			// Expiry candidates: observation notes, age >30 days, no expires/expires_when, not permanent.
			var expiryCandidates []*note.Note
			for _, n := range notes {
				if n.Type != note.TypeObservation {
					continue
				}
				if n.Status == note.StatusPermanent {
					continue
				}
				if n.Expires != nil || n.ExpiresWhen != "" {
					continue
				}
				if now.Sub(n.Modified) > 30*24*time.Hour {
					expiryCandidates = append(expiryCandidates, n)
				}
			}
			sortOldestFirst(expiryCandidates)

			// Friction candidates: observation notes tagged friction-candidate but not reviewed.
			var frictionCandidates []*note.Note
			for _, n := range notes {
				if n.Type != note.TypeObservation {
					continue
				}
				hasFriction, hasReviewed := false, false
				for _, tag := range n.Tags {
					if tag == "friction-candidate" {
						hasFriction = true
					}
					if tag == "reviewed" {
						hasReviewed = true
					}
				}
				if hasFriction && !hasReviewed {
					frictionCandidates = append(frictionCandidates, n)
				}
			}
			sortByID(frictionCandidates)

			w := outWriter(cmd)
			protocolCounts := readProtocolPresenceCounts()
			accessCounts := readNoteAccessCounts()

			if format == "json" {
				type noteRef struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}
				toRefs := func(ns []*note.Note) []noteRef {
					out := make([]noteRef, len(ns))
					for i, n := range ns {
						out[i] = noteRef{ID: n.ID, Title: n.Title}
					}
					return out
				}
				type growthJSON struct {
					TotalNotes  int            `json:"total_notes"`
					ByType      map[string]int `json:"by_type"`
					Last7Days   int            `json:"last_7_days"`
					Last30Days  int            `json:"last_30_days"`
				}
				type connectivityJSON struct {
					TotalLinks  int       `json:"total_links"`
					AvgLinks    float64   `json:"avg_links_per_note"`
					OrphanCount int       `json:"orphan_count"`
					DeadEndCount int      `json:"dead_end_count"`
					Orphans     []noteRef `json:"orphans"`
					DeadEnds    []noteRef `json:"dead_ends"`
				}
				type reviewJSON struct {
					Growth              growthJSON       `json:"growth"`
					Connectivity        connectivityJSON `json:"connectivity"`
					Drafts              []noteRef        `json:"drafts"`
					LongNotes           []noteRef        `json:"long_notes"`
					AgingNotes          []noteRef        `json:"aging_notes"`
					StaleNotes          []noteRef        `json:"stale_notes"`
					ExpiredNotes        []noteRef        `json:"expired_notes"`
					PendingConditions   []noteRef        `json:"pending_conditions"`
					ExpiryCandidates    []noteRef        `json:"expiry_candidates"`
					FrictionCandidates  []noteRef        `json:"friction_candidates"`
					ProtocolTelemetry   map[string]int   `json:"protocol_telemetry"`
					NoteAccess          map[string]int   `json:"note_access"`
				}
				out := reviewJSON{
					Growth: growthJSON{
						TotalNotes: totalNotes,
						ByType:     byType,
						Last7Days:  recent7,
						Last30Days: recent30,
					},
					Connectivity: connectivityJSON{
						TotalLinks:   totalLinks,
						AvgLinks:     avgLinks,
						OrphanCount:  len(orphans),
						DeadEndCount: len(deadEnds),
						Orphans:      toRefs(orphans),
						DeadEnds:     toRefs(deadEnds),
					},
					Drafts:             toRefs(drafts),
					LongNotes:          toRefs(longNotes),
					AgingNotes:         toRefs(agingNotes),
					StaleNotes:         toRefs(staleNotes),
					ExpiredNotes:       toRefs(expiredNotes),
					PendingConditions:  toRefs(pendingConditions),
					ExpiryCandidates:   toRefs(expiryCandidates),
					FrictionCandidates: toRefs(frictionCandidates),
					ProtocolTelemetry:  protocolCounts,
					NoteAccess:         accessCounts,
				}
				if out.Drafts == nil {
					out.Drafts = []noteRef{}
				}
				if out.LongNotes == nil {
					out.LongNotes = []noteRef{}
				}
				if out.AgingNotes == nil {
					out.AgingNotes = []noteRef{}
				}
				if out.StaleNotes == nil {
					out.StaleNotes = []noteRef{}
				}
				if out.ExpiredNotes == nil {
					out.ExpiredNotes = []noteRef{}
				}
				if out.PendingConditions == nil {
					out.PendingConditions = []noteRef{}
				}
				if out.ExpiryCandidates == nil {
					out.ExpiryCandidates = []noteRef{}
				}
				if out.FrictionCandidates == nil {
					out.FrictionCandidates = []noteRef{}
				}
				if out.ProtocolTelemetry == nil {
					out.ProtocolTelemetry = map[string]int{}
				}
				if out.NoteAccess == nil {
					out.NoteAccess = map[string]int{}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			// Plain Markdown format.
			fmt.Fprintf(w, "# Notebook Review\n\n")

			fmt.Fprintf(w, "## Growth\n\n")
			fmt.Fprintf(w, "Total notes: %d\n", totalNotes)
			fmt.Fprintf(w, "Created in last 7 days: %d\n", recent7)
			fmt.Fprintf(w, "Created in last 30 days: %d\n", recent30)
			if len(byType) > 0 {
				// Sort type names for stable output.
				types := make([]string, 0, len(byType))
				for t := range byType {
					types = append(types, t)
				}
				sort.Strings(types)
				fmt.Fprintf(w, "By type:\n")
				for _, t := range types {
					fmt.Fprintf(w, "  %s: %d\n", t, byType[t])
				}
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Connectivity\n\n")
			fmt.Fprintf(w, "Total links: %d\n", totalLinks)
			fmt.Fprintf(w, "Avg links per note: %.2f\n", avgLinks)
			fmt.Fprintf(w, "Orphans: %d (no links in either direction)\n", len(orphans))
			for _, n := range orphans {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "Dead-ends: %d (outbound links but no inbound)\n", len(deadEnds))
			for _, n := range deadEnds {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Structural gaps\n\n")
			fmt.Fprintf(w, "Draft notes: %d\n", len(drafts))
			for _, n := range drafts {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "Long notes: %d (body > %d bytes — consider splitting)\n", len(longNotes), atomicityThreshold)
			for _, n := range longNotes {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Aging notes\n\n")
			fmt.Fprintf(w, "aging (3–14 days): %d\n", len(agingNotes))
			for _, n := range agingNotes {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "stale (>14 days): %d\n", len(staleNotes))
			for _, n := range staleNotes {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Expiring notes\n\n")
			fmt.Fprintf(w, "expired: %d\n", len(expiredNotes))
			for _, n := range expiredNotes {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Pending conditions\n\n")
			fmt.Fprintf(w, "notes with expires_when: %d\n", len(pendingConditions))
			for _, n := range pendingConditions {
				fmt.Fprintf(w, "  %s  %s — %s\n", n.ID, n.Title, n.ExpiresWhen)
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Expiry candidates\n\n")
			fmt.Fprintf(w, "observations >30 days old with no expiry set: %d\n", len(expiryCandidates))
			for _, n := range expiryCandidates {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			if len(expiryCandidates) > 0 {
				fmt.Fprintf(w, "\nFor each: set nn update <id> --expires DATE or --expires-when \"condition\", or promote to permanent if still relevant.\n")
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Friction candidates\n\n")
			fmt.Fprintf(w, "Unreviewed friction observations: %d\n", len(frictionCandidates))
			for _, n := range frictionCandidates {
				fmt.Fprintf(w, "  %s  %s\n", n.ID, n.Title)
			}
			if len(frictionCandidates) > 0 {
				fmt.Fprintf(w, "\nFor each: promote to protocol (nn new --type protocol), discard (nn update <id> --tags reviewed), or link as evidence (nn link <id> <protocol-id> --type supports && nn update <id> --tags reviewed).\n")
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Protocol telemetry\n\n")
			if len(protocolCounts) == 0 {
				fmt.Fprintf(w, "No data yet (protocol-presence.log not found or empty).\n")
			} else {
				type protocolEntry struct {
					ID    string
					Count int
				}
				pentries := make([]protocolEntry, 0, len(protocolCounts))
				for id, count := range protocolCounts {
					pentries = append(pentries, protocolEntry{id, count})
				}
				sort.Slice(pentries, func(i, j int) bool {
					if pentries[i].Count != pentries[j].Count {
						return pentries[i].Count > pentries[j].Count
					}
					return pentries[i].ID < pentries[j].ID
				})
				for _, e := range pentries {
					fmt.Fprintf(w, "  %s: %d sessions\n", e.ID, e.Count)
				}
			}
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "## Note access\n\n")
			if len(accessCounts) == 0 {
				fmt.Fprintf(w, "No data yet (access.log not found or empty).\n")
			} else {
				type accessEntry struct {
					ID    string
					Count int
				}
				aentries := make([]accessEntry, 0, len(accessCounts))
				for id, count := range accessCounts {
					aentries = append(aentries, accessEntry{id, count})
				}
				sort.Slice(aentries, func(i, j int) bool {
					if aentries[i].Count != aentries[j].Count {
						return aentries[i].Count > aentries[j].Count
					}
					return aentries[i].ID < aentries[j].ID
				})
				for _, e := range aentries {
					fmt.Fprintf(w, "  %s: %d views\n", e.ID, e.Count)
				}
			}
			fmt.Fprintf(w, "\n")

			// Required Actions — imperative checklist for the LLM to work through.
			fmt.Fprintf(w, "## Required Actions\n\n")
			fmt.Fprintf(w, "Work through each item below before closing this session.\n\n")
			if len(orphans) > 0 {
				fmt.Fprintf(w, "- [ ] Link or delete %d orphan notes (run: nn list --orphan)\n", len(orphans))
			}
			if len(deadEnds) > 0 {
				fmt.Fprintf(w, "- [ ] Add backlinks to %d dead-end notes (run: nn list --orphan to find candidates)\n", len(deadEnds))
			}
			if len(drafts) > 0 {
				fmt.Fprintf(w, "- [ ] Promote or prune %d draft notes\n", len(drafts))
			}
			if len(longNotes) > 0 {
				fmt.Fprintf(w, "- [ ] Split or rewrite %d long notes:", len(longNotes))
				for _, n := range longNotes {
					fmt.Fprintf(w, " %s", n.ID)
				}
				fmt.Fprintf(w, "\n")
			}
			if len(agingNotes) > 0 {
				fmt.Fprintf(w, "- [ ] Review %d aging notes for accuracy (run: nn list --since <3-days-ago>)\n", len(agingNotes))
			}
			if len(expiredNotes) > 0 {
				fmt.Fprintf(w, "- [ ] Delete or update %d expired notes\n", len(expiredNotes))
			}
			if len(pendingConditions) > 0 {
				fmt.Fprintf(w, "- [ ] Resolve %d conditional-friction notes (run: nn list --has-expires)\n", len(pendingConditions))
			}
			if len(frictionCandidates) > 0 {
				fmt.Fprintf(w, "- [ ] Promote or discard %d friction candidates\n", len(frictionCandidates))
			}
			if len(orphans) == 0 && len(deadEnds) == 0 && len(drafts) == 0 && len(longNotes) == 0 &&
				len(agingNotes) == 0 && len(expiredNotes) == 0 && len(pendingConditions) == 0 && len(frictionCandidates) == 0 {
				fmt.Fprintf(w, "No actions required — notebook is healthy.\n")
			}
			fmt.Fprintf(w, "\n")

			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Output format: json")
	return cmd
}

// notesByStatus filters notes by status.
func notesByStatus(notes []*note.Note, status note.Status) []*note.Note {
	var out []*note.Note
	for _, n := range notes {
		if n.Status == status {
			out = append(out, n)
		}
	}
	return out
}

// formatTypeList formats a type→count map as a sorted string list.
func formatTypeList(byType map[string]int) string {
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	parts := make([]string, len(types))
	for i, t := range types {
		parts[i] = fmt.Sprintf("%s=%d", t, byType[t])
	}
	return strings.Join(parts, ", ")
}
