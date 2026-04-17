package note

import (
	"math"
	"strings"
)

// BM25 parameters.
const (
	bm25K1 = 1.5
	bm25B  = 0.75
)

// BM25Score computes BM25 relevance scores for a set of notes against a query.
// Returns a map from note ID to score. Notes with score 0 are not included.
// Title tokens are weighted by repeating them titleWeight times in the document.
const titleWeight = 5

// BM25Scores returns BM25 scores for each note against the query terms.
// Only notes matching at least one query term are included.
func BM25Scores(notes []*Note, query string) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	// Build per-note token frequency maps (title weighted).
	type docInfo struct {
		tf  map[string]int
		len int
	}
	docs := make([]docInfo, len(notes))
	totalLen := 0
	for i, n := range notes {
		tf := make(map[string]int)
		titleTokens := tokenize(n.Title)
		bodyTokens := tokenize(n.Body)
		for _, t := range titleTokens {
			tf[t] += titleWeight
		}
		for _, t := range bodyTokens {
			tf[t]++
		}
		dlen := len(titleTokens)*titleWeight + len(bodyTokens)
		docs[i] = docInfo{tf: tf, len: dlen}
		totalLen += dlen
	}

	N := float64(len(notes))
	avgdl := float64(totalLen) / math.Max(N, 1)

	// IDF per term.
	idf := make(map[string]float64, len(terms))
	for _, term := range terms {
		df := 0
		for _, d := range docs {
			if d.tf[term] > 0 {
				df++
			}
		}
		idf[term] = math.Log((N-float64(df)+0.5)/(float64(df)+0.5) + 1)
	}

	scores := make(map[string]float64)
	for i, n := range notes {
		d := docs[i]
		score := 0.0
		for _, term := range terms {
			tf := float64(d.tf[term])
			if tf == 0 {
				continue
			}
			dl := float64(d.len)
			score += idf[term] * (tf * (bm25K1 + 1)) /
				(tf + bm25K1*(1-bm25B+bm25B*dl/avgdl))
		}
		if score > 0 {
			scores[n.ID] = score
		}
	}
	return scores
}

// tokenize splits text into lowercase tokens.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	var tokens []string
	for _, word := range strings.FieldsFunc(s, func(r rune) bool {
		return !('a' <= r && r <= 'z') && !('0' <= r && r <= '9')
	}) {
		if len(word) > 1 {
			tokens = append(tokens, word)
		}
	}
	return tokens
}
