package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const graphShowPageMaxBytes = 48_000

type graphShowSummaryBudget struct {
	Tier    string   `json:"tier"`
	Length  string   `json:"length"`
	Include []string `json:"include"`
	Note    string   `json:"note,omitempty"`
}

type graphShowNode struct {
	ID            string                  `json:"id"`
	Title         string                  `json:"title"`
	Type          string                  `json:"type"`
	Tags          []string                `json:"tags"`
	Zone          string                  `json:"zone,omitempty"`
	Body          string                  `json:"body,omitempty"`
	OutDegree     int                     `json:"out_degree"`
	InDegree      int                     `json:"in_degree"`
	SummaryBudget *graphShowSummaryBudget `json:"summary_budget,omitempty"`
}

type graphShowEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Annotation string `json:"annotation,omitempty"`
	LinkType   string `json:"type,omitempty"`
}

type graphShowPage struct {
	Snapshot string          `json:"snapshot"`
	Page     int             `json:"page"`
	Pages    int             `json:"pages"`
	NextPage int             `json:"next_page"`
	Center   string          `json:"center,omitempty"`
	Nodes    []graphShowNode `json:"nodes"`
	Edges    []graphShowEdge `json:"edges"`
}

type graphShowRecord struct {
	node *graphShowNode
	edge *graphShowEdge
}

func writeGraphShowPage(cmdOut interface{ Write([]byte) (int, error) }, center, requestKey, suppliedSnapshot string, page int, nodes []graphShowNode, edges []graphShowEdge) error {
	canonical, err := json.Marshal(struct {
		Request string          `json:"request"`
		Center  string          `json:"center,omitempty"`
		Nodes   []graphShowNode `json:"nodes"`
		Edges   []graphShowEdge `json:"edges"`
	}{requestKey, center, nodes, edges})
	if err != nil {
		return fmt.Errorf("graph show: encode snapshot: %w", err)
	}
	sum := sha256.Sum256(canonical)
	snapshot := hex.EncodeToString(sum[:])
	if suppliedSnapshot != "" && suppliedSnapshot != snapshot {
		return fmt.Errorf("graph show: stale or mismatched --snapshot; notebook or traversal changed")
	}

	records := make([]graphShowRecord, 0, len(nodes)+len(edges))
	for i := range nodes {
		records = append(records, graphShowRecord{node: &nodes[i]})
	}
	for i := range edges {
		records = append(records, graphShowRecord{edge: &edges[i]})
	}
	pages := [][]graphShowRecord{{}}
	for _, record := range records {
		candidate := append(append([]graphShowRecord(nil), pages[len(pages)-1]...), record)
		if graphShowEncodedPageSize(snapshot, center, 999999, 999999, candidate) <= graphShowPageMaxBytes {
			pages[len(pages)-1] = candidate
			continue
		}
		if len(pages[len(pages)-1]) == 0 {
			return fmt.Errorf("graph show: one topology record exceeds the %d-byte page limit", graphShowPageMaxBytes)
		}
		pages = append(pages, []graphShowRecord{record})
		if graphShowEncodedPageSize(snapshot, center, 999999, 999999, pages[len(pages)-1]) > graphShowPageMaxBytes {
			return fmt.Errorf("graph show: one topology record exceeds the %d-byte page limit", graphShowPageMaxBytes)
		}
	}
	if page > len(pages) {
		return fmt.Errorf("graph show: --page %d out of range (pages: %d)", page, len(pages))
	}
	response := graphShowPageFromRecords(snapshot, center, page, len(pages), pages[page-1])
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("graph show: encode page: %w", err)
	}
	if len(encoded)+1 > graphShowPageMaxBytes {
		return fmt.Errorf("graph show: encoded page is %d bytes (limit %d)", len(encoded)+1, graphShowPageMaxBytes)
	}
	if _, err := cmdOut.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("graph show: write page: %w", err)
	}
	return nil
}

func graphShowPageFromRecords(snapshot, center string, page, pages int, records []graphShowRecord) graphShowPage {
	result := graphShowPage{Snapshot: snapshot, Page: page, Pages: pages, Center: center, Nodes: []graphShowNode{}, Edges: []graphShowEdge{}}
	if page < pages {
		result.NextPage = page + 1
	}
	for _, record := range records {
		if record.node != nil {
			result.Nodes = append(result.Nodes, *record.node)
		} else {
			result.Edges = append(result.Edges, *record.edge)
		}
	}
	return result
}

func graphShowEncodedPageSize(snapshot, center string, page, pages int, records []graphShowRecord) int {
	encoded, _ := json.Marshal(graphShowPageFromRecords(snapshot, center, page, pages, records))
	return len(encoded) + 1
}
