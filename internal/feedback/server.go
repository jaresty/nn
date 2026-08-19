package feedback

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Outcome is the terminal result of a feedback session's blocking Wait.
type Outcome string

const (
	OutcomeSubmitted Outcome = "submitted"
	OutcomeCancelled Outcome = "cancelled"
)

// Server hosts a single feedback session over an ephemeral loopback listener.
// The process is disposable; the session directory on disk is durable.
type Server struct {
	id   string
	dir  string
	ln   net.Listener
	srv  *http.Server
	done chan Outcome
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
	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln)
	return nil
}

// Addr is the concrete host:port the server is listening on.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Wait blocks until the session is submitted or cancelled, then returns the
// outcome. After it returns, the caller may Close the server.
func (s *Server) Wait() Outcome {
	o := <-s.done
	if s.srv != nil {
		s.srv.Close()
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
	case r.Method == http.MethodPut && strings.HasSuffix(path, "/draft"):
		s.handlePutDraft(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/submit"):
		s.handleSubmit(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/cancel"):
		s.handleCancel(w, r)
	default:
		http.NotFound(w, r)
	}
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
// the surface's native output; the result wraps it in the thin envelope with a
// single artifact reference to the persisted draft.
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
	if _, err := os.Stat(draftPath); err == nil {
		result.Artifacts = []Artifact{{Format: "draft", Path: "draft.json"}}
	}
	return WriteResult(s.dir, result)
}
