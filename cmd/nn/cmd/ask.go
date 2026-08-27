package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/config"
	"github.com/jaresty/nn/internal/feedback"
	"github.com/jaresty/nn/internal/note"
)

// feedbackRetention is how long a completed feedback session directory is kept
// before nn ask reclaims it on its next run. Mirrors the daily-note expiry.
const feedbackRetention = 7 * 24 * time.Hour

// askOptions configures a runAsk invocation. open and runPlannotator are
// injected so tests can drive a session without launching a real surface.
type askOptions struct {
	surface        string
	mermaid        string
	instructions   string
	document       string          // for --surface document: file or folder to annotate
	focus          string          // for --surface graph: required ego note id bounding the scope
	nodes          string          // for --surface graph: optional explicit allowed-node-id allowlist (comma-separated)
	backend        backend.Backend // for --surface graph: source of notes for scope resolution + serving
	open           func(url string) error
	runPlannotator func(argv []string) error
	out            io.Writer
}

// askSession identifies a completed feedback session.
type askSession struct {
	id  string
	dir string
}

func newAskCmd(state *rootState) *cobra.Command {
	var surface, mermaid, instructions, document, focus, nodes string

	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask a human for feedback via a chosen surface and return the result",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// ask is exempt from initState's notebook setup (it is note-agnostic
			// at the boundary), so state.backend is nil. The graph surface is the
			// one surface that reads the notebook — open the backend lazily here.
			b := state.backend
			if surface == "graph" && b == nil {
				opened, err := openDefaultBackend()
				if err != nil {
					return fmt.Errorf("ask --surface graph: %w", err)
				}
				b = opened
			}

			open := openBrowser
			// Automation affordance: instead of launching a browser, print the
			// surface URL to stdout so a harness can drive it. The session still
			// blocks on submit, exactly as with a real browser.
			if os.Getenv("NN_ASK_PRINT_URL_ONLY") == "1" {
				out := cmd.OutOrStdout()
				open = func(url string) error {
					fmt.Fprintf(out, "ASK_SURFACE_URL %s\n", url)
					return nil
				}
			}
			_, err := runAsk(askOptions{
				surface:      surface,
				mermaid:      mermaid,
				instructions: instructions,
				document:     document,
				focus:        focus,
				nodes:        nodes,
				backend:      b,
				open:         open,
				out:          cmd.OutOrStdout(),
			})
			return err
		},
	}
	cmd.Flags().StringVar(&surface, "surface", "canvas", "Feedback surface (canvas, document, graph, web)")
	cmd.Flags().StringVar(&instructions, "instructions", "", "Instructions shown to the human on the surface")
	cmd.Flags().StringVar(&mermaid, "mermaid", "", "Mermaid diagram source to seed the canvas (converted to editable elements)")
	cmd.Flags().StringVar(&document, "document", "", "For --surface document or web: file, folder, or URL to annotate (defaults to a session file from --instructions)")
	cmd.Flags().StringVar(&focus, "focus", "", "For --surface graph: required ego note id — the human sees only its depth-1 neighborhood")
	cmd.Flags().StringVar(&nodes, "nodes", "", "For --surface graph: optional comma-separated allowlist of note ids to scope the surface to")
	return cmd
}

