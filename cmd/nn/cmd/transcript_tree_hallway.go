package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type treeHallwaySummary struct {
	TotalAgents      int            `json:"total_agents"`
	DirectChildren   int            `json:"direct_children"`
	NestedEdges      int            `json:"nested_edges"`
	Parentage        map[string]int `json:"parentage"`
	Lifecycle        map[string]int `json:"lifecycle"`
	UnknownParentage int            `json:"unknown_parentage"`
	UnknownLifecycle int            `json:"unknown_lifecycle"`
	OmittedAgents    int            `json:"omitted_agents"`
}

type treeChildPage struct {
	Snapshot      string         `json:"snapshot"`
	Parent        string         `json:"parent"`
	TotalChildren int            `json:"total_children"`
	Returned      int            `json:"returned"`
	Omitted       int            `json:"omitted"`
	NextCursor    string         `json:"next_cursor,omitempty"`
	Children      []treeChildRow `json:"children"`
}

type treeChildRow struct {
	ID                string `json:"id"`
	ParentID          string `json:"parent_id"`
	ParentageStatus   string `json:"parentage_status,omitempty"`
	Type              string `json:"type"`
	Description       string `json:"description,omitempty"`
	Started           string `json:"started"`
	Ended             string `json:"ended"`
	Cost              int    `json:"cost"`
	SubtreeCost       int    `json:"subtree_cost"`
	CostStatus        string `json:"cost_status"`
	SubtreeCostStatus string `json:"subtree_cost_status"`
	Status            string `json:"status"`
}

type treeChildCursor struct {
	Version          int `json:"version"`
	Snapshot, Parent string
	After            int `json:"after"`
}

func summarizeTreeHallway(agents []agent) treeHallwaySummary {
	s := treeHallwaySummary{TotalAgents: len(agents), Parentage: map[string]int{}, Lifecycle: map[string]int{}}
	for _, a := range agents {
		if a.ParentID == "ROOT" {
			s.DirectChildren++
		}
		if a.ParentID != "" && a.ParentID != "ROOT" {
			s.NestedEdges++
		}
		if a.ParentageStatus == "" {
			s.UnknownParentage++
		} else {
			s.Parentage[a.ParentageStatus]++
		}
		if a.Status == "" {
			s.UnknownLifecycle++
		} else {
			s.Lifecycle[a.Status]++
		}
	}
	return s
}

func buildTreeChildPage(agents []agent, parent string, limit int, cursor string) (treeChildPage, error) {
	known := parent == "ROOT"
	for _, a := range agents {
		if a.ID == parent {
			known = true
			break
		}
	}
	if !known {
		return treeChildPage{}, fmt.Errorf("tree: unknown parent %q", parent)
	}
	children := []agent{}
	for _, a := range agents {
		if a.ParentID == parent {
			children = append(children, a)
		}
	}
	encoded, _ := json.Marshal(struct {
		Version  int
		Parent   string
		Children []agent
	}{1, parent, children})
	h := sha256.Sum256(encoded)
	snapshot := hex.EncodeToString(h[:])
	after := -1
	if cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return treeChildPage{}, fmt.Errorf("tree: invalid cursor encoding: %w", err)
		}
		var c treeChildCursor
		if json.Unmarshal(b, &c) != nil || c.Version != 1 || c.Parent != parent || c.Snapshot != snapshot {
			return treeChildPage{}, fmt.Errorf("tree: stale or mismatched cursor")
		}
		after = c.After
		if after < 0 || after >= len(children) {
			return treeChildPage{}, fmt.Errorf("tree: invalid cursor position")
		}
	}
	start := after + 1
	end := len(children)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	rows := make([]treeChildRow, 0, end-start)
	for _, a := range children[start:end] {
		rows = append(rows, treeChildRow{ID: a.ID, ParentID: a.ParentID, ParentageStatus: a.ParentageStatus, Type: a.Type, Description: a.Description, Started: a.Started, Ended: a.Ended, Cost: a.Cost, SubtreeCost: a.SubtreeCost, CostStatus: a.CostStatus, SubtreeCostStatus: a.SubtreeCostStatus, Status: a.Status})
	}
	page := treeChildPage{Snapshot: snapshot, Parent: parent, TotalChildren: len(children), Children: rows}
	page.Returned = len(page.Children)
	page.Omitted = len(children) - end
	if end < len(children) {
		c, _ := json.Marshal(treeChildCursor{Version: 1, Snapshot: snapshot, Parent: parent, After: end - 1})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(c)
	}
	return page, nil
}
