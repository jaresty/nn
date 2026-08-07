package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchWebCommandRegistered(t *testing.T) {
	state := &rootState{}
	cmd := newSearchWebCmd(state)
	if cmd.Use != "search-web <query>" {
		t.Fatalf("expected Use=search-web <query>, got %q", cmd.Use)
	}
}

func TestDDGExtractURLs(t *testing.T) {
	// property [3a]: extractDDGURLs parses the DDG HTML response and returns result URLs
	ddgHTML := `<html><body>
<div class="results">
  <div class="result">
    <a class="result__a" href="https://example.com/article-one">Article One</a>
  </div>
  <div class="result">
    <a class="result__a" href="https://example.org/article-two">Article Two</a>
  </div>
</div>
</body></html>`

	urls := extractDDGURLs(ddgHTML, 5)
	if len(urls) == 0 {
		t.Fatal("property [3a]: extractDDGURLs returned no URLs from DDG HTML")
	}
	found := false
	for _, u := range urls {
		if u == "https://example.com/article-one" {
			found = true
		}
	}
	if !found {
		t.Errorf("property [3a]: expected https://example.com/article-one in results, got %v", urls)
	}
}

func TestDDGExtractURLs_RedirectFormat(t *testing.T) {
	// property [3c]: extractDDGURLs decodes DDG uddg= redirect links to actual destination URLs.
	// DDG wraps results as: //duckduckgo.com/l/?uddg=<url-encoded-target>&rut=...
	ddgHTML := `<html><body>
<a href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgithub.com%2Flenaxia%2Fbm25-golang&amp;rut=abc123">BM25 Go</a>
<a href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fpkg.go.dev%2Fgithub.com%2Fcrawlab-team%2Fbm25&amp;rut=def456">BM25 pkg</a>
</body></html>`

	urls := extractDDGURLs(ddgHTML, 5)
	if len(urls) == 0 {
		t.Fatal("property [3c]: extractDDGURLs returned no URLs from DDG redirect-format HTML")
	}
	found := false
	for _, u := range urls {
		if u == "https://github.com/lenaxia/bm25-golang" {
			found = true
		}
	}
	if !found {
		t.Errorf("property [3c]: expected decoded URL https://github.com/lenaxia/bm25-golang, got %v", urls)
	}
}

func TestSearchWebUserAgentSent(t *testing.T) {
	// property [3d]: runSearchWeb sends a User-Agent header to DDG — without it DDG returns no results.
	var receivedUA string
	ddgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body></body></html>`))
	}))
	defer ddgSrv.Close()

	_ = runSearchWeb("test", 1, ddgSrv.URL+"?q=%s", nil, nil, nil, false)

	if receivedUA == "" {
		t.Fatal("property [3d]: no User-Agent header sent to DDG endpoint")
	}
}

func TestSearchWebPerResultUserAgent(t *testing.T) {
	// property [1b]: per-result page fetches also send the nn User-Agent
	var resultUA string
	resultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resultUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><p>content</p></body></html>`))
	}))
	defer resultSrv.Close()

	ddgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><a href="%s">Result</a></body></html>`, resultSrv.URL)
	}))
	defer ddgSrv.Close()

	_ = runSearchWeb("test", 1, ddgSrv.URL+"?q=%s", nil, nil, nil, false)

	if resultUA == "" {
		t.Fatal("property [1b]: per-result fetch sent no User-Agent header")
	}
	if resultUA == "Go-http-client/1.1" || resultUA == "Go-http-client/2.0" {
		t.Fatalf("property [1b]: per-result fetch sent default Go UA %q", resultUA)
	}
}

func TestSearchWebCommandRunsWithMockDDG(t *testing.T) {
	// property [3a]: search-web hits DDG endpoint when given a query
	var ddgCalled bool
	ddgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ddgCalled = true
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body></body></html>`))
	}))
	defer ddgSrv.Close()

	var resultSrv *httptest.Server
	resultSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><p>result content</p></body></html>`))
	}))
	defer resultSrv.Close()
	_ = resultSrv

	err := runSearchWeb("test query", 3, ddgSrv.URL+"?q=%s", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("runSearchWeb: %v", err)
	}
	if !ddgCalled {
		t.Fatal("property [3a]: DDG endpoint was not called")
	}
}
