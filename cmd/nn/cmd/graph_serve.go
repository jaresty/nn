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
	"time"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

type serveState struct {
	backend      backend.Backend
	notebookPath string
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
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
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
			ID     string   `json:"id"`
			Title  string   `json:"title"`
			Type   string   `json:"type"`
			Status string   `json:"status"`
			Tags   []string `json:"tags"`
			Body   string   `json:"body"`
			Zone   string   `json:"zone,omitempty"`
			Degree int      `json:"degree"`
		}

		// Full-graph degree (in + out incident edges) per node, so the focused
		// viewer can show how many direct connections a rim node hides.
		degByID := make(map[string]int, len(notes))
		for _, n := range notes {
			for _, lnk := range n.Links {
				if _, ok := byID[lnk.TargetID]; ok {
					degByID[n.ID]++
					degByID[lnk.TargetID]++
				}
			}
		}
		type outEdge struct {
			Source     string `json:"source"`
			Target     string `json:"target"`
			Type       string `json:"type,omitempty"`
			Annotation string `json:"annotation,omitempty"`
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
			entries := bfsDepthBoth(root, byID, depth)
			// Zone each node by its direct link to the focus (root): root -> N
			// is dirOut, N -> root is dirIn. Reuses zoneOf, the same mapping as
			// `nn graph show --zones`, so the viewer and CLI agree.
			zoneByID := make(map[string]string)
			for _, lnk := range root.Links {
				if lnk.TargetID != root.ID {
					if z := zoneOf(lnk.Type, dirOut); z != zoneNone {
						zoneByID[lnk.TargetID] = string(z)
					}
				}
			}
			for _, n := range byID {
				if n.ID == root.ID {
					continue
				}
				for _, lnk := range n.Links {
					if lnk.TargetID == root.ID {
						if z := zoneOf(lnk.Type, dirIn); z != zoneNone {
							zoneByID[n.ID] = string(z)
						}
					}
				}
			}
			for _, e := range entries {
				nodeSet[e.n.ID] = true
				tags := e.n.Tags
				if tags == nil {
					tags = []string{}
				}
				outNodes = append(outNodes, outNode{
					ID: e.n.ID, Title: e.n.Title, Type: string(e.n.Type),
					Status: string(e.n.Status), Tags: tags, Body: e.n.Body,
					Zone: zoneByID[e.n.ID], Degree: degByID[e.n.ID],
				})
			}
			for _, e := range entries {
				for _, lnk := range e.n.Links {
					if nodeSet[lnk.TargetID] {
						outEdges = append(outEdges, outEdge{e.n.ID, lnk.TargetID, lnk.Type, lnk.Annotation})
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
				outNodes = append(outNodes, outNode{
					ID: n.ID, Title: n.Title, Type: string(n.Type),
					Status: string(n.Status), Tags: tags, Body: n.Body,
					Degree: degByID[n.ID],
				})
			}
			for _, n := range notes {
				for _, lnk := range n.Links {
					if nodeSet[lnk.TargetID] {
						outEdges = append(outEdges, outEdge{n.ID, lnk.TargetID, lnk.Type, lnk.Annotation})
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
