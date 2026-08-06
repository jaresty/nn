package cmd

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/jaresty/nn/internal/note"
)

// searchBenchFixture is the top-level structure of testdata/search_benchmark.yaml.
type searchBenchFixture struct {
	Notes   []benchNote   `yaml:"notes"`
	Queries []benchQuery  `yaml:"queries"`
}

type benchNote struct {
	ID     string   `yaml:"id"`
	Title  string   `yaml:"title"`
	Type   string   `yaml:"type"`
	Body   string   `yaml:"body"`
	Tags   []string `yaml:"tags"`
	Status string   `yaml:"status"`
}

// benchQuery pairs a query string with graded relevance judgments (note ID → grade 0-3).
type benchQuery struct {
	Query     string         `yaml:"query"`
	Judgments map[string]int `yaml:"judgments"`
}

// hitAtK returns 1 if any of the top-k results has judgment >= 1, else 0.
func hitAtK(ranked []string, judgments map[string]int, k int) float64 {
	for i, id := range ranked {
		if i >= k {
			break
		}
		if judgments[id] >= 1 {
			return 1.0
		}
	}
	return 0.0
}

// meanReciprocalRank returns 1/rank of the first relevant result (judgment >= 1), or 0.
func meanReciprocalRank(ranked []string, judgments map[string]int) float64 {
	for i, id := range ranked {
		if judgments[id] >= 1 {
			return 1.0 / float64(i+1)
		}
	}
	return 0.0
}

