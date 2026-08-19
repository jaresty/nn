package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/feedback"
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

// property [11a] + [11b]: CLI flags for mermaid and instructions flow into the
// prepared request.json before the surface launches.
func TestRunAskSeedsRequestFromFlags(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	// The hook stands in for the browser: the request is already on disk when
	// it fires, so assert the seeded fields there, then drive submit so
	// runAsk's Wait() returns instead of blocking forever.
	var seededErr error
	hook := func(url string) error {
		u, _ := neturl.Parse(url)
		id := u.Query().Get("session")
		req, err := feedback.ReadRequest(feedback.SessionDir(id))
		if err != nil {
			seededErr = err
			return err
		}
		if req.Mermaid != "graph TD; A-->B" {
			seededErr = fmt.Errorf("request Mermaid = %q, want seeded value", req.Mermaid)
		}
		if req.Instructions != "edit the flow" {
			seededErr = fmt.Errorf("request Instructions = %q, want seeded value", req.Instructions)
		}
		base := u.Scheme + "://" + u.Host
		resp, err := http.Post(base+"/session/"+id+"/submit", "application/json", nil)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}

	if _, err := runAsk(askOptions{
		surface:      "canvas",
		mermaid:      "graph TD; A-->B",
		instructions: "edit the flow",
		open:         hook,
		out:          io.Discard,
	}); err != nil {
		t.Fatalf("runAsk: %v", err)
	}
	if seededErr != nil {
		t.Fatal(seededErr)
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
