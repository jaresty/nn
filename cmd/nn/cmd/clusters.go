package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newClustersCmd(state *rootState) *cobra.Command {
	var minSize int
	var singletons bool
	var jsonOut bool
	var search string
	var summary bool
	var matchLimit int

	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Detect topological clusters of notes using label propagation",
		RunE: func(cmd *cobra.Command, args []string) error {
			searchMode := cmd.Flags().Changed("search")
			if cmd.Flags().Changed("match-limit") && !summary {
				return fmt.Errorf("clusters: --match-limit requires --summary")
			}
			if summary && matchLimit < 0 {
				return fmt.Errorf("clusters: --match-limit must be non-negative")
			}
			if summary && !searchMode {
				return fmt.Errorf("clusters: --summary requires --search")
			}
			if summary && !jsonOut {
				return fmt.Errorf("clusters: --summary requires --json")
			}
			if searchMode && strings.TrimSpace(search) == "" {
				return fmt.Errorf("clusters: --search requires a non-blank query")
			}
			if searchMode && !jsonOut {
				return fmt.Errorf("clusters: --search requires --json")
			}
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("clusters: %w", err)
			}

			clustering := labelPropagationClusters(notes)

			var normalizedSearchScores map[string]float64
			if searchMode {
				scoringNotes := append([]*note.Note(nil), notes...)
				sort.Slice(scoringNotes, func(i, j int) bool { return scoringNotes[i].ID < scoringNotes[j].ID })
				rawScores := RankedByQuery(scoringNotes, scoringNotes, search, state.notebookDir)
				maxScore := 0.0
				for _, score := range rawScores {
					if score > maxScore {
						maxScore = score
					}
				}
				normalizedSearchScores = make(map[string]float64, len(rawScores))
				if maxScore > 0 {
					for id, score := range rawScores {
						if score > 0 {
							normalizedSearchScores[id] = score / maxScore
						}
					}
				}
			}

			groups := clustering.groups

			// Build sorted cluster list. Search evidence projects onto these full-graph groups.
			type cluster struct {
				notes          []*note.Note
				matches        []*note.Note
				score          float64
				representative *note.Note
			}
			var clusters []cluster
			for _, members := range groups {
				size := len(members)
				effectiveMin := minSize
				if singletons && !cmd.Flags().Changed("min") {
					effectiveMin = 1
				}
				if effectiveMin <= 0 {
					effectiveMin = 2
				}
				if !singletons && size < 2 {
					continue
				}
				if size < effectiveMin {
					continue
				}
				c := cluster{notes: members}
				if searchMode {
					for _, n := range members {
						if score := normalizedSearchScores[n.ID]; score > 0 {
							c.matches = append(c.matches, n)
						}
					}
					if len(c.matches) == 0 {
						continue
					}
					c.representative = labelPropagationRepresentative(members, clustering.outDegree, clustering.inDegree)
					sort.Slice(c.matches, func(i, j int) bool {
						si, sj := normalizedSearchScores[c.matches[i].ID], normalizedSearchScores[c.matches[j].ID]
						if si != sj {
							return si > sj
						}
						return c.matches[i].ID < c.matches[j].ID
					})
					scoreLimit := len(c.matches)
					if scoreLimit > 3 {
						scoreLimit = 3
					}
					for _, match := range c.matches[:scoreLimit] {
						c.score += normalizedSearchScores[match.ID]
					}
				}
				clusters = append(clusters, c)
			}
			if searchMode {
				sort.Slice(clusters, func(i, j int) bool {
					if clusters[i].score != clusters[j].score {
						return clusters[i].score > clusters[j].score
					}
					return clusters[i].representative.ID < clusters[j].representative.ID
				})
			} else {
				// Preserve legacy order: size descending, then first member ID.
				sort.Slice(clusters, func(i, j int) bool {
					if len(clusters[i].notes) != len(clusters[j].notes) {
						return len(clusters[i].notes) > len(clusters[j].notes)
					}
					return clusters[i].notes[0].ID < clusters[j].notes[0].ID
				})
			}

			w := outWriter(cmd)
			if jsonOut {
				type noteEntry struct {
					ID    string  `json:"id"`
					Title string  `json:"title"`
					Score float64 `json:"score,omitempty"`
				}
				if searchMode {
					if summary {
						type summaryClusterEntry struct {
							Size             int         `json:"size"`
							MatchCount       int         `json:"match_count"`
							MatchDensity     float64     `json:"match_density"`
							Score            float64     `json:"score"`
							Representative   noteEntry   `json:"representative"`
							Matches          []noteEntry `json:"matches"`
							MatchesReturned  int         `json:"matches_returned"`
							MatchesTruncated bool        `json:"matches_truncated"`
						}
						out := make([]summaryClusterEntry, len(clusters))
						for i, c := range clusters {
							returnedMatches := c.matches
							if matchLimit > 0 && len(returnedMatches) > matchLimit {
								returnedMatches = returnedMatches[:matchLimit]
							}
							matches := make([]noteEntry, len(returnedMatches))
							for j, n := range returnedMatches {
								matches[j] = noteEntry{ID: n.ID, Title: n.Title, Score: normalizedSearchScores[n.ID]}
							}
							rep := c.representative
							out[i] = summaryClusterEntry{
								Size: len(c.notes), MatchCount: len(c.matches), MatchDensity: float64(len(c.matches)) / float64(len(c.notes)), Score: c.score,
								Representative:   noteEntry{ID: rep.ID, Title: rep.Title},
								Matches:          matches,
								MatchesReturned:  len(matches),
								MatchesTruncated: len(matches) < len(c.matches),
							}
						}
						enc := json.NewEncoder(w)
						enc.SetIndent("", "  ")
						return enc.Encode(out)
					}

					type searchClusterEntry struct {
						Size           int         `json:"size"`
						MatchCount     int         `json:"match_count"`
						MatchDensity   float64     `json:"match_density"`
						Score          float64     `json:"score"`
						Representative noteEntry   `json:"representative"`
						Matches        []noteEntry `json:"matches"`
						Notes          []noteEntry `json:"notes,omitempty"`
					}
					out := make([]searchClusterEntry, len(clusters))
					for i, c := range clusters {
						entries := make([]noteEntry, len(c.notes))
						for j, n := range c.notes {
							entries[j] = noteEntry{ID: n.ID, Title: n.Title}
						}
						matches := make([]noteEntry, len(c.matches))
						for j, n := range c.matches {
							matches[j] = noteEntry{ID: n.ID, Title: n.Title, Score: normalizedSearchScores[n.ID]}
						}
						rep := c.representative
						out[i] = searchClusterEntry{
							Size: len(c.notes), MatchCount: len(c.matches), MatchDensity: float64(len(c.matches)) / float64(len(c.notes)), Score: c.score,
							Representative: noteEntry{ID: rep.ID, Title: rep.Title},
							Matches:        matches, Notes: entries,
						}
					}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(out)
				}
				type clusterEntry struct {
					Notes []noteEntry `json:"notes"`
				}
				out := make([]clusterEntry, len(clusters))
				for i, c := range clusters {
					entries := make([]noteEntry, len(c.notes))
					for j, n := range c.notes {
						entries[j] = noteEntry{ID: n.ID, Title: n.Title}
					}
					out[i] = clusterEntry{Notes: entries}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			for i, c := range clusters {
				entries := make([]string, len(c.notes))
				for j, n := range c.notes {
					entries[j] = n.ID + "  " + n.Title
				}
				fmt.Fprintf(w, "cluster %d (%d notes):\n", i+1, len(c.notes))
				for _, e := range entries {
					fmt.Fprintf(w, "  %s\n", e)
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&minSize, "min", 2, "Omit clusters smaller than N notes")
	cmd.Flags().BoolVar(&singletons, "singletons", false, "Include singleton clusters (notes with no links)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&search, "search", "", "Return only full-graph clusters containing notes matching this query (requires --json)")
	cmd.Flags().BoolVar(&summary, "summary", false, "Omit full cluster membership from search JSON while retaining landing handles")
	cmd.Flags().IntVar(&matchLimit, "match-limit", 3, "Maximum ranked matches per summary cluster (0 = all; requires --summary)")
	return cmd
}
