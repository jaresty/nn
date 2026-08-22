package feedback

import (
	"encoding/json"
	"net/http"
	"testing"
)

// fakeGraphSource returns a fixed node/edge set regardless of query; the server
// is responsible for bounding the response to the request's AllowedNodes.
type fakeGraphSource struct {
	nodes []GraphNode
	edges []GraphEdge
}

func (f fakeGraphSource) Graph() ([]GraphNode, []GraphEdge, error) {
	return f.nodes, f.edges, nil
}

// property [1]: the served graph is bounded to the request's AllowedNodes. A
// node outside that set must never appear in the response, and an edge touching
// an excluded node must be dropped — the agent-supplied scope bounds what the
// human sees, enforced server-side.
func TestGraphEndpointServesOnlyAllowedNodes(t *testing.T) {
	src := fakeGraphSource{
		nodes: []GraphNode{
			{ID: "ego", Title: "Ego"},
			{ID: "nbr", Title: "Neighbor"},
			{ID: "far", Title: "Far"}, // outside scope — must be excluded
		},
		edges: []GraphEdge{
			{Source: "ego", Target: "nbr", Type: "refines"},
			{Source: "nbr", Target: "far", Type: "supports"}, // touches excluded node
		},
	}
	q := FeedbackRequest{ID: "s1", Surface: "graph", Focus: "ego", AllowedNodes: []string{"ego", "nbr"}}

	dir := t.TempDir()
	if err := WriteRequest(dir, q); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	srv, err := NewServer(q.ID, dir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srv.SetGraphSource(src)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.stopForTest()
	base := "http://" + srv.Addr()

	resp, err := http.Get(base + "/session/s1/graph")
	if err != nil {
		t.Fatalf("GET graph: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		Nodes []GraphNode `json:"nodes"`
		Edges []GraphEdge `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, n := range got.Nodes {
		if n.ID == "far" {
			t.Fatalf("served out-of-scope node %q; scope not bounded server-side", n.ID)
		}
	}
	seen := map[string]bool{}
	for _, n := range got.Nodes {
		seen[n.ID] = true
	}
	if !seen["ego"] || !seen["nbr"] {
		t.Fatalf("served nodes = %v, want ego+nbr", got.Nodes)
	}
	for _, e := range got.Edges {
		if e.Target == "far" || e.Source == "far" {
			t.Fatalf("served edge touching out-of-scope node: %+v", e)
		}
	}
}
