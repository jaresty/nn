package note

import "sort"

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

// CanonicalChannels returns channel identities in direction/type order.
func (channels AnnotationChannels) CanonicalChannels() []AnnotationChannel {
	seen := make(map[AnnotationChannel]struct{}, len(channels))
	for channel := range channels {
		seen[NewAnnotationChannel(channel.Direction, channel.EdgeType)] = struct{}{}
	}
	ordered := make([]AnnotationChannel, 0, len(seen))
	for channel := range seen {
		ordered = append(ordered, channel)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Direction != ordered[j].Direction {
			return ordered[i].Direction < ordered[j].Direction
		}
		return ordered[i].EdgeType < ordered[j].EdgeType
	})
	return ordered
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
	for channel, assignments := range channels {
		for noteID, annotations := range assignments {
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
	Title       map[string]float64     `json:"title"`
	Body        map[string]float64     `json:"body"`
	Tags        map[string]float64     `json:"tags"`
	Annotations []AnnotationChannelIDF `json:"annotations"`
}

// BM25TypedFieldIDF is a compatibility scaffold that currently projects graph
// evidence into direction-only channels. Typed channel tests intentionally pin
// the semantics this implementation must replace.
func BM25TypedFieldIDF(notes []*Note, channels AnnotationChannels) TypedFieldIDF {
	inbound, outbound := channels.FlatMaps()
	lexical := BM25FieldIDF(notes, inbound)
	result := TypedFieldIDF{Title: lexical.Title, Body: lexical.Body, Tags: lexical.Tags}
	if len(inbound) > 0 {
		result.Annotations = append(result.Annotations, AnnotationChannelIDF{
			Channel: NewAnnotationChannel(AnnotationInbound, ""), IDF: lexical.Inbound,
		})
	}
	if len(outbound) > 0 {
		outboundIDF := BM25FieldIDF(notes, outbound)
		result.Annotations = append(result.Annotations, AnnotationChannelIDF{
			Channel: NewAnnotationChannel(AnnotationOutbound, ""), IDF: outboundIDF.Inbound,
		})
	}
	return result
}

func legacyFieldIDF(fidf TypedFieldIDF, channels AnnotationChannels) FieldIDF {
	legacy := FieldIDF{Title: fidf.Title, Body: fidf.Body, Tags: fidf.Tags}
	for _, channelIDF := range fidf.Annotations {
		if channelIDF.Channel.Direction == AnnotationInbound {
			legacy.Inbound = channelIDF.IDF
			break
		}
	}
	if legacy.Inbound == nil {
		inbound, _ := channels.FlatMaps()
		legacy.Inbound = BM25FieldIDF(nil, inbound).Inbound
	}
	return legacy
}

// BM25RRFPerFieldTypedForCorpus is the typed scorer entry point. The temporary
// direction-only projection keeps it buildable before channel fusion lands.
func BM25RRFPerFieldTypedForCorpus(corpus, candidates []*Note, fidf TypedFieldIDF, query string, channels AnnotationChannels) map[string]float64 {
	inbound, outbound := channels.FlatMaps()
	return BM25RRFPerFieldForCorpus(corpus, candidates, legacyFieldIDF(fidf, channels), query, inbound, outbound)
}

// TypedCorpusScorer scores repeated queries over typed annotation inputs.
type TypedCorpusScorer struct {
	corpus   []*Note
	fidf     TypedFieldIDF
	channels AnnotationChannels
}

// NewTypedCorpusScorer constructs the typed scorer entry point.
func NewTypedCorpusScorer(corpus []*Note, fidf TypedFieldIDF, channels AnnotationChannels) *TypedCorpusScorer {
	return &TypedCorpusScorer{corpus: corpus, fidf: fidf, channels: channels}
}

// Score ranks candidates using the typed scorer entry point.
func (s *TypedCorpusScorer) Score(candidates []*Note, query string) map[string]float64 {
	return BM25RRFPerFieldTypedForCorpus(s.corpus, candidates, s.fidf, query, s.channels)
}

// channelTokenCache is the graph-token compatibility scaffold. Its temporary
// direction-only key is intentionally insufficient for typed channels.
type channelTokenCache struct {
	byDirection map[AnnotationDirection]map[*Note][]string
}

func newChannelTokenCache() *channelTokenCache {
	return &channelTokenCache{byDirection: make(map[AnnotationDirection]map[*Note][]string)}
}

func (cache *channelTokenCache) get(channel AnnotationChannel, n *Note, compute func(*Note) []string) []string {
	byNote := cache.byDirection[channel.Direction]
	if byNote == nil {
		byNote = make(map[*Note][]string)
		cache.byDirection[channel.Direction] = byNote
	}
	if tokens, ok := byNote[n]; ok {
		return tokens
	}
	tokens := compute(n)
	byNote[n] = tokens
	return tokens
}
