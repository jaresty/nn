package note

import (
	"math"
	"sort"
)

// AnnotationDirection identifies which endpoint receives annotation evidence.
type AnnotationDirection string

const (
	AnnotationInbound  AnnotationDirection = "inbound"
	AnnotationOutbound AnnotationDirection = "outbound"

	// UnclassifiedEdgeType is the read-only scoring identity for legacy links
	// whose stored edge type is empty.
	UnclassifiedEdgeType = "UNCLASSIFIED"
)

// AnnotationChannel is the canonical identity of graph annotation evidence.
type AnnotationChannel struct {
	Direction AnnotationDirection `json:"direction"`
	EdgeType  string              `json:"edge_type"`
}

// NewAnnotationChannel maps a legacy empty edge type to UNCLASSIFIED.
func NewAnnotationChannel(direction AnnotationDirection, edgeType string) AnnotationChannel {
	if edgeType == "" {
		edgeType = UnclassifiedEdgeType
	}
	return AnnotationChannel{Direction: direction, EdgeType: edgeType}
}

func annotationChannelLess(a, b AnnotationChannel) bool {
	directionOrder := func(direction AnnotationDirection) int {
		switch direction {
		case AnnotationInbound:
			return 0
		case AnnotationOutbound:
			return 1
		default:
			return 2
		}
	}
	if ai, bi := directionOrder(a.Direction), directionOrder(b.Direction); ai != bi {
		return ai < bi
	}
	if a.Direction != b.Direction {
		return a.Direction < b.Direction
	}
	return a.EdgeType < b.EdgeType
}

// AnnotationChannels assigns annotation text to notes independently for each
// direction and edge-type channel.
type AnnotationChannels map[AnnotationChannel]map[string][]string

// Add appends annotation text to the canonical channel and note assignment.
func (channels AnnotationChannels) Add(direction AnnotationDirection, edgeType, noteID, text string) {
	channel := NewAnnotationChannel(direction, edgeType)
	if channels[channel] == nil {
		channels[channel] = make(map[string][]string)
	}
	channels[channel][noteID] = append(channels[channel][noteID], text)
}

// Canonicalized normalizes legacy empty edge types and merges equivalent
// channel entries. Annotation order and multiplicity are preserved.
func (channels AnnotationChannels) Canonicalized() AnnotationChannels {
	canonical := make(AnnotationChannels)
	for channel, assignments := range channels {
		channel = NewAnnotationChannel(channel.Direction, channel.EdgeType)
		if canonical[channel] == nil {
			canonical[channel] = make(map[string][]string)
		}
		for noteID, annotations := range assignments {
			canonical[channel][noteID] = append(canonical[channel][noteID], annotations...)
		}
	}
	return canonical
}

func orderedAnnotationChannels(canonical AnnotationChannels) []AnnotationChannel {
	ordered := make([]AnnotationChannel, 0, len(canonical))
	for channel := range canonical {
		ordered = append(ordered, channel)
	}
	sort.Slice(ordered, func(i, j int) bool { return annotationChannelLess(ordered[i], ordered[j]) })
	return ordered
}

// CanonicalChannels returns normalized channel identities in direction/type order.
func (channels AnnotationChannels) CanonicalChannels() []AnnotationChannel {
	return orderedAnnotationChannels(channels.Canonicalized())
}

// FlatAnnotationChannels adapts legacy flat maps to UNCLASSIFIED channels.
func FlatAnnotationChannels(inbound, outbound map[string][]string) AnnotationChannels {
	channels := make(AnnotationChannels)
	for noteID, annotations := range inbound {
		for _, annotation := range annotations {
			channels.Add(AnnotationInbound, "", noteID, annotation)
		}
	}
	for noteID, annotations := range outbound {
		for _, annotation := range annotations {
			channels.Add(AnnotationOutbound, "", noteID, annotation)
		}
	}
	return channels
}

// FlatMaps projects all edge types back into the legacy direction-only shape.
func (channels AnnotationChannels) FlatMaps() (map[string][]string, map[string][]string) {
	inbound := make(map[string][]string)
	outbound := make(map[string][]string)
	canonical := channels.Canonicalized()
	for _, channel := range orderedAnnotationChannels(canonical) {
		for noteID, annotations := range canonical[channel] {
			switch channel.Direction {
			case AnnotationInbound:
				inbound[noteID] = append(inbound[noteID], annotations...)
			case AnnotationOutbound:
				outbound[noteID] = append(outbound[noteID], annotations...)
			}
		}
	}
	return inbound, outbound
}

