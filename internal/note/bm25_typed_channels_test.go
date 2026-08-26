package note

import (
	"math"
	"reflect"
	"testing"
)

func typedChannelFixture() ([]*Note, AnnotationChannels) {
	notes := []*Note{
		{ID: "one-channel", Title: "plain one", Body: "ordinary body"},
		{ID: "two-channels", Title: "plain two", Body: "ordinary body"},
	}
	channels := make(AnnotationChannels)
	channels.Add(AnnotationOutbound, "supports", notes[0].ID, "channelnonce")
	channels.Add(AnnotationOutbound, "extends", notes[1].ID, "channelnonce")
	channels.Add(AnnotationOutbound, "contradicts", notes[1].ID, "channelnonce")
	return notes, channels
}

// Properties [1]-[3]: edge types remain independent RRF channels, matching
// channels are averaged within one direction, and channel IDF identities are
// canonical. A note does not earn extra graph votes merely by matching more
// relationship types with equal per-channel evidence.
func TestTypedAnnotationChannelsMeanNormalizePerDirection(t *testing.T) {
	notes, channels := typedChannelFixture()
	fidf := BM25TypedFieldIDF(notes, channels)
	scores := BM25RRFPerFieldTypedForCorpus(notes, notes, fidf, "channelnonce", channels)

	want := 1.0 / (rrfK + 1)
	for _, n := range notes {
		if diff := math.Abs(scores[n.ID] - want); diff > 1e-12 {
			t.Errorf("properties [1]-[3]: %s graph score = %.12f, want one mean-normalized directional vote %.12f", n.ID, scores[n.ID], want)
		}
	}
	if scores[notes[0].ID] != scores[notes[1].ID] {
		t.Errorf("properties [1]-[3]: one matching type and two equally ranked matching types must tie: one=%.12f two=%.12f", scores[notes[0].ID], scores[notes[1].ID])
	}
}

