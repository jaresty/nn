package note

import (
	"strings"
	"testing"
	"time"
)

func TestAppliesWhenRoundTrips(t *testing.T) {
	n := &Note{
		ID:          "test-id",
		Title:       "Test Protocol",
		Type:        TypeProtocol,
		Status:      StatusPermanent,
		AppliesWhen: "before any external action",
		Created:     time.Now(),
		Modified:    time.Now(),
	}
	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.AppliesWhen != "before any external action" {
		t.Errorf("expected AppliesWhen='before any external action'; got %q", parsed.AppliesWhen)
	}
}

func TestAppliesWhenOmittedWhenEmpty(t *testing.T) {
	n := &Note{
		ID:       "test-id",
		Title:    "Test Protocol",
		Type:     TypeProtocol,
		Status:   StatusPermanent,
		Created:  time.Now(),
		Modified: time.Now(),
	}
	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "applies_when") {
		t.Errorf("expected applies_when to be omitted when empty; got:\n%s", string(data))
	}
}
