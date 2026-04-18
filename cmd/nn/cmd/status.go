package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

const hubThreshold = 10 // minimum notes for hub section to appear
const defaultHubs = 5

func newStatusCmd(state *rootState) *cobra.Command {
	var jsonOut bool
	var hubsN int

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Notebook health: orphan notes, draft count, broken links, long notes, hub notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("status: %w", err)
			}

			// Collect all note IDs and link targets.
			allIDs := make(map[string]bool, len(notes))
			targetIDs := make(map[string]bool)
			hasOutbound := make(map[string]bool)
			for _, n := range notes {
				allIDs[n.ID] = true
				for _, lnk := range n.Links {
					targetIDs[lnk.TargetID] = true
					hasOutbound[n.ID] = true
				}
			}

			// Compute link degree (inbound + outbound) for hub detection.
			degree := make(map[string]int, len(notes))
			for _, n := range notes {
				for _, lnk := range n.Links {
					degree[n.ID]++        // outbound
					degree[lnk.TargetID]++ // inbound
				}
			}

			var drafts, broken, unknownTypes, draftLinks int
			var orphanList []*note.Note
			var brokenList []string
			var longNotes []*note.Note

			for _, n := range notes {
				if !hasOutbound[n.ID] && !targetIDs[n.ID] {
					orphanList = append(orphanList, n)
				}
				if n.Status == note.StatusDraft {
					drafts++
				}
				if len(n.Body) > atomicityThreshold {
					longNotes = append(longNotes, n)
				}
				for _, lnk := range n.Links {
					if !allIDs[lnk.TargetID] {
						broken++
						brokenList = append(brokenList, fmt.Sprintf("%s→%s", n.ID, lnk.TargetID))
					}
					if !note.IsKnownLinkType(lnk.Type) {
						unknownTypes++
					}
					if lnk.Status == "draft" {
						draftLinks++
					}
				}
			}

			// Hub notes: top N by degree, only when enough notes exist.
			var hubList []*note.Note
			if len(notes) >= hubThreshold {
				sorted := make([]*note.Note, len(notes))
				copy(sorted, notes)
				sort.Slice(sorted, func(i, j int) bool {
					return degree[sorted[i].ID] > degree[sorted[j].ID]
				})
				n := hubsN
				if n <= 0 {
					n = defaultHubs
				}
				if n > len(sorted) {
					n = len(sorted)
				}
				// Only include notes with degree > 0.
				for _, note := range sorted[:n] {
					if degree[note.ID] > 0 {
						hubList = append(hubList, note)
					}
				}
			}

			w := outWriter(cmd)

			if jsonOut {
				type orphanEntry struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}
				type brokenEntry struct {
					From string `json:"from"`
					To   string `json:"to"`
				}
				type longEntry struct {
					ID      string `json:"id"`
					Title   string `json:"title"`
					BodyLen int    `json:"body_len"`
				}
				type hubEntry struct {
					ID     string `json:"id"`
					Title  string `json:"title"`
					Degree int    `json:"degree"`
				}
				orphans := make([]orphanEntry, len(orphanList))
				for i, o := range orphanList {
					orphans[i] = orphanEntry{ID: o.ID, Title: o.Title}
				}
				brokens := make([]brokenEntry, len(brokenList))
				for i, b := range brokenList {
					brokens[i] = brokenEntry{From: b}
				}
				longs := make([]longEntry, len(longNotes))
				for i, ln := range longNotes {
					longs[i] = longEntry{ID: ln.ID, Title: ln.Title, BodyLen: len(ln.Body)}
				}
				hubs := make([]hubEntry, len(hubList))
				for i, h := range hubList {
					hubs[i] = hubEntry{ID: h.ID, Title: h.Title, Degree: degree[h.ID]}
				}
				out := struct {
					Total            int           `json:"total"`
					Orphans          []orphanEntry `json:"orphans"`
					Drafts           int           `json:"drafts"`
					BrokenLinks      []brokenEntry `json:"broken_links"`
					UnknownLinkTypes int           `json:"unknown_link_types"`
					DraftLinks       int           `json:"draft_links"`
					LongNotes        []longEntry   `json:"long_notes"`
					HubNotes         []hubEntry    `json:"hub_notes"`
				}{
					Total:            len(notes),
					Orphans:          orphans,
					Drafts:           drafts,
					BrokenLinks:      brokens,
					UnknownLinkTypes: unknownTypes,
					DraftLinks:       draftLinks,
					LongNotes:        longs,
					HubNotes:         hubs,
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			fmt.Fprintf(w, "total:   %d notes\n", len(notes))
			fmt.Fprintf(w, "orphans: %d\n", len(orphanList))
			for _, o := range orphanList {
				fmt.Fprintf(w, "  %s  %s\n", o.ID, o.Title)
			}
			fmt.Fprintf(w, "drafts:  %d\n", drafts)
			fmt.Fprintf(w, "broken links: %d\n", broken)
			for _, b := range brokenList {
				fmt.Fprintf(w, "  broken: %s\n", b)
			}
			fmt.Fprintf(w, "unknown link types: %d\n", unknownTypes)
			fmt.Fprintf(w, "draft links: %d\n", draftLinks)
			if len(longNotes) > 0 {
				fmt.Fprintf(w, "long notes (%d):\n", len(longNotes))
				for _, ln := range longNotes {
					fmt.Fprintf(w, "  %s  %s  %d chars\n", ln.ID, ln.Title, len(ln.Body))
				}
			}
			if len(hubList) > 0 {
				fmt.Fprintf(w, "hub notes (top %d by link degree):\n", len(hubList))
				for _, h := range hubList {
					fmt.Fprintf(w, "  %s  %s  degree %d\n", h.ID, h.Title, degree[h.ID])
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().IntVar(&hubsN, "hubs", defaultHubs, "Number of hub notes to show (default 5)")
	return cmd
}
