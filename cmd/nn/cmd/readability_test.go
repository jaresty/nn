package cmd

import (
	"regexp"
	"strings"
	"testing"
)

var reRawTag = regexp.MustCompile(`<[a-zA-Z]`)

func TestExtractReadableNonEmpty(t *testing.T) {
	// property [1a]: extractReadable returns non-empty text for well-formed article HTML
	html := `<html><head><title>Test Article</title></head><body>
<nav>Site nav irrelevant content here</nav>
<article>
<h1>The Main Article Title</h1>
<p>This is the first paragraph of the article with substantial content about the topic.</p>
<p>This is the second paragraph continuing the discussion with more detail and explanation.</p>
</article>
<footer>Footer content</footer>
</body></html>`

	result := extractReadable(html, "https://example.com/article")
	if strings.TrimSpace(result) == "" {
		t.Fatal("property [1a]: extractReadable returned empty string for well-formed article HTML")
	}
}

func TestExtractReadableNoRawTags(t *testing.T) {
	// property [1b]: extractReadable result contains no raw HTML tags
	html := `<html><body><article><p>Hello <b>world</b>, this is content.</p></article></body></html>`

	result := extractReadable(html, "https://example.com/page")
	if reRawTag.MatchString(result) {
		preview := result
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Errorf("property [1b]: extractReadable result contains raw HTML tags: %q", preview)
	}
}

func TestExtractReadableFallback(t *testing.T) {
	// property [2]: when readability returns empty, falls back to htmlToText
	// Minimal HTML that readability will reject (no article-like structure)
	bare := `<html><body><p>x</p></body></html>`
	result := extractReadable(bare, "https://example.com/")
	if strings.TrimSpace(result) == "" {
		t.Fatal("property [2]: fallback htmlToText also returned empty — at least one path must return content")
	}
	if reRawTag.MatchString(result) {
		t.Errorf("property [2]: fallback result contains raw HTML tags: %q", result)
	}
}

