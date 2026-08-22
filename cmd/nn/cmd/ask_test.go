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

	"github.com/jaresty/nn/internal/backend/gitlocal"
	"github.com/jaresty/nn/internal/feedback"
	"github.com/jaresty/nn/internal/note"
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

// property [3]: --surface graph requires --focus. Without a focus there is no
// ego to bound the scope, so the load-bearing scoping constraint (ADR-0021)
// cannot be satisfied and runAsk must error before opening a surface.
func TestRunAskGraphRequiresFocus(t *testing.T) {
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	opened := false
	_, err := runAsk(askOptions{
		surface: "graph",
		focus:   "",
		open:    func(string) error { opened = true; return nil },
		out:     io.Discard,
	})
	if err == nil {
		t.Fatalf("runAsk(surface=graph, focus=\"\") = nil error, want error")
	}
	if opened {
		t.Fatalf("surface opened despite missing --focus; scope guard must block before serving")
	}
}

// property [5a]+[5b]: the graph feedback surface's HTML carries a persistently
// visible commentary affordance (so it is obvious WHERE comments go) and a
// labeled Done/submit control (so it is obvious HOW to finish). Structural check
// on the rendered viewer; the behavioral guard is the Playwright spec.
func TestGraphViewerHTMLHasCommentaryAndDoneControls(t *testing.T) {
	html, err := renderGraphViewerHTML()
	if err != nil {
		t.Fatalf("renderGraphViewerHTML: %v", err)
	}
	s := string(html)
	// property [5a]: a commentary affordance is present and labeled for comments.
	// Includes the inline per-node comment box (the most discoverable path —
	// shown on node click, not buried in a modal).
	for _, want := range []string{`id="brief-answer"`, `id="feedback-banner"`, `id="panel-comment"`} {
		if !strings.Contains(s, want) {
			t.Errorf("property [5a]: rendered viewer missing commentary affordance %q", want)
		}
	}
	// property [5b]: a labeled Done/submit control is present.
	if !strings.Contains(s, `id="btn-done"`) || !strings.Contains(s, "Done") {
		t.Errorf("property [5b]: rendered viewer missing labeled Done control")
	}
	if !strings.Contains(s, `id="btn-submit"`) {
		t.Errorf("property [5b]: rendered viewer missing submit control")
	}
}

// property [2]: for --surface graph, the request written to disk BEFORE the
// surface opens carries the resolved allowed-node set — the focus's depth-1 ego
// neighborhood (ego + direct neighbors in both directions), excluding notes
// outside that neighborhood. This is the agent-supplied scope bound; the server
// serves only these, so what the human sees is bounded here at prepare time.
func TestRunAskGraphResolvesEgoScopeIntoRequest(t *testing.T) {
	nbDir, _ := setupNotebookWithCfg(t)
	t.Setenv("NN_CONFIG_DIR", t.TempDir())

	ego := newTestNoteForCLI(note.GenerateID(), "Ego", note.TypeModel)
	out := newTestNoteForCLI(note.GenerateID(), "Outbound", note.TypeConcept)
	in := newTestNoteForCLI(note.GenerateID(), "Inbound", note.TypeArgument)
	far := newTestNoteForCLI(note.GenerateID(), "Far", note.TypeConcept)
	ego.Links = []note.Link{{TargetID: out.ID, Type: "refines", Annotation: "a"}}
	in.Links = []note.Link{{TargetID: ego.ID, Type: "supports", Annotation: "b"}}
	// far links only to out, so it is depth-2 from ego and must be excluded.
	far.Links = []note.Link{{TargetID: out.ID, Type: "supports", Annotation: "c"}}
	writeNoteFile(t, nbDir, ego)
	writeNoteFile(t, nbDir, out)
	writeNoteFile(t, nbDir, in)
	writeNoteFile(t, nbDir, far)

	b, err := gitlocal.New(nbDir)
	if err != nil {
		t.Fatalf("gitlocal.New: %v", err)
	}

	var checkErr error
	hook := func(url string) error {
		u, _ := neturl.Parse(url)
		id := u.Query().Get("session")
		req, err := feedback.ReadRequest(feedback.SessionDir(id))
		if err != nil {
			checkErr = err
			return err
		}
		got := map[string]bool{}
		for _, n := range req.AllowedNodes {
			got[n] = true
		}
		want := []string{ego.ID, out.ID, in.ID}
		for _, w := range want {
			if !got[w] {
				checkErr = fmt.Errorf("AllowedNodes missing %s; got %v", w, req.AllowedNodes)
			}
		}
		if got[far.ID] {
			checkErr = fmt.Errorf("AllowedNodes leaked depth-2 node %s; scope not bounded: %v", far.ID, req.AllowedNodes)
		}
		if req.Focus != ego.ID {
			checkErr = fmt.Errorf("request Focus = %q, want %q", req.Focus, ego.ID)
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
		surface: "graph",
		focus:   ego.ID,
		backend: b,
		open:    hook,
		out:     io.Discard,
	}); err != nil {
		t.Fatalf("runAsk: %v", err)
	}
	if checkErr != nil {
		t.Fatal(checkErr)
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