// runAsk prepares a feedback session, launches its surface via the injected
// open hook, blocks until the human submits or cancels, and prints the result
// path. The process is disposable; the session directory is durable.
func runAsk(opts askOptions) (askSession, error) {
	if opts.open == nil {
		opts.open = openBrowser
	}
	if opts.out == nil {
		opts.out = io.Discard
	}

	// property [3]: the graph surface is scoped by an ego node; without --focus
	// there is no neighborhood to bound what the human sees, so the load-bearing
	// scoping constraint (ADR-0021) cannot be satisfied. Fail before opening any
	// surface rather than dropping the human into the whole graph.
	if opts.surface == "graph" && opts.focus == "" {
		return askSession{}, fmt.Errorf("ask --surface graph requires --focus <note-id> to bound the scope")
	}

	// Reclaim aged session directories before starting a new one. Sessions are
	// ephemeral scratch — the durable artifact is whatever the agent persisted
	// from a result — so this is safe and best-effort (errors are ignored).
	_ = feedback.CleanupSessions(feedback.FeedbackRoot(), feedbackRetention)

	id := note.GenerateID()
	dir := feedback.SessionDir(id)
	sess := askSession{id: id, dir: dir}

	req := feedback.FeedbackRequest{
		ID:           id,
		Surface:      opts.surface,
		Mode:         "create",
		Instructions: opts.instructions,
		Mermaid:      opts.mermaid,
	}

	// property [2]: for the graph surface, resolve the agent-supplied scope — the
	// focus's depth-1 ego neighborhood, or an explicit --nodes allowlist — and
	// seed it into the request so the bound is on disk before the surface opens.
	if opts.surface == "graph" {
		allowed, err := resolveGraphScope(opts.backend, opts.focus, opts.nodes)
		if err != nil {
			return sess, err
		}
		req.Focus = opts.focus
		req.AllowedNodes = allowed
	}

	// property [2]: request must be on disk before the surface opens.
	if err := feedback.WriteRequest(dir, req); err != nil {
		return sess, err
	}

	// property [15a]/[15b]: document and its web compatibility alias are
	// delegated adapters — they invoke the plannotator peer rather than hosting
	// a server.
	if opts.surface == "document" || opts.surface == "web" {
		if err := runDocumentSurface(opts, dir, id); err != nil {
			return sess, err
		}
		printCompletion(opts.out, dir, feedback.OutcomeSubmitted)
		return sess, nil
	}

	srv, err := feedback.NewServer(id, dir)
	if err != nil {
		return sess, err
	}

	// The graph surface reuses the interactive viewer (graph.html) as its UI and
	// serves the notebook graph bounded to the request's AllowedNodes. Attaching
	// the source + shell before Start makes the server serve the graph viewer at
	// / and the scoped data at /graph.
	if opts.surface == "graph" {
		html, err := renderGraphViewerHTML()
		if err != nil {
			return sess, err
		}
		srv.SetGraphHTML(html)
		srv.SetGraphSource(&backendGraphSource{backend: opts.backend, focus: opts.focus})
	}

	if err := srv.Start(); err != nil {
		return sess, err
	}

	// Open the static UI entry with the session id as a query param; the
	// embedded frontend reads ?session=<id> and drives the /session/<id>
	// endpoints. Opening /session/<id> directly would show raw JSON.
	sessionURL := fmt.Sprintf("http://%s/?session=%s", srv.Addr(), id)
	// Print the URL before attempting to open the browser. Auto-open is
	// best-effort (exec "open".Start() swallows failure and can "succeed" while
	// no tab appears), so an always-printed URL is the reliable fallback.
	fmt.Fprintf(opts.out, "Feedback surface: %s\n", sessionURL)
	if err := opts.open(sessionURL); err != nil {
		return sess, err
	}

	outcome := srv.Wait()

	printCompletion(opts.out, dir, outcome)
	return sess, nil
}

// openDefaultBackend resolves and opens the configured notebook backend the
// same way initState does. The graph surface needs it even though ask is
// otherwise notebook-agnostic.
func openDefaultBackend() (backend.Backend, error) {
	cfgFile := defaultConfigPath()
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("no notebook config found at %s — run `nn init` first", cfgFile)
	}
	nb, err := cfg.Notebook(os.Getenv("NN_NOTEBOOK"))
	if err != nil {
		return nil, fmt.Errorf("resolve notebook: %w", err)
	}
	return gitlocal.New(nb.Path)
}