func TestTypedFieldIDFUsesCanonicalDirectionAndEdgeTypeChannels(t *testing.T) {
	notes := []*Note{{ID: "n1"}, {ID: "n2"}}
	channels := make(AnnotationChannels)
	channels.Add(AnnotationOutbound, "supports", "n1", "supportterm")
	channels.Add(AnnotationInbound, "", "n2", "legacyterm")
	channels.Add(AnnotationOutbound, "extends", "n2", "extendterm")

	fidf := BM25TypedFieldIDF(notes, channels)
	got := make([]AnnotationChannel, 0, len(fidf.Annotations))
	for _, channelIDF := range fidf.Annotations {
		got = append(got, channelIDF.Channel)
	}
	want := []AnnotationChannel{
		NewAnnotationChannel(AnnotationInbound, UnclassifiedEdgeType),
		NewAnnotationChannel(AnnotationOutbound, "extends"),
		NewAnnotationChannel(AnnotationOutbound, "supports"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("typed IDF channel order/identity = %#v, want canonical direction/type order %#v", got, want)
	}
}

// Property [4a]: title, body, and tag remain independently ranked lexical
// fields with their established RRF weights.
func TestTypedAnnotationRRFRetainsLexicalTitleBodyTagWeights(t *testing.T) {
	notes := []*Note{
		{ID: "title", Title: "titlenonce"},
		{ID: "body", Body: "bodynonce"},
		{ID: "tag", Tags: []string{"tagnonce"}},
	}
	fidf := BM25TypedFieldIDF(notes, nil)
	cases := []struct {
		query string
		id    string
		want  float64
	}{
		{query: "titlenonce", id: "title", want: titleWeight / (rrfK + 1)},
		{query: "bodynonce", id: "body", want: 1 / (rrfK + 1)},
		{query: "tagnonce", id: "tag", want: 1 / (rrfK + 1)},
	}
	for _, tc := range cases {
		scores := BM25RRFPerFieldTypedForCorpus(notes, notes, fidf, tc.query, nil)
		if got := scores[tc.id]; got != tc.want {
			t.Errorf("property [4a]: %s score = %.12f, want retained lexical contribution %.12f", tc.id, got, tc.want)
		}
	}
}

// Property [4b]: graph directions remain independently weighted and production
// ranking keeps the established outbound-only policy.
func TestTypedAnnotationRRFRetainsOutboundOnlyPolicy(t *testing.T) {
	notes := []*Note{{ID: "source"}, {ID: "target"}}
	channels := make(AnnotationChannels)
	channels.Add(AnnotationOutbound, "supports", "source", "directionnonce")
	channels.Add(AnnotationInbound, "supports", "target", "directionnonce")
	fidf := BM25TypedFieldIDF(notes, channels)

	scores := BM25RRFPerFieldTypedForCorpus(notes, notes, fidf, "directionnonce", channels)
	if scores["source"] == 0 {
		t.Fatal("property [4b]: outbound annotation did not contribute")
	}
	if scores["target"] != 0 {
		t.Fatalf("property [4b]: inbound-only target score = %.12f, want zero under outbound-only production policy", scores["target"])
	}
}

// Property [5]: equal BM25 field scores retain the historical candidate-order
// permutation rather than acquiring a new tie-break rule.
func TestTypedAnnotationRRFRetainsCandidateTiePermutation(t *testing.T) {
	a := &Note{ID: "a", Body: "query"}
	b := &Note{ID: "b", Body: "query"}
	c := &Note{ID: "c", Body: "query query"}
	notes := []*Note{a, b, c}
	scores := BM25RRFPerFieldTypedForCorpus(notes, notes, BM25TypedFieldIDF(notes, nil), "query", nil)
	if !(scores[c.ID] > scores[b.ID] && scores[b.ID] > scores[a.ID]) {
		t.Fatalf("property [5]: compatibility permutation changed: c=%f b=%f a=%f", scores[c.ID], scores[b.ID], scores[a.ID])
	}
}

// Property [6]: existing flat annotation APIs are exact adapters to inbound and
// outbound UNCLASSIFIED channels.
func TestFlatAnnotationAPIsAdaptExactlyToUnclassifiedChannels(t *testing.T) {
	corpus := []*Note{
		{ID: "a", Title: "alpha", Body: "shared body", Tags: []string{"tagged"}},
		{ID: "b", Title: "beta", Body: "shared body"},
		{ID: "c", Title: "gamma", Body: "other"},
	}
	inbound := map[string][]string{"a": {"inboundnonce"}}
	outbound := map[string][]string{"b": {"outboundnonce"}}
	flatIDF := BM25FieldIDF(corpus, inbound)
	channels := FlatAnnotationChannels(inbound, outbound)
	typedIDF := BM25TypedFieldIDF(corpus, channels)

	for _, query := range []string{"alpha shared tagged", "inboundnonce", "outboundnonce"} {
		want := BM25RRFPerFieldForCorpus(corpus, corpus[:2], flatIDF, query, inbound, outbound)
		got := BM25RRFPerFieldTypedForCorpus(corpus, corpus[:2], typedIDF, query, channels)
		assertScoreMapsEqual(t, "property [6] query="+query, want, got)
	}

	for channel := range channels {
		if channel.EdgeType != UnclassifiedEdgeType {
			t.Fatalf("property [6]: flat annotation mapped to edge type %q, want %q", channel.EdgeType, UnclassifiedEdgeType)
		}
	}
}

func TestGraphTokenCacheKeysCompleteChannelIdentity(t *testing.T) {
	cache := newChannelTokenCache()
	n := &Note{ID: "n"}
	cases := []struct {
		channel AnnotationChannel
		text    string
	}{
		{NewAnnotationChannel(AnnotationInbound, "supports"), "alpha"},
		{NewAnnotationChannel(AnnotationInbound, "contradicts"), "beta"},
		{NewAnnotationChannel(AnnotationOutbound, "supports"), "gamma"},
	}
	for _, tc := range cases {
		got := cache.get(tc.channel, n, func(*Note) []string { return []string{tc.text} })
		if !reflect.DeepEqual(got, []string{tc.text}) {
			t.Errorf("token cache channel %s/%s returned %v, want independent tokens [%s]", tc.channel.Direction, tc.channel.EdgeType, got, tc.text)
		}
	}
}
