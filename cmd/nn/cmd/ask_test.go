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

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/feedback"
)

// property [14]: nn ask does not require a notebook config — it is note-agnostic
// at the boundary (ADR-0020), so initState must exempt it like init/skills.
func TestAskDoesNotRequireNotebookConfig(t *testing.T) {
	cmd := &cobra.Command{Use: "ask"}
	err := initState(cmd, &rootState{}, "/tmp/nonexistent-nn-config/config.toml")
	if err != nil {
		t.Fatalf("initState for ask returned %v, want nil (ask must not require config)", err)
	}
}

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

// TestRunAskCompletionListsArtifactsAndInstructs pins the completion output
// contract so the agent cannot finish without consuming the feedback:
//   property A1:  the session status appears
//   property A2a: each artifact's format appears
//   property A2b: each artifact's absolute path appears
//   property A3:  an explicit "not complete until read/acted on" instruction appears
func TestRunAskCompletionListsArtifactsAndInstructs(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	hook := func(url string) error {
		u, err := neturl.Parse(url)
		if err != nil {
			return err
		}
		id := u.Query().Get("session")
		base := u.Scheme + "://" + u.Host
		// Seed a draft so the submitted result envelope names an artifact.
		scene := bytes.NewReader([]byte(`{"type":"excalidraw","elements":[],"appState":{}}`))
		putReq, _ := http.NewRequest(http.MethodPut, base+"/session/"+id+"/draft", scene)
		putReq.Header.Set("Content-Type", "application/json")
		if resp, err := http.DefaultClient.Do(putReq); err == nil {
			resp.Body.Close()
		}
		resp, err := http.Post(base+"/session/"+id+"/submit", "application/json", nil)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}

	var out bytes.Buffer
	sess, err := runAsk(askOptions{surface: "canvas", open: hook, out: &out})
	if err != nil {
		t.Fatalf("runAsk: %v", err)
	}

	got := out.String()

	// property A1: session status is reported.
	result, err := feedback.ReadResult(sess.dir)
	if err != nil {
		t.Fatalf("ReadResult: %v", err)
	}
	if !strings.Contains(got, result.Status) {
		t.Errorf("property A1: output does not contain status %q:\n%s", result.Status, got)
	}

	// property A2a/A2b: each artifact's format and absolute path appear.
	if len(result.Artifacts) == 0 {
		t.Fatalf("expected at least one artifact in result envelope, got none")
	}
	for _, a := range result.Artifacts {
		if !strings.Contains(got, a.Format) {
			t.Errorf("property A2a: output missing artifact format %q:\n%s", a.Format, got)
		}
		absPath := filepath.Join(sess.dir, a.Path)
		if !strings.Contains(got, absPath) {
			t.Errorf("property A2b: output missing artifact absolute path %q:\n%s", absPath, got)
		}
	}

	// property A3: explicit not-complete-until-read instruction.
	if !strings.Contains(strings.ToLower(got), "not complete until") {
		t.Errorf("property A3: output missing 'not complete until' instruction:\n%s", got)
	}
}
