package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

type graphRouteCandidate struct {
	id        string
	relevance float64
	hops      int
}

func sortGraphRouteCandidates(candidates []graphRouteCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].relevance != candidates[j].relevance {
			return candidates[i].relevance > candidates[j].relevance
		}
		if candidates[i].hops != candidates[j].hops {
			return candidates[i].hops < candidates[j].hops
		}
		return candidates[i].id < candidates[j].id
	})
}

func newGraphRoutesCmd(state *rootState) *cobra.Command {
	var focus string
	var links string
	var search string
	var limit int
	var jsonOut bool
	var explain bool

	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Discover query-relevant destinations reachable by typed directed links",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			focus = strings.TrimSpace(focus)
			if focus == "" {
				return fmt.Errorf("graph routes: --focus requires a non-blank ID")
			}
			if strings.TrimSpace(links) == "" {
				return fmt.Errorf("graph routes: --links requires at least one link type")
			}
			allowedTypes := make(map[string]bool)
			for _, raw := range strings.Split(links, ",") {
				linkType := strings.TrimSpace(raw)
				if linkType == "" {
					return fmt.Errorf("graph routes: --links contains an empty value")
				}
				if !note.IsKnownLinkType(linkType) {
					return fmt.Errorf("graph routes: unknown link type %q (known: %s)", linkType, strings.Join(note.LinkTypeOrder, ", "))
				}
				allowedTypes[linkType] = true
			}
			search = strings.TrimSpace(search)
			if search == "" {
				return fmt.Errorf("graph routes: --search requires a non-blank query")
			}
			if explain && !jsonOut {
				return fmt.Errorf("graph routes: --explain requires --json")
			}
			if !jsonOut {
				return fmt.Errorf("graph routes: requires --json")
			}
			if limit <= 0 {
				return fmt.Errorf("graph routes: --limit must be greater than zero")
			}

			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph routes: %w", err)
			}
			byID := make(map[string]*note.Note, len(notes))
			titles := make(map[string]string, len(notes))
			adj := make(map[string][]typedTraversalEdge, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
				titles[n.ID] = n.Title
				for _, lnk := range n.Links {
					if allowedTypes[lnk.Type] {
						adj[n.ID] = append(adj[n.ID], typedTraversalEdge{
							next: lnk.TargetID,
							edge: typedWitnessEdge{From: n.ID, To: lnk.TargetID, Type: lnk.Type, Annotation: lnk.Annotation},
						})
					}
				}
			}
			if byID[focus] == nil {
				return fmt.Errorf("graph routes: note %q not found", focus)
			}
			witnessSearch := findShortestTypedWitnesses(focus, titles, adj, 0)

			scoringNotes := append([]*note.Note(nil), notes...)
			sort.Slice(scoringNotes, func(i, j int) bool { return scoringNotes[i].ID < scoringNotes[j].ID })
			directScores := note.BM25Scores(scoringNotes, search, nil)
			rawScores := RankedByQuery(scoringNotes, scoringNotes, search, state.notebookDir)
			maxScore := 0.0
			for _, score := range rawScores {
				if score > maxScore {
					maxScore = score
				}
			}

			type graphRouteDiagnostics struct {
				QueryTokens          []string `json:"query_tokens"`
				TotalNotes           int      `json:"total_notes"`
				DirectLexicalMatches int      `json:"direct_lexical_matches"`
				FocusExcluded        int      `json:"focus_excluded"`
				TypedReachable       int      `json:"typed_reachable"`
				EligibleDestinations int      `json:"eligible_destinations"`
				GraphScoredMatches   int      `json:"graph_scored_matches"`
				Returned             int      `json:"returned"`
				TruncatedByLimit     bool     `json:"truncated_by_limit"`
			}
			diagnostics := graphRouteDiagnostics{
				QueryTokens: note.Tokenize(search),
				TotalNotes:  len(scoringNotes),
			}
			if diagnostics.QueryTokens == nil {
				diagnostics.QueryTokens = []string{}
			}

			var candidates []graphRouteCandidate
			for _, n := range scoringNotes {
				directMatch := directScores[n.ID] > 0
				if directMatch {
					diagnostics.DirectLexicalMatches++
					if n.ID == focus {
						diagnostics.FocusExcluded = 1
					}
				}
				if rawScores[n.ID] > 0 {
					diagnostics.GraphScoredMatches++
				}
				if n.ID == focus {
					continue
				}
				if _, reachable := witnessSearch.depthByID[n.ID]; !reachable {
					continue
				}
				diagnostics.TypedReachable++
				if !directMatch {
					continue
				}
				diagnostics.EligibleDestinations++
				if maxScore > 0 {
					if score := rawScores[n.ID]; score > 0 {
						candidates = append(candidates, graphRouteCandidate{id: n.ID, relevance: score / maxScore, hops: witnessSearch.depthByID[n.ID]})
					}
				}
			}
			sortGraphRouteCandidates(candidates)
			diagnostics.TruncatedByLimit = len(candidates) > limit
			if diagnostics.TruncatedByLimit {
				candidates = candidates[:limit]
			}

			type routeDestination struct {
				ID             string  `json:"id"`
				Title          string  `json:"title"`
				RelevanceScore float64 `json:"relevance_score"`
			}
			type routeResult struct {
				Destination routeDestination `json:"destination"`
				Witnesses   []typedWitness   `json:"witnesses"`
			}
			results := make([]routeResult, 0, len(candidates))
			for _, candidate := range candidates {
				results = append(results, routeResult{
					Destination: routeDestination{ID: candidate.id, Title: byID[candidate.id].Title, RelevanceScore: candidate.relevance},
					Witnesses:   witnessSearch.witnessesTo(candidate.id),
				})
			}
			diagnostics.Returned = len(results)
			enc := json.NewEncoder(outWriter(cmd))
			enc.SetIndent("", "  ")
			if explain {
				return enc.Encode(struct {
					Routes      []routeResult         `json:"routes"`
					Diagnostics graphRouteDiagnostics `json:"diagnostics"`
				}{Routes: results, Diagnostics: diagnostics})
			}
			return enc.Encode(results)
		},
	}
	cmd.Flags().StringVar(&focus, "focus", "", "Starting note ID")
	cmd.Flags().StringVar(&links, "links", "", "Comma-separated link types to follow in stored direction")
	cmd.Flags().StringVar(&search, "search", "", "Destination relevance query")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum destinations to return")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output typed route witnesses as JSON (required)")
	cmd.Flags().BoolVar(&explain, "explain", false, "Wrap routes with bounded aggregate diagnostics (requires --json)")
	return cmd
}