// AnnotationChannelIDF persists one channel's field-specific token IDF.
type AnnotationChannelIDF struct {
	Channel AnnotationChannel  `json:"channel"`
	IDF     map[string]float64 `json:"idf"`
}

// TypedFieldIDF holds lexical and typed annotation-channel IDF values.
type TypedFieldIDF struct {
	DocumentCount int                    `json:"document_count"`
	Title         map[string]float64     `json:"title"`
	Body          map[string]float64     `json:"body"`
	Tags          map[string]float64     `json:"tags"`
	Annotations   []AnnotationChannelIDF `json:"annotations"`
}

// Canonicalized returns a payload whose annotation IDFs are normalized and
// sorted by direction then edge type, suitable for deterministic persistence.
func (fidf TypedFieldIDF) Canonicalized() TypedFieldIDF {
	byChannel := make(map[AnnotationChannel]map[string]float64, len(fidf.Annotations))
	for _, channelIDF := range fidf.Annotations {
		channel := NewAnnotationChannel(channelIDF.Channel.Direction, channelIDF.Channel.EdgeType)
		if byChannel[channel] == nil {
			byChannel[channel] = make(map[string]float64)
		}
		for term, value := range channelIDF.IDF {
			byChannel[channel][term] = value
		}
	}
	channels := make([]AnnotationChannel, 0, len(byChannel))
	for channel := range byChannel {
		channels = append(channels, channel)
	}
	sort.Slice(channels, func(i, j int) bool { return annotationChannelLess(channels[i], channels[j]) })

	canonical := TypedFieldIDF{DocumentCount: fidf.DocumentCount, Title: fidf.Title, Body: fidf.Body, Tags: fidf.Tags}
	for _, channel := range channels {
		canonical.Annotations = append(canonical.Annotations, AnnotationChannelIDF{
			Channel: channel,
			IDF:     byChannel[channel],
		})
	}
	return canonical
}

func (fidf TypedFieldIDF) annotationIDFs() map[AnnotationChannel]map[string]float64 {
	result := make(map[AnnotationChannel]map[string]float64, len(fidf.Annotations))
	for _, channelIDF := range fidf.Annotations {
		channel := NewAnnotationChannel(channelIDF.Channel.Direction, channelIDF.Channel.EdgeType)
		result[channel] = channelIDF.IDF
	}
	return result
}

// FlatFieldIDF adapts typed IDFs back to the legacy lexical plus one-direction
// UNCLASSIFIED payload.
func (fidf TypedFieldIDF) FlatFieldIDF(direction AnnotationDirection) FieldIDF {
	flat := FieldIDF{Title: fidf.Title, Body: fidf.Body, Tags: fidf.Tags, Inbound: make(map[string]float64)}
	channel := NewAnnotationChannel(direction, "")
	for term, value := range fidf.annotationIDFs()[channel] {
		flat.Inbound[term] = value
	}
	// Legacy FieldIDF includes df=0 entries for terms that occur elsewhere in
	// the corpus. Typed channels persist only their own vocabulary, so restore
	// those compatibility values without re-tokenizing the corpus.
	absentIDF := math.Log((float64(fidf.DocumentCount)+0.5)/0.5 + 1)
	for term := range fidf.Title {
		if _, ok := flat.Inbound[term]; !ok {
			flat.Inbound[term] = absentIDF
		}
	}
	return flat
}

// BM25TypedFieldIDF computes field-specific IDFs for lexical fields and every
// canonical annotation channel. Lexical maps retain their established shape;
// each graph channel persists only its own vocabulary.
func BM25TypedFieldIDF(notes []*Note, channels AnnotationChannels) TypedFieldIDF {
	canonical := channels.Canonicalized()
	allAnnotations := make(map[string][]string)
	for _, assignments := range canonical {
		for noteID, annotations := range assignments {
			allAnnotations[noteID] = append(allAnnotations[noteID], annotations...)
		}
	}
	lexical := BM25FieldIDF(notes, allAnnotations)
	result := TypedFieldIDF{DocumentCount: len(notes), Title: lexical.Title, Body: lexical.Body, Tags: lexical.Tags}

	for _, channel := range orderedAnnotationChannels(canonical) {
		field := make([][]string, len(notes))
		termsSeen := make(map[string]struct{})
		assignments := canonical[channel]
		for i, n := range notes {
			for _, annotation := range assignments[n.ID] {
				field[i] = append(field[i], tokenize(annotation)...)
			}
			for _, term := range field[i] {
				termsSeen[term] = struct{}{}
			}
		}
		terms := make([]string, 0, len(termsSeen))
		for term := range termsSeen {
			terms = append(terms, term)
		}
		sort.Strings(terms)
		result.Annotations = append(result.Annotations, AnnotationChannelIDF{
			Channel: channel,
			IDF:     fieldIDF(field, len(notes), terms),
		})
	}
	return result
}

