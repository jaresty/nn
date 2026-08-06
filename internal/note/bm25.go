package note

import (
	"math"
	"strings"

	"github.com/kljensen/snowball/english"
)

// BM25 parameters.
const (
	bm25K1 = 1.5
	bm25B  = 0.75
)

// BM25Score computes BM25 relevance scores for a set of notes against a query.
// Returns a map from note ID to score. Notes with score 0 are not included.
// Title tokens are weighted by repeating them titleWeight times in the document.
// Tag tokens are weighted by tagWeight — tags are curated intent signals.
const titleWeight = 5
const tagWeight = 3

// inboundWeight is the fractional body-token weight applied to inbound annotation tokens.
const inboundWeight = 0.5

// propagationFactor is the fraction of a note's score passed to its 1-hop neighbors.
const propagationFactor = 0.15

// inboundWeightForLinkType returns the TF weight for an inbound annotation by link type.
func inboundWeightForLinkType(linkType string) float64 {
	switch linkType {
	case "governs", "supports", "source-of":
		return 1.0
	case "refines", "extends":
		return 0.75
	case "questions", "contradicts":
		return 0.25
	default:
		return inboundWeight
	}
}

// TypedAnnotation pairs an annotation string with the link type that produced it.
type TypedAnnotation struct {
	Text     string
	LinkType string
}

type docInfo struct {
	tf  map[string]float64
	len float64
}

// buildDocs constructs per-note TF maps and document lengths using flat inbound weight.
func buildDocs(notes []*Note, inbound map[string][]string) ([]docInfo, float64) {
	docs := make([]docInfo, len(notes))
	totalLen := 0.0
	for i, n := range notes {
		tf := make(map[string]float64)
		titleTokens := tokenize(n.Title)
		bodyTokens := tokenize(n.Body)
		for _, t := range titleTokens {
			tf[t] += titleWeight
		}
		for _, t := range bodyTokens {
			tf[t]++
		}
		for _, tag := range n.Tags {
			for _, t := range tokenize(tag) {
				tf[t] += tagWeight
			}
		}
		inboundLen := 0.0
		for _, ann := range inbound[n.ID] {
			for _, t := range tokenize(ann) {
				tf[t] += inboundWeight
				inboundLen += inboundWeight
			}
		}
		tagLen := float64(len(n.Tags) * tagWeight)
		dlen := float64(len(titleTokens)*titleWeight+len(bodyTokens)) + tagLen + inboundLen
		docs[i] = docInfo{tf: tf, len: dlen}
		totalLen += dlen
	}
	return docs, totalLen
}

// buildDocsTyped constructs per-note TF maps using link-type-weighted inbound annotations.
func buildDocsTyped(notes []*Note, inbound map[string][]TypedAnnotation) ([]docInfo, float64) {
	docs := make([]docInfo, len(notes))
	totalLen := 0.0
	for i, n := range notes {
		tf := make(map[string]float64)
		titleTokens := tokenize(n.Title)
		bodyTokens := tokenize(n.Body)
		for _, t := range titleTokens {
			tf[t] += titleWeight
		}
		for _, t := range bodyTokens {
			tf[t]++
		}
		for _, tag := range n.Tags {
			for _, t := range tokenize(tag) {
				tf[t] += tagWeight
			}
		}
		inboundLen := 0.0
		for _, ann := range inbound[n.ID] {
			w := inboundWeightForLinkType(ann.LinkType)
			for _, t := range tokenize(ann.Text) {
				tf[t] += w
				inboundLen += w
			}
		}
		tagLen := float64(len(n.Tags) * tagWeight)
		dlen := float64(len(titleTokens)*titleWeight+len(bodyTokens)) + tagLen + inboundLen
		docs[i] = docInfo{tf: tf, len: dlen}
		totalLen += dlen
	}
	return docs, totalLen
}

// scoreDocsWithIDF scores pre-built docs against pre-computed IDF.
func scoreDocsWithIDF(notes []*Note, docs []docInfo, totalLen float64, idf map[string]float64, terms []string) map[string]float64 {
	N := float64(len(docs))
	avgdl := totalLen / math.Max(N, 1)
	scores := make(map[string]float64)
	for i, n := range notes {
		d := docs[i]
		score := 0.0
		for _, term := range terms {
			tf := d.tf[term]
			if tf == 0 {
				continue
			}
			dl := d.len
			score += idf[term] * (tf * (bm25K1 + 1)) /
				(tf + bm25K1*(1-bm25B+bm25B*dl/avgdl))
		}
		if score > 0 {
			scores[n.ID] = score * statusMultiplier(n.Status)
		}
	}
	return scores
}

