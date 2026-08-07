package cmd

import (
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

	err := runSearchWeb("test query", 3, ddgSrv.URL+"?q=%s", nil, nil, nil)
	if err != nil {
		t.Fatalf("runSearchWeb: %v", err)
	}
	if !ddgCalled {
		t.Fatal("property [3a]: DDG endpoint was not called")
	}
}