// resolveGraphScope computes the exact set of note ids the graph surface may
// show. An explicit --nodes allowlist wins verbatim; otherwise the scope is the
// focus's depth-1 ego neighborhood (the focus plus its direct neighbors in both
// link directions), reusing bfsDepthBoth so it matches the viewer's focused
// fetch. This resolved set is the agent-supplied bound on what the human sees.
func resolveGraphScope(b backend.Backend, focus, nodes string) ([]string, error) {
	if strings.TrimSpace(nodes) != "" {
		var ids []string
		for _, part := range strings.Split(nodes, ",") {
			if id := strings.TrimSpace(part); id != "" {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}
	if b == nil {
		return nil, fmt.Errorf("ask --surface graph requires a configured notebook to resolve --focus scope")
	}
	notes, err := b.List()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*note.Note, len(notes))
	for _, n := range notes {
		byID[n.ID] = n
	}
	root, ok := byID[focus]
	if !ok {
		return nil, fmt.Errorf("ask --surface graph: focus note %q not found", focus)
	}
	entries := bfsDepthBoth(root, byID, 1)
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.n.ID)
	}
	return ids, nil
}

// renderGraphViewerHTML produces the interactive graph-viewer shell (graph.html
// with d3 inlined) in serve mode, so the graph feedback surface reuses the same
// UI as `nn graph export --serve`. The seed graph is empty; the viewer fetches
// the scoped data from /graph at load.
func renderGraphViewerHTML() ([]byte, error) {
	return buildHTML(nil, nil, nil, true)
}

// backendGraphSource adapts the notebook backend to feedback.GraphSource. It
// returns the whole notebook graph annotated with degree and focus-relative
// zone; the feedback server bounds the response to the request's AllowedNodes,
// so this source is deliberately scope-agnostic.
type backendGraphSource struct {
	backend backend.Backend
	focus   string
}

func (g *backendGraphSource) Graph() ([]feedback.GraphNode, []feedback.GraphEdge, error) {
	if g.backend == nil {
		return nil, nil, fmt.Errorf("graph surface: no notebook backend configured")
	}
	notes, err := g.backend.List()
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]*note.Note, len(notes))
	for _, n := range notes {
		byID[n.ID] = n
	}

	// Full-graph incident degree per node, so rim nodes can show hidden-edge counts.
	degByID := make(map[string]int, len(notes))
	for _, n := range notes {
		for _, lnk := range n.Links {
			if _, ok := byID[lnk.TargetID]; ok {
				degByID[n.ID]++
				degByID[lnk.TargetID]++
			}
		}
	}

	// Zone each node by its direct link to the focus, reusing zoneOf so the
	// surface and `nn graph show --zones` agree.
	zoneByID := make(map[string]string)
	if root, ok := byID[g.focus]; ok {
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
	}

	nodes := make([]feedback.GraphNode, 0, len(notes))
	for _, n := range notes {
		tags := n.Tags
		if tags == nil {
			tags = []string{}
		}
		nodes = append(nodes, feedback.GraphNode{
			ID: n.ID, Title: n.Title, Type: string(n.Type),
			Status: string(n.Status), Tags: tags, Body: n.Body,
			Zone: zoneByID[n.ID], Degree: degByID[n.ID],
		})
	}
	var edges []feedback.GraphEdge
	for _, n := range notes {
		for _, lnk := range n.Links {
			if _, ok := byID[lnk.TargetID]; ok {
				edges = append(edges, feedback.GraphEdge{
					Source: n.ID, Target: lnk.TargetID, Type: lnk.Type, Annotation: lnk.Annotation,
				})
			}
		}
	}
	return nodes, edges, nil
}

