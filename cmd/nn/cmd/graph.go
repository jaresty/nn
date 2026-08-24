package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

type graphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type graphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Annotation string `json:"annotation"`
	LinkType   string `json:"type,omitempty"`
}

type graphOutput struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

func newGraphCmd(state *rootState) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Output link relationships and graph queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph: %w", err)
			}

			g := graphOutput{
				Nodes: make([]graphNode, 0, len(notes)),
				Edges: []graphEdge{},
			}
			for _, n := range notes {
				g.Nodes = append(g.Nodes, graphNode{
					ID:    n.ID,
					Title: n.Title,
					Type:  string(n.Type),
				})
				for _, lnk := range n.Links {
					g.Edges = append(g.Edges, graphEdge{
						From:       n.ID,
						To:         lnk.TargetID,
						Annotation: lnk.Annotation,
						LinkType:   lnk.Type,
					})
				}
			}

			if jsonOut {
				enc := json.NewEncoder(outWriter(cmd))
				enc.SetIndent("", "  ")
				return enc.Encode(g)
			}
			for _, e := range g.Edges {
				if e.LinkType != "" {
					fmt.Fprintf(outWriter(cmd), "%s -> %s [%s] -- %s\n", e.From, e.To, e.LinkType, e.Annotation)
				} else {
					fmt.Fprintf(outWriter(cmd), "%s -> %s -- %s\n", e.From, e.To, e.Annotation)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.AddCommand(
		newGraphTopCmd(state),
		newGraphOrphansCmd(state),
		newGraphBridgesCmd(state),
		newGraphRoutesCmd(state),
		newGraphImpactCmd(state),
		newGraphShowCmd(state),
		newGraphExportCmd(state),
		newGraphApplyCmd(state),
	)
	return cmd
}

// ── nn graph top ──────────────────────────────────────────────────────────────

func newGraphTopCmd(state *rootState) *cobra.Command {
	var limit int
	var format string

	cmd := &cobra.Command{
		Use:   "top",
		Short: "Notes ranked by inbound link count",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph top: %w", err)
			}

			inbound := make(map[string]int)
			for _, n := range notes {
				for _, lnk := range n.Links {
					inbound[lnk.TargetID]++
				}
			}

			type entry struct {
				id, title string
				count     int
			}
			var entries []entry
			for _, n := range notes {
				if c := inbound[n.ID]; c > 0 {
					entries = append(entries, entry{n.ID, n.Title, c})
				}
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].count != entries[j].count {
					return entries[i].count > entries[j].count
				}
				return entries[i].id < entries[j].id
			})
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}

			w := outWriter(cmd)
			if format == "json" {
				type je struct {
					ID           string `json:"id"`
					Title        string `json:"title"`
					InboundCount int    `json:"inbound_count"`
				}
				out := make([]je, len(entries))
				for i, e := range entries {
					out[i] = je{e.id, e.title, e.count}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, e := range entries {
				fmt.Fprintf(w, "%s  %s  (%d inbound)\n", e.id, e.title, e.count)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum entries to show")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

// ── nn graph orphans ──────────────────────────────────────────────────────────

func newGraphOrphansCmd(state *rootState) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "orphans",
		Short: "Notes with no inbound or outbound links",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph orphans: %w", err)
			}

			targeted := make(map[string]bool)
			hasOutbound := make(map[string]bool)
			for _, n := range notes {
				for _, lnk := range n.Links {
					targeted[lnk.TargetID] = true
					hasOutbound[n.ID] = true
				}
			}

			// Global protocols (type=protocol with no outgoing governs links) are
			// not orphans — they intentionally have no governs targets.
			globalProtocol := make(map[string]bool)
			for _, n := range notes {
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
				if !hasGoverns {
					globalProtocol[n.ID] = true
				}
			}

			isOrphan := func(n *note.Note) bool {
				return !globalProtocol[n.ID] && !hasOutbound[n.ID] && !targeted[n.ID]
			}

			w := outWriter(cmd)
			if format == "json" {
				type je struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				}
				var out []je
				for _, n := range notes {
					if isOrphan(n) {
						out = append(out, je{n.ID, n.Title})
					}
				}
				if out == nil {
					out = []je{}
				}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			for _, n := range notes {
				if isOrphan(n) {
					fmt.Fprintf(w, "%s  %s\n", n.ID, n.Title)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

// ── nn graph bridges ──────────────────────────────────────────────────────────

func newGraphBridgesCmd(state *rootState) *cobra.Command {
	var limit int
	var format string
	var search string
	var exclude []string

	cmd := &cobra.Command{
		Use:   "bridges",
		Short: "Notes that connect otherwise-disconnected parts of the graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			searchMode := cmd.Flags().Changed("search")
			if format != "text" && format != "json" {
				return fmt.Errorf("graph bridges: unsupported format %q (want text or json)", format)
			}
			if len(exclude) > 0 && !searchMode {
				return fmt.Errorf("graph bridges: --exclude requires --search")
			}
			excludeSet := make(map[string]bool, len(exclude))
			for _, id := range exclude {
				id = strings.TrimSpace(id)
				if id == "" {
					return fmt.Errorf("graph bridges: --exclude requires a non-blank ID")
				}
				excludeSet[id] = true
			}
			if searchMode && strings.TrimSpace(search) == "" {
				return fmt.Errorf("graph bridges: --search requires a non-blank query")
			}
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph bridges: %w", err)
			}

			// inboundFrom[N] = set of notes that link TO N
			// outboundTo[N]  = set of notes that N links TO
			inboundFrom := make(map[string]map[string]bool)
			outboundTo := make(map[string]map[string]bool)
			for _, n := range notes {
				for _, lnk := range n.Links {
					if inboundFrom[lnk.TargetID] == nil {
						inboundFrom[lnk.TargetID] = make(map[string]bool)
					}
					inboundFrom[lnk.TargetID][n.ID] = true
					if outboundTo[n.ID] == nil {
						outboundTo[n.ID] = make(map[string]bool)
					}
					outboundTo[n.ID][lnk.TargetID] = true
				}
			}

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

			var candidates []bridgeCandidate
			for _, n := range notes {
				inCount := len(inboundFrom[n.ID])
				outCount := len(outboundTo[n.ID])
				if inCount == 0 || outCount == 0 {
					continue
				}
				candidate := bridgeCandidate{id: n.ID, title: n.Title, score: inCount * outCount}
				if searchMode {
					relevance := normalizedSearchScores[n.ID]
					if relevance <= 0 {
						continue
					}
					candidate.relevanceScore = &relevance
				}
				candidates = append(candidates, candidate)
			}
			sort.Slice(candidates, func(i, j int) bool {
				left, right := candidates[i], candidates[j]
				if searchMode && *left.relevanceScore != *right.relevanceScore {
					return *left.relevanceScore > *right.relevanceScore
				}
				if left.score != right.score {
					return left.score > right.score
				}
				return left.id < right.id
			})
			if len(excludeSet) > 0 {
				filtered := candidates[:0]
				for _, candidate := range candidates {
					if !excludeSet[candidate.id] {
						filtered = append(filtered, candidate)
					}
				}
				candidates = filtered
			}
			if limit > 0 && len(candidates) > limit {
				candidates = candidates[:limit]
			}

			records := buildBridgeRecords(candidates, notes)
			w := outWriter(cmd)
			if format == "json" {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(records)
			}
			writeBridgeRecordsText(w, records)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum entries to show")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	cmd.Flags().StringVar(&search, "search", "", "Return only full-graph bridges matching this query")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Exclude a bridge note ID from search results (repeatable; requires --search)")
	return cmd
}

// ── nn graph show ─────────────────────────────────────────────────────────────

func newGraphShowCmd(state *rootState) *cobra.Command {
	var focus string
	var depth int
	var format string
	var direction string
	var links string
	var statuses string
	var representation string
	var zones bool
	var bodies bool
	var presentationHints bool
	var colorMode string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Subgraph as structured data (LLM-facing)",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch format {
			case "text", "json", "mermaid":
			default:
				return fmt.Errorf("graph show: unsupported format %q (want text, json, or mermaid)", format)
			}
			switch colorMode {
			case "auto", "always", "never":
			default:
				return fmt.Errorf("graph show: unsupported --color %q (want auto, always, or never)", colorMode)
			}
			opts, err := newGraphShowTraversalOptions(direction, links, statuses, representation)
			if err != nil {
				return err
			}
			if focus == "" {
				for _, flag := range []string{"depth", "direction", "links", "status", "representation", "zones"} {
					if cmd.Flags().Changed(flag) {
						return fmt.Errorf("graph show: --%s requires --focus", flag)
					}
				}
			}
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph show: %w", err)
			}

			byID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
			}

			type summaryBudget struct {
				Tier    string   `json:"tier"`
				Length  string   `json:"length"`
				Include []string `json:"include"`
				Note    string   `json:"note,omitempty"`
			}
			type showNode struct {
				ID            string         `json:"id"`
				Title         string         `json:"title"`
				Type          string         `json:"type"`
				Tags          []string       `json:"tags"`
				Zone          string         `json:"zone,omitempty"`
				Body          string         `json:"body,omitempty"`
				OutDegree     int            `json:"out_degree"`
				InDegree      int            `json:"in_degree"`
				SummaryBudget *summaryBudget `json:"summary_budget,omitempty"`
			}
			type showEdge struct {
				From       string `json:"from"`
				To         string `json:"to"`
				Annotation string `json:"annotation,omitempty"`
				LinkType   string `json:"type,omitempty"`
			}

			var resultNodes []showNode
			var resultEdges []showEdge
			resultLevels := make(map[string]int)

			if focus != "" {
				root, ok := byID[focus]
				if !ok {
					return fmt.Errorf("graph show: note %q not found", focus)
				}
				entries := graphShowBFS(root, byID, depth, opts)
				visited := make(map[string]bool, len(entries))
				for _, e := range entries {
					visited[e.n.ID] = true
					resultLevels[e.n.ID] = e.level
					tags := e.n.Tags
					if tags == nil {
						tags = []string{}
					}
					resultNodes = append(resultNodes, showNode{ID: e.n.ID, Title: e.n.Title, Type: string(e.n.Type), Tags: tags})
				}
				sort.Slice(resultNodes, func(i, j int) bool { return resultNodes[i].ID < resultNodes[j].ID })
				for _, e := range entries {
					for _, lnk := range e.n.Links {
						if visited[lnk.TargetID] && opts.allowsLink(lnk.Type) {
							resultEdges = append(resultEdges, showEdge{e.n.ID, lnk.TargetID, lnk.Annotation, lnk.Type})
						}
					}
				}
				if zones {
					// Zone is determined by a node's direct edge to the ego (root).
					// ego -> N is dirOut; N -> ego is dirIn. Nodes with no direct
					// ego edge (and the ego itself) keep an empty zone.
					zoneByID := make(map[string]string)
					for _, lnk := range root.Links {
						if lnk.TargetID != root.ID {
							if z := zoneOf(lnk.Type, dirOut); z != zoneNone {
								zoneByID[lnk.TargetID] = string(z)
							}
						}
					}
					for _, n := range byID {
						if n.ID == root.ID {
							continue
						}
						for _, lnk := range n.Links {
							if lnk.TargetID == root.ID {
								if z := zoneOf(lnk.Type, dirIn); z != zoneNone {
									zoneByID[n.ID] = string(z)
								}
							}
						}
					}
					for i := range resultNodes {
						resultNodes[i].Zone = zoneByID[resultNodes[i].ID]
					}
				}
			} else {
				for _, n := range notes {
					tags := n.Tags
					if tags == nil {
						tags = []string{}
					}
					resultNodes = append(resultNodes, showNode{ID: n.ID, Title: n.Title, Type: string(n.Type), Tags: tags})
					for _, lnk := range n.Links {
						resultEdges = append(resultEdges, showEdge{n.ID, lnk.TargetID, lnk.Annotation, lnk.Type})
					}
				}
				sort.Slice(resultNodes, func(i, j int) bool { return resultNodes[i].ID < resultNodes[j].ID })
			}
			if bodies {
				for i := range resultNodes {
					if n, ok := byID[resultNodes[i].ID]; ok {
						resultNodes[i].Body = n.Body
					}
				}
			}
			// Degree is computed over the whole notebook, not the traversed
			// subgraph, so it reflects a node's true connectivity (hub vs leaf).
			inDegree := make(map[string]int, len(notes))
			for _, n := range notes {
				for _, lnk := range n.Links {
					inDegree[lnk.TargetID]++
				}
			}
			for i := range resultNodes {
				if n, ok := byID[resultNodes[i].ID]; ok {
					resultNodes[i].OutDegree = len(n.Links)
				}
				resultNodes[i].InDegree = inDegree[resultNodes[i].ID]
				if presentationHints {
					budget := summaryBudget{Tier: "leaf", Length: "one clause", Include: []string{"title", "type", "central claim"}}
					if resultNodes[i].InDegree >= 2 {
						budget = summaryBudget{Tier: "connected", Length: "one sentence", Include: []string{"central claim", "relationship to focus"}}
					}
					if resultNodes[i].InDegree >= 5 {
						budget = summaryBudget{Tier: "hub", Length: "2–3 sentences", Include: []string{"central claim", "why load-bearing", "relationship to focus"}}
						for _, tag := range resultNodes[i].Tags {
							if strings.EqualFold(tag, "daily") || strings.EqualFold(tag, "index") {
								budget = summaryBudget{Tier: "connected", Length: "one sentence", Include: []string{"central claim", "connector role"}, Note: "aggregation hub: summarize as connected"}
								break
							}
						}
					}
					resultNodes[i].SummaryBudget = &budget
				}
			}
			if resultEdges == nil {
				resultEdges = []showEdge{}
			}

			w := outWriter(cmd)
			if format == "mermaid" {
				normalizeCSV := func(value string) string {
					if strings.TrimSpace(value) == "" {
						return "-"
					}
					values := strings.Split(value, ",")
					for i := range values {
						values[i] = strings.TrimSpace(values[i])
					}
					sort.Strings(values)
					unique := values[:0]
					for _, value := range values {
						if len(unique) == 0 || unique[len(unique)-1] != value {
							unique = append(unique, value)
						}
					}
					return escapeMermaidLabel(strings.Join(unique, ","))
				}
				if focus == "" {
					fmt.Fprintln(w, "%% nn graph show scope=full")
				} else {
					metadataRepresentation := representation
					if metadataRepresentation == "" {
						metadataRepresentation = "-"
					}
					fmt.Fprintf(w, "%%%% nn graph show focus=%s depth=%d direction=%s links=%s status=%s representation=%s\n",
						escapeMermaidLabel(focus), depth, direction, normalizeCSV(links), normalizeCSV(statuses), escapeMermaidLabel(metadataRepresentation))
				}
				fmt.Fprintln(w, "flowchart TD")
				aliases := make(map[string]string, len(resultNodes))
				for i, n := range resultNodes {
					alias := fmt.Sprintf("n%d", i)
					aliases[n.ID] = alias
					fmt.Fprintf(w, "  %s[\"%s\"]\n", alias, escapeMermaidLabel(n.ID+"  "+n.Title))
				}
				missingSet := make(map[string]bool)
				for _, e := range resultEdges {
					if _, ok := aliases[e.From]; !ok {
						missingSet[e.From] = true
					}
					if _, ok := aliases[e.To]; !ok {
						missingSet[e.To] = true
					}
				}
				missingIDs := make([]string, 0, len(missingSet))
				for id := range missingSet {
					missingIDs = append(missingIDs, id)
				}
				sort.Strings(missingIDs)
				for _, id := range missingIDs {
					alias := fmt.Sprintf("n%d", len(aliases))
					aliases[id] = alias
					fmt.Fprintf(w, "  %s[\"%s\"]\n", alias, escapeMermaidLabel(id+"  [missing]"))
				}
				for _, e := range resultEdges {
					from, fromOK := aliases[e.From]
					to, toOK := aliases[e.To]
					if fromOK && toOK {
						label := e.LinkType
						if e.Annotation != "" {
							label += " — " + e.Annotation
						}
						fmt.Fprintf(w, "  %s -->|\"%s\"| %s\n", from, escapeMermaidLabel(label), to)
					}
				}
				return nil
			}
			if format == "json" {
				if focus != "" {
					out := struct {
						Center string     `json:"center"`
						Nodes  []showNode `json:"nodes"`
						Edges  []showEdge `json:"edges"`
					}{focus, resultNodes, resultEdges}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					return enc.Encode(out)
				}
				out := struct {
					Nodes []showNode `json:"nodes"`
					Edges []showEdge `json:"edges"`
				}{resultNodes, resultEdges}
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}
			// renderNodeBody prints a node's tags and body indented beneath its
			// title line when --bodies is set; a no-op otherwise. indent is the
			// leading whitespace of the title line the body hangs under.
			nodeByID := make(map[string]showNode, len(resultNodes))
			for _, n := range resultNodes {
				nodeByID[n.ID] = n
			}
			// degreeMarker renders a node's connectivity as "↑out ↓in" for
			// text views, so a hub is distinguishable from a leaf at a glance.
			degreeMarker := func(id string) string {
				n := nodeByID[id]
				return fmt.Sprintf("↑%d ↓%d", n.OutDegree, n.InDegree)
			}
			renderPresentationHint := func(id, indent string) {
				budget := nodeByID[id].SummaryBudget
				if budget == nil {
					return
				}
				fmt.Fprintf(w, "%s    relay budget: %s — include %s", indent, budget.Length, strings.Join(budget.Include, ", "))
				if budget.Note != "" {
					fmt.Fprintf(w, " — %s", budget.Note)
				}
				fmt.Fprintln(w)
			}
			renderNodeBody := func(id, indent string) {
				if !bodies {
					return
				}
				n := nodeByID[id]
				if len(n.Tags) > 0 {
					fmt.Fprintf(w, "%s    tags: %s\n", indent, strings.Join(n.Tags, ", "))
				}
				body := strings.TrimSpace(n.Body)
				if body == "" {
					return
				}
				for _, line := range strings.Split(body, "\n") {
					fmt.Fprintf(w, "%s    %s\n", indent, line)
				}
			}
			// Markers are colored-circle emoji, which survive markdown/relay
			// (unlike ANSI, which only renders in a raw terminal). --color always
			// forces them; auto emits them only to a TTY so default/piped output
			// stays clean and parseable; never suppresses them.
			useMarkers := colorMode == "always" || (colorMode == "auto" && isTTYFn())
			// zoneEmoji: the same zone→meaning mapping the viewer uses.
			zoneEmoji := map[string]string{"top": "🔵", "bottom": "🟢", "left": "🔴", "right": "🔷"}
			// familyEmoji maps a link type to its semantic family circle, kept
			// consistent with the zone families: tension = red, lateral = blue,
			// structural = blue-square.
			familyEmoji := func(linkType string) string {
				switch linkType {
				case "contradicts", "questions":
					return "🔴" // tension
				case "source-of", "requires", "related":
					return "🔵" // lateral
				case "refines", "extends", "grounded-by", "governs", "supports":
					return "🟦" // structural
				default:
					return ""
				}
			}
			// typeEmoji maps a note type to its color circle, matching the
			// viewer's scheme of coloring nodes by type.
			typeEmoji := func(t string) string {
				switch t {
				case "concept":
					return "🟢"
				case "model":
					return "🟣"
				case "observation":
					return "⚪"
				case "hypothesis":
					return "🟡"
				case "question":
					return "🔵"
				case "argument":
					return "🔴"
				case "protocol":
					return "🟦"
				default:
					return ""
				}
			}
			// prefix attaches an emoji marker before s when markers are active.
			prefix := func(emoji, s string) string {
				if !useMarkers || emoji == "" {
					return s
				}
				return emoji + " " + s
			}
			paint := func(zoneKey, s string) string {
				return prefix(zoneEmoji[zoneKey], s)
			}
			paintNode := func(id, s string) string {
				return prefix(typeEmoji(nodeByID[id].Type), s)
			}
			paintNodeType := func(noteType, s string) string {
				return prefix(typeEmoji(noteType), s)
			}
			paintEdge := func(linkType, s string) string {
				return prefix(familyEmoji(linkType), s)
			}
			if focus != "" && zones {
				// Zone-grouped view: nodes under directional headers.
				fmt.Fprintf(w, "%s  %s  %s\n", focus, paintNode(focus, byID[focus].Title), degreeMarker(focus))
				renderPresentationHint(focus, "")
				fmt.Fprintln(w)
				renderNodeBody(focus, "")
				// Zone → link-type key: names what each zone means, so text
				// output carries the same legend as the interactive viewer.
				fmt.Fprintln(w, "Key (zone = link types):")
				fmt.Fprintf(w, "  %s — answers-to: governs/supports (in), refines/extends/grounded-by (out)\n", paint("top", "TOP"))
				fmt.Fprintf(w, "  %s — tension: contradicts, questions\n", paint("left", "LEFT"))
				fmt.Fprintf(w, "  %s — lateral: source-of, requires\n", paint("right", "RIGHT"))
				fmt.Fprintf(w, "  %s — builds-on: governs/supports (out), refines/extends/grounded-by (in)\n", paint("bottom", "BOTTOM"))
				if useMarkers {
					// Marker scheme: node titles carry a note-type circle, edge
					// labels a link-family circle (same families as the zones).
					fmt.Fprintf(w, "  node marker = type: %s %s %s %s %s %s %s\n",
						paintNodeType("concept", "concept"), paintNodeType("model", "model"),
						paintNodeType("observation", "observation"), paintNodeType("hypothesis", "hypothesis"),
						paintNodeType("question", "question"), paintNodeType("argument", "argument"),
						paintNodeType("protocol", "protocol"))
					fmt.Fprintf(w, "  edge marker = link family: %s %s %s\n",
						paintEdge("contradicts", "tension"), paintEdge("source-of", "lateral"), paintEdge("refines", "structural"))
				}
				fmt.Fprintln(w)
				byZone := map[string][]showNode{}
				for _, n := range resultNodes {
					if n.ID == focus || n.Zone == "" {
						continue
					}
					byZone[n.Zone] = append(byZone[n.Zone], n)
				}
				order := []struct{ key, header string }{
					{"top", "TOP"},
					{"left", "LEFT"},
					{"right", "RIGHT"},
					{"bottom", "BOTTOM"},
				}
				for _, z := range order {
					group := byZone[z.key]
					if len(group) == 0 {
						continue
					}
					fmt.Fprintf(w, "%s\n", paint(z.key, z.header))
					for _, n := range group {
						fmt.Fprintf(w, "  %s  %s  %s\n", n.ID, paintNode(n.ID, n.Title), degreeMarker(n.ID))
						renderPresentationHint(n.ID, "  ")
						renderNodeBody(n.ID, "  ")
					}
					fmt.Fprintln(w)
				}
				return nil
			}
			if focus != "" {
				type treeEdge struct {
					neighbor string
					edge     showEdge
					arrow    string
				}
				adj := make(map[string][]treeEdge, len(resultEdges))
				for _, e := range resultEdges {
					if opts.direction != "incoming" {
						adj[e.From] = append(adj[e.From], treeEdge{e.To, e, "→"})
					}
					if opts.direction != "outgoing" {
						adj[e.To] = append(adj[e.To], treeEdge{e.From, e, "←"})
					}
				}
				rendered := make(map[string]bool)
				var renderTree func(id, indent string)
				renderTree = func(id, indent string) {
					n, ok := byID[id]
					if !ok {
						return
					}
					if indent == "" {
						fmt.Fprintf(w, "%s  %s  %s\n", id, paintNode(id, n.Title), degreeMarker(id))
						renderPresentationHint(id, "")
						renderNodeBody(id, "")
					}
					rendered[id] = true
					for _, tree := range adj[id] {
						if rendered[tree.neighbor] || resultLevels[tree.neighbor] != resultLevels[id]+1 {
							continue
						}
						e := tree.edge
						target, ok := byID[tree.neighbor]
						if !ok {
							continue
						}
						linkLabel := e.LinkType
						if linkLabel == "" {
							linkLabel = "link"
						}
						if e.Annotation != "" {
							fmt.Fprintf(w, "%s  %s [%s] %s  %s  %s — %s\n", indent, tree.arrow, paintEdge(e.LinkType, linkLabel), tree.neighbor, paintNode(tree.neighbor, target.Title), degreeMarker(tree.neighbor), e.Annotation)
						} else {
							fmt.Fprintf(w, "%s  %s [%s] %s  %s  %s\n", indent, tree.arrow, paintEdge(e.LinkType, linkLabel), tree.neighbor, paintNode(tree.neighbor, target.Title), degreeMarker(tree.neighbor))
						}
						renderPresentationHint(tree.neighbor, indent+"  ")
						renderNodeBody(tree.neighbor, indent+"  ")
						renderTree(tree.neighbor, indent+"  ")
					}
				}
				renderTree(focus, "")
			} else {
				for _, n := range resultNodes {
					fmt.Fprintf(w, "%s  %s  %s\n", n.ID, paintNode(n.ID, n.Title), degreeMarker(n.ID))
					renderPresentationHint(n.ID, "")
					renderNodeBody(n.ID, "")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&focus, "focus", "", "Center note ID for ego-graph")
	cmd.Flags().IntVar(&depth, "depth", 2, "BFS depth from focus note")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json, or mermaid")
	cmd.Flags().StringVar(&direction, "direction", "outgoing", "Traversal direction: outgoing, incoming, or both")
	cmd.Flags().StringVar(&links, "links", "", "Comma-separated link types to traverse")
	cmd.Flags().StringVar(&statuses, "status", "", "Comma-separated note statuses to traverse")
	cmd.Flags().StringVar(&representation, "representation", "", "Representation required for traversed notes")
	cmd.Flags().BoolVar(&zones, "zones", false, "Annotate each node with its directional zone (top/bottom/left/right) relative to the focus")
	cmd.Flags().BoolVar(&bodies, "bodies", false, "Include each node's body (and tags in text output) so the graph can be read as well as navigated")
	cmd.Flags().BoolVar(&presentationHints, "presentation-hints", false, "Annotate nodes with degree-based LLM relay budgets without truncating bodies")
	cmd.Flags().StringVar(&colorMode, "color", "auto", "Colorize zoned text output by zone: auto (color only to a TTY), always, or never")
	return cmd
}

// ── nn graph export ───────────────────────────────────────────────────────────

func newGraphExportCmd(state *rootState) *cobra.Command {
	var format string
	var focus string
	var depth int
	var open bool
	var output string
	var serve bool
	var port int
	var verbose bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the graph as DOT or SVG",
		RunE: func(cmd *cobra.Command, args []string) error {
			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph export: %w", err)
			}

			byID := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				byID[n.ID] = n
			}

			var enodes []dotNode
			var eedges []dotEdge

			if focus != "" {
				root, ok := byID[focus]
				if !ok {
					return fmt.Errorf("graph export: note %q not found", focus)
				}
				entries := bfsDepth(root, byID, depth)
				visited := make(map[string]bool, len(entries))
				for _, e := range entries {
					visited[e.n.ID] = true
					enodes = append(enodes, dotNode{e.n.ID, e.n.Title})
				}
				for _, e := range entries {
					for _, lnk := range e.n.Links {
						if visited[lnk.TargetID] {
							eedges = append(eedges, dotEdge{e.n.ID, lnk.TargetID, lnk.Annotation})
						}
					}
				}
			} else {
				for _, n := range notes {
					enodes = append(enodes, dotNode{n.ID, n.Title})
					for _, lnk := range n.Links {
						eedges = append(eedges, dotEdge{n.ID, lnk.TargetID, lnk.Annotation})
					}
				}
			}

			dot := buildDOT(enodes, eedges)
			w := outWriter(cmd)

			switch format {
			case "dot":
				fmt.Fprint(w, dot)
				return nil
			case "svg":
				svg, err := dotToSVG(dot)
				if err != nil {
					return fmt.Errorf("graph export svg: %w (is graphviz installed?)", err)
				}
				if open {
					return openFile([]byte(svg), "")
				}
				fmt.Fprint(w, svg)
				return nil
			case "html":
				htmlBytes, err := buildHTML(enodes, eedges, byID, serve)
				if err != nil {
					return fmt.Errorf("graph export html: %w", err)
				}
				if serve {
					return startServeMode(cmd.Context(), state.backend, state.notebookDir, htmlBytes, port, open, verbose)
				}
				if output != "" {
					if err := os.WriteFile(output, htmlBytes, 0o644); err != nil {
						return fmt.Errorf("graph export html: write %s: %w", output, err)
					}
					if open {
						return openFile(nil, output) // open existing file by path
					}
					return nil
				}
				if open {
					return openFile(htmlBytes, ".html")
				}
				fmt.Fprint(w, string(htmlBytes))
				return nil
			default:
				return fmt.Errorf("graph export: unknown format %q (use dot, svg, or html)", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "dot", "Output format: dot, svg, or html")
	cmd.Flags().StringVar(&focus, "focus", "", "Center note ID for ego-graph")
	cmd.Flags().IntVar(&depth, "depth", 2, "BFS depth from focus note (when --focus is set)")
	cmd.Flags().BoolVar(&open, "open", false, "Open output in default viewer (or browser when --serve)")
	cmd.Flags().StringVar(&output, "output", "", "Write output to file path (html only)")
	cmd.Flags().BoolVar(&serve, "serve", false, "Serve the HTML graph interactively on a local HTTP server (html only)")
	cmd.Flags().IntVar(&port, "port", 7734, "Port for --serve mode")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Log server activity to stderr (--serve mode)")
	return cmd
}

// ── DOT / file helpers ────────────────────────────────────────────────────────

type dotNode struct{ id, title string }
type dotEdge struct{ from, to, annotation string }

func buildDOT(nodes []dotNode, edges []dotEdge) string {
	var sb strings.Builder
	sb.WriteString("digraph nn {\n")
	sb.WriteString("  graph [rankdir=LR];\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("  %q [label=%q];\n", n.id, n.title))
	}
	for _, e := range edges {
		if e.annotation != "" {
			sb.WriteString(fmt.Sprintf("  %q -> %q [label=%q];\n", e.from, e.to, e.annotation))
		} else {
			sb.WriteString(fmt.Sprintf("  %q -> %q;\n", e.from, e.to))
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

func buildHTML(nodes []dotNode, edges []dotEdge, notesByID map[string]*note.Note, serveMode bool) ([]byte, error) {
	type jsonNode struct {
		ID    string   `json:"id"`
		Title string   `json:"title"`
		Type  string   `json:"type"`
		Tags  []string `json:"tags"`
		Body  string   `json:"body"`
	}
	type jsonEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	nodeSet := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n.id] = true
	}
	jnodes := make([]jsonNode, len(nodes))
	for i, n := range nodes {
		tags := []string{}
		body := ""
		if nn, ok := notesByID[n.id]; ok {
			if nn.Tags != nil {
				tags = nn.Tags
			}
			body = nn.Body
		}
		nodeType := ""
		if nn, ok := notesByID[n.id]; ok {
			nodeType = string(nn.Type)
		}
		jnodes[i] = jsonNode{n.id, n.title, nodeType, tags, body}
	}
	var jedges []jsonEdge
	for _, e := range edges {
		if nodeSet[e.from] && nodeSet[e.to] {
			jedges = append(jedges, jsonEdge{e.from, e.to})
		}
	}
	if jedges == nil {
		jedges = []jsonEdge{}
	}
	graphData, err := json.Marshal(struct {
		Nodes []jsonNode `json:"nodes"`
		Edges []jsonEdge `json:"edges"`
	}{jnodes, jedges})
	if err != nil {
		return nil, err
	}

	d3Bundle, err := graphTemplateFS.ReadFile("templates/d3.min.js")
	if err != nil {
		return nil, fmt.Errorf("read d3 bundle: %w", err)
	}
	tmplSrc, err := graphTemplateFS.ReadFile("templates/graph.html")
	if err != nil {
		return nil, fmt.Errorf("read graph template: %w", err)
	}

	tmpl, err := template.New("graph").Delims("{{", "}}").Parse(string(tmplSrc))
	if err != nil {
		return nil, fmt.Errorf("parse graph template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		D3        template.JS
		GraphJSON template.JS
		ServeMode bool
	}{
		D3:        template.JS(d3Bundle),
		GraphJSON: template.JS(graphData),
		ServeMode: serveMode,
	}); err != nil {
		return nil, fmt.Errorf("execute graph template: %w", err)
	}
	return buf.Bytes(), nil
}

func dotToSVG(dot string) (string, error) {
	c := exec.Command("dot", "-Tsvg")
	c.Stdin = strings.NewReader(dot)
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// openFile opens data in the default viewer. If path is non-empty, it is opened
// directly (data is ignored). Otherwise data is written to a temp file with ext.
func openFile(data []byte, path string) error {
	if path == "" {
		f, err := os.CreateTemp("", "nn-graph-*")
		if err != nil {
			return err
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			return err
		}
		f.Close()
		path = f.Name()
	}
	openCmd := "xdg-open"
	if runtime.GOOS == "darwin" {
		openCmd = "open"
	}
	return exec.Command(openCmd, path).Start()
}
