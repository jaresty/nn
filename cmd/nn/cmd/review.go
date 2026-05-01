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

			w := outWriter(cmd)

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
					Growth       growthJSON       `json:"growth"`
					Connectivity connectivityJSON `json:"connectivity"`
					Drafts       []noteRef        `json:"drafts"`
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
					Drafts: toRefs(drafts),
				}
				if out.Drafts == nil {
					out.Drafts = []noteRef{}
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
