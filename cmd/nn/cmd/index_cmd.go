package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newIndexCmd(state *rootState) *cobra.Command {
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "index <topic>",
		Short: "Format topic cluster context for LLM-driven Map of Content creation",
		Long: `Loads all notes matching <topic> via BM25 search, groups them into clusters
using link-based label propagation, and formats the result as a structured
context block for the LLM to name clusters, identify tensions and gaps, and
create an index (Map of Content) note via 'nn new'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("index: %w", err)
			}

			// BM25 search for topic notes.
			scores := note.BM25Scores(notes, topic)
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

			// Build topic note ID set for cluster scoping.
			topicIDs := make(map[string]bool, len(topicNotes))
			for _, n := range topicNotes {
				topicIDs[n.ID] = true
			}

			// Label propagation over topic subset (undirected).
			adj := make(map[string][]string)
			for _, n := range topicNotes {
				for _, lnk := range n.Links {
					if topicIDs[lnk.TargetID] {
						adj[n.ID] = append(adj[n.ID], lnk.TargetID)
						adj[lnk.TargetID] = append(adj[lnk.TargetID], n.ID)
					}
				}
			}

			labels := make(map[string]string, len(topicNotes))
			for _, n := range topicNotes {
				labels[n.ID] = n.ID
			}

			for iter := 0; iter < 20; iter++ {
				changed := false
				ids := make([]string, 0, len(topicNotes))
				for _, n := range topicNotes {
					ids = append(ids, n.ID)
				}
				sort.Strings(ids)
				for _, id := range ids {
					neighbors := adj[id]
					if len(neighbors) == 0 {
						continue
					}
					freq := make(map[string]int)
					for _, nb := range neighbors {
						freq[labels[nb]]++
					}
					bestLabel := labels[id]
					bestCount := 0
					for lbl, cnt := range freq {
						if cnt > bestCount || (cnt == bestCount && lbl < bestLabel) {
							bestLabel = lbl
							bestCount = cnt
						}
					}
					if bestLabel != labels[id] {
						labels[id] = bestLabel
						changed = true
					}
				}
				if !changed {
					break
				}
			}

			// Group topic notes by label.
			groups := make(map[string][]*note.Note)
			for _, n := range topicNotes {
				lbl := labels[n.ID]
				groups[lbl] = append(groups[lbl], n)
			}

			type cluster struct {
				label string
				notes []*note.Note
			}
			var clusters []cluster
			for lbl, members := range groups {
				sort.Slice(members, func(i, j int) bool { return members[i].ID < members[j].ID })
				clusters = append(clusters, cluster{label: lbl, notes: members})
			}
			sort.Slice(clusters, func(i, j int) bool {
				if len(clusters[i].notes) != len(clusters[j].notes) {
					return len(clusters[i].notes) > len(clusters[j].notes)
				}
				return clusters[i].label < clusters[j].label
			})

			w := outWriter(cmd)

			if format == "json" {
				type noteEntry struct {
					ID      string   `json:"id"`
					Title   string   `json:"title"`
					Type    string   `json:"type"`
					Tags    []string `json:"tags"`
					Summary string   `json:"summary"`
				}
				type clusterJSON struct {
					Notes []noteEntry `json:"notes"`
				}
				type outputJSON struct {
					Topic      string       `json:"topic"`
					TopicNotes []noteEntry  `json:"topic_notes"`
					Clusters   []clusterJSON `json:"clusters"`
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
				out := outputJSON{
					Topic:      topic,
					TopicNotes: make([]noteEntry, len(topicNotes)),
					Clusters:   make([]clusterJSON, len(clusters)),
				}
				for i, n := range topicNotes {
					out.TopicNotes[i] = toEntry(n)
				}
				for i, c := range clusters {
					entries := make([]noteEntry, len(c.notes))
					for j, n := range c.notes {
						entries[j] = toEntry(n)
					}
					out.Clusters[i] = clusterJSON{Notes: entries}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			// Plain text format.
			fmt.Fprintf(w, "## Topic: %q (%d notes)\n\n", topic, len(topicNotes))
			for _, n := range topicNotes {
				fmt.Fprintf(w, "- %s  %s [%s]\n", n.ID, n.Title, n.Type)
				fmt.Fprintf(w, "  %s\n", summarize(n.Body, 200))
			}
			fmt.Fprintf(w, "\n## Clusters (%d)\n\n", len(clusters))
			for i, c := range clusters {
				fmt.Fprintf(w, "### Cluster %d (%d notes)\n", i+1, len(c.notes))
				for _, n := range c.notes {
					fmt.Fprintf(w, "  %s  %s [%s]\n", n.ID, n.Title, n.Type)
					fmt.Fprintf(w, "    %s\n", summarize(n.Body, 100))
				}
				fmt.Fprintf(w, "\n")
			}
			fmt.Fprintf(w, "---\n")
			fmt.Fprintf(w, "Suggested next step: name each cluster, identify tensions and gaps, then run:\n")
			fmt.Fprintf(w, "  nn new --title \"Index: %s\" --type concept --content \"...\" --no-edit\n\n", topic)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 30, "Maximum number of topic notes to include")
	cmd.Flags().StringVar(&format, "format", "", "Output format: json")
	return cmd
}
