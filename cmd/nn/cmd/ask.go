package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/feedback"
	"github.com/jaresty/nn/internal/note"
)

// askOptions configures a runAsk invocation. open is injected so tests can
// drive the session without launching a real browser.
type askOptions struct {
	surface string
	open    func(url string) error
	out     io.Writer
}

// askSession identifies a completed feedback session.
type askSession struct {
	id  string
	dir string
}

func newAskCmd(state *rootState) *cobra.Command {
	var surface string

	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Ask a human for feedback via a chosen surface and return the result",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := runAsk(askOptions{
				surface: surface,
				open:    openBrowser,
				out:     cmd.OutOrStdout(),
			})
			return err
		},
	}
	cmd.Flags().StringVar(&surface, "surface", "canvas", "Feedback surface (canvas, document, web)")
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

	id := note.GenerateID()
	dir := feedback.SessionDir(id)
	sess := askSession{id: id, dir: dir}

	req := feedback.FeedbackRequest{
		ID:      id,
		Surface: opts.surface,
		Mode:    "create",
	}
	// property [2]: request must be on disk before the surface opens.
	if err := feedback.WriteRequest(dir, req); err != nil {
		return sess, err
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