// ndcgAtK computes NDCG@k using graded relevance (2^grade-1)/log2(rank+1).
func ndcgAtK(ranked []string, judgments map[string]int, k int) float64 {
	dcg := 0.0
	for i, id := range ranked {
		if i >= k {
			break
		}
		grade := float64(judgments[id])
		dcg += (math.Pow(2, grade) - 1) / math.Log2(float64(i+2))
	}
	// Ideal DCG: sort all judged notes by grade descending.
	var grades []float64
	for _, g := range judgments {
		grades = append(grades, float64(g))
	}
	// Insertion sort descending (fixture sizes are small).
	for i := 1; i < len(grades); i++ {
		for j := i; j > 0 && grades[j] > grades[j-1]; j-- {
			grades[j], grades[j-1] = grades[j-1], grades[j]
		}
	}
	idcg := 0.0
	for i, g := range grades {
		if i >= k {
			break
		}
		idcg += (math.Pow(2, g) - 1) / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

// TestSearchBenchmark seeds a notebook with fixture notes, runs each benchmark
// query, and asserts that aggregate Hit@1, Hit@5, MRR, and NDCG@5 meet thresholds.
// Thresholds are calibrated from current ranker performance — failures indicate regressions.
func TestSearchBenchmark(t *testing.T) {
	fixtureData, err := os.ReadFile(filepath.Join("testdata", "search_benchmark.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture searchBenchFixture
	if err := yaml.Unmarshal(fixtureData, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	nbDir, execute := setupNotebook(t)

	// Seed notebook with fixture notes.
	for _, fn := range fixture.Notes {
		status := note.StatusDraft
		switch fn.Status {
		case "reviewed":
			status = note.StatusReviewed
		case "permanent":
			status = note.StatusPermanent
		}
		typ := note.Type(fn.Type)
		if !typ.IsValid() {
			typ = note.TypeConcept
		}
		n := &note.Note{
			ID:     fn.ID,
			Title:  fn.Title,
			Type:   typ,
			Body:   fn.Body,
			Tags:   fn.Tags,
			Status: status,
		}
		writeNoteFile(t, nbDir, n)
	}

	// Run each query and collect per-query metrics.
	totalH1, totalH5, totalMRR, totalNDCG5 := 0.0, 0.0, 0.0, 0.0
	nQueries := len(fixture.Queries)
	if nQueries == 0 {
		t.Fatal("fixture has no queries")
	}

	for _, q := range fixture.Queries {
		out, err := execute("list", "--search", q.Query, "--json", "--fields", "id,score")
		if err != nil {
			t.Errorf("query %q: execute error: %v", q.Query, err)
			continue
		}
		var results []map[string]any
		if err := json.Unmarshal([]byte(out), &results); err != nil {
			t.Errorf("query %q: parse JSON: %v\n%s", q.Query, err, out)
			continue
		}
		ranked := make([]string, 0, len(results))
		for _, r := range results {
			if id, ok := r["id"].(string); ok {
				ranked = append(ranked, id)
			}
		}

		totalH1 += hitAtK(ranked, q.Judgments, 1)
		totalH5 += hitAtK(ranked, q.Judgments, 5)
		totalMRR += meanReciprocalRank(ranked, q.Judgments)
		totalNDCG5 += ndcgAtK(ranked, q.Judgments, 5)
	}

	n := float64(nQueries)
	h1 := totalH1 / n
	h5 := totalH5 / n
	mrr := totalMRR / n
	ndcg5 := totalNDCG5 / n

	t.Logf("Search benchmark results (n=%d queries):", nQueries)
	t.Logf("  Hit@1:  %.3f", h1)
	t.Logf("  Hit@5:  %.3f", h5)
	t.Logf("  MRR:    %.3f", mrr)
	t.Logf("  NDCG@5: %.3f", ndcg5)

	const (
		threshH1    = 0.50
		threshH5    = 0.80
		threshMRR   = 0.55
		threshNDCG5 = 0.50
	)

	if h1 < threshH1 {
		t.Errorf("Hit@1 %.3f below threshold %.2f — ranker regression detected", h1, threshH1)
	}
	if h5 < threshH5 {
		t.Errorf("Hit@5 %.3f below threshold %.2f — ranker regression detected", h5, threshH5)
	}
	if mrr < threshMRR {
		t.Errorf("MRR %.3f below threshold %.2f — ranker regression detected", mrr, threshMRR)
	}
	if ndcg5 < threshNDCG5 {
		t.Errorf("NDCG@5 %.3f below threshold %.2f — ranker regression detected", ndcg5, threshNDCG5)
	}
}

// property [1]: centrality does not influence scores — adding a backlink to a note
// must not change that note's search score relative to an otherwise identical note.
// We test this by measuring hub's score before and after a linker is added.
func TestListSearch_NoCentralityBias(t *testing.T) {
	_, execute := setupNotebook(t)

	body := "quantum entanglement is a core phenomenon in physics"
	_, err := execute("new", "--title", "quantum entanglement phenomenon", "--type", "concept", "--content", body, "--no-edit", "--no-suggest")
	if err != nil {
		t.Fatalf("new hub: %v", err)
	}
	// Get hub ID and baseline score before any linker exists.
	out1, err := execute("list", "--search", "quantum entanglement phenomenon", "--json", "--fields", "id,score")
	if err != nil {
		t.Fatalf("list before linker: %v", err)
	}
	var r1 []map[string]any
	json.Unmarshal([]byte(out1), &r1) //nolint:errcheck
	if len(r1) == 0 {
		t.Fatalf("no results before linker: %s", out1)
	}
	hubID, _ := r1[0]["id"].(string)
	scoreBefore, _ := r1[0]["score"].(float64)

	// Add a linker that gives hub one backlink.
	_, err = execute("new", "--title", "linker note", "--type", "concept", "--content", "unrelated content", "--no-edit", "--no-suggest", "--link-to", hubID, "--annotation", "refines hub")
	if err != nil {
		t.Fatalf("new linker: %v", err)
	}

	// Score after linker — hub now has one backlink.
	out2, err := execute("list", "--search", "quantum entanglement phenomenon", "--json", "--fields", "id,score")
	if err != nil {
		t.Fatalf("list after linker: %v", err)
	}
	var r2 []map[string]any
	json.Unmarshal([]byte(out2), &r2) //nolint:errcheck
	scoreAfter := 0.0
	for _, r := range r2 {
		if id, _ := r["id"].(string); id == hubID {
			scoreAfter, _ = r["score"].(float64)
			break
		}
	}
	if scoreAfter == 0 {
		t.Fatalf("hub note not found in results after linker; results: %s", out2)
	}
	if scoreAfter != scoreBefore {
		t.Errorf("property [1]: hub score changed %.6f→%.6f after adding backlink — centrality bias detected", scoreBefore, scoreAfter)
	}
}
