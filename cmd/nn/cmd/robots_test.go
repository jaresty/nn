package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRobotsAllowed(t *testing.T) {
	// property [2a]: robotsAllowed fetches and parses robots.txt for the host
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprintln(w, "User-agent: *\nDisallow: /private/")
			return
		}
		fmt.Fprintln(w, "content")
	}))
	defer srv.Close()

	if !robotsAllowed(srv.URL+"/public/page", nnUserAgent) {
		t.Error("property [2a]: /public/page should be allowed but was not")
	}
	if robotsAllowed(srv.URL+"/private/secret", nnUserAgent) {
		t.Error("property [2a]: /private/secret should be disallowed but was allowed")
	}
}

func TestRobotsAllowedOnFetchError(t *testing.T) {
	// property [2a]: if robots.txt cannot be fetched, allow the URL (fail open)
	if !robotsAllowed("http://127.0.0.1:1/page", nnUserAgent) {
		t.Error("property [2a]: should allow URL when robots.txt fetch fails")
	}
}

func TestSearchWebSkipsDisallowedURL(t *testing.T) {
	// property [2b]: runSearchWeb skips URLs disallowed by robots.txt
	var resultFetched bool
	resultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprintln(w, "User-agent: *\nDisallow: /")
			return
		}
		resultFetched = true
		fmt.Fprintln(w, "content")
	}))
	defer resultSrv.Close()

	ddgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><body><a href="%s/page">Result</a></body></html>`, resultSrv.URL)
	}))
	defer ddgSrv.Close()

	_ = runSearchWeb("test", 1, ddgSrv.URL+"?q=%s", nil, nil, nil)

	if resultFetched {
		t.Error("property [2b]: result page was fetched despite robots.txt Disallow: /")
	}
}
