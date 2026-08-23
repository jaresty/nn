package cmd

import (
	"strings"
	"testing"
)

// The nn-cli-reference protocol must route relational/synthesis questions from
// ranked search to graph inspection. Guards the rendered virtual protocol.
func TestSearchToGraphRoutingDocumented(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("nn show virtual-nn-cli-reference: %v", err)
	}

	// property [1]: don't stop at ranked search for relational questions.
	if !strings.Contains(out, "Search-to-graph routing") {
		t.Errorf("protocol missing 'Search-to-graph routing' section; got:\n%s", out)
	}
	if !strings.Contains(out, "Do not stop at ranked") {
		t.Errorf("protocol does not direct the agent past ranked search results; got:\n%s", out)
	}

	// property [2]: names the default neighborhood command (zoned, depth-1).
	if !strings.Contains(out, "nn graph show --focus <id> --depth 1 --direction both") {
		t.Errorf("protocol missing default graph-show command; got:\n%s", out)
	}

	// property [3]: BM25 similarity is not evidence of a relationship.
	if !strings.Contains(out, "not treat BM25 similarity as evidence") {
		t.Errorf("protocol missing the BM25-is-not-a-relationship caveat; got:\n%s", out)
	}

	// property [4]: bridges are referenced as a query-conditioned structural projection.
	if !strings.Contains(out, "nn graph bridges --search \"<query>\" --format json") {
		t.Errorf("protocol missing query-conditioned nn graph bridges reference; got:\n%s", out)
	}

	// property [5]: the default neighborhood is inspected with --zones so the
	// direct neighbors are grouped by relationship direction. Asserted on the
	// default-neighborhood command specifically (not just anywhere in the doc).
	if !strings.Contains(out, "--depth 1 --direction both --zones --format text") {
		t.Errorf("protocol zoned default-neighborhood command should use --depth 1 --zones; got:\n%s", out)
	}
}
