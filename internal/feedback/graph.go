package feedback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

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

// SelectionKind records whether graph material was pointed at directly or was
// included because both endpoints of a selected relationship group were chosen.
// It is feedback metadata only; it never changes notebook topology.
type SelectionKind string

const (
	SelectionExplicit SelectionKind = "explicit"
	SelectionImplicit SelectionKind = "implicit"
)

// GraphSelectionNode and GraphSelectionEdge are members of a saved graph Ask
// group. Member comments keep note and relationship feedback inside the group
// schema rather than duplicating membership at the result's top level.
type GraphSelectionNode struct {
	ID        string        `json:"id"`
	Selection SelectionKind `json:"selection"`
	Comment   string        `json:"comment,omitempty"`
}

type GraphSelectionEdge struct {
	Source    string        `json:"source"`
	Target    string        `json:"target"`
	Type      string        `json:"type,omitempty"`
	Selection SelectionKind `json:"selection"`
	Comment   string        `json:"comment,omitempty"`
}

// GraphSelectionGroup is one optionally named, annotated answer unit. Its
// classification is a human feedback label, not a notebook link type.
type GraphSelectionGroup struct {
	ID             string               `json:"id"`
	Name           string               `json:"name,omitempty"`
	Classification string               `json:"classification,omitempty"`
	Nodes          []GraphSelectionNode `json:"nodes"`
	Edges          []GraphSelectionEdge `json:"edges"`
	Comment        string               `json:"comment,omitempty"`
}

// GraphSelection is the graph surface's native result. These three fields are
// the complete top-level schema; all note and relationship membership belongs
// to Groups. Handoff records one terminal intent only: the graph surface never
// launches a destination surface.
type GraphSelection struct {
	Groups         []GraphSelectionGroup `json:"groups"`
	OverallComment string                `json:"overall_comment"`
	Handoff        *GraphHandoff         `json:"handoff"`
}

type GraphHandoff string

const (
	GraphHandoffCanvas   GraphHandoff = "canvas"
	GraphHandoffDocument GraphHandoff = "document"
)

// DecodeGraphSelection rejects unknown fields so removed, unreleased schema
// keys cannot silently reappear in drafts or promoted results.
func DecodeGraphSelection(body []byte) (GraphSelection, error) {
	selection := GraphSelection{Groups: []GraphSelectionGroup{}}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&selection); err != nil {
		return GraphSelection{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return GraphSelection{}, fmt.Errorf("graph-selection contains multiple JSON values")
		}
		return GraphSelection{}, err
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(body, &topLevel); err != nil {
		return GraphSelection{}, err
	}
	for _, required := range []string{"groups", "overall_comment", "handoff"} {
		if _, ok := topLevel[required]; !ok {
			return GraphSelection{}, fmt.Errorf("graph-selection missing required top-level key %q", required)
		}
	}

	if selection.Handoff != nil && *selection.Handoff != GraphHandoffCanvas && *selection.Handoff != GraphHandoffDocument {
		return GraphSelection{}, fmt.Errorf("unknown graph-selection handoff %q", *selection.Handoff)
	}
	return selection, nil
}

func (s GraphSelection) hasHandoff(want GraphHandoff) bool {
	return s.Handoff != nil && *s.Handoff == want
}

// CanvasSeed is an answer-composition handoff, not a notebook mutation. The
// explicit NON_STORED label prevents grouping and explanatory relationships
// from being mistaken for persisted note links or spatial meaning.
type CanvasSeed struct {
	Storage string                `json:"storage"`
	Source  string                `json:"source"`
	Notice  string                `json:"notice"`
	Groups  []GraphSelectionGroup `json:"groups"`
}

func newCanvasSeed(selection GraphSelection) CanvasSeed {
	return CanvasSeed{
		Storage: "NON_STORED",
		Source:  "graph-selection",
		Notice:  "Derived grouping and explanatory structure; not notebook truth or stored edges.",
		Groups:  selection.Groups,
	}
}
