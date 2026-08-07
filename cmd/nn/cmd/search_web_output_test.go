package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchWebPerResultOutput(t *testing.T) {
	// property [3b]: for each result URL, fetch+strip+print preview + Related notes section
	var resultSrv *httptest.Server
	resultSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>Result Page</h1><p>Content about machine learning and neural networks.</p></body></html>`))
	}))
	defer resultSrv.Close()

	// DDG mock returns one result pointing at resultSrv
	ddgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><a class="result__a" href="%s">A result</a></body></html>`, resultSrv.URL)
	}))
	defer ddgSrv.Close()

	var stdout bytes.Buffer
	err := runSearchWeb("machine learning", 1, ddgSrv.URL+"?q=%s", &stdout, nil, nil, false)
	if err != nil {
		t.Fatalf("runSearchWeb: %v", err)
	}

	out := stdout.String()

	// property [3b]: output contains "## Result N:" header
	if !strings.Contains(out, "## Result 1:") {
		t.Errorf("property [3b]: expected '## Result 1:' header in output, got:\n%s", out)
	}
	// property [3b]: output contains stripped text from the fetched page
	if !strings.Contains(out, "Result Page") && !strings.Contains(out, "machine learning") {
		t.Errorf("property [3b]: expected fetched page content in output, got:\n%s", out)
	}
	// HTML tags must be stripped
	if strings.Contains(out, "<html>") || strings.Contains(out, "<body>") {
		t.Errorf("property [3b]: raw HTML tags found in output: %s", out)
	}
}