// BM25IDF computes inverse document frequency for the given terms over the corpus.
func BM25IDF(notes []*Note, terms []string) map[string]float64 {
	docs, _ := buildDocs(notes, nil)
	N := float64(len(notes))
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
	return idf
}

// BM25ScoresWithIDF scores candidates using a pre-computed IDF map.
// IDF should be computed over the full corpus via BM25IDF; candidates may be a subset.
func BM25ScoresWithIDF(candidates []*Note, idf map[string]float64, query string, inbound map[string][]string) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	docs, totalLen := buildDocs(candidates, inbound)
	return scoreDocsWithIDF(candidates, docs, totalLen, idf, terms)
}

// BM25Scores returns BM25 scores for each note against the query terms.
// inbound maps note ID to annotation strings from notes that link to it.
// IDF is computed over the same corpus as candidates (correct when corpus==candidates).
// For filtered candidate sets, prefer BM25IDF + BM25ScoresWithIDF.
func BM25Scores(notes []*Note, query string, inbound map[string][]string) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	idf := BM25IDF(notes, terms)
	return BM25ScoresWithIDF(notes, idf, query, inbound)
}

// BM25ScoresTyped scores notes using link-type-weighted inbound annotations.
func BM25ScoresTyped(notes []*Note, query string, inbound map[string][]TypedAnnotation) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	docs, totalLen := buildDocsTyped(notes, inbound)
	N := float64(len(notes))
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
	return scoreDocsWithIDF(notes, docs, totalLen, idf, terms)
}

// BM25ScoresWithPropagation scores notes then propagates scores 1 hop through links.
// links maps source note ID to slice of target note IDs.
func BM25ScoresWithPropagation(notes []*Note, query string, inbound map[string][]string, links map[string][]string) map[string]float64 {
	base := BM25Scores(notes, query, inbound)
	if base == nil {
		return nil
	}
	result := make(map[string]float64, len(base))
	for id, s := range base {
		result[id] = s
	}
	for srcID, targets := range links {
		srcScore := base[srcID]
		if srcScore == 0 {
			continue
		}
		boost := srcScore * propagationFactor
		for _, tgtID := range targets {
			result[tgtID] += boost
		}
	}
	return result
}

// rrfK is the RRF rank constant — standard value is 60.
const rrfK = 60.0

// FieldIDF holds per-field IDF maps, each computed over only that field's token corpus.
// Using field-specific IDF ensures BM25 scores within each field are self-consistent.
type FieldIDF struct {
	Title   map[string]float64
	Body    map[string]float64
	Tags    map[string]float64
	Inbound map[string]float64
}

// fieldIDF computes IDF for a set of token slices (one per document) over N total documents.
func fieldIDF(fieldTokens [][]string, N int, terms []string) map[string]float64 {
	idf := make(map[string]float64, len(terms))
	for _, term := range terms {
		df := 0
		for _, tokens := range fieldTokens {
			for _, t := range tokens {
				if t == term {
					df++
					break
				}
			}
		}
		idf[term] = math.Log((float64(N)-float64(df)+0.5)/(float64(df)+0.5) + 1)
	}
	return idf
}

// BM25FieldIDF computes per-field IDF maps for the given notes corpus and query terms.
// inbound maps note ID to annotation strings from notes that link to it.
func BM25FieldIDF(notes []*Note, inbound map[string][]string) FieldIDF {
	// Collect all corpus terms across all fields for full-corpus field IDFs.
	seen := make(map[string]struct{})
	titleField := make([][]string, len(notes))
	bodyField := make([][]string, len(notes))
	tagField := make([][]string, len(notes))
	inboundField := make([][]string, len(notes))
	for i, n := range notes {
		titleField[i] = tokenize(n.Title)
		bodyField[i] = tokenize(n.Body)
		var tagToks []string
		for _, tag := range n.Tags {
			tagToks = append(tagToks, tokenize(tag)...)
		}
		tagField[i] = tagToks
		var ibToks []string
		for _, ann := range inbound[n.ID] {
			ibToks = append(ibToks, tokenize(ann)...)
		}
		inboundField[i] = ibToks
		for _, t := range titleField[i] {
			seen[t] = struct{}{}
		}
		for _, t := range bodyField[i] {
			seen[t] = struct{}{}
		}
		for _, t := range tagField[i] {
			seen[t] = struct{}{}
		}
		for _, t := range inboundField[i] {
			seen[t] = struct{}{}
		}
	}
	allTerms := make([]string, 0, len(seen))
	for t := range seen {
		allTerms = append(allTerms, t)
	}
	N := len(notes)
	return FieldIDF{
		Title:   fieldIDF(titleField, N, allTerms),
		Body:    fieldIDF(bodyField, N, allTerms),
		Tags:    fieldIDF(tagField, N, allTerms),
		Inbound: fieldIDF(inboundField, N, allTerms),
	}
}