func typedFieldIDFFromLegacy(corpus []*Note, fidf FieldIDF, channels AnnotationChannels) TypedFieldIDF {
	typed := BM25TypedFieldIDF(corpus, channels)
	typed.Title = fidf.Title
	typed.Body = fidf.Body
	typed.Tags = fidf.Tags
	inboundLegacy := NewAnnotationChannel(AnnotationInbound, "")
	for i := range typed.Annotations {
		if typed.Annotations[i].Channel == inboundLegacy {
			typed.Annotations[i].IDF = fidf.Inbound
		}
	}
	return typed
}

// BM25RRFPerFieldTypedForCorpus scores canonical typed annotation channels.
// Production's established policy is retained: graph evidence is outbound-only.
func BM25RRFPerFieldTypedForCorpus(corpus, candidates []*Note, fidf TypedFieldIDF, query string, channels AnnotationChannels) map[string]float64 {
	return bm25RRFPerFieldTypedCached(corpus, candidates, fidf, query, channels.Canonicalized(), 1, 0, 1, nil, nil)
}

// TypedCorpusScorer scores many queries against a fixed typed corpus while
// reusing lexical and complete-channel graph tokenization.
type TypedCorpusScorer struct {
	corpus       []*Note
	fidf         TypedFieldIDF
	channels     AnnotationChannels
	lexicalCache *tokenCache
	channelCache *channelTokenCache
}

// NewTypedCorpusScorer constructs the typed scorer entry point.
func NewTypedCorpusScorer(corpus []*Note, fidf TypedFieldIDF, channels AnnotationChannels) *TypedCorpusScorer {
	return &TypedCorpusScorer{
		corpus:       corpus,
		fidf:         fidf,
		channels:     channels.Canonicalized(),
		lexicalCache: newTokenCache(),
		channelCache: newChannelTokenCache(),
	}
}

// Score ranks candidates with the production outbound-only direction policy.
func (s *TypedCorpusScorer) Score(candidates []*Note, query string) map[string]float64 {
	return bm25RRFPerFieldTypedCached(s.corpus, candidates, s.fidf, query, s.channels, 1, 0, 1, s.lexicalCache, s.channelCache)
}

// channelTokenCache memoizes graph tokens by note and complete canonical
// direction/edge-type channel identity.
type channelTokenCache struct {
	byChannel map[AnnotationChannel]map[*Note][]string
}

func newChannelTokenCache() *channelTokenCache {
	return &channelTokenCache{byChannel: make(map[AnnotationChannel]map[*Note][]string)}
}

func (cache *channelTokenCache) get(channel AnnotationChannel, n *Note, compute func(*Note) []string) []string {
	if cache == nil {
		return compute(n)
	}
	channel = NewAnnotationChannel(channel.Direction, channel.EdgeType)
	byNote := cache.byChannel[channel]
	if byNote == nil {
		byNote = make(map[*Note][]string)
		cache.byChannel[channel] = byNote
	}
	if tokens, ok := byNote[n]; ok {
		return tokens
	}
	tokens := compute(n)
	byNote[n] = tokens
	return tokens
}

