package cmd

import "sort"

const transcriptSummaryTypeLimit = 16
const transcriptSummaryTypeBytes = 64

type sessionSummary struct {
	Cost               sessionCostSummary      `json:"cost"`
	TopologyStatus     string                  `json:"topology_status"`
	Topology           *sessionTopologySummary `json:"topology"`
	AgentTypes         []sessionTypeCount      `json:"agent_types"`
	DistinctAgentTypes int                     `json:"distinct_agent_types"`
	OmittedTypeCount   int                     `json:"omitted_type_count"`
	OmittedAgentCount  int                     `json:"omitted_agent_count"`
	TypesTruncated     bool                    `json:"types_truncated"`
}

type sessionCostSummary struct {
	Status              string `json:"status"`
	TotalTokens         int    `json:"total_tokens"`
	InputTokens         int    `json:"input_tokens"`
	OutputTokens        int    `json:"output_tokens"`
	CacheReadTokens     int    `json:"cache_read_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	MeasuredAgents      int    `json:"measured_agents"`
	UnavailableAgents   int    `json:"unavailable_agents"`
}

type sessionTopologySummary struct {
	RootCount   int `json:"root_count"`
	EdgeCount   int `json:"edge_count"`
	MaxDepth    int `json:"max_depth"`
	MaxChildren int `json:"max_children"`
}

type sessionTypeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// summarizeSession projects the existing, un-repaired tree. Own costs are summed
// once; neither subtree rollups nor missing measurements establish exact totals.
func summarizeSession(agents []agent) *sessionSummary {
	s := &sessionSummary{AgentTypes: make([]sessionTypeCount, 0)}
	counts := make(map[string]int)
	for _, a := range agents {
		s.Cost.TotalTokens += a.Cost
		s.Cost.InputTokens += a.InputTokens
		s.Cost.OutputTokens += a.OutputTokens
		s.Cost.CacheReadTokens += a.CacheReadTokens
		s.Cost.CacheCreationTokens += a.CacheCreationTokens
		if a.CostStatus == "complete" {
			s.Cost.MeasuredAgents++
		} else {
			s.Cost.UnavailableAgents++
		}
		counts[a.Type]++
	}
	switch {
	case s.Cost.MeasuredAgents == 0:
		s.Cost.Status = "unavailable"
	case s.Cost.UnavailableAgents == 0:
		s.Cost.Status = "complete"
	default:
		s.Cost.Status = "partial"
	}
	s.Topology = summaryTopology(agents)
	s.TopologyStatus = "complete"
	if s.Topology == nil {
		s.TopologyStatus = "invalid"
	}
	s.DistinctAgentTypes = len(counts)
	types := make([]sessionTypeCount, 0, len(counts))
	for label, count := range counts {
		types = append(types, sessionTypeCount{Type: label, Count: count})
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].Count != types[j].Count {
			return types[i].Count > types[j].Count
		}
		return types[i].Type < types[j].Type
	})
	for _, entry := range types {
		if len(entry.Type) > transcriptSummaryTypeBytes || len(s.AgentTypes) >= transcriptSummaryTypeLimit {
			s.OmittedTypeCount++
			s.OmittedAgentCount += entry.Count
			continue
		}
		s.AgentTypes = append(s.AgentTypes, entry)
	}
	s.TypesTruncated = s.OmittedTypeCount > 0
	return s
}

// summaryTopology validates and measures a forest in O(n), without repairing it
// or depending on agent order. Unreachable nodes after root traversal imply a cycle.
func summaryTopology(agents []agent) *sessionTopologySummary {
	ids := make(map[string]int, len(agents))
	for i, a := range agents {
		if a.ID == "" {
			return nil
		}
		if _, exists := ids[a.ID]; exists {
			return nil
		}
		ids[a.ID] = i
	}
	children := make([][]int, len(agents))
	queue := make([]int, 0, len(agents))
	depth := make([]int, len(agents))
	result := &sessionTopologySummary{}
	for i, a := range agents {
		if a.ParentID == "" {
			queue = append(queue, i)
			result.RootCount++
			continue
		}
		parent, exists := ids[a.ParentID]
		if !exists {
			return nil
		}
		children[parent] = append(children[parent], i)
		result.EdgeCount++
	}
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		if len(children[i]) > result.MaxChildren {
			result.MaxChildren = len(children[i])
		}
		if depth[i] > result.MaxDepth {
			result.MaxDepth = depth[i]
		}
		for _, child := range children[i] {
			depth[child] = depth[i] + 1
			queue = append(queue, child)
		}
	}
	if len(queue) != len(agents) {
		return nil
	}
	return result
}
