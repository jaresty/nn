package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

const ddgEndpoint = "https://html.duckduckgo.com/html/?q=%s"

var reAnchorHref = regexp.MustCompile(`(?i)<a[^>]+href=["']([^"']+)["'][^>]*>`)

func newSearchWebCmd(state *rootState) *cobra.Command {
	var results int
	var listOnly bool
	var queries []string
	var parallel int

	cmd := &cobra.Command{
		Use:   "search-web <query>",
		Short: "Search the web via DuckDuckGo and annotate results with related notes",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && len(queries) == 0 {
				return fmt.Errorf("requires a positional query or at least one --query")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			allQueries := make([]string, 0, len(queries)+1)
			if len(args) > 0 {
				allQueries = append(allQueries, strings.Join(args, " "))
			}
			allQueries = append(allQueries, queries...)
			return runSearchWebBatch(allQueries, results, parallel, ddgEndpoint, cmd.OutOrStdout(), cmd.ErrOrStderr(), state, listOnly)
		},
	}
	cmd.Flags().IntVar(&results, "results", 3, "Number of results to fetch and display per query")
	cmd.Flags().BoolVar(&listOnly, "list", false, "Print result URLs only, one per line (no fetching)")
	cmd.Flags().StringArrayVar(&queries, "query", nil, "Search query; repeat for parallel batch search")
	cmd.Flags().IntVar(&parallel, "parallel", 4, "Maximum concurrent queries")
	return cmd
}

func prepareSearchWebCorpus(notesCorpus []*note.Note, state *rootState) preparedCorpus {
	if state == nil {
		return prepareCorpusUncached(notesCorpus)
	}
	return prepareCorpus(notesCorpus, state.notebookDir)
}

func runSearchWebBatch(queries []string, maxResults, parallel int, endpointFmt string, stdout, stderr io.Writer, state *rootState, listOnly bool) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if parallel < 1 {
		return fmt.Errorf("search-web: parallel must be at least 1")
	}

	var notesCorpus []*note.Note
	if state != nil && state.backend != nil {
		notesCorpus, _ = state.backend.List()
	}
	prepared := prepareSearchWebCorpus(notesCorpus, state)
	var rankMu sync.Mutex

	outputs := make([]bytes.Buffer, len(queries))
	errs := make([]error, len(queries))
	jobs := make(chan int)
	var workers sync.WaitGroup
	workerCount := min(parallel, len(queries))
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for i := range jobs {
				errs[i] = runSearchWebQuery(queries[i], maxResults, endpointFmt, &outputs[i], notesCorpus, prepared, &rankMu, listOnly)
			}
		}()
	}
	for i := range queries {
		jobs <- i
	}
	close(jobs)
	workers.Wait()

	for i, query := range queries {
		if len(queries) > 1 {
			fmt.Fprintf(stdout, "## Query %d: %s\n", i+1, query)
		}
		_, _ = io.Copy(stdout, &outputs[i])
		if errs[i] != nil {
			return fmt.Errorf("search-web query %q: %w", query, errs[i])
		}
	}
	return nil
}

func runSearchWeb(query string, maxResults int, endpointFmt string, stdout, stderr io.Writer, state *rootState, listOnly bool) error {
	var notesCorpus []*note.Note
	if state != nil && state.backend != nil {
		notesCorpus, _ = state.backend.List()
	}
	prepared := prepareSearchWebCorpus(notesCorpus, state)
	var rankMu sync.Mutex
	return runSearchWebQuery(query, maxResults, endpointFmt, stdout, notesCorpus, prepared, &rankMu, listOnly)
}

func runSearchWebQuery(query string, maxResults int, endpointFmt string, stdout io.Writer, notesCorpus []*note.Note, prepared preparedCorpus, rankMu *sync.Mutex, listOnly bool) error {
	if stdout == nil {
		stdout = io.Discard
	}
	searchURL := fmt.Sprintf(endpointFmt, url.QueryEscape(query))
	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return fmt.Errorf("search-web: build request: %w", err)
	}
	req.Header.Set("User-Agent", nnUserAgent)
	resp, err := http.DefaultClient.Do(req) //nolint:gosec
	if err != nil {
		return fmt.Errorf("search-web: DDG request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("search-web: read DDG response: %w", err)
	}

	urls := extractDDGURLs(string(body), maxResults)
	if len(urls) == 0 {
		fmt.Fprintln(stdout, "(no results found)")
		return nil
	}

	if listOnly {
		for _, u := range urls {
			fmt.Fprintln(stdout, u)
		}
		return nil
	}

	for i, u := range urls {
		if !robotsAllowed(u, nnUserAgent) {
			continue
		}
		fmt.Fprintf(stdout, "\n## Result %d: %s\n\n", i+1, u)

		pageReq, pageReqErr := http.NewRequest(http.MethodGet, u, nil)
		if pageReqErr != nil {
			fmt.Fprintf(stdout, "(request error: %v)\n", pageReqErr)
			continue
		}
		pageReq.Header.Set("User-Agent", nnUserAgent)
		pageResp, pageErr := http.DefaultClient.Do(pageReq)
		if pageErr != nil {
			fmt.Fprintf(stdout, "(fetch error: %v)\n", pageErr)
			continue
		}
		pageBody, readErr := io.ReadAll(pageResp.Body)
		pageResp.Body.Close()
		if readErr != nil {
			fmt.Fprintf(stdout, "(read error: %v)\n", readErr)
			continue
		}

		plain := extractReadable(string(pageBody), u)
		// Print a preview (first 500 chars).
		preview := plain
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Fprintln(stdout, preview)

		if len(notesCorpus) == 0 {
			continue
		}
		rankMu.Lock()
		scores := prepared.rankedByQuery(notesCorpus, plain)
		rankMu.Unlock()
		type scored struct {
			n     *note.Note
			score float64
		}
		var matches []scored
		for _, n := range notesCorpus {
			if s := scores[n.ID]; s > 0 {
				matches = append(matches, scored{n, s})
			}
		}
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
		if len(matches) > 3 {
			matches = matches[:3]
		}
		if len(matches) > 0 {
			fmt.Fprintln(stdout, "\n## Related notes")
			for _, m := range matches {
				fmt.Fprintf(stdout, "- %s — %s %s\n", m.n.ID, m.n.Title, scoreLabel(m.score))
			}
		}
	}
	return nil
}

// extractDDGURLs parses URLs from DuckDuckGo HTML result page.
// DDG wraps result URLs in redirect links of the form:
//
//	//duckduckgo.com/l/?uddg=<url-encoded-target>&rut=...
//
// We decode the uddg= parameter to extract the actual destination URL.
func extractDDGURLs(html string, max int) []string {
	var urls []string
	seen := make(map[string]bool)

	matches := reAnchorHref.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		href := m[1]

		// Handle DDG redirect links: extract uddg= parameter.
		if strings.Contains(href, "uddg=") {
			parsed, err := url.Parse(strings.TrimPrefix(strings.TrimPrefix(href, "//"), "https:"))
			if err == nil {
				if target := parsed.Query().Get("uddg"); target != "" {
					// uddg value is itself URL-encoded.
					decoded, decErr := url.QueryUnescape(target)
					if decErr == nil {
						href = decoded
					}
				}
			}
		}

		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
			continue
		}
		if strings.Contains(href, "duckduckgo.com") {
			continue
		}
		if seen[href] {
			continue
		}
		seen[href] = true
		urls = append(urls, href)
		if len(urls) >= max {
			break
		}
	}
	return urls
}
