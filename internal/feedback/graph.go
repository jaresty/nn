package feedback

// GraphNode and GraphEdge are the surface-native shape the graph feedback
// surface renders. They mirror the interactive viewer's /graph payload so the
// same frontend (graph.html) can drive against either server.
type GraphNode struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Type   string   `json:"type"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
	Body   string   `json:"body"`
	Zone   string   `json:"zone,omitempty"`
	Degree int      `json:"degree"`
}

type GraphEdge struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Type       string `json:"type,omitempty"`
	Annotation string `json:"annotation,omitempty"`
}

// GraphSource supplies the candidate notebook graph for the graph surface. The
// server, not the source, applies the request's AllowedNodes bound — the source
// is deliberately scope-agnostic so the bound has a single enforcement point.
type GraphSource interface {
	Graph() ([]GraphNode, []GraphEdge, error)
}

// SetGraphSource attaches the graph data provider used by the /graph endpoint.
func (s *Server) SetGraphSource(src GraphSource) { s.graph = src }

// SetGraphHTML attaches the graph-viewer HTML shell served at / for the graph
// surface, in place of the embedded canvas bundle.
func (s *Server) SetGraphHTML(html []byte) { s.graphHTML = html }
