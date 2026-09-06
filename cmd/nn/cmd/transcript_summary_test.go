package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSessionSummaryCostAuthority(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		agents                []agent
		status                string
		measured, unavailable int
	}{
		{"empty", nil, "unavailable", 0, 0},
		{"measured zero", []agent{{ID: "ROOT", CostStatus: "complete"}}, "complete", 1, 0},
		{"unknown zero", []agent{{ID: "ROOT", CostStatus: "unavailable"}}, "unavailable", 0, 1},
		{"missing authority", []agent{{ID: "ROOT"}}, "unavailable", 0, 1},
		{"unrecognized authority", []agent{{ID: "ROOT", CostStatus: "future"}}, "unavailable", 0, 1},
		{"partial zero", []agent{{ID: "ROOT", CostStatus: "complete"}, {ID: "child", ParentID: "ROOT"}}, "partial", 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := summarizeSession(tc.agents).Cost
			if c.Status != tc.status || c.MeasuredAgents != tc.measured || c.UnavailableAgents != tc.unavailable {
				t.Fatalf("ASSERT_SUMMARY_AUTHORITY: %+v", c)
			}
		})
	}
	agents := []agent{{ID: "ROOT", Cost: 10, SubtreeCost: 1000, InputTokens: 1, OutputTokens: 2, CacheReadTokens: 3, CacheCreationTokens: 4, CostStatus: "complete"}, {ID: "child", ParentID: "ROOT", Cost: 100, SubtreeCost: 2000, InputTokens: 10, OutputTokens: 20, CacheReadTokens: 30, CacheCreationTokens: 40, CostStatus: "complete"}}
	c := summarizeSession(agents).Cost
	if c.TotalTokens != 110 || c.InputTokens != 11 || c.OutputTokens != 22 || c.CacheReadTokens != 33 || c.CacheCreationTokens != 44 {
		t.Fatalf("ASSERT_SUMMARY_OWN_COMPONENTS: %+v", c)
	}
}

func TestSessionSummaryForest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		agents []agent
		want   *sessionTopologySummary
	}{
		{"empty", nil, &sessionTopologySummary{}},
		{"root", []agent{{ID: "ROOT"}}, &sessionTopologySummary{RootCount: 1}},
		{"unordered forest", []agent{{ID: "leaf", ParentID: "child"}, {ID: "child", ParentID: "ROOT"}, {ID: "other"}, {ID: "sibling", ParentID: "ROOT"}, {ID: "ROOT"}}, &sessionTopologySummary{RootCount: 2, EdgeCount: 3, MaxDepth: 2, MaxChildren: 2}},
		{"missing parent", []agent{{ID: "ROOT"}, {ID: "a", ParentID: "absent"}}, nil},
		{"self cycle", []agent{{ID: "a", ParentID: "a"}}, nil},
		{"disconnected cycle", []agent{{ID: "ROOT"}, {ID: "a", ParentID: "b"}, {ID: "b", ParentID: "a"}}, nil},
		{"duplicate", []agent{{ID: "ROOT"}, {ID: "ROOT"}}, nil},
		{"empty ID", []agent{{ID: ""}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]agent(nil), tc.agents...)
			s := summarizeSession(tc.agents)
			status := "complete"
			if tc.want == nil {
				status = "invalid"
			}
			if !reflect.DeepEqual(s.Topology, tc.want) || s.TopologyStatus != status {
				t.Fatalf("ASSERT_SUMMARY_FOREST: %+v status=%s", s.Topology, s.TopologyStatus)
			}
			if !reflect.DeepEqual(before, tc.agents) {
				t.Fatal("ASSERT_SUMMARY_NO_REPAIR: input changed")
			}
		})
	}
}

func TestSessionSummaryTypesBound(t *testing.T) {
	agents := []agent{}
	add := func(label string, n int) {
		for i := 0; i < n; i++ {
			agents = append(agents, agent{ID: fmt.Sprint(len(agents)), Type: label, CostStatus: "complete"})
		}
	}
	// Exact 64-byte boundary survives; 65-byte and oversized multibyte labels do not.
	add(strings.Repeat("界", 21)+"x", 3)
	add(strings.Repeat("x", 65), 4)
	add(strings.Repeat("界", 22), 5)
	add("", 2)
	for i := 0; i < 17; i++ {
		add(fmt.Sprintf("type-%02d", i), 1)
	}
	s := summarizeSession(agents)
	if len(s.AgentTypes) != 16 || s.DistinctAgentTypes != 21 || s.OmittedTypeCount != 5 || s.OmittedAgentCount != 12 || !s.TypesTruncated {
		t.Fatalf("ASSERT_SUMMARY_TYPES_BOUND: %+v", s)
	}
	if s.AgentTypes[0].Count != 3 || len(s.AgentTypes[0].Type) != 64 || s.AgentTypes[1].Type != "" || s.AgentTypes[2].Type != "type-00" || s.AgentTypes[15].Type != "type-13" {
		t.Fatalf("ASSERT_SUMMARY_TYPES_ORDER: %+v", s.AgentTypes)
	}
	for i, j := 0, len(agents)-1; i < j; i, j = i+1, j-1 {
		agents[i], agents[j] = agents[j], agents[i]
	}
	if !reflect.DeepEqual(s, summarizeSession(agents)) {
		t.Fatal("ASSERT_SUMMARY_TYPES_ORDER: order-dependent")
	}
	// Worst-case six-byte JSON escapes for every retained label byte.
	agents = nil
	for i := 0; i < 16; i++ {
		add(strings.Repeat(string(rune(i)), 64), 1)
	}
	data, err := json.Marshal(summarizeSession(agents))
	if err != nil || len(data) > 8192 {
		t.Fatalf("ASSERT_SUMMARY_BYTE_BOUND: bytes=%d err=%v", len(data), err)
	}
	exact := summarizeSession(agents)
	if len(exact.AgentTypes) != 16 || exact.TypesTruncated || exact.OmittedTypeCount != 0 || exact.OmittedAgentCount != 0 {
		t.Fatalf("ASSERT_SUMMARY_TYPES_BOUND: exact cap %+v", exact)
	}
	add("extra", 1)
	if s := summarizeSession(agents); len(s.AgentTypes) != 16 || !s.TypesTruncated || s.OmittedTypeCount != 1 || s.OmittedAgentCount != 1 {
		t.Fatalf("ASSERT_SUMMARY_TYPES_BOUND: over cap %+v", s)
	}
}
