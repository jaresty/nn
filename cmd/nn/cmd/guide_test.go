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
	if !strings.Contains(out, "guide") {
		t.Errorf("expected 'guide' topic in nn guide output:\n%s", out)
	}
	if !strings.Contains(out, "workflow") {
		t.Errorf("expected 'workflow' topic in nn guide output:\n%s", out)
	}
}

// Assertion: nn guide workflow prints skill content.
func TestGuideWorkflowPrintsContent(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("guide", "workflow")
	if err != nil {
		t.Fatalf("nn guide workflow: %v", err)
	}
	// nn-workflow SKILL.md contains these strings.
	if !strings.Contains(out, "Session Start") {
		t.Errorf("expected 'Session Start' in workflow guide:\n%s", out)
	}
	if !strings.Contains(out, "nn new") {
		t.Errorf("expected 'nn new' in workflow guide:\n%s", out)
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
