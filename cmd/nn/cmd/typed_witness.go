package cmd

import (
	"sort"
	"strconv"
	"strings"
)

const (
	// Public command output is deliberately small. The larger cubic portfolio is
	// retained at each shortest-path DAG node so alternatives survive later
	// convergence without allowing the number of partial paths to grow
	// exponentially. Three output slots × three diversity alternatives × three
	// lexical fallbacks gives a fixed 27-candidate reconstruction bound per node.
	typedWitnessOutputLimit    = 3
	typedWitnessPortfolioLimit = typedWitnessOutputLimit * typedWitnessOutputLimit * typedWitnessOutputLimit
)

// typedWitnessNode and typedWitnessEdge are the canonical JSON model shared by
// typed path, routes, and impact output.
type typedWitnessNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type typedWitnessEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Annotation string `json:"annotation"`
}

type typedWitness struct {
	Nodes []typedWitnessNode `json:"nodes"`
	Edges []typedWitnessEdge `json:"edges"`
}

// typedTraversalEdge separates traversal direction from stored edge direction.
// For outgoing traversal next is edge.To. Incoming impact traversal instead
// sets next to edge.From while leaving edge unchanged for truthful output.
type typedTraversalEdge struct {
	next string
	edge typedWitnessEdge
}

type typedPredecessor struct {
	from string
	edge typedWitnessEdge
}

type typedWitnessSearch struct {
	depthByID    map[string]int
	predecessors map[string][]typedPredecessor
	portfolios   map[string][]typedWitness
}

// findShortestTypedWitnesses performs two deliberately separate phases:
//
//  1. BFS records every transition whose endpoint has the same shortest depth.
//     Those transitions form an acyclic predecessor DAG even when the stored
//     graph contains cycles. maxDepth == 0 means unbounded traversal.
//  2. Nodes are processed in depth order. Each predecessor's bounded portfolio
//     is extended by one transition, then reduced to at most 27 candidates.
//     Extension cannot change a path's first hop or collapse two distinct edge
//     type sequences, so the portfolio keeps the categories needed by the final
//     three-result greedy selection while preventing exponential reconstruction
//     in layered or heavily convergent DAGs.
//
// Selection is deterministic and follows the command contract: take the
// lexically earliest representative of each new first-hop node first, then take
// candidates with new edge-type sequences, then fill remaining slots by the
// lexical full-path key. Selected witnesses are finally emitted in lexical path
// order. The full-path key uses traversal node IDs followed by stored edge
// from/to/type/annotation fields; titles never affect structural selection.
func findShortestTypedWitnesses(source string, titles map[string]string, adjacency map[string][]typedTraversalEdge, maxDepth int) typedWitnessSearch {
	sortedAdjacency := make(map[string][]typedTraversalEdge, len(adjacency))
	for id, edges := range adjacency {
		sortedAdjacency[id] = append([]typedTraversalEdge(nil), edges...)
		sort.Slice(sortedAdjacency[id], func(i, j int) bool {
			return compareTypedTraversalEdges(sortedAdjacency[id][i], sortedAdjacency[id][j]) < 0
		})
	}

	search := typedWitnessSearch{
		depthByID:    map[string]int{source: 0},
		predecessors: make(map[string][]typedPredecessor),
		portfolios:   make(map[string][]typedWitness),
	}
	queue := []string{source}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		currentDepth := search.depthByID[current]
		if maxDepth > 0 && currentDepth >= maxDepth {
			continue
		}
		for _, transition := range sortedAdjacency[current] {
			nextDepth := currentDepth + 1
			knownDepth, seen := search.depthByID[transition.next]
			if !seen {
				search.depthByID[transition.next] = nextDepth
				search.predecessors[transition.next] = []typedPredecessor{{from: current, edge: transition.edge}}
				queue = append(queue, transition.next)
				continue
			}
			if knownDepth == nextDepth {
				search.predecessors[transition.next] = append(search.predecessors[transition.next], typedPredecessor{
					from: current,
					edge: transition.edge,
				})
			}
		}
	}

	search.portfolios[source] = []typedWitness{{
		Nodes: []typedWitnessNode{{ID: source, Title: titles[source]}},
		Edges: []typedWitnessEdge{},
	}}

	maxFoundDepth := 0
	nodesByDepth := make(map[int][]string)
	for id, depth := range search.depthByID {
		if depth == 0 {
			continue
		}
		nodesByDepth[depth] = append(nodesByDepth[depth], id)
		if depth > maxFoundDepth {
			maxFoundDepth = depth
		}
	}
	for depth := 1; depth <= maxFoundDepth; depth++ {
		sort.Strings(nodesByDepth[depth])
		for _, id := range nodesByDepth[depth] {
			predecessors := search.predecessors[id]
			sort.Slice(predecessors, func(i, j int) bool {
				if predecessors[i].from != predecessors[j].from {
					return predecessors[i].from < predecessors[j].from
				}
				return compareTypedWitnessEdges(predecessors[i].edge, predecessors[j].edge) < 0
			})
			search.predecessors[id] = predecessors

			// Each predecessor contributes at most one bounded portfolio. The
			// temporary union is therefore linear in indegree, never in the number
			// of represented shortest paths; avoid multiplication-based capacity
			// arithmetic so even adversarially large graphs cannot overflow it.
			var candidates []typedWitness
			for _, predecessor := range predecessors {
				for _, prefix := range search.portfolios[predecessor.from] {
					candidates = append(candidates, extendTypedWitness(prefix, id, titles[id], predecessor.edge))
				}
			}
			search.portfolios[id] = selectDiverseTypedWitnesses(candidates, typedWitnessPortfolioLimit)
		}
	}

	return search
}

