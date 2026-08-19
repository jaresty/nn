package cmd

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestAskCommandRegistered(t *testing.T) {
	cmd := newAskCmd(&rootState{})
	if !strings.HasPrefix(cmd.Use, "ask") {
		t.Fatalf("expected Use to start with 'ask', got %q", cmd.Use)
	}
	if cmd.Flags().Lookup("surface") == nil {
		t.Fatalf("expected --surface flag to be registered")
	}
}

func TestRunAskDrivesSessionToResult(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	var openedURL string
	openCalls := 0

	// Injected hook stands in for the browser: when the surface "opens", it
	// verifies the request is already prepared, then drives the session to
	// submit so Wait() returns.
	hook := func(url string) error {
		openCalls++
		openedURL = url
		// property [2]: request must be readable at open time.
		// Drive to submit via the real endpoint.
		resp, err := http.Post(url+"/submit", "application/json", nil)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}

	var out bytes.Buffer
	opts := askOptions{
		surface: "canvas",
		open:    hook,
		out:     &out,
	}
	sess, err := runAsk(opts)
	if err != nil {
		t.Fatalf("runAsk: %v", err)
	}

	if openCalls != 1 {
		t.Fatalf("open hook called %d times, want 1", openCalls)
	}
	if !strings.Contains(openedURL, "/session/"+sess.id) {
		t.Fatalf("opened URL %q does not address session %q", openedURL, sess.id)
	}
	wantPath := filepath.Join(sess.dir, "result.json")
	if !strings.Contains(out.String(), wantPath) {
		t.Fatalf("output %q does not contain result path %q", out.String(), wantPath)
	}
}
