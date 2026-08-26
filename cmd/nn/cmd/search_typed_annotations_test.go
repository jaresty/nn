package cmd

import (
	"math"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Production wiring: prepareCorpus must retain edge types when projecting links
// into outbound annotation channels. Equal per-channel evidence gets one mean
// directional vote whether a source matches one edge type or several.
func TestPrepareCorpusUsesTypedOutboundAnnotationChannels(t *testing.T) {
	target := &note.Note{ID: "target", Title: "plain target"}
	one := &note.Note{
		ID:    "one-channel",
		Title: "plain one",
		Links: []note.Link{{TargetID: target.ID, Type: "supports", Annotation: "productionnonce"}},
	}
	two := &note.Note{
		ID:    "two-channels",
		Title: "plain two",
		Links: []note.Link{
			{TargetID: target.ID, Type: "extends", Annotation: "productionnonce"},
			{TargetID: target.ID, Type: "contradicts", Annotation: "productionnonce"},
		},
	}
	corpus := []*note.Note{one, two, target}

	scores := prepareCorpus(corpus, "").rankedByQuery(corpus, "productionnonce")
	if scores[one.ID] == 0 || scores[two.ID] == 0 {
		t.Fatalf("typed production annotation matches missing: one=%.12f two=%.12f", scores[one.ID], scores[two.ID])
	}
	if diff := math.Abs(scores[one.ID] - scores[two.ID]); diff > 1e-12 {
		t.Fatalf("prepareCorpus flattened edge types instead of mean-normalizing typed outbound channels: one=%.12f two=%.12f", scores[one.ID], scores[two.ID])
	}
}

// Legacy empty link types remain queryable through the explicit UNCLASSIFIED
// read channel; this does not alter writable link-type validation.
func TestPrepareCorpusMapsLegacyEmptyTypeToUnclassifiedScoringChannel(t *testing.T) {
	target := &note.Note{ID: "target", Title: "plain target"}
	legacy := &note.Note{
		ID:    "legacy-source",
		Title: "plain legacy",
		Links: []note.Link{{TargetID: target.ID, Type: "", Annotation: "legacynonce"}},
	}
	corpus := []*note.Note{legacy, target}

	scores := prepareCorpus(corpus, "").rankedByQuery(corpus, "legacynonce")
	if scores[legacy.ID] == 0 {
		t.Fatal("legacy empty-type outbound annotation was not scored through UNCLASSIFIED")
	}
	if scores[target.ID] != 0 {
		t.Fatalf("legacy inbound target score = %.12f, want zero under outbound-only production policy", scores[target.ID])
	}
}