func (search typedWitnessSearch) witnessesTo(id string) []typedWitness {
	return selectDiverseTypedWitnesses(search.portfolios[id], typedWitnessOutputLimit)
}

func extendTypedWitness(prefix typedWitness, id, title string, edge typedWitnessEdge) typedWitness {
	nodes := make([]typedWitnessNode, len(prefix.Nodes)+1)
	copy(nodes, prefix.Nodes)
	nodes[len(prefix.Nodes)] = typedWitnessNode{ID: id, Title: title}
	edges := make([]typedWitnessEdge, len(prefix.Edges)+1)
	copy(edges, prefix.Edges)
	edges[len(prefix.Edges)] = edge
	return typedWitness{Nodes: nodes, Edges: edges}
}

type keyedTypedWitness struct {
	witness      typedWitness
	firstHopKey  string
	typeSequence string
}

func selectDiverseTypedWitnesses(candidates []typedWitness, limit int) []typedWitness {
	if limit <= 0 || len(candidates) == 0 {
		return []typedWitness{}
	}

	keyed := make([]keyedTypedWitness, 0, len(candidates))
	seenPaths := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		fullPathKey := typedWitnessFullPathKey(candidate)
		if seenPaths[fullPathKey] {
			continue
		}
		seenPaths[fullPathKey] = true
		firstHop := ""
		if len(candidate.Nodes) > 1 {
			firstHop = candidate.Nodes[1].ID
		}
		keyed = append(keyed, keyedTypedWitness{
			witness:      candidate,
			firstHopKey:  firstHop,
			typeSequence: typedWitnessTypeSequenceKey(candidate),
		})
	}
	sort.Slice(keyed, func(i, j int) bool {
		return compareTypedWitnesses(keyed[i].witness, keyed[j].witness) < 0
	})

	selected := make([]keyedTypedWitness, 0, min(limit, len(keyed)))
	chosen := make([]bool, len(keyed))
	firstHops := make(map[string]bool)
	typeSequences := make(map[string]bool)
	choose := func(i int) {
		chosen[i] = true
		selected = append(selected, keyed[i])
		firstHops[keyed[i].firstHopKey] = true
		typeSequences[keyed[i].typeSequence] = true
	}

	// Priority 1: one lexical representative for every new traversal first hop.
	for i := range keyed {
		if len(selected) == limit {
			break
		}
		if !firstHops[keyed[i].firstHopKey] {
			choose(i)
		}
	}
	// Priority 2: among candidates left after first-hop selection, introduce new
	// complete edge-type sequences before allowing duplicate sequences.
	for i := range keyed {
		if len(selected) == limit {
			break
		}
		if !chosen[i] && !typeSequences[keyed[i].typeSequence] {
			choose(i)
		}
	}
	// Priority 3: deterministic lexical fallback when diversity is exhausted.
	for i := range keyed {
		if len(selected) == limit {
			break
		}
		if !chosen[i] {
			choose(i)
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return compareTypedWitnesses(selected[i].witness, selected[j].witness) < 0
	})
	result := make([]typedWitness, len(selected))
	for i := range selected {
		result[i] = selected[i].witness
	}
	return result
}

// typedWitnessFullPathKey is an unambiguous identity key used only for exact
// deduplication. Lexical ordering uses compareTypedWitnesses so length prefixes
// cannot distort ordinary string order.
func typedWitnessFullPathKey(witness typedWitness) string {
	var key strings.Builder
	for _, node := range witness.Nodes {
		appendTypedWitnessKeyPart(&key, node.ID)
	}
	key.WriteByte('|')
	for _, edge := range witness.Edges {
		appendTypedWitnessKeyPart(&key, edge.From)
		appendTypedWitnessKeyPart(&key, edge.To)
		appendTypedWitnessKeyPart(&key, edge.Type)
		appendTypedWitnessKeyPart(&key, edge.Annotation)
	}
	return key.String()
}

func typedWitnessTypeSequenceKey(witness typedWitness) string {
	var key strings.Builder
	for _, edge := range witness.Edges {
		appendTypedWitnessKeyPart(&key, edge.Type)
	}
	return key.String()
}

func appendTypedWitnessKeyPart(key *strings.Builder, value string) {
	key.WriteString(strconv.Itoa(len(value)))
	key.WriteByte(':')
	key.WriteString(value)
	key.WriteByte(';')
}

func compareTypedWitnesses(a, b typedWitness) int {
	for i := 0; i < min(len(a.Nodes), len(b.Nodes)); i++ {
		if a.Nodes[i].ID != b.Nodes[i].ID {
			return strings.Compare(a.Nodes[i].ID, b.Nodes[i].ID)
		}
	}
	if len(a.Nodes) != len(b.Nodes) {
		if len(a.Nodes) < len(b.Nodes) {
			return -1
		}
		return 1
	}
	for i := 0; i < min(len(a.Edges), len(b.Edges)); i++ {
		if comparison := compareTypedWitnessEdges(a.Edges[i], b.Edges[i]); comparison != 0 {
			return comparison
		}
	}
	if len(a.Edges) < len(b.Edges) {
		return -1
	}
	if len(a.Edges) > len(b.Edges) {
		return 1
	}
	return 0
}

func compareTypedTraversalEdges(a, b typedTraversalEdge) int {
	if a.next != b.next {
		return strings.Compare(a.next, b.next)
	}
	return compareTypedWitnessEdges(a.edge, b.edge)
}

func compareTypedWitnessEdges(a, b typedWitnessEdge) int {
	if a.From != b.From {
		return strings.Compare(a.From, b.From)
	}
	if a.To != b.To {
		return strings.Compare(a.To, b.To)
	}
	if a.Type != b.Type {
		return strings.Compare(a.Type, b.Type)
	}
	return strings.Compare(a.Annotation, b.Annotation)
}
