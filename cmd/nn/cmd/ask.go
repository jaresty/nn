package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/feedback"
	"github.com/jaresty/nn/internal/note"
)

// feedbackRetention is how long a completed feedback session directory is kept
// before nn ask reclaims it on its next run. Mirrors the daily-note expiry.
const feedbackRetention = 7 * 24 * time.Hour

// askOptions configures a runAsk invocation. open and runPlannotator are
// injected so tests can drive a session without launching a real surface.
type askOptions struct {
	surface      string
	mermaid      string
	instructions string
	document     string // for --surface document: file or folder to annotate
	open         func(url string) error
	runPlannotator func(argv []string) error
	out          io.Writer
}

// askSession identifies a completed feedback session.
type askSession struct {
	id  string
	dir string
}

func newAskCmd(state *rootState) *cobra.Command {
	var surface, mermaid, instructions, document string

	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask a human for feedback via a chosen surface and return the result",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := runAsk(askOptions{
				surface:      surface,
				mermaid:      mermaid,
				instructions: instructions,
				document:     document,
				open:         openBrowser,
				out:          cmd.OutOrStdout(),
			})
			return err
		},
	}
	cmd.Flags().StringVar(&surface, "surface", "canvas", "Feedback surface (canvas, document, web)")
	cmd.Flags().StringVar(&instructions, "instructions", "", "Instructions shown to the human on the surface")
	cmd.Flags().StringVar(&mermaid, "mermaid", "", "Mermaid diagram source to seed the canvas (converted to editable elements)")
	cmd.Flags().StringVar(&document, "document", "", "For --surface document: file, folder, or URL to annotate (defaults to a session file from --instructions)")
	return cmd
}

// runAsk prepares a feedback session, launches its surface via the injected
// open hook, blocks until the human submits or cancels, and prints the result
// path. The process is disposable; the session directory is durable.
func runAsk(opts askOptions) (askSession, error) {
	if opts.open == nil {
		opts.open = openBrowser
	}
	if opts.runPlannotator == nil {
		opts.runPlannotator = runPlannotator
	}
	if opts.out == nil {
		opts.out = io.Discard
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
	// property [2]: request must be on disk before the surface opens.
	if err := feedback.WriteRequest(dir, req); err != nil {
		return sess, err
	}

	// property [15a]/[15b]: the document surface is a delegated adapter — it
	// invokes the plannotator peer rather than hosting a server.
	if opts.surface == "document" {
		if err := runDocumentSurface(opts, dir, id); err != nil {
			return sess, err
		}
		resultPath := filepath.Join(dir, "result.json")
		fmt.Fprintf(opts.out, "Feedback collected.\nResult: %s\n", resultPath)
		return sess, nil
	}

	srv, err := feedback.NewServer(id, dir)
	if err != nil {
		return sess, err
	}
	if err := srv.Start(); err != nil {
		return sess, err
	}

	// Open the static UI entry with the session id as a query param; the
	// embedded frontend reads ?session=<id> and drives the /session/<id>
	// endpoints. Opening /session/<id> directly would show raw JSON.
	sessionURL := fmt.Sprintf("http://%s/?session=%s", srv.Addr(), id)
	if err := opts.open(sessionURL); err != nil {
		return sess, err
	}

	srv.Wait()

	resultPath := filepath.Join(dir, "result.json")
	fmt.Fprintf(opts.out, "Feedback collected.\nResult: %s\n", resultPath)
	return sess, nil
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
	if err := opts.runPlannotator(argv); err != nil {
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

// runPlannotator invokes the plannotator binary with argv, wiring stdio through
// so the human interacts with its UI.
func runPlannotator(argv []string) error {
	cmd := exec.Command("plannotator", argv...)
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
