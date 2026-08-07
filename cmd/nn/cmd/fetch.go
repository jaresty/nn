package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

const nnUserAgent = "Mozilla/5.0 (compatible; nn/1.0; +https://github.com/jaresty/nn)"

var (
	reHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	reHTMLEntity  = regexp.MustCompile(`&[a-zA-Z]+;|&#[0-9]+;`)
	reMultiSpace  = regexp.MustCompile(`[ \t]+`)
	reMultiNewline = regexp.MustCompile(`\n{3,}`)
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
)

func newFetchCmd(state *rootState) *cobra.Command {
	var capture bool

	cmd := &cobra.Command{
		Use:   "fetch <url>",
		Short: "Fetch a URL, strip HTML, and print plaintext with related notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(args[0], capture, cmd.OutOrStdout(), cmd.ErrOrStderr(), state)
		},
	}
	cmd.Flags().BoolVar(&capture, "capture", false, "Create a note from the fetched content")
	return cmd
}

func runFetch(rawURL string, capture bool, stdout, stderr io.Writer, state *rootState) error {
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return fmt.Errorf("fetch: invalid URL %q: %w", rawURL, err)
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("fetch: URL must begin with http:// or https://")
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("fetch: build request: %w", err)
	}
	req.Header.Set("User-Agent", nnUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fetch: read body: %w", err)
	}

	plain := htmlToText(string(body))
	fmt.Fprintln(stdout, plain)

	if state == nil || state.backend == nil {
		return nil
	}

	notes, err := state.backend.List()
	if err != nil {
		return nil
	}

	scores := RankedByQuery(notes, notes, plain, state.notebookDir)

	type scored struct {
		n     *note.Note
		score float64
	}
	var matches []scored
	for _, n := range notes {
		if s := scores[n.ID]; s > 0 {
			matches = append(matches, scored{n, s})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})
	if len(matches) > 5 {
		matches = matches[:5]
	}
	if len(matches) > 0 {
		fmt.Fprintln(stdout, "\n## Related notes")
		for _, m := range matches {
			fmt.Fprintf(stdout, "- %s — %s %s\n", m.n.ID, m.n.Title, scoreLabel(m.score))
		}
	}

	if capture {
		title := rawURL
		if t := extractTitle(string(body)); t != "" {
			title = t
		}
		now := time.Now().UTC()
		n := &note.Note{
			ID:       note.GenerateID(),
			Title:    title,
			Type:     note.TypeObservation,
			Status:   note.StatusDraft,
			Created:  now,
			Modified: now,
			Body:     plain,
		}
		if writeErr := state.backend.Write(n); writeErr != nil {
			fmt.Fprintf(stderr, "fetch: --capture: %v\n", writeErr)
		} else {
			fmt.Fprintf(stderr, "fetch: captured as %s\n", n.ID)
		}
	}

	return nil
}

// htmlToText strips HTML tags and decodes common entities, returning readable plaintext.
func htmlToText(html string) string {
	// Remove script and style blocks first.
	text := reScriptStyle.ReplaceAllString(html, "")
	// Strip all remaining tags.
	text = reHTMLTag.ReplaceAllString(text, " ")
	// Decode common HTML entities.
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = reHTMLEntity.ReplaceAllString(text, " ")
	// Normalize whitespace.
	text = reMultiSpace.ReplaceAllString(text, " ")
	// Collapse blank lines.
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	text = strings.Join(lines, "\n")
	text = reMultiNewline.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// extractTitle returns the contents of the first <title> tag, or empty string.
func extractTitle(html string) string {
	lower := strings.ToLower(html)
	start := strings.Index(lower, "<title>")
	if start < 0 {
		return ""
	}
	start += len("<title>")
	end := strings.Index(lower[start:], "</title>")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(html[start : start+end])
}