// BM25RRFPerField preserves the original four-field scorer for callers that
// intentionally use one slice as both the statistics corpus and candidate set.
func BM25RRFPerField(candidates []*Note, fidf FieldIDF, query string, inbound map[string][]string) map[string]float64 {
	return bm25RRFPerField(candidates, candidates, fidf, query, inbound, nil, float64(tagWeight), 1, 0)
}

// BM25RRFPerFieldForCorpus scores candidates while deriving document-length
// statistics and outbound query-term IDF from the full corpus.
func BM25RRFPerFieldForCorpus(corpus, candidates []*Note, fidf FieldIDF, query string, inbound, outbound map[string][]string) map[string]float64 {
	return bm25RRFPerField(corpus, candidates, fidf, query, inbound, outbound, 1, 0, 1)
}

func bm25RRFPerField(corpus, candidates []*Note, fidf FieldIDF, query string, inbound, outbound map[string][]string, tagFieldWeight, inboundFieldWeight, outboundFieldWeight float64) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	scoreField := func(getTokens func(*Note) []string, idf map[string]float64) []string {
		type fs struct {
			id    string
			score float64
		}
		totalLen := 0.0
		for _, n := range corpus {
			totalLen += float64(len(getTokens(n)))
		}
		avgdl := totalLen / math.Max(float64(len(corpus)), 1)
		type docF struct {
			tf  map[string]float64
			len float64
		}
		docs := make([]docF, len(candidates))
		for i, n := range candidates {
			tokens := getTokens(n)
			tf := make(map[string]float64, len(tokens))
			for _, t := range tokens {
				tf[t]++
			}
			docs[i] = docF{tf: tf, len: float64(len(tokens))}
		}
		var ranked []fs
		for i, n := range candidates {
			d := docs[i]
			score := 0.0
			for _, term := range terms {
				tf := d.tf[term]
				if tf == 0 {
					continue
				}
				score += idf[term] * (tf * (bm25K1 + 1)) /
					(tf + bm25K1*(1-bm25B+bm25B*d.len/avgdl))
			}
			if score > 0 {
				ranked = append(ranked, fs{id: n.ID, score: score})
			}
		}
		// Preserve the historical candidate-sequence-dependent permutation for
		// equal scores. Changing tie semantics is intentionally a separate decision.
		for i := 0; i < len(ranked)-1; i++ {
			for j := i + 1; j < len(ranked); j++ {
				if ranked[j].score > ranked[i].score {
					ranked[i], ranked[j] = ranked[j], ranked[i]
				}
			}
		}
		ids := make([]string, len(ranked))
		for i, r := range ranked {
			ids[i] = r.id
		}
		return ids
	}

	tagTokens := func(n *Note) []string {
		var all []string
		for _, tag := range n.Tags {
			all = append(all, tokenize(tag)...)
		}
		return all
	}
	inboundTokens := func(n *Note) []string {
		var all []string
		for _, ann := range inbound[n.ID] {
			all = append(all, tokenize(ann)...)
		}
		return all
	}
	outboundTokens := func(n *Note) []string {
		var all []string
		for _, ann := range outbound[n.ID] {
			all = append(all, tokenize(ann)...)
		}
		return all
	}

	type weightedField struct {
		weight float64
		ranked []string
	}
	fields := []weightedField{
		{titleWeight, scoreField(func(n *Note) []string { return tokenize(n.Title) }, fidf.Title)},
		{1, scoreField(func(n *Note) []string { return tokenize(n.Body) }, fidf.Body)},
		{tagFieldWeight, scoreField(tagTokens, fidf.Tags)},
	}
	if inboundFieldWeight > 0 {
		fields = append(fields, weightedField{inboundFieldWeight, scoreField(inboundTokens, fidf.Inbound)})
	}
	if outboundFieldWeight > 0 {
		outboundField := make([][]string, len(corpus))
		for i, n := range corpus {
			outboundField[i] = outboundTokens(n)
		}
		outboundIDF := fieldIDF(outboundField, len(corpus), terms)
		fields = append(fields, weightedField{outboundFieldWeight, scoreField(outboundTokens, outboundIDF)})
	}

	rrf := make(map[string]float64)
	for _, f := range fields {
		for rank, id := range f.ranked {
			rrf[id] += f.weight / (rrfK + float64(rank+1))
		}
	}
	if len(rrf) == 0 {
		return nil
	}
	return rrf
}

