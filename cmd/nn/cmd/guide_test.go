package cmd

import (
	"strings"
	"testing"
)

// Assertion: nn guide lists available topics.
func TestGuideListsTopics(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("guide")
	if err != nil {
		t.Fatalf("nn guide: %v", err)
	}
	if !strings.Contains(out, "ref") {
		t.Errorf("expected 'ref' topic in nn guide output:\n%s", out)
	}
	if !strings.Contains(out, "workflow") {
		t.Errorf("expected 'workflow' topic in nn guide output:\n%s", out)
	}
}

// Assertion: nn guide workflow prints skill content.
func TestGuideRefPrintsContent(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("guide", "ref")
	if err != nil {
		t.Fatalf("nn guide ref: %v", err)
	}
	// nn-guide SKILL.md contains these strings.
	if !strings.Contains(out, "concept") {
		t.Errorf("expected 'concept' in ref guide:\n%s", out)
	}
	if !strings.Contains(out, "nn new") {
		t.Errorf("expected 'nn new' in ref guide:\n%s", out)
	}
}

// Assertion: nn guide unknown returns an error.
func TestGuideUnknownTopicErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("guide", "no-such-topic")
	if err == nil {
		t.Errorf("expected error for unknown topic, got nil")
	}
}
