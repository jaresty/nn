package index

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func oneTypedChannel(direction note.AnnotationDirection, edgeType, noteID, text string) note.AnnotationChannels {
	channels := make(note.AnnotationChannels)
	channels.Add(direction, edgeType, noteID, text)
	return channels
}

func TestTypedFieldIDFCacheKeyIncludesCompleteChannelIdentity(t *testing.T) {
	notes := makeTestNotes()
	baseChannels := oneTypedChannel(note.AnnotationOutbound, "supports", "n1", "annotation text")
	base := typedFieldIDFCacheKey("/repo", "head", notes, baseChannels)

	t.Run("version", func(t *testing.T) {
		if !strings.HasPrefix(base, "field-idf-v4:") {
			t.Fatalf("typed cache key prefix = %q, want bumped field-idf-v4 namespace", strings.SplitN(base, ":", 2)[0])
		}
	})
	t.Run("edge type", func(t *testing.T) {
		changed := oneTypedChannel(note.AnnotationOutbound, "extends", "n1", "annotation text")
		if got := typedFieldIDFCacheKey("/repo", "head", notes, changed); got == base {
			t.Fatal("typed cache key ignored an edge-type-only change")
		}
	})
	t.Run("direction", func(t *testing.T) {
		changed := oneTypedChannel(note.AnnotationInbound, "supports", "n1", "annotation text")
		if got := typedFieldIDFCacheKey("/repo", "head", notes, changed); got == base {
			t.Fatal("typed cache key ignored an annotation-direction-only change")
		}
	})
	t.Run("note assignment", func(t *testing.T) {
		changed := oneTypedChannel(note.AnnotationOutbound, "supports", "n2", "annotation text")
		if got := typedFieldIDFCacheKey("/repo", "head", notes, changed); got == base {
			t.Fatal("typed cache key ignored an annotation note-assignment change")
		}
	})
	t.Run("annotation text", func(t *testing.T) {
		changed := oneTypedChannel(note.AnnotationOutbound, "supports", "n1", "different text")
		if got := typedFieldIDFCacheKey("/repo", "head", notes, changed); got == base {
			t.Fatal("typed cache key ignored annotation text")
		}
	})
}

func TestTypedFieldIDFCacheKeyCanonicalizesMapAndAnnotationOrder(t *testing.T) {
	notes := makeTestNotes()
	first := make(note.AnnotationChannels)
	first.Add(note.AnnotationOutbound, "supports", "n1", "z text")
	first.Add(note.AnnotationInbound, "extends", "n2", "middle text")
	first.Add(note.AnnotationOutbound, "supports", "n1", "a text")

	second := make(note.AnnotationChannels)
	second.Add(note.AnnotationOutbound, "supports", "n1", "a text")
	second.Add(note.AnnotationOutbound, "supports", "n1", "z text")
	second.Add(note.AnnotationInbound, "extends", "n2", "middle text")

	firstKey := typedFieldIDFCacheKey("/repo", "head", notes, first)
	secondKey := typedFieldIDFCacheKey("/repo", "head", notes, second)
	if firstKey != secondKey {
		t.Fatalf("canonical-equivalent typed evidence produced different cache keys:\n%s\n%s", firstKey, secondKey)
	}
}

func TestTypedFieldIDFCachePersistsCanonicalChannelOrder(t *testing.T) {
	idx, err := Open(t.TempDir() + "/index.db")
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	fidf := note.TypedFieldIDF{
		Title: map[string]float64{"title": 1},
		Annotations: []note.AnnotationChannelIDF{
			{Channel: note.NewAnnotationChannel(note.AnnotationOutbound, "supports"), IDF: map[string]float64{"support": 2}},
			{Channel: note.NewAnnotationChannel(note.AnnotationInbound, ""), IDF: map[string]float64{"legacy": 3}},
			{Channel: note.NewAnnotationChannel(note.AnnotationOutbound, "extends"), IDF: map[string]float64{"extend": 4}},
		},
	}
	if err := idx.storeTypedFieldIDF("typed", fidf); err != nil {
		t.Fatal(err)
	}
	got, err := idx.getTypedFieldIDFFromCache("typed")
	if err != nil {
		t.Fatal(err)
	}
	gotChannels := make([]note.AnnotationChannel, 0, len(got.Annotations))
	for _, channelIDF := range got.Annotations {
		gotChannels = append(gotChannels, channelIDF.Channel)
	}
	wantChannels := []note.AnnotationChannel{
		note.NewAnnotationChannel(note.AnnotationInbound, ""),
		note.NewAnnotationChannel(note.AnnotationOutbound, "extends"),
		note.NewAnnotationChannel(note.AnnotationOutbound, "supports"),
	}
	if !reflect.DeepEqual(gotChannels, wantChannels) {
		t.Fatalf("persisted typed IDF channels = %#v, want canonical direction/type order %#v", gotChannels, wantChannels)
	}
	if got.Title["title"] != 1 || got.Annotations[0].IDF["legacy"] != 3 {
		t.Fatalf("typed IDF cache roundtrip lost payload values: %#v", got)
	}
}