// BM25RRF scores notes using reciprocal rank fusion across per-field BM25 rankings.
// Each field (title, body, tags, inbound) is scored independently; the four rankings
// are fused: score(n) = Σ_f 1/(rrfK + rank_f(n)), where rank is 1-based and
// notes scoring 0 in a field are excluded from that field's ranking.
func BM25RRF(candidates []*Note, idf map[string]float64, query string, inbound map[string][]string) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}

	// scoreField builds a per-field BM25 ranking.
	// getTokens returns the raw tokens for that field (repetition encodes weight).
	scoreField := func(getTokens func(*Note) []string) []string {
		type fs struct {
			id    string
			score float64
		}
		N := float64(len(candidates))
		type docF struct {
			tf  map[string]float64
			len float64
		}
		docs := make([]docF, len(candidates))
		totalLen := 0.0
		for i, n := range candidates {
			tokens := getTokens(n)
			tf := make(map[string]float64, len(tokens))
			for _, t := range tokens {
				tf[t]++
			}
			dl := float64(len(tokens))
			docs[i] = docF{tf: tf, len: dl}
			totalLen += dl
		}
		avgdl := totalLen / math.Max(N, 1)
		var ranked []fs
		for i, n := range candidates {
			d := docs[i]
			score := 0.0
			for _, term := range terms {
				tf := d.tf[term]
				if tf == 0 {
					continue
				}
				score += idf[term] * (tf * (bm25K1 + 1)) /
					(tf + bm25K1*(1-bm25B+bm25B*d.len/avgdl))
			}
			if score > 0 {
				ranked = append(ranked, fs{id: n.ID, score: score})
			}
		}
		// Sort descending.
		for i := 0; i < len(ranked)-1; i++ {
			for j := i + 1; j < len(ranked); j++ {
				if ranked[j].score > ranked[i].score {
					ranked[i], ranked[j] = ranked[j], ranked[i]
				}
			}
		}
		ids := make([]string, len(ranked))
		for i, r := range ranked {
			ids[i] = r.id
		}
		return ids
	}

	repeat := func(tokens []string, n int) []string {
		out := make([]string, 0, len(tokens)*n)
		for i := 0; i < n; i++ {
			out = append(out, tokens...)
		}
		return out
	}

	type weightedField struct {
		weight float64
		ranked []string
	}
	fields := []weightedField{
		{titleWeight, scoreField(func(n *Note) []string { return repeat(tokenize(n.Title), titleWeight) })},
		{1, scoreField(func(n *Note) []string { return tokenize(n.Body) })},
		{float64(tagWeight), scoreField(func(n *Note) []string {
			var all []string
			for _, tag := range n.Tags {
				all = append(all, repeat(tokenize(tag), tagWeight)...)
			}
			return all
		})},
		{1, scoreField(func(n *Note) []string {
			var all []string
			for _, ann := range inbound[n.ID] {
				all = append(all, tokenize(ann)...)
			}
			return all
		})},
	}

	rrf := make(map[string]float64)
	for _, f := range fields {
		for rank, id := range f.ranked {
			rrf[id] += f.weight / (rrfK + float64(rank+1))
		}
	}

	result := make(map[string]float64, len(rrf))
	for id, s := range rrf {
		if s > 0 {
			result[id] = s
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// statusMultiplier returns a score multiplier based on note status.
// Settled notes (permanent, reviewed) rank above drafts for the same BM25 content score.
func statusMultiplier(s Status) float64 {
	switch s {
	case StatusPermanent:
		return 1.05
	case StatusReviewed:
		return 1.02
	default:
		return 1.0
	}
}

// Tokenize splits text into lowercase tokens. Exported for use in match-reason computation.
func Tokenize(s string) []string {
	return tokenize(s)
}

// tokenize splits text into lowercase tokens.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	var tokens []string
	for _, word := range strings.FieldsFunc(s, func(r rune) bool {
		return !('a' <= r && r <= 'z') && !('0' <= r && r <= '9')
	}) {
		if len(word) > 1 {
			tokens = append(tokens, english.Stem(word, false))
		}
	}
	return tokens
}
