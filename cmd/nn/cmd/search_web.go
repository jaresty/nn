package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

const ddgEndpoint = "https://html.duckduckgo.com/html/?q=%s"

var reAnchorHref = regexp.MustCompile(`(?i)<a[^>]+href=["']([^"']+)["'][^>]*>`)

func newSearchWebCmd(state *rootState) *cobra.Command {
	var results int

	cmd := &cobra.Command{
		Use:   "search-web <query>",
		Short: "Search the web via DuckDuckGo and annotate results with related notes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runSearchWeb(query, results, ddgEndpoint, cmd.OutOrStdout(), cmd.ErrOrStderr(), state)
		},
	}
	cmd.Flags().IntVar(&results, "results", 3, "Number of results to fetch and display")
	return cmd
}

func runSearchWeb(query string, maxResults int, endpointFmt string, stdout, stderr io.Writer, state *rootState) error {
	if stdout == nil {
		stdout = io.Discard
	}
	searchURL := fmt.Sprintf(endpointFmt, url.QueryEscape(query))
	resp, err := http.Get(searchURL) //nolint:gosec
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

	var notesCorpus []*note.Note
	if state != nil && state.backend != nil {
		notesCorpus, _ = state.backend.List()
	}

	for i, u := range urls {
		fmt.Fprintf(stdout, "\n## Result %d: %s\n\n", i+1, u)

		pageResp, pageErr := http.Get(u) //nolint:gosec
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

		plain := htmlToText(string(pageBody))
		// Print a preview (first 500 chars).
		preview := plain
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Fprintln(stdout, preview)

		if len(notesCorpus) == 0 {
			continue
		}
		scores := RankedByQuery(notesCorpus, notesCorpus, plain, "")
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
// DDG HTML results use anchor tags with class "result__a" pointing directly to result URLs.
func extractDDGURLs(html string, max int) []string {
	var urls []string
	seen := make(map[string]bool)

	matches := reAnchorHref.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		href := m[1]
		// Skip DDG-internal links and ad redirects.
		if strings.HasPrefix(href, "/") || strings.Contains(href, "duckduckgo.com") {
			continue
		}
		if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
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
