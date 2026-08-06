package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newGapCmd(state *rootState) *cobra.Command {
	var limit int
	var depth int
	var format string

	cmd := &cobra.Command{
		Use:   "gap <topic>",
		Short: "Format topic neighborhood context for LLM gap analysis",
		Long: `Loads all notes matching <topic> via BM25 search, then expands to their
direct neighbors (backlinks and forward links) and formats the result as a
structured context block for LLM-driven gap analysis.

The LLM receiving this output is expected to identify what is thoroughly
covered, what is shallow, what is absent, and what questions the notes raise
but do not answer.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("gap: %w", err)
			}

			// BM25 search for topic notes.
			scores := note.BM25Scores(notes, topic, nil)
			var topicNotes []*note.Note
			for _, n := range notes {
				if scores[n.ID] > 0 {
					topicNotes = append(topicNotes, n)
				}
			}
			sort.SliceStable(topicNotes, func(i, j int) bool {
				return scores[topicNotes[i].ID] > scores[topicNotes[j].ID]
			})
			if limit > 0 && len(topicNotes) > limit {
				topicNotes = topicNotes[:limit]
			}

			// Build index for quick lookup.
			noteIndex := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				noteIndex[n.ID] = n
			}

			// Build inbound link index.
			inbound := make(map[string][]string) // targetID -> []sourceID
			for _, n := range notes {
				for _, lnk := range n.Links {
					inbound[lnk.TargetID] = append(inbound[lnk.TargetID], n.ID)
				}
			}

			// Expand neighborhood up to `depth` hops (default 1).
			topicIDs := make(map[string]bool)
			for _, n := range topicNotes {
				topicIDs[n.ID] = true
			}

			neighborIDs := make(map[string]bool)
			frontier := make(map[string]bool)
			for id := range topicIDs {
				frontier[id] = true
			}
			for hop := 0; hop < depth; hop++ {
				next := make(map[string]bool)
				for id := range frontier {
					n, ok := noteIndex[id]
					if !ok {
						continue
					}
					// Forward links.
					for _, lnk := range n.Links {
						if !topicIDs[lnk.TargetID] {
							neighborIDs[lnk.TargetID] = true
							next[lnk.TargetID] = true
						}
					}
					// Backlinks.
					for _, srcID := range inbound[id] {
						if !topicIDs[srcID] {
							neighborIDs[srcID] = true
							next[srcID] = true
						}
					}
				}
				frontier = next
			}

			// Collect neighbor notes, sorted by ID.
			var neighbors []*note.Note
			for id := range neighborIDs {
				if n, ok := noteIndex[id]; ok {
					neighbors = append(neighbors, n)
				}
			}
			sort.Slice(neighbors, func(i, j int) bool { return neighbors[i].ID < neighbors[j].ID })

			w := outWriter(cmd)

			if format == "json" {
				type noteEntry struct {
					ID      string   `json:"id"`
					Title   string   `json:"title"`
					Type    string   `json:"type"`
					Tags    []string `json:"tags"`
					Summary string   `json:"summary"`
				}
				toEntry := func(n *note.Note) noteEntry {
					return noteEntry{
						ID:      n.ID,
						Title:   n.Title,
						Type:    string(n.Type),
						Tags:    tagsOrEmpty(n.Tags),
						Summary: summarize(n.Body, 200),
					}
				}
				type gapJSON struct {
					Topic      string      `json:"topic"`
					TopicNotes []noteEntry `json:"topic_notes"`
					Neighbors  []noteEntry `json:"neighbors"`
				}
				out := gapJSON{
					Topic:      topic,
					TopicNotes: make([]noteEntry, len(topicNotes)),
					Neighbors:  make([]noteEntry, len(neighbors)),
				}
				for i, n := range topicNotes {
					out.TopicNotes[i] = toEntry(n)
				}
				for i, n := range neighbors {
					out.Neighbors[i] = toEntry(n)
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			// Plain text format.
			fmt.Fprintf(w, "## Topic notes (%d matching %q)\n\n", len(topicNotes), topic)
			for _, n := range topicNotes {
				fmt.Fprintf(w, "### %s — %s [%s]\n", n.ID, n.Title, n.Type)
				fmt.Fprintf(w, "tags: %s\n", strings.Join(n.Tags, ", "))
				fmt.Fprintf(w, "summary: %s\n\n", summarize(n.Body, 200))
			}

			fmt.Fprintf(w, "## Neighborhood (%d notes, depth %d)\n\n", len(neighbors), depth)
			for _, n := range neighbors {
				fmt.Fprintf(w, "### %s — %s [%s]\n", n.ID, n.Title, n.Type)
				fmt.Fprintf(w, "tags: %s\n", strings.Join(n.Tags, ", "))
				fmt.Fprintf(w, "summary: %s\n\n", summarize(n.Body, 200))
			}

			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of topic notes to include")
	cmd.Flags().IntVar(&depth, "depth", 1, "Neighborhood expansion depth")
	cmd.Flags().StringVar(&format, "format", "", "Output format: json")
	return cmd
}