func bm25RRFPerFieldTypedCached(
	corpus, candidates []*Note,
	fidf TypedFieldIDF,
	query string,
	channels AnnotationChannels,
	tagFieldWeight, inboundFieldWeight, outboundFieldWeight float64,
	lexicalCache *tokenCache,
	channelCache *channelTokenCache,
) map[string]float64 {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	type fieldScore struct {
		id    string
		score float64
	}
	scoreField := func(getTokens func(*Note) []string, idf map[string]float64) []string {
		totalLen := 0.0
		for _, n := range corpus {
			totalLen += float64(len(getTokens(n)))
		}
		avgdl := totalLen / math.Max(float64(len(corpus)), 1)
		type document struct {
			tf  map[string]float64
			len float64
		}
		documents := make([]document, len(candidates))
		for i, n := range candidates {
			tokens := getTokens(n)
			tf := make(map[string]float64, len(tokens))
			for _, token := range tokens {
				tf[token]++
			}
			documents[i] = document{tf: tf, len: float64(len(tokens))}
		}

		var ranked []fieldScore
		for i, n := range candidates {
			document := documents[i]
			score := 0.0
			for _, term := range terms {
				tf := document.tf[term]
				if tf == 0 {
					continue
				}
				score += idf[term] * (tf * (bm25K1 + 1)) /
					(tf + bm25K1*(1-bm25B+bm25B*document.len/avgdl))
			}
			if score > 0 {
				ranked = append(ranked, fieldScore{id: n.ID, score: score})
			}
		}
		// Preserve the historical candidate-sequence-dependent permutation for
		// equal scores. Tie redesign remains a separate decision.
		for i := 0; i < len(ranked)-1; i++ {
			for j := i + 1; j < len(ranked); j++ {
				if ranked[j].score > ranked[i].score {
					ranked[i], ranked[j] = ranked[j], ranked[i]
				}
			}
		}
		ids := make([]string, len(ranked))
		for i, scored := range ranked {
			ids[i] = scored.id
		}
		return ids
	}

	titleTokens := func(n *Note) []string {
		return lexicalCache.get(fieldTitleTokens, n, func(n *Note) []string { return tokenize(n.Title) })
	}
	bodyTokens := func(n *Note) []string {
		return lexicalCache.get(fieldBodyTokens, n, func(n *Note) []string { return tokenize(n.Body) })
	}
	tagTokens := func(n *Note) []string {
		return lexicalCache.get(fieldTagTokens, n, func(n *Note) []string {
			var all []string
			for _, tag := range n.Tags {
				all = append(all, tokenize(tag)...)
			}
			return all
		})
	}

	rrf := make(map[string]float64)
	addRanked := func(ranked []string, weight float64) {
		for rank, id := range ranked {
			rrf[id] += weight / (rrfK + float64(rank+1))
		}
	}
	addRanked(scoreField(titleTokens, fidf.Title), titleWeight)
	addRanked(scoreField(bodyTokens, fidf.Body), 1)
	addRanked(scoreField(tagTokens, fidf.Tags), tagFieldWeight)

	channelIDF := fidf.annotationIDFs()
	directionalSum := map[AnnotationDirection]map[string]float64{
		AnnotationInbound:  make(map[string]float64),
		AnnotationOutbound: make(map[string]float64),
	}
	directionalMatches := map[AnnotationDirection]map[string]int{
		AnnotationInbound:  make(map[string]int),
		AnnotationOutbound: make(map[string]int),
	}
	directionWeight := func(direction AnnotationDirection) float64 {
		switch direction {
		case AnnotationInbound:
			return inboundFieldWeight
		case AnnotationOutbound:
			return outboundFieldWeight
		default:
			return 0
		}
	}
	for _, channel := range orderedAnnotationChannels(channels) {
		if directionWeight(channel.Direction) == 0 {
			continue
		}
		assignments := channels[channel]
		getTokens := func(n *Note) []string {
			return channelCache.get(channel, n, func(n *Note) []string {
				var all []string
				for _, annotation := range assignments[n.ID] {
					all = append(all, tokenize(annotation)...)
				}
				return all
			})
		}
		idf := channelIDF[channel]
		if idf == nil {
			field := make([][]string, len(corpus))
			for i, n := range corpus {
				field[i] = getTokens(n)
			}
			idf = fieldIDF(field, len(corpus), terms)
		}
		for rank, id := range scoreField(getTokens, idf) {
			directionalSum[channel.Direction][id] += 1 / (rrfK + float64(rank+1))
			directionalMatches[channel.Direction][id]++
		}
	}
	for _, direction := range []AnnotationDirection{AnnotationInbound, AnnotationOutbound} {
		weight := directionWeight(direction)
		if weight == 0 {
			continue
		}
		for id, sum := range directionalSum[direction] {
			// Only matching channels increment the denominator; absent channel
			// evidence contributes no zero vote.
			rrf[id] += weight * sum / float64(directionalMatches[direction][id])
		}
	}

	if len(rrf) == 0 {
		return nil
	}
	return rrf
}
