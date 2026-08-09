package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

type serveState struct {
	backend      backend.Backend
	notebookPath string
	mu           sync.Mutex
	messages     []serveMessage
}

type serveMessage struct {
	Text string `json:"text"`
	Ts   string `json:"ts"`
}

func startServeMode(ctx context.Context, b backend.Backend, notebookPath string, htmlBytes []byte, port int, openBrowser bool, verbose bool) error {
	s := &serveState{backend: b, notebookPath: notebookPath}

	var logger *log.Logger
	if verbose {
		logger = log.New(os.Stderr, "nn-serve: ", log.LstdFlags)
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlBytes)
	})

	mux.HandleFunc("GET /graph", func(w http.ResponseWriter, r *http.Request) {
		focusID := r.URL.Query().Get("focus")
		depthStr := r.URL.Query().Get("depth")
		depth := 2
		if depthStr != "" {
			if d, err := strconv.Atoi(depthStr); err == nil {
				depth = d
			}
		}

		notes, err := s.backend.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		byID := make(map[string]*note.Note, len(notes))
		for _, n := range notes {
			byID[n.ID] = n
		}

		type outNode struct {
			ID    string   `json:"id"`
			Title string   `json:"title"`
			Type  string   `json:"type"`
			Tags  []string `json:"tags"`
			Body  string   `json:"body"`
		}
		type outEdge struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}

		var outNodes []outNode
		var outEdges []outEdge
		nodeSet := make(map[string]bool)

		if focusID != "" {
			root, ok := byID[focusID]
			if !ok {
				http.Error(w, "note not found", http.StatusNotFound)
				return
			}
			entries := bfsDepth(root, byID, depth)
			for _, e := range entries {
				nodeSet[e.n.ID] = true
				tags := e.n.Tags
				if tags == nil {
					tags = []string{}
				}
				outNodes = append(outNodes, outNode{e.n.ID, e.n.Title, string(e.n.Type), tags, e.n.Body})
			}
			for _, e := range entries {
				for _, lnk := range e.n.Links {
					if nodeSet[lnk.TargetID] {
						outEdges = append(outEdges, outEdge{e.n.ID, lnk.TargetID})
					}
				}
			}
		} else {
			for _, n := range notes {
				nodeSet[n.ID] = true
				tags := n.Tags
				if tags == nil {
					tags = []string{}
				}
				outNodes = append(outNodes, outNode{n.ID, n.Title, string(n.Type), tags, n.Body})
			}
			for _, n := range notes {
				for _, lnk := range n.Links {
					if nodeSet[lnk.TargetID] {
						outEdges = append(outEdges, outEdge{n.ID, lnk.TargetID})
					}
				}
			}
		}
		if outEdges == nil {
			outEdges = []outEdge{}
		}
		if outNodes == nil {
			outNodes = []outNode{}
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(struct {
			Nodes []outNode `json:"nodes"`
			Edges []outEdge `json:"edges"`
		}{outNodes, outEdges})
	})

	mux.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		notes, err := s.backend.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scores := RankedByQuery(notes, notes, req.Query, s.notebookPath)
		type result struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		}
		var results []result
		for _, n := range notes {
			if sc := scores[n.ID]; sc > 0 {
				results = append(results, result{n.ID, sc})
			}
		}
		if results == nil {
			results = []result{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	mux.HandleFunc("POST /event", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payload["event"] = "select"
		payload["ts"] = time.Now().UTC().Format(time.RFC3339)
		line, _ := json.Marshal(payload)
		fmt.Fprintf(os.Stdout, "%s\n", line)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /chat", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payload["event"] = "chat"
		payload["ts"] = time.Now().UTC().Format(time.RFC3339)
		line, _ := json.Marshal(payload)
		fmt.Fprintf(os.Stdout, "%s\n", line)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /message", func(w http.ResponseWriter, r *http.Request) {
		var msg serveMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if msg.Ts == "" {
			msg.Ts = time.Now().UTC().Format(time.RFC3339)
		}
		s.mu.Lock()
		s.messages = append(s.messages, msg)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /messages", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		msgs := s.messages
		s.messages = nil
		s.mu.Unlock()
		if msgs == nil {
			msgs = []serveMessage{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msgs)
	})

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	logger.Printf("listening on http://localhost:%d", port)
	fmt.Fprintf(os.Stdout, "nn graph serve: listening on http://localhost:%d\n", port)

	if openBrowser {
		openCmd := "xdg-open"
		if runtime.GOOS == "darwin" {
			openCmd = "open"
		}
		go exec.Command(openCmd, fmt.Sprintf("http://localhost:%d", port)).Start()
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("graph serve: %w", err)
	}
	return nil
}
