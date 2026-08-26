package feedback

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// webFS holds the static feedback-surface bundle served at GET / and asset
// paths. The bundle is a placeholder now; a later phase replaces web/ with the
// built Excalidraw React output.
//
//go:embed web
var webFS embed.FS

// Outcome is the terminal result of a feedback session's blocking Wait.
type Outcome string

const (
	OutcomeSubmitted Outcome = "submitted"
	OutcomeCancelled Outcome = "cancelled"
)

// Server hosts a single feedback session over an ephemeral loopback listener.
// The process is disposable; the session directory on disk is durable.
type Server struct {
	id        string
	dir       string
	ln        net.Listener
	srv       *http.Server
	done      chan Outcome
	graph     GraphSource // optional: supplies data for the graph surface's /graph endpoint
	graphHTML []byte      // optional: graph-viewer shell served at / for the graph surface
}

// NewServer creates a server for session id whose files live in dir.
func NewServer(id, dir string) (*Server, error) {
	return &Server{id: id, dir: dir, done: make(chan Outcome, 1)}, nil
}

// Start binds an ephemeral loopback port and begins serving. It does not block.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.ln = ln
	mux := http.NewServeMux()
	mux.HandleFunc("/session/", s.handleSession)
	// Graph-surface endpoints the viewer (graph.html) fetches at top level. They
	// are no-ops for the canvas surface (no graph source attached).
	mux.HandleFunc("/graph", s.handleGetGraph)
	mux.HandleFunc("/event", s.handleGraphEvent)
	mux.HandleFunc("/", s.handleStatic)
	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln)
	return nil
}

// Addr is the concrete host:port the server is listening on.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Wait blocks until the session is submitted or cancelled, then returns the
// outcome. It shuts the server down gracefully so the in-flight submit/cancel
// request that produced the outcome finishes flushing its response to the
// client before the connection closes — an abrupt Close races that response and
// surfaces as an EOF on the client.
func (s *Server) Wait() Outcome {
	o := <-s.done
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.srv.Shutdown(ctx); err != nil {
			s.srv.Close()
		}
	}
	return o
}

// stopForTest shuts the server down without a submit/cancel, for tests that
// exercise a single endpoint and do not drive the lifecycle to completion.
func (s *Server) stopForTest() {
	if s.srv != nil {
		s.srv.Close()
	}
}