// printCompletion reports the terminal outcome. A cancellation is complete in
// itself and has no result artifact to read. A submission names each native
// artifact and tells the agent to consume it before treating the task as done.
// Raw artifact content is never dumped (a canvas scene is a graph, not prose).
func printCompletion(out io.Writer, dir string, outcome feedback.Outcome) {
	if outcome == feedback.OutcomeCancelled {
		fmt.Fprintf(out, "Feedback cancelled (session %s).\n", filepath.Base(dir))
		return
	}

	resultPath := filepath.Join(dir, "result.json")
	result, err := feedback.ReadResult(dir)
	if err != nil {
		fmt.Fprintf(out, "Feedback collected.\nResult: %s\n", resultPath)
		fmt.Fprintf(out, "NEXT: read %s — the task is not complete until you have read and acted on the human's feedback.\n", resultPath)
		return
	}

	fmt.Fprintf(out, "Feedback collected (session %s, status: %s).\n\n", result.ID, result.Status)
	if len(result.Artifacts) == 0 {
		fmt.Fprintf(out, "No artifacts were produced. Result envelope: %s\n", resultPath)
		fmt.Fprintln(out, "NEXT: the task is not complete until you have confirmed and acted on this outcome.")
		return
	}

	fmt.Fprintln(out, "Artifacts (read and interpret these before treating the task as complete):")
	for i, a := range result.Artifacts {
		fmt.Fprintf(out, "  %d. %-12s %s\n", i+1, a.Format, filepath.Join(dir, a.Path))
	}
	fmt.Fprintln(out, "\nNEXT: the task is not complete until you have read and acted on the artifact(s) above — they contain the human's feedback you requested.")
}

// runDocumentSurface is the delegated adapter for --surface document. It hands
// a document (file, folder, or URL) to the plannotator peer on the contract
// "path in -> result path out -> exit", then records a thin envelope naming the
// native plannotator decision file. nn does not parse plannotator's JSON — the
// agent reads it (ADR-0020).
func runDocumentSurface(opts askOptions, dir, id string) error {
	// property [16a'-i]/[16a'-ii]: annotate the supplied --document path when
	// given, otherwise write the instructions to a session file and annotate it.
	contentPath := opts.document
	if contentPath == "" {
		contentPath = filepath.Join(dir, "document.md")
		if err := os.WriteFile(contentPath, []byte(opts.instructions+"\n"), 0o644); err != nil {
			return err
		}
	}

	resultFile := filepath.Join(dir, "result.plannotator.json")
	// property [16b]: request path in, --result-file out, then exit.
	argv := []string{"annotate", contentPath, "--gate", "--json", "--result-file", resultFile}
	var err error
	if opts.runPlannotator != nil {
		err = opts.runPlannotator(argv)
	} else {
		dataDir := ""
		if isLocalHTMLDocument(contentPath) {
			// Plannotator 0.27.8 can blank its client while constructing a large
			// historical HTML diff. Keep local HTML review history inside this
			// disposable Ask session; every other document target retains the
			// ordinary inherited Plannotator environment.
			dataDir = filepath.Join(dir, "plannotator-data")
		}
		err = runPlannotator(argv, dataDir)
	}
	if err != nil {
		return err
	}

	// property [17a]/[17b]: normalize under the thin envelope.
	result := feedback.FeedbackResult{
		ID:      id,
		Surface: "document",
		Status:  string(feedback.OutcomeSubmitted),
		Artifacts: []feedback.Artifact{
			{Format: "plannotator-decision", Path: "result.plannotator.json"},
		},
	}
	return feedback.WriteResult(dir, result)
}

func isLocalHTMLDocument(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".html" && ext != ".htm" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// runPlannotator invokes the plannotator binary with argv, wiring stdio through
// so the human interacts with its UI. A non-empty dataDir replaces that variable
// in the child only; the nn process environment remains unchanged.
func runPlannotator(argv []string, dataDir string) error {
	cmd := exec.Command("plannotator", argv...)
	if dataDir != "" {
		env := make([]string, 0, len(os.Environ())+1)
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "PLANNOTATOR_DATA_DIR=") {
				env = append(env, entry)
			}
		}
		cmd.Env = append(env, "PLANNOTATOR_DATA_DIR="+dataDir)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// openBrowser opens url in the user's default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
