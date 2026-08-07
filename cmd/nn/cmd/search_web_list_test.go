package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchWebListFlag(t *testing.T) {
	// --list prints one URL per line and does not fetch result pages
	var resultFetched bool
	resultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resultFetched = true
		fmt.Fprintln(w, "content")
	}))
	defer resultSrv.Close()

	ddgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s/page">Result One</a><a href="%s/other">Result Two</a></body></html>`,
			resultSrv.URL, resultSrv.URL)
	}))
	defer ddgSrv.Close()

	var stdout bytes.Buffer
	err := runSearchWeb("test", 5, ddgSrv.URL+"?q=%s", &stdout, nil, nil, true)
	if err != nil {
		t.Fatalf("runSearchWeb --list: %v", err)
	}

	if resultFetched {
		t.Error("--list must not fetch result pages")
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("--list produced no output")
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "http") {
			t.Errorf("--list output line is not a URL: %q", line)
		}
	}
}