func (s *Server) signal(o Outcome) {
	select {
	case s.done <- o:
	default:
	}
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && !strings.Contains(strings.TrimPrefix(path, "/session/"), "/"):
		s.handleGet(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/graph"):
		s.handleGetGraph(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/draft"):
		s.handleGetDraft(w, r)
	case r.Method == http.MethodPut && strings.HasSuffix(path, "/draft"):
		s.handlePutDraft(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/png"):
		s.handlePostPng(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/submit"):
		s.handleSubmit(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/cancel"):
		s.handleCancel(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleStatic serves the embedded feedback-surface bundle. GET / maps to
// index.html; any other path maps to the correspondingly named embedded file.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		// The graph surface swaps the default canvas bundle for the graph viewer.
		if s.graphHTML != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(s.graphHTML))
			return
		}
		name = "index.html"
	}
	body, err := webFS.ReadFile("web/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(body))
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	q, err := ReadRequest(s.dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func (s *Server) handlePutDraft(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(filepath.Join(s.dir, "draft.json"), body, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetDraft returns the persisted draft so the surface can restore it as
// initialData on reopen. A missing draft is 404 (no draft yet).
// handleGetGraph serves the scoped notebook graph for the graph surface. The
// response is bounded to the request's AllowedNodes: only nodes whose id is in
// that set, and only edges whose endpoints are both in it, are returned. This
// is the single server-side enforcement point for the agent-supplied scope.
func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	if s.graph == nil {
		http.Error(w, "no graph source configured", http.StatusInternalServerError)
		return
	}
	nodes, edges, err := s.graph.Graph()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q, err := ReadRequest(s.dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	allowed := make(map[string]bool, len(q.AllowedNodes))
	for _, id := range q.AllowedNodes {
		allowed[id] = true
	}

	// Bound to the agent-supplied scope: keep only in-scope nodes, and only
	// edges whose both endpoints are in scope. Non-empty AllowedNodes means the
	// request declared a bound; an empty set serves nothing rather than leaking
	// the whole notebook.
	outEdges := make([]GraphEdge, 0, len(edges))
	scopedDegree := make(map[string]int, len(allowed))
	for _, e := range edges {
		if allowed[e.Source] && allowed[e.Target] {
			outEdges = append(outEdges, e)
			scopedDegree[e.Source]++
			scopedDegree[e.Target]++
		}
	}
	outNodes := make([]GraphNode, 0, len(nodes))
	for _, n := range nodes {
		if allowed[n.ID] {
			// Degree must describe the graph this endpoint can actually reveal. Using
			// full-notebook degree advertises out-of-scope neighbors that recentering
			// is intentionally forbidden from serving.
			n.Degree = scopedDegree[n.ID]
			outNodes = append(outNodes, n)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Nodes []GraphNode `json:"nodes"`
		Edges []GraphEdge `json:"edges"`
	}{outNodes, outEdges})
}

// handleGraphEvent accepts the viewer's node-click relay POSTs. The graph
// surface is one-shot (prepare → submit), so these are acknowledged and dropped
// rather than streamed anywhere.
func (s *Server) handleGraphEvent(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		io.Copy(io.Discard, r.Body)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetDraft(w http.ResponseWriter, r *http.Request) {
	body, err := os.ReadFile(filepath.Join(s.dir, "draft.json"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// handlePostPng persists the surface's exported PNG so submit can name it as a
// result.png artifact.
func (s *Server) handlePostPng(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(filepath.Join(s.dir, "result.png"), body, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if err := s.promoteDraftToResult(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	s.signal(OutcomeSubmitted)
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	s.signal(OutcomeCancelled)
}

// promoteDraftToResult builds result.json from the latest draft. The draft is
// the surface's native output; on submit it is materialized into a native
// result artifact whose format matches the surface, and the thin envelope
// references it by path.
func (s *Server) promoteDraftToResult() error {
	draftPath := filepath.Join(s.dir, "draft.json")
	result := FeedbackResult{
		ID:      s.id,
		Status:  string(OutcomeSubmitted),
		Surface: "",
	}
	if q, err := ReadRequest(s.dir); err == nil {
		result.Surface = q.Surface
	}
	if draft, err := os.ReadFile(draftPath); err == nil {
		switch result.Surface {
		case "canvas":
			// The canvas draft is the Excalidraw scene JSON. Materialize it as
			// the native result.excalidraw artifact so the agent reads a scene,
			// not an opaque draft.
			scenePath := filepath.Join(s.dir, "result.excalidraw")
			if err := os.WriteFile(scenePath, draft, 0o644); err != nil {
				return err
			}
			result.Artifacts = []Artifact{{Format: "excalidraw", Path: "result.excalidraw"}}
			// property [8a]: name the exported png if the surface uploaded one.
			if _, err := os.Stat(filepath.Join(s.dir, "result.png")); err == nil {
				result.Artifacts = append(result.Artifacts, Artifact{Format: "png", Path: "result.png"})
			}
		case "graph":
			// Decode before promotion so the result can contain only the canonical
			// graph-selection schema. Graph feedback remains an artifact only and
			// never applies notebook mutations.
			selection, err := DecodeGraphSelection(draft)
			if err != nil {
				return err
			}
			selPath := filepath.Join(s.dir, "result.graph.json")
			if err := writeJSON(selPath, selection); err != nil {
				return err
			}
			result.Artifacts = []Artifact{{Format: "graph-selection", Path: "result.graph.json"}}

			// Canvas intent receives a deterministic NON_STORED seed for the later
			// agent-mediated handoff. Document intent needs no extra artifact: it is
			// represented completely by the graph-selection handoff field. No action
			// launches a surface or changes notebook files here.
			if selection.hasHandoff(GraphHandoffCanvas) {
				seedPath := filepath.Join(s.dir, "result.canvas-seed.json")
				if err := writeJSON(seedPath, newCanvasSeed(selection)); err != nil {
					return err
				}
				result.Artifacts = append(result.Artifacts, Artifact{Format: "canvas-seed", Path: "result.canvas-seed.json"})
			}
		default:
			result.Artifacts = []Artifact{{Format: "draft", Path: "draft.json"}}
		}
	}
	return WriteResult(s.dir, result)
}
