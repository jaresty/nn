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
			adj := make(map[string][]pathAdjacencyEdge, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
				for _, lnk := range n.Links {
					if allowedTypes[lnk.Type] {
						adj[n.ID] = append(adj[n.ID], pathAdjacencyEdge{
							to: lnk.TargetID, linkType: lnk.Type, annotation: lnk.Annotation,
						})
					}
				}
			}
			if byID[focus] == nil {
				return fmt.Errorf("graph routes: note %q not found", focus)
			}
			for id := range adj {
				sort.Slice(adj[id], func(i, j int) bool {
					a, b := adj[id][i], adj[id][j]
					if a.to != b.to {
						return a.to < b.to
					}
					if a.linkType != b.linkType {
						return a.linkType < b.linkType
					}
					return a.annotation < b.annotation
				})
			}

			prev := map[string]string{focus: ""}
			prevEdge := make(map[string]pathWitnessEdge)
			hops := map[string]int{focus: 0}
			queue := []string{focus}
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				for _, edge := range adj[current] {
					if _, visited := prev[edge.to]; visited {
						continue
					}
					prev[edge.to] = current
					prevEdge[edge.to] = pathWitnessEdge{
						From: current, To: edge.to, Type: edge.linkType, Annotation: edge.annotation,
					}
					hops[edge.to] = hops[current] + 1
					queue = append(queue, edge.to)
				}
			}

			scoringNotes := append([]*note.Note(nil), notes...)
			sort.Slice(scoringNotes, func(i, j int) bool { return scoringNotes[i].ID < scoringNotes[j].ID })
			rawScores := RankedByQuery(scoringNotes, scoringNotes, search, state.notebookDir)
			maxScore := 0.0
			for _, score := range rawScores {
				if score > maxScore {
					maxScore = score
				}
			}

			var candidates []graphRouteCandidate
			if maxScore > 0 {
				for _, n := range scoringNotes {
					if n.ID == focus {
						continue
					}
					if _, reachable := prev[n.ID]; !reachable {
						continue
					}
					if score := rawScores[n.ID]; score > 0 {
						candidates = append(candidates, graphRouteCandidate{id: n.ID, relevance: score / maxScore, hops: hops[n.ID]})
					}
				}
			}
			sortGraphRouteCandidates(candidates)
			if len(candidates) > limit {
				candidates = candidates[:limit]
			}

			type routeNode struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			}
			type routeDestination struct {
				ID             string  `json:"id"`
				Title          string  `json:"title"`
				RelevanceScore float64 `json:"relevance_score"`
			}
			type routeResult struct {
				Destination routeDestination  `json:"destination"`
				Nodes       []routeNode       `json:"nodes"`
				Edges       []pathWitnessEdge `json:"edges"`
			}
			results := make([]routeResult, 0, len(candidates))
			for _, candidate := range candidates {
				var path []string
				var edges []pathWitnessEdge
				for current := candidate.id; current != ""; current = prev[current] {
					path = append(path, current)
					if edge, ok := prevEdge[current]; ok {
						edges = append(edges, edge)
					}
				}
				for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
					path[left], path[right] = path[right], path[left]
				}
				for left, right := 0, len(edges)-1; left < right; left, right = left+1, right-1 {
					edges[left], edges[right] = edges[right], edges[left]
				}
				nodes := make([]routeNode, len(path))
				for i, id := range path {
					nodes[i] = routeNode{ID: id, Title: byID[id].Title}
				}
				results = append(results, routeResult{
					Destination: routeDestination{ID: candidate.id, Title: byID[candidate.id].Title, RelevanceScore: candidate.relevance},
					Nodes:       nodes,
					Edges:       edges,
				})
			}
			enc := json.NewEncoder(outWriter(cmd))
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		},
	}
	cmd.Flags().StringVar(&focus, "focus", "", "Starting note ID")
	cmd.Flags().StringVar(&links, "links", "", "Comma-separated link types to follow in stored direction")
	cmd.Flags().StringVar(&search, "search", "", "Destination relevance query")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum destinations to return")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output typed route witnesses as JSON (required)")
	return cmd
}
