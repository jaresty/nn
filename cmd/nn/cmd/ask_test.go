package cmd

import (
	"bytes"
	"net/http"
	neturl "net/url"
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
		// property [4]: the browser lands on the static UI entry at
		// "/?session=<id>", not the raw JSON endpoint. Derive the session
		// endpoint base from the query param and drive to submit through it.
		u, err := neturl.Parse(url)
		if err != nil {
			return err
		}
		id := u.Query().Get("session")
		base := u.Scheme + "://" + u.Host
		resp, err := http.Post(base+"/session/"+id+"/submit", "application/json", nil)
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
	// property [4]: opens the static UI entry with the session id as a query
	// param, not the raw JSON /session/<id> endpoint.
	pu, err := neturl.Parse(openedURL)
	if err != nil {
		t.Fatalf("parse opened URL: %v", err)
	}
	if pu.Path != "/" {
		t.Fatalf("opened URL path = %q, want %q", pu.Path, "/")
	}
	if got := pu.Query().Get("session"); got != sess.id {
		t.Fatalf("opened URL session param = %q, want %q", got, sess.id)
	}
	wantPath := filepath.Join(sess.dir, "result.json")
	if !strings.Contains(out.String(), wantPath) {
		t.Fatalf("output %q does not contain result path %q", out.String(), wantPath)
	}
}
